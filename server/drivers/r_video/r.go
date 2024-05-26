package r_video

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"videocapture/server"
	"videocapture/server/drivers"
	"videocapture/server/spider"
	"videocapture/utils"
	"videocapture/utils/clogs"
	"videocapture/vars"

	"github.com/go-resty/resty/v2"
	"github.com/panjf2000/ants/v2"
)

func rTitleByRegexp(body string) (title string) {
	reg := regexp.MustCompile(`<div class="text-base md:text-xl mb-1">(.*?)</div>`)
	params := reg.FindStringSubmatch(body)
	if len(params) != 2 {
		return ""
	}
	return params[1]
}

type R struct {
	Log             clogs.LogInterface
	DownloadLogFile string
	downloadLogs    map[string]bool
}

type RConfig struct {
	Host                string `yaml:"host"`
	ResourceUrlContains string `yaml:"resource_url_contains"`
}

func (r *R) GetHost(c interface{}) string {
	return c.(*RConfig).Host
}
func (r *R) SetDB(db *gorm.DB) {

}
func (r *R) GetConfig() (interface{}, string) {
	r.downloadLogs = server.LoadDownloaded(r.DownloadLogFile)
	return &RConfig{}, ConfigPath
}

func (r *R) ParseResource(u *url.URL, body []byte, c interface{}) (rs *spider.Resource, ok bool, err error) {
	if strings.Contains(u.String(), c.(*RConfig).ResourceUrlContains) {
		resourceUrl, title := "", ""
		title = rTitleByRegexp(string(body))
		if title == "" {
			r.Log.WarnLog(map[string]interface{}{
				"requestUrl": u.Path,
			}, "没有匹配到视频名称")
			return
		}
		resourceUrl, err = url.JoinPath(
			fmt.Sprintf("%s://%s", u.Scheme, u.Host),
			"api",
			u.Path,
		)
		if err != nil {
			r.Log.ErrLog(map[string]interface{}{
				"requestUrl": u.Path,
				"title":      title,
			}, "视频地址拼接失败："+utils.ParseError(err))
			return
		}

		return &spider.Resource{
			ResourceUrl: resourceUrl,
			Title:       title,
			SavePath:    path.Join(vars.BasePath, vars.Config.Video.SavePath, "r"),
		}, true, nil
	}
	return nil, false, nil
}

func (r *R) IsRequest(host string) bool {
	u, _ := url.Parse(host)
	split := strings.Split(u.Path, "/")
	if _, ok := r.downloadLogs[split[len(split)-1]]; ok {
		return false
	}
	if ok, _ := regexp.MatchString(`^(\/v\/)[a-z0-9]+$`, host); ok {
		return true
	}
	return false
}

func (r *R) log(videoKey string) {
	server.RecordDownload(videoKey, r.DownloadLogFile)
}

type rSpider struct {
	*resty.Client
	ld         clogs.LogInterface
	video      *Video
	title      string
	savePath   string
	keyPrefix  string
	isFetchKey bool
	wg         *sync.WaitGroup
	method     string
	keyUri     string
	key        []byte
	iv         []byte
}

func (r *R) Crawl(res *spider.Resource, wg *sync.WaitGroup, ld clogs.LogInterface) error {
	defer wg.Done()

	u, _ := url.Parse(res.ResourceUrl)
	paths := strings.Split(u.Path, "/")
	rs := &rSpider{
		Client:   resty.New(),
		ld:       ld,
		savePath: path.Join(res.SavePath, paths[len(paths)-1]),
		title:    res.Title,
		wg:       &sync.WaitGroup{},
	}
	defer r.log(paths[len(paths)-1])

	resp, err := rs.R().EnableTrace().Get(res.ResourceUrl)
	if err != nil || resp.StatusCode() != 200 {
		rs.ld.ErrLog(map[string]interface{}{
			"url":        res.ResourceUrl,
			"StatusCode": resp.StatusCode(),
		}, "地址请求失败 [spider]: "+utils.ParseError(err))
		return ErrAddrRequest
	}

	var videoResponse VideoApiResponse
	err = json.Unmarshal(resp.Body(), &videoResponse)
	if err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"url": res.ResourceUrl,
		}, "Json 解码失败: "+utils.ParseError(err))
		return drivers.ErrJsonUnmarshalFail
	}

	rs.video = videoResponse.Video
	return rs.spiderM3u8()
}

