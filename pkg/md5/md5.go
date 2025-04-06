package md5

import (
	"crypto/md5"
	"encoding/hex"
)

// MockSum 测试用例变量
var MockSum func([]byte) (string, error)

// Sum 对传入参数求MD5值
func Sum(data []byte) (string, error) {
	if MockSum != nil {
		return MockSum(data)
	}

	h := md5.New()

	_, err := h.Write(data)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
