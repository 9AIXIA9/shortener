package errorx

import "github.com/zeromicro/go-zero/core/logx"

// 用于判断日志层级
type logLevel uint8

// 日志层级
const (
	ErrorLevel = +iota
	DebugLevel
)

// 返回错误信息
const (
	internalErrorMsg = "There is an error inside the server"
)

// Log 记录错误日志并返回错误
func Log(level logLevel, code Code, msg string, fields ...logx.LogField) error {
	switch level {
	case ErrorLevel:
		logx.Errorw(msg, fields...)
		//避免直接暴露系统错误
		return New(code, internalErrorMsg)
	case DebugLevel:
		logx.Debugw(msg, fields...)
		//给出信息debug
		return New(code, msg)
	default:
		logx.Errorw("invalid error code")
		return New(code, internalErrorMsg)
	}
}
