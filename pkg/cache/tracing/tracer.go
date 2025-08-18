package tracing

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer Redis链路追踪器
type Tracer struct {
	tracer      trace.Tracer
	serviceName string
	version     string
	enabled     bool
}

// NewTracer 创建新的追踪器
func NewTracer(serviceName, version string, enabled bool) *Tracer {
	return &Tracer{
		tracer:      otel.Tracer(serviceName),
		serviceName: serviceName,
		version:     version,
		enabled:     enabled,
	}
}

// StartSpan 开始一个新的追踪span
func (t *Tracer) StartSpan(ctx context.Context, operation string, key string, args ...interface{}) (context.Context, trace.Span) {
	if !t.enabled {
		return ctx, trace.SpanFromContext(ctx)
	}

	spanName := fmt.Sprintf("redis.%s", operation)
	ctx, span := t.tracer.Start(ctx, spanName)

	// 设置基础属性
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", operation),
		attribute.String("service.name", t.serviceName),
		attribute.String("service.version", t.version),
	)

	// 设置键名
	if key != "" {
		span.SetAttributes(attribute.String("db.redis.key", key))
	}

	// 设置参数
	if len(args) > 0 {
		span.SetAttributes(attribute.String("db.redis.args", fmt.Sprintf("%v", args)))
	}

	return ctx, span
}

// FinishSpan 结束追踪span
func (t *Tracer) FinishSpan(span trace.Span, err error, result interface{}) {
	if !t.enabled {
		return
	}

	defer span.End()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.Bool("db.redis.success", false))
	} else {
		span.SetStatus(codes.Ok, "")
		span.SetAttributes(attribute.Bool("db.redis.success", true))
	}

	// 记录结果信息
	if result != nil {
		span.SetAttributes(attribute.String("db.redis.result", fmt.Sprintf("%v", result)))
	}
}

// TraceOperation 追踪Redis操作的辅助函数
func (t *Tracer) TraceOperation(ctx context.Context, operation, key string, fn func(ctx context.Context) (interface{}, error), args ...interface{}) (interface{}, error) {
	if !t.enabled {
		return fn(ctx)
	}

	start := time.Now()
	ctx, span := t.StartSpan(ctx, operation, key, args...)

	result, err := fn(ctx)

	// 记录执行时间
	duration := time.Since(start)
	span.SetAttributes(attribute.Int64("db.redis.duration_ms", duration.Milliseconds()))

	t.FinishSpan(span, err, result)
	return result, err
}

// TraceBatchOperation 追踪批量操作
func (t *Tracer) TraceBatchOperation(ctx context.Context, operation string, keys []string, fn func(ctx context.Context) (interface{}, error), args ...interface{}) (interface{}, error) {
	if !t.enabled {
		return fn(ctx)
	}

	start := time.Now()
	spanName := fmt.Sprintf("redis.%s.batch", operation)
	ctx, span := t.tracer.Start(ctx, spanName)

	// 设置批量操作属性
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", operation),
		attribute.String("service.name", t.serviceName),
		attribute.String("service.version", t.version),
		attribute.Int("db.redis.batch_size", len(keys)),
		attribute.StringSlice("db.redis.keys", keys),
	)

	if len(args) > 0 {
		span.SetAttributes(attribute.String("db.redis.args", fmt.Sprintf("%v", args)))
	}

	result, err := fn(ctx)

	// 记录执行时间
	duration := time.Since(start)
	span.SetAttributes(attribute.Int64("db.redis.duration_ms", duration.Milliseconds()))

	t.FinishSpan(span, err, result)
	return result, err
}

// TraceLuaScript 追踪Lua脚本执行
func (t *Tracer) TraceLuaScript(ctx context.Context, script string, keys []string, args []interface{}, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	if !t.enabled {
		return fn(ctx)
	}

	start := time.Now()
	ctx, span := t.tracer.Start(ctx, "redis.eval")

	// 设置Lua脚本属性
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "eval"),
		attribute.String("service.name", t.serviceName),
		attribute.String("service.version", t.version),
		attribute.String("db.redis.script", script),
		attribute.StringSlice("db.redis.keys", keys),
		attribute.String("db.redis.args", fmt.Sprintf("%v", args)),
	)

	result, err := fn(ctx)

	// 记录执行时间
	duration := time.Since(start)
	span.SetAttributes(attribute.Int64("db.redis.duration_ms", duration.Milliseconds()))

	t.FinishSpan(span, err, result)
	return result, err
}

// AddEvent 添加事件到当前span
func (t *Tracer) AddEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	if !t.enabled {
		return
	}

	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attributes...))
}

// SetAttribute 设置span属性
func (t *Tracer) SetAttribute(ctx context.Context, key string, value interface{}) {
	if !t.enabled {
		return
	}

	span := trace.SpanFromContext(ctx)
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	default:
		span.SetAttributes(attribute.String(key, fmt.Sprintf("%v", v)))
	}
}
