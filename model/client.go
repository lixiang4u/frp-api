package model

import (
	"fmt"
	"github.com/go-jose/go-jose/v3/json"
	"github.com/lixiang4u/frp-api/utils"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

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

func LoadClientMap() {
	// 从文件加载
	var err = filepath.WalkDir("data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		buf, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var client Client
		if err = json.Unmarshal(buf, &client); err != nil {
			return err
		}
		var key = strings.TrimRight(d.Name(), filepath.Ext(d.Name()))
		ClientMap.Store(key, client)

		return nil
	})
	if err != nil {
		log.Println("[LoadMapFromFileError]", err.Error())
	}
}

func SaveClientMap() {
	// 保存到文件
	var file string
	ClientMap.Range(func(key, value any) bool {
		file = utils.AppFilePathMake("data", key.(string)[:2], fmt.Sprintf("%s.json", key.(string)))
		if err := os.WriteFile(file, utils.ToJsonBytes(value), os.ModePerm); err != nil {
			log.Println(fmt.Sprintf("[SaveMapToFileError] %s, %s", key.(string), err.Error()))
		}
		return true
	})
}
