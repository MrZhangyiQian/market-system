package consumer

import (
	"context"
	"fmt"
	"log"
	"market-system/common/models"
	"market-system/common/utils"

	"github.com/segmentio/kafka-go"
)

// MessageHandler 消息处理器
type MessageHandler func(data *models.MarketData) error

// KafkaConsumer Kafka 消费者
type KafkaConsumer struct {
	readers  map[string]*kafka.Reader
	handlers map[string]MessageHandler
	brokers  []string
	groupID  string
}

// NewKafkaConsumer 创建 Kafka 消费者
func NewKafkaConsumer(brokers []string, groupID string) *KafkaConsumer {
	return &KafkaConsumer{
		readers:  make(map[string]*kafka.Reader),
		handlers: make(map[string]MessageHandler),
		brokers:  brokers,
		groupID:  groupID,
	}
}

// Subscribe 订阅 Topic
func (c *KafkaConsumer) Subscribe(topic string, handler MessageHandler) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		Topic:          topic,
		GroupID:        c.groupID,
		MinBytes:       1e3,  // 1KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1000, // 1s
		StartOffset:    kafka.LastOffset,
	})

	c.readers[topic] = reader
	c.handlers[topic] = handler

	log.Printf("[Kafka Consumer] Subscribed to topic: %s\n", topic)
	return nil
}

// Start 启动消费
func (c *KafkaConsumer) Start(ctx context.Context) error {
	for topic, reader := range c.readers {
		handler := c.handlers[topic]
		go c.consume(ctx, topic, reader, handler)
	}
	return nil
}

// consume 消费消息
func (c *KafkaConsumer) consume(ctx context.Context, topic string, reader *kafka.Reader, handler MessageHandler) {
	log.Printf("[Kafka Consumer] Started consuming topic: %s\n", topic)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[Kafka Consumer] Stopping consumer for topic: %s\n", topic)
			return
		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("[Kafka Consumer] Error fetching message: %v\n", err)
				continue
			}

			// 解析消息
			var data models.MarketData
			if err := utils.FromJSONBytes(msg.Value, &data); err != nil {
				log.Printf("[Kafka Consumer] Failed to parse message: %v\n", err)
				reader.CommitMessages(ctx, msg)
				continue
			}

			// 处理消息
			if err := handler(&data); err != nil {
				log.Printf("[Kafka Consumer] Failed to handle message: %v\n", err)
			}

			// 提交消息
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("[Kafka Consumer] Failed to commit message: %v\n", err)
			}
		}
	}
}

// Close 关闭所有 Reader
func (c *KafkaConsumer) Close() error {
	for topic, reader := range c.readers {
		if err := reader.Close(); err != nil {
			log.Printf("[Kafka Consumer] Failed to close reader for topic %s: %v\n", topic, err)
		} else {
			log.Printf("[Kafka Consumer] Closed reader for topic: %s\n", topic)
		}
	}
	return nil
}

// GetStats 获取统计信息
func (c *KafkaConsumer) GetStats() map[string]kafka.ReaderStats {
	stats := make(map[string]kafka.ReaderStats)
	for topic, reader := range c.readers {
		stats[topic] = reader.Stats()
	}
	return stats
}

// Lag 获取消费延迟
func (c *KafkaConsumer) Lag(topic string) (int64, error) {
	reader, ok := c.readers[topic]
	if !ok {
		return 0, fmt.Errorf("reader not found for topic: %s", topic)
	}

	stats := reader.Stats()
	return stats.Lag, nil
}
