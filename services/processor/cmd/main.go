package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"market-system/common/config"
	"market-system/common/constants"
	"market-system/common/models"
	"market-system/services/processor/internal/consumer"
	"market-system/services/processor/internal/handler"
	"market-system/services/processor/internal/storage"
	"os"
	"os/signal"
	"syscall"
)

var (
	configPath = flag.String("config", "configs/processor.json", "配置文件路径")
)

type Processor struct {
	config        *config.ProcessorConfig
	consumer      *consumer.KafkaConsumer
	storage       *storage.RedisStorage
	klineHandler  *handler.KlineHandler
	depthHandler  *handler.DepthHandler
	ctx           context.Context
	cancel        context.CancelFunc
}

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// 创建 Processor
	processor, err := NewProcessor(cfg)
	if err != nil {
		log.Fatalf("Failed to create processor: %v\n", err)
	}

	// 启动服务
	if err := processor.Start(); err != nil {
		log.Fatalf("Failed to start processor: %v\n", err)
	}

	// 等待退出信号
	waitForSignal()

	// 停止服务
	processor.Stop()
}

func NewProcessor(cfg *config.ProcessorConfig) (*Processor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化 Redis 存储
	redisStorage, err := storage.NewRedisStorage(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		cancel()
		return nil, err
	}

	// 初始化处理器
	klineHandler := handler.NewKlineHandler(redisStorage)
	depthHandler := handler.NewDepthHandler(redisStorage)

	// 初始化 Kafka 消费者
	kafkaConsumer := consumer.NewKafkaConsumer(cfg.Kafka.Brokers, cfg.Kafka.Consumer.Group)

	return &Processor{
		config:       cfg,
		consumer:     kafkaConsumer,
		storage:      redisStorage,
		klineHandler: klineHandler,
		depthHandler: depthHandler,
		ctx:          ctx,
		cancel:       cancel,
	}, nil
}

func (p *Processor) Start() error {
	log.Println("Starting Market Data Processor...")

	// 订阅 Ticker Topic
	p.consumer.Subscribe(constants.TopicMarketTicker, func(data *models.MarketData) error {
		ticker, ok := data.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		// 转换为 Ticker 对象
		t := parseTickerFromMap(ticker, data.Symbol)
		return p.storage.SaveTicker(t)
	})

	// 订阅 Depth Topic
	p.consumer.Subscribe(constants.TopicMarketDepth, func(data *models.MarketData) error {
		depthMap, ok := data.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		depth := parseDepthFromMap(depthMap, data.Symbol, data.Timestamp)
		return p.depthHandler.HandleDepth(depth)
	})

	// 订阅 Trade Topic
	p.consumer.Subscribe(constants.TopicMarketTrade, func(data *models.MarketData) error {
		tradeMap, ok := data.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		trade := parseTradeFromMap(tradeMap, data.Symbol)

		// 保存交易数据
		if err := p.storage.SaveTrade(trade); err != nil {
			log.Printf("[Trade] Failed to save: %v\n", err)
		}

		// 生成K线
		return p.klineHandler.HandleTrade(trade)
	})

	// 启动消费
	if err := p.consumer.Start(p.ctx); err != nil {
		return err
	}

	log.Println("Processor started successfully!")
	return nil
}

func (p *Processor) Stop() {
	log.Println("Stopping processor...")

	// 取消上下文
	p.cancel()

	// 关闭消费者
	if p.consumer != nil {
		p.consumer.Close()
	}

	// 关闭存储
	if p.storage != nil {
		p.storage.Close()
	}

	log.Println("Processor stopped")
}

// parseTickerFromMap 从 map 解析 Ticker
func parseTickerFromMap(data map[string]interface{}, symbol string) *models.Ticker {
	return &models.Ticker{
		Symbol:    symbol,
		LastPrice: getFloat(data, "last_price"),
		BidPrice:  getFloat(data, "bid_price"),
		AskPrice:  getFloat(data, "ask_price"),
		High24h:   getFloat(data, "high_24h"),
		Low24h:    getFloat(data, "low_24h"),
		Volume24h: getFloat(data, "volume_24h"),
		Timestamp: getInt64(data, "timestamp"),
	}
}

// parseDepthFromMap 从 map 解析深度
func parseDepthFromMap(data map[string]interface{}, symbol string, timestamp int64) *models.OrderBook {
	return &models.OrderBook{
		Symbol:    symbol,
		Bids:      parsePriceLevels(data["bids"]),
		Asks:      parsePriceLevels(data["asks"]),
		Timestamp: timestamp,
	}
}

// parseTradeFromMap 从 map 解析交易
func parseTradeFromMap(data map[string]interface{}, symbol string) *models.Trade {
	return &models.Trade{
		Symbol:    symbol,
		TradeID:   getString(data, "trade_id"),
		Price:     getFloat(data, "price"),
		Amount:    getFloat(data, "amount"),
		Side:      getString(data, "side"),
		Timestamp: getInt64(data, "timestamp"),
	}
}

// 辅助函数
func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok {
		if i, ok := v.(float64); ok {
			return int64(i)
		}
	}
	return 0
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
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
		levelMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		levels = append(levels, models.PriceLevel{
			Price:  getFloat(levelMap, "price"),
			Amount: getFloat(levelMap, "amount"),
		})
	}

	return levels
}

// loadConfig 加载配置文件
func loadConfig(path string) (*config.ProcessorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg config.ProcessorConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// waitForSignal 等待退出信号
func waitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Received shutdown signal")
}
