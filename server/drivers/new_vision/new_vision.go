package new_vision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"gorm.io/gorm"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"videocapture/core/model"
	"videocapture/server/spider"
	"videocapture/utils"
	"videocapture/utils/clogs"
)

type NewVision struct {
	Log             clogs.LogInterface
	DownloadLogFile string
	downloadLogs    map[string]bool
	db              *gorm.DB
}

func (nv *NewVision) GetHost(c interface{}) string {
	return c.(*NvConfig).Host
}

func (nv *NewVision) GetConfig() (interface{}, string) {
	//nv.downloadLogs = server.LoadDownloaded(nv.DownloadLogFile)
	return &NvConfig{}, ConfigPath
}
func (nv *NewVision) SetDB(db *gorm.DB) {
	nv.db = db
}

func (nv *NewVision) IsRequest(host string) bool {
	logKey := utils.MD5(host)
	if _, ok := nv.downloadLogs[logKey]; ok {
		return false
	}
	ok, _ := nvSpiderPages(host)
	return ok
}

func nvSpiderPages(host string) (bool, spider.SpiderType) {
	//if ok, _ := regexp.MatchString(`^(\/vplay\/)([a-z0-9\-]+)(\.html)$`, host); ok {
	//	return true, PageVideoUrl
	//}
	if ok, _ := regexp.MatchString(`^(\/video\/)([a-z0-9]+)(\.html)$`, host); ok {
		return true, PageVideoInfo
	}
	return false, 0
}

func (nv *NewVision) ParseResource(u *url.URL, body []byte, c interface{}) (*spider.Resource, bool, error) {
	if ok, pt := nvSpiderPages(u.Path); ok {
		fmt.Println("正在抓取地址：", u.String())
		r, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, false, err
		}
		// 根据不同的页面返回要抓取的不同元素
		documents, err := nv.crawlDocuments(r, GenerateOpts(pt)...)
		if err != nil {
			return nil, false, err
		}
		saveVideoInfo(nv.db, documents, u.String())
		fmt.Println("抓取完成：", u.String())
	}
	return nil, false, nil
}

func (nv *NewVision) crawlDocuments(r *goquery.Document, opts ...DomOpt) (documents []*Dom, err error) {
	documents = make([]*Dom, 0)
	for _, opt := range opts {
		documents = append(documents, opt(r)...)
	}
	return
}

func (nv *NewVision) Crawl(*spider.Resource, *sync.WaitGroup, clogs.LogInterface) error {
	return nil
}

func saveVideoInfo(db *gorm.DB, documents []*Dom, uri string) {
	video := &model.Video{
		State:       1,
		FromSiteUrl: uri,
		Timestamp:   model.Timestamp{},
	}
	exts := make([]*model.VideoExt, 0)
	actors := make([]string, 0)
	directors := make([]string, 0)
	for _, document := range documents {
		switch document.DomKey {
		case VideoColumnVideoName:
			video.VideoName = document.DomVal
			video.VideoId = utils.MD5(document.DomVal)
		case VideoColumnActor:
			actors = append(actors, document.DomVal)
		case VideoColumnDirector:
			directors = append(directors, document.DomVal)
		case VideoColumnEpisodes:
			video.Episodes = document.DomVal
		case VideoColumnCoverImg:
			for _, v := range document.Attrs {
				if v != "" {
					video.CoverImg = v
					break
				}
			}
		case VideoColumnVideoDesc:
			video.VideoDesc = document.DomVal
		case VideoColumnOnline:
			video.OnlineDate = document.DomVal
		case VideoColumnEpisodesList:
			exts = append(exts, &model.VideoExt{
				VideoId:   "",
				ExtKey:    VideoColumnEpisodesList,
				ExtVal:    document.DomVal,
				ExtDetail: jsonEncode(document.Attrs),
				State:     0,
			})
		default:
			continue
		}
	}

	if video.VideoId == "" {
		return
	}
	video.Actors = strings.Join(actors, ",")
	video.Directors = strings.Join(directors, ",")
	for i := range exts {
		exts[i].VideoId = video.VideoId
	}
	_ = model.NewVideo().CreateVideo(
		db,
		video,
		exts,
	)
}

func jsonEncode(d interface{}) string {
	marshal, _ := json.Marshal(d)
	return string(marshal)
}
