package publisher

import (
	"context"
	"fmt"
	"log"
	"market-system/common/constants"
	"market-system/common/models"
	"market-system/common/utils"

	"github.com/segmentio/kafka-go"
)

// KafkaPublisher Kafka 发布者
type KafkaPublisher struct {
	writers map[string]*kafka.Writer
	brokers []string
}

// NewKafkaPublisher 创建 Kafka 发布者
func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{
		writers: make(map[string]*kafka.Writer),
		brokers: brokers,
	}
}

// Init 初始化 Writer
func (p *KafkaPublisher) Init() error {
	topics := []string{
		constants.TopicMarketTicker,
		constants.TopicMarketDepth,
		constants.TopicMarketTrade,
		constants.TopicMarketKline,
	}

	for _, topic := range topics {
		writer := &kafka.Writer{
			Addr:         kafka.TCP(p.brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchSize:    100,
			BatchTimeout: 10, // 10ms
			Async:        true,
			RequiredAcks: kafka.RequireOne,
		}
		p.writers[topic] = writer
		log.Printf("[Kafka] Initialized writer for topic: %s\n", topic)
	}

	return nil
}

// Publish 发布消息
func (p *KafkaPublisher) Publish(data *models.MarketData) error {
	topic := p.getTopicByType(data.Type)
	if topic == "" {
		return fmt.Errorf("unknown data type: %s", data.Type)
	}

	writer, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("writer not found for topic: %s", topic)
	}

	// 转换为 JSON
	value, err := utils.ToJSONBytes(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// 使用 symbol 作为 key，保证同一 symbol 的消息发送到同一分区
	key := []byte(data.Symbol)

	msg := kafka.Message{
		Key:   key,
		Value: value,
	}

	// 异步写入
	err = writer.WriteMessages(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Close 关闭所有 Writer
func (p *KafkaPublisher) Close() error {
	for topic, writer := range p.writers {
		if err := writer.Close(); err != nil {
			log.Printf("[Kafka] Failed to close writer for topic %s: %v\n", topic, err)
		} else {
			log.Printf("[Kafka] Closed writer for topic: %s\n", topic)
		}
	}
	return nil
}

// getTopicByType 根据数据类型获取 Topic
func (p *KafkaPublisher) getTopicByType(dataType string) string {
	switch dataType {
	case constants.DataTypeTicker:
		return constants.TopicMarketTicker
	case constants.DataTypeDepth:
		return constants.TopicMarketDepth
	case constants.DataTypeTrade:
		return constants.TopicMarketTrade
	case constants.DataTypeKline:
		return constants.TopicMarketKline
	default:
		return ""
	}
}

// GetStats 获取统计信息
func (p *KafkaPublisher) GetStats() map[string]kafka.WriterStats {
	stats := make(map[string]kafka.WriterStats)
	for topic, writer := range p.writers {
		stats[topic] = writer.Stats()
	}
	return stats
}
