package storage

import (
	"context"
	"fmt"
	"log"
	"market-system/common/constants"
	"market-system/common/models"
	"market-system/common/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage Redis 存储
type RedisStorage struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisStorage 创建 Redis 存储
func NewRedisStorage(host string, port int, password string, db int) (*RedisStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
		MinIdleConns: 10,
	})

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("[Redis] Connected successfully")

	return &RedisStorage{
		client: client,
		ctx:    ctx,
	}, nil
}

// SaveKline 保存K线数据
func (s *RedisStorage) SaveKline(kline *models.Kline) error {
	key := fmt.Sprintf("%s%s:%s", constants.RedisKeyKline, kline.Symbol, kline.Interval)

	// 将K线转换为JSON
	data, err := utils.ToJSON(kline)
	if err != nil {
		return err
	}

	// 使用 List 存储，保留最近1000根K线
	pipe := s.client.Pipeline()
	pipe.LPush(s.ctx, key, data)
	pipe.LTrim(s.ctx, key, 0, 999) // 只保留最近1000根
	pipe.Expire(s.ctx, key, 7*24*time.Hour) // 7天过期

	_, err = pipe.Exec(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to save kline to redis: %w", err)
	}

	return nil
}

// SaveTicker 保存Ticker数据
func (s *RedisStorage) SaveTicker(ticker *models.Ticker) error {
	key := constants.RedisKeyTicker + ticker.Symbol

	// 使用 Hash 存储
	data := map[string]interface{}{
		"last_price": ticker.LastPrice,
		"bid_price":  ticker.BidPrice,
		"ask_price":  ticker.AskPrice,
		"high_24h":   ticker.High24h,
		"low_24h":    ticker.Low24h,
		"volume_24h": ticker.Volume24h,
		"timestamp":  ticker.Timestamp,
	}

	if err := s.client.HSet(s.ctx, key, data).Err(); err != nil {
		return fmt.Errorf("failed to save ticker to redis: %w", err)
	}

	// 设置过期时间
	s.client.Expire(s.ctx, key, 1*time.Hour)

	// 发布到 Redis Pub/Sub
	channel := fmt.Sprintf("%s%s:%s", constants.RedisChannelMarket, ticker.Symbol, constants.DataTypeTicker)
	jsonData, _ := utils.ToJSON(ticker)
	s.client.Publish(s.ctx, channel, jsonData)

	return nil
}

// SaveDepth 保存深度数据
func (s *RedisStorage) SaveDepth(depth *models.OrderBook) error {
	key := constants.RedisKeyDepth + depth.Symbol

	// 将深度数据转换为JSON
	data, err := utils.ToJSON(depth)
	if err != nil {
		return err
	}

	// 使用 String 存储
	if err := s.client.Set(s.ctx, key, data, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to save depth to redis: %w", err)
	}

	// 发布到 Redis Pub/Sub
	channel := fmt.Sprintf("%s%s:%s", constants.RedisChannelMarket, depth.Symbol, constants.DataTypeDepth)
	s.client.Publish(s.ctx, channel, data)

	return nil
}

// SaveTrade 保存交易数据
func (s *RedisStorage) SaveTrade(trade *models.Trade) error {
	key := constants.RedisKeyTrade + trade.Symbol

	// 将交易转换为JSON
	data, err := utils.ToJSON(trade)
	if err != nil {
		return err
	}

	// 使用 List 存储最近的交易记录
	pipe := s.client.Pipeline()
	pipe.LPush(s.ctx, key, data)
	pipe.LTrim(s.ctx, key, 0, 99) // 只保留最近100条
	pipe.Expire(s.ctx, key, 1*time.Hour)

	_, err = pipe.Exec(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to save trade to redis: %w", err)
	}

	// 发布到 Redis Pub/Sub
	channel := fmt.Sprintf("%s%s:%s", constants.RedisChannelMarket, trade.Symbol, constants.DataTypeTrade)
	s.client.Publish(s.ctx, channel, data)

	return nil
}

// GetKlines 获取K线数据
func (s *RedisStorage) GetKlines(symbol, interval string, limit int64) ([]*models.Kline, error) {
	key := fmt.Sprintf("%s%s:%s", constants.RedisKeyKline, symbol, interval)

	// 从 List 中获取数据
	results, err := s.client.LRange(s.ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get klines from redis: %w", err)
	}

	klines := make([]*models.Kline, 0, len(results))
	for _, data := range results {
		var kline models.Kline
		if err := utils.FromJSON(data, &kline); err != nil {
			continue
		}
		klines = append(klines, &kline)
	}

	return klines, nil
}

// GetTicker 获取Ticker数据
func (s *RedisStorage) GetTicker(symbol string) (*models.Ticker, error) {
	key := constants.RedisKeyTicker + symbol

	data, err := s.client.HGetAll(s.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker from redis: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("ticker not found")
	}

	ticker := &models.Ticker{
		Symbol: symbol,
	}

	// 解析数据
	if val, ok := data["last_price"]; ok {
		fmt.Sscanf(val, "%f", &ticker.LastPrice)
	}
	if val, ok := data["bid_price"]; ok {
		fmt.Sscanf(val, "%f", &ticker.BidPrice)
	}
	if val, ok := data["ask_price"]; ok {
		fmt.Sscanf(val, "%f", &ticker.AskPrice)
	}
	if val, ok := data["high_24h"]; ok {
		fmt.Sscanf(val, "%f", &ticker.High24h)
	}
	if val, ok := data["low_24h"]; ok {
		fmt.Sscanf(val, "%f", &ticker.Low24h)
	}
	if val, ok := data["volume_24h"]; ok {
		fmt.Sscanf(val, "%f", &ticker.Volume24h)
	}
	if val, ok := data["timestamp"]; ok {
		fmt.Sscanf(val, "%d", &ticker.Timestamp)
	}

	return ticker, nil
}

// GetDepth 获取深度数据
func (s *RedisStorage) GetDepth(symbol string) (*models.OrderBook, error) {
	key := constants.RedisKeyDepth + symbol

	data, err := s.client.Get(s.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get depth from redis: %w", err)
	}

	var depth models.OrderBook
	if err := utils.FromJSON(data, &depth); err != nil {
		return nil, err
	}

	return &depth, nil
}

// Close 关闭连接
func (s *RedisStorage) Close() error {
	return s.client.Close()
}
