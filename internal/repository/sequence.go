package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"shortener/internal/config"
	"shortener/pkg/errorx"
	"strconv"
	"sync"
	"time"
)

// Sequence 定义序列号生成器接口
type Sequence interface {
	// NextID 返回下一个唯一序列ID
	NextID(ctx context.Context) (uint64, error)
}

// NewSequence 创建一个新的序列生成器
func NewSequence(conf config.SequenceConf) Sequence {
	redisConf := redis.RedisConf{
		Host: conf.Redis.Addr,
		Type: conf.Redis.Type,
		Pass: conf.Redis.Password,
	}

	rdb, err := redis.NewRedis(redisConf)
	if err != nil {
		logx.Severef("new sequence failed,err:%v", errorx.NewWithCause(errorx.CodeCacheError, "connect to redis failed", err))
	}

	seq := &sequence{
		batch:            conf.BatchSize,
		threshold:        conf.Threshold,
		db:               sqlx.NewMysql(conf.Mysql.DSN()),
		rdb:              rdb,
		redisOK:          false,
		retryBackoff:     200 * time.Millisecond,
		maxRetries:       3,
		keySequenceID:    conf.KeySequenceID,
		keySequenceState: conf.KeySequenceState,
	}

	seq.preHeatRedisCache()

	return seq
}

type sequence struct {
	db               sqlx.SqlConn
	rdb              *redis.Redis
	batch            uint64
	keySequenceID    string
	keySequenceState string
	threshold        int
	mutex            sync.Mutex
	redisOK          bool
	retryBackoff     time.Duration
	maxRetries       int
}

const (
	updateQuery = `UPDATE sequence SET id = (@current_id := id) + ? WHERE stub = 'a'`
)

const (
	redisOpTimeout      = 500 * time.Millisecond
	redisConnTimeout    = 1 * time.Second
	redisRecoveryPeriod = 5 * time.Second
)

// NextID 生成并返回下一个唯一ID
func (s *sequence) NextID(ctx context.Context) (uint64, error) {
	if s.redisOK {
		logx.Info("get id from redis")
		id, err := s.getIDFromRedisWithRetry(ctx)
		if err == nil {
			return id, nil
		}

		// Redis出错，降级到MySQL
		logx.Errorf("Redis错误，降级到MySQL: %v", err)
		s.redisOK = false
	}

	// 直接从MySQL获取
	logx.Info("get id from mysql")
	id, err := s.nextIDFromMysqlWithBatch(ctx, 1)
	if err != nil {
		return 0, err
	}

	// 后台尝试恢复Redis
	go s.tryRecoverRedis(context.Background())

	return id, nil
}

// getIDFromRedisWithRetry 带重试的从Redis获取ID
func (s *sequence) getIDFromRedisWithRetry(ctx context.Context) (uint64, error) {
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		// 创建一个带超时的上下文
		timeoutCtx, cancel := context.WithTimeout(ctx, redisOpTimeout)

		// 使用上下文
		id, err := s.getIDFromRedis(timeoutCtx)

		cancel()

		if err == nil {
			return id, nil
		}

		lastErr = err
		// 如果是找不到键的错误，不重试
		if errorx.Is(err, errorx.CodeNotFound) {
			break
		}

		// 指数退避重试
		backoff := s.retryBackoff * time.Duration(1<<uint(i))
		time.Sleep(backoff)
	}
	return 0, fmt.Errorf("redis操作失败(重试%d次): %w", s.maxRetries, lastErr)
}

