package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"market-system/common/constants"
	"market-system/common/models"
	"market-system/common/utils"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceAdapter Binance 交易所适配器
type BinanceAdapter struct {
	wsURL         string
	conn          *websocket.Conn
	connected     bool
	mu            sync.RWMutex
	handler       MessageHandler
	closeChan     chan struct{}
	reconnect     bool
	subscriptions []string    // 保存订阅列表
	lastPong      time.Time   // 最后一次PONG时间
	reconnectConf ReconnectConfig
}

// NewBinanceAdapter 创建 Binance 适配器
func NewBinanceAdapter(wsURL string) ExchangeAdapter {
	if wsURL == "" {
		wsURL = "wss://stream.binance.com:9443/ws"
	}
	return &BinanceAdapter{
		wsURL:     wsURL,
		closeChan: make(chan struct{}),
		reconnect: true,
		lastPong:  time.Now(),
		reconnectConf: ReconnectConfig{
			MaxRetries:   10,
			InitialDelay: 1 * time.Second,
			MaxDelay:     60 * time.Second,
			Multiplier:   2.0,
		},
	}
}

// Connect 建立连接
func (b *BinanceAdapter) Connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(b.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to binance: %w", err)
	}

	// 设置读取限制
	conn.SetReadLimit(512 * 1024) // 512KB

	// 设置Pong处理器
	conn.SetPongHandler(func(string) error {
		b.lastPong = time.Now()
		return nil
	})

	b.conn = conn
	b.connected = true

	// 启动消息读取
	go b.readMessages()

	// 启动心跳
	go b.keepAlive()

	log.Printf("[Binance] Connected to %s\n", b.wsURL)
	return nil
}

// Subscribe 订阅数据
func (b *BinanceAdapter) Subscribe(symbols []string, channels []string) error {
	if !b.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// 构建订阅消息
	streams := make([]string, 0)
	for _, symbol := range symbols {
		symbolLower := strings.ToLower(symbol)
		for _, channel := range channels {
			switch channel {
			case constants.DataTypeTicker:
				streams = append(streams, fmt.Sprintf("%s@ticker", symbolLower))
			case constants.DataTypeDepth:
				streams = append(streams, fmt.Sprintf("%s@depth20@100ms", symbolLower))
			case constants.DataTypeTrade:
				streams = append(streams, fmt.Sprintf("%s@trade", symbolLower))
			case constants.DataTypeKline:
				streams = append(streams, fmt.Sprintf("%s@kline_1m", symbolLower))
			}
		}
	}

	subMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": streams,
		"id":     1,
	}

	b.mu.Lock()
	err := b.conn.WriteJSON(subMsg)
	if err == nil {
		// 保存订阅列表（用于重连后重新订阅）
		b.subscriptions = append(b.subscriptions, streams...)
	}
	b.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("[Binance] Subscribed to %d streams\n", len(streams))
	return nil
}

// OnMessage 设置消息处理器
func (b *BinanceAdapter) OnMessage(handler MessageHandler) {
	b.handler = handler
}

// Close 关闭连接
func (b *BinanceAdapter) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.reconnect = false
	close(b.closeChan)

	if b.conn != nil {
		b.connected = false
		return b.conn.Close()
	}
	return nil
}

// IsConnected 检查连接状态
func (b *BinanceAdapter) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// GetName 获取交易所名称
func (b *BinanceAdapter) GetName() string {
	return constants.ExchangeBinance
}

// readMessages 读取消息
func (b *BinanceAdapter) readMessages() {
	defer func() {
		b.mu.Lock()
		b.connected = false
		b.mu.Unlock()
	}()

	for {
		select {
		case <-b.closeChan:
			return
		default:
			_, message, err := b.conn.ReadMessage()
			if err != nil {
				log.Printf("[Binance] Read error: %v\n", err)
				if b.reconnect {
					b.handleReconnect()
				}
				return
			}

			// 解析并处理消息
			b.handleMessage(message)
		}
	}
}

// handleMessage 处理消息
func (b *BinanceAdapter) handleMessage(message []byte) {
	if b.handler == nil {
		return
	}

	var rawMsg map[string]interface{}
	if err := json.Unmarshal(message, &rawMsg); err != nil {
		log.Printf("[Binance] Failed to parse message: %v\n", err)
		return
	}

	// 检查是否是订阅响应
	if _, ok := rawMsg["result"]; ok {
		return
	}

	// 获取事件类型
	eventType, ok := rawMsg["e"].(string)
	if !ok {
		return
	}

	timestamp := utils.GetCurrentTimestamp()
	symbol := strings.ToUpper(rawMsg["s"].(string))

	var marketData *models.MarketData

	switch eventType {
	case "24hrTicker":
		marketData = b.parseTicker(rawMsg, symbol, timestamp)
	case "depthUpdate":
		marketData = b.parseDepth(rawMsg, symbol, timestamp)
	case "trade":
		marketData = b.parseTrade(rawMsg, symbol, timestamp)
	case "kline":
		marketData = b.parseKline(rawMsg, symbol, timestamp)
	}

	if marketData != nil {
		b.handler(marketData)
	}
}

