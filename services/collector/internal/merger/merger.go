package merger

import (
	"log"
	"market-system/common/constants"
	"market-system/common/models"
	"sort"
	"sync"
	"time"
)

// DataMerger 数据融合器
type DataMerger struct {
	symbolConfigs map[string]*models.SymbolConfig // 交易对配置
	internalData  map[string]*CachedData           // 内部数据缓存
	externalData  map[string]*CachedData           // 外部数据缓存
	mu            sync.RWMutex
}

// CachedData 缓存的数据
type CachedData struct {
	Ticker    *models.Ticker
	Depth     *models.OrderBook
	Trades    []*models.Trade
	Timestamp int64
}

// NewDataMerger 创建数据融合器
func NewDataMerger(configs []*models.SymbolConfig) *DataMerger {
	merger := &DataMerger{
		symbolConfigs: make(map[string]*models.SymbolConfig),
		internalData:  make(map[string]*CachedData),
		externalData:  make(map[string]*CachedData),
	}

	// 加载配置
	for _, config := range configs {
		merger.symbolConfigs[config.Symbol] = config
	}

	log.Printf("[Merger] Initialized with %d symbol configs\n", len(configs))
	return merger
}

// ProcessData 处理数据（判断是否需要融合）
func (m *DataMerger) ProcessData(data *models.MarketData) *models.MarketData {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取交易对配置
	config, ok := m.symbolConfigs[data.Symbol]
	if !ok {
		// 如果没有配置，直接返回原数据
		return data
	}

	// 根据模式处理
	switch config.Mode {
	case constants.ModeInternalOnly:
		// 仅内部数据
		if data.Source == constants.SourceInternal {
			return data
		}
		return nil // 丢弃外部数据

	case constants.ModeExternalOnly:
		// 仅外部数据
		if data.Source == constants.SourceExternal {
			return data
		}
		return nil // 丢弃内部数据

	case constants.ModeHybrid:
		// 混合模式，缓存数据并融合
		return m.mergeData(data, config)

	default:
		return data
	}
}

// mergeData 融合数据
func (m *DataMerger) mergeData(data *models.MarketData, config *models.SymbolConfig) *models.MarketData {
	// 缓存数据
	m.cacheData(data)

	// 根据数据类型和策略融合
	switch data.Type {
	case constants.DataTypeTicker:
		return m.mergeTicker(data.Symbol, config)
	case constants.DataTypeDepth:
		return m.mergeDepth(data.Symbol, config)
	case constants.DataTypeTrade:
		// Trade 数据不融合，直接返回
		return data
	default:
		return data
	}
}

// cacheData 缓存数据
func (m *DataMerger) cacheData(data *models.MarketData) {
	symbol := data.Symbol
	isInternal := data.Source == constants.SourceInternal

	// 获取或创建缓存
	var cache *CachedData
	if isInternal {
		if m.internalData[symbol] == nil {
			m.internalData[symbol] = &CachedData{}
		}
		cache = m.internalData[symbol]
	} else {
		if m.externalData[symbol] == nil {
			m.externalData[symbol] = &CachedData{}
		}
		cache = m.externalData[symbol]
	}

	// 缓存数据
	switch data.Type {
	case constants.DataTypeTicker:
		if ticker, ok := data.Data.(*models.Ticker); ok {
			cache.Ticker = ticker
		}
	case constants.DataTypeDepth:
		if depth, ok := data.Data.(*models.OrderBook); ok {
			cache.Depth = depth
		}
	case constants.DataTypeTrade:
		if trade, ok := data.Data.(*models.Trade); ok {
			cache.Trades = append([]*models.Trade{trade}, cache.Trades...)
			if len(cache.Trades) > 100 {
				cache.Trades = cache.Trades[:100] // 只保留最近100条
			}
		}
	}

	cache.Timestamp = time.Now().UnixMilli()
}

// mergeTicker 融合 Ticker 数据
func (m *DataMerger) mergeTicker(symbol string, config *models.SymbolConfig) *models.MarketData {
	internalCache := m.internalData[symbol]
	externalCache := m.externalData[symbol]

	var mergedTicker *models.TickerWithSource

	switch config.MergeStrategy {
	case constants.MergeStrategyPriority:
		// 优先级策略：内部数据优先
		mergedTicker = m.mergeTickerPriority(internalCache, externalCache, symbol)

	case constants.MergeStrategySupplement:
		// 补充策略：内部数据为主，外部数据补充
		mergedTicker = m.mergeTickerSupplement(internalCache, externalCache, symbol)

	default:
		// 默认使用优先级策略
		mergedTicker = m.mergeTickerPriority(internalCache, externalCache, symbol)
	}

	if mergedTicker == nil {
		return nil
	}

	return &models.MarketData{
		Exchange:  "merged",
		Symbol:    symbol,
		Type:      constants.DataTypeTicker,
		Source:    constants.SourceMerged,
		Timestamp: time.Now().UnixMilli(),
		Data:      mergedTicker,
	}
}

