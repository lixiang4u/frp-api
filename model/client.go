package model

import "sync"

type Vhost struct {
	Id           string `json:"id"` // type+custom_domain值的md5,默认每个用户的type+custom_domain都是唯一
	Type         string `json:"type"`
	Name         string `json:"name"`
	CustomDomain string `json:"custom_domain"`
	LocalAddr    string `json:"local_addr"`
	CrtPath      string `json:"crt_path"`
	KeyPath      string `json:"key_path"`
}

type Client struct {
	Id     string           `json:"id"` //客户端机器码
	Vhosts map[string]Vhost `json:"vhosts"`
}

var ClientMap sync.Map
