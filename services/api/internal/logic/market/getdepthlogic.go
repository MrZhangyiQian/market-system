package market

import (
	"context"
	"encoding/json"
	"fmt"
	"market-system/common/constants"
	"market-system/common/models"

	"market-system/services/api/internal/svc"
	"market-system/services/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDepthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDepthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDepthLogic {
	return &GetDepthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDepthLogic) GetDepth(req *types.DepthRequest) (resp *types.DepthResponse, err error) {
	// 从 Redis 获取深度数据
	key := constants.RedisKeyDepth + req.Symbol

	data, err := l.svcCtx.Redis.Get(l.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get depth: %w", err)
	}

	var depth models.OrderBook
	if err := json.Unmarshal([]byte(data), &depth); err != nil {
		return nil, fmt.Errorf("failed to parse depth data: %w", err)
	}

	// 转换为响应格式
	bids := make([]types.PriceLevel, 0)
	asks := make([]types.PriceLevel, 0)

	// 限制档位数量
	limit := int(req.Limit)
	if limit > len(depth.Bids) {
		limit = len(depth.Bids)
	}
	for i := 0; i < limit; i++ {
		bids = append(bids, types.PriceLevel{
			Price:  depth.Bids[i].Price,
			Amount: depth.Bids[i].Amount,
		})
	}

	limit = int(req.Limit)
	if limit > len(depth.Asks) {
		limit = len(depth.Asks)
	}
	for i := 0; i < limit; i++ {
		asks = append(asks, types.PriceLevel{
			Price:  depth.Asks[i].Price,
			Amount: depth.Asks[i].Amount,
		})
	}

	resp = &types.DepthResponse{
		Symbol:    req.Symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: depth.Timestamp,
	}

	return resp, nil
}
