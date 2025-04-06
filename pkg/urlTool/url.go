package urlTool

import (
	"fmt"
	"net/url"
	"path"
)

// GetBasePath 从 URL 中提取基础路径
// http://example.com/path/to/resource.html`
// `GetBasePath 将返回 resource.html
func GetBasePath(URL string) (string, error) {
	myUrl, err := url.Parse(URL)
	if err != nil {
		return "", fmt.Errorf("parse url failed,err:%w", err)
	}

	basePath := path.Base(myUrl.Path)
	return basePath, nil
}
