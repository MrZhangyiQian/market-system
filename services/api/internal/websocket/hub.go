package websocket

import (
	"log"
	"sync"
)

// Hub 管理所有WebSocket客户端连接
type Hub struct {
	// 已注册的客户端
	clients map[*Client]bool

	// 客户端注册请求
	register chan *Client

	// 客户端注销请求
	unregister chan *Client

	// 广播消息到所有客户端
	broadcast chan *BroadcastMessage

	// 订阅管理器
	subscriptionManager *SubscriptionManager

	// 读写锁保护clients map
	mu sync.RWMutex

	// 停止信号
	stopChan chan struct{}
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	Channel string      // 频道名称，如 "ticker:BTCUSDT"
	Data    interface{} // 消息数据
}

// NewHub 创建新的Hub实例
func NewHub() *Hub {
	return &Hub{
		clients:             make(map[*Client]bool),
		register:            make(chan *Client, 256),
		unregister:          make(chan *Client, 256),
		broadcast:           make(chan *BroadcastMessage, 1024),
		subscriptionManager: NewSubscriptionManager(),
		stopChan:            make(chan struct{}),
	}
}

// Run 启动Hub，处理注册/注销/广播
func (h *Hub) Run() {
	log.Println("[WebSocket Hub] Starting...")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client registered, total clients: %d\n", h.ClientCount())

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.subscriptionManager.UnsubscribeAll(client)
				log.Printf("[WebSocket Hub] Client unregistered, total clients: %d\n", h.ClientCount())
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.broadcastToChannel(message)

		case <-h.stopChan:
			log.Println("[WebSocket Hub] Stopping...")
			h.closeAllClients()
			return
		}
	}
}

// broadcastToChannel 将消息广播到订阅了指定频道的客户端
func (h *Hub) broadcastToChannel(message *BroadcastMessage) {
	subscribers := h.subscriptionManager.GetSubscribers(message.Channel)

	if len(subscribers) == 0 {
		return
	}

	// 构造JSON消息
	jsonMessage := map[string]interface{}{
		"channel": message.Channel,
		"data":    message.Data,
	}

	successCount := 0
	failCount := 0

	for client := range subscribers {
		select {
		case client.send <- jsonMessage:
			successCount++
		default:
			// 客户端发送队列已满，关闭连接
			failCount++
			h.unregister <- client
		}
	}

	if successCount > 0 || failCount > 0 {
		log.Printf("[WebSocket Hub] Broadcast to channel '%s': success=%d, failed=%d\n",
			message.Channel, successCount, failCount)
	}
}

// Broadcast 广播消息到指定频道
func (h *Hub) Broadcast(channel string, data interface{}) {
	h.broadcast <- &BroadcastMessage{
		Channel: channel,
		Data:    data,
	}
}

// Register 注册客户端
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Subscribe 订阅频道
func (h *Hub) Subscribe(client *Client, channel string) {
	h.subscriptionManager.Subscribe(client, channel)
	log.Printf("[WebSocket Hub] Client subscribed to channel: %s\n", channel)
}

// Unsubscribe 取消订阅频道
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.subscriptionManager.Unsubscribe(client, channel)
	log.Printf("[WebSocket Hub] Client unsubscribed from channel: %s\n", channel)
}

// ClientCount 返回当前连接的客户端数量
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetSubscriptions 获取客户端的所有订阅
func (h *Hub) GetSubscriptions(client *Client) []string {
	return h.subscriptionManager.GetClientSubscriptions(client)
}

// Stop 停止Hub
func (h *Hub) Stop() {
	close(h.stopChan)
}

// closeAllClients 关闭所有客户端连接
func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		h.subscriptionManager.UnsubscribeAll(client)
	}
	h.clients = make(map[*Client]bool)
	log.Println("[WebSocket Hub] All clients closed")
}
