package response

import (
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"shortener/internal/types/errorx"
)

const (
	systemErrorMsg   = "an error occurred inside the system"
	databaseErrorMsg = "an error occurred inside the database"
	cacheErrorMsg    = "an error occurred inside the cache"
	logicErrorMsg    = "user business logic error"
)

// Response 统一响应结构
type Response struct {
	Msg  string      `json:"msg"`
	Code int         `json:"code"`
	Data interface{} `json:"data,omitempty"` // 为空时不输出
}

// Success 处理成功响应
func Success(w http.ResponseWriter, resp any) {
	result := Response{
		Msg:  "success",
		Code: int(errorx.CodeSuccess),
		Data: resp,
	}
	httpx.WriteJson(w, http.StatusOK, result)
}

// Error 处理错误响应
func Error(w http.ResponseWriter, err error) {
	processedErr := handleError(err)

	var code errorx.Code
	var msg string

	// 从错误中提取错误码和消息
	var ex *errorx.ErrorX
	if errors.As(processedErr, &ex) {
		code = ex.Code
		msg = ex.Msg
	}

	result := Response{
		Msg:  msg,
		Code: int(code),
	}

	// 使用适当的HTTP状态码
	httpStatus := errorx.ToHTTPStatus(code)
	httpx.WriteJson(w, httpStatus, result)
}

func handleError(err error) error {
	var targetError *errorx.ErrorX

	if errors.As(err, &targetError) {
		// 记录详细错误信息
		switch targetError.Code {
		// 保留原始错误信息和代码，但避免将敏感信息暴露给客户端
		case errorx.CodeSystemError, errorx.CodeDatabaseError, errorx.CodeCacheError:
			// 系统级错误使用通用消息
			logx.Errorw(systemErrorMsg, logx.Field("err", targetError.Detail()))
			return errorx.New(targetError.Code, getPublicErrorMessage(targetError.Code))
		case errorx.CodeParamError, errorx.CodeNotFound, errorx.CodeServiceUnavailable, errorx.CodeTimeout, errorx.CodeTooFrequent:
			logx.Debugw(logicErrorMsg, logx.Field("msg", targetError.Msg))
			return errorx.New(targetError.Code, targetError.Msg)
		default:
			logx.Errorw("invalid error code", logx.Field("err", targetError.Detail()))
			return errorx.New(errorx.CodeSystemError, systemErrorMsg)
		}
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
