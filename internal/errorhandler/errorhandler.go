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

//todo:
//错误处理策略不够灵活
//缺少错误分类和优先级机制
//没有错误过滤和聚合能力
//无法根据错误类型选择不同的处理器
//配置不够灵活

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

func Submit(err error) {
	if defaultErrorHandler != nil {
		defaultErrorHandler.submit(err)
	} else {
		logx.Error("Error handler not initialized")
	}
}

func Shutdown() {
	if defaultErrorHandler != nil {
		defaultErrorHandler.shutdown()
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

func (h *errorHandler) submit(err error) bool {
	if err == nil {
		return false
	}

	// 确保输入是 ErrorX 类型
	var ex *errorx.ErrorX
	if !errors.As(err, &ex) {
		ex = errorx.NewWithCause(errorx.CodeSystemError, "unpackaged errors", err)
	}

	// 非阻塞方式提交错误
	select {
	case h.errorChan <- ex:
		return true
	default:
		// 通道已满，记录丢弃
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
