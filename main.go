package main

import (
	"log"
	"videocapture/server/drivers"
	"videocapture/server/spider"
	"videocapture/utils/clogs"
)

func main() {
	logDriver := clogs.NewCLog()
	err := spider.NewSpider(
		&drivers.Rou{Log: logDriver},
		logDriver,
		spider.SetAsync(20),
	).Start()
	log.Println(err)
}
