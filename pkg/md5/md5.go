package md5

import (
	"crypto/md5"
	"encoding/hex"
)

// Sum 对传入参数求MD5值
func Sum(data []byte) (string, error) {
	h := md5.New()

	_, err := h.Write(data)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
