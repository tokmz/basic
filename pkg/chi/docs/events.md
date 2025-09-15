# 事件驱动架构

## 概述

Chi 框架提供了强大的事件驱动架构支持，通过企业级事件总线实现服务间的解耦通信，支持同步和异步事件处理，提供完整的事件生命周期管理和可靠的事件传递机制。

## 事件系统架构

### 整体架构图
```
┌─────────────────────────────────────────────────────────────────┐
│                        Event System                             │
│                                                                 │
│  ┌───────────────┐   ┌───────────────┐   ┌──────────────────┐   │
│  │  Event        │   │  Event        │   │  Event           │   │
│  │  Producer     │──▶│  Bus          │◄──│  Consumer        │   │
│  └───────────────┘   └───────────────┘   └──────────────────┘   │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                  Event Router                               │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │ │
│  │  │   Topic     │ │   Filter    │ │    Subscription     │   │ │
│  │  │  Manager    │ │   Engine    │ │     Manager         │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────────┘   │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                            │                                    │
│                            ▼                                    │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                 Event Storage                               │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐   │ │
│  │  │  Memory     │ │  Database   │ │    Message          │   │ │
│  │  │  Store      │ │   Store     │ │    Queue            │   │ │
│  │  └─────────────┘ └─────────────┘ └─────────────────────┘   │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

1. **Event Bus**: 事件总线，协调事件的发布和订阅
2. **Event Router**: 事件路由器，负责事件的分发和过滤
3. **Topic Manager**: 主题管理器，管理事件主题
4. **Subscription Manager**: 订阅管理器，管理事件订阅关系
5. **Event Storage**: 事件存储，提供事件持久化能力

## 事件定义

### 基础事件结构

```go
package events

import (
    "time"
    "context"
)

// Event 基础事件接口
type Event interface {
    // 获取事件ID
    GetID() string

    // 获取事件类型
    GetType() string

    // 获取事件主题
    GetTopic() string

    // 获取事件数据
    GetData() interface{}

    // 获取事件元数据
    GetMetadata() map[string]interface{}

    // 获取事件时间戳
    GetTimestamp() time.Time

    // 获取事件版本
    GetVersion() string
}

// BaseEvent 基础事件实现
type BaseEvent struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`
    Topic     string                 `json:"topic"`
    Data      interface{}            `json:"data"`
    Metadata  map[string]interface{} `json:"metadata"`
    Timestamp time.Time              `json:"timestamp"`
    Version   string                 `json:"version"`
}

func (e *BaseEvent) GetID() string                        { return e.ID }
func (e *BaseEvent) GetType() string                      { return e.Type }
func (e *BaseEvent) GetTopic() string                     { return e.Topic }
func (e *BaseEvent) GetData() interface{}                 { return e.Data }
func (e *BaseEvent) GetMetadata() map[string]interface{}  { return e.Metadata }
func (e *BaseEvent) GetTimestamp() time.Time              { return e.Timestamp }
func (e *BaseEvent) GetVersion() string                   { return e.Version }

// 事件构建器
func NewEvent(eventType, topic string, data interface{}) *BaseEvent {
    return &BaseEvent{
        ID:        generateEventID(),
        Type:      eventType,
        Topic:     topic,
        Data:      data,
        Metadata:  make(map[string]interface{}),
        Timestamp: time.Now(),
        Version:   "1.0",
    }
}
```

### 业务事件示例

```go
// 用户相关事件
type UserEvent struct {
    *BaseEvent
    UserID string `json:"user_id"`
}

func NewUserEvent(eventType string, userID string, data interface{}) *UserEvent {
    return &UserEvent{
        BaseEvent: NewEvent(eventType, "user", data),
        UserID:    userID,
    }
}

// 用户创建事件
type UserCreatedEvent struct {
    *UserEvent
    User User `json:"user"`
}

func NewUserCreatedEvent(user User) *UserCreatedEvent {
    return &UserCreatedEvent{
        UserEvent: NewUserEvent("user.created", user.ID, user),
        User:      user,
    }
}

// 订单相关事件
type OrderEvent struct {
    *BaseEvent
    OrderID string `json:"order_id"`
}

func NewOrderEvent(eventType string, orderID string, data interface{}) *OrderEvent {
    return &OrderEvent{
        BaseEvent: NewEvent(eventType, "order", data),
        OrderID:   orderID,
    }
}

// 订单创建事件
type OrderCreatedEvent struct {
    *OrderEvent
    Order Order `json:"order"`
}

func NewOrderCreatedEvent(order Order) *OrderCreatedEvent {
    return &OrderCreatedEvent{
        OrderEvent: NewOrderEvent("order.created", order.ID, order),
        Order:      order,
    }
}
```

