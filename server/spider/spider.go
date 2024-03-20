package spider

import (
	"fmt"
	"net/url"
	"path"
	"sync"
	"videocapture/utils"
	"videocapture/utils/clogs"
	"videocapture/vars"

	"github.com/gocolly/colly/v2"
	"github.com/panjf2000/ants/v2"
)

type SpiderDriver interface {
	GetHost(interface{}) string
	ParseResource(u *url.URL, body []byte, c interface{}) (*Resource, bool, error)
	IsRequest(host string) bool
	Crawl(*Resource) error
	GetConfig() interface{}
}

type Spider struct {
	logDriver    clogs.LogInterface
	sd           SpiderDriver
	bathPath     string
	isOnce       bool // 是否只抓一个传入的地址
	Async        bool // 是否开启协程
	LimitGos     int  // 限制协程数量
	maxSize      int  // MB
	userAgent    string
	pool         *ants.Pool
	driverConfig interface{}
	*sync.WaitGroup
}

func NewSpider(sd SpiderDriver, logDriver clogs.LogInterface, opts ...Option) (s *Spider) {
	s = &Spider{
		maxSize:   50,
		logDriver: logDriver,
		bathPath:  path.Join(vars.BasePath, vars.Config.Video.SavePath),
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}
	s.LoadConfig(sd.GetConfig())

	for _, opt := range opts {
		opt(s)
	}

	return
}

func (s *Spider) Start() (err error) {
	crawlHost := s.sd.GetHost(s.driverConfig)
	u, err := url.Parse(crawlHost)
	if err != nil {
		err = fmt.Errorf("抓取地址错误：%s", err.Error())
		return
	}

	if s.Async {
		defer ants.Release()
		s.pool, err = ants.NewPool(s.LimitGos)
	}

	c := colly.NewCollector(
		colly.UserAgent(s.userAgent),
		colly.DetectCharset(),
		colly.AllowedDomains(u.Host),
		colly.MaxBodySize(s.maxSize*1024*1024),
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		host := e.Attr("href")
		if !s.isOnce && s.sd.IsRequest(host) {
			e.Request.Visit(host)
		}
	})

	c.OnResponse(func(res *colly.Response) {
		// 资源解析详情
		resource, ok, err := s.sd.ParseResource(res.Request.URL, res.Body, s.driverConfig)
		if err != nil {
			s.logDriver.ErrLog(map[string]interface{}{
				"requestUrl": res.Request.URL.String(),
			}, "解析资源地址失败 [ParseResource]: "+utils.ParseError(err))
			return
		}
		if ok {
			ants.Submit(func() {
				err = s.sd.Crawl(resource)
				if err != nil {
					s.logDriver.ErrLog(map[string]interface{}{"resource": resource}, "抓取资源失败 [Crawl]: "+utils.ParseError(err))
					return
				}
			})
		}
		return
	})

	_ = c.Visit(crawlHost)
	c.Wait()

	return
}

func (s *Spider) LoadConfig(conf interface{}) {
	s.driverConfig = conf
}
