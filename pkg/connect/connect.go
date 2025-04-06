package connect

import (
	"fmt"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type Client interface {
	Check(url string) (bool, error)
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

func (c *clientImpl) Check(url string) (bool, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return false, fmt.Errorf("connect get url failed,err:%w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			logx.Errorw("connect close resp body failed",
				logx.Field("url", url),
				logx.Field("err", err))
		}
	}()

	return resp.StatusCode == http.StatusOK, nil
}
