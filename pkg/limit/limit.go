//go:generate mockgen -source=$GOFILE -destination=./mock/limit_mock.go -package=limit
package limit

import (
	"context"
	"time"
)

type Limit interface {
	Allow() bool
	AllowCtx(ctx context.Context) bool
	AllowN(now time.Time, n int) bool
	AllowNCtx(ctx context.Context, now time.Time, n int) bool
}
