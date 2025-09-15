# WebSocket 开发指南

## 概述

Chi 框架提供了企业级的 WebSocket 支持，实现高并发实时通信功能。框架内置智能连接管理、消息路由、集群支持和完整的认证授权机制，能够处理大规模的 WebSocket 连接并保证消息的可靠传递。

## WebSocket 架构

### 整体架构图
```
┌─────────────────────────────────────────────────────────────────┐
│                    WebSocket System                             │
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │ Connection  │   │   Message   │   │   Session   │          │
│  │   Manager   │   │   Router    │   │   Manager   │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
│          │                 │                 │                 │
│          └─────────────────┼─────────────────┘                 │
│                            │                                   │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                WebSocket Gateway                           ││
│  └─────────────────────────────────────────────────────────────┐│
│                            │                                   │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐          │
│  │   Room      │   │  Broadcast  │   │   Event     │          │
│  │  Manager    │   │   Engine    │   │  Publisher  │          │
│  └─────────────┘   └─────────────┘   └─────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

### 核心组件

1. **Connection Manager**: 连接生命周期管理
2. **Message Router**: 消息路由和分发
3. **Session Manager**: 会话状态管理
4. **Room Manager**: 房间和群组管理
5. **Broadcast Engine**: 消息广播引擎
6. **Event Publisher**: 事件发布系统

## 基础WebSocket服务

### WebSocket服务器配置

```go
package websocket

import (
    "context"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
    "github.com/gin-gonic/gin"
)

// WebSocket配置
type Config struct {
    // 连接配置
    ReadBufferSize    int           // 读缓冲区大小
    WriteBufferSize   int           // 写缓冲区大小
    HandshakeTimeout  time.Duration // 握手超时时间
    CheckOrigin       func(*http.Request) bool

    // 消息配置
    MaxMessageSize    int64         // 最大消息大小
    ReadTimeout       time.Duration // 读超时时间
    WriteTimeout      time.Duration // 写超时时间
    PongTimeout       time.Duration // Pong超时时间
    PingPeriod        time.Duration // Ping周期

    // 连接池配置
    MaxConnections    int           // 最大连接数
    ConnectionTimeout time.Duration // 连接超时时间
    CleanupInterval   time.Duration // 清理间隔
}

// WebSocket服务器
type Server struct {
    config     Config
    upgrader   websocket.Upgrader
    clients    sync.Map                 // 客户端连接映射
    rooms      sync.Map                 // 房间映射
    handlers   map[string]MessageHandler // 消息处理器
    middleware []Middleware             // 中间件
    mu         sync.RWMutex
    stats      ServerStats
}

// 服务器统计
type ServerStats struct {
    ActiveConnections int64
    TotalMessages     int64
    TotalBytes        int64
    Errors            int64
}

func NewServer(config Config) *Server {
    server := &Server{
        config: config,
        upgrader: websocket.Upgrader{
            ReadBufferSize:  config.ReadBufferSize,
            WriteBufferSize: config.WriteBufferSize,
            CheckOrigin:     config.CheckOrigin,
        },
        handlers: make(map[string]MessageHandler),
    }

    // 设置默认值
    if server.config.MaxMessageSize == 0 {
        server.config.MaxMessageSize = 1024 * 1024 // 1MB
    }
    if server.config.PingPeriod == 0 {
        server.config.PingPeriod = 54 * time.Second
    }
    if server.config.PongTimeout == 0 {
        server.config.PongTimeout = 60 * time.Second
    }

    return server
}

// 启动服务器
func (s *Server) Start() error {
    // 启动连接清理协程
    go s.cleanupRoutine()

    return nil
}

// 处理WebSocket连接升级
func (s *Server) HandleConnection() gin.HandlerFunc {
    return func(c *gin.Context) {
        conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "WebSocket upgrade failed",
            })
            return
        }

        client := s.createClient(conn, c.Request)
        s.handleClient(client)
    }
}

