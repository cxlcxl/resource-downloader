package r_video

import (
	"errors"
	"path"
	"videocapture/vars"
)

var (
	ConfigPath = path.Join(vars.BasePath, "config/driver_r.yaml")
)

var (
	ErrAddrRequest = errors.New("RV: 地址请求失败")
)

type VideoApiResponse struct {
	Video *Video
}

type Video struct {
	ThumbVTTUrl string `json:"thumbVTTUrl"`
	VideoUrl    string `json:"videoUrl"`
}
