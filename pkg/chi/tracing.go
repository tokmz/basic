package chi

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TraceContext 链路追踪上下文
type TraceContext struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Operation  string            `json:"operation"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time,omitempty"`
	Duration   time.Duration     `json:"duration,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Logs       []TraceLog        `json:"logs,omitempty"`
	Status     TraceStatus       `json:"status"`
	Error      string            `json:"error,omitempty"`
}

// TraceLog 追踪日志
type TraceLog struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// TraceStatus 追踪状态
type TraceStatus string

const (
	TraceStatusOK    TraceStatus = "OK"
	TraceStatusError TraceStatus = "ERROR"
)

// Tracer 追踪器接口
type Tracer interface {
	StartSpan(ctx context.Context, operation string) *Span
	InjectHeaders(span *Span, headers map[string]string)
	ExtractHeaders(headers map[string]string) (*TraceContext, error)
	Report(trace *TraceContext) error
}

// Span 追踪跨度
type Span struct {
	context *TraceContext
	tracer  Tracer
	mu      sync.RWMutex
}

// NewSpan 创建新的跨度
func NewSpan(tracer Tracer, traceID, spanID, parentID, operation string) *Span {
	return &Span{
		tracer: tracer,
		context: &TraceContext{
			TraceID:   traceID,
			SpanID:    spanID,
			ParentID:  parentID,
			Operation: operation,
			StartTime: time.Now(),
			Tags:      make(map[string]string),
			Logs:      make([]TraceLog, 0),
			Status:    TraceStatusOK,
		},
	}
}

// SetTag 设置标签
func (s *Span) SetTag(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.context.Tags[key] = value
}

// LogInfo 记录信息日志
func (s *Span) LogInfo(message string, fields map[string]string) {
	s.log("INFO", message, fields)
}

// LogError 记录错误日志
func (s *Span) LogError(message string, fields map[string]string) {
	s.log("ERROR", message, fields)
	s.mu.Lock()
	s.context.Status = TraceStatusError
	s.context.Error = message
	s.mu.Unlock()
}

// log 记录日志
func (s *Span) log(level, message string, fields map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	log := TraceLog{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
	}
	s.context.Logs = append(s.context.Logs, log)
}

// Finish 完成跨度
func (s *Span) Finish() {
	s.mu.Lock()
	s.context.EndTime = time.Now()
	s.context.Duration = s.context.EndTime.Sub(s.context.StartTime)
	s.mu.Unlock()
	
	// 报告追踪信息
	if s.tracer != nil {
		s.tracer.Report(s.context)
	}
}

// Context 获取追踪上下文
func (s *Span) Context() *TraceContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回副本以防止并发修改
	ctx := *s.context
	ctx.Tags = make(map[string]string)
	for k, v := range s.context.Tags {
		ctx.Tags[k] = v
	}
	
	ctx.Logs = make([]TraceLog, len(s.context.Logs))
	copy(ctx.Logs, s.context.Logs)
	
	return &ctx
}

// MemoryTracer 内存追踪器实现
type MemoryTracer struct {
	mu     sync.RWMutex
	traces map[string][]*TraceContext
	config TracerConfig
}

// TracerConfig 追踪器配置
type TracerConfig struct {
	MaxTraces   int           `json:"max_traces"`
	MaxDuration time.Duration `json:"max_duration"`
	SampleRate  float64       `json:"sample_rate"`
}

// DefaultTracerConfig 默认追踪器配置
func DefaultTracerConfig() TracerConfig {
	return TracerConfig{
		MaxTraces:   1000,
		MaxDuration: time.Hour,
		SampleRate:  1.0,
	}
}

// NewMemoryTracer 创建内存追踪器
func NewMemoryTracer(config TracerConfig) *MemoryTracer {
	tracer := &MemoryTracer{
		traces: make(map[string][]*TraceContext),
		config: config,
	}
	
	// 启动清理goroutine
	go tracer.cleanup()
	
	return tracer
}

// StartSpan 开始新的跨度
func (mt *MemoryTracer) StartSpan(ctx context.Context, operation string) *Span {
	// 从上下文获取父跨度信息
	var traceID, parentID string
	if parentSpan := GetSpanFromContext(ctx); parentSpan != nil {
		traceID = parentSpan.context.TraceID
		parentID = parentSpan.context.SpanID
	} else {
		traceID = generateTraceID()
	}
	
	spanID := generateSpanID()
	return NewSpan(mt, traceID, spanID, parentID, operation)
}

// InjectHeaders 注入追踪头部
func (mt *MemoryTracer) InjectHeaders(span *Span, headers map[string]string) {
	if span == nil || span.context == nil {
		return
	}
	
	headers["X-Trace-ID"] = span.context.TraceID
	headers["X-Span-ID"] = span.context.SpanID
	if span.context.ParentID != "" {
		headers["X-Parent-ID"] = span.context.ParentID
	}
}

// ExtractHeaders 提取追踪头部
func (mt *MemoryTracer) ExtractHeaders(headers map[string]string) (*TraceContext, error) {
	traceID := headers["X-Trace-ID"]
	spanID := headers["X-Span-ID"]
	parentID := headers["X-Parent-ID"]
	
	if traceID == "" || spanID == "" {
		return nil, fmt.Errorf("missing trace headers")
	}
	
	return &TraceContext{
		TraceID:  traceID,
		SpanID:   spanID,
		ParentID: parentID,
	}, nil
}

// Report 报告追踪信息
func (mt *MemoryTracer) Report(trace *TraceContext) error {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	
	traces := mt.traces[trace.TraceID]
	traces = append(traces, trace)
	mt.traces[trace.TraceID] = traces
	
	// 限制追踪数量
	if len(mt.traces) > mt.config.MaxTraces {
		// 删除最旧的追踪
		oldestTime := time.Now()
		var oldestTraceID string
		for traceID, traceList := range mt.traces {
			if len(traceList) > 0 && traceList[0].StartTime.Before(oldestTime) {
				oldestTime = traceList[0].StartTime
				oldestTraceID = traceID
			}
		}
		if oldestTraceID != "" {
			delete(mt.traces, oldestTraceID)
		}
	}
	
	return nil
}

// GetTrace 获取追踪信息
func (mt *MemoryTracer) GetTrace(traceID string) []*TraceContext {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	traces := mt.traces[traceID]
	result := make([]*TraceContext, len(traces))
	copy(result, traces)
	return result
}

// GetAllTraces 获取所有追踪信息
func (mt *MemoryTracer) GetAllTraces() map[string][]*TraceContext {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	result := make(map[string][]*TraceContext)
	for traceID, traces := range mt.traces {
		result[traceID] = make([]*TraceContext, len(traces))
		copy(result[traceID], traces)
	}
	return result
}

// cleanup 清理过期追踪
func (mt *MemoryTracer) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mt.mu.Lock()
		cutoff := time.Now().Add(-mt.config.MaxDuration)
		for traceID, traces := range mt.traces {
			if len(traces) > 0 && traces[0].StartTime.Before(cutoff) {
				delete(mt.traces, traceID)
			}
		}
		mt.mu.Unlock()
	}
}

// TracingMiddleware 追踪中间件
func TracingMiddleware(tracer Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头提取追踪信息
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		
		var parentCtx *TraceContext
		if extractedCtx, err := tracer.ExtractHeaders(headers); err == nil {
			parentCtx = extractedCtx
		}
		
		// 创建新的跨度
		operation := fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		span := tracer.StartSpan(c.Request.Context(), operation)
		
		// 如果有父上下文，设置父ID
		if parentCtx != nil {
			span.context.TraceID = parentCtx.TraceID
			span.context.ParentID = parentCtx.SpanID
		}
		
		// 设置基本标签
		span.SetTag("http.method", c.Request.Method)
		span.SetTag("http.url", c.Request.URL.String())
		span.SetTag("http.remote_addr", c.ClientIP())
		span.SetTag("http.user_agent", c.Request.UserAgent())
		
		// 注入追踪头部到响应
		responseHeaders := make(map[string]string)
		tracer.InjectHeaders(span, responseHeaders)
		for key, value := range responseHeaders {
			c.Header(key, value)
		}
		
		// 将跨度存储到上下文
		ctx := SetSpanToContext(c.Request.Context(), span)
		c.Request = c.Request.WithContext(ctx)
		
		// 设置Gin上下文中的追踪信息
		c.Set("trace_id", span.context.TraceID)
		c.Set("span_id", span.context.SpanID)
		c.Set("span", span)
		
		defer func() {
			// 记录响应信息
			span.SetTag("http.status_code", fmt.Sprintf("%d", c.Writer.Status()))
			
			// 如果有错误，记录错误信息
			if c.Writer.Status() >= 400 {
				span.LogError(fmt.Sprintf("HTTP %d", c.Writer.Status()), map[string]string{
					"status_code": fmt.Sprintf("%d", c.Writer.Status()),
				})
			}
			
			span.Finish()
		}()
		
		c.Next()
	}
}

// 上下文键
type contextKey string

const spanContextKey contextKey = "span"

// SetSpanToContext 将跨度设置到上下文
func SetSpanToContext(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanContextKey, span)
}

// GetSpanFromContext 从上下文获取跨度
func GetSpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanContextKey).(*Span); ok {
		return span
	}
	return nil
}

// GetSpanFromGinContext 从Gin上下文获取跨度
func GetSpanFromGinContext(c *gin.Context) *Span {
	if span, exists := c.Get("span"); exists {
		if s, ok := span.(*Span); ok {
			return s
		}
	}
	return nil
}

// generateTraceID 生成追踪ID
func generateTraceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// generateSpanID 生成跨度ID
func generateSpanID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// TracingStats 追踪统计
type TracingStats struct {
	TotalTraces    int64         `json:"total_traces"`
	ActiveTraces   int64         `json:"active_traces"`
	AverageDuration time.Duration `json:"average_duration"`
	ErrorRate      float64       `json:"error_rate"`
}

// GetTracingStats 获取追踪统计
func (mt *MemoryTracer) GetTracingStats() TracingStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	
	var totalTraces int64
	var totalDuration time.Duration
	var errorCount int64
	
	for _, traces := range mt.traces {
		for _, trace := range traces {
			totalTraces++
			totalDuration += trace.Duration
			if trace.Status == TraceStatusError {
				errorCount++
			}
		}
	}
	
	var avgDuration time.Duration
	if totalTraces > 0 {
		avgDuration = totalDuration / time.Duration(totalTraces)
	}
	
	var errorRate float64
	if totalTraces > 0 {
		errorRate = float64(errorCount) / float64(totalTraces) * 100
	}
	
	return TracingStats{
		TotalTraces:     totalTraces,
		ActiveTraces:    int64(len(mt.traces)),
		AverageDuration: avgDuration,
		ErrorRate:       errorRate,
	}
}