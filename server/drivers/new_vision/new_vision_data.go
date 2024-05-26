package new_vision

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"path"
	"videocapture/server/spider"
	"videocapture/vars"
)

var (
	ConfigPath = path.Join(vars.BasePath, "config/driver_new_vision.yaml")
)

const (
	VideoColumnVideoName    = "video_name"
	VideoColumnVideoDesc    = "video_desc"
	VideoColumnCoverImg     = "cover_img"
	VideoColumnDirector     = "director"
	VideoColumnActor        = "actor"
	VideoColumnOnline       = "online"
	VideoColumnEpisodes     = "episodes"
	VideoColumnEpisodesList = "episodes_list"

	PageMain spider.SpiderType = iota
	PageVideoInfo
	PageVideoUrl
)

const (
	AttrTypeText = iota
	AttrTypeAttr
)

type NvConfig struct {
	Host string `yaml:"host"`
}

var (
	ErrAddrRequest  = errors.New("NewVision: 地址请求失败")
	PageTypeColumns = map[spider.SpiderType][]string{
		PageMain: {},
		PageVideoInfo: {
			VideoColumnVideoName,
			VideoColumnVideoDesc,
			VideoColumnCoverImg,
			VideoColumnDirector,
			VideoColumnActor,
			VideoColumnOnline,
			VideoColumnEpisodes,
			VideoColumnEpisodesList,
		},
		PageVideoUrl: {},
	}
	VideoColumnConfigs = map[string]*ColumnSpiderConfig{
		VideoColumnVideoName: {
			Selector:   "#main > div > div.box.view-heading > div.video-info > div.video-info-header > h1",
			ResultType: "normal",
			DataMap:    map[int]string{AttrTypeText: ""},
			Desc:       "电影名称",
		},
		VideoColumnVideoDesc: {
			Selector:   "#main > div > div.box.view-heading > div.video-info > div.video-info-main > div:nth-child(6) > div > span",
			ResultType: "normal",
			DataMap:    map[int]string{AttrTypeText: ""},
			Desc:       "剧情、历史",
		},
		VideoColumnCoverImg: {
			Selector:   "#main > div > div.box.view-heading > div.video-cover > div > div.module-item-pic > img",
			ResultType: "normal",
			DataMap:    map[int]string{AttrTypeAttr: "data-src"},
			Desc:       "封面图",
		},
		VideoColumnDirector: {
			Selector:   "#main > div > div.box.view-heading > div.video-info > div.video-info-main > div:nth-child(1) > div > a",
			ResultType: "list",
			DataMap:    map[int]string{AttrTypeText: ""},
			Desc:       "导演",
		},
		VideoColumnActor: {
			Selector:   "#main > div > div.box.view-heading > div.video-info > div.video-info-main > div:nth-child(2) > div > a",
			ResultType: "list",
			DataMap:    map[int]string{AttrTypeText: ""},
			Desc:       "演员表",
		},
		VideoColumnOnline: {
			Selector:   "#main > div > div.box.view-heading > div.video-info > div.video-info-main > div:nth-child(3) > div",
			ResultType: "normal",
			DataMap:    map[int]string{AttrTypeText: ""},
			Desc:       "上映时间",
		},
		VideoColumnEpisodes: {
			Selector:   "#main > div > div.box.view-heading > div.video-info > div.video-info-main > div:nth-child(4) > div",
			ResultType: "normal",
			DataMap:    map[int]string{AttrTypeText: ""},
			Desc:       "电视剧的集数",
		},
		VideoColumnEpisodesList: {
			Selector:   "#glist-1 > div.module-blocklist.scroll-box.scroll-box-y > div > a",
			ResultType: "list",
			DataMap:    map[int]string{AttrTypeText: "", AttrTypeAttr: "href"},
			Desc:       "电视剧的集数播放列表",
		},
	}
)

type ColumnSpiderConfig struct {
	Selector   string
	ResultType string
	DataMap    map[int]string
	Desc       string
}

type Dom struct {
	DomKey string
	DomVal string
	Attrs  map[string]string
	Sort   int
}

type DomOpt func(*goquery.Document) []*Dom
