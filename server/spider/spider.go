package spider

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"net/url"
	"os"
	"path"
	"reflect"
	"sync"
	"time"
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
	Crawl(*Resource, *sync.WaitGroup, clogs.LogInterface) error
	GetConfig() (interface{}, string)
	SetDB(*gorm.DB)
}

type Spider struct {
	logDriver    clogs.LogInterface
	sd           SpiderDriver
	bathPath     string
	isOnce       bool   // 是否只抓一个传入的地址
	onceUrl      string //
	Async        bool   // 是否开启协程
	LimitGos     int    // 限制协程数量
	maxSize      int    // MB
	userAgent    string
	pool         *ants.Pool
	driverConfig interface{}
	wg           *sync.WaitGroup
	db           *gorm.DB
}

func NewSpider(sd SpiderDriver, logDriver clogs.LogInterface, opts ...Option) (s *Spider, err error) {
	s = &Spider{
		sd:        sd,
		maxSize:   50,
		logDriver: logDriver,
		bathPath:  path.Join(vars.BasePath, vars.Config.Video.SavePath),
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		wg:        &sync.WaitGroup{},
	}
	if err = s.LoadConfig(sd.GetConfig()); err != nil {
		return
	}

	for _, opt := range opts {
		opt(s)
	}
	if s.db != nil {
		sd.SetDB(s.db)
	}

	return
}

func (s *Spider) CrawlOne(host string) (err error) {
	u, err := url.Parse(host)
	if err != nil {
		err = fmt.Errorf("抓取地址错误：%s", err.Error())
		return
	}

	c := colly.NewCollector(
		colly.UserAgent(s.userAgent),
		colly.DetectCharset(),
		colly.AllowedDomains(u.Host),
		colly.MaxBodySize(s.maxSize*1024*1024),
	)

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
			s.wg.Add(1)
			err = s.sd.Crawl(resource, s.wg, s.logDriver)
			if err != nil {
				s.logDriver.ErrLog(map[string]interface{}{"resource": resource}, "抓取资源失败 [Crawl]: "+utils.ParseError(err))
				return
			}
		}
		return
	})

	_ = c.Visit(host)
	s.wg.Wait()
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
		s.pool, err = ants.NewPool(s.LimitGos)
		if err != nil {
			err = fmt.Errorf("协程池启动失败：%s", err.Error())
			return
		}
		defer s.pool.Release()

		// 默认启动一个，防止抓取网页慢直接退出了
		s.wg.Add(1)
		s.pool.Submit(func() {
			time.Sleep(time.Second * 20)
			fmt.Println("结束默认协程")
			s.wg.Done()
		})
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
			s.wg.Add(1)
			if s.isOnce {
				err = s.sd.Crawl(resource, s.wg, s.logDriver)
				if err != nil {
					s.logDriver.ErrLog(map[string]interface{}{"resource": resource}, "抓取资源失败 [Crawl]: "+utils.ParseError(err))
					return
				}
			} else {
				s.pool.Submit(func() {
					err = s.sd.Crawl(resource, s.wg, s.logDriver)
					if err != nil {
						s.logDriver.ErrLog(map[string]interface{}{"resource": resource}, "抓取资源失败 [Crawl]: "+utils.ParseError(err))
						return
					}
				})
			}
		}
		return
	})

	_ = c.Visit(crawlHost)
	s.wg.Wait()

	return
}

func (s *Spider) LoadConfig(configStruct interface{}, filePath string) (err error) {
	if _, err = os.Stat(filePath); err != nil {
		s.logDriver.ErrLog(map[string]interface{}{"filepath": filePath, "error": err}, "驱动配置文件不存在")
		return
	}

	if of := reflect.TypeOf(configStruct); of.Kind() != reflect.Pointer {
		s.logDriver.ErrLog(map[string]interface{}{"filepath": filePath, "error": err}, "非指针配置信息不可使用")
		return
	}

	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		s.logDriver.ErrLog(map[string]interface{}{"filepath": filePath, "error": err}, "驱动配置文件读取失败 [ReadFile]")
		return
	}

	err = yaml.Unmarshal(yamlFile, configStruct)
	if err != nil {
		s.logDriver.ErrLog(map[string]interface{}{"filepath": filePath, "error": err}, "驱动配置文件读取失败 [Unmarshal]")
		return
	}

	s.driverConfig = configStruct
	return
}
