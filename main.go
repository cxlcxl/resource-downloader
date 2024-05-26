package main

import (
	"log"
	"path"
	"videocapture/server/drivers/r_video"
	"videocapture/server/spider"
	"videocapture/utils/clogs"
	"videocapture/vars"
)

func main() {
	crawlVideoSite()
}

func crawlVideoSite() {
	logDriver := clogs.NewCLog()
	err := spider.NewSpider(
		&r_video.R{
			Log:             logDriver,
			DownloadLogFile: path.Join(vars.BasePath, vars.Config.Video.SavePath, "r", "log.log"),
		},
		logDriver,
	).CrawlOne("单个url地址")
	log.Println(err)
}

func crawlROne() {
	logDriver := clogs.NewCLog()
	err := spider.NewSpider(
		&r_video.R{
			Log:             logDriver,
			DownloadLogFile: path.Join(vars.BasePath, vars.Config.Video.SavePath, "r", "log.log"),
		},
		logDriver,
	).CrawlOne("单个url地址")
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
