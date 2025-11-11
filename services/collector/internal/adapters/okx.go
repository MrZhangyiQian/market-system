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

// OKXAdapter OKX 交易所适配器
type OKXAdapter struct {
	wsURL         string
	conn          *websocket.Conn
	connected     bool
	mu            sync.RWMutex
	handler       MessageHandler
	closeChan     chan struct{}
	reconnect     bool
	subscriptions []string      // 保存订阅列表
	lastPong      time.Time     // 最后一次PONG时间
	reconnectConf ReconnectConfig
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// NewOKXAdapter 创建 OKX 适配器
func NewOKXAdapter(wsURL string) ExchangeAdapter {
	if wsURL == "" {
		wsURL = "wss://ws.okx.com:8443/ws/v5/public"
	}
	return &OKXAdapter{
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
func (o *OKXAdapter) Connect() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(o.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to OKX: %w", err)
	}

	// 设置读取限制
	conn.SetReadLimit(512 * 1024) // 512KB

	// 设置Pong处理器
	conn.SetPongHandler(func(string) error {
		o.lastPong = time.Now()
		return nil
	})

	o.conn = conn
	o.connected = true

	// 启动消息读取
	go o.readMessages()

	// 启动心跳
	go o.keepAlive()

	log.Printf("[OKX] Connected to %s\n", o.wsURL)
	return nil
}

// Subscribe 订阅数据
func (o *OKXAdapter) Subscribe(symbols []string, channels []string) error {
	if !o.IsConnected() {
		return fmt.Errorf("not connected")
	}

	// 构建订阅参数
	args := make([]map[string]string, 0)
	subscriptions := make([]string, 0)

	for _, symbol := range symbols {
		// OKX使用 BTC-USDT 格式
		instId := o.formatSymbol(symbol)

		for _, channel := range channels {
			var okxChannel string
			switch channel {
			case constants.DataTypeTicker:
				okxChannel = "tickers"
			case constants.DataTypeDepth:
				okxChannel = "books5" // 5档深度，也可以使用books、books50-l2-tbt等
			case constants.DataTypeTrade:
				okxChannel = "trades"
			case constants.DataTypeKline:
				okxChannel = "candle1m" // 1分钟K线
			default:
				continue
			}

			args = append(args, map[string]string{
				"channel": okxChannel,
				"instId":  instId,
			})

			// 保存订阅信息
			subscriptions = append(subscriptions, fmt.Sprintf("%s:%s", okxChannel, instId))
		}
	}

	// OKX订阅消息格式
	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
	}

	o.mu.Lock()
	err := o.conn.WriteJSON(subMsg)
	if err == nil {
		// 保存订阅列表（用于重连后重新订阅）
		o.subscriptions = append(o.subscriptions, subscriptions...)
	}
	o.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("[OKX] Subscribed to %d channels\n", len(args))
	return nil
}

// OnMessage 设置消息处理器
func (o *OKXAdapter) OnMessage(handler MessageHandler) {
	o.handler = handler
}

// Close 关闭连接
func (o *OKXAdapter) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.reconnect = false
	close(o.closeChan)

	if o.conn != nil {
		o.connected = false
		return o.conn.Close()
	}
	return nil
}

// IsConnected 检查连接状态
func (o *OKXAdapter) IsConnected() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.connected
}

// GetName 获取交易所名称
func (o *OKXAdapter) GetName() string {
	return constants.ExchangeOKX
}

// readMessages 读取消息
func (o *OKXAdapter) readMessages() {
	defer func() {
		o.mu.Lock()
		o.connected = false
		o.mu.Unlock()
	}()

	for {
		select {
		case <-o.closeChan:
			return
		default:
			_, message, err := o.conn.ReadMessage()
			if err != nil {
				log.Printf("[OKX] Read error: %v\n", err)
				if o.reconnect {
					o.handleReconnect()
				}
				return
			}

			// 解析并处理消息
			o.handleMessage(message)
		}
	}
}

