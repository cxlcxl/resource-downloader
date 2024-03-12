package collyserver

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
	"videocapture/utils/clogs"
)

type CollyServer struct {
	Host       string
	LogDriver  clogs.LogInterface
	IsSingle   bool
	SingleName string
}

func parseUriChart(s string) string {
	return strings.Replace(s, "%3F", "?", 1)
}

// 解密ts文件内容，并写入到输出文件
func decodeAES128CBC(key []byte, index int, inFile string, out io.Writer) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	inBuf, err := os.ReadFile(inFile)
	if err != nil || len(inBuf) == 0 {
		return err
	}

	var iv [16]byte
	binary.BigEndian.PutUint32(iv[12:], uint32(index))

	outBuf := make([]byte, len(inBuf))
	mode := cipher.NewCBCDecrypter(block, iv[:])
	mode.CryptBlocks(outBuf, inBuf)

	pad := int(outBuf[len(outBuf)-1])
	_, err = out.Write(outBuf[:len(outBuf)-pad])
	return err
}

// AesDecrypt 解密
func AesDecrypt(data []byte, key, iv []byte) ([]byte, error) {
	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, iv)
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))
	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}
