package errorx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := New(CodeParamError, "参数错误")
	if err.Code != CodeParamError {
		t.Errorf("错误码不匹配，期望 %d，实际 %d", CodeParamError, err.Code)
	}
	if err.Msg != "参数错误" {
		t.Errorf("错误消息不匹配，期望 %s，实际 %s", "参数错误", err.Msg)
	}
	if len(err.Stack) == 0 {
		t.Error("错误堆栈为空，期望包含调用栈信息")
	}
}

func TestWrapError(t *testing.T) {
	cause := errors.New("网络错误")
	err := Wrap(cause, CodeServiceUnavailable, "服务不可用")

	if err.Code != CodeServiceUnavailable {
		t.Errorf("错误码不匹配，期望 %d，实际 %d", CodeServiceUnavailable, err.Code)
	}

	if !errors.Is(cause, err.Cause) {
		t.Error("原始错误未正确封装")
	}
}

func TestIsFunction(t *testing.T) {
	err := New(CodeNotFound, "资源未找到")

	if !Is(err, CodeNotFound) {
		t.Error("Is函数未能正确识别错误码")
	}

	if Is(err, CodeSuccess) {
		t.Error("Is函数错误识别为成功码")
	}

	if !Is(nil, CodeSuccess) {
		t.Error("nil错误应该匹配CodeSuccess")
	}
}

func TestErrorMetadata(t *testing.T) {
	err := New(CodeSystemError, "系统错误")
	err = err.WithMeta("userId", 12345).WithMeta("requestId", "abc-123")

	if val, ok := err.Meta.Get("userId"); !ok || val.(int) != 12345 {
		t.Error("元数据userId设置或获取失败")
	}

	if val, ok := err.Meta.Get("requestId"); !ok || val.(string) != "abc-123" {
		t.Error("元数据requestId设置或获取失败")
	}

	if val, _ := err.Meta.Get("notExist"); val != nil {
		t.Error("不存在的元数据应返回nil")
	}
}

func TestWithContext(t *testing.T) {
	// 创建带有值的上下文
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "req-123")
	ctx = context.WithValue(ctx, TraceIDKey, "trace-456")
	ctx = context.WithValue(ctx, UserIDKey, "user-789")

	// 创建错误并关联上下文
	err := New(CodeParamError, "参数错误").WithContext(ctx)

	// 验证元数据是否正确提取
	if val, ok := err.Meta.Get(RequestIDKey); !ok || val != "req-123" {
		t.Errorf("上下文requestID未正确关联，期望值:%s，实际值:%v", "req-123", val)
	}

	if val, ok := err.Meta.Get(TraceIDKey); !ok || val != "trace-456" {
		t.Errorf("上下文traceID未正确关联，期望值:%s，实际值:%v", "trace-456", val)
	}

	// 测试nil上下文情况，使用context.TO-DO()替代nil
	err2 := New(CodeParamError, "测试").WithContext(context.TODO())
	if err2 == nil {
		t.Error("WithContext(context.TODO())应返回原错误对象而非nil")
	}
}

func TestInitialize(t *testing.T) {
	// 测试自定义配置初始化
	Initialize(
		WithPoolSize(128),
		WithStackTracing(false),
	)

	// 验证堆栈跟踪是否已禁用
	err := New(CodeSystemError, "测试初始化")
	if len(err.Stack) > 0 {
		t.Error("堆栈跟踪应已禁用，但错误对象包含堆栈")
	}

	// 恢复默认配置
	Initialize()
}

func TestPoolCapacityAdjustment(t *testing.T) {
	pool := NewAdaptivePool(PoolConfig{
		MaxSize:      20,
		BufferSize:   10,
		MonitorCycle: 50 * time.Millisecond,
	})

	// 填满池
	var errs []*ErrorX
	for i := 0; i < 15; i++ {
		errs = append(errs, pool.Get())
	}

	// 回收所有对象
	for _, e := range errs {
		pool.Put(e)
	}

	// 等待调整周期
	time.Sleep(100 * time.Millisecond)

	pool.Close()
}

// 添加基准测试
func BenchmarkErrorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := New(CodeParamError, "benchmark-error")
		_ = err
	}
}

// 重新启用堆栈跟踪（为了不影响其他测试）
func TestMain(m *testing.M) {
	// 测试前设置
	originalConfig := struct {
		stackEnabled bool
	}{
		stackEnabled: atomic.LoadUint32(&globalStack) == 1,
	}

	// 运行测试
	code := m.Run()

	// 测试后恢复
	EnableStackTracing(originalConfig.stackEnabled)

	// 退出测试
	os.Exit(code)
}

func TestFormatStack(t *testing.T) {
	// 有堆栈的情况
	err := New(CodeParamError, "参数错误")
	formatted := err.FormatStack()
	if !strings.Contains(formatted, "TestFormatStack") {
		t.Error("格式化堆栈应该包含当前函数名")
	}

	// 无堆栈的情况
	EnableStackTracing(false)
	defer EnableStackTracing(true)

	err2 := New(CodeParamError, "无堆栈错误")
	formatted2 := err2.FormatStack()
	if !strings.Contains(formatted2, "Stack trace not enabled or unavailable") {
		t.Errorf("应返回未启用提示，实际返回: %s", formatted2)
	}
}

// TestErrorWithCause 测试带原因的错误创建
func TestErrorWithCause(t *testing.T) {
	cause := errors.New("原始错误")
	err := NewWithCause(CodeSystemError, "服务器错误", cause)

	if !errors.Is(cause, err.Cause) {
		t.Error("原始错误未正确保存")
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is 无法识别原始错误")
	}
}