// handleMessage 处理消息
func (o *OKXAdapter) handleMessage(message []byte) {
	if o.handler == nil {
		return
	}

	var rawMsg map[string]interface{}
	if err := json.Unmarshal(message, &rawMsg); err != nil {
		log.Printf("[OKX] Failed to parse message: %v\n", err)
		return
	}

	// 检查是否是订阅响应或错误消息
	if event, ok := rawMsg["event"].(string); ok {
		if event == "subscribe" {
			log.Printf("[OKX] Subscription confirmed: %v\n", rawMsg["arg"])
			return
		} else if event == "error" {
			log.Printf("[OKX] Error: %v\n", rawMsg["msg"])
			return
		}
	}

	// 解析数据消息
	arg, ok := rawMsg["arg"].(map[string]interface{})
	if !ok {
		return
	}

	channel, ok := arg["channel"].(string)
	if !ok {
		return
	}

	data, ok := rawMsg["data"].([]interface{})
	if !ok || len(data) == 0 {
		return
	}

	// 解析第一条数据
	dataItem, ok := data[0].(map[string]interface{})
	if !ok {
		return
	}

	instId, ok := arg["instId"].(string)
	if !ok {
		return
	}

	// 转换为标准符号格式 BTC-USDT -> BTCUSDT
	symbol := o.parseSymbol(instId)
	timestamp := utils.GetCurrentTimestamp()

	var marketData *models.MarketData

	switch {
	case strings.HasPrefix(channel, "tickers"):
		marketData = o.parseTicker(dataItem, symbol, timestamp)
	case strings.HasPrefix(channel, "books"):
		marketData = o.parseDepth(dataItem, symbol, timestamp)
	case strings.HasPrefix(channel, "trades"):
		marketData = o.parseTrade(dataItem, symbol, timestamp)
	case strings.HasPrefix(channel, "candle"):
		marketData = o.parseKline(dataItem, symbol, channel, timestamp)
	}

	if marketData != nil {
		o.handler(marketData)
	}
}

// parseTicker 解析 Ticker 数据
func (o *OKXAdapter) parseTicker(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	ticker := &models.Ticker{
		Symbol:    symbol,
		LastPrice: parseFloat(raw["last"]),
		BidPrice:  parseFloat(raw["bidPx"]),
		AskPrice:  parseFloat(raw["askPx"]),
		High24h:   parseFloat(raw["high24h"]),
		Low24h:    parseFloat(raw["low24h"]),
		Volume24h: parseFloat(raw["vol24h"]),
		Timestamp: timestamp,
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeOKX,
		Symbol:    symbol,
		Type:      constants.DataTypeTicker,
		Timestamp: timestamp,
		Data:      ticker,
	}
}

// parseDepth 解析深度数据
func (o *OKXAdapter) parseDepth(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	bids := parsePriceLevels(raw["bids"])
	asks := parsePriceLevels(raw["asks"])

	depth := &models.OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: timestamp,
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeOKX,
		Symbol:    symbol,
		Type:      constants.DataTypeDepth,
		Timestamp: timestamp,
		Data:      depth,
	}
}

// parseTrade 解析交易数据
func (o *OKXAdapter) parseTrade(raw map[string]interface{}, symbol string, timestamp int64) *models.MarketData {
	side := constants.SideSell
	if sideStr, ok := raw["side"].(string); ok && sideStr == "buy" {
		side = constants.SideBuy
	}

	// OKX返回毫秒时间戳
	ts := int64(parseFloat(raw["ts"]))

	trade := &models.Trade{
		Symbol:    symbol,
		TradeID:   fmt.Sprintf("%v", raw["tradeId"]),
		Price:     parseFloat(raw["px"]),
		Amount:    parseFloat(raw["sz"]),
		Side:      side,
		Timestamp: ts,
	}

	return &models.MarketData{
		Exchange:  constants.ExchangeOKX,
		Symbol:    symbol,
		Type:      constants.DataTypeTrade,
		Timestamp: timestamp,
		Data:      trade,
	}
}

