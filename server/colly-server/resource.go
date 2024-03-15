package collyserver

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"videocapture/merge"
	"videocapture/utils"
	"videocapture/utils/clogs"

	"github.com/go-resty/resty/v2"
	ants "github.com/panjf2000/ants/v2"
)

type resource struct {
	logDriver    clogs.LogInterface
	resourceUrl  string
	resourceName string
	savePath     string
	video        *Video
	isSingle     bool
	wg           *sync.WaitGroup
	keyPrefix    string
	method       string
	keyUri       string
	key          []byte
	iv           []byte // 0x26ee0b9b8f4aec3b1ebf867f263a6063
	isFetchKey   bool
	c            *resty.Client
}

type VideoApiResponse struct {
	Video *Video
}

type Video struct {
	ThumbVTTUrl string `json:"thumbVTTUrl"`
	VideoUrl    string `json:"videoUrl"`
}

func (res *resource) spider() (err error) {
	resp, err := res.c.R().EnableTrace().Get(res.resourceUrl)
	if err != nil || resp.StatusCode() != 200 {
		res.logDriver.ErrLog(map[string]interface{}{
			"url":        res.resourceUrl,
			"StatusCode": resp.StatusCode(),
		}, "地址请求失败 [spider]: "+utils.ParseError(err))
		return
	}

	var videoResponse VideoApiResponse
	err = json.Unmarshal(resp.Body(), &videoResponse)
	if err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"url": res.resourceUrl,
		}, "Json 解码失败: "+utils.ParseError(err))
		return
	}

	res.video = videoResponse.Video
	return res.spiderM3u8()
}

func (res *resource) spiderM3u8() (err error) {
	resp, err := res.c.R().EnableTrace().Get(res.video.VideoUrl)
	if err != nil || resp.StatusCode() != 200 {
		res.logDriver.ErrLog(map[string]interface{}{
			"url":        res.video.VideoUrl,
			"StatusCode": resp.StatusCode(),
		}, "地址请求失败 [spiderM3u8]: "+utils.ParseError(err))
		return
	}
	videoM3u8 := resp.String()
	videoM3u8s := strings.Split(strings.ReplaceAll(videoM3u8, "\r\n", "\n"), "\n")

	videoUrlParse, _ := url.Parse(res.video.VideoUrl)
	for _, m3u8 := range videoM3u8s {
		if strings.HasPrefix(m3u8, "#") {
			continue
		} else {
			u, err := url.Parse(m3u8)
			if err != nil {
				res.logDriver.ErrLog(map[string]interface{}{
					"url":        res.video.VideoUrl,
					"StatusCode": resp.StatusCode(),
				}, "地址解析失败 [spiderM3u8]: "+utils.ParseError(err))
				continue
			}
			m3u8Url := videoUrlParse.ResolveReference(u)
			res.spiderM3u8Video(m3u8Url.String())
			break
		}
	}

	return
}

func (res *resource) spiderM3u8Video(m3u8Url string) {
	resp, err := res.c.R().EnableTrace().Get(m3u8Url)
	if err != nil {
		return
	}
	if err != nil || resp.StatusCode() != 200 {
		res.logDriver.ErrLog(map[string]interface{}{
			"url":        m3u8Url,
			"StatusCode": resp.StatusCode(),
		}, "地址请求失败 [spiderM3u8Video]: "+utils.ParseError(err))
		return
	}

	videoM3u8 := resp.String()
	videoM3u8s := strings.Split(strings.ReplaceAll(videoM3u8, "\r\n", "\n"), "\n")

	if err = res.checkDir(); err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"path": res.savePath,
		}, "路径检查失败 [spiderM3u8Video]: "+utils.ParseError(err))
		return
	}
	go res.writeM3u8File(resp.Body())

	defer ants.Release()

	videoIdx := 0
	for _, m3u8 := range videoM3u8s {
		if strings.HasPrefix(m3u8, "#") {
			if err = res.readEncryptionKey(m3u8); err != nil {
				break
			}
			continue
		} else {
			if !res.isFetchKey {
				if err = res.fetchKey(m3u8); err != nil {
					res.logDriver.ErrLog(map[string]interface{}{
						"url": m3u8,
					}, "key 获取失败 [fetchKey]: "+utils.ParseError(err))
					break
				}
			}

			res.wg.Add(1)
			videoIdx++
			go res.spiderPart(videoIdx, parseUriChart(m3u8), 0, nil)
		}
	}

	res.wg.Wait()
	// 下载完成

	if err = merge.Merge(res.savePath); err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"path": res.savePath,
		}, "路径检查失败 [spiderPart]: "+utils.ParseError(err))
		return
	}

	return
}

func (res *resource) readEncryptionKey(m3u8 string) (err error) {
	if strings.HasPrefix(m3u8, res.keyPrefix) {
		keys := strings.Split(m3u8[len(res.keyPrefix):], ",")
		for _, key := range keys {
			if strings.HasPrefix(key, "METHOD=") {
				res.method = key[len("METHOD="):]
			}
			if strings.HasPrefix(key, "URI=") {
				res.keyUri = key[len("URI=")+1 : len(key)-1]
			}
			if strings.HasPrefix(key, "IV=") {
				res.iv, err = hex.DecodeString(key[len("IV=")+2:])
				if err != nil {
					res.logDriver.ErrLog(map[string]interface{}{
						"key": key[len("IV="):],
					}, "IV 获取失败 [spiderM3u8Video]: "+utils.ParseError(err))
					break
				}
			}
		}
	}
	return
}

func (res *resource) spiderPart(idx int, u string, retryTimes int, err error) {
	if retryTimes > 100 {
		res.logDriver.ErrLog(map[string]interface{}{
			"url": u,
		}, "片段下载失败 [spiderPart]: "+utils.ParseError(err))
		res.wg.Done()
		return
	}

	resp, err := res.c.R().EnableTrace().Get(u)
	if err != nil || resp.StatusCode() != 200 {
		time.Sleep(time.Millisecond * 500)
		res.spiderPart(idx, u, retryTimes+1, err)
		return
	}

	filename := path.Join(res.savePath, strconv.Itoa(idx)+".mp4")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		time.Sleep(time.Millisecond * 500)
		res.spiderPart(idx, u, retryTimes+1, err)
		return
	}
	defer f.Close()

	var entryBytes []byte
	if len(res.key) > 0 {
		entryBytes, err = AesDecrypt(resp.Body(), res.key, res.iv)
		if err != nil {
			res.logDriver.ErrLog(map[string]interface{}{
				"url":   u,
				"index": idx,
			}, "AesDecrypt 失败 [spiderPart]: "+utils.ParseError(err))
			res.wg.Done()
			return
		}
	} else {
		entryBytes = resp.Body()
	}
	_, err = f.Write(entryBytes)
	if err != nil {
		time.Sleep(time.Millisecond * 500)
		res.spiderPart(idx, u, retryTimes+1, err)
		return
	}

	res.logDriver.InfoLog(map[string]interface{}{"filename": filename}, "Success")
	res.wg.Done()
}

func (res *resource) fetchKey(m3u8Url string) (err error) {
	if len(res.key) == 0 && res.keyUri == "" {
		res.isFetchKey = true
		return
	}
	u, _ := url.Parse(m3u8Url)
	split := strings.Split(u.Path, "/")
	host := fmt.Sprintf("%s://%s/%s/%s", u.Scheme, u.Host, strings.Join(split[0:len(split)-1], "/"), res.keyUri)

	resp, err := res.c.R().EnableTrace().Get(host)
	if err != nil || resp.StatusCode() != 200 {
		err = fmt.Errorf("key 获取失败 [fetchKey]: " + utils.ParseError(err))
		return
	}

	res.key = resp.Body()
	res.isFetchKey = true
	return
}

func (res *resource) checkDir() (err error) {
	if _, err = os.Stat(res.savePath); err != nil {
		for i := 0; i < 3; i++ {
			err = os.MkdirAll(res.savePath, 0777)
			if err == nil {
				break
			}
		}
	}

	go res.writeName()
	return
}

func (res *resource) writeName() {
	filename := path.Join(res.savePath, "filename.txt")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "文件名创建失败 [writeName]: "+utils.ParseError(err))
		return
	}
	defer f.Close()

	_, err = f.WriteString(res.resourceName)
	if err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "文件名写入失败 [writeName]: "+utils.ParseError(err))
		return
	}
}

func (res *resource) writeM3u8File(bytes []byte) {
	filename := path.Join(res.savePath, "m3u8.txt")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "m3u8文件创建失败 [writeM3u8File]: "+utils.ParseError(err))
		return
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		res.logDriver.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "文件名写入失败 [writeM3u8File]: "+utils.ParseError(err))
		return
	}
}
