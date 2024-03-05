package main

import (
	"context"
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/fatedier/frp/pkg/config/v1/validation"
	frpLog "github.com/fatedier/frp/pkg/util/log"
	"github.com/fatedier/frp/server"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/frp-api/handler"
	"github.com/lixiang4u/frp-api/model"
	"os"
	"time"
)

func main() {
	go runFrpServer()
	httpServer()
}

func httpServer() {
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

	r.GET("/api/config", handler.ApiConfig)
	r.POST("/api/vhost", handler.ApiNewClientVhost)
	r.GET("/api/vhosts", handler.ApiClientVhostList)
	r.DELETE("/api/vhost/:machine_id/:vhost_id", handler.ApiClientVhostRemove)

	//r.Static("/files/", ".")
	r.NoRoute(handler.ApiNotRoute)

	_ = r.Run(fmt.Sprintf(":%d", model.AppServerPort))
}

func runFrpServer() {
	var cfg v1.ServerConfig
	cfg.Complete()

	var appConfig = model.GetAppConfig()
	if appConfig.BindPort > 0 {
		cfg.BindPort = appConfig.BindPort
	}
	if appConfig.VhostHTTPPort > 0 {
		cfg.VhostHTTPPort = appConfig.VhostHTTPPort
	}
	if appConfig.VhostHTTPSPort > 0 {
		cfg.VhostHTTPSPort = appConfig.VhostHTTPSPort
	}

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
