package main

import (
	"flag"
	"fmt"
	"log"

	"market-system/services/api/internal/config"
	"market-system/services/api/internal/handler"
	"market-system/services/api/internal/svc"
	ws "market-system/services/api/internal/websocket"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/market-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 添加WebSocket路由
	wsHandler := ws.NewHandler(ctx.WsHub)
	server.AddRoute(rest.Route{
		Method:  "GET",
		Path:    "/ws",
		Handler: wsHandler.ServeHTTP,
	})

	// 启动WebSocket Hub
	go ctx.WsHub.Run()
	log.Println("[Main] WebSocket Hub started")

	// 启动Redis广播器
	go ctx.Broadcaster.Start()
	log.Println("[Main] Redis Broadcaster started")

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	fmt.Printf("WebSocket endpoint: ws://%s:%d/ws\n", c.Host, c.Port)
	server.Start()
}
