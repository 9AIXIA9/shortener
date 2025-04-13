package errorx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ExampleErrorX_FormatStack 展示如何格式化查看堆栈
func ExampleErrorX_FormatStack() {
	EnableStackTracing(true)
	err := New(CodeParamError, "参数错误")

	// 格式化查看堆栈
	stack := err.FormatStack()

	// 仅打印第一行以便于示例测试
	lines := strings.Split(stack, "\n")
	if len(lines) > 0 {
		fmt.Println("堆栈跟踪已捕获")
	}
	// Output: 堆栈跟踪已捕获
}

// ExampleErrorX_Detail 展示如何获取详细错误信息
func ExampleErrorX_Detail() {
	baseErr := errors.New("底层错误")
	err := NewWithCause(CodeDatabaseError, "数据库查询失败", baseErr)
	err = err.WithMeta("sql", "SELECT * FROM performance_schema.users")

	// 获取详细错误信息
	detail := err.Detail()
	fmt.Println("已生成详细错误信息", detail)
	// Output中不包含堆栈，因为它每次都不同
	// Output: 已生成详细错误信息
}

// ExamplePrintErrorTree 展示如何打印错误树
func ExamplePrintErrorTree() {
	// 构建错误链
	baseErr := errors.New("连接超时")
	dbErr := NewWithCause(CodeDatabaseError, "数据库查询失败", baseErr)
	apiErr := Wrap(dbErr, CodeSystemError, "API处理失败")
	apiErr = apiErr.WithMeta("endpoint", "/api/users")

	// 打印错误树到缓冲区
	var buf bytes.Buffer
	err := PrintErrorTree(apiErr, &buf)
	if err != nil {
		return
	}
	fmt.Println("错误树已打印")
	// Output: 错误树已打印
}

// ExampleWithStackFilters 展示如何使用堆栈过滤
func ExampleWithStackFilters() {
	// 保存当前配置
	origFilters := defaultStackFilters
	defer func() { defaultStackFilters = origFilters }()

	// 初始化时设置过滤
	Initialize(
		WithStackFilters("runtime", "testing"),
	)

	_ = New(CodeSystemError, "系统错误")
	fmt.Println("已创建带过滤堆栈的错误")

	// Output: 已创建带过滤堆栈的错误
}

// ExampleWrap 展示如何包装现有错误
func ExampleWrap() {
	// 原始错误
	dbErr := errors.New("数据库连接失败")

	// 包装错误，添加更多上下文
	err := Wrap(dbErr, CodeDatabaseError, "查询用户数据失败")
	err = err.WithMeta("userId", 12345)

	fmt.Println("已包装错误并添加元数据")
	// Output: 已包装错误并添加元数据
}

// ExampleNew_withContext 展示如何创建带上下文的错误
func ExampleNew_withContext() {
	// 创建带有请求ID的上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "req-abc-123")

	// 创建关联上下文的错误
	err := New(CodeParamError, "参数验证失败").WithContext(ctx)

	// 验证元数据已关联
	if val, ok := err.Meta.Get(RequestIDKey); ok {
		fmt.Printf("错误已关联请求ID: %s\n", val)
	}
	// Output: 错误已关联请求ID: req-abc-123
}

// ExampleIs 展示如何使用Is函数判断错误类型
func ExampleIs() {
	// 创建错误
	err := New(CodeNotFound, "资源不存在")

	// 检查错误类型
	if Is(err, CodeNotFound) {
		fmt.Println("资源不存在错误")
	}

	// nil错误默认是成功
	if Is(nil, CodeSuccess) {
		fmt.Println("nil错误被识别为成功")
	}
	// Output: 资源不存在错误
	// nil错误被识别为成功
}

// TestToHTTPStatus 测试错误代码到HTTP状态码的映射
func TestToHTTPStatus(t *testing.T) {
	testCases := []struct {
		code     Code
		expected int
	}{
		{CodeSuccess, 200},
		{CodeParamError, 400},
		{CodeNotFound, 404},
		{CodeDatabaseError, 500},
		{CodeCacheError, 500},
		{CodeSystemError, 500},
		{CodeServiceUnavailable, 503},
		{Code(9999), 500}, // 未定义的错误码应返回500
	}

	for _, tc := range testCases {
		status := ToHTTPStatus(tc.code)
		if status != tc.expected {
			t.Errorf("错误�� %d 的HTTP状态码映射错误，期望 %d，实际 %d", tc.code, tc.expected, status)
		}
	}
}

// TestErrorDetail 详细测试Detail方法
func TestErrorDetail(t *testing.T) {
	EnableStackTracing(true)

	// 构建带有完整信息的错误
	baseErr := errors.New("基础错误")
	err := NewWithCause(CodeDatabaseError, "数据库查询失败", baseErr)
	err = err.WithMeta("sql", "SELECT * FROM performance_schema.users")
	err = err.WithMeta("params", map[string]interface{}{
		"id":    12345,
		"limit": 10,
	})

	detail := err.Detail()

	// 检查详细输出包含所有关键信息
	expectedParts := []string{
		fmt.Sprintf("Error: [%d] 数据库查询失败", CodeDatabaseError),
		"Time:",
		"Cause: 基础错误",
		"Metadata:",
		"sql: SELECT * FROM performance_schema.users",
		"params: map",
		"Stack trace:",
	}

	for _, part := range expectedParts {
		if !strings.Contains(detail, part) {
			t.Errorf("Detail()输出缺少预期内容：%q", part)
		}
	}
}

