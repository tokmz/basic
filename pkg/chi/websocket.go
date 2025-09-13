package chi

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler WebSocket处理函数类型
type WebSocketHandler func(*WebSocketConn)

// WebSocketConn WebSocket连接封装
type WebSocketConn struct {
	*websocket.Conn
	hub      *WebSocketHub
	clientID string
	userID   interface{}
	metadata map[string]interface{}
	send     chan []byte
	mu       sync.RWMutex
}

// NewWebSocketConn 创建WebSocket连接
func NewWebSocketConn(conn *websocket.Conn, hub *WebSocketHub) *WebSocketConn {
	return &WebSocketConn{
		Conn:     conn,
		hub:      hub,
		clientID: generateID(),
		metadata: make(map[string]interface{}),
		send:     make(chan []byte, 256),
	}
}

// ClientID 获取客户端ID
func (c *WebSocketConn) ClientID() string {
	return c.clientID
}

// UserID 获取用户ID
func (c *WebSocketConn) UserID() interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.userID
}

// SetUserID 设置用户ID
func (c *WebSocketConn) SetUserID(userID interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userID = userID
}

// SetMetadata 设置元数据
func (c *WebSocketConn) SetMetadata(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metadata[key] = value
}

// GetMetadata 获取元数据
func (c *WebSocketConn) GetMetadata(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, exists := c.metadata[key]
	return value, exists
}

// Send 发送消息
func (c *WebSocketConn) Send(data []byte) error {
	select {
	case c.send <- data:
		return nil
	default:
		close(c.send)
		return ErrConnectionClosed
	}
}

// SendJSON 发送JSON消息
func (c *WebSocketConn) SendJSON(v interface{}) error {
	return c.WriteJSON(v)
}

// SendText 发送文本消息
func (c *WebSocketConn) SendText(message string) error {
	return c.Send([]byte(message))
}

// CloseWithMessage 带消息关闭连接
func (c *WebSocketConn) CloseWithMessage(code int, text string) error {
	message := websocket.FormatCloseMessage(code, text)
	c.WriteMessage(websocket.CloseMessage, message)
	return c.Close()
}

// writePump 写入消息泵
func (c *WebSocketConn) writePump() {
	ticker := time.NewTicker(c.hub.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.SetWriteDeadline(time.Now().Add(c.hub.config.WriteWait))
			if !ok {
				c.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送排队的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.SetWriteDeadline(time.Now().Add(c.hub.config.WriteWait))
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 读取消息泵
func (c *WebSocketConn) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.Close()
	}()

	c.SetReadLimit(c.hub.config.MaxMessageSize)
	c.SetReadDeadline(time.Now().Add(c.hub.config.PongWait))
	c.SetPongHandler(func(string) error {
		c.SetReadDeadline(time.Now().Add(c.hub.config.PongWait))
		return nil
	})

	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 记录错误日志
			}
			break
		}

		// 处理接收到的消息
		c.hub.handleMessage(c, message)
	}
}

// WebSocketHub WebSocket连接管理器
type WebSocketHub struct {
	config     *WebSocketConfig
	clients    map[*WebSocketConn]bool
	rooms      map[string]map[*WebSocketConn]bool
	broadcast  chan []byte
	register   chan *WebSocketConn
	unregister chan *WebSocketConn
	mu         sync.RWMutex

	// 消息处理器
	messageHandlers map[string]func(*WebSocketConn, []byte)
}

// NewWebSocketHub 创建WebSocket连接管理器
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		config:          DefaultWebSocketConfig(),
		clients:         make(map[*WebSocketConn]bool),
		rooms:           make(map[string]map[*WebSocketConn]bool),
		broadcast:       make(chan []byte),
		register:        make(chan *WebSocketConn),
		unregister:      make(chan *WebSocketConn),
		messageHandlers: make(map[string]func(*WebSocketConn, []byte)),
	}
}

// SetConfig 设置配置
func (h *WebSocketHub) SetConfig(config *WebSocketConfig) {
	h.config = config
}

// OnMessage 注册消息处理器
func (h *WebSocketHub) OnMessage(messageType string, handler func(*WebSocketConn, []byte)) {
	h.messageHandlers[messageType] = handler
}

// Run 运行WebSocket管理器
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			// 启动读写泵
			go client.writePump()
			go client.readPump()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// 从所有房间移除
				for _, room := range h.rooms {
					delete(room, client)
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register 注册客户端
func (h *WebSocketHub) Register(client *WebSocketConn) {
	h.register <- client
}

// Unregister 注销客户端
func (h *WebSocketHub) Unregister(client *WebSocketConn) {
	h.unregister <- client
}

// Broadcast 广播消息
func (h *WebSocketHub) Broadcast(message []byte) {
	h.broadcast <- message
}

// BroadcastJSON 广播JSON消息
func (h *WebSocketHub) BroadcastJSON(v interface{}) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if err := client.SendJSON(v); err != nil {
			// 记录错误但继续发送给其他客户端
		}
	}
	return nil
}

// JoinRoom 加入房间
func (h *WebSocketHub) JoinRoom(client *WebSocketConn, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*WebSocketConn]bool)
	}
	h.rooms[room][client] = true
}

// LeaveRoom 离开房间
func (h *WebSocketHub) LeaveRoom(client *WebSocketConn, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if roomClients, exists := h.rooms[room]; exists {
		delete(roomClients, client)
		if len(roomClients) == 0 {
			delete(h.rooms, room)
		}
	}
}

// BroadcastToRoom 向房间广播消息
func (h *WebSocketHub) BroadcastToRoom(room string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if roomClients, exists := h.rooms[room]; exists {
		for client := range roomClients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
				delete(roomClients, client)
			}
		}
	}
}

// GetClientCount 获取客户端数量
func (h *WebSocketHub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetRoomCount 获取房间数量
func (h *WebSocketHub) GetRoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}

// GetRoomClients 获取房间客户端数量
func (h *WebSocketHub) GetRoomClients(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if roomClients, exists := h.rooms[room]; exists {
		return len(roomClients)
	}
	return 0
}

// handleMessage 处理消息
func (h *WebSocketHub) handleMessage(client *WebSocketConn, message []byte) {
	// 这里可以解析消息类型并调用相应的处理器
	// 简单示例：假设消息格式为 "type:data"
	// 实际项目中可能使用JSON等格式
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中应该进行适当的来源检查
		return true
	},
}

// WebSocketUpgradeHandler WebSocket升级处理器
func WebSocketUpgradeHandler(config *WebSocketConfig, handler WebSocketHandler) gin.HandlerFunc {
	if config != nil {
		upgrader.ReadBufferSize = config.ReadBufferSize
		upgrader.WriteBufferSize = config.WriteBufferSize
		upgrader.EnableCompression = config.EnableCompression
	}

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to upgrade connection",
				"details": err.Error(),
			})
			return
		}

		wsConn := NewWebSocketConn(conn, nil)

		// 设置连接参数
		if config != nil {
			conn.SetReadLimit(config.MaxMessageSize)
			conn.SetReadDeadline(time.Now().Add(config.PongWait))
			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(config.PongWait))
				return nil
			})
		}

		// 调用处理函数
		handler(wsConn)
	}
}

// generateID 生成唯一ID
func generateID() string {
	return string(rune(time.Now().UnixNano()))
}

// 错误定义
var (
	ErrConnectionClosed = fmt.Errorf("connection closed")
)
