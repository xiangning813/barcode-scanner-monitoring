package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"userclient/internal/config"
	"userclient/pkg/barcode"
)

// Client WebSocket客户端
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	hub    *Hub
	logger *logrus.Logger
}

// Hub WebSocket连接管理中心
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	config     *config.WebSocketConfig
	logger     *logrus.Logger
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
}

// Message WebSocket消息结构
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
	Time time.Time   `json:"time"`
}

// NewHub 创建新的WebSocket Hub
func NewHub(cfg *config.WebSocketConfig, logger *logrus.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		config:     cfg,
		logger:     logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.ReadBufferSize,
			WriteBufferSize: cfg.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return cfg.CheckOrigin // 根据配置决定是否检查来源
			},
		},
	}
}

// Run 启动Hub
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub 已启动")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			h.logger.WithField("client_count", len(h.clients)).Info("新客户端连接")

			// 发送欢迎消息
			welcomeMsg := Message{
				Type: "welcome",
				Data: map[string]string{
					"message": "WebSocket连接成功，等待扫码数据...",
				},
				Time: time.Now(),
			}

			if data, err := json.Marshal(welcomeMsg); err == nil {
				select {
				case client.send <- data:
				default:
					close(client.send)
					h.mu.Lock()
					delete(h.clients, client)
					h.mu.Unlock()
				}
			}

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.logger.WithField("client_count", len(h.clients)).Info("客户端断开连接")
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

// HandleWebSocket 处理WebSocket连接
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WithError(err).Error("WebSocket升级失败")
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    h,
		logger: h.logger,
	}

	client.hub.register <- client

	// 启动客户端的读写协程
	go client.writePump()
	go client.readPump()
}

// BroadcastBarcode 广播条码数据
func (h *Hub) BroadcastBarcode(barcodeData *barcode.BarcodeData) {
	message := Message{
		Type: "barcode",
		Data: barcodeData,
		Time: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("序列化条码数据失败")
		return
	}

	select {
	case h.broadcast <- data:
		h.logger.WithField("client_count", len(h.clients)).Debug("条码数据已广播")
	default:
		h.logger.Warn("广播通道已满，丢弃消息")
	}
}

// GetClientCount 获取当前连接的客户端数量
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Close 关闭Hub
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		client.conn.Close()
		close(client.send)
	}

	close(h.broadcast)
	close(h.register)
	close(h.unregister)

	h.logger.Info("WebSocket Hub 已关闭")
}

// readPump 读取客户端消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(c.hub.config.PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(c.hub.config.PongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.WithError(err).Error("WebSocket读取错误")
			}
			break
		}
	}
}

// writePump 向客户端写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(c.hub.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.hub.config.WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
