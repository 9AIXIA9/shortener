package handler

import (
	"errors"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"shortener/pkg/errorx"
)

// Response 统一响应结构
type Response struct {
	Msg  string      `json:"msg"`
	Code int         `json:"code"`
	Data interface{} `json:"data,omitempty"` // 为空时不输出
}

// ResponseSuccess 处理成功响应
func ResponseSuccess(w http.ResponseWriter, resp any) {
	result := Response{
		Msg:  "success",
		Code: int(errorx.CodeSuccess),
		Data: resp,
	}
	httpx.WriteJson(w, http.StatusOK, result)
}

// ResponseError 处理错误响应
func ResponseError(w http.ResponseWriter, err error) {
	processedErr := HandleError(err)

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
