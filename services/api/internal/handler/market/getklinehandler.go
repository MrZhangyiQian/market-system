package market

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"market-system/services/api/internal/logic/market"
	"market-system/services/api/internal/svc"
	"market-system/services/api/internal/types"
)

func GetKlineHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.KlineRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := market.NewGetKlineLogic(r.Context(), svcCtx)
		resp, err := l.GetKline(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
