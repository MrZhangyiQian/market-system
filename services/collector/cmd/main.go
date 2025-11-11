package main

import (
	"encoding/json"
	"flag"
	"log"
	"market-system/common/config"
	"market-system/common/models"
	"market-system/services/collector/internal/adapters"
	"market-system/services/collector/internal/publisher"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	configPath = flag.String("config", "configs/collector.json", "配置文件路径")
)

type Collector struct {
	config    *config.CollectorConfig
	factory   *adapters.AdapterFactory
	adapters  []adapters.ExchangeAdapter
	publisher *publisher.KafkaPublisher
	wg        sync.WaitGroup
}

func main() {
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// 创建 Collector
	collector := NewCollector(cfg)

	// 启动服务
	if err := collector.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v\n", err)
	}

	// 等待退出信号
	waitForSignal()

	// 停止服务
	collector.Stop()
}

func NewCollector(cfg *config.CollectorConfig) *Collector {
	return &Collector{
		config:   cfg,
		factory:  adapters.NewAdapterFactory(),
		adapters: make([]adapters.ExchangeAdapter, 0),
	}
}

func (c *Collector) Start() error {
	log.Println("Starting Market Data Collector...")

	// 初始化 Kafka Publisher
	c.publisher = publisher.NewKafkaPublisher(c.config.Kafka.Brokers)
	if err := c.publisher.Init(); err != nil {
		return err
	}

	// 初始化交易所适配器
	for _, exchangeCfg := range c.config.Exchanges {
		if !exchangeCfg.Enable {
			log.Printf("[%s] Disabled, skipping...\n", exchangeCfg.Name)
			continue
		}

		adapter := c.factory.Create(exchangeCfg.Name, exchangeCfg.WSUrl)
		if adapter == nil {
			log.Printf("[%s] Adapter not found, skipping...\n", exchangeCfg.Name)
			continue
		}

		// 设置消息处理器
		adapter.OnMessage(c.handleMarketData)

		// 连接
		if err := adapter.Connect(); err != nil {
			log.Printf("[%s] Failed to connect: %v\n", exchangeCfg.Name, err)
			continue
		}

		// 订阅
		if err := adapter.Subscribe(exchangeCfg.Symbols, exchangeCfg.Channels); err != nil {
			log.Printf("[%s] Failed to subscribe: %v\n", exchangeCfg.Name, err)
			continue
		}

		c.adapters = append(c.adapters, adapter)
		log.Printf("[%s] Started successfully\n", exchangeCfg.Name)
	}

	// 启动统计输出
	go c.printStats()

	log.Println("Collector started successfully!")
	return nil
}

func (c *Collector) Stop() {
	log.Println("Stopping collector...")

	// 关闭所有适配器
	for _, adapter := range c.adapters {
		if err := adapter.Close(); err != nil {
			log.Printf("[%s] Failed to close: %v\n", adapter.GetName(), err)
		}
	}

	// 关闭 Kafka Publisher
	if c.publisher != nil {
		c.publisher.Close()
	}

	c.wg.Wait()
	log.Println("Collector stopped")
}

// handleMarketData 处理市场数据
func (c *Collector) handleMarketData(data *models.MarketData) {
	// 发布到 Kafka
	if err := c.publisher.Publish(data); err != nil {
		log.Printf("[ERROR] Failed to publish data: %v\n", err)
		return
	}

	// 日志输出（可选）
	if c.config.Log.Level == "debug" {
		log.Printf("[%s] %s %s: received\n", data.Exchange, data.Symbol, data.Type)
	}
}

// printStats 定期打印统计信息
func (c *Collector) printStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := c.publisher.GetStats()
		log.Println("=== Kafka Stats ===")
		for topic, stat := range stats {
			log.Printf("[%s] Messages: %d, Bytes: %d, Errors: %d\n",
				topic, stat.Messages, stat.Bytes, stat.Errors)
		}
	}
}

// loadConfig 加载配置文件
func loadConfig(path string) (*config.CollectorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg config.CollectorConfig
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
