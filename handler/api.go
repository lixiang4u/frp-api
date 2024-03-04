package handler

import (
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/frp-api/model"
	"github.com/lixiang4u/frp-api/utils"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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

	root, _ := filepath.Abs(filepath.Join("static"))
	tmpFile, _ := filepath.Abs(filepath.Join("static", ctx.Request.RequestURI))
	_, err := os.Stat(tmpFile)
	if err == nil && strings.HasPrefix(tmpFile, root) {
		ctx.File(tmpFile)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 404,
		"msg":  "请求地址错误",
		"uri":  ctx.Request.RequestURI,
	})
}

func ApiNewClientVhost(ctx *gin.Context) {
	type Req struct {
		Type      string `json:"type" form:"type"`
		MachineId string `json:"machine_id" form:"machine_id"`
		LocalAddr string `json:"local_addr" form:"local_addr"`
		Name      string `json:"name" form:"name"` // 代码名称
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
	}
	if len(client.Vhosts) > 5 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "虚拟主机创建太多啦",
		})
		return
	}
	v2, ok := client.Vhosts[vhostId]
	if !ok {
		client.Vhosts[vhostId] = model.Vhost{
			Id:           vhostId,
			Type:         string(v1.ProxyTypeHTTP),
			Name:         req.Name,
			CustomDomain: fmt.Sprintf("%s.frp.lixiang4u.xyz", vhostId),
			LocalAddr:    req.LocalAddr,
			CrtPath:      "",
			KeyPath:      "",
		}
	} else {
		v2.Type = req.Type
		v2.LocalAddr = req.LocalAddr
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
