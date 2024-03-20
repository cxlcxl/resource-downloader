package collyserver

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"
	"videocapture/utils"

	"github.com/go-resty/resty/v2"
	"github.com/gocolly/colly/v2"
)

func (s *Spider) Run(basePath string) (err error) {
	u, err := url.Parse(s.sk.GetHost())
	if err != nil {
		err = fmt.Errorf("抓取地址错误：%s", err.Error())
		return
	}
	if s.IsSingle {
		if s.SingleName == "" {
			err = fmt.Errorf("单视频抓取请设置名称 [SingleName]")
			return
		}
		return s.startCrawl(u, basePath)
	}

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"),
		//colly.Async(true),
		colly.DetectCharset(),
		colly.AllowedDomains(u.Host),
		colly.MaxBodySize(50*1024*1024),
	)

	//_ = c.Limit(&colly.LimitRule{
	//	RandomDelay: time.Second,
	//	Parallelism: 10,
	//})

	// Find and visit all links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		e.Request.Visit(e.Attr("href"))
	})

	c.OnResponse(func(res *colly.Response) {
		// 视频详情地址
		currentUrl := res.Request.URL
		if strings.HasPrefix(currentUrl.String(), "") {
			resourceUrl, resourceName := "", ""
			resourceName = ByRegexp(string(res.Body))
			if resourceName == "" {
				s.logDriver.WarnLog(map[string]interface{}{
					"requestUrl": currentUrl.Path,
				}, "没有匹配到视频名称")
				return
			}
			resourceUrl, err = url.JoinPath(
				fmt.Sprintf("%s://%s", currentUrl.Scheme, currentUrl.Host),
				"api",
				currentUrl.Path,
			)
			if err != nil {
				s.logDriver.ErrLog(map[string]interface{}{
					"requestUrl":   currentUrl.Path,
					"resourceName": resourceName,
				}, "视频地址拼接失败："+utils.ParseError(err))
				return
			}

			err = (&resource{
				logDriver:    s.logDriver,
				resourceUrl:  resourceUrl,
				resourceName: resourceName,
				savePath:     path.Join(basePath, currentUrl.Path),
				wg:           &sync.WaitGroup{},
				keyPrefix:    "#EXT-X-KEY:",
				c:            resty.New(),
			}).spider()
			if err != nil {
				return
			}
		}
	})

	_ = c.Visit(s.sk.GetHost())

	c.Wait()

	return
}

func (s *Spider) startCrawl(currentUrl *url.URL, basePath string) (err error) {
	resourceUrl, err := url.JoinPath(
		fmt.Sprintf("%s://%s", currentUrl.Scheme, currentUrl.Host),
		"api",
		currentUrl.Path,
	)
	if err != nil {
		s.logDriver.ErrLog(map[string]interface{}{
			"requestUrl":   currentUrl.Path,
			"resourceName": s.SingleName,
		}, "视频地址拼接失败："+utils.ParseError(err))
		return
	}

	err = (&resource{
		logDriver:    s.logDriver,
		resourceUrl:  resourceUrl,
		resourceName: s.SingleName,
		savePath:     path.Join(basePath, currentUrl.Path),
		isSingle:     true,
		wg:           &sync.WaitGroup{},
		keyPrefix:    "#EXT-X-KEY:",
		c:            resty.New(),
	}).spider()

	return
}
