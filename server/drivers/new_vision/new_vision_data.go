package new_vision

import (
	"errors"
	"path"
	"videocapture/vars"
)

var (
	ConfigPath = path.Join(vars.BasePath, "config/driver_new_vision.yaml")
)

var (
	ErrAddrRequest = errors.New("NewVision: 地址请求失败")
)
