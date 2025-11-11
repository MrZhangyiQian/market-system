package constants

// 数据类型常量
const (
	DataTypeTicker = "ticker"
	DataTypeDepth  = "depth"
	DataTypeTrade  = "trade"
	DataTypeKline  = "kline"
)

// 交易所常量
const (
	ExchangeBinance = "binance"
	ExchangeOKX     = "okx"
	ExchangeBybit   = "bybit"
	ExchangeGate    = "gate"
)

// K线周期常量
const (
	Interval1m  = "1m"
	Interval5m  = "5m"
	Interval15m = "15m"
	Interval30m = "30m"
	Interval1h  = "1h"
	Interval4h  = "4h"
	Interval1d  = "1d"
	Interval1w  = "1w"
)

// 交易方向
const (
	SideBuy  = "buy"
	SideSell = "sell"
)

// WebSocket 状态
const (
	WSStateConnecting = "connecting"
	WSStateConnected  = "connected"
	WSStateClosing    = "closing"
	WSStateClosed     = "closed"
)

// Kafka Topic 常量
const (
	TopicMarketTicker = "market.ticker"
	TopicMarketDepth  = "market.depth"
	TopicMarketTrade  = "market.trade"
	TopicMarketKline  = "market.kline"
)

// Redis Key 前缀
const (
	RedisKeyTicker     = "ticker:"     // ticker:{symbol}
	RedisKeyDepth      = "depth:"      // depth:{symbol}
	RedisKeyKline      = "kline:"      // kline:{symbol}:{interval}
	RedisKeyTrade      = "trade:"      // trade:{symbol}
	RedisChannelMarket = "market:"     // market:{symbol}:{type}
)

// 时间常量（毫秒）
const (
	Second = 1000
	Minute = 60 * Second
	Hour   = 60 * Minute
	Day    = 24 * Hour
)

// 重连配置
const (
	MaxReconnectRetry  = 10          // 最大重连次数
	InitReconnectDelay = 1 * Second  // 初始重连延迟
	MaxReconnectDelay  = 60 * Second // 最大重连延迟
)

// WebSocket 配置
const (
	PingInterval     = 30 * Second // 心跳间隔
	PongTimeout      = 60 * Second // 心跳超时
	WriteWait        = 10 * Second // 写超时
	ReadBufferSize   = 4096        // 读缓冲区大小
	WriteBufferSize  = 4096        // 写缓冲区大小
	MaxMessageSize   = 512 * 1024  // 最大消息大小 512KB
)

// 限流配置
const (
	MaxSubscriptionsPerConn = 20  // 每个连接最多订阅数
	MessageRateLimit        = 100 // 每秒最大消息数
)

// 深度档位
const (
	DefaultDepthLevel = 20  // 默认深度档位
	MaxDepthLevel     = 100 // 最大深度档位
)

// ========== 混合模式常量 ==========

// 数据源类型
const (
	SourceInternal = "internal" // 内部数据源
	SourceExternal = "external" // 外部数据源
	SourceMerged   = "merged"   // 合并数据
)

// 交易对模式
const (
	ModeInternalOnly = "INTERNAL_ONLY" // 仅内部数据
	ModeExternalOnly = "EXTERNAL_ONLY" // 仅外部数据
	ModeHybrid       = "HYBRID"        // 混合模式
)

// 数据融合策略
const (
	MergeStrategyPriority   = "priority"   // 优先级策略（内部优先）
	MergeStrategySupplement = "supplement" // 补充策略（外部补充）
	MergeStrategyOverride   = "override"   // 覆盖策略（外部覆盖）
)

// 内部数据源标识
const (
	ExchangeInternal = "internal" // 内部交易引擎
)

// 数据新鲜度（毫秒）
const (
	DataFreshnessThreshold = 5 * Second // 数据新鲜度阈值：5秒
	PriceDeviationLimit    = 10.0       // 价格偏离限制：10%
)
