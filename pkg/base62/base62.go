package base62

import (
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"sync"
)

const defaultBase62Str = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const base62EnvKey = "BASE62STR"

var (
	base62Str string
	once      sync.Once
)

func Convert(number uint64) string {
	once.Do(initBase62Str)

	if number == 0 {
		return string(base62Str[0])
	}

	var result []byte
	for number > 0 {
		result = append(result, base62Str[number%62])
		number /= 62
	}

	// 反转结果
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

func initBase62Str() {
	base62Str = os.Getenv(base62EnvKey)
	if base62Str == "" {
		base62Str = defaultBase62Str
	}

	if len(base62Str) != 62 {
		logx.Severef("BASE62STR must contain exactly 62 characters")
	}

	if hasDuplicateChars(base62Str) {
		logx.Severef("BASE62STR contains duplicate characters")
	}
}

func hasDuplicateChars(s string) bool {
	seen := make(map[rune]struct{}, 62)
	for _, c := range s {
		if _, exists := seen[c]; exists {
			return true
		}
		seen[c] = struct{}{}
	}
	return false
}