// parseTicker 解析 Ticker 数据
func (b *BinanceAdapter) parseTicker(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	ticker := &models.Ticker{
		Symbol:    symbol,
		LastPrice: parseFloat(raw["c"]),
		BidPrice:  parseFloat(raw["b"]),
		AskPrice:  parseFloat(raw["a"]),
		High24h:   parseFloat(raw["h"]),
		Low24h:    parseFloat(raw["l"]),
		Volume24h: parseFloat(raw["v"]),
		Timestamp: timestamp,
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeBinance,
		Symbol:    symbol,
		Type:      constants.DataTypeTicker,
		Timestamp: timestamp,
		Data:      ticker,
	}
}

// parseDepth 解析深度数据
func (b *BinanceAdapter) parseDepth(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	bids := parsePriceLevels(raw["b"])
	asks := parsePriceLevels(raw["a"])

	depth := &models.OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: timestamp,
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeBinance,
		Symbol:    symbol,
		Type:      constants.DataTypeDepth,
		Timestamp: timestamp,
		Data:      depth,
	}
}

// parseTrade 解析交易数据
func (b *BinanceAdapter) parseTrade(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	side := constants.SideSell
	if m, ok := raw["m"].(bool); ok && m {
		side = constants.SideBuy
	}

	// 安全获取时间戳
	ts := timestamp
	if T, ok := raw["T"].(float64); ok {
		ts = int64(T)
	}

	trade := &models.Trade{
		Symbol:    symbol,
		TradeID:   fmt.Sprintf("%v", raw["t"]),
		Price:     parseFloat(raw["p"]),
		Amount:    parseFloat(raw["q"]),
		Side:      side,
		Timestamp: ts,
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeBinance,
		Symbol:    symbol,
		Type:      constants.DataTypeTrade,
		Timestamp: timestamp,
		Data:      trade,
	}
}

// parseKline 解析K线数据
func (b *BinanceAdapter) parseKline(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	k := raw["k"].(map[string]interface{})

	kline := &models.Kline{
		Symbol:    symbol,
		Interval:  k["i"].(string),
		OpenTime:  int64(k["t"].(float64)),
		CloseTime: int64(k["T"].(float64)),
		Open:      parseFloat(k["o"]),
		High:      parseFloat(k["h"]),
		Low:       parseFloat(k["l"]),
		Close:     parseFloat(k["c"]),
		Volume:    parseFloat(k["v"]),
		QuoteVol:  parseFloat(k["q"]),
		TradeNum:  int64(k["n"].(float64)),
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeBinance,
		Symbol:    symbol,
		Type:      constants.DataTypeKline,
		Timestamp: timestamp,
		Data:      kline,
	}
}

// keepAlive 保持连接
func (b *BinanceAdapter) keepAlive() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.closeChan:
			return
		case <-ticker.C:
			if b.IsConnected() {
				// 检查心跳超时
				if time.Since(b.lastPong) > 60*time.Second {
					log.Println("[Binance] Pong timeout, reconnecting...")
					b.handleReconnect()
					return
				}

				// 发送ping
				b.mu.Lock()
				err := b.conn.WriteMessage(websocket.PingMessage, []byte("ping"))
				b.mu.Unlock()
				if err != nil {
					log.Printf("[Binance] Ping error: %v\n", err)
				}
			}
		}
	}
}

// handleReconnect 处理重连（指数退避）
func (b *BinanceAdapter) handleReconnect() {
	retries := 0
	delay := b.reconnectConf.InitialDelay

	for retries < b.reconnectConf.MaxRetries {
		log.Printf("[Binance] Attempting to reconnect (attempt %d/%d)...\n", retries+1, b.reconnectConf.MaxRetries)
		time.Sleep(delay)

		if err := b.Connect(); err == nil {
			log.Println("[Binance] Reconnected successfully")
			// 重新订阅
			b.resubscribe()
			return
		}

		retries++
		// 指数退避
		delay = time.Duration(float64(delay) * b.reconnectConf.Multiplier)
		if delay > b.reconnectConf.MaxDelay {
			delay = b.reconnectConf.MaxDelay
		}
	}

	log.Printf("[Binance] Max retries (%d) reached, giving up\n", b.reconnectConf.MaxRetries)
}

// resubscribe 重新订阅
func (b *BinanceAdapter) resubscribe() error {
	if len(b.subscriptions) == 0 {
		return nil
	}

	subMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": b.subscriptions,
		"id":     time.Now().Unix(),
	}

	b.mu.Lock()
	err := b.conn.WriteJSON(subMsg)
	b.mu.Unlock()

	if err != nil {
		log.Printf("[Binance] Resubscribe failed: %v\n", err)
		return err
	}

	log.Printf("[Binance] Resubscribed to %d streams\n", len(b.subscriptions))
	return nil
}

// 辅助函数
func parseFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

func parsePriceLevels(v interface{}) []models.PriceLevel {
	if v == nil {
		return []models.PriceLevel{}
	}

	arr, ok := v.([]interface{})
	if !ok {
		return []models.PriceLevel{}
	}

	levels := make([]models.PriceLevel, 0, len(arr))
	for _, item := range arr {
		level, ok := item.([]interface{})
		if !ok || len(level) < 2 {
			continue
		}

		levels = append(levels, models.PriceLevel{
			Price:  parseFloat(level[0]),
			Amount: parseFloat(level[1]),
		})
	}

	return levels
}
