package models

import "time"

// MarketData 统一的市场数据格式
type MarketData struct {
	Exchange  string      `json:"exchange"`
	Symbol    string      `json:"symbol"`
	Type      string      `json:"type"` // ticker, depth, trade, kline
	Source    string      `json:"source"` // internal, external, merged
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Ticker 行情快照
type Ticker struct {
	Symbol    string  `json:"symbol"`
	LastPrice float64 `json:"last_price"`
	BidPrice  float64 `json:"bid_price"`
	AskPrice  float64 `json:"ask_price"`
	High24h   float64 `json:"high_24h"`
	Low24h    float64 `json:"low_24h"`
	Volume24h float64 `json:"volume_24h"`
	Timestamp int64   `json:"timestamp"`
}

// Trade 成交记录
type Trade struct {
	Symbol    string  `json:"symbol"`
	TradeID   string  `json:"trade_id"`
	Price     float64 `json:"price"`
	Amount    float64 `json:"amount"`
	Side      string  `json:"side"` // buy, sell
	Timestamp int64   `json:"timestamp"`
}

// Kline K线数据
type Kline struct {
	Symbol    string  `json:"symbol"`
	Interval  string  `json:"interval"` // 1m, 5m, 15m, 1h, 4h, 1d
	OpenTime  int64   `json:"open_time"`
	CloseTime int64   `json:"close_time"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	QuoteVol  float64 `json:"quote_vol"` // 成交额
	TradeNum  int64   `json:"trade_num"` // 成交笔数
}

// OrderBook 订单簿
type OrderBook struct {
	Symbol    string      `json:"symbol"`
	Bids      []PriceLevel `json:"bids"` // 买盘，价格从高到低
	Asks      []PriceLevel `json:"asks"` // 卖盘，价格从低到高
	Timestamp int64       `json:"timestamp"`
}

// PriceLevel 价格档位
type PriceLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

// DepthUpdate 深度增量更新
type DepthUpdate struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp int64        `json:"timestamp"`
}

// KafkaMessage Kafka 消息格式
type KafkaMessage struct {
	Topic     string      `json:"topic"`
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Timestamp time.Time   `json:"timestamp"`
}

// ========== 混合模式相关模型 ==========

// SymbolConfig 交易对配置
type SymbolConfig struct {
	Symbol          string `json:"symbol"`
	Mode            string `json:"mode"` // INTERNAL_ONLY, EXTERNAL_ONLY, HYBRID
	PrimarySource   string `json:"primary_source"` // internal, external
	ExternalSource  string `json:"external_source"` // binance, okx, etc.
	MergeStrategy   string `json:"merge_strategy"` // priority, supplement, override
	Enable          bool   `json:"enable"`
	Description     string `json:"description"`
}

// PriceLevelWithSource 带来源的价格档位
type PriceLevelWithSource struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
	Source string  `json:"source"` // internal, external
}

// OrderBookWithSource 带来源的订单簿
type OrderBookWithSource struct {
	Symbol               string                 `json:"symbol"`
	Bids                 []PriceLevelWithSource `json:"bids"`
	Asks                 []PriceLevelWithSource `json:"asks"`
	InternalBidsCount    int                    `json:"internal_bids_count"`
	ExternalBidsCount    int                    `json:"external_bids_count"`
	InternalAsksCount    int                    `json:"internal_asks_count"`
	ExternalAsksCount    int                    `json:"external_asks_count"`
	Timestamp            int64                  `json:"timestamp"`
}

// TickerWithSource 带来源的行情数据
type TickerWithSource struct {
	Symbol             string  `json:"symbol"`
	LastPrice          float64 `json:"last_price"`
	LastPriceSource    string  `json:"last_price_source"` // internal, external
	BidPrice           float64 `json:"bid_price"`
	AskPrice           float64 `json:"ask_price"`
	High24h            float64 `json:"high_24h"`
	Low24h             float64 `json:"low_24h"`
	InternalVolume24h  float64 `json:"internal_volume_24h"`
	ExternalVolume24h  float64 `json:"external_volume_24h"`
	TotalVolume24h     float64 `json:"total_volume_24h"`
	Timestamp          int64   `json:"timestamp"`
}

// InternalTradeMessage 内部交易引擎推送的消息
type InternalTradeMessage struct {
	Symbol      string  `json:"symbol"`
	TradeID     int64   `json:"trade_id"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
	Side        string  `json:"side"` // buy, sell
	BuyerID     int64   `json:"buyer_id"`
	SellerID    int64   `json:"seller_id"`
	BuyOrderID  int64   `json:"buy_order_id"`
	SellerOrderID int64 `json:"sell_order_id"`
	Timestamp   int64   `json:"timestamp"`
	IsMaker     bool    `json:"is_maker"`
}

// InternalDepthMessage 内部深度消息
type InternalDepthMessage struct {
	Symbol    string       `json:"symbol"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp int64        `json:"timestamp"`
	SeqNum    int64        `json:"seq_num"` // 序列号，用于增量更新
}

// DataSourceStats 数据源统计
type DataSourceStats struct {
	Symbol            string  `json:"symbol"`
	InternalDataRate  float64 `json:"internal_data_rate"` // 内部数据占比
	ExternalDataRate  float64 `json:"external_data_rate"` // 外部数据占比
	InternalTradeNum  int64   `json:"internal_trade_num"` // 内部成交笔数
	ExternalTradeNum  int64   `json:"external_trade_num"` // 外部成交笔数
	LastUpdateTime    int64   `json:"last_update_time"`
}
