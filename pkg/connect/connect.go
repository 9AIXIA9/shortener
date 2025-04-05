package connect

import (
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"time"
)

// NewClient 创建一个新的 HTTP 客户端
func NewClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
		Timeout:   2 * time.Second,
	}
}

// Check 测试 URL 连通性
func Check(client *http.Client, url string) bool {
	// 尝试连接
	resp, err := client.Get(url)
	if err != nil {
		logx.Errorw("connect get url failed", logx.Field("url", url), logx.Field("err", err))
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logx.Errorw("connect close resp body failed", logx.Field("url", url), logx.Field("err", err))
		}
	}()

	return resp.StatusCode == http.StatusOK
}
