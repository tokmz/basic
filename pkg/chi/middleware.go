package chi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return CORSWithConfig(*DefaultCORSConfig())
}

// CORSWithConfig 带配置的CORS中间件
func CORSWithConfig(config CORSConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 检查来源是否允许
		if len(config.AllowOrigins) > 0 {
			allowed := false
			for _, allowedOrigin := range config.AllowOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		
		// 设置CORS头
		if len(config.AllowOrigins) > 0 && config.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		
		if config.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		if len(config.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
		}
		
		if len(config.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
		}
		
		if len(config.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
		}
		
		if config.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", strconv.Itoa(int(config.MaxAge.Seconds())))
		}
		
		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})
}

// SecurityMiddleware 安全中间件
func SecurityMiddleware(config SecurityConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if config.ContentTypeNosniff {
			c.Header("X-Content-Type-Options", "nosniff")
		}
		
		if config.XFrameOptions != "" {
			c.Header("X-Frame-Options", config.XFrameOptions)
		}
		
		if config.XSSProtection != "" {
			c.Header("X-XSS-Protection", config.XSSProtection)
		}
		
		if config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", config.ContentSecurityPolicy)
		}
		
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}
		
		if config.HSTSMaxAge > 0 {
			hsts := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
			if config.HSTSIncludeSubdomains {
				hsts += "; includeSubDomains"
			}
			c.Header("Strict-Transport-Security", hsts)
		}
		
		c.Next()
	})
}

// RequestIDMiddleware 请求ID中间件
func RequestIDMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("RequestID", requestID)
		c.Next()
	})
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 创建带超时的context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		// 替换请求的context
		c.Request = c.Request.WithContext(ctx)
		
		// 创建channel来接收处理完成信号
		done := make(chan struct{})
		
		go func() {
			defer close(done)
			c.Next()
		}()
		
		select {
		case <-done:
			// 请求正常完成
			return
		case <-ctx.Done():
			// 超时
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":      "Request timeout",
				"timeout":    timeout.String(),
				"request_id": c.GetString("RequestID"),
			})
			c.Abort()
		}
	})
}

// AuthMiddleware 认证中间件
func AuthMiddleware(validateFunc func(token string) (interface{}, error)) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 从Header获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "Authorization header required",
				"request_id": c.GetString("RequestID"),
			})
			c.Abort()
			return
		}
		
		// 解析Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "Invalid authorization header format",
				"request_id": c.GetString("RequestID"),
			})
			c.Abort()
			return
		}
		
		token := authHeader[len(bearerPrefix):]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "Token required",
				"request_id": c.GetString("RequestID"),
			})
			c.Abort()
			return
		}
		
		// 验证token
		user, err := validateFunc(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "Invalid token: " + err.Error(),
				"request_id": c.GetString("RequestID"),
			})
			c.Abort()
			return
		}
		
		// 设置用户信息到上下文
		c.Set("user", user)
		c.Next()
	})
}

// RateLimiter 限流器
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	rate     int
	window   time.Duration
}

// NewRateLimiter 创建限流器
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		requests: make(map[string][]time.Time),
		rate:     rate,
		window:   window,
	}
	
	// 启动清理goroutine
	go limiter.cleanup()
	
	return limiter
}

// Allow 检查是否允许请求
func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	
	// 清理过期请求
	if requests, exists := r.requests[key]; exists {
		validRequests := make([]time.Time, 0)
		for _, req := range requests {
			if now.Sub(req) < r.window {
				validRequests = append(validRequests, req)
			}
		}
		r.requests[key] = validRequests
	}
	
	// 检查请求数量
	if len(r.requests[key]) >= r.rate {
		return false
	}
	
	// 记录新请求
	r.requests[key] = append(r.requests[key], now)
	return true
}

// cleanup 清理过期数据
func (r *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for key, requests := range r.requests {
			validRequests := make([]time.Time, 0)
			for _, req := range requests {
				if now.Sub(req) < r.window {
					validRequests = append(validRequests, req)
				}
			}
			if len(validRequests) == 0 {
				delete(r.requests, key)
			} else {
				r.requests[key] = validRequests
			}
		}
		r.mu.Unlock()
	}
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	limiter := NewRateLimiter(config.Rate, config.Window)
	
	return gin.HandlerFunc(func(c *gin.Context) {
		// 生成限流key（可以根据IP、用户ID等自定义）
		key := c.ClientIP()
		
		if !limiter.Allow(key) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"rate_limit":  config.Rate,
				"window":      config.Window.String(),
				"retry_after": config.Window.Seconds(),
				"request_id":  c.GetString("RequestID"),
			})
			c.Abort()
			return
		}
		
		c.Next()
	})
}

// LoggingMiddleware 日志中间件
func LoggingMiddleware(logger Logger) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		c.Next()
		
		// 计算处理时间
		latency := time.Since(start)
		
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()
		
		if raw != "" {
			path = path + "?" + raw
		}
		
		// 记录日志
		fields := []interface{}{
			"method", method,
			"path", path,
			"status", statusCode,
			"latency", latency,
			"client_ip", clientIP,
			"body_size", bodySize,
			"user_agent", c.Request.UserAgent(),
			"request_id", c.GetString("RequestID"),
		}
		
		if statusCode >= 500 {
			logger.Error("HTTP request completed", fields...)
		} else if statusCode >= 400 {
			logger.Warn("HTTP request completed", fields...)
		} else {
			logger.Info("HTTP request completed", fields...)
		}
	})
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware(logger Logger) gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, recovered interface{}) {
		if logger != nil {
			logger.Error("Panic recovered", 
				"error", recovered,
				"request_id", c.GetString("RequestID"),
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
			)
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Internal server error",
			"request_id": c.GetString("RequestID"),
		})
	})
}

// ValidationMiddleware 验证中间件
func ValidationMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 设置最大请求体大小（默认32MB）
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<20)
		
		c.Next()
	})
}

// CompressMiddleware 压缩中间件
func CompressMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 检查客户端是否支持gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		
		// 只压缩特定类型的响应
		contentType := c.GetHeader("Content-Type")
		if !shouldCompress(contentType) {
			c.Next()
			return
		}
		
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
		c.Next()
	})
}

// shouldCompress 判断是否应该压缩
func shouldCompress(contentType string) bool {
	compressibleTypes := []string{
		"application/json",
		"application/xml",
		"text/html",
		"text/plain",
		"text/css",
		"application/javascript",
		"text/javascript",
	}
	
	for _, ct := range compressibleTypes {
		if strings.Contains(contentType, ct) {
			return true
		}
	}
	return false
}

// MetricsMiddleware 指标中间件
func MetricsMiddleware(metrics *Metrics) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()
		metrics.IncCurrentRequests()
		
		c.Next()
		
		duration := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		
		metrics.Record(method, path, status, duration)
	})
}