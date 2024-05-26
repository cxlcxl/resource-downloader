package server

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	_ "videocapture/vars"
)

func Merge(mergePath string, completeWithDel bool) (err error) {
	dir, err := os.ReadDir(mergePath)
	if err != nil {
		return fmt.Errorf("文件夹读取失败: %s", err.Error())
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
		return fmt.Errorf("新建文件失败: %s", err.Error())
	}
	defer f.Close()

	filenames := make([]string, 0)
	for i := 1; i <= maxNum; i++ {
		filename := path.Join(mergePath, fmt.Sprintf("%d.mp4", i))
		var bytes []byte
		bytes, err = os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("文件读取失败: %s", err.Error())
		}
		_, err = f.Write(bytes)
		if err != nil {
			return fmt.Errorf("文件写入失败: %s", err.Error())
		}
		filenames = append(filenames, filename)
	}

	if completeWithDel {
		releaseDir(filenames)
	}
	return
}

func releaseDir(filenames []string) {
	for _, val := range filenames {
		os.Remove(val)
	}
}

func TidyDir(tidyPath string) (err error) {
	dir, err := os.ReadDir(tidyPath)
	if err != nil {
		return fmt.Errorf("文件夹读取失败: %s", err.Error())
	}

	for _, entry := range dir {
		filename := path.Join(tidyPath, entry.Name(), "new.mp4")
		if _, err = os.Stat(filename); err == nil {
			if err = copyFile(filename, path.Join(tidyPath, entry.Name()+".mp4")); err == nil {
				_ = os.Remove(filename)
			}
		} else {
			fmt.Println("os.Stat:", filename, err)
		}
	}
	return
}

func copyFile(source, dest string) (err error) {
	fmt.Println("copy:", source, dest)
	sourceFile, err := os.Open(source)
	if err != nil {
		return
	}
	defer sourceFile.Close()

	create, err := os.Create(dest)
	if err != nil {
		return
	}
	defer create.Close()

	fmt.Println("copy:", source, dest)
	_, err = io.Copy(create, sourceFile)
	if err != nil {
		fmt.Println("io.Copy", err)
	}

	return
}
