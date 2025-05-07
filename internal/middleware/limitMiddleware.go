package middleware

import (
	"net/http"
	"shortener/internal/types/errorx"
	"shortener/internal/types/response"
	"shortener/pkg/limit"
)

type LimitMiddleware struct {
	limit limit.Limit
}

func NewLimitMiddleware(limit limit.Limit) *LimitMiddleware {
	return &LimitMiddleware{limit: limit}
}

func (m *LimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 检查限流器是否允许请求
		if !m.limit.Allow() {
			response.Error(w, errorx.New(errorx.CodeTooFrequent, "requests are too frequent"))
			return
		}

		// 继续处理下一个处理器
		next(w, r)
	}
}
