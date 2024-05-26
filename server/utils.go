package server

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io"
	"log"
	"os"
	"strings"
)

func ParseUriChart(s string) string {
	return strings.Replace(s, "%3F", "?", 1)
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

func LoadDownloaded(filepath string) (l map[string]bool) {
	l = make(map[string]bool)
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDONLY, 0777)
	if err != nil {
		log.Fatal("缓存文件访问失败", err)
		return
	}

	defer f.Close()

	br := bufio.NewReader(f)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		l[string(a)] = true
	}
	return
}

func RecordDownload(s, filepath string) {
	f, err := os.OpenFile(filepath, os.O_APPEND, 0777)
	if err != nil {
		log.Fatal("缓存文件访问失败", err)
		return
	}

	defer f.Close()

	f.WriteString(s)
	f.WriteString("\n")
}
