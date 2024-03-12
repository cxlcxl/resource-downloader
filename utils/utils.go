package utils

import (
	"crypto/md5"
	"encoding/hex"
)

func MD5(params string) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(params))
	return hex.EncodeToString(md5Ctx.Sum(nil))
}

func ParseError(err error) string {
	if err != nil {
		return err.Error()
	} else {
		return ""
	}
}