## 事件发布

### 基础事件发布

```go
package main

import (
    "context"
    "github.com/your-org/basic/pkg/chi"
    "github.com/your-org/basic/pkg/chi/events"
)

func publishUserCreatedEvent(user User) {
    // 获取事件总线
    eventBus := chi.GetEventBus()

    // 创建用户创建事件
    event := events.NewUserCreatedEvent(user)

    // 发布事件
    ctx := context.Background()
    err := eventBus.Publish(ctx, event)
    if err != nil {
        log.Error("Failed to publish user created event:", err)
        return
    }

    log.Info("User created event published successfully")
}

// 批量发布事件
func publishBatchEvents(events []events.Event) {
    eventBus := chi.GetEventBus()

    ctx := context.Background()
    err := eventBus.PublishBatch(ctx, events)
    if err != nil {
        log.Error("Failed to publish batch events:", err)
        return
    }

    log.Info("Batch events published successfully")
}
```

### 异步事件发布

```go
// 异步发布事件
func publishEventAsync(event events.Event) {
    eventBus := chi.GetEventBus()

    // 异步发布，不阻塞当前goroutine
    go func() {
        ctx := context.Background()
        err := eventBus.Publish(ctx, event)
        if err != nil {
            log.Error("Failed to publish event asynchronously:", err)
        }
    }()
}

// 使用channel进行异步发布
func setupAsyncEventPublisher() {
    eventBus := chi.GetEventBus()
    eventChannel := make(chan events.Event, 1000)

    // 启动事件发布协程
    go func() {
        for event := range eventChannel {
            ctx := context.Background()
            err := eventBus.Publish(ctx, event)
            if err != nil {
                log.Error("Failed to publish event from channel:", err)
                // 可以实现重试逻辑或死信队列
            }
        }
    }()

    // 发布事件到channel
    chi.SetAsyncEventChannel(eventChannel)
}

// 使用异步channel发布事件
func publishToChannel(event events.Event) {
    eventChannel := chi.GetAsyncEventChannel()
    select {
    case eventChannel <- event:
        // 事件已加入发布队列
    default:
        log.Warn("Event channel is full, event dropped")
    }
}
```

### 事务性事件发布

```go
// 事务性事件发布
func createUserWithEvents(user User) error {
    // 开始数据库事务
    tx, err := db.BeginTx(context.Background(), nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 创建用户
    err = createUser(tx, user)
    if err != nil {
        return err
    }

    // 在事务中记录事件
    event := events.NewUserCreatedEvent(user)
    err = recordEventInTransaction(tx, event)
    if err != nil {
        return err
    }

    // 提交事务
    err = tx.Commit()
    if err != nil {
        return err
    }

    // 事务成功后发布事件
    eventBus := chi.GetEventBus()
    ctx := context.Background()
    return eventBus.Publish(ctx, event)
}

// Outbox模式实现
type EventOutbox struct {
    db          Database
    eventBus    events.EventBus
    pollInterval time.Duration
}

func NewEventOutbox(db Database, eventBus events.EventBus) *EventOutbox {
    return &EventOutbox{
        db:           db,
        eventBus:     eventBus,
        pollInterval: 5 * time.Second,
    }
}

func (eo *EventOutbox) Start() {
    go eo.pollAndPublishEvents()
}

func (eo *EventOutbox) pollAndPublishEvents() {
    ticker := time.NewTicker(eo.pollInterval)
    defer ticker.Stop()

    for range ticker.C {
        events, err := eo.db.GetPendingEvents(100)
        if err != nil {
            log.Error("Failed to get pending events:", err)
            continue
        }

        for _, event := range events {
            ctx := context.Background()
            err := eo.eventBus.Publish(ctx, event)
            if err != nil {
                log.Error("Failed to publish event:", err)
                continue
            }

            // 标记事件为已发布
            eo.db.MarkEventAsPublished(event.GetID())
        }
    }
}
```

## 事件订阅

### 基础事件订阅

```go
// 事件处理器接口
type EventHandler interface {
    Handle(ctx context.Context, event events.Event) error
    CanHandle(event events.Event) bool
}

// 用户创建事件处理器
type UserCreatedHandler struct {
    emailService EmailService
    logger       Logger
}

func (h *UserCreatedHandler) CanHandle(event events.Event) bool {
    return event.GetType() == "user.created"
}

func (h *UserCreatedHandler) Handle(ctx context.Context, event events.Event) error {
    userEvent, ok := event.(*events.UserCreatedEvent)
    if !ok {
        return errors.New("invalid event type")
    }

    // 发送欢迎邮件
    err := h.emailService.SendWelcomeEmail(userEvent.User)
    if err != nil {
        h.logger.Error("Failed to send welcome email:", err)
        return err
    }

    h.logger.Info("Welcome email sent for user:", userEvent.User.ID)
    return nil
}

// 注册事件处理器
func setupEventHandlers() {
    eventBus := chi.GetEventBus()

    // 注册用户创建事件处理器
    userCreatedHandler := &UserCreatedHandler{
        emailService: getEmailService(),
        logger:       chi.GetLogger(),
    }

    err := eventBus.Subscribe("user.created", userCreatedHandler)
    if err != nil {
        log.Fatal("Failed to subscribe to user.created events:", err)
    }
}
```

### 模式匹配订阅

```go
// 模式匹配事件处理器
type PatternHandler struct {
    pattern string
    handler func(ctx context.Context, event events.Event) error
}

func NewPatternHandler(pattern string, handler func(ctx context.Context, event events.Event) error) *PatternHandler {
    return &PatternHandler{
        pattern: pattern,
        handler: handler,
    }
}

func (h *PatternHandler) CanHandle(event events.Event) bool {
    matched, _ := filepath.Match(h.pattern, event.GetType())
    return matched
}

func (h *PatternHandler) Handle(ctx context.Context, event events.Event) error {
    return h.handler(ctx, event)
}

// 注册模式匹配处理器
func setupPatternHandlers() {
    eventBus := chi.GetEventBus()

    // 处理所有用户相关事件
    userPatternHandler := NewPatternHandler("user.*", func(ctx context.Context, event events.Event) error {
        log.Info("Handling user event:", event.GetType())
        // 用户事件通用处理逻辑
        return nil
    })

    eventBus.Subscribe("user.*", userPatternHandler)

    // 处理所有创建事件
    createdPatternHandler := NewPatternHandler("*.created", func(ctx context.Context, event events.Event) error {
        log.Info("Handling created event:", event.GetType())
        // 创建事件通用处理逻辑
        return auditService.RecordCreatedEvent(event)
    })

    eventBus.Subscribe("*.created", createdPatternHandler)
}
```

### 异步事件处理

```go
// 异步事件处理器
type AsyncEventHandler struct {
    baseHandler EventHandler
    workerPool  *WorkerPool
}

func NewAsyncEventHandler(baseHandler EventHandler, workerCount int) *AsyncEventHandler {
    return &AsyncEventHandler{
        baseHandler: baseHandler,
        workerPool:  NewWorkerPool(workerCount),
    }
}

func (h *AsyncEventHandler) CanHandle(event events.Event) bool {
    return h.baseHandler.CanHandle(event)
}

func (h *AsyncEventHandler) Handle(ctx context.Context, event events.Event) error {
    // 将事件处理任务提交给工作池
    task := &EventTask{
        handler: h.baseHandler,
        event:   event,
        ctx:     ctx,
    }

    return h.workerPool.Submit(task)
}

// 事件处理任务
type EventTask struct {
    handler EventHandler
    event   events.Event
    ctx     context.Context
}

func (t *EventTask) Execute() error {
    return t.handler.Handle(t.ctx, t.event)
}

// 工作池实现
type WorkerPool struct {
    tasks   chan Task
    workers int
}

func NewWorkerPool(workers int) *WorkerPool {
    pool := &WorkerPool{
        tasks:   make(chan Task, workers*2),
        workers: workers,
    }
    pool.start()
    return pool
}

func (wp *WorkerPool) start() {
    for i := 0; i < wp.workers; i++ {
        go wp.worker()
    }
}

func (wp *WorkerPool) worker() {
    for task := range wp.tasks {
        err := task.Execute()
        if err != nil {
            log.Error("Task execution failed:", err)
        }
    }
}

func (wp *WorkerPool) Submit(task Task) error {
    select {
    case wp.tasks <- task:
        return nil
    default:
        return errors.New("worker pool is full")
    }
}
```

