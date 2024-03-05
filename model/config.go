package model

import (
	v1 "github.com/fatedier/frp/pkg/config/v1"
	"github.com/spf13/viper"
	"log"
	"os"
)

var appConfig AppConfig
var AppSecret = "frp-api-xxx"

func init() {
	viper.SetConfigFile("frps.toml")
	if e := viper.ReadInConfig(); e != nil {
		log.Println("[configError]", e.Error())
		os.Exit(1)
	}
	appConfig.BindPort = viper.GetInt("bindPort")
	appConfig.VhostHTTPPort = viper.GetInt("vhostHTTPPort")
	appConfig.VhostHTTPSPort = viper.GetInt("vhostHTTPSPort")

	appConfig.TLS = v1.TLSServerConfig{
		Force: viper.GetBool("transport.tls.force"),
		TLSConfig: v1.TLSConfig{
			CertFile:      viper.GetString("transport.tls.certFile"),
			KeyFile:       viper.GetString("transport.tls.keyFile"),
			TrustedCaFile: viper.GetString("transport.tls.trustedCaFile"),
			ServerName:    viper.GetString("transport.tls.certFile"),
		},
	}
}

type AppConfig struct {
	BindPort       int                `json:"bindPort"`
	VhostHTTPPort  int                `json:"vhostHTTPPort"`
	VhostHTTPSPort int                `json:"vhostHTTPSPort"`
	TLS            v1.TLSServerConfig `json:"tls"`
}

func GetAppConfig() AppConfig {
	return appConfig
}
