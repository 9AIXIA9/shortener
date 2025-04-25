package urlTool

import (
	"net/url"
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