## 事件过滤和路由

### 事件过滤器

```go
// 事件过滤器接口
type EventFilter interface {
    Filter(event events.Event) bool
}

// 主题过滤器
type TopicFilter struct {
    allowedTopics map[string]bool
}

func NewTopicFilter(topics ...string) *TopicFilter {
    filter := &TopicFilter{
        allowedTopics: make(map[string]bool),
    }
    for _, topic := range topics {
        filter.allowedTopics[topic] = true
    }
    return filter
}

func (f *TopicFilter) Filter(event events.Event) bool {
    return f.allowedTopics[event.GetTopic()]
}

// 元数据过滤器
type MetadataFilter struct {
    requiredMetadata map[string]interface{}
}

func NewMetadataFilter(metadata map[string]interface{}) *MetadataFilter {
    return &MetadataFilter{
        requiredMetadata: metadata,
    }
}

func (f *MetadataFilter) Filter(event events.Event) bool {
    eventMetadata := event.GetMetadata()
    for key, value := range f.requiredMetadata {
        if eventValue, exists := eventMetadata[key]; !exists || eventValue != value {
            return false
        }
    }
    return true
}

// 复合过滤器
type CompositeFilter struct {
    filters []EventFilter
    mode    string // "AND" 或 "OR"
}

func NewCompositeFilter(mode string, filters ...EventFilter) *CompositeFilter {
    return &CompositeFilter{
        filters: filters,
        mode:    mode,
    }
}

func (f *CompositeFilter) Filter(event events.Event) bool {
    if f.mode == "AND" {
        for _, filter := range f.filters {
            if !filter.Filter(event) {
                return false
            }
        }
        return true
    } else { // OR
        for _, filter := range f.filters {
            if filter.Filter(event) {
                return true
            }
        }
        return false
    }
}
```

### 事件路由器

```go
// 事件路由规则
type RouteRule struct {
    Filter  EventFilter
    Handler EventHandler
    Priority int
}

// 事件路由器
type EventRouter struct {
    rules []RouteRule
    mutex sync.RWMutex
}

func NewEventRouter() *EventRouter {
    return &EventRouter{
        rules: make([]RouteRule, 0),
    }
}

func (r *EventRouter) AddRule(rule RouteRule) {
    r.mutex.Lock()
    defer r.mutex.Unlock()

    r.rules = append(r.rules, rule)
    // 按优先级排序
    sort.Slice(r.rules, func(i, j int) bool {
        return r.rules[i].Priority > r.rules[j].Priority
    })
}

func (r *EventRouter) Route(ctx context.Context, event events.Event) error {
    r.mutex.RLock()
    defer r.mutex.RUnlock()

    var errors []error

    for _, rule := range r.rules {
        if rule.Filter.Filter(event) {
            err := rule.Handler.Handle(ctx, event)
            if err != nil {
                errors = append(errors, err)
            }
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("routing errors: %v", errors)
    }

    return nil
}

// 设置路由规则
func setupEventRouting() {
    router := chi.GetEventRouter()

    // 高优先级：安全相关事件
    securityFilter := NewTopicFilter("security")
    securityHandler := NewSecurityEventHandler()
    router.AddRule(RouteRule{
        Filter:   securityFilter,
        Handler:  securityHandler,
        Priority: 100,
    })

    // 中优先级：业务事件
    businessFilter := NewTopicFilter("user", "order", "payment")
    businessHandler := NewBusinessEventHandler()
    router.AddRule(RouteRule{
        Filter:   businessFilter,
        Handler:  businessHandler,
        Priority: 50,
    })

    // 低优先级：日志记录
    allEventsFilter := &AllEventsFilter{}
    logHandler := NewLogEventHandler()
    router.AddRule(RouteRule{
        Filter:   allEventsFilter,
        Handler:  logHandler,
        Priority: 1,
    })
}
```

