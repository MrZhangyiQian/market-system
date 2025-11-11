package handler

import (
	"log"
	"market-system/common/models"
	"sort"
	"sync"
)

// DepthHandler 深度数据处理器
type DepthHandler struct {
	managers map[string]*DepthManager // key: symbol
	storage  StorageInterface
	mu       sync.RWMutex
}

// NewDepthHandler 创建深度处理器
func NewDepthHandler(storage StorageInterface) *DepthHandler {
	return &DepthHandler{
		managers: make(map[string]*DepthManager),
		storage:  storage,
	}
}

// HandleDepth 处理深度数据
func (h *DepthHandler) HandleDepth(depth *models.OrderBook) error {
	h.mu.Lock()
	manager, ok := h.managers[depth.Symbol]
	if !ok {
		manager = NewDepthManager(depth.Symbol, h.storage)
		h.managers[depth.Symbol] = manager
	}
	h.mu.Unlock()

	return manager.UpdateDepth(depth)
}

// DepthManager 深度管理器
type DepthManager struct {
	symbol  string
	bids    []models.PriceLevel // 买盘，价格从高到低
	asks    []models.PriceLevel // 卖盘，价格从低到高
	storage StorageInterface
	mu      sync.RWMutex
}

// NewDepthManager 创建深度管理器
func NewDepthManager(symbol string, storage StorageInterface) *DepthManager {
	return &DepthManager{
		symbol:  symbol,
		bids:    make([]models.PriceLevel, 0),
		asks:    make([]models.PriceLevel, 0),
		storage: storage,
	}
}

// UpdateDepth 更新深度
func (m *DepthManager) UpdateDepth(depth *models.OrderBook) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 全量更新
	m.bids = depth.Bids
	m.asks = depth.Asks

	// 排序
	m.sortDepth()

	// 保存到存储
	orderBook := &models.OrderBook{
		Symbol:    m.symbol,
		Bids:      m.bids,
		Asks:      m.asks,
		Timestamp: depth.Timestamp,
	}

	if err := m.storage.SaveDepth(orderBook); err != nil {
		log.Printf("[Depth] Failed to save depth for %s: %v\n", m.symbol, err)
		return err
	}

	return nil
}

// sortDepth 排序深度
func (m *DepthManager) sortDepth() {
	// 买盘按价格从高到低排序
	sort.Slice(m.bids, func(i, j int) bool {
		return m.bids[i].Price > m.bids[j].Price
	})

	// 卖盘按价格从低到高排序
	sort.Slice(m.asks, func(i, j int) bool {
		return m.asks[i].Price < m.asks[j].Price
	})
}

// GetBestBid 获取最优买价
func (m *DepthManager) GetBestBid() *models.PriceLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.bids) > 0 {
		return &m.bids[0]
	}
	return nil
}

// GetBestAsk 获取最优卖价
func (m *DepthManager) GetBestAsk() *models.PriceLevel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.asks) > 0 {
		return &m.asks[0]
	}
	return nil
}

// GetSpread 获取买卖价差
func (m *DepthManager) GetSpread() float64 {
	bestBid := m.GetBestBid()
	bestAsk := m.GetBestAsk()

	if bestBid == nil || bestAsk == nil {
		return 0
	}

	return bestAsk.Price - bestBid.Price
}
