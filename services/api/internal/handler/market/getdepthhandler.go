package market

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"market-system/services/api/internal/logic/market"
	"market-system/services/api/internal/svc"
	"market-system/services/api/internal/types"
)

func GetDepthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DepthRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := market.NewGetDepthLogic(r.Context(), svcCtx)
		resp, err := l.GetDepth(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
