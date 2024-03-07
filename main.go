package main

import (
	"context"
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	frpLog "github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/pkg/util/util"
	"github.com/fatedier/frp/server"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/frp-api/handler"
	"github.com/lixiang4u/frp-api/model"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	go loadSaveClientMap()
	go runFrpServer()
	httpServer()
}

func httpServer() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGKILL)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	r.GET("/api/config", handler.ApiRecover(handler.ApiConfig))
	r.POST("/api/vhost", handler.ApiRecover(handler.ApiNewClientVhost))
	r.GET("/api/vhosts", handler.ApiRecover(handler.ApiClientVhostList))
	r.DELETE("/api/vhost/:machine_id/:vhost_id", handler.ApiRecover(handler.ApiClientVhostRemove))

	r.POST("/api/debug/vhosts", handler.ApiRecover(handler.ApiDebugVhostList))
	r.POST("/api/use-port-check", handler.ApiRecover(handler.ApiUsePortCheck))

	r.NoRoute(handler.ApiRecover(handler.ApiNotRoute))

	go func() {
		_ = r.Run(fmt.Sprintf(":%d", model.AppServerPort))
	}()

	select {
	case _sig := <-sig:
		model.SaveClientMap()
		log.Println(fmt.Sprintf("[stop] %v", _sig))
	}
}

func runFrpServer() {
	var cfg v1.ServerConfig
	cfg.Complete()

	var appConfig = model.GetAppConfig()

	cfg.BindPort = util.EmptyOr(appConfig.BindPort, 0)
	cfg.VhostHTTPPort = util.EmptyOr(appConfig.VhostHTTPPort, 0)
	cfg.VhostHTTPSPort = util.EmptyOr(appConfig.VhostHTTPSPort, 0)
	cfg.TCPMuxHTTPConnectPort = util.EmptyOr(appConfig.TcpMuxHTTPConnectPort, 0)

	warning, err := validation.ValidateServerConfig(&cfg)
	if warning != nil {
		frpLog.Info("WARNING: %v\n", warning)
	}
	if err != nil {
		frpLog.Error("[frpConfigError]: %v\n", err.Error())
		os.Exit(1)
	}

	frpLog.InitLog(cfg.Log.To, cfg.Log.Level, cfg.Log.MaxDays, cfg.Log.DisablePrintColor)

	svr, err := server.NewService(&cfg)
	if err != nil {
		frpLog.Error("[frpNewServiceError]: %v\n", err.Error())
		os.Exit(1)
	}
	frpLog.Info("frps started successfully")
	svr.Run(context.Background())

	frpLog.Info("frps stop")
}

func loadSaveClientMap() {
	model.LoadClientMap()
	var t = time.NewTicker(time.Hour)
	for {
		select {
		case <-t.C:
			log.Println("[SaveClientMap]")
			model.SaveClientMap()
		}
	}
}
