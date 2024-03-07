package handler

import (
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/frp-api/model"
	"github.com/lixiang4u/frp-api/utils"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func ApiRecover(h gin.HandlerFunc) gin.HandlerFunc {
	defer func() {
		if err := recover(); err != nil {
			log.Println("[recover]", err)
		}
	}()
	return func(ctx *gin.Context) {
		h(ctx)
	}
}

func ApiConfig(ctx *gin.Context) {
	var appConfig = model.GetAppConfig()

	host, _, _ := net.SplitHostPort(ctx.Request.Host)

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"config": gin.H{
			"bind_port":        appConfig.BindPort,
			"vhost_http_port":  appConfig.VhostHTTPPort,
			"vhost_https_port": appConfig.VhostHTTPSPort,
			"host":             host,
		},
	})
}

func ApiNotRoute(ctx *gin.Context) {
	tmpUrl, _ := url.PathUnescape(ctx.Request.RequestURI)

	root, _ := filepath.Abs(filepath.Join("static"))
	tmpFile, _ := filepath.Abs(filepath.Join("static", tmpUrl))
	_, err := os.Stat(tmpFile)
	if err == nil && strings.HasPrefix(tmpFile, root) {
		ctx.Status(http.StatusOK)
		ctx.File(tmpFile)
		return
	}

	ctx.JSON(http.StatusNotFound, gin.H{
		"code": 404,
		"msg":  "请求地址错误",
		"path": tmpUrl,
	})
}

func ApiNewClientVhost(ctx *gin.Context) {
	type Req struct {
		Id         string `json:"id" form:"id"`
		Type       string `json:"type" form:"type"`
		MachineId  string `json:"machine_id" form:"machine_id"`
		LocalAddr  string `json:"local_addr" form:"local_addr"`
		RemotePort int    `json:"remote_port" json:"remote_port"`
		Name       string `json:"name" form:"name"` // 代码名称
		//LocalPort int    `json:"local_port" form:"local_port"`
	}
	var req Req
	_ = ctx.ShouldBind(&req)
	if len(req.MachineId) < 16 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "参数错误",
		})
		return
	}
	_, p, err := net.SplitHostPort(req.LocalAddr)
	if err != nil || len(p) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  fmt.Sprintf("本地地址错误：%s", err.Error()),
		})
		return
	}
	if len(req.Type) == 0 {
		req.Type = string(v1.ProxyTypeHTTP)
	}
	var appConfig = model.GetAppConfig()
	if req.Type == string(v1.ProxyTypeHTTPS) && utils.FileExists(appConfig.ClientDefaultTls.CertFile, appConfig.ClientDefaultTls.KeyFile) == false {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "系统没有配置默认证书",
		})
		return
	}

	var vhostId = utils.NewHostName(req.MachineId, req.Type, req.LocalAddr, req.Name)
	var client model.Client
	v, ok := model.ClientMap.Load(req.MachineId)
	if !ok {
		client = model.Client{
			Id:     req.MachineId,
			Vhosts: make(map[string]model.Vhost),
		}
	} else {
		client = v.(model.Client)
		// 如果已经存在该vhost，则修改
		if _, ok = client.Vhosts[req.Id]; ok {
			vhostId = req.Id
		}
	}
	if len(client.Vhosts) >= 10 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "虚拟主机创建太多啦",
		})
		return
	}
	v2, ok := client.Vhosts[vhostId]
	if !ok {
		var tmpVhost = model.Vhost{
			Id:           vhostId,
			Type:         req.Type,
			Name:         req.Name,
			CustomDomain: fmt.Sprintf("%s.%s", vhostId, model.AppServerPrefix),
			LocalAddr:    req.LocalAddr,
			RemotePort:   req.RemotePort,
			CrtPath:      "",
			KeyPath:      "",
		}
		if req.Type == string(v1.ProxyTypeHTTPS) {
			tmpVhost.CrtPath = string(utils.FileContents(appConfig.ClientDefaultTls.CertFile))
			tmpVhost.KeyPath = string(utils.FileContents(appConfig.ClientDefaultTls.KeyFile))
		}
		client.Vhosts[vhostId] = tmpVhost
	} else {
		v2.Type = req.Type
		v2.LocalAddr = req.LocalAddr
		v2.RemotePort = req.RemotePort
		v2.Name = req.Name
		client.Vhosts[vhostId] = v2
	}

	model.ClientMap.Store(req.MachineId, client)

	ctx.JSON(http.StatusOK, gin.H{
		"code":  200,
		"vhost": client.Vhosts[vhostId],
	})
}

func ApiClientVhostList(ctx *gin.Context) {
	var machineId = ctx.Query("machine_id")
	var client model.Client
	v, ok := model.ClientMap.Load(machineId)
	if !ok {
		client = model.Client{Vhosts: make(map[string]model.Vhost)}
	} else {
		client = v.(model.Client)
	}

	var lst = make([]model.Vhost, 0)
	for _, vhost := range client.Vhosts {
		lst = append(lst, vhost)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":   200,
		"vhosts": lst,
	})
}

func ApiClientVhostRemove(ctx *gin.Context) {
	var machineId = ctx.Param("machine_id")
	var vhostId = ctx.Param("vhost_id")
	if len(machineId) < 16 || len(vhostId) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "参数错误",
		})
		return
	}

	v, ok := model.ClientMap.Load(machineId)
	if !ok {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "数据不存在",
		})
		return
	}
	var client = v.(model.Client)
	delete(client.Vhosts, vhostId)

	ctx.JSON(http.StatusOK, gin.H{
		"code":   200,
		"vhosts": client.Vhosts,
	})
}

func ApiDebugVhostList(ctx *gin.Context) {
	model.ClientMap.Range(func(key, value any) bool {
		log.Println("[vhost]", key, utils.ToJsonString(value))
		return true
	})

	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
	})
}
