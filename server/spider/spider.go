package spider

import (
	"path"
	"videocapture/utils/clogs"
	"videocapture/vars"
)

type SpiderKind interface {
	MultiCrawl() error
	CrawlOne(resourceName string, logDriver *clogs.CLog) error
}

type Spider struct {
	logDriver *clogs.CLog
	sk        SpiderKind
	bathPath  string
}

func NewSpider(sk SpiderKind) *Spider {
	return &Spider{
		logDriver: clogs.NewCLog(),
		sk:        sk,
		bathPath:  path.Join(vars.BasePath, vars.Config.Video.SavePath),
	}
}

func (s *Spider) CrawlOne() error {
	return s.sk.CrawlOne("", s.logDriver)
}
