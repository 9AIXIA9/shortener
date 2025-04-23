//go:generate mockgen -source=$GOFILE -destination=./mock/cache_mock.go -package=cachex

package cachex

import "context"

type SequenceCache interface {
	GetSingleID(ctx context.Context) (uint64, error)
	FillIDs(ctx context.Context, ids []uint64) error
	IsOK(ctx context.Context) bool
	IsLessThanThreshold(ctx context.Context, threshold int) (bool, error)
}