func (s *Server) createClient(conn *websocket.Conn, req *http.Request) *Client {
    client := &Client{
        ID:         generateClientID(),
        Conn:       conn,
        Send:       make(chan []byte, 256),
        Server:     s,
        Request:    req,
        CreatedAt:  time.Now(),
        LastActive: time.Now(),
        Metadata:   make(map[string]interface{}),
    }

    // 设置连接参数
    conn.SetReadLimit(s.config.MaxMessageSize)
    conn.SetReadDeadline(time.Now().Add(s.config.PongTimeout))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(s.config.PongTimeout))
        client.LastActive = time.Now()
        return nil
    })

    return client
}
```

### WebSocket客户端管理

```go
// WebSocket客户端
type Client struct {
    ID         string                 `json:"id"`
    Conn       *websocket.Conn        `json:"-"`
    Send       chan []byte            `json:"-"`
    Server     *Server                `json:"-"`
    Request    *http.Request          `json:"-"`
    UserID     string                 `json:"user_id,omitempty"`
    SessionID  string                 `json:"session_id,omitempty"`
    Rooms      map[string]bool        `json:"rooms,omitempty"`
    CreatedAt  time.Time              `json:"created_at"`
    LastActive time.Time              `json:"last_active"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
    mu         sync.RWMutex
}

func (c *Client) GetUserID() string {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.UserID
}

func (c *Client) SetUserID(userID string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.UserID = userID
}

func (c *Client) GetMetadata(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    value, exists := c.Metadata[key]
    return value, exists
}

func (c *Client) SetMetadata(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Metadata[key] = value
}

func (c *Client) IsInRoom(roomID string) bool {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.Rooms[roomID]
}

// 读取消息协程
func (c *Client) readPump() {
    defer func() {
        c.Server.unregisterClient(c)
        c.Conn.Close()
    }()

    c.Conn.SetReadLimit(c.Server.config.MaxMessageSize)
    c.Conn.SetReadDeadline(time.Now().Add(c.Server.config.PongTimeout))

    for {
        messageType, message, err := c.Conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err,
                websocket.CloseGoingAway,
                websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }

        c.LastActive = time.Now()
        c.Server.handleMessage(c, messageType, message)
    }
}

// 写入消息协程
func (c *Client) writePump() {
    ticker := time.NewTicker(c.Server.config.PingPeriod)
    defer func() {
        ticker.Stop()
        c.Conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.Send:
            c.Conn.SetWriteDeadline(time.Now().Add(c.Server.config.WriteTimeout))
            if !ok {
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.Conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // 批量发送缓冲区中的消息
            n := len(c.Send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.Send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(c.Server.config.WriteTimeout))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// 发送消息
func (c *Client) SendMessage(message []byte) error {
    select {
    case c.Send <- message:
        return nil
    default:
        close(c.Send)
        return errors.New("client send channel is closed")
    }
}

// 发送JSON消息
func (c *Client) SendJSON(data interface{}) error {
    message, err := json.Marshal(data)
    if err != nil {
        return err
    }
    return c.SendMessage(message)
}
```

## 消息处理系统

### 消息定义

```go
// WebSocket消息结构
type Message struct {
    Type      string                 `json:"type"`
    ID        string                 `json:"id,omitempty"`
    To        string                 `json:"to,omitempty"`        // 目标客户端ID
    Room      string                 `json:"room,omitempty"`      // 目标房间ID
    Data      interface{}            `json:"data"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// 消息处理器接口
type MessageHandler interface {
    Handle(client *Client, message *Message) error
    MessageType() string
}

// 消息中间件接口
type Middleware interface {
    Process(client *Client, message *Message, next func() error) error
}

// 基础消息处理器
type BaseMessageHandler struct {
    messageType string
    handler     func(*Client, *Message) error
}

func NewMessageHandler(messageType string, handler func(*Client, *Message) error) MessageHandler {
    return &BaseMessageHandler{
        messageType: messageType,
        handler:     handler,
    }
}

func (h *BaseMessageHandler) MessageType() string {
    return h.messageType
}

func (h *BaseMessageHandler) Handle(client *Client, message *Message) error {
    return h.handler(client, message)
}
```

### 消息路由器

```go
// 消息路由器
type MessageRouter struct {
    handlers   map[string]MessageHandler
    middleware []Middleware
    mu         sync.RWMutex
}

func NewMessageRouter() *MessageRouter {
    return &MessageRouter{
        handlers: make(map[string]MessageHandler),
    }
}

func (mr *MessageRouter) AddHandler(handler MessageHandler) {
    mr.mu.Lock()
    defer mr.mu.Unlock()
    mr.handlers[handler.MessageType()] = handler
}

func (mr *MessageRouter) AddMiddleware(middleware Middleware) {
    mr.mu.Lock()
    defer mr.mu.Unlock()
    mr.middleware = append(mr.middleware, middleware)
}

func (mr *MessageRouter) Route(client *Client, message *Message) error {
    mr.mu.RLock()
    handler, exists := mr.handlers[message.Type]
    middlewares := mr.middleware
    mr.mu.RUnlock()

    if !exists {
        return fmt.Errorf("no handler for message type: %s", message.Type)
    }

    // 构建中间件调用链
    return mr.executeMiddlewareChain(client, message, middlewares, 0, func() error {
        return handler.Handle(client, message)
    })
}

func (mr *MessageRouter) executeMiddlewareChain(client *Client, message *Message,
    middlewares []Middleware, index int, final func() error) error {

    if index >= len(middlewares) {
        return final()
    }

    return middlewares[index].Process(client, message, func() error {
        return mr.executeMiddlewareChain(client, message, middlewares, index+1, final)
    })
}
```

### 常用消息处理器

```go
// Echo消息处理器
func NewEchoHandler() MessageHandler {
    return NewMessageHandler("echo", func(client *Client, message *Message) error {
        response := &Message{
            Type:      "echo_response",
            ID:        message.ID,
            Data:      message.Data,
            Timestamp: time.Now(),
        }

        return client.SendJSON(response)
    })
}

// 聊天消息处理器
func NewChatHandler(roomManager *RoomManager) MessageHandler {
    return NewMessageHandler("chat", func(client *Client, message *Message) error {
        chatData, ok := message.Data.(map[string]interface{})
        if !ok {
            return errors.New("invalid chat message data")
        }

        roomID, exists := chatData["room_id"].(string)
        if !exists {
            return errors.New("room_id is required for chat messages")
        }

        content, exists := chatData["content"].(string)
        if !exists {
            return errors.New("content is required for chat messages")
        }

        // 构建聊天消息
        chatMessage := &Message{
            Type: "chat_message",
            Data: map[string]interface{}{
                "room_id":   roomID,
                "user_id":   client.GetUserID(),
                "username":  client.GetMetadata("username"),
                "content":   content,
                "timestamp": time.Now(),
            },
            Timestamp: time.Now(),
        }

        // 广播到房间
        return roomManager.BroadcastToRoom(roomID, chatMessage)
    })
}

// 私聊消息处理器
func NewPrivateMessageHandler(server *Server) MessageHandler {
    return NewMessageHandler("private_message", func(client *Client, message *Message) error {
        pmData, ok := message.Data.(map[string]interface{})
        if !ok {
            return errors.New("invalid private message data")
        }

        targetUserID, exists := pmData["target_user_id"].(string)
        if !exists {
            return errors.New("target_user_id is required")
        }

        content, exists := pmData["content"].(string)
        if !exists {
            return errors.New("content is required")
        }

        // 查找目标用户的连接
        targetClient := server.FindClientByUserID(targetUserID)
        if targetClient == nil {
            return errors.New("target user not found or offline")
        }

        // 发送私聊消息
        privateMessage := &Message{
            Type: "private_message",
            Data: map[string]interface{}{
                "from_user_id": client.GetUserID(),
                "from_username": client.GetMetadata("username"),
                "content":      content,
                "timestamp":    time.Now(),
            },
            Timestamp: time.Now(),
        }

        return targetClient.SendJSON(privateMessage)
    })
}
```

## 房间管理系统

### 房间管理器

```go
// 房间接口
type Room interface {
    ID() string
    AddClient(client *Client) error
    RemoveClient(clientID string) error
    GetClients() []*Client
    Broadcast(message *Message) error
    GetClientCount() int
    IsEmpty() bool
    GetMetadata() map[string]interface{}
    SetMetadata(key string, value interface{})
}

// 基础房间实现
type BaseRoom struct {
    id       string
    clients  sync.Map
    metadata sync.Map
    mu       sync.RWMutex
}

func NewRoom(id string) Room {
    return &BaseRoom{
        id: id,
    }
}

func (r *BaseRoom) ID() string {
    return r.id
}

func (r *BaseRoom) AddClient(client *Client) error {
    r.clients.Store(client.ID, client)

    client.mu.Lock()
    if client.Rooms == nil {
        client.Rooms = make(map[string]bool)
    }
    client.Rooms[r.id] = true
    client.mu.Unlock()

    return nil
}

func (r *BaseRoom) RemoveClient(clientID string) error {
    if value, exists := r.clients.LoadAndDelete(clientID); exists {
        if client, ok := value.(*Client); ok {
            client.mu.Lock()
            delete(client.Rooms, r.id)
            client.mu.Unlock()
        }
    }

    return nil
}

func (r *BaseRoom) GetClients() []*Client {
    var clients []*Client

    r.clients.Range(func(key, value interface{}) bool {
        if client, ok := value.(*Client); ok {
            clients = append(clients, client)
        }
        return true
    })

    return clients
}

func (r *BaseRoom) Broadcast(message *Message) error {
    clients := r.GetClients()

    var wg sync.WaitGroup
    for _, client := range clients {
        wg.Add(1)
        go func(c *Client) {
            defer wg.Done()
            c.SendJSON(message)
        }(client)
    }

    wg.Wait()
    return nil
}

func (r *BaseRoom) GetClientCount() int {
    count := 0
    r.clients.Range(func(_, _ interface{}) bool {
        count++
        return true
    })
    return count
}

func (r *BaseRoom) IsEmpty() bool {
    return r.GetClientCount() == 0
}

func (r *BaseRoom) GetMetadata() map[string]interface{} {
    metadata := make(map[string]interface{})
    r.metadata.Range(func(key, value interface{}) bool {
        if k, ok := key.(string); ok {
            metadata[k] = value
        }
        return true
    })
    return metadata
}

func (r *BaseRoom) SetMetadata(key string, value interface{}) {
    r.metadata.Store(key, value)
}

// 房间管理器
type RoomManager struct {
    rooms       sync.Map
    roomFactory func(id string) Room
    mu          sync.RWMutex
}

func NewRoomManager(roomFactory func(id string) Room) *RoomManager {
    if roomFactory == nil {
        roomFactory = NewRoom
    }

    return &RoomManager{
        roomFactory: roomFactory,
    }
}

func (rm *RoomManager) GetOrCreateRoom(roomID string) Room {
    if value, exists := rm.rooms.Load(roomID); exists {
        return value.(Room)
    }

    room := rm.roomFactory(roomID)
    actual, _ := rm.rooms.LoadOrStore(roomID, room)
    return actual.(Room)
}

func (rm *RoomManager) GetRoom(roomID string) (Room, bool) {
    if value, exists := rm.rooms.Load(roomID); exists {
        return value.(Room), true
    }
    return nil, false
}

func (rm *RoomManager) JoinRoom(roomID string, client *Client) error {
    room := rm.GetOrCreateRoom(roomID)
    return room.AddClient(client)
}

func (rm *RoomManager) LeaveRoom(roomID string, client *Client) error {
    if room, exists := rm.GetRoom(roomID); exists {
        err := room.RemoveClient(client.ID)

        // 如果房间为空，删除房间
        if room.IsEmpty() {
            rm.rooms.Delete(roomID)
        }

        return err
    }
    return nil
}

func (rm *RoomManager) BroadcastToRoom(roomID string, message *Message) error {
    if room, exists := rm.GetRoom(roomID); exists {
        return room.Broadcast(message)
    }
    return fmt.Errorf("room %s not found", roomID)
}

func (rm *RoomManager) GetRoomList() []RoomInfo {
    var rooms []RoomInfo

    rm.rooms.Range(func(key, value interface{}) bool {
        if roomID, ok := key.(string); ok {
            if room, ok := value.(Room); ok {
                rooms = append(rooms, RoomInfo{
                    ID:          roomID,
                    ClientCount: room.GetClientCount(),
                    Metadata:    room.GetMetadata(),
                })
            }
        }
        return true
    })

    return rooms
}

type RoomInfo struct {
    ID          string                 `json:"id"`
    ClientCount int                    `json:"client_count"`
    Metadata    map[string]interface{} `json:"metadata"`
}
```

## 认证和授权

### WebSocket认证中间件

```go
// 认证中间件
type AuthMiddleware struct {
    tokenValidator TokenValidator
    userProvider   UserProvider
}

type TokenValidator interface {
    ValidateToken(token string) (*TokenClaims, error)
}

type UserProvider interface {
    GetUser(userID string) (*User, error)
}

type TokenClaims struct {
    UserID   string   `json:"user_id"`
    Username string   `json:"username"`
    Roles    []string `json:"roles"`
    ExpireAt int64    `json:"exp"`
}

type User struct {
    ID       string   `json:"id"`
    Username string   `json:"username"`
    Roles    []string `json:"roles"`
    Metadata map[string]interface{} `json:"metadata"`
}

func NewAuthMiddleware(tokenValidator TokenValidator, userProvider UserProvider) *AuthMiddleware {
    return &AuthMiddleware{
        tokenValidator: tokenValidator,
        userProvider:   userProvider,
    }
}

func (am *AuthMiddleware) Process(client *Client, message *Message, next func() error) error {
    // 对于认证消息，直接处理
    if message.Type == "auth" {
        return am.handleAuthMessage(client, message)
    }

    // 检查客户端是否已认证
    if client.GetUserID() == "" {
        return client.SendJSON(&Message{
            Type: "error",
            Data: map[string]interface{}{
                "message": "authentication required",
                "code":    "AUTH_REQUIRED",
            },
            Timestamp: time.Now(),
        })
    }

    return next()
}

func (am *AuthMiddleware) handleAuthMessage(client *Client, message *Message) error {
    authData, ok := message.Data.(map[string]interface{})
    if !ok {
        return am.sendAuthError(client, "invalid auth data")
    }

    token, ok := authData["token"].(string)
    if !ok {
        return am.sendAuthError(client, "token is required")
    }

    // 验证token
    claims, err := am.tokenValidator.ValidateToken(token)
    if err != nil {
        return am.sendAuthError(client, "invalid token")
    }

    // 获取用户信息
    user, err := am.userProvider.GetUser(claims.UserID)
    if err != nil {
        return am.sendAuthError(client, "user not found")
    }

    // 设置客户端用户信息
    client.SetUserID(user.ID)
    client.SetMetadata("username", user.Username)
    client.SetMetadata("roles", user.Roles)

    // 发送认证成功响应
    return client.SendJSON(&Message{
        Type: "auth_success",
        Data: map[string]interface{}{
            "user_id":  user.ID,
            "username": user.Username,
            "roles":    user.Roles,
        },
        Timestamp: time.Now(),
    })
}

func (am *AuthMiddleware) sendAuthError(client *Client, message string) error {
    return client.SendJSON(&Message{
        Type: "auth_error",
        Data: map[string]interface{}{
            "message": message,
        },
        Timestamp: time.Now(),
    })
}
```

### 权限控制中间件

```go
// 权限控制中间件
type AuthorizationMiddleware struct {
    permissions map[string][]string // 消息类型 -> 需要的权限
}

func NewAuthorizationMiddleware() *AuthorizationMiddleware {
    return &AuthorizationMiddleware{
        permissions: make(map[string][]string),
    }
}

func (azm *AuthorizationMiddleware) AddPermission(messageType string, requiredRoles ...string) {
    azm.permissions[messageType] = requiredRoles
}

func (azm *AuthorizationMiddleware) Process(client *Client, message *Message, next func() error) error {
    requiredRoles, exists := azm.permissions[message.Type]
    if !exists {
        // 没有权限要求，直接通过
        return next()
    }

    userRoles, exists := client.GetMetadata("roles")
    if !exists {
        return azm.sendAuthorizationError(client, "user roles not found")
    }

    roles, ok := userRoles.([]string)
    if !ok {
        return azm.sendAuthorizationError(client, "invalid user roles")
    }

    // 检查权限
    if !azm.hasRequiredRole(roles, requiredRoles) {
        return azm.sendAuthorizationError(client, "insufficient permissions")
    }

    return next()
}

func (azm *AuthorizationMiddleware) hasRequiredRole(userRoles, requiredRoles []string) bool {
    for _, required := range requiredRoles {
        for _, userRole := range userRoles {
            if userRole == required || userRole == "admin" {
                return true
            }
        }
    }
    return false
}

func (azm *AuthorizationMiddleware) sendAuthorizationError(client *Client, message string) error {
    return client.SendJSON(&Message{
        Type: "authorization_error",
        Data: map[string]interface{}{
            "message": message,
            "code":    "INSUFFICIENT_PERMISSIONS",
        },
        Timestamp: time.Now(),
    })
}
```

## 集群支持

### Redis消息广播

```go
// Redis集群适配器
type RedisClusterAdapter struct {
    client        redis.Client
    pubsub        *redis.PubSub
    localServer   *Server
    subscriptions sync.Map
}

func NewRedisClusterAdapter(client redis.Client, localServer *Server) *RedisClusterAdapter {
    adapter := &RedisClusterAdapter{
        client:      client,
        localServer: localServer,
    }

    adapter.pubsub = client.Subscribe(context.Background())
    go adapter.listenForMessages()

    return adapter
}

func (rca *RedisClusterAdapter) BroadcastToCluster(roomID string, message *Message) error {
    clusterMessage := &ClusterMessage{
        Type:   "room_broadcast",
        RoomID: roomID,
        Data:   message,
        Source: rca.localServer.GetServerID(),
    }

    data, err := json.Marshal(clusterMessage)
    if err != nil {
        return err
    }

    return rca.client.Publish(context.Background(),
        fmt.Sprintf("websocket:room:%s", roomID),
        data).Err()
}

func (rca *RedisClusterAdapter) SendToUser(userID string, message *Message) error {
    clusterMessage := &ClusterMessage{
        Type:   "user_message",
        UserID: userID,
        Data:   message,
        Source: rca.localServer.GetServerID(),
    }

    data, err := json.Marshal(clusterMessage)
    if err != nil {
        return err
    }

    return rca.client.Publish(context.Background(),
        fmt.Sprintf("websocket:user:%s", userID),
        data).Err()
}

func (rca *RedisClusterAdapter) listenForMessages() {
    ch := rca.pubsub.Channel()

    for msg := range ch {
        var clusterMessage ClusterMessage
        if err := json.Unmarshal([]byte(msg.Payload), &clusterMessage); err != nil {
            log.Printf("Failed to unmarshal cluster message: %v", err)
            continue
        }

        // 忽略来自本服务器的消息
        if clusterMessage.Source == rca.localServer.GetServerID() {
            continue
        }

        rca.handleClusterMessage(&clusterMessage)
    }
}

func (rca *RedisClusterAdapter) handleClusterMessage(clusterMessage *ClusterMessage) {
    switch clusterMessage.Type {
    case "room_broadcast":
        rca.localServer.BroadcastToRoom(clusterMessage.RoomID, clusterMessage.Data)
    case "user_message":
        client := rca.localServer.FindClientByUserID(clusterMessage.UserID)
        if client != nil {
            client.SendJSON(clusterMessage.Data)
        }
    }
}

type ClusterMessage struct {
    Type   string   `json:"type"`
    RoomID string   `json:"room_id,omitempty"`
    UserID string   `json:"user_id,omitempty"`
    Data   *Message `json:"data"`
    Source string   `json:"source"`
}
```

## 监控和指标

### WebSocket指标收集

```go
// WebSocket指标收集器
type WSMetricsCollector struct {
    // 连接指标
    ActiveConnections prometheus.Gauge
    TotalConnections  prometheus.Counter
    ConnectionErrors  prometheus.Counter

    // 消息指标
    MessagesReceived prometheus.Counter
    MessagesSent     prometheus.Counter
    MessageErrors    prometheus.Counter
    MessageLatency   prometheus.Histogram

    // 房间指标
    ActiveRooms      prometheus.Gauge
    RoomOperations   *prometheus.CounterVec
}

func NewWSMetricsCollector() *WSMetricsCollector {
    collector := &WSMetricsCollector{
        ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "websocket_active_connections",
            Help: "Number of active WebSocket connections",
        }),

        TotalConnections: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "websocket_total_connections",
            Help: "Total number of WebSocket connections",
        }),

        ConnectionErrors: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "websocket_connection_errors_total",
            Help: "Total number of WebSocket connection errors",
        }),

        MessagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "websocket_messages_received_total",
            Help: "Total number of WebSocket messages received",
        }),

        MessagesSent: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "websocket_messages_sent_total",
            Help: "Total number of WebSocket messages sent",
        }),

        MessageErrors: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "websocket_message_errors_total",
            Help: "Total number of WebSocket message errors",
        }),

        MessageLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "websocket_message_latency_seconds",
            Help:    "WebSocket message processing latency",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
        }),

        ActiveRooms: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "websocket_active_rooms",
            Help: "Number of active WebSocket rooms",
        }),

        RoomOperations: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "websocket_room_operations_total",
                Help: "Total number of WebSocket room operations",
            },
            []string{"operation", "result"},
        ),
    }

    // 注册指标
    prometheus.MustRegister(
        collector.ActiveConnections,
        collector.TotalConnections,
        collector.ConnectionErrors,
        collector.MessagesReceived,
        collector.MessagesSent,
        collector.MessageErrors,
        collector.MessageLatency,
        collector.ActiveRooms,
        collector.RoomOperations,
    )

    return collector
}

