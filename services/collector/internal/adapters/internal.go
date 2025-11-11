package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"market-system/common/constants"
	"market-system/common/models"
	"market-system/common/utils"
	"net/http"
	"sync"
)

// InternalAdapter 内部数据源适配器
type InternalAdapter struct {
	name       string
	httpServer *http.Server
	handler    MessageHandler
	mu         sync.RWMutex
	connected  bool
	port       int
}

// NewInternalAdapter 创建内部适配器
func NewInternalAdapter(port int) ExchangeAdapter {
	if port == 0 {
		port = 9001 // 默认端口
	}
	return &InternalAdapter{
		name: constants.ExchangeInternal,
		port: port,
	}
}

// Connect 启动 HTTP 服务器接收内部数据
func (a *InternalAdapter) Connect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/api/market/trade", a.handleTrade)
	mux.HandleFunc("/api/market/depth", a.handleDepth)
	mux.HandleFunc("/api/market/ticker", a.handleTicker)
	mux.HandleFunc("/health", a.handleHealth)

	// 创建 HTTP 服务器
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Printf("[Internal] HTTP server listening on port %d\n", a.port)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Internal] HTTP server error: %v\n", err)
		}
	}()

	a.connected = true
	log.Printf("[Internal] Adapter started on port %d\n", a.port)
	return nil
}

// Subscribe 内部适配器不需要订阅
func (a *InternalAdapter) Subscribe(symbols []string, channels []string) error {
	log.Printf("[Internal] Listening for data on symbols: %v, channels: %v\n", symbols, channels)
	return nil
}

// OnMessage 设置消息处理器
func (a *InternalAdapter) OnMessage(handler MessageHandler) {
	a.handler = handler
}

// Close 关闭服务器
func (a *InternalAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.httpServer != nil {
		if err := a.httpServer.Close(); err != nil {
			return err
		}
	}
	a.connected = false
	log.Println("[Internal] Adapter closed")
	return nil
}

// IsConnected 检查连接状态
func (a *InternalAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected
}

// GetName 获取名称
func (a *InternalAdapter) GetName() string {
	return a.name
}

// ========== HTTP Handlers ==========

// handleTrade 处理交易数据
func (a *InternalAdapter) handleTrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var trade models.InternalTradeMessage
	if err := json.NewDecoder(r.Body).Decode(&trade); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 转换为标准 Trade 格式
	standardTrade := &models.Trade{
		Symbol:    trade.Symbol,
		TradeID:   fmt.Sprintf("%d", trade.TradeID),
		Price:     trade.Price,
		Amount:    trade.Amount,
		Side:      trade.Side,
		Timestamp: trade.Timestamp,
	}

	// 创建 MarketData
	marketData := &models.MarketData{
		Exchange:  constants.ExchangeInternal,
		Symbol:    trade.Symbol,
		Type:      constants.DataTypeTrade,
		Source:    constants.SourceInternal,
		Timestamp: trade.Timestamp,
		Data:      standardTrade,
	}

	// 调用处理器
	if a.handler != nil {
		a.handler(marketData)
	}

	// 响应成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"msg":  "success",
	})

	log.Printf("[Internal] Trade received: %s @ %.2f, amount: %.4f\n",
		trade.Symbol, trade.Price, trade.Amount)
}

// handleDepth 处理深度数据
func (a *InternalAdapter) handleDepth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var depth models.InternalDepthMessage
	if err := json.NewDecoder(r.Body).Decode(&depth); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 转换为标准 OrderBook 格式
	orderBook := &models.OrderBook{
		Symbol:    depth.Symbol,
		Bids:      depth.Bids,
		Asks:      depth.Asks,
		Timestamp: depth.Timestamp,
	}

	// 创建 MarketData
	marketData := &models.MarketData{
		Exchange:  constants.ExchangeInternal,
		Symbol:    depth.Symbol,
		Type:      constants.DataTypeDepth,
		Source:    constants.SourceInternal,
		Timestamp: depth.Timestamp,
		Data:      orderBook,
	}

	// 调用处理器
	if a.handler != nil {
		a.handler(marketData)
	}

	// 响应成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"msg":  "success",
	})

	log.Printf("[Internal] Depth received: %s, bids: %d, asks: %d\n",
		depth.Symbol, len(depth.Bids), len(depth.Asks))
}

// handleTicker 处理 Ticker 数据
func (a *InternalAdapter) handleTicker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var ticker models.Ticker
	if err := json.NewDecoder(r.Body).Decode(&ticker); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 设置时间戳
	if ticker.Timestamp == 0 {
		ticker.Timestamp = utils.GetCurrentTimestamp()
	}

	// 创建 MarketData
	marketData := &models.MarketData{
		Exchange:  constants.ExchangeInternal,
		Symbol:    ticker.Symbol,
		Type:      constants.DataTypeTicker,
		Source:    constants.SourceInternal,
		Timestamp: ticker.Timestamp,
		Data:      ticker,
	}

	// 调用处理器
	if a.handler != nil {
		a.handler(marketData)
	}

	// 响应成功
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code": 0,
		"msg":  "success",
	})

	log.Printf("[Internal] Ticker received: %s @ %.2f\n", ticker.Symbol, ticker.LastPrice)
}

// handleHealth 健康检查
func (a *InternalAdapter) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"name":   "internal-adapter",
	})
}
