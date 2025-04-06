package repository

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type Sequence interface {
	Next(ctx context.Context) (uint64, error)
}

func NewSequence(dsn string) Sequence {
	return sequence{
		conn: sqlx.NewMysql(dsn),
	}
}

type sequence struct {
	conn sqlx.SqlConn
}

const replaceIntoStub = `REPLACE INTO sequence (stub) VALUES ('a')`

func (s sequence) Next(ctx context.Context) (uint64, error) {
	//预处理
	stmt, err := s.conn.PrepareCtx(ctx, replaceIntoStub)
	if err != nil {
		return 0, fmt.Errorf("sequence.Next() conn.PrepareCtx failed,err:%w", err)
	}
	defer func() {
		if err = stmt.Close(); err != nil {
			logx.Errorw("sequence.Next() conn.PrepareCtx.stmt close failed", logx.Field("err", err))
		}
	}()

	//执行
	result, err := stmt.ExecCtx(ctx)
	if err != nil {
		return 0, fmt.Errorf("sequence Next() stmt.ExecCtx failed,err:%w", err)
	}

	//获取结果
	lastID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sequence Next() get result.LastInsertId() failed,err:%w", err)
	}

	return uint64(lastID), nil
}
