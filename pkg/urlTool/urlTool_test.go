package urlTool

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"shortener/internal/config"
	"shortener/internal/types/errorx"
	"testing"
	"time"
)

func TestGetDomainAndPath(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantDomain   string
		wantBasePath string
	}{
		{
			name:         "完整URL",
			url:          "https://example.com/path/to/resource",
			wantDomain:   "example.com",
			wantBasePath: "path/to/resource",
		},
		{
			name:         "带端口号的URL",
			url:          "http://example.com:8080/path",
			wantDomain:   "example.com:8080",
			wantBasePath: "path",
		},
		{
			name:         "不带路径的URL",
			url:          "https://example.com",
			wantDomain:   "example.com",
			wantBasePath: "",
		},
		{
			name:         "带查询参数的URL",
			url:          "https://example.com/path?key=value",
			wantDomain:   "example.com",
			wantBasePath: "path",
		},
		{
			name:         "空URL",
			url:          "",
			wantDomain:   "",
			wantBasePath: "",
		},
		{
			name:         "无效URL",
			url:          "not-a-url",
			wantDomain:   "",
			wantBasePath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, path := GetDomainAndPath(tt.url)
			assert.Equal(t, tt.wantDomain, domain, "域名不匹配")
			assert.Equal(t, tt.wantBasePath, path, "路径不匹配")
		})
	}
}

func TestClient_Check(t *testing.T) {
	// 创建测试服务器
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()

	notFoundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundServer.Close()

	// 创建测试配置
	conf := config.ConnectConf{
		DNSServer:       "8.8.8.8:53",
		Timeout:         200 * time.Millisecond,
		MaxRetries:      1,
		MaxIdleConns:    10,
		IdleConnTimeout: 5 * time.Second,
	}

	client := NewClientWithConfig(conf)

	tests := []struct {
		name        string
		url         string
		wantErr     bool
		expectedErr error
	}{
		{
			name:    "成功的URL检查",
			url:     successServer.URL,
			wantErr: false,
		},
		{
			name:        "404错误",
			url:         notFoundServer.URL,
			wantErr:     true,
			expectedErr: errorx.New(errorx.CodeParamError, "abnormal http code"),
		},
		{
			name:        "无效URL",
			url:         "http://invalid.domain.that.does.not.exist.example",
			wantErr:     true,
			expectedErr: errorx.NewWithCause(errorx.CodeParamError, "can't connect to this url", errors.New("")),
		},
		{
			name:        "空URL",
			url:         "",
			wantErr:     true,
			expectedErr: errorx.New(errorx.CodeParamError, "URL is null"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Check(tt.url)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					var e *errorx.ErrorX
					if errors.As(err, &e) {
						assert.Contains(t, e.Error(), tt.expectedErr.Error())
					} else {
						t.Errorf("预期错误类型为errorx.ErrorX，实际为: %T", err)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	// 测试默认配置创建客户端
	client := NewClient()
	assert.NotNil(t, client)

	// 测试自定义配置创建客户端
	conf := config.ConnectConf{
		DNSServer:       "1.1.1.1:53",
		Timeout:         500 * time.Millisecond,
		MaxRetries:      3,
		MaxIdleConns:    50,
		IdleConnTimeout: 10 * time.Second,
	}

	customClient := NewClientWithConfig(conf)
	assert.NotNil(t, customClient)
}

func TestClientImpl_backoff(t *testing.T) {
	conf := config.ConnectConf{
		MaxRetries: 2,
	}

	client := &clientImpl{
		config: conf,
	}

	// 测试退避逻辑
	start := time.Now()
	client.backoff(0) // 第一次重试
	firstDuration := time.Since(start)

	start = time.Now()
	client.backoff(1) // 第二次重试
	secondDuration := time.Since(start)

	// 第二次退避时间应该大于第一次
	assert.Greater(t, secondDuration.Milliseconds(), firstDuration.Milliseconds())

	// 超过最大重试次数应该立即返回
	start = time.Now()
	client.backoff(2)
	assert.Less(t, time.Since(start).Milliseconds(), int64(10)) // 应该几乎立即返回
}

func TestIsSuccessStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"2xx状态码应成功", 200, false},
		{"2xx状态码应成功", 201, false},
		{"2xx状态码应成功", 299, false},
		{"非2xx应失败", 199, true},
		{"非2xx应失败", 300, true},
		{"非2xx应失败", 404, true},
		{"非2xx应失败", 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isSuccessStatusCode(tt.statusCode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
