package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 允许跨域（生产环境应该配置具体的允许源）
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler WebSocket HTTP处理器
type Handler struct {
	hub *Hub
}

// NewHandler 创建新的WebSocket处理器
func NewHandler(hub *Hub) *Handler {
	return &Handler{
		hub: hub,
	}
}

// ServeHTTP 处理WebSocket连接请求
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket Handler] Failed to upgrade connection: %v\n", err)
		return
	}

	// 生成唯一的客户端ID
	clientID := uuid.New().String()

	// 创建客户端实例
	client := NewClient(h.hub, conn, clientID)

	// 注册客户端到Hub
	h.hub.Register(client)

	// 发送欢迎消息
	welcomeMsg := map[string]interface{}{
		"type":      "welcome",
		"client_id": clientID,
		"timestamp": time.Now().Unix(),
		"message":   "Connected to Market WebSocket Server",
	}
	select {
	case client.send <- welcomeMsg:
	default:
		log.Printf("[WebSocket Handler] Failed to send welcome message to client %s\n", clientID)
	}

	// 启动客户端的读写goroutine
	client.Start()

	log.Printf("[WebSocket Handler] New client connected: %s, remote: %s\n", clientID, r.RemoteAddr)
}
