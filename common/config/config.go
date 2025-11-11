package config

import "market-system/common/models"

// CollectorConfig 采集服务配置
type CollectorConfig struct {
	Server        ServerConfig          `json:"server"`
	Exchanges     []ExchangeConfig      `json:"exchanges"`
	SymbolConfigs []models.SymbolConfig `json:"symbol_configs"` // 交易对混合模式配置
	Kafka         KafkaConfig           `json:"kafka"`
	Log           LogConfig             `json:"log"`
	HybridMode    HybridModeConfig      `json:"hybrid_mode"` // 混合模式配置
}

// ProcessorConfig 处理服务配置
type ProcessorConfig struct {
	Server  ServerConfig    `json:"server"`
	Kafka   KafkaConfig     `json:"kafka"`
	Redis   RedisConfig     `json:"redis"`
	InfluxDB InfluxDBConfig `json:"influxdb"`
	Log     LogConfig       `json:"log"`
}

// APIConfig API服务配置
type APIConfig struct {
	Server   ServerConfig `json:"server"`
	Redis    RedisConfig  `json:"redis"`
	InfluxDB InfluxDBConfig `json:"influxdb"`
	Log      LogConfig    `json:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

// ExchangeConfig 交易所配置
type ExchangeConfig struct {
	Name      string   `json:"name"`
	WSUrl     string   `json:"ws_url"`
	Symbols   []string `json:"symbols"`
	Channels  []string `json:"channels"` // ticker, depth, trade, kline
	Enable    bool     `json:"enable"`
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `json:"brokers"`
	Topics  struct {
		Ticker string `json:"ticker"`
		Depth  string `json:"depth"`
		Trade  string `json:"trade"`
		Kline  string `json:"kline"`
	} `json:"topics"`
	Consumer struct {
		Group string `json:"group"`
	} `json:"consumer"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	PoolSize int    `json:"pool_size"`
}

// InfluxDBConfig InfluxDB配置
type InfluxDBConfig struct {
	URL    string `json:"url"`
	Token  string `json:"token"`
	Org    string `json:"org"`
	Bucket string `json:"bucket"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`       // debug, info, warn, error
	Format     string `json:"format"`      // json, text
	Output     string `json:"output"`      // stdout, file
	FilePath   string `json:"file_path"`
	MaxSize    int    `json:"max_size"`    // MB
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`     // days
}

// HybridModeConfig 混合模式配置
type HybridModeConfig struct {
	Enable                 bool    `json:"enable"`                    // 是否启用混合模式
	InternalPort           int     `json:"internal_port"`             // 内部数据接收端口
	DataFreshnessThreshold int64   `json:"data_freshness_threshold"`  // 数据新鲜度阈值（毫秒）
	PriceDeviationLimit    float64 `json:"price_deviation_limit"`     // 价格偏离限制（百分比）
}
