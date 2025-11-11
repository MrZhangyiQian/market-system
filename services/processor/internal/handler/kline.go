package handler

import (
	"log"
	"market-system/common/constants"
	"market-system/common/models"
	"market-system/common/utils"
	"sync"
)

// KlineHandler K线处理器
type KlineHandler struct {
	aggregators map[string]*KlineAggregator // key: symbol:interval
	storage     StorageInterface
	mu          sync.RWMutex
	intervals   []string
}

// StorageInterface 存储接口
type StorageInterface interface {
	SaveKline(kline *models.Kline) error
	SaveTicker(ticker *models.Ticker) error
	SaveDepth(depth *models.OrderBook) error
	SaveTrade(trade *models.Trade) error
}

// NewKlineHandler 创建K线处理器
func NewKlineHandler(storage StorageInterface) *KlineHandler {
	return &KlineHandler{
		aggregators: make(map[string]*KlineAggregator),
		storage:     storage,
		intervals: []string{
			constants.Interval1m,
			constants.Interval5m,
			constants.Interval15m,
			constants.Interval1h,
			constants.Interval4h,
			constants.Interval1d,
		},
	}
}

// HandleTrade 处理交易数据生成K线
func (h *KlineHandler) HandleTrade(trade *models.Trade) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 为每个周期生成K线
	for _, interval := range h.intervals {
		key := trade.Symbol + ":" + interval
		aggregator, ok := h.aggregators[key]
		if !ok {
			aggregator = NewKlineAggregator(trade.Symbol, interval, h.storage)
			h.aggregators[key] = aggregator
		}

		// 添加交易数据
		if err := aggregator.AddTrade(trade); err != nil {
			log.Printf("[Kline] Failed to add trade for %s: %v\n", key, err)
		}
	}

	return nil
}

// KlineAggregator K线聚合器
type KlineAggregator struct {
	symbol       string
	interval     string
	currentKline *models.Kline
	storage      StorageInterface
	mu           sync.RWMutex
}

// NewKlineAggregator 创建K线聚合器
func NewKlineAggregator(symbol, interval string, storage StorageInterface) *KlineAggregator {
	return &KlineAggregator{
		symbol:   symbol,
		interval: interval,
		storage:  storage,
	}
}

// AddTrade 添加交易数据
func (a *KlineAggregator) AddTrade(trade *models.Trade) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	openTime := utils.GetKlineOpenTime(trade.Timestamp, a.interval)

	// 检查是否需要生成新K线
	if a.currentKline == nil || a.currentKline.OpenTime != openTime {
		// 保存旧K线
		if a.currentKline != nil {
			if err := a.saveKline(); err != nil {
				log.Printf("[Kline] Failed to save kline: %v\n", err)
			}
		}

		// 创建新K线
		a.currentKline = &models.Kline{
			Symbol:    a.symbol,
			Interval:  a.interval,
			OpenTime:  openTime,
			CloseTime: utils.GetKlineCloseTime(openTime, a.interval),
			Open:      trade.Price,
			High:      trade.Price,
			Low:       trade.Price,
			Close:     trade.Price,
			Volume:    0,
			QuoteVol:  0,
			TradeNum:  0,
		}
	}

	// 更新K线数据
	a.updateKline(trade)

	return nil
}

// updateKline 更新K线数据
func (a *KlineAggregator) updateKline(trade *models.Trade) {
	k := a.currentKline

	// 更新最高价
	if trade.Price > k.High {
		k.High = trade.Price
	}

	// 更新最低价
	if trade.Price < k.Low {
		k.Low = trade.Price
	}

	// 更新收盘价
	k.Close = trade.Price

	// 更新成交量
	k.Volume += trade.Amount
	k.QuoteVol += trade.Price * trade.Amount
	k.TradeNum++
}

// saveKline 保存K线
func (a *KlineAggregator) saveKline() error {
	if a.currentKline == nil {
		return nil
	}

	log.Printf("[Kline] Saving %s %s: O:%.2f H:%.2f L:%.2f C:%.2f V:%.2f\n",
		a.symbol, a.interval,
		a.currentKline.Open, a.currentKline.High,
		a.currentKline.Low, a.currentKline.Close,
		a.currentKline.Volume)

	return a.storage.SaveKline(a.currentKline)
}

// GetCurrentKline 获取当前K线
func (a *KlineAggregator) GetCurrentKline() *models.Kline {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentKline
}
