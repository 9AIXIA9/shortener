package errorx

import "github.com/zeromicro/go-zero/core/logx"

// 用于判断日志层级
type logLevel uint8

const (
	Error = +iota
	Info
)

// Log 记录错误日志并返回错误
func Log(level logLevel, code Code, msg string, fields ...logx.LogField) error {
	switch level {
	case Error:
		logx.Errorw(msg, fields...)
	case Info:
		logx.Infow(msg, fields...)
	default:
		logx.Errorw("invalid error code")
	}
	return New(code, msg)
}
