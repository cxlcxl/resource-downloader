package merror

import "errors"

type MyError error

var (
	ParamErrorUrl MyError = errors.New("抓取的 URL 地址有误")
)