// 记录连接事件
func (wmc *WSMetricsCollector) RecordConnection() {
    wmc.TotalConnections.Inc()
    wmc.ActiveConnections.Inc()
}

func (wmc *WSMetricsCollector) RecordDisconnection() {
    wmc.ActiveConnections.Dec()
}

func (wmc *WSMetricsCollector) RecordConnectionError() {
    wmc.ConnectionErrors.Inc()
}

// 记录消息事件
func (wmc *WSMetricsCollector) RecordMessageReceived() {
    wmc.MessagesReceived.Inc()
}

func (wmc *WSMetricsCollector) RecordMessageSent() {
    wmc.MessagesSent.Inc()
}

func (wmc *WSMetricsCollector) RecordMessageError() {
    wmc.MessageErrors.Inc()
}

func (wmc *WSMetricsCollector) RecordMessageLatency(duration time.Duration) {
    wmc.MessageLatency.Observe(duration.Seconds())
}

// 记录房间事件
func (wmc *WSMetricsCollector) RecordRoomOperation(operation, result string) {
    wmc.RoomOperations.WithLabelValues(operation, result).Inc()
}

func (wmc *WSMetricsCollector) UpdateActiveRooms(count int) {
    wmc.ActiveRooms.Set(float64(count))
}
```

## 完整使用示例

### 创建WebSocket服务

```go
func main() {
    // 创建WebSocket配置
    config := websocket.Config{
        ReadBufferSize:    1024,
        WriteBufferSize:   1024,
        HandshakeTimeout:  10 * time.Second,
        MaxMessageSize:    1024 * 1024, // 1MB
        ReadTimeout:       60 * time.Second,
        WriteTimeout:      10 * time.Second,
        PongTimeout:       60 * time.Second,
        PingPeriod:        54 * time.Second,
        MaxConnections:    10000,
        ConnectionTimeout: 30 * time.Second,
        CheckOrigin: func(r *http.Request) bool {
            return true // 在生产环境中应该检查Origin
        },
    }

    // 创建WebSocket服务器
    wsServer := websocket.NewServer(config)

    // 创建消息路由器
    router := websocket.NewMessageRouter()

    // 添加认证中间件
    authMiddleware := NewAuthMiddleware(tokenValidator, userProvider)
    router.AddMiddleware(authMiddleware)

    // 添加授权中间件
    authzMiddleware := NewAuthorizationMiddleware()
    authzMiddleware.AddPermission("admin_message", "admin")
    router.AddMiddleware(authzMiddleware)

    // 添加消息处理器
    router.AddHandler(NewEchoHandler())
    router.AddHandler(NewChatHandler(roomManager))
    router.AddHandler(NewPrivateMessageHandler(wsServer))

    // 设置消息路由器
    wsServer.SetMessageRouter(router)

    // 创建Gin路由
    r := gin.Default()

    // WebSocket端点
    r.GET("/ws", wsServer.HandleConnection())

    // 启动服务器
    wsServer.Start()
    r.Run(":8080")
}
```

### 前端JavaScript客户端示例

```javascript
class WebSocketClient {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.messageHandlers = new Map();
        this.connected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
    }

    connect() {
        return new Promise((resolve, reject) => {
            this.ws = new WebSocket(this.url);

            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.connected = true;
                this.reconnectAttempts = 0;
                resolve();
            };

            this.ws.onclose = (event) => {
                console.log('WebSocket disconnected:', event.code, event.reason);
                this.connected = false;
                this.handleReconnect();
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                reject(error);
            };

            this.ws.onmessage = (event) => {
                this.handleMessage(event.data);
            };
        });
    }

    handleMessage(data) {
        try {
            const message = JSON.parse(data);
            const handler = this.messageHandlers.get(message.type);
            if (handler) {
                handler(message);
            } else {
                console.warn('No handler for message type:', message.type);
            }
        } catch (error) {
            console.error('Failed to parse message:', error);
        }
    }

    send(type, data) {
        if (!this.connected) {
            console.error('WebSocket is not connected');
            return false;
        }

        const message = {
            type: type,
            data: data,
            timestamp: new Date().toISOString()
        };

        this.ws.send(JSON.stringify(message));
        return true;
    }

    authenticate(token) {
        return this.send('auth', { token: token });
    }

    joinRoom(roomId) {
        return this.send('join_room', { room_id: roomId });
    }

    sendChatMessage(roomId, content) {
        return this.send('chat', {
            room_id: roomId,
            content: content
        });
    }

    sendPrivateMessage(targetUserId, content) {
        return this.send('private_message', {
            target_user_id: targetUserId,
            content: content
        });
    }

    onMessage(type, handler) {
        this.messageHandlers.set(type, handler);
    }

    handleReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            const delay = Math.pow(2, this.reconnectAttempts) * 1000; // 指数退避
            console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);

            setTimeout(() => {
                this.connect().catch(err => {
                    console.error('Reconnect failed:', err);
                });
            }, delay);
        } else {
            console.error('Max reconnect attempts reached');
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
            this.connected = false;
        }
    }
}

// 使用示例
const wsClient = new WebSocketClient('ws://localhost:8080/ws');

// 设置消息处理器
wsClient.onMessage('auth_success', (message) => {
    console.log('Authenticated successfully:', message.data);
});

wsClient.onMessage('chat_message', (message) => {
    console.log('Received chat message:', message.data);
    displayChatMessage(message.data);
});

wsClient.onMessage('private_message', (message) => {
    console.log('Received private message:', message.data);
    displayPrivateMessage(message.data);
});

// 连接并认证
wsClient.connect()
    .then(() => {
        return wsClient.authenticate('your-jwt-token');
    })
    .then(() => {
        return wsClient.joinRoom('general');
    })
    .catch(err => {
        console.error('Connection failed:', err);
    });
```

通过以上完整的WebSocket开发方案，Chi框架能够提供企业级的实时通信能力，支持大规模并发连接，提供可靠的消息传递和完善的集群支持。