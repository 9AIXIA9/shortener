//go:generate mockgen -source=$GOFILE -destination=./mock/connect_mock.go -package=urlTool
package urlTool

import (
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"time"
)

type Client interface {
	Check(url string) bool
}

type clientImpl struct {
	client *http.Client
}

var NewClient = func() Client {
	return &clientImpl{
		client: &http.Client{
			Transport: &http.Transport{DisableKeepAlives: true},
			Timeout:   2 * time.Second,
		},
	}
}

func (c *clientImpl) Check(url string) bool {
	// 添加panic恢复，确保在任何情况下函数都能正常返回
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("a panic occurred during the URL check: %v, url: %s", r, url)
		}
	}()

	// 使用HEAD请求代替GET请求，更高效地检查URL可访问性
	resp, err := c.client.Head(url)
	if err != nil {
		return false
	}

	// 确保resp不为nil
	if resp == nil {
		return false
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			logx.Errorf("failed to close the response body: %v, url: %s", err, url)
		}
	}()

	// 验证状态码是否为成功（2xx）
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
