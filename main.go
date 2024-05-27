package main

import (
	"log"
	"path"
	"videocapture/core/model"
	"videocapture/server/drivers/new_vision"
	"videocapture/server/drivers/r_video"
	"videocapture/server/spider"
	"videocapture/utils/clogs"
	"videocapture/vars"
)

func main() {
	logDriver := clogs.NewCLog()
	db, err := model.NewDB()
	if err != nil {
		log.Fatalln("数据库连接失败：", err)
	}
	s, err := spider.NewSpider(
		&new_vision.NewVision{
			Log:    logDriver,
			DB:     db,
			Config: &new_vision.NvConfig{},
		},
		logDriver,
	)
	if err != nil {
		log.Println(err)
	}
	err = s.Start()
	log.Println(err)
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
