package handler

import (
	"fmt"
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/gin-gonic/gin"
	"github.com/lixiang4u/frp-api/model"
	"github.com/lixiang4u/frp-api/utils"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	minUsePort = 30000
	maxUsePort = 60000
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
			"bind_port":                 appConfig.BindPort,
			"vhost_http_port":           appConfig.VhostHTTPPort,
			"vhost_https_port":          appConfig.VhostHTTPSPort,
			"tcp_mux_http_connect_port": appConfig.TcpMuxHTTPConnectPort,
			"host":                      host,
			"max_use_port":              maxUsePort,
			"min_use_port":              minUsePort,
		},
	})
}

func ApiUsePortCheck(ctx *gin.Context) {
	var p = 0
	var i int
	for i = 1; i <= 30; i++ {
		p = rand.IntN(maxUsePort-minUsePort+1) + minUsePort
		if _, err := utils.IsPortAvailable(p); err == nil {
			break
		}
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":   200,
		"port":   p,
		"effort": i,
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
		Name       string `json:"name" form:"name"`     // 代码名称
		Status     bool   `json:"status" form:"status"` //true.开启，false.关闭
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
	}
	if len(client.Vhosts) >= 5 && len(req.Id) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 500,
			"msg":  "代理创建太多啦",
		})
		return
	}
	// 如果指定了id，则查询是否存在
	if len(req.Id) > 0 {
		_, ok = client.Vhosts[req.Id]
		if !ok {
			ctx.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  fmt.Sprintf("查找vhost失败：%s", req.Id),
			})
			return
		}
		if ok {
			vhostId = req.Id
		}
	} else if req.Type == string(v1.ProxyTypeTCP) {
		// 如果是新增tcp代理(独占端口)，需要检测可用性
		if req.RemotePort <= 0 {
			ctx.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  fmt.Sprintf("服务器端口设置错误：%d", req.RemotePort),
			})
			return
		}
		if _, err = utils.IsPortAvailable(req.RemotePort); err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  fmt.Sprintf("服务器端口不可用：%s", err.Error()),
			})
			return
		}
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
			Status:       req.Status,
			CreatedAt:    time.Now().Unix(),
		}
		if req.Type == string(v1.ProxyTypeHTTPS) {
			tmpVhost.CrtPath = string(utils.FileContents(appConfig.ClientDefaultTls.CertFile))
			tmpVhost.KeyPath = string(utils.FileContents(appConfig.ClientDefaultTls.KeyFile))
		}
		client.Vhosts[vhostId] = tmpVhost
	} else {
		v2.Type = req.Type
		v2.LocalAddr = req.LocalAddr
		//v2.RemotePort = req.RemotePort// 不支持修改，防止端口占用没检测
		v2.Name = req.Name
		v2.Status = req.Status

		// 重置自定义域名
		cnameDomain, _ := model.LoadCustomDomainMap()[v2.CustomDomain]
		v2.CnameDomain = cnameDomain

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
