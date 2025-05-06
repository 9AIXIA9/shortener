package handler

import (
	"net/http"
	"shortener/internal/logic"
	"shortener/internal/types/format"
	"shortener/pkg/validate"

	"github.com/zeromicro/go-zero/rest/httpx"
	"shortener/internal/svc"
	"shortener/internal/types"
)

func ResolveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResolveRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		//参数校验
		if err := validate.Check(r.Context(), &req); err != nil {
			format.ResponseError(w, err)
			return
		}

		l := logic.NewResolveLogic(r.Context(), svcCtx)
		resp, err := l.Resolve(&req)
		if err != nil {
			format.ResponseError(w, err)
		} else {
			http.Redirect(w, r, resp.OriginalUrl, http.StatusFound)
		}
	}
}
