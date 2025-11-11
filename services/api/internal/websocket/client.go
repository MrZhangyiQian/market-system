package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 客户端写超时
	writeWait = 10 * time.Second

	// 客户端读超时（心跳间隔 * 倍数）
	pongWait = 60 * time.Second

	// 心跳间隔（必须小于pongWait）
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client WebSocket客户端包装器
type Client struct {
	// Hub引用
	hub *Hub

	// WebSocket连接
	conn *websocket.Conn

	// 发送消息的缓冲通道
	send chan interface{}

	// 客户端ID（可选，用于日志）
	id string
}

// NewClient 创建新的客户端实例
func NewClient(hub *Hub, conn *websocket.Conn, id string) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan interface{}, 256),
		id:   id,
	}
}

// readPump 从WebSocket连接读取消息
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	// 设置读取限制和超时
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket Client %s] Read error: %v\n", c.id, err)
			}
			break
		}

		// 处理客户端消息
		c.handleMessage(message)
	}
}

// handleMessage 处理客户端发送的消息
func (c *Client) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("[WebSocket Client %s] Invalid JSON: %v\n", c.id, err)
		c.sendError("Invalid JSON format")
		return
	}

	action, ok := msg["action"].(string)
	if !ok {
		c.sendError("Missing 'action' field")
		return
	}

	switch action {
	case "subscribe":
		c.handleSubscribe(msg)
	case "unsubscribe":
		c.handleUnsubscribe(msg)
	case "ping":
		c.handlePing()
	default:
		c.sendError("Unknown action: " + action)
	}
}

// handleSubscribe 处理订阅请求
func (c *Client) handleSubscribe(msg map[string]interface{}) {
	channel, ok := msg["channel"].(string)
	if !ok {
		c.sendError("Missing 'channel' field")
		return
	}

	symbol, _ := msg["symbol"].(string) // symbol可选

	// 构建完整的频道名
	fullChannel := c.buildChannelName(channel, symbol)

	c.hub.Subscribe(c, fullChannel)

	// 发送订阅成功响应
	c.sendResponse("subscribed", map[string]interface{}{
		"channel": channel,
		"symbol":  symbol,
	})
}

// handleUnsubscribe 处理取消订阅请求
func (c *Client) handleUnsubscribe(msg map[string]interface{}) {
	channel, ok := msg["channel"].(string)
	if !ok {
		c.sendError("Missing 'channel' field")
		return
	}

	symbol, _ := msg["symbol"].(string) // symbol可选

	// 构建完整的频道名
	fullChannel := c.buildChannelName(channel, symbol)

	c.hub.Unsubscribe(c, fullChannel)

	// 发送取消订阅成功响应
	c.sendResponse("unsubscribed", map[string]interface{}{
		"channel": channel,
		"symbol":  symbol,
	})
}

// handlePing 处理ping请求
func (c *Client) handlePing() {
	c.sendResponse("pong", nil)
}

// buildChannelName 构建频道名称
// 格式: channel:symbol 或 channel (如果symbol为空)
func (c *Client) buildChannelName(channel, symbol string) string {
	if symbol != "" {
		return channel + ":" + symbol
	}
	return channel
}

// writePump 向WebSocket连接写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub关闭了send通道
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 将消息编码为JSON
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("[WebSocket Client %s] JSON marshal error: %v\n", c.id, err)
				continue
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(jsonData)

			// 批量发送队列中的其他消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				additionalMsg := <-c.send
				additionalJSON, err := json.Marshal(additionalMsg)
				if err != nil {
					log.Printf("[WebSocket Client %s] JSON marshal error: %v\n", c.id, err)
					continue
				}
				w.Write(additionalJSON)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendError 发送错误响应
func (c *Client) sendError(errMsg string) {
	response := map[string]interface{}{
		"type":  "error",
		"error": errMsg,
	}
	select {
	case c.send <- response:
	default:
		log.Printf("[WebSocket Client %s] Send buffer full, dropping error message\n", c.id)
	}
}

// sendResponse 发送响应消息
func (c *Client) sendResponse(responseType string, data interface{}) {
	response := map[string]interface{}{
		"type": responseType,
	}
	if data != nil {
		response["data"] = data
	}
	select {
	case c.send <- response:
	default:
		log.Printf("[WebSocket Client %s] Send buffer full, dropping response\n", c.id)
	}
}

// Start 启动客户端的读写goroutine
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}
