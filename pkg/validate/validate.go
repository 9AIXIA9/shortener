package validate

import (
	"context"
	"github.com/go-playground/validator/v10"
	"shortener/internal/types/errorx"
	"sync"
)

var (
	instance *validator.Validate
	once     sync.Once
)

// Check 进行检验
func Check(ctx context.Context, param interface{}) error {
	err := initInstance()
	if err != nil {
		return err
	}

	err = instance.StructCtx(ctx, param)
	if err != nil {
		return errorx.NewWithCause(errorx.CodeParamError, "invalid param", err)
	}

	return nil
}

func initInstance() (err error) {
	once.Do(func() {
		instance = validator.New()

		// 注册自定义验证器
		err = instance.RegisterValidation("validLongUrl", validLongUrlValidator)
		if err != nil {
			err = errorx.NewWithCause(errorx.CodeSystemError, "RegisterValidation validLongUrl failed", err)
		}

		err = instance.RegisterValidation("validShortUrl", validShortUrlValidator)
		if err != nil {
			err = errorx.NewWithCause(errorx.CodeSystemError, "RegisterValidation validShortUrl failed", err)
		}
	})

	return err
}
