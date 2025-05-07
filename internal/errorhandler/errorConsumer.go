package errorhandler

import (
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/internal/types/errorx"
)

// PriorityConsumer 根据错误优先级选择不同的处理策略
func PriorityConsumer(err *errorx.ErrorX) error {
	// 从错误元数据中获取优先级
	priorityVal, ok := err.GetMeta(priorityKey)
	if !ok {
		// 没有优先级信息时使用默认优先级
		priorityVal = DefaultPriority
	}

	priority, ok := priorityVal.(ErrorPriority)
	if !ok {
		// 类型转换失败时使用默认优先级
		priority = DefaultPriority
	}

	switch priority {
	case PriorityFatal:
		// 致命错误：详细日志 + 堆栈 + 告警
		logx.Errorf("[fatal error]: %s", err.Detail())
		logx.ErrorStack(err)
		// 致命错误可能需要额外处理，如自动重启服务等
		panic(err)
	case PriorityCritical:
		// 关键错误：详细日志 + 堆栈 + 告警
		logx.Errorf("[critical error]: %s", err.Detail())
		logx.ErrorStack(err)
	case PriorityError:
		// 一般错误：详细日志 + 堆栈
		logx.Errorf("[error]: %s", err.Detail())
		logx.ErrorStack(err)
	case PriorityWarn:
		// 警告：简短日志
		logx.Errorf("[warn]: %s", err.Error())
	case PriorityInfo:
		// 信息：简单日志
		logx.Infof("[info]: %s", err.Error())
	case PriorityDebug:
		// 调试：调试日志
		logx.Debugf("[debug]: %s", err.Error())
	default:
		panic("unhandled default case")
	}

	return nil
}
