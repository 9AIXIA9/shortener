package urlTool

import (
	"net/http"
	"net/http/httptest"
	"shortener/internal/config"
	"shortener/internal/types/errorx"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGetDomainAndPath 测试URL解析函数
func TestGetDomainAndPath(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectDomain   string
		expectBasePath string
	}{
		{
			name:           "正常URL",
			input:          "https://example.com/path/to/resource",
			expectDomain:   "example.com",
			expectBasePath: "path/to/resource",
		},
		{
			name:           "带查询参数的URL",
			input:          "https://example.com/path?query=value",
			expectDomain:   "example.com",
			expectBasePath: "path",
		},
		{
			name:           "无路径URL",
			input:          "https://example.com",
			expectDomain:   "example.com",
			expectBasePath: "",
		},
		{
			name:           "空字符串",
			input:          "",
			expectDomain:   "",
			expectBasePath: "",
		},
		{
			name:           "无效URL",
			input:          "://invalid-url",
			expectDomain:   "",
			expectBasePath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, path := GetDomainAndPath(tt.input)
			assert.Equal(t, tt.expectDomain, domain)
			assert.Equal(t, tt.expectBasePath, path)
		})
	}
}

// TestClientCheck 测试URL连接检查
func TestClientCheck(t *testing.T) {
	// 设置一个成功的测试服务器
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()

	// 设置一个返回错误的测试服务器
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	// 创建自定义配置的客户端
	customConfig := config.ConnectConf{
		DNSServer:       "8.8.8.8:53",
		Timeout:         500 * time.Millisecond,
		MaxRetries:      1,
		MaxIdleConns:    50,
		IdleConnTimeout: 20 * time.Second,
	}
	client := NewClient(customConfig)

	// 测试用例
	tests := []struct {
		name        string
		url         string
		expectValid bool
		expectError bool
		errorCode   errorx.Code
	}{
		{
			name:        "成功URL",
			url:         successServer.URL,
			expectValid: true,
			expectError: false,
		},
		{
			name:        "错误状态码URL",
			url:         errorServer.URL,
			expectValid: false,
			expectError: true,
			errorCode:   errorx.CodeTimeout,
		},
		{
			name:        "无效URL",
			url:         "http://localhost:12345", // 假设这个端口没有服务
			expectValid: false,
			expectError: true,
			errorCode:   errorx.CodeParamError,
		},
		{
			name:        "空URL",
			url:         "",
			expectValid: false,
			expectError: true,
			errorCode:   errorx.CodeParamError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := client.Check(tt.url)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorCode != 0 {
					assert.True(t, errorx.Is(err, tt.errorCode), "预期错误码 %v, 实际: %v", tt.errorCode, err)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectValid, valid)
		})
	}
}

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	// 测试默认配置
	client1 := NewClient()
	assert.NotNil(t, client1)

	// 测试自定义配置
	customConfig := config.ConnectConf{
		DNSServer:       "1.1.1.1:53",
		Timeout:         1 * time.Second,
		MaxRetries:      3,
		MaxIdleConns:    200,
		IdleConnTimeout: 60 * time.Second,
	}
	client2 := NewClient(customConfig)
	assert.NotNil(t, client2)

	// 检查两个客户端的类型
	_, ok1 := client1.(*clientImpl)
	assert.True(t, ok1, "client1应该是clientImpl类型")

	_, ok2 := client2.(*clientImpl)
	assert.True(t, ok2, "client2应该是clientImpl类型")
}

// TestIsSuccessStatusCode 测试HTTP状态码检查
func TestIsSuccessStatusCode(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		expectResult bool
	}{
		{"200 OK", 200, true},
		{"201 Created", 201, true},
		{"204 No Content", 204, true},
		{"299 边界值", 299, true},
		{"300 重定向", 300, false},
		{"400 Bad Request", 400, false},
		{"404 Not Found", 404, false},
		{"500 Server Error", 500, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSuccessStatusCode(tt.statusCode)
			assert.Equal(t, tt.expectResult, result)
		})
	}
}
