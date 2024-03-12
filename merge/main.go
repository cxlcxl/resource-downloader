package merge

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	_ "videocapture/vars"
)

func Merge(mergePath string) (err error) {
	dir, err := os.ReadDir(mergePath)
	if err != nil {
		log.Fatal(err)
	}
	maxNum := 1
	for _, entry := range dir {
		if strings.Contains(entry.Name(), "mp4") {
			split := strings.Split(entry.Name(), ".")
			idx, _ := strconv.Atoi(split[0])
			if idx > maxNum {
				maxNum = idx
			}
		}
	}

	newFilename := path.Join(mergePath, "new.mp4")
	f, err := os.OpenFile(newFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		log.Println("新建文件失败", err)
		return
	}
	defer f.Close()

	for i := 1; i <= maxNum; i++ {
		filename := path.Join(mergePath, fmt.Sprintf("%d.mp4", i))
		var bytes []byte
		bytes, err = os.ReadFile(filename)
		if err != nil {
			log.Println("文件读取失败：", filename, err)
			return
		}
		_, err = f.Write(bytes)
		if err != nil {
			log.Println("文件写入失败：", filename, err)
			return
		}
	}
	return
}
