package main

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"log"
	"path"
	"reflect"
	"time"
	"videocapture/core/model"
	"videocapture/server/drivers/new_vision"
	"videocapture/server/drivers/r_video"
	"videocapture/server/spider"
	"videocapture/utils/clogs"
	"videocapture/vars"
)

var (
	spiders = map[string]spider.SpiderDriver{
		"new_vision": &new_vision.NewVision{Config: &new_vision.NvConfig{}},
	}
	refreshChan = make(chan bool)
)

type job struct {
	entryId cron.EntryID
	version uint64
}

func setOptions(driver spider.SpiderDriver, opts map[string]interface{}) {
	of := reflect.TypeOf(driver)
	for structName, opt := range opts {
		if _, ok := of.Elem().FieldByName(structName); ok {
			reflect.ValueOf(driver).Elem().FieldByName(structName).Set(reflect.ValueOf(opt))
		}
	}
}

func main() {
	db, err := model.NewDB()
	if err != nil {
		log.Fatalln("数据库连接失败：", err)
	}
	logDriver := clogs.NewCLog()
	sites := getSiteJobs(logDriver, db)
	c := cron.New()

	var s *spider.Spider
	entryIds := make(map[string]*job)
	for _, site := range sites {
		if driver, ok := spiders[site.SiteEnCode]; ok {
			setOptions(driver, map[string]interface{}{"DB": db, "Log": logDriver})

			s, err = spider.NewSpider(driver, logDriver)
			if err != nil {
				logDriver.ErrLog(map[string]interface{}{
					"SiteEnCode":   site.SiteEnCode,
					"SiteUrl":      site.SiteUrl,
					"ScheduleSpec": site.ScheduleSpec,
				}, fmt.Sprintf("爬虫启动失败：%s", err.Error()))
				continue
			}
			var entryId cron.EntryID
			entryId, err = c.AddJob(site.ScheduleSpec, s)
			if err != nil {
				logDriver.ErrLog(map[string]interface{}{
					"SiteEnCode":   site.SiteEnCode,
					"SiteUrl":      site.SiteUrl,
					"ScheduleSpec": site.ScheduleSpec,
				}, fmt.Sprintf("任务添加失败：%s", err.Error()))
			}
			entryIds[site.SiteEnCode] = &job{
				entryId: entryId,
				version: site.Version,
			}
		}
	}
	go func() {
		time.Sleep(time.Second * 300)
		refreshChan <- true
	}()
	go refreshSchedule(c, entryIds, logDriver, db)

	c.Run()
	select {}
}

func crawlROne() {
	logDriver := clogs.NewCLog()
	s, err := spider.NewSpider(
		&r_video.R{
			Log:             logDriver,
			DownloadLogFile: path.Join(vars.BasePath, vars.Config.Video.SavePath, "r", "log.log"),
		},
		logDriver,
	)
	if err != nil {
		log.Fatal(err)
	}
	err = s.CrawlOne("单个url地址")
	log.Println(err)

	// logDriver := clogs.NewCLog()
	// err := spider.NewSpider(
	// 	&drivers.Rou{
	// 		Log:             logDriver,
	// 		DownloadLogFile: path.Join(vars.BasePath, vars.Config.Video.SavePath, "rou", "log.log"),
	// 	},
	// 	logDriver,
	// 	spider.SetAsync(20),
	// ).Start()
	// log.Println(err)
}

func getSiteJobs(l *clogs.CLog, db *gorm.DB) []*model.Site {
	sites, err := model.NewSite().FindSiteJobs(db)
	if err != nil {
		l.ErrLog(map[string]interface{}{}, fmt.Sprintf("任务查询失败：%s", err.Error()))
		sites = make([]*model.Site, 0)
	}
	return sites
}

func refreshSchedule(c *cron.Cron, entryIds map[string]*job, l *clogs.CLog, db *gorm.DB) {
	for i := range refreshChan {
		if i {
			siteJob := make(map[string]*job)
			for _, site := range getSiteJobs(l, db) {
				siteJob[site.SiteEnCode] = &job{
					entryId: 0,
					version: site.Version,
				}
			}
			for code, _job := range entryIds {
				if s, ok := siteJob[code]; !ok {
					// 没有 - 已经被删了
					c.Remove(_job.entryId)
				} else {
					if _job.version != s.version {
						c.Remove(_job.entryId)
						//entryIds[code], _ = c.AddJob()
					}
				}
			}
		}
	}
}
