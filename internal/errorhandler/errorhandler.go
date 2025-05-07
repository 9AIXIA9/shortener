package errorhandler

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/internal/config"
	"shortener/internal/types/errorx"
	"sync"
	"time"
)

const priorityKey = "priority"

//todo:
//错误处理策略不够灵活
//没有错误过滤和聚合能力
//无法根据错误类型选择不同的处理器
//配置不够灵活

type ErrorPriority uint8

const (
	PriorityDebug ErrorPriority = iota
	PriorityInfo
	PriorityWarn     // 默认级别，一般业务错误
	PriorityError    // 需要关注的错误，但不影响系统运行
	PriorityCritical // 关键错误，可能影响部分功能
	PriorityFatal    // 致命错误，系统无法正常运行

	DefaultPriority = PriorityWarn
)

var (
	defaultErrorHandler *errorHandler
	once                sync.Once
)

// Init 使用配置初始化错误处理器
func Init(conf config.ErrorHandlerConf, consumers ...errorConsumer) {
	once.Do(func() {
		defaultErrorHandler = newErrorHandler(conf.BufferSize, conf.MaxWorkers, conf.ShutdownTimeout, consumers...)
	})
}

func SubmitWithPriority(err error, priority ErrorPriority) {
	if defaultErrorHandler != nil {
		defaultErrorHandler.submit(ConvertIntoErrorx(err), priority)
	} else {
		logx.Error("Error handler not initialized")
	}
}

func Submit(err error) {
	if err == nil {
		return
	}

	ex := ConvertIntoErrorx(err)

	priority := getDefaultPriority(ex.Code)

	SubmitWithPriority(ex, priority)
}

func Shutdown() {
	if defaultErrorHandler != nil {
		defaultErrorHandler.shutdown()
	}
}

// getDefaultPriority 根据错误代码获取默认优先级
func getDefaultPriority(code errorx.Code) ErrorPriority {
	switch code {
	case errorx.CodeSystemError:
		return PriorityCritical
	case errorx.CodeDatabaseError, errorx.CodeCacheError:
		return PriorityError
	case errorx.CodeTimeout, errorx.CodeServiceUnavailable:
		return PriorityWarn
	case errorx.CodeParamError, errorx.CodeNotFound:
		return PriorityInfo
	default:
		return PriorityWarn
	}
}

// errorConsumer 表示错误处理函数
type errorConsumer func(err *errorx.ErrorX) error

type errorHandler struct {
	errorChan   chan *errorx.ErrorX // 错误缓冲通道
	consumers   []errorConsumer     // 错误消费者列表
	workerCount int                 // 工作协程数
	wg          sync.WaitGroup      // 等待组
	ctx         context.Context     // 上下文控制
	cancel      context.CancelFunc  // 取消函数
	timeout     time.Duration
}

func newErrorHandler(bufferSize int, maxWorkers int, timeout time.Duration, consumers ...errorConsumer) *errorHandler {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &errorHandler{
		errorChan:   make(chan *errorx.ErrorX, bufferSize),
		consumers:   consumers,
		workerCount: maxWorkers,
		ctx:         ctx,
		cancel:      cancel,
		timeout:     timeout,
	}

	// 启动工作协程
	handler.start()

	return handler
}

// 启动错误处理工作协程
func (h *errorHandler) start() {
	for i := 0; i < h.workerCount; i++ {
		h.wg.Add(1)
		go h.processErrors()
	}
}

// 处理错误的工作协程
func (h *errorHandler) processErrors() {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			return
		case err := <-h.errorChan:
			h.handleError(err)
		}
	}
}

// 处理单个错误
func (h *errorHandler) handleError(err *errorx.ErrorX) {
	for _, consumer := range h.consumers {
		// 错误处理器的错误不应该影响其他错误处理
		if processErr := consumer(err); processErr != nil {
			logx.Errorf("consumer handle error failed,err:%v", err)
		}
	}
}

func (h *errorHandler) submit(err *errorx.ErrorX, priority ErrorPriority) bool {
	// 将优先级附加到错误元数据
	err = err.WithMeta(priorityKey, priority)

	// 高优先级错误直接处理，绕过队列
	if priority >= PriorityCritical {
		go h.handleError(err)
		return true
	}

	// 非阻塞方式提交普通错误
	select {
	case h.errorChan <- err:
		return true
	default:
		// 队列已满但是Error优先级以上的错误，阻塞提交
		if priority >= PriorityError {
			h.errorChan <- err
			return true
		}
		// 低优先级错误丢弃
		logx.Errorf("discard unhandled error:%v", err)
		return false
	}
}

func (h *errorHandler) shutdown() {
	// 停止接收新错误
	h.cancel()

	// 等待现有错误处理完成或超时
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logx.Info("errorHandler is successfully closed")
		return
	case <-time.After(h.timeout):
		// 超时处理：记录日志，通知调用方
		logx.Error("errorHandler shutdown timeout, some error processing may not complete")
		// 这里我们已经调用了cancel()，所以工作协程最终会退出
		// 虽然我们不等它们完成，但协程不会泄露
	}
}

func ConvertIntoErrorx(err error) *errorx.ErrorX {
	// 确保输入是 ErrorX 类型
	var ex *errorx.ErrorX
	if !errors.As(err, &ex) {
		ex = errorx.NewWithCause(errorx.CodeSystemError, "unpackaged error", err)
	}
	return ex
}
