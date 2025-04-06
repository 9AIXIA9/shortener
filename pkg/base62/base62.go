package base62

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"sync"
)

//实现62进制转换

// 使用环境变量或默认值的base62字符集
var (
	base62Str string
	once      sync.Once
)

// Convert 将数字转换为62进制字符串
// 例如:
// 63 -> "11"
// 1163 -> "iL"
func Convert(number uint64) string {
	//懒加载
	once.Do(func() {
		Init()
	})

	if number == 0 {
		return "0"
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

// Parse 将62进制字符串转换为数字
// 例如:
// "11" -> 63
// "iL" -> 1163
func Parse(encoded string) (uint64, error) {
	//懒加载
	once.Do(func() {
		Init()
	})

	if encoded == "" {
		return 0, fmt.Errorf("base62.Parse failed,encoded string is empty")
	}

	if encoded == "0" {
		return 0, nil
	}

	// 创建字符到索引的映射表
	charToIndex := make(map[byte]uint64)
	for i, char := range []byte(base62Str) {
		charToIndex[char] = uint64(i)
	}

	var result uint64

	// 从高位到低位处理每个字符
	for i := 0; i < len(encoded); i++ {
		char := encoded[i]
		value, exists := charToIndex[char]
		if !exists {
			return 0, fmt.Errorf("base62.Parse failed,there is an invalid char in encoded string: %c", char)
		}

		// 当前结果乘以进制基数，再加上当前位的值
		result = result*62 + value
	}

	return result, nil
}

func Init() {
	// 从环境变量获取字符集
	base62Str = os.Getenv("BASE62STR")

	// 如果环境变量未设置，使用默认值
	if base62Str == "" {
		base62Str = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	// 验证字符集长度是否为62
	if len(base62Str) != 62 {
		logx.Severef("BASE62STR must contain exactly 62 characters")
	}

	// 验证字符集中的字符是否唯一
	if hasDuplicateChars(base62Str) {
		logx.Severef("BASE62STR contains duplicate characters")
	}
}

// hasDuplicateChars 检查字符串是否包含重复字符
func hasDuplicateChars(s string) bool {
	charSet := make(map[rune]struct{})
	for _, c := range s {
		if _, exists := charSet[c]; exists {
			return true
		}
		charSet[c] = struct{}{}
	}
	return false
}
