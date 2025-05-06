//go:generate mockgen -source=$GOFILE -destination=./mock/connect_mock.go -package=urlTool
package urlTool

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
	"net"
	"net/http"
	"shortener/internal/config"
	"shortener/internal/types/errorx"
	"strings"
	"time"
)

var globalSF = syncx.NewSingleFlight()

var (
	defaultDNSServer = "8.8.8.8:53"
	defaultConfig    = config.ConnectConf{
		DNSServer:       defaultDNSServer,
		Timeout:         800 * time.Millisecond,
		MaxRetries:      2,
		MaxIdleConns:    100,
		IdleConnTimeout: 30 * time.Second,
	}
)

type Client interface {
	Check(url string) (bool, error)
}

type clientImpl struct {
	config config.ConnectConf
	client *http.Client
}

func NewClient(cfg ...config.ConnectConf) Client {
	conf := defaultConfig
	if len(cfg) > 0 {
		conf = cfg[0]
	}

	dialer := &net.Dialer{
		Resolver:  newResolver(conf.DNSServer),
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        conf.MaxIdleConns,
		IdleConnTimeout:     conf.IdleConnTimeout,
		MaxIdleConnsPerHost: 10, // 限制单主机连接数
		TLSHandshakeTimeout: 1 * time.Second,
	}

	return &clientImpl{
		client: &http.Client{
			Transport: transport,
			Timeout:   conf.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // 禁止自动重定向
			},
		},
		config: conf,
	}

}

func newResolver(dnsServer string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 500 * time.Millisecond}
			return d.DialContext(ctx, "udp", dnsServer)
		},
	}
}

func (c *clientImpl) Check(URL string) (bool, error) {
	if len(URL) == 0 {
		return false, errorx.New(errorx.CodeParamError, "URL is null")
	}

	result, err := globalSF.Do(URL, func() (any, error) {
		return c.checkWithRetry(URL)
	})

	return result.(bool), err
}

func (c *clientImpl) checkWithRetry(url string) (success bool, err error) {
	for i := 0; i <= c.config.MaxRetries; i++ {
		success, err = c.check(url)
		if success && err == nil {
			return true, nil
		}

		if err != nil && !errorx.Is(err, errorx.CodeTimeout) {
			return false, err
		}

		if i < c.config.MaxRetries {
			time.Sleep(time.Duration(50*i) * time.Millisecond) // 退避策略
		}
	}

	// 所有重试都失败后
	if err != nil {
		return false, err
	}
	// 处理没有明确错误但请求未成功的情况（如状态码非2xx）
	return false, errorx.New(errorx.CodeTimeout, "check URL failed after all retries")
}

func (c *clientImpl) check(url string) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("panic during URL check: %v, url: %s", r, url)
		}
	}()

	resp, err := c.client.Head(url)
	if err != nil {
		// 检查是否是超时错误
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return false, errorx.NewWithCause(errorx.CodeTimeout, "the connection to this url timed out", err)
		}

		// 通过错误消息检查是否为超时
		if strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "deadline exceeded") {
			return false, errorx.NewWithCause(errorx.CodeTimeout, "the connection timed out", err)
		}

		// 其他错误情况
		return false, errorx.NewWithCause(errorx.CodeParamError, "can't connect to this url", err)
	}

	if resp == nil {
		return false, errorx.New(errorx.CodeParamError, "there is no reply")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logx.Errorf("failed to close response body: %v", err)
		}
	}()

	return isSuccessStatusCode(resp.StatusCode), nil
}

func isSuccessStatusCode(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
