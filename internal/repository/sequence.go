//go:generate mockgen -source=$GOFILE -destination=./mock/sequence_mock.go -package=repository
package repository

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/pkg/errorx"
)

// Sequence 定义序列号生成器接口
type Sequence interface {
	// NextID 返回下一个唯一序列ID
	NextID(ctx context.Context) (uint64, error)
}

// NewSequence 创建一个新的序列生成器
func NewSequence(dsn string) Sequence {
	return &sequence{
		db: sqlx.NewMysql(dsn),
	}
}

type sequence struct {
	db sqlx.SqlConn
}

const replaceIntoStub = `REPLACE INTO sequence (stub) VALUES ('a')`

// NextID 生成并返回下一个唯一ID
func (s *sequence) NextID(ctx context.Context) (uint64, error) {
	stmt, err := s.db.PrepareCtx(ctx, replaceIntoStub)
	if err != nil {
		return 0, errorx.NewWithCause(errorx.CodeDatabaseError, "prepare statement failed", err).
			WithContext(ctx)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			logx.Errorf("close statement failed: %v",
				errorx.NewWithCause(errorx.CodeDatabaseError, "close statement failed", err).WithContext(ctx))
		}
	}()

	result, err := stmt.ExecCtx(ctx)
	if err != nil {
		return 0, errorx.NewWithCause(errorx.CodeDatabaseError, "execute statement failed", err).
			WithContext(ctx)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return 0, errorx.NewWithCause(errorx.CodeDatabaseError, "get last sequence ID failed", err).
			WithContext(ctx)
	}

	return uint64(lastID), nil
}
