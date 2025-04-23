//go:generate mockgen -source=$GOFILE -destination=./mock/database_mock.go -package=database

package database

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/pkg/errorx"
)

const (
	updateQuery  = `UPDATE sequence SET id = (@current_id := id) + ? WHERE stub = 'a'`
	currentQuery = `SELECT @current_id`
)

type SequenceDatabase interface {
	GetBatchIDs(ctx context.Context, batch uint64) ([]uint64, error)
}

func NewMysqlSequenceDatabase(conn sqlx.SqlConn) SequenceDatabase {
	return &sequence{db: conn}
}

type sequence struct {
	db sqlx.SqlConn
}

func (s *sequence) GetBatchIDs(ctx context.Context, batch uint64) ([]uint64, error) {
	var first uint64
	err := s.db.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		// Execute UPDATE and set @current_id
		_, err := tx.ExecCtx(ctx, updateQuery, batch)
		if err != nil {
			return errorx.Wrap(err, errorx.CodeDatabaseError, "failed to update ID")
		}

		// Query the previous @current_id value
		err = tx.QueryRowCtx(ctx, &first, currentQuery)
		if err != nil {
			return errorx.Wrap(err, errorx.CodeDatabaseError, "failed to get current ID")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.generateIDList(first, batch), nil // Return the next available ID
}

// generateIDList generates ID list
func (s *sequence) generateIDList(startID uint64, count uint64) []uint64 {
	if count <= 0 {
		return nil
	}

	ids := make([]uint64, count)
	for i := uint64(0); i < count; i++ {
		ids[i] = startID + i
	}
	return ids
}
