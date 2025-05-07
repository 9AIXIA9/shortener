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
	"shortener/internal/errorhandler"
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
	Check(url string) error
}

type clientImpl struct {
	config config.ConnectConf
	client *http.Client
}

func NewClient() Client {
	conf := defaultConfig

	return NewClientWithConfig(conf)
}

func NewClientWithConfig(conf config.ConnectConf) Client {
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

func (c *clientImpl) Check(URL string) error {
	if len(URL) == 0 {
		return errorx.New(errorx.CodeParamError, "URL is null")
	}

	_, err := globalSF.Do(URL, func() (any, error) {
		return nil, c.checkWithRetry(URL)
	})

	return err
}

func (c *clientImpl) checkWithRetry(url string) (err error) {
	//多一轮循环用于判断返回值
	for i := 0; i <= c.config.MaxRetries; i++ {
		err = c.check(url)
		if err == nil {
			return nil
		}

		if errorx.Is(err, errorx.CodeTimeout) {
			c.backoff(i)
			continue
		}

		return err
	}

	return err
}

func (c *clientImpl) check(url string) error {
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
			return errorx.NewWithCause(errorx.CodeTimeout, "the connection timed out", err)
		}

		// 通过错误消息检查是否为超时
		if strings.Contains(err.Error(), "timeout") ||
			strings.Contains(err.Error(), "deadline exceeded") {
			return errorx.NewWithCause(errorx.CodeTimeout, "the connection timed out", err)
		}

		// 其他错误情况
		return errorx.NewWithCause(errorx.CodeParamError, "can't connect to this url", err)
	}

	if resp == nil {
		return errorx.New(errorx.CodeParamError, "no response was received")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			errorhandler.SubmitWithPriority(errorx.NewWithCause(errorx.CodeSystemError, "failed to close the response body", err), errorhandler.DefaultPriority)
		}
	}()

	return isSuccessStatusCode(resp.StatusCode)
}

func isSuccessStatusCode(statusCode int) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return errorx.New(errorx.CodeParamError, "abnormal http code").WithMeta("httpCode", statusCode)
}

// 退避策略
func (c *clientImpl) backoff(times int) {
	if times >= c.config.MaxRetries {
		return
	}
	time.Sleep(time.Duration(50*times) * time.Millisecond)
}