## 事件存储和重放

### 事件存储接口

```go
// 事件存储接口
type EventStore interface {
    Store(ctx context.Context, event events.Event) error
    Load(ctx context.Context, id string) (events.Event, error)
    LoadByTopic(ctx context.Context, topic string, offset, limit int) ([]events.Event, error)
    LoadByTimeRange(ctx context.Context, start, end time.Time) ([]events.Event, error)
    Delete(ctx context.Context, id string) error
}

// 内存事件存储
type MemoryEventStore struct {
    events map[string]events.Event
    topics map[string][]string
    mutex  sync.RWMutex
}

func NewMemoryEventStore() *MemoryEventStore {
    return &MemoryEventStore{
        events: make(map[string]events.Event),
        topics: make(map[string][]string),
    }
}

func (s *MemoryEventStore) Store(ctx context.Context, event events.Event) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()

    eventID := event.GetID()
    topic := event.GetTopic()

    s.events[eventID] = event

    if _, exists := s.topics[topic]; !exists {
        s.topics[topic] = make([]string, 0)
    }
    s.topics[topic] = append(s.topics[topic], eventID)

    return nil
}

func (s *MemoryEventStore) Load(ctx context.Context, id string) (events.Event, error) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    event, exists := s.events[id]
    if !exists {
        return nil, errors.New("event not found")
    }

    return event, nil
}

func (s *MemoryEventStore) LoadByTopic(ctx context.Context, topic string, offset, limit int) ([]events.Event, error) {
    s.mutex.RLock()
    defer s.mutex.RUnlock()

    eventIDs, exists := s.topics[topic]
    if !exists {
        return []events.Event{}, nil
    }

    start := offset
    end := offset + limit
    if start >= len(eventIDs) {
        return []events.Event{}, nil
    }
    if end > len(eventIDs) {
        end = len(eventIDs)
    }

    result := make([]events.Event, 0, end-start)
    for _, eventID := range eventIDs[start:end] {
        if event, exists := s.events[eventID]; exists {
            result = append(result, event)
        }
    }

    return result, nil
}
```

### 数据库事件存储

```go
// 数据库事件存储
type DatabaseEventStore struct {
    db     *sql.DB
    logger Logger
}

func NewDatabaseEventStore(db *sql.DB, logger Logger) *DatabaseEventStore {
    return &DatabaseEventStore{
        db:     db,
        logger: logger,
    }
}

func (s *DatabaseEventStore) Store(ctx context.Context, event events.Event) error {
    query := `
        INSERT INTO events (id, type, topic, data, metadata, timestamp, version)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

    dataJSON, err := json.Marshal(event.GetData())
    if err != nil {
        return err
    }

    metadataJSON, err := json.Marshal(event.GetMetadata())
    if err != nil {
        return err
    }

    _, err = s.db.ExecContext(ctx, query,
        event.GetID(),
        event.GetType(),
        event.GetTopic(),
        string(dataJSON),
        string(metadataJSON),
        event.GetTimestamp(),
        event.GetVersion(),
    )

    if err != nil {
        s.logger.Error("Failed to store event:", err)
        return err
    }

    return nil
}

