package model

import (
	"github.com/spf13/viper"
	"log"
)

var appConfig AppConfig
var AppSecret = "frp-api-xxx"

func init() {
	viper.SetConfigFile("frps.toml")
	if e := viper.ReadInConfig(); e != nil {
		log.Println("[configError]", e.Error())
	}
	appConfig.BindPort = viper.GetInt("bindPort")
	appConfig.VhostHTTPPort = viper.GetInt("vhostHTTPPort")
	appConfig.VhostHTTPSPort = viper.GetInt("vhostHTTPSPort")
}

type AppConfig struct {
	BindPort       int `json:"bindPort"`
	VhostHTTPPort  int `json:"vhostHTTPPort"`
	VhostHTTPSPort int `json:"vhostHTTPSPort"`
}

func GetAppConfig() AppConfig {
	return appConfig
}
