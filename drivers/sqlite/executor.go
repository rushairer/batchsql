package sqlite

import (
	"context"
	"database/sql"

	"github.com/rushairer/batchsql"
)

// BatchExecutor SQLite批量执行器
type BatchExecutor struct {
	processor *batchsql.BatchProcessor
}

// NewBatchExecutor 创建SQLite批量执行器（使用默认Driver）
func NewBatchExecutor(db *sql.DB) *BatchExecutor {
	return &BatchExecutor{
		processor: batchsql.NewBatchProcessor(db, DefaultDriver),
	}
}

// NewBatchExecutorWithDriver 创建SQLite批量执行器（使用自定义Driver）
func NewBatchExecutorWithDriver(db *sql.DB, driver batchsql.SQLDriver) *BatchExecutor {
	return &BatchExecutor{
		processor: batchsql.NewBatchProcessor(db, driver),
	}
}

// ExecuteBatch 执行批量操作
func (e *BatchExecutor) ExecuteBatch(ctx context.Context, batchData []*batchsql.Request) error {
	return e.processor.ProcessBatch(ctx, batchData)
}