func (s *DatabaseEventStore) LoadByTopic(ctx context.Context, topic string, offset, limit int) ([]events.Event, error) {
    query := `
        SELECT id, type, topic, data, metadata, timestamp, version
        FROM events
        WHERE topic = $1
        ORDER BY timestamp DESC
        OFFSET $2 LIMIT $3
    `

    rows, err := s.db.QueryContext(ctx, query, topic, offset, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []events.Event
    for rows.Next() {
        var id, eventType, eventTopic, dataJSON, metadataJSON, version string
        var timestamp time.Time

        err := rows.Scan(&id, &eventType, &eventTopic, &dataJSON, &metadataJSON, &timestamp, &version)
        if err != nil {
            s.logger.Error("Failed to scan event row:", err)
            continue
        }

        var data interface{}
        if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
            s.logger.Error("Failed to unmarshal event data:", err)
            continue
        }

        var metadata map[string]interface{}
        if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
            s.logger.Error("Failed to unmarshal event metadata:", err)
            continue
        }

        event := &events.BaseEvent{
            ID:        id,
            Type:      eventType,
            Topic:     eventTopic,
            Data:      data,
            Metadata:  metadata,
            Timestamp: timestamp,
            Version:   version,
        }

        result = append(result, event)
    }

    return result, nil
}
```

### 事件重放

```go
// 事件重放器
type EventReplayer struct {
    eventStore EventStore
    eventBus   events.EventBus
    logger     Logger
}

func NewEventReplayer(eventStore EventStore, eventBus events.EventBus, logger Logger) *EventReplayer {
    return &EventReplayer{
        eventStore: eventStore,
        eventBus:   eventBus,
        logger:     logger,
    }
}

// 重放指定主题的事件
func (r *EventReplayer) ReplayTopic(ctx context.Context, topic string, startTime time.Time) error {
    offset := 0
    limit := 100

    for {
        events, err := r.eventStore.LoadByTopic(ctx, topic, offset, limit)
        if err != nil {
            return fmt.Errorf("failed to load events: %w", err)
        }

        if len(events) == 0 {
            break
        }

        for _, event := range events {
            if event.GetTimestamp().Before(startTime) {
                continue
            }

            r.logger.Info("Replaying event:", event.GetID())

            err := r.eventBus.Publish(ctx, event)
            if err != nil {
                r.logger.Error("Failed to replay event:", err)
                return err
            }
        }

        offset += limit
    }

    return nil
}

// 重放指定时间范围的事件
func (r *EventReplayer) ReplayTimeRange(ctx context.Context, startTime, endTime time.Time) error {
    events, err := r.eventStore.LoadByTimeRange(ctx, startTime, endTime)
    if err != nil {
        return fmt.Errorf("failed to load events: %w", err)
    }

    r.logger.Info("Replaying events in time range:", "start", startTime, "end", endTime, "count", len(events))

    for _, event := range events {
        err := r.eventBus.Publish(ctx, event)
        if err != nil {
            r.logger.Error("Failed to replay event:", err)
            return err
        }
    }

    return nil
}
```

## 分布式事件

### 事件消息队列集成

```go
// RabbitMQ事件发布器
type RabbitMQEventPublisher struct {
    connection *amqp.Connection
    channel    *amqp.Channel
}

func NewRabbitMQEventPublisher(url string) (*RabbitMQEventPublisher, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    return &RabbitMQEventPublisher{
        connection: conn,
        channel:    ch,
    }, nil
}

func (p *RabbitMQEventPublisher) Publish(ctx context.Context, event events.Event) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }

    return p.channel.Publish(
        event.GetTopic(), // exchange
        "",               // routing key
        false,            // mandatory
        false,            // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
            MessageId:   event.GetID(),
            Timestamp:   event.GetTimestamp(),
        },
    )
}

// Apache Kafka事件发布器
type KafkaEventPublisher struct {
    producer sarama.SyncProducer
}

func NewKafkaEventPublisher(brokers []string) (*KafkaEventPublisher, error) {
    config := sarama.NewConfig()
    config.Producer.Return.Successes = true

    producer, err := sarama.NewSyncProducer(brokers, config)
    if err != nil {
        return nil, err
    }

    return &KafkaEventPublisher{
        producer: producer,
    }, nil
}

