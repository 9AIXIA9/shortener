package handler

import (
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"shortener/internal/logic"
	"shortener/internal/svc"
	"shortener/internal/types"
	"shortener/internal/types/format"
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
			format.ResponseError(w, err)
			return
		}

		l := logic.NewShortenLogic(r.Context(), svcCtx, urlTool.NewClient(svcCtx.Config.Connect))
		resp, err := l.Shorten(&req)
		if err != nil {
			format.ResponseError(w, err)
		} else {
			format.ResponseSuccess(w, resp)
		}
	}
}
