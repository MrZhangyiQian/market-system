package market

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"market-system/services/api/internal/logic/market"
	"market-system/services/api/internal/svc"
	"market-system/services/api/internal/types"
)

func GetTickerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TickerRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := market.NewGetTickerLogic(r.Context(), svcCtx)
		resp, err := l.GetTicker(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
