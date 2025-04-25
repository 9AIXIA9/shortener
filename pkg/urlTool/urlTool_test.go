package urlTool

import (
	"net/http"
	"net/http/httptest"
	"shortener/internal/config"
	"sync"
	"testing"
	"time"
)

// 测试创建客户端实例
func TestNewClient(t *testing.T) {
	// 测试默认配置
	client := NewClient()
	if client == nil {
		t.Fatal("使用默认配置创建客户端失败")
	}

	// 测试自定义配置
	customConfig := config.ConnectConf{
		DNSServer:       "1.1.1.1:53",
		Timeout:         500 * time.Millisecond,
		MaxRetries:      1,
		MaxIdleConns:    50,
		IdleConnTimeout: 20 * time.Second,
	}
	customClient := NewClient(customConfig)
	if customClient == nil {
		t.Fatal("使用自定义配置创建客户端失败")
	}
}

// 测试状态码判断逻辑
func TestIsSuccessStatusCode(t *testing.T) {
	testCases := []struct {
		statusCode int
		expected   bool
		name       string
	}{
		{199, false, "低于成功范围"},
		{200, true, "成功范围下限"},
		{201, true, "成功范围内"},
		{299, true, "成功范围上限"},
		{300, false, "高于成功范围"},
		{404, false, "Not Found错误"},
		{500, false, "服务器错误"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSuccessStatusCode(tc.statusCode)
			if result != tc.expected {
				t.Errorf("isSuccessStatusCode(%d) = %v, 期望 %v", tc.statusCode, result, tc.expected)
			}
		})
	}
}

// 使用httptest测试URL检查功能
func TestClientImpl_Check(t *testing.T) {
	// 模拟成功响应的服务器
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()

	// 模拟失败响应的服务器
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	tests := []struct {
		name      string
		url       string
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "成功响应",
			url:       successServer.URL,
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "错误响应",
			url:       failServer.URL,
			wantValid: false,
			wantErr:   false,
		},
		{
			name:      "空URL",
			url:       "",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "无效URL格式",
			url:       "not-a-url",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient()
			gotValid, err := client.Check(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("Check() 错误 = %v, 期望错误 %v", err, tt.wantErr)
				return
			}

			if gotValid != tt.wantValid {
				t.Errorf("Check() 结果 = %v, 期望 %v", gotValid, tt.wantValid)
			}
		})
	}
}

// 测试SingleFlight功能 - 确保对同一域名的并发请求只执行一次实际检查
func TestSingleFlightBehavior(t *testing.T) {
	// 创建测试服务器，记录请求次数
	var requestCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		// 模拟耗时操作
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()

	// 并发执行多个相同URL的检查
	const concurrentRequests = 5
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			defer wg.Done()
			valid, err := client.Check(server.URL)
			if err != nil {
				t.Errorf("意外错误: %v", err)
			}
			if !valid {
				t.Errorf("期望URL有效")
			}
		}()
	}

	wg.Wait()

	// 由于SingleFlight去重，请求次数应该远小于并发数
	if requestCount > 2 {
		t.Errorf("SingleFlight功能未按预期工作: 对于%d个并发请求，发出了%d个实际请求",
			concurrentRequests, requestCount)
	}
}

// 测试客户端对网络错误的处理
func TestClientImpl_NetworkErrors(t *testing.T) {
	// 使用不可达的地址
	unreachableURL := "http://localhost:54321" // 假设这个端口没有服务

	client := NewClient()
	valid, err := client.Check(unreachableURL)

	if err != nil {
		t.Errorf("对于不可达URL，期望无错误，但得到: %v", err)
	}

	if valid {
		t.Errorf("期望不可达URL检查结果为无效，但得到了有效")
	}
}