func (rs *rSpider) spiderM3u8() (err error) {
	resp, err := rs.R().EnableTrace().Get(rs.video.VideoUrl)
	if err != nil || resp.StatusCode() != 200 {
		rs.ld.ErrLog(map[string]interface{}{
			"url":        rs.video.VideoUrl,
			"StatusCode": resp.StatusCode(),
		}, "地址请求失败 [spiderM3u8]: "+utils.ParseError(err))
		return
	}
	videoM3u8 := resp.String()
	videoM3u8s := strings.Split(strings.ReplaceAll(videoM3u8, "\r\n", "\n"), "\n")

	videoUrlParse, _ := url.Parse(rs.video.VideoUrl)
	for _, m3u8 := range videoM3u8s {
		if strings.HasPrefix(m3u8, "#") {
			continue
		} else {
			u, err := url.Parse(m3u8)
			if err != nil {
				rs.ld.ErrLog(map[string]interface{}{
					"url":        rs.video.VideoUrl,
					"StatusCode": resp.StatusCode(),
				}, "地址解析失败 [spiderM3u8]: "+utils.ParseError(err))
				continue
			}
			m3u8Url := videoUrlParse.ResolveReference(u)
			rs.spiderM3u8Video(m3u8Url.String())
			break
		}
	}

	return
}

func (rs *rSpider) spiderM3u8Video(m3u8Url string) {
	resp, err := rs.R().EnableTrace().Get(m3u8Url)
	if err != nil {
		return
	}
	if err != nil || resp.StatusCode() != 200 {
		rs.ld.ErrLog(map[string]interface{}{
			"url":        m3u8Url,
			"StatusCode": resp.StatusCode(),
		}, "地址请求失败 [spiderM3u8Video]: "+utils.ParseError(err))
		return
	}

	videoM3u8 := resp.String()
	videoM3u8s := strings.Split(strings.ReplaceAll(videoM3u8, "\r\n", "\n"), "\n")

	if err = rs.checkDir(); err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"path": rs.savePath,
		}, "路径检查失败 [spiderM3u8Video]: "+utils.ParseError(err))
		return
	}
	go rs.writeM3u8File(resp.Body())

	defer ants.Release()

	videoIdx := 0
	for _, m3u8 := range videoM3u8s {
		if strings.HasPrefix(m3u8, "#") {
			if err = rs.readEncryptionKey(m3u8); err != nil {
				break
			}
			continue
		} else {
			if !rs.isFetchKey {
				if err = rs.fetchKey(m3u8); err != nil {
					rs.ld.ErrLog(map[string]interface{}{
						"url": m3u8,
					}, "key 获取失败 [fetchKey]: "+utils.ParseError(err))
					break
				}
			}

			rs.wg.Add(1)
			videoIdx++
			go rs.spiderPart(videoIdx, server.ParseUriChart(m3u8), 0, nil)
		}
	}

	rs.wg.Wait()
	// 下载完成

	if err = server.Merge(rs.savePath, true); err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"path": rs.savePath,
		}, "文件合并失败 [spiderPart.Merge]: "+utils.ParseError(err))
		return
	}

	if err = server.TidyDir(rs.savePath); err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"path": rs.savePath,
		}, "文件整理失败 [spiderPart.TidyDir]: "+utils.ParseError(err))
		return
	}

	return
}

