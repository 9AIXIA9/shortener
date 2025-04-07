package shortener

import (
	"github.com/AIXIA/shortener/pkg/connect"
	"github.com/AIXIA/shortener/pkg/validate"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"

	"github.com/AIXIA/shortener/internal/logic/shortener"
	"github.com/AIXIA/shortener/internal/svc"
	"github.com/AIXIA/shortener/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ConvertHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ConvertRequest
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

		l := shortener.NewConvertLogic(r.Context(), svcCtx, connect.NewClient())
		resp, err := l.Convert(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
