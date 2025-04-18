package handler

import (
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"shortener/internal/logic"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/pkg/urlTool"
	"shortener/pkg/validate"
)

func ShortenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ShortenRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		//参数校验
		if err := validate.Check(r.Context(), &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, LogError(err))
			return
		}

		l := logic.NewShortenLogic(r.Context(), svcCtx, urlTool.NewClient())
		resp, err := l.Shorten(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, LogError(err))
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
