package handler

import (
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"shortener/pkg/errorx"
)

const (
	systemErrorMsg   = "an error occurred inside the system"
	databaseErrorMsg = "an error occurred inside the database"
	cacheErrorMsg    = "an error occurred inside the cache"
	logicErrorMsg    = "user business logic error"
)

func LogError(err error) error {
	var targetError *errorx.ErrorX

	if errors.As(err, &targetError) {
		// 记录详细错误信息
		switch {
		// 保留原始错误信息和代码，但避免将敏感信息暴露给客户端
		case (targetError.Code) <= errorx.CodeSuccess && (targetError.Code) >= errorx.CodeCacheError:
			// 系统级错误使用通用消息
			logx.Errorw(systemErrorMsg, logx.Field("err", targetError.Detail()))
			return errorx.New(targetError.Code, getPublicErrorMessage(targetError.Code))
		case (targetError.Code) >= errorx.CodeParamError && (targetError.Code) <= errorx.CodeServiceUnavailable:
			logx.Debugw(logicErrorMsg, logx.Field("msg", targetError.Msg))
			return errorx.New(targetError.Code, getPublicErrorMessage(targetError.Code))
		default:
			logx.Errorw("invalid error code", logx.Field("err", targetError.Detail()))
		}

		// 业务错误保留原始信息
		return targetError
	}

	// 非ErrorX类型
	logx.Errorf("unrecognized error:%v", err)
	return errorx.New(errorx.CodeSystemError, systemErrorMsg)
}

func getPublicErrorMessage(code errorx.Code) string {
	// 返回适合展示给用户的错误信息
	switch code {
	case errorx.CodeSystemError:
		return systemErrorMsg
	case errorx.CodeDatabaseError:
		return databaseErrorMsg
	case errorx.CodeCacheError:
		return cacheErrorMsg
	default:
		return systemErrorMsg
	}
}
