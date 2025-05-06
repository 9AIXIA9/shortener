package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
	limitmock "shortener/pkg/limit/mock"
)

// 测试中间件创建函数
func TestNewLimitMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLimit := limitmock.NewMockLimit(ctrl)
	middleware := NewLimitMiddleware(mockLimit)

	if middleware == nil {
		t.Fatal("创建的中间件不应为空")
	}
	if middleware.limit != mockLimit {
		t.Errorf("预期中间件的limit为 %v, 实际为 %v", mockLimit, middleware.limit)
	}
}

// 测试请求通过限流的情况
func TestLimitMiddleware_Handle_AllowsRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLimit := limitmock.NewMockLimit(ctrl)
	middleware := NewLimitMiddleware(mockLimit)

	// 设置模拟限流器返回允许请求
	mockLimit.EXPECT().Allow().Return(true)

	// 记录下一个处理器是否被调用
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	// 创建测试请求和响应记录器
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// 执行中间件
	handler := middleware.Handle(next)
	handler(rr, req)

	// 验证下一个处理器被调用
	if !nextCalled {
		t.Error("下一个处理器应该被调用")
	}
}

// 测试请求被限流的情况
func TestLimitMiddleware_Handle_BlocksRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLimit := limitmock.NewMockLimit(ctrl)
	middleware := NewLimitMiddleware(mockLimit)

	// 设置模拟限流器返回拒绝请求
	mockLimit.EXPECT().Allow().Return(false)

	// 记录下一个处理器是否被调用
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	// 创建测试请求和响应记录器
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// 执行中间件
	handler := middleware.Handle(next)
	handler(rr, req)

	// 验证下一个处理器未被调用
	if nextCalled {
		t.Error("下一个处理器不应被调用")
	}

	// 验证响应内容
	if rr.Body.String() == "" {
		t.Error("响应体不应为空")
	}
}