func (p *KafkaEventPublisher) Publish(ctx context.Context, event events.Event) error {
    body, err := json.Marshal(event)
    if err != nil {
        return err
    }

    message := &sarama.ProducerMessage{
        Topic: event.GetTopic(),
        Key:   sarama.StringEncoder(event.GetID()),
        Value: sarama.ByteEncoder(body),
        Headers: []sarama.RecordHeader{
            {
                Key:   []byte("event-type"),
                Value: []byte(event.GetType()),
            },
        },
    }

    _, _, err = p.producer.SendMessage(message)
    return err
}
```

### 事件消费者

```go
// RabbitMQ事件消费者
type RabbitMQEventConsumer struct {
    connection *amqp.Connection
    channel    *amqp.Channel
    handlers   map[string]EventHandler
}

func NewRabbitMQEventConsumer(url string) (*RabbitMQEventConsumer, error) {
    conn, err := amqp.Dial(url)
    if err != nil {
        return nil, err
    }

    ch, err := conn.Channel()
    if err != nil {
        return nil, err
    }

    return &RabbitMQEventConsumer{
        connection: conn,
        channel:    ch,
        handlers:   make(map[string]EventHandler),
    }, nil
}

func (c *RabbitMQEventConsumer) Subscribe(topic string, handler EventHandler) error {
    c.handlers[topic] = handler

    // 声明队列
    queue, err := c.channel.QueueDeclare(
        topic,  // queue name
        true,   // durable
        false,  // delete when unused
        false,  // exclusive
        false,  // no-wait
        nil,    // arguments
    )
    if err != nil {
        return err
    }

    // 绑定队列到交换机
    err = c.channel.QueueBind(
        queue.Name, // queue name
        "",         // routing key
        topic,      // exchange
        false,
        nil,
    )
    if err != nil {
        return err
    }

    // 开始消费
    messages, err := c.channel.Consume(
        queue.Name, // queue
        "",         // consumer
        false,      // auto-ack
        false,      // exclusive
        false,      // no-local
        false,      // no-wait
        nil,        // args
    )
    if err != nil {
        return err
    }

    go func() {
        for msg := range messages {
            var event events.BaseEvent
            err := json.Unmarshal(msg.Body, &event)
            if err != nil {
                log.Error("Failed to unmarshal event:", err)
                msg.Nack(false, false)
                continue
            }

            ctx := context.Background()
            err = handler.Handle(ctx, &event)
            if err != nil {
                log.Error("Failed to handle event:", err)
                msg.Nack(false, true) // requeue
            } else {
                msg.Ack(false)
            }
        }
    }()

    return nil
}
```

## 配置示例

### 完整配置文件

```yaml
# events-config.yaml
events:
  # 事件总线配置
  bus:
    type: "memory" # memory, redis, kafka, rabbitmq
    buffer_size: 10000
    worker_count: 10

    # Redis配置（当type为redis时）
    redis:
      address: "localhost:6379"
      password: ""
      db: 0

    # Kafka配置（当type为kafka时）
    kafka:
      brokers:
        - "localhost:9092"
      consumer_group: "chi-events"

    # RabbitMQ配置（当type为rabbitmq时）
    rabbitmq:
      url: "amqp://guest:guest@localhost:5672/"

  # 事件存储配置
  store:
    enabled: true
    type: "database" # memory, database, file

    # 数据库配置
    database:
      driver: "postgres"
      dsn: "postgres://user:pass@localhost/events?sslmode=disable"
      table_name: "events"

    # 文件存储配置
    file:
      directory: "/var/log/events"
      max_file_size: "100MB"
      max_files: 10

  # 事件重放配置
  replay:
    enabled: true
    batch_size: 100
    concurrent_workers: 5

  # 监控配置
  monitoring:
    metrics_enabled: true
    tracing_enabled: true
    health_check_enabled: true

  # 路由配置
  routing:
    rules:
      # 高优先级安全事件
      - priority: 100
        filter:
          type: "topic"
          values: ["security"]
        handler: "security_handler"

      # 业务事件
      - priority: 50
        filter:
          type: "pattern"
          values: ["user.*", "order.*"]
        handler: "business_handler"

      # 所有事件记录日志
      - priority: 1
        filter:
          type: "all"
        handler: "log_handler"
```

通过以上完整的事件驱动架构方案，Chi框架能够提供企业级的事件处理能力，实现服务间的松耦合通信，支持复杂的业务流程编排和数据一致性保证。