package sensitive

import "strings"

// Exist 检测敏感词
func Exist(sensitiveWords []string, word string) bool {
	if len(sensitiveWords) == 0 || word == "" {
		return false
	}

	// 将检查的词转为小写，提高匹配准确性
	lowerWord := strings.ToLower(word)

	// 检查是否包含任何敏感词
	for _, key := range sensitiveWords {
		if strings.Contains(lowerWord, strings.ToLower(key)) {
			return true
		}
	}

	return false
}
