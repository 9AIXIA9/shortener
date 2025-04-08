package handler

import (
	"github.com/zeromicro/go-zero/core/logx"
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
		if err := validate.Get().StructCtx(r.Context(), &req); err != nil {
			logx.Infow("validator check failed", logx.Field("err", err))
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewShowLogic(r.Context(), svcCtx)
		resp, err := l.Show(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			//httpx.OkJsonCtx(r.Context(), w, resp)
			http.Redirect(w, r, resp.LongUrl, http.StatusFound)
		}
	}
}
