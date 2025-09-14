package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SimpleEventBus 简单事件总线实现
type SimpleEventBus struct {
	subscribers map[EventType][]*eventSubscription
	mutex       sync.RWMutex
	closed      bool
}

type eventSubscription struct {
	id        string
	eventType EventType
	handler   EventHandler
	active    bool
	mutex     sync.RWMutex
}

// NewSimpleEventBus 创建简单事件总线
func NewSimpleEventBus() EventBus {
	return &SimpleEventBus{
		subscribers: make(map[EventType][]*eventSubscription),
	}
}

func (e *SimpleEventBus) Publish(event Event) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	
	if e.closed {
		return
	}
	
	subscribers, exists := e.subscribers[event.Type]
	if !exists || len(subscribers) == 0 {
		return
	}
	
	// 异步发送事件给所有订阅者
	for _, sub := range subscribers {
		if sub.IsActive() {
			go func(subscription *eventSubscription) {
				defer func() {
					if r := recover(); r != nil {
						// 忽略处理器panic
					}
				}()
				subscription.handler(event)
			}(sub)
		}
	}
}

func (e *SimpleEventBus) Subscribe(eventType EventType, handler EventHandler) Subscription {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.closed {
		return nil
	}
	
	subscription := &eventSubscription{
		id:        generateSubscriptionID(),
		eventType: eventType,
		handler:   handler,
		active:    true,
	}
	
	if e.subscribers[eventType] == nil {
		e.subscribers[eventType] = make([]*eventSubscription, 0)
	}
	
	e.subscribers[eventType] = append(e.subscribers[eventType], subscription)
	
	return subscription
}

func (e *SimpleEventBus) Unsubscribe(subscription Subscription) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.closed || subscription == nil {
		return
	}
	
	eventType := subscription.EventType()
	subscribers, exists := e.subscribers[eventType]
	if !exists {
		return
	}
	
	// 查找并移除订阅
	for i, sub := range subscribers {
		if sub.ID() == subscription.ID() {
			sub.Cancel()
			e.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
	
	// 如果没有更多订阅者，清理事件类型
	if len(e.subscribers[eventType]) == 0 {
		delete(e.subscribers, eventType)
	}
}

func (e *SimpleEventBus) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if e.closed {
		return nil
	}
	
	e.closed = true
	
	// 取消所有订阅
	for eventType, subscribers := range e.subscribers {
		for _, sub := range subscribers {
			sub.Cancel()
		}
		delete(e.subscribers, eventType)
	}
	
	return nil
}

// eventSubscription 方法实现
func (s *eventSubscription) ID() string {
	return s.id
}

func (s *eventSubscription) EventType() EventType {
	return s.eventType
}

func (s *eventSubscription) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.active
}

func (s *eventSubscription) Cancel() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.active = false
}

// generateSubscriptionID 生成订阅ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

// NoOpEventBus 空事件总线实现
type NoOpEventBus struct{}

func NewNoOpEventBus() EventBus {
	return &NoOpEventBus{}
}

func (n *NoOpEventBus) Publish(event Event) {}

func (n *NoOpEventBus) Subscribe(eventType EventType, handler EventHandler) Subscription {
	return &noOpSubscription{eventType: eventType}
}

func (n *NoOpEventBus) Unsubscribe(subscription Subscription) {}

func (n *NoOpEventBus) Close() error {
	return nil
}

// noOpSubscription 空订阅实现
type noOpSubscription struct {
	eventType EventType
}

func (n *noOpSubscription) ID() string {
	return "noop"
}

func (n *noOpSubscription) EventType() EventType {
	return n.eventType
}

func (n *noOpSubscription) IsActive() bool {
	return false
}

func (n *noOpSubscription) Cancel() {}

// SimpleHookManager 简单钩子管理器实现
type SimpleHookManager struct {
	hooks map[HookType][]HookFunc
	mutex sync.RWMutex
}

// NewSimpleHookManager 创建简单钩子管理器
func NewSimpleHookManager() HookManager {
	return &SimpleHookManager{
		hooks: make(map[HookType][]HookFunc),
	}
}

func (h *SimpleHookManager) RegisterHook(hookType HookType, hookFunc HookFunc) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	if h.hooks[hookType] == nil {
		h.hooks[hookType] = make([]HookFunc, 0)
	}
	
	h.hooks[hookType] = append(h.hooks[hookType], hookFunc)
	return nil
}

func (h *SimpleHookManager) ExecuteHook(ctx context.Context, hookType HookType, data map[string]interface{}) error {
	h.mutex.RLock()
	hooks := h.hooks[hookType]
	h.mutex.RUnlock()
	
	if len(hooks) == 0 {
		return nil
	}
	
	// 顺序执行所有钩子
	for _, hook := range hooks {
		if err := hook(ctx, data); err != nil {
			return err
		}
		
		// 检查上下文取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	
	return nil
}

func (h *SimpleHookManager) RemoveHook(hookType HookType, hookFunc HookFunc) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	hooks := h.hooks[hookType]
	if len(hooks) == 0 {
		return nil
	}
	
	// 由于函数不能直接比较，这里简化处理
	// 实际实现中可能需要使用标识符或其他方式
	return fmt.Errorf("hook removal not implemented for function comparison")
}

func (h *SimpleHookManager) ClearHooks(hookType HookType) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	
	delete(h.hooks, hookType)
}

func (h *SimpleHookManager) GetHookCount(hookType HookType) int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	return len(h.hooks[hookType])
}