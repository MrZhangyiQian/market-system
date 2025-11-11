package svc

import (
	"context"
	"fmt"
	"market-system/services/api/internal/config"
	ws "market-system/services/api/internal/websocket"
	"time"

	"github.com/redis/go-redis/v9"
)

type ServiceContext struct {
	Config      config.Config
	Redis       *redis.Client
	WsHub       *ws.Hub
	Broadcaster *ws.Broadcaster
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 初始化 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port),
		Password:     c.Redis.Password,
		DB:           c.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     100,
		MinIdleConns: 10,
	})

	// 测试连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	// 初始化 WebSocket Hub
	hub := ws.NewHub()

	// 初始化 Broadcaster
	broadcaster := ws.NewBroadcaster(hub, rdb)

	return &ServiceContext{
		Config:      c,
		Redis:       rdb,
		WsHub:       hub,
		Broadcaster: broadcaster,
	}
}