// parseKline 解析K线数据
func (o *OKXAdapter) parseKline(raw map[string]interface{}, symbol, channel string, timestamp int64) *models.MarketData {
	// 从channel提取间隔，如 candle1m -> 1m
	interval := "1m"
	if strings.HasPrefix(channel, "candle") {
		interval = strings.TrimPrefix(channel, "candle")
	}

	// OKX K线数据格式: [ts, o, h, l, c, vol, volCcy, volCcyQuote, confirm]
	dataArr, ok := raw["data"].([]interface{})
	if !ok || len(dataArr) < 6 {
		// 尝试直接解析字段
		kline := &models.Kline{
			Symbol:    symbol,
			Interval:  interval,
			OpenTime:  int64(parseFloat(raw["ts"])),
			Open:      parseFloat(raw["o"]),
			High:      parseFloat(raw["h"]),
			Low:       parseFloat(raw["l"]),
			Close:     parseFloat(raw["c"]),
			Volume:    parseFloat(raw["vol"]),
			QuoteVol:  parseFloat(raw["volCcy"]),
			TradeNum:  0, // OKX不提供交易数量
		}
		kline.CloseTime = kline.OpenTime + 60000 // 假设1分钟

		return &models.MarketData{
			Exchange:  constants.ExchangeOKX,
			Symbol:    symbol,
			Type:      constants.DataTypeKline,
			Timestamp: timestamp,
			Data:      kline,
		}
	}

	return nil
}

// keepAlive 保持连接
func (o *OKXAdapter) keepAlive() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.closeChan:
			return
		case <-ticker.C:
			if o.IsConnected() {
				// 检查心跳超时
				if time.Since(o.lastPong) > 60*time.Second {
					log.Println("[OKX] Pong timeout, reconnecting...")
					o.handleReconnect()
					return
				}

				// 发送ping
				o.mu.Lock()
				err := o.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
				o.mu.Unlock()
				if err != nil {
					log.Printf("[OKX] Ping error: %v\n", err)
				}
			}
		}
	}
}

// handleReconnect 处理重连（指数退避）
func (o *OKXAdapter) handleReconnect() {
	retries := 0
	delay := o.reconnectConf.InitialDelay

	for retries < o.reconnectConf.MaxRetries {
		log.Printf("[OKX] Attempting to reconnect (attempt %d/%d)...\n", retries+1, o.reconnectConf.MaxRetries)
		time.Sleep(delay)

		if err := o.Connect(); err == nil {
			log.Println("[OKX] Reconnected successfully")
			// 重新订阅
			o.resubscribe()
			return
		}

		retries++
		// 指数退避
		delay = time.Duration(float64(delay) * o.reconnectConf.Multiplier)
		if delay > o.reconnectConf.MaxDelay {
			delay = o.reconnectConf.MaxDelay
		}
	}

	log.Printf("[OKX] Max retries (%d) reached, giving up\n", o.reconnectConf.MaxRetries)
}

// resubscribe 重新订阅
func (o *OKXAdapter) resubscribe() error {
	if len(o.subscriptions) == 0 {
		return nil
	}

	// 构建订阅参数
	args := make([]map[string]string, 0, len(o.subscriptions))
	for _, sub := range o.subscriptions {
		parts := strings.Split(sub, ":")
		if len(parts) == 2 {
			args = append(args, map[string]string{
				"channel": parts[0],
				"instId":  parts[1],
			})
		}
	}

	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
	}

	o.mu.Lock()
	err := o.conn.WriteJSON(subMsg)
	o.mu.Unlock()

	if err != nil {
		log.Printf("[OKX] Resubscribe failed: %v\n", err)
		return err
	}

	log.Printf("[OKX] Resubscribed to %d channels\n", len(args))
	return nil
}

// formatSymbol 格式化符号 BTCUSDT -> BTC-USDT
func (o *OKXAdapter) formatSymbol(symbol string) string {
	// 简单处理，假设USDT结尾
	if strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USDT")
		return base + "-USDT"
	}
	// 其他情况返回原样
	return symbol
}

// parseSymbol 解析符号 BTC-USDT -> BTCUSDT
func (o *OKXAdapter) parseSymbol(instId string) string {
	return strings.ReplaceAll(instId, "-", "")
}