// mergeTickerPriority 优先级融合 Ticker
func (m *DataMerger) mergeTickerPriority(internal, external *CachedData, symbol string) *models.TickerWithSource {
	ticker := &models.TickerWithSource{
		Symbol: symbol,
	}

	// 优先使用内部数据
	if internal != nil && internal.Ticker != nil && m.isDataFresh(internal.Timestamp) {
		ticker.LastPrice = internal.Ticker.LastPrice
		ticker.LastPriceSource = constants.SourceInternal
		ticker.BidPrice = internal.Ticker.BidPrice
		ticker.AskPrice = internal.Ticker.AskPrice
		ticker.High24h = internal.Ticker.High24h
		ticker.Low24h = internal.Ticker.Low24h
		ticker.InternalVolume24h = internal.Ticker.Volume24h
		ticker.Timestamp = internal.Ticker.Timestamp
	} else if external != nil && external.Ticker != nil && m.isDataFresh(external.Timestamp) {
		// 没有内部数据或内部数据过期，使用外部数据
		ticker.LastPrice = external.Ticker.LastPrice
		ticker.LastPriceSource = constants.SourceExternal
		ticker.BidPrice = external.Ticker.BidPrice
		ticker.AskPrice = external.Ticker.AskPrice
		ticker.High24h = external.Ticker.High24h
		ticker.Low24h = external.Ticker.Low24h
		ticker.ExternalVolume24h = external.Ticker.Volume24h
		ticker.Timestamp = external.Ticker.Timestamp
	} else {
		return nil
	}

	// 合并成交量
	if internal != nil && internal.Ticker != nil {
		ticker.InternalVolume24h = internal.Ticker.Volume24h
	}
	if external != nil && external.Ticker != nil {
		ticker.ExternalVolume24h = external.Ticker.Volume24h
	}
	ticker.TotalVolume24h = ticker.InternalVolume24h + ticker.ExternalVolume24h

	return ticker
}

// mergeTickerSupplement 补充融合 Ticker
func (m *DataMerger) mergeTickerSupplement(internal, external *CachedData, symbol string) *models.TickerWithSource {
	// 与优先级策略类似，但更强调外部数据作为补充
	return m.mergeTickerPriority(internal, external, symbol)
}

// mergeDepth 融合深度数据
func (m *DataMerger) mergeDepth(symbol string, config *models.SymbolConfig) *models.MarketData {
	internalCache := m.internalData[symbol]
	externalCache := m.externalData[symbol]

	var mergedDepth *models.OrderBookWithSource

	switch config.MergeStrategy {
	case constants.MergeStrategyPriority:
		mergedDepth = m.mergeDepthPriority(internalCache, externalCache, symbol)

	case constants.MergeStrategySupplement:
		mergedDepth = m.mergeDepthSupplement(internalCache, externalCache, symbol)

	default:
		mergedDepth = m.mergeDepthPriority(internalCache, externalCache, symbol)
	}

	if mergedDepth == nil {
		return nil
	}

	return &models.MarketData{
		Exchange:  "merged",
		Symbol:    symbol,
		Type:      constants.DataTypeDepth,
		Source:    constants.SourceMerged,
		Timestamp: time.Now().UnixMilli(),
		Data:      mergedDepth,
	}
}

