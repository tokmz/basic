package logger

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HTTPMiddleware HTTP中间件
func HTTPMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// 创建响应写入器包装器
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 添加请求ID到上下文
			ctx := context.WithValue(r.Context(), RequestIDKey, generateRequestID())
			r = r.WithContext(ctx)

			// 记录请求日志
			logger.WithContext(ctx).Info("HTTP Request Started",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("query", r.URL.RawQuery),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("referer", r.Referer()),
			)

			// 执行下一个处理器
			next.ServeHTTP(ww, r)

			// 记录响应日志
			duration := time.Since(start)
			fields := []zap.Field{
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status_code", ww.statusCode),
				zap.Duration("duration", duration),
				zap.Int("response_size", ww.size),
			}

			// 根据状态码决定日志级别
			if ww.statusCode >= 500 {
				logger.WithContext(ctx).Error("HTTP Request Completed", fields...)
			} else if ww.statusCode >= 400 {
				logger.WithContext(ctx).Warn("HTTP Request Completed", fields...)
			} else {
				logger.WithContext(ctx).Info("HTTP Request Completed", fields...)
			}
		})
	}
}

// responseWriter 响应写入器包装器
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// RequestLoggingMiddleware 请求体日志中间件
func RequestLoggingMiddleware(logger *Logger, logBody bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if logBody && r.Body != nil {
				// 读取请求体
				body, err := io.ReadAll(r.Body)
				if err == nil {
					// 重新设置请求体
					r.Body = io.NopCloser(bytes.NewBuffer(body))

					// 记录请求体
					logger.WithContext(r.Context()).Debug("HTTP Request Body",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.String("body", string(body)),
					)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithContext(r.Context()).Error("HTTP Request Panic",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.Any("panic", err),
						zap.String("remote_addr", r.RemoteAddr),
					)

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	// 简单的请求ID生成
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

// randString 生成随机字符串
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)] // 使用索引取模生成字符
	}
	return string(b)
}

// ContextKey 上下文键类型
type ContextKey string

const (
	// RequestIDKey 请求ID键
	RequestIDKey ContextKey = "request_id"
	// TraceIDKey 追踪ID键
	TraceIDKey ContextKey = "trace_id"
	// UserIDKey 用户ID键
	UserIDKey ContextKey = "user_id"
)

// WithRequestID 添加请求ID到上下文
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithTraceID 添加追踪ID到上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithUserID 添加用户ID到上下文
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetRequestID 从上下文获取请求ID
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetTraceID 从上下文获取追踪ID
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetUserID 从上下文获取用户ID
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