// TestErrorMethods 测试错误方法
func TestErrorMethods(t *testing.T) {
	// 测试 Error() 方法
	t.Run("Error方法", func(t *testing.T) {
		err1 := New(CodeParamError, "测试消息")
		expected := fmt.Sprintf("[%d] 测试消息", CodeParamError)
		if err1.Error() != expected {
			t.Errorf("错误消息格式不匹配，期望 %q，实际 %q", expected, err1.Error())
		}

		cause := errors.New("原因")
		err2 := NewWithCause(CodeParamError, "测试消息", cause)
		expected = fmt.Sprintf("[%d] 测试消息: 原因", CodeParamError)
		if err2.Error() != expected {
			t.Errorf("带原因的错误消息格式不匹配，期望 %q，实际 %q", expected, err2.Error())
		}
	})

	// 测试 Unwrap() 方法
	t.Run("Unwrap方法", func(t *testing.T) {
		cause := errors.New("原始错误")
		err := NewWithCause(CodeSystemError, "服务器错误", cause)

		if !errors.Is(cause, err.Unwrap()) {
			t.Error("Unwrap方法未返回原始错误")
		}
	})

	// 测试错误克隆
	t.Run("错误克隆", func(t *testing.T) {
		orig := New(CodeParamError, "原始错误").WithMeta("key1", "value1")
		cloned := Wrap(orig, CodeSystemError, "包装错误")

		// 检查包装后的错误
		if cloned.Code != CodeSystemError {
			t.Errorf("错误码不匹配，期望 %d，实际 %d", CodeSystemError, cloned.Code)
		}

		if cloned.Msg != "包装错误" {
			t.Errorf("错误消息不匹配，期望 %s，实际 %s", "包装错误", cloned.Msg)
		}

		if !errors.Is(orig, cloned.Cause) {
			t.Error("原始错误未正确设置为包装错误的原因")
		}

		// 检查元数据复制
		var ex *ErrorX
		if !errors.As(cloned.Cause, &ex) {
			t.Error("无法通过errors.As获取原始错误")
		}
	})
}

// TestMetadataMapOperations 测试元数据操作
func TestMetadataMapOperations(t *testing.T) {
	t.Run("清空操作", func(t *testing.T) {
		m := newMetadataMap()
		m.Set("key1", "value1")
		m.Set("key2", "value2")

		m.Clear()

		if _, ok := m.Get("key1"); ok {
			t.Error("Clear方法未清除元数据")
		}
	})

	t.Run("复制操作", func(t *testing.T) {
		m := newMetadataMap()
		m.Set("key1", 100)
		m.Set("key2", "test")

		c := m.Copy()

		// 验证复制的map包含原始值
		if v, ok := c.Get("key1"); !ok || v.(int) != 100 {
			t.Errorf("复制的元数据不包含原始值，期望100，实际%v", v)
		}

		// 验证两个map是独立的
		m.Set("key1", 200)
		if v, _ := c.Get("key1"); v.(int) != 100 {
			t.Error("复制的map和原map不独立，原map的修改影响了复制的map")
		}
	})
}

// TestAdaptivePool 测试自适应对象池
func TestAdaptivePool(t *testing.T) {
	// 创建一个小容量的池进行测试
	pool := NewAdaptivePool(PoolConfig{
		MaxSize:      10,
		BufferSize:   5,
		MonitorCycle: 100 * time.Millisecond,
	})

	// 测试基本的获取和回收
	err := pool.Get()
	err.Code = CodeParamError
	err.Msg = "测试错误"

	pool.Put(err)
	time.Sleep(10 * time.Millisecond) // 给一点时间让处理完成

	// 再次获取，应该得到一个重置的对象
	err2 := pool.Get()
	if err2.Code != 0 || err2.Msg != "" {
		t.Error("回收的错误对象未被正确重置")
	}

	// 测试并发操作
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := pool.Get()
			time.Sleep(time.Millisecond)
			pool.Put(e)
		}()
	}
	wg.Wait()

	// 测试关闭
	pool.Close()

	// 关闭后仍应能获取对象
	err3 := pool.Get()
	if err3 == nil {
		t.Error("关闭后Get方法应返回新创建的对象而非nil")
	}

	// 测试对nil的处理
	pool.Put(nil) // 不应该导致panic
}

// BenchmarkErrorWithStack 对比有无堆栈追踪的性能差异
func BenchmarkErrorWithStack(b *testing.B) {
	// 开启和关闭堆栈跟踪的性能比较
	b.Run("WithStack", func(b *testing.B) {
		EnableStackTracing(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := New(CodeParamError, "benchmark-error")
			_ = err
		}
	})

	b.Run("WithoutStack", func(b *testing.B) {
		EnableStackTracing(false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := New(CodeParamError, "benchmark-error")
			_ = err
		}
	})
}

func TestStackFilters(t *testing.T) {
	// 保存原始配置
	origFilters := defaultStackFilters
	defer func() { defaultStackFilters = origFilters }()

	// 测试过滤功能
	Initialize(
		WithStackFilters("testing"),
	)

	err := New(CodeParamError, "测试错误")

	// 堆栈中不应包含testing包路径
	for _, frame := range err.Stack {
		if strings.Contains(frame, "testing.") {
			t.Errorf("堆栈帧应被过滤: %s", frame)
		}
	}
}