// TestPrintErrorTreeComplex 测试复杂错误树的打印
func TestPrintErrorTreeComplex(t *testing.T) {
	t.Run("ComplexErrorTree", func(t *testing.T) {
		// 构建多层错误链
		lvl1 := errors.New("网络连接失败")
		lvl2 := NewWithCause(CodeDatabaseError, "数据库查询超时", lvl1)
		lvl3 := Wrap(lvl2, CodeSystemError, "后端服务异常")
		lvl3 = lvl3.WithMeta("service", "user-service")
		lvl4 := Wrap(lvl3, CodeParamError, "请求处理失败")

		var buf bytes.Buffer
		err := PrintErrorTree(lvl4, &buf)
		if err != nil {
			t.Fatalf("PrintErrorTree返回错误: %v", err)
		}

		output := buf.String()

		// 验证输出包含所有错误层级
		expectedParts := []string{
			"Error tree:",
			fmt.Sprintf("[%d] 请求处理失败", CodeParamError),
			fmt.Sprintf("[%d] 后端服务异常", CodeSystemError),
			"Metadata:",
			"service: user-service",
			fmt.Sprintf("[%d] 数据库查询超时", CodeDatabaseError),
			"网络连接失败",
		}

		for _, part := range expectedParts {
			if !strings.Contains(output, part) {
				t.Errorf("错误树输出缺少预期内容：%q", part)
			}
		}
	})

	t.Run("NilError", func(t *testing.T) {
		// 测试空错误情况
		var buf2 bytes.Buffer
		err := PrintErrorTree(nil, &buf2)
		if err != nil {
			t.Errorf("PrintErrorTree(nil)返回了错误: %v", err)
		}
		if buf2.Len() != 0 {
			t.Errorf("对nil错误的PrintErrorTree应输出空，但实际输出：%q", buf2.String())
		}
	})
}

// TestErrorPoolReuse 测试错误对象池复用机制
func TestErrorPoolReuse(t *testing.T) {
	// 重置错误池
	oldPool := errorPool
	defer func() { errorPool = oldPool }()

	poolConfig := PoolConfig{
		MaxSize:      10,
		BufferSize:   5,
		MonitorCycle: time.Millisecond * 10,
	}
	errorPool = NewAdaptivePool(poolConfig)

	// 创建和回收多个错误对象
	var errs []*ErrorX
	for i := 0; i < 20; i++ {
		err := New(CodeParamError, "测试错误")
		errs = append(errs, err)
	}

	// 回收前10个错误
	for i := 0; i < 10; i++ {
		errorPool.Put(errs[i])
	}

	// 再获取10个错误对象，应该有部分是复用的
	newErrs := make([]*ErrorX, 10)
	for i := 0; i < 10; i++ {
		newErrs[i] = errorPool.Get()
	}

	// 关闭池并再次尝试获取和回收，确保不会出现panic
	errorPool.Close()

	e := errorPool.Get()
	if e == nil {
		t.Error("关闭后的错误池Get()不应返回nil")
	}

	// 回收到已关闭的池不应导致panic
	errorPool.Put(e)
}

// TestConcurrentErrorHandling 测试错误处理的并发安全性
func TestConcurrentErrorHandling(t *testing.T) {
	const goroutines = 50
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// 创建并操作错误对象
				err := New(CodeParamError, "并发测试错误")
				err = err.WithMeta("goroutine", id)
				err = err.WithMeta("iteration", j)

				// 创建嵌套错误
				nestedErr := Wrap(err, CodeSystemError, "嵌套错误")
				_ = nestedErr.Detail() // 触发堆栈和元数据处理

				// 回收错误对象
				if j%2 == 0 {
					// 练习错误池的Put方法
					errorPool.Put(err)
					errorPool.Put(nestedErr)
				}
			}
		}(i)
	}

	wg.Wait() // 等待所有goroutine完成
}

// TestComplexMetadata 测试带有复杂元数据的错误处理
func TestComplexMetadata(t *testing.T) {
	// 创建具有不同类型元数据的错误
	err := New(CodeSystemError, "系统内部错误")
	err = err.WithMeta("userId", 123456)
	err = err.WithMeta("roles", []string{"admin", "user"})
	err = err.WithMeta("requestTime", time.Now())
	err = err.WithMeta("config", map[string]interface{}{
		"timeout": 30,
		"retry":   true,
		"cache":   false,
	})

	// 验证元数据是否正确存储和获取
	if val, ok := err.Meta.Get("userId"); !ok || val.(int) != 123456 {
		t.Errorf("元数据userId获取失败，期望123456，实际%v", val)
	}

	if val, ok := err.Meta.Get("roles"); !ok {
		t.Error("元数据roles获取失败")
	} else {
		roles, ok := val.([]string)
		if !ok || len(roles) != 2 || roles[0] != "admin" || roles[1] != "user" {
			t.Errorf("元数据roles内容不匹配，实际值: %v", val)
		}
	}

	if val, ok := err.Meta.Get("config"); !ok {
		t.Error("元数据config获取失败")
	} else {
		cfg, ok := val.(map[string]interface{})
		if !ok || cfg["timeout"].(int) != 30 || !cfg["retry"].(bool) {
			t.Errorf("元数据config内容不匹配，实际值: %v", val)
		}
	}

	// 测试错误的JSON序列化兼容性(如果有需要)
	detail := err.Detail()
	if !strings.Contains(detail, "userId") || !strings.Contains(detail, "roles") {
		t.Error("Detail()输出中应包含所有��数据字段")
	}
}
