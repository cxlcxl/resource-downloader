package vars

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
	"videocapture/config"
)

var (
	BasePath string
	Config   *config.Config
)

const (
	DateTimeFormat = "2006-01-02 15:04:05"
)

func init() {
	if dir, err := os.Getwd(); err != nil {
		log.Fatal("文件目录获取失败")
	} else {
		// 路径进行处理，兼容单元测试程序程序启动时的奇怪路径
		if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-test") {
			BasePath = strings.Replace(strings.Replace(dir, `\test`, "", 1), `/test`, "", 1)
		} else {
			BasePath = dir
		}
	}

	yamlFile, err := os.ReadFile(BasePath + "/config/config.yml")
	if err != nil {
		log.Fatal("配置文件读取失败", err)
	}
	err = yaml.Unmarshal(yamlFile, &Config)
	if err != nil {
		log.Fatal("配置文件读取失败", err)
	}
}
