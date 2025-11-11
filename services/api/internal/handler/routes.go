package handler

import (
	"net/http"

	market "market-system/services/api/internal/handler/market"
	"market-system/services/api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/ticker/:symbol",
				Handler: market.GetTickerHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/kline",
				Handler: market.GetKlineHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/depth/:symbol",
				Handler: market.GetDepthHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api/v1"),
	)
}
