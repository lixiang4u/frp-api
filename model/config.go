package model

import (
	"github.com/spf13/viper"
	"log"
	"os"
)

var appConfig AppConfig
var AppSecret = "frp-api-xxx"
var AppServerPort int

func init() {
	viper.SetConfigFile("frps.toml")
	if e := viper.ReadInConfig(); e != nil {
		log.Println("[configError]", e.Error())
		os.Exit(1)
	}
	appConfig.BindPort = viper.GetInt("bindPort")
	appConfig.VhostHTTPPort = viper.GetInt("vhostHTTPPort")
	appConfig.VhostHTTPSPort = viper.GetInt("vhostHTTPSPort")

	appConfig.ClientDefaultTls = ClientDefaultTls{
		Force:    viper.GetBool("client_default_tls.force"),
		CertFile: viper.GetString("client_default_tls.certFile"),
		KeyFile:  viper.GetString("client_default_tls.keyFile"),
	}
	AppServerPort = viper.GetInt("server_port")
}

type ClientDefaultTls struct {
	Force    bool   `json:"force"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
}

type AppConfig struct {
	BindPort         int              `json:"bindPort"`
	VhostHTTPPort    int              `json:"vhostHTTPPort"`
	VhostHTTPSPort   int              `json:"vhostHTTPSPort"`
	ClientDefaultTls ClientDefaultTls `json:"client_default_tls"`
}

func GetAppConfig() AppConfig {
	return appConfig
}
