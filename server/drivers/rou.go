package drivers

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"videocapture/server/spider"
	"videocapture/utils"
	"videocapture/utils/clogs"
	"videocapture/vars"

	"gopkg.in/yaml.v3"
)

type Rou struct {
	Log clogs.LogInterface
}

type RouConfig struct {
	Host                string `yaml:"host"`
	ResourceUrlContains string `yaml:"resource_url_contains"`
}

func (r *Rou) GetHost(c interface{}) string {
	return c.(*RouConfig).Host
}

func (r *Rou) GetConfig() interface{} {
	yamlFile, err := os.ReadFile(vars.BasePath + "/config/default_driver.yaml")
	if err != nil {
		log.Fatal("驱动配置文件读取失败:", err)
	}
	rc := &RouConfig{}
	err = yaml.Unmarshal(yamlFile, rc)
	if err != nil {
		log.Fatal("驱动配置文件读取失败:", err)
	}
	return rc
}

func (r *Rou) ParseResource(u *url.URL, body []byte, c interface{}) (rs *spider.Resource, ok bool, err error) {
	if strings.HasPrefix(u.String(), c.(*RouConfig).ResourceUrlContains) {
		resourceUrl, title := "", ""
		title = titleByRegexp(string(body))
		if title == "" {
			r.Log.WarnLog(map[string]interface{}{
				"requestUrl": u.Path,
			}, "没有匹配到视频名称")
			return
		}
		resourceUrl, err = url.JoinPath(
			fmt.Sprintf("%s://%s", u.Scheme, u.Host),
			"api",
			u.Path,
		)
		if err != nil {
			r.Log.ErrLog(map[string]interface{}{
				"requestUrl": u.Path,
				"title":      title,
			}, "视频地址拼接失败："+utils.ParseError(err))
			return
		}

		return &spider.Resource{
			ResourceUrl: resourceUrl,
			Title:       title,
			SavePath:    path.Join(vars.BasePath, vars.Config.Video.SavePath, "rou"),
		}, true, nil
	}
	return nil, false, nil
}

func (r *Rou) IsRequest(host string) bool {
	return true
}

func (r *Rou) Crawl(*spider.Resource) error {
	return nil
}

func titleByRegexp(body string) (title string) {
	reg := regexp.MustCompile(`<div class="text-base md:text-xl mb-1">(.*?)</div>`)
	params := reg.FindStringSubmatch(body)
	if len(params) != 2 {
		return ""
	}
	return params[1]
}
