package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path"
	"path/filepath"
	"sync"
	collyserver "videocapture/server/colly-server"
	"videocapture/utils/clogs"
	"videocapture/vars"
	_ "videocapture/vars"
)

func main() {
	crawlByConfig()
}

func crawlByOne(videoName, videoHost string, wg *sync.WaitGroup) {
	defer wg.Done()

	cs := &collyserver.CollyServer{
		Host:       videoHost,
		LogDriver:  clogs.NewCLog(),
		IsSingle:   true,
		SingleName: videoName,
	}
	err := cs.Run(path.Join(vars.BasePath, vars.Config.Video.SavePath))
	fmt.Println("crawlByOne", videoName, err)
}

type VideoConfig struct {
	Videos []*Video `yaml:"videos"`
}

type Video struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

func crawlByConfig() {
	targetConfigPath := filepath.Join(vars.BasePath, "targets")
	dir, err := os.ReadDir(targetConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	wg := &sync.WaitGroup{}
	for _, entry := range dir {
		file, err := os.ReadFile(filepath.Join(targetConfigPath, entry.Name()))
		if err != nil {
			log.Println("os.ReadFile err", err)
			continue
		}
		var videoConfig VideoConfig
		err = yaml.Unmarshal(file, &videoConfig)
		if err != nil {
			log.Println("yaml.Unmarshal err", err)
			continue
		}
		for _, video := range videoConfig.Videos {
			wg.Add(1)
			go crawlByOne(video.Name, video.Url, wg)
		}
	}

	wg.Wait()
}
