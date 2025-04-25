package urlTool

import (
	"net/url"
	"shortener/pkg/errorx"
)

// GetDomainAndPath 获取URL中的域名和基础路径
func GetDomainAndPath(URL string) (domain string, basePath string) {
	if len(URL) == 0 {
		return "", ""
	}

	//解析URL
	myUrl, err := url.Parse(URL)
	if err != nil {
		return "", ""
	}

	if len(myUrl.Host) == 0 || len(myUrl.Path) == 0 {
		return "", ""
	}

	basePath = myUrl.Path
	if len(basePath) > 0 && basePath[0] == '/' {
		basePath = basePath[1:]
	}

	return myUrl.Host, basePath
}

// GetDomain 获取URL中的域名
func getDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u == nil {
		return "", errorx.NewWithCause(errorx.CodeParamError, "failed to parse url", err)
	}
	return u.Hostname(), nil
}
