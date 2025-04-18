package validate

import (
	"github.com/go-playground/validator/v10"
	"net/url"
	"regexp"
	"strings"
)

const (
	minShortUrlLen = 1
	maxShortUrlLen = 11
)

var (
	// 预编译正则表达式提高性能 - 支持更多合法字符和格式
	urlRegex   = regexp.MustCompile(`^(http|https)://([a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?\.)+[a-zA-Z0-9\-]{2,}(:[0-9]{1,5})?(/[-a-zA-Z0-9_%.~+&=:#?]*)*$`)
	shortRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

// validLongUrlValidator 验证长链接
func validLongUrlValidator(fl validator.FieldLevel) bool {
	urlStr := fl.Field().String()

	// 快速检查常见错误 - 空字符串或明显无效URL
	if urlStr == "" {
		return false
	}

	// 使用标准库解析URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// 检查协议
	scheme := parsedURL.Scheme
	if !strings.EqualFold(scheme, "http") && !strings.EqualFold(scheme, "https") {
		return false
	}

	// 检查主机部分
	if parsedURL.Host == "" {
		return false
	}

	// 过滤特定域名
	host := strings.ToLower(parsedURL.Host)
	if host == "localhost" || strings.HasPrefix(host, "127.0.0.") || host == "::1" {
		return false
	}

	// 最后使用正则表达式进行完整验证
	return urlRegex.MatchString(urlStr)
}

// validShortUrlValidator 验证短链接
func validShortUrlValidator(fl validator.FieldLevel) bool {
	shortUrl := fl.Field().String()

	// 检查长度限制(1-11个字符)
	urlLen := len(shortUrl)
	if urlLen < minShortUrlLen || urlLen > maxShortUrlLen {
		return false
	}

	// 短链接只能包含字母和数字
	return shortRegex.MatchString(shortUrl)
}
