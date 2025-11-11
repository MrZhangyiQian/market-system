package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Broadcaster 从Redis订阅消息并广播到WebSocket客户端
type Broadcaster struct {
	hub         *Hub
	redisClient *redis.Client
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewBroadcaster 创建新的广播器
func NewBroadcaster(hub *Hub, redisClient *redis.Client) *Broadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	return &Broadcaster{
		hub:         hub,
		redisClient: redisClient,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动Redis订阅
func (b *Broadcaster) Start() {
	log.Println("[WebSocket Broadcaster] Starting Redis subscription...")

	// 订阅所有市场数据频道
	// 频道格式: market:ticker:BTCUSDT, market:depth:BTCUSDT, market:trade:BTCUSDT, market:kline:BTCUSDT:1m
	pubsub := b.redisClient.PSubscribe(b.ctx, "market:*")

	// 确保pubsub关闭
	defer func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("[WebSocket Broadcaster] Failed to close pubsub: %v\n", err)
		}
	}()

	// 等待订阅确认
	_, err := pubsub.Receive(b.ctx)
	if err != nil {
		log.Printf("[WebSocket Broadcaster] Failed to subscribe: %v\n", err)
		return
	}

	log.Println("[WebSocket Broadcaster] Successfully subscribed to market:*")

	// 接收消息
	ch := pubsub.Channel()
	for {
		select {
		case <-b.ctx.Done():
			log.Println("[WebSocket Broadcaster] Stopping...")
			return

		case msg, ok := <-ch:
			if !ok {
				log.Println("[WebSocket Broadcaster] Channel closed")
				return
			}

			b.handleRedisMessage(msg)
		}
	}
}

// handleRedisMessage 处理从Redis接收到的消息
func (b *Broadcaster) handleRedisMessage(msg *redis.Message) {
	// 解析频道名称
	// 格式: market:ticker:BTCUSDT -> ticker:BTCUSDT
	// 格式: market:kline:BTCUSDT:1m -> kline:BTCUSDT:1m
	channel := strings.TrimPrefix(msg.Channel, "market:")

	// 解析消息数据
	var data interface{}
	if err := json.Unmarshal([]byte(msg.Payload), &data); err != nil {
		log.Printf("[WebSocket Broadcaster] Failed to parse message: %v\n", err)
		return
	}

	// 广播到订阅了该频道的客户端
	b.hub.Broadcast(channel, data)
}

// Stop 停止广播器
func (b *Broadcaster) Stop() {
	b.cancel()
}

// BroadcastTicker 广播Ticker消息（供Processor服务调用）
func (b *Broadcaster) BroadcastTicker(symbol string, ticker interface{}) error {
	channel := "ticker:" + symbol
	data, err := json.Marshal(ticker)
	if err != nil {
		return err
	}

	// 发布到Redis
	return b.redisClient.Publish(b.ctx, "market:"+channel, data).Err()
}

// BroadcastDepth 广播深度消息
func (b *Broadcaster) BroadcastDepth(symbol string, depth interface{}) error {
	channel := "depth:" + symbol
	data, err := json.Marshal(depth)
	if err != nil {
		return err
	}

	return b.redisClient.Publish(b.ctx, "market:"+channel, data).Err()
}

// BroadcastTrade 广播成交消息
func (b *Broadcaster) BroadcastTrade(symbol string, trade interface{}) error {
	channel := "trade:" + symbol
	data, err := json.Marshal(trade)
	if err != nil {
		return err
	}

	return b.redisClient.Publish(b.ctx, "market:"+channel, data).Err()
}

// BroadcastKline 广播K线消息
func (b *Broadcaster) BroadcastKline(symbol, interval string, kline interface{}) error {
	channel := "kline:" + symbol + ":" + interval
	data, err := json.Marshal(kline)
	if err != nil {
		return err
	}

	return b.redisClient.Publish(b.ctx, "market:"+channel, data).Err()
}