// mergeDepthPriority 优先级融合深度
func (m *DataMerger) mergeDepthPriority(internal, external *CachedData, symbol string) *models.OrderBookWithSource {
	depth := &models.OrderBookWithSource{
		Symbol: symbol,
		Bids:   make([]models.PriceLevelWithSource, 0),
		Asks:   make([]models.PriceLevelWithSource, 0),
	}

	// 添加内部深度
	if internal != nil && internal.Depth != nil && m.isDataFresh(internal.Timestamp) {
		for _, bid := range internal.Depth.Bids {
			depth.Bids = append(depth.Bids, models.PriceLevelWithSource{
				Price:  bid.Price,
				Amount: bid.Amount,
				Source: constants.SourceInternal,
			})
		}
		for _, ask := range internal.Depth.Asks {
			depth.Asks = append(depth.Asks, models.PriceLevelWithSource{
				Price:  ask.Price,
				Amount: ask.Amount,
				Source: constants.SourceInternal,
			})
		}
		depth.InternalBidsCount = len(internal.Depth.Bids)
		depth.InternalAsksCount = len(internal.Depth.Asks)
	}

	// 添加外部深度
	if external != nil && external.Depth != nil && m.isDataFresh(external.Timestamp) {
		for _, bid := range external.Depth.Bids {
			depth.Bids = append(depth.Bids, models.PriceLevelWithSource{
				Price:  bid.Price,
				Amount: bid.Amount,
				Source: constants.SourceExternal,
			})
		}
		for _, ask := range external.Depth.Asks {
			depth.Asks = append(depth.Asks, models.PriceLevelWithSource{
				Price:  ask.Price,
				Amount: ask.Amount,
				Source: constants.SourceExternal,
			})
		}
		depth.ExternalBidsCount = len(external.Depth.Bids)
		depth.ExternalAsksCount = len(external.Depth.Asks)
	}

	// 排序合并
	sort.Slice(depth.Bids, func(i, j int) bool {
		return depth.Bids[i].Price > depth.Bids[j].Price // 买盘价格从高到低
	})
	sort.Slice(depth.Asks, func(i, j int) bool {
		return depth.Asks[i].Price < depth.Asks[j].Price // 卖盘价格从低到高
	})

	// 限制档位数量
	if len(depth.Bids) > 100 {
		depth.Bids = depth.Bids[:100]
	}
	if len(depth.Asks) > 100 {
		depth.Asks = depth.Asks[:100]
	}

	depth.Timestamp = time.Now().UnixMilli()
	return depth
}

// mergeDepthSupplement 补充融合深度
func (m *DataMerger) mergeDepthSupplement(internal, external *CachedData, symbol string) *models.OrderBookWithSource {
	// 如果内部深度档位不足，用外部深度补充
	depth := &models.OrderBookWithSource{
		Symbol: symbol,
		Bids:   make([]models.PriceLevelWithSource, 0),
		Asks:   make([]models.PriceLevelWithSource, 0),
	}

	// 优先添加内部深度
	if internal != nil && internal.Depth != nil && m.isDataFresh(internal.Timestamp) {
		for _, bid := range internal.Depth.Bids {
			depth.Bids = append(depth.Bids, models.PriceLevelWithSource{
				Price:  bid.Price,
				Amount: bid.Amount,
				Source: constants.SourceInternal,
			})
		}
		for _, ask := range internal.Depth.Asks {
			depth.Asks = append(depth.Asks, models.PriceLevelWithSource{
				Price:  ask.Price,
				Amount: ask.Amount,
				Source: constants.SourceInternal,
			})
		}
		depth.InternalBidsCount = len(internal.Depth.Bids)
		depth.InternalAsksCount = len(internal.Depth.Asks)
	}

	// 如果档位不足 20 档，用外部数据补充
	if len(depth.Bids) < 20 && external != nil && external.Depth != nil && m.isDataFresh(external.Timestamp) {
		for _, bid := range external.Depth.Bids {
			depth.Bids = append(depth.Bids, models.PriceLevelWithSource{
				Price:  bid.Price,
				Amount: bid.Amount,
				Source: constants.SourceExternal,
			})
		}
		depth.ExternalBidsCount = len(external.Depth.Bids)
	}

	if len(depth.Asks) < 20 && external != nil && external.Depth != nil && m.isDataFresh(external.Timestamp) {
		for _, ask := range external.Depth.Asks {
			depth.Asks = append(depth.Asks, models.PriceLevelWithSource{
				Price:  ask.Price,
				Amount: ask.Amount,
				Source: constants.SourceExternal,
			})
		}
		depth.ExternalAsksCount = len(external.Depth.Asks)
	}

	// 排序
	sort.Slice(depth.Bids, func(i, j int) bool {
		return depth.Bids[i].Price > depth.Bids[j].Price
	})
	sort.Slice(depth.Asks, func(i, j int) bool {
		return depth.Asks[i].Price < depth.Asks[j].Price
	})

	depth.Timestamp = time.Now().UnixMilli()
	return depth
}

// isDataFresh 检查数据是否新鲜
func (m *DataMerger) isDataFresh(timestamp int64) bool {
	now := time.Now().UnixMilli()
	return (now - timestamp) < constants.DataFreshnessThreshold
}

// GetSymbolConfig 获取交易对配置
func (m *DataMerger) GetSymbolConfig(symbol string) *models.SymbolConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.symbolConfigs[symbol]
}

// UpdateSymbolConfig 更新交易对配置
func (m *DataMerger) UpdateSymbolConfig(config *models.SymbolConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.symbolConfigs[config.Symbol] = config
	log.Printf("[Merger] Updated config for symbol: %s, mode: %s\n", config.Symbol, config.Mode)
}
