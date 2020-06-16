// commonutils
package utils

import (
	"fmt"
	"io/ioutil"

	"crypto/md5"
)

func MD5(pData []byte) string {
	md5 := md5.Sum(pData)
	return string(md5[:])
}

func ReadFileData(filePath string) ([]byte, error) {
	datas, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return datas, nil
}

func CalcFileMD5(filePath string) string {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return ""
	}
	md5 := md5.Sum(bytes)
	return fmt.Sprintf("%x", md5)
}
