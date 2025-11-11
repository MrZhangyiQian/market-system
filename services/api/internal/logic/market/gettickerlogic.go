package market

import (
	"context"
	"fmt"
	"market-system/common/constants"

	"market-system/services/api/internal/svc"
	"market-system/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTickerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTickerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTickerLogic {
	return &GetTickerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTickerLogic) GetTicker(req *types.TickerRequest) (resp *types.TickerResponse, err error) {
	// 从 Redis 获取 Ticker 数据
	key := constants.RedisKeyTicker + req.Symbol

	data, err := l.svcCtx.Redis.HGetAll(l.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("ticker not found for symbol: %s", req.Symbol)
	}

	resp = &types.TickerResponse{
		Symbol: req.Symbol,
	}

	// 解析数据
	if val, ok := data["last_price"]; ok {
		fmt.Sscanf(val, "%f", &resp.LastPrice)
	}
	if val, ok := data["bid_price"]; ok {
		fmt.Sscanf(val, "%f", &resp.BidPrice)
	}
	if val, ok := data["ask_price"]; ok {
		fmt.Sscanf(val, "%f", &resp.AskPrice)
	}
	if val, ok := data["high_24h"]; ok {
		fmt.Sscanf(val, "%f", &resp.High24h)
	}
	if val, ok := data["low_24h"]; ok {
		fmt.Sscanf(val, "%f", &resp.Low24h)
	}
	if val, ok := data["volume_24h"]; ok {
		fmt.Sscanf(val, "%f", &resp.Volume24h)
	}
	if val, ok := data["timestamp"]; ok {
		fmt.Sscanf(val, "%d", &resp.Timestamp)
	}

	return resp, nil
}