func (rs *rSpider) readEncryptionKey(m3u8 string) (err error) {
	if strings.HasPrefix(m3u8, rs.keyPrefix) {
		keys := strings.Split(m3u8[len(rs.keyPrefix):], ",")
		for _, key := range keys {
			if strings.HasPrefix(key, "METHOD=") {
				rs.method = key[len("METHOD="):]
			}
			if strings.HasPrefix(key, "URI=") {
				rs.keyUri = key[len("URI=")+1 : len(key)-1]
			}
			if strings.HasPrefix(key, "IV=") {
				rs.iv, err = hex.DecodeString(key[len("IV=")+2:])
				if err != nil {
					rs.ld.ErrLog(map[string]interface{}{
						"key": key[len("IV="):],
					}, "IV 获取失败 [spiderM3u8Video]: "+utils.ParseError(err))
					break
				}
			}
		}
	}
	return
}

func (rs *rSpider) spiderPart(idx int, u string, retryTimes int, err error) {
	if retryTimes > 100 {
		rs.ld.ErrLog(map[string]interface{}{
			"url": u,
		}, "片段下载失败 [spiderPart]: "+utils.ParseError(err))
		rs.wg.Done()
		return
	}

	resp, err := rs.R().EnableTrace().Get(u)
	if err != nil || resp.StatusCode() != 200 {
		time.Sleep(time.Millisecond * 500)
		rs.spiderPart(idx, u, retryTimes+1, err)
		return
	}

	filename := path.Join(rs.savePath, strconv.Itoa(idx)+".mp4")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		time.Sleep(time.Millisecond * 500)
		rs.spiderPart(idx, u, retryTimes+1, err)
		return
	}
	defer f.Close()

	var entryBytes []byte
	if len(rs.key) > 0 {
		entryBytes, err = server.AesDecrypt(resp.Body(), rs.key, rs.iv)
		if err != nil {
			rs.ld.ErrLog(map[string]interface{}{
				"url":   u,
				"index": idx,
			}, "AesDecrypt 失败 [spiderPart]: "+utils.ParseError(err))
			rs.wg.Done()
			return
		}
	} else {
		entryBytes = resp.Body()
	}
	_, err = f.Write(entryBytes)
	if err != nil {
		time.Sleep(time.Millisecond * 500)
		rs.spiderPart(idx, u, retryTimes+1, err)
		return
	}

	rs.ld.InfoLog(map[string]interface{}{"filename": filename}, "Success")
	rs.wg.Done()
}

func (rs *rSpider) fetchKey(m3u8Url string) (err error) {
	if len(rs.key) == 0 && rs.keyUri == "" {
		rs.isFetchKey = true
		return
	}
	u, _ := url.Parse(m3u8Url)
	split := strings.Split(u.Path, "/")
	host := fmt.Sprintf("%s://%s/%s/%s", u.Scheme, u.Host, strings.Join(split[0:len(split)-1], "/"), rs.keyUri)

	resp, err := rs.R().EnableTrace().Get(host)
	if err != nil || resp.StatusCode() != 200 {
		err = fmt.Errorf("key 获取失败 [fetchKey]: " + utils.ParseError(err))
		return
	}

	rs.key = resp.Body()
	rs.isFetchKey = true
	return
}

func (rs *rSpider) checkDir() (err error) {
	if _, err = os.Stat(rs.savePath); err != nil {
		for i := 0; i < 3; i++ {
			err = os.MkdirAll(rs.savePath, 0777)
			if err == nil {
				break
			}
		}
	}

	go rs.writeName()
	return
}

func (rs *rSpider) writeName() {
	filename := path.Join(rs.savePath, "filename.txt")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "文件名创建失败 [writeName]: "+utils.ParseError(err))
		return
	}
	defer f.Close()

	_, err = f.WriteString(rs.title)
	if err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "文件名写入失败 [writeName]: "+utils.ParseError(err))
		return
	}
}

func (rs *rSpider) writeM3u8File(bytes []byte) {
	filename := path.Join(rs.savePath, "m3u8.txt")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "m3u8文件创建失败 [writeM3u8File]: "+utils.ParseError(err))
		return
	}
	defer f.Close()

	_, err = f.Write(bytes)
	if err != nil {
		rs.ld.ErrLog(map[string]interface{}{
			"filename": filename,
		}, "文件名写入失败 [writeM3u8File]: "+utils.ParseError(err))
		return
	}
}
