package chi

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

// Event 事件接口
type Event interface {
	Type() string
	Data() interface{}
	Timestamp() time.Time
	Source() string
	ID() string
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	EventType      string                 `json:"type"`
	EventData      interface{}           `json:"data"`
	EventTimestamp time.Time             `json:"timestamp"`
	EventSource    string                `json:"source"`
	EventID        string                `json:"id"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

func (e *BaseEvent) Type() string        { return e.EventType }
func (e *BaseEvent) Data() interface{}   { return e.EventData }
func (e *BaseEvent) Timestamp() time.Time { return e.EventTimestamp }
func (e *BaseEvent) Source() string      { return e.EventSource }
func (e *BaseEvent) ID() string          { return e.EventID }

// NewEvent 创建新事件
func NewEvent(eventType, source string, data interface{}) *BaseEvent {
	return &BaseEvent{
		EventType:      eventType,
		EventData:      data,
		EventTimestamp: time.Now(),
		EventSource:    source,
		EventID:        generateEventID(),
		Metadata:       make(map[string]interface{}),
	}
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	SupportedTypes() []string
}

// EventHandlerFunc 事件处理函数
type EventHandlerFunc func(ctx context.Context, event Event) error

func (f EventHandlerFunc) Handle(ctx context.Context, event Event) error {
	return f(ctx, event)
}

func (f EventHandlerFunc) SupportedTypes() []string {
	return []string{"*"} // 支持所有类型
}

// EventBus 事件总线
type EventBus struct {
	mu                sync.RWMutex
	handlers          map[string][]EventHandler
	middlewares       []EventMiddleware
	asyncProcessors   []AsyncEventProcessor
	eventStore        EventStore
	publishedEvents   int64
	processedEvents   int64
	failedEvents      int64
}

// EventMiddleware 事件中间件
type EventMiddleware func(next EventHandler) EventHandler

// AsyncEventProcessor 异步事件处理器
type AsyncEventProcessor interface {
	Process(ctx context.Context, event Event) error
	BufferSize() int
	Start() error
	Stop() error
}

// EventStore 事件存储接口
type EventStore interface {
	Store(event Event) error
	Get(eventID string) (Event, error)
	List(filters EventFilters) ([]Event, error)
	Delete(eventID string) error
}

// EventFilters 事件过滤器
type EventFilters struct {
	EventType string
	Source    string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// NewEventBus 创建事件总线
func NewEventBus(opts ...EventBusOption) *EventBus {
	bus := &EventBus{
		handlers:        make(map[string][]EventHandler),
		middlewares:     make([]EventMiddleware, 0),
		asyncProcessors: make([]AsyncEventProcessor, 0),
		eventStore:      NewMemoryEventStore(),
	}

	for _, opt := range opts {
		opt(bus)
	}

	return bus
}

// EventBusOption 事件总线选项
type EventBusOption func(*EventBus)

// WithEventStore 设置事件存储
func WithEventStore(store EventStore) EventBusOption {
	return func(bus *EventBus) {
		bus.eventStore = store
	}
}

// WithAsyncProcessor 添加异步处理器
func WithAsyncProcessor(processor AsyncEventProcessor) EventBusOption {
	return func(bus *EventBus) {
		bus.asyncProcessors = append(bus.asyncProcessors, processor)
	}
}

// Subscribe 订阅事件
func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// SubscribeFunc 订阅事件（函数形式）
func (eb *EventBus) SubscribeFunc(eventType string, fn func(ctx context.Context, event Event) error) {
	eb.Subscribe(eventType, EventHandlerFunc(fn))
}

// Unsubscribe 取消订阅
func (eb *EventBus) Unsubscribe(eventType string, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	handlers := eb.handlers[eventType]
	for i, h := range handlers {
		if reflect.ValueOf(h).Pointer() == reflect.ValueOf(handler).Pointer() {
			eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

// Use 添加中间件
func (eb *EventBus) Use(middleware EventMiddleware) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.middlewares = append(eb.middlewares, middleware)
}

// Publish 发布事件
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	eb.publishedEvents++

	// 存储事件
	if eb.eventStore != nil {
		if err := eb.eventStore.Store(event); err != nil {
			// 记录错误但不影响事件处理
			fmt.Printf("Failed to store event: %v\n", err)
		}
	}

	// 同步处理
	if err := eb.handleEvent(ctx, event); err != nil {
		eb.failedEvents++
		return err
	}

	// 异步处理
	for _, processor := range eb.asyncProcessors {
		go func(p AsyncEventProcessor) {
			if err := p.Process(ctx, event); err != nil {
				fmt.Printf("Async processor error: %v\n", err)
			}
		}(processor)
	}

	eb.processedEvents++
	return nil
}

// handleEvent 处理事件
func (eb *EventBus) handleEvent(ctx context.Context, event Event) error {
	handlers := make([]EventHandler, 0)

	// 获取特定类型的处理器
	if specificHandlers, exists := eb.handlers[event.Type()]; exists {
		handlers = append(handlers, specificHandlers...)
	}

	// 获取通用处理器
	if wildcardHandlers, exists := eb.handlers["*"]; exists {
		handlers = append(handlers, wildcardHandlers...)
	}

	// 应用中间件链
	for _, handler := range handlers {
		finalHandler := handler
		for i := len(eb.middlewares) - 1; i >= 0; i-- {
			finalHandler = eb.middlewares[i](finalHandler)
		}

		if err := finalHandler.Handle(ctx, event); err != nil {
			return fmt.Errorf("handler error for event %s: %w", event.ID(), err)
		}
	}

	return nil
}

// Stats 获取统计信息
func (eb *EventBus) Stats() EventBusStats {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	handlerCount := 0
	for _, handlers := range eb.handlers {
		handlerCount += len(handlers)
	}

	return EventBusStats{
		PublishedEvents: eb.publishedEvents,
		ProcessedEvents: eb.processedEvents,
		FailedEvents:    eb.failedEvents,
		HandlerCount:    int64(handlerCount),
		MiddlewareCount: int64(len(eb.middlewares)),
	}
}

// EventBusStats 事件总线统计
type EventBusStats struct {
	PublishedEvents int64 `json:"published_events"`
	ProcessedEvents int64 `json:"processed_events"`
	FailedEvents    int64 `json:"failed_events"`
	HandlerCount    int64 `json:"handler_count"`
	MiddlewareCount int64 `json:"middleware_count"`
}

// MemoryEventStore 内存事件存储
type MemoryEventStore struct {
	mu     sync.RWMutex
	events map[string]Event
	order  []string
	maxSize int
}

// NewMemoryEventStore 创建内存事件存储
func NewMemoryEventStore() *MemoryEventStore {
	return &MemoryEventStore{
		events:  make(map[string]Event),
		order:   make([]string, 0),
		maxSize: 10000,
	}
}

// Store 存储事件
func (mes *MemoryEventStore) Store(event Event) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	mes.events[event.ID()] = event
	mes.order = append(mes.order, event.ID())

	// 限制大小
	if len(mes.order) > mes.maxSize {
		oldestID := mes.order[0]
		delete(mes.events, oldestID)
		mes.order = mes.order[1:]
	}

	return nil
}

// Get 获取事件
func (mes *MemoryEventStore) Get(eventID string) (Event, error) {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	event, exists := mes.events[eventID]
	if !exists {
		return nil, fmt.Errorf("event not found: %s", eventID)
	}

	return event, nil
}

// List 列出事件
func (mes *MemoryEventStore) List(filters EventFilters) ([]Event, error) {
	mes.mu.RLock()
	defer mes.mu.RUnlock()

	result := make([]Event, 0)
	count := 0

	for i := len(mes.order) - 1; i >= 0; i-- {
		if filters.Offset > 0 && count < filters.Offset {
			count++
			continue
		}

		if filters.Limit > 0 && len(result) >= filters.Limit {
			break
		}

		eventID := mes.order[i]
		event := mes.events[eventID]

		// 应用过滤器
		if filters.EventType != "" && event.Type() != filters.EventType {
			continue
		}

		if filters.Source != "" && event.Source() != filters.Source {
			continue
		}

		if filters.StartTime != nil && event.Timestamp().Before(*filters.StartTime) {
			continue
		}

		if filters.EndTime != nil && event.Timestamp().After(*filters.EndTime) {
			continue
		}

		result = append(result, event)
		count++
	}

	return result, nil
}

// Delete 删除事件
func (mes *MemoryEventStore) Delete(eventID string) error {
	mes.mu.Lock()
	defer mes.mu.Unlock()

	if _, exists := mes.events[eventID]; !exists {
		return fmt.Errorf("event not found: %s", eventID)
	}

	delete(mes.events, eventID)

	// 从顺序中移除
	for i, id := range mes.order {
		if id == eventID {
			mes.order = append(mes.order[:i], mes.order[i+1:]...)
			break
		}
	}

	return nil
}

// BufferedAsyncProcessor 缓冲异步处理器
type BufferedAsyncProcessor struct {
	buffer   chan Event
	handler  EventHandler
	stopCh   chan struct{}
	wg       sync.WaitGroup
	workers  int
}

// NewBufferedAsyncProcessor 创建缓冲异步处理器
func NewBufferedAsyncProcessor(bufferSize, workers int, handler EventHandler) *BufferedAsyncProcessor {
	return &BufferedAsyncProcessor{
		buffer:  make(chan Event, bufferSize),
		handler: handler,
		stopCh:  make(chan struct{}),
		workers: workers,
	}
}

// BufferSize 获取缓冲区大小
func (bap *BufferedAsyncProcessor) BufferSize() int {
	return cap(bap.buffer)
}

// Start 启动处理器
func (bap *BufferedAsyncProcessor) Start() error {
	for i := 0; i < bap.workers; i++ {
		bap.wg.Add(1)
		go bap.worker()
	}
	return nil
}

// Stop 停止处理器
func (bap *BufferedAsyncProcessor) Stop() error {
	close(bap.stopCh)
	bap.wg.Wait()
	return nil
}

// Process 处理事件
func (bap *BufferedAsyncProcessor) Process(ctx context.Context, event Event) error {
	select {
	case bap.buffer <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("buffer full")
	}
}

// worker 工作协程
func (bap *BufferedAsyncProcessor) worker() {
	defer bap.wg.Done()

	for {
		select {
		case event := <-bap.buffer:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := bap.handler.Handle(ctx, event); err != nil {
				fmt.Printf("Async handler error: %v\n", err)
			}
			cancel()
		case <-bap.stopCh:
			return
		}
	}
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}

// 预定义事件类型
const (
	EventTypeAppStarted    = "app.started"
	EventTypeAppStopped    = "app.stopped"
	EventTypeRequestStart  = "request.start"
	EventTypeRequestEnd    = "request.end"
	EventTypeError         = "error"
	EventTypeMetricUpdated = "metric.updated"
	EventTypeConfigChanged = "config.changed"
	EventTypePluginLoaded  = "plugin.loaded"
	EventTypePluginUnloaded = "plugin.unloaded"
)

// 事件中间件
func LoggingEventMiddleware(logger Logger) EventMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event Event) error {
			start := time.Now()
			err := next.Handle(ctx, event)
			duration := time.Since(start)

			if logger != nil {
				if err != nil {
					logger.Error("Event handling failed", map[string]interface{}{
						"event_id":   event.ID(),
						"event_type": event.Type(),
						"duration":   duration,
						"error":      err.Error(),
					})
				} else {
					logger.Info("Event handled", map[string]interface{}{
						"event_id":   event.ID(),
						"event_type": event.Type(),
						"duration":   duration,
					})
				}
			}

			return err
		})
	}
}

// MetricsEventMiddleware 指标事件中间件
func MetricsEventMiddleware(metrics *Metrics) EventMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event Event) error {
			start := time.Now()
			err := next.Handle(ctx, event)
			_ = time.Since(start) // duration calculation for future metrics integration

			// TODO: Add proper metrics integration when metrics interface is extended

			return err
		})
	}
}

// RetryEventMiddleware 重试事件中间件
func RetryEventMiddleware(maxRetries int) EventMiddleware {
	return func(next EventHandler) EventHandler {
		return EventHandlerFunc(func(ctx context.Context, event Event) error {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				err := next.Handle(ctx, event)
				if err == nil {
					return nil
				}
				lastErr = err

				if i < maxRetries {
					// 指数退避
					backoff := time.Duration(1<<uint(i)) * time.Second
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			return lastErr
		})
	}
}