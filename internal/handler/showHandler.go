package handler

import (
	"net/http"
	"shortener/internal/logic"
	"shortener/pkg/validate"

	"github.com/zeromicro/go-zero/rest/httpx"
	"shortener/internal/svc"
	"shortener/internal/types"
)

func ShowHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ShowRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		//参数校验
		if err := validate.Check(r.Context(), &req); err != nil {
			ResponseError(w, err)
			return
		}

		l := logic.NewShowLogic(r.Context(), svcCtx)
		resp, err := l.Show(&req)
		if err != nil {
			ResponseError(w, err)
		} else {
			http.Redirect(w, r, resp.LongUrl, http.StatusFound)
		}
	}
}
