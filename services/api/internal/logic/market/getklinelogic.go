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

type GetKlineLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetKlineLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKlineLogic {
	return &GetKlineLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetKlineLogic) GetKline(req *types.KlineRequest) (resp *types.KlineResponse, err error) {
	// 从 Redis 获取 K线数据
	key := fmt.Sprintf("%s%s:%s", constants.RedisKeyKline, req.Symbol, req.Interval)

	// 获取数据
	results, err := l.svcCtx.Redis.LRange(l.ctx, key, 0, req.Limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// 解析数据
	klines := make([]types.Kline, 0, len(results))
	for _, data := range results {
		var kline models.Kline
		if err := json.Unmarshal([]byte(data), &kline); err != nil {
			continue
		}

		klines = append(klines, types.Kline{
			OpenTime:  kline.OpenTime,
			CloseTime: kline.CloseTime,
			Open:      kline.Open,
			High:      kline.High,
			Low:       kline.Low,
			Close:     kline.Close,
			Volume:    kline.Volume,
			QuoteVol:  kline.QuoteVol,
			TradeNum:  kline.TradeNum,
		})
	}

	resp = &types.KlineResponse{
		Symbol:   req.Symbol,
		Interval: req.Interval,
		Data:     klines,
	}

	return resp, nil
}
