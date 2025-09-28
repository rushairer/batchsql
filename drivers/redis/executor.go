package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rushairer/batchsql/drivers"
)

// BatchProcessor 批量处理器
type BatchProcessor struct {
	db *redis.Client
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(db *redis.Client) *BatchProcessor {
	return &BatchProcessor{
		db: db,
	}
}

// ExecuteBatch 执行批量操作
func (bp *BatchProcessor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// 使用Redis的Pipeline特性实现批量插入
	pipeline := bp.db.Pipeline()
	for _, row := range data {
		key := row[schema.Columns[0]].(string)
		value := row[schema.Columns[1]]
		pipeline.Do(ctx, key, value)
		if ttlInt, ok := row[schema.Columns[2]].(int64); ok && ttlInt > 0 {
			ttl := time.Duration(ttlInt) * time.Millisecond
			pipeline.Expire(ctx, key, ttl)
		}
	}
	_, err := pipeline.Exec(ctx)
	return err
}

// Executor Redis批量执行器
type Executor struct {
	processor *BatchProcessor
}

// NewBatchExecutor 创建Redis批量执行器
func NewBatchExecutor(db *redis.Client) *Executor {
	return &Executor{
		processor: NewBatchProcessor(db),
	}
}

// ExecuteBatch 执行批量操作
func (e *Executor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
	return e.processor.ExecuteBatch(ctx, schema, data)
}