// nextIDFromMysqlWithBatch 获取MySQL中的ID，返回当前��次最小ID
func (s *sequence) nextIDFromMysqlWithBatch(ctx context.Context, batch uint64) (uint64, error) {
	var first uint64
	err := s.db.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 执行 UPDATE 并设置 @current_id
		_, err := tx.ExecCtx(ctx, updateQuery, batch)
		if err != nil {
			return errorx.NewWithCause(errorx.CodeDatabaseError, "更新ID失败", err)
		}

		// 查询更新前的 @current_id 值
		err = tx.QueryRowCtx(ctx, &first, "SELECT @current_id")
		if err != nil {
			return errorx.NewWithCause(errorx.CodeDatabaseError, "获取当前ID失败", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return first + 1, nil // 返回下一个可用ID
}

// preHeatRedisCache 预热Redis缓存
func (s *sequence) preHeatRedisCache() {
	ctx, cancel := context.WithTimeout(context.Background(), redisConnTimeout)
	defer cancel()

	s.redisOK = s.checkRedisConnection(ctx)

	if !s.redisOK {
		logx.Errorf("Redis不可用，使用MySQL生成序列")
		return
	}

	// 检查缓存是否为空
	count, err := s.rdb.LlenCtx(ctx, s.keySequenceID)
	if err != nil {
		logx.Errorf("检查Redis序列长度失败: %v", err)
		s.redisOK = false
		return
	}

	if count == 0 {
		// 首次加载，从MySQL获取批量ID
		lastID, err := s.nextIDFromMysqlWithBatch(ctx, s.batch)
		if err != nil {
			logx.Errorf("从MySQL获取初始批次失败: %v", err)
			s.redisOK = false
			return
		}

		ids := s.generateIDList(lastID, s.batch)
		if err := s.refillRedisSequence(ctx, ids); err != nil {
			logx.Errorf("预填充Redis失败: %v", err)
			s.redisOK = false
		}
	}
}

// getIDFromRedis 从Redis中获取ID，如需要会从MySQL补充
func (s *sequence) getIDFromRedis(ctx context.Context) (uint64, error) {
	id, err := s.nextIDFromRedis(ctx)
	if err == nil {
		// 当剩余ID数量少于阈值时，异步补充
		go s.asyncRefillIfNeeded(context.Background())
		return id, nil
	}

	// Redis数据为空
	if errorx.Is(err, errorx.CodeNotFound) {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		// 再次检查，避免竞态条件
		id, err := s.nextIDFromRedis(ctx)
		if err == nil {
			return id, nil
		}

		// 确实为空，从MySQL批量获取
		lastID, err := s.nextIDFromMysqlWithBatch(ctx, s.batch)
		if err != nil {
			return 0, err
		}

		ids := s.generateIDList(lastID, s.batch)
		if err := s.refillRedisSequence(ctx, ids); err != nil {
			return 0, err
		}

		// 取出第一个ID
		return ids[0], nil
	}

	// 其他Redis错误，返回错误
	return 0, err
}

// nextIDFromRedis 从Redis中取出一个ID
func (s *sequence) nextIDFromRedis(ctx context.Context) (uint64, error) {
	val, err := s.rdb.LpopCtx(ctx, s.keySequenceID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, errorx.New(errorx.CodeNotFound, "Redis中未找到序列")
		}
		return 0, errorx.NewWithCause(errorx.CodeCacheError, "从Redis获取序列失败", err)
	}

	id, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, errorx.NewWithCause(errorx.CodeSystemError, "解析序列ID失败", err)
	}

	return id, nil
}

// asyncRefillIfNeeded 检查并异步补充ID
func (s *sequence) asyncRefillIfNeeded(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, redisOpTimeout)
	defer cancel()

	// 检查剩余ID数量，低于阈值时补充
	count, err := s.rdb.LlenCtx(ctx, s.keySequenceID)
	if err != nil {
		return
	}

	threshold := int(s.batch * uint64(s.threshold) / 100)
	if count >= threshold {
		return // 数量充足，不需要补充
	}

	// 获取锁避免并发补充
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 再次检查，避免并发问题
	count, err = s.rdb.LlenCtx(ctx, s.keySequenceID)
	if err != nil || count >= threshold {
		return
	}

	// 从MySQL批量获取
	lastID, err := s.nextIDFromMysqlWithBatch(ctx, s.batch)
	if err != nil {
		logx.Errorf("从MySQL获取批次失败: %v", err)
		return
	}

	ids := s.generateIDList(lastID, s.batch)
	if err := s.refillRedisSequence(ctx, ids); err != nil {
		logx.Errorf("补充Redis失败: %v", err)
	}
}

// refillRedisSequence 填充Redis序号列表
func (s *sequence) refillRedisSequence(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}

	// 使用pipeline批量写入
	err := s.rdb.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		for _, id := range ids {
			pipe.RPush(ctx, s.keySequenceID, strconv.FormatUint(id, 10))
		}
		// 更新状态，记录最后刷新时间
		pipe.Set(ctx, s.keySequenceState, time.Now().Format(time.RFC3339), 0)
		return nil
	})

	if err != nil {
		return errorx.NewWithCause(errorx.CodeCacheError, "批量写入Redis失败", err)
	}

	logx.Infof("成功补充Redis序列，数量: %d", len(ids))
	return nil
}

// generateIDList 生成ID列表
func (s *sequence) generateIDList(startID uint64, count uint64) []uint64 {
	ids := make([]uint64, count)
	for i := uint64(0); i < count; i++ {
		ids[i] = startID + i
	}
	return ids
}

// checkRedisConnection 检查Redis连接状态
func (s *sequence) checkRedisConnection(ctx context.Context) bool {
	return s.rdb.PingCtx(ctx)
}

// tryRecoverRedis 尝试恢复Redis连接
func (s *sequence) tryRecoverRedis(ctx context.Context) {
	// 已经恢复则不处理
	if s.redisOK {
		return
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, redisConnTimeout)
	defer cancel()

	// 检查连接
	if s.checkRedisConnection(ctx) {
		s.redisOK = true
		logx.Info("Redis连接已恢复")

		// 恢复后尝试预热缓存
		s.preHeatRedisCache()
	} else {
		// 定期重试直到恢复
		time.AfterFunc(redisRecoveryPeriod, func() {
			s.tryRecoverRedis(context.Background())
		})
	}
}
