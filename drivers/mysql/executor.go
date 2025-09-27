package mysql

import (
	"context"
	"database/sql"

	"github.com/rushairer/batchsql/drivers"
)

// BatchProcessor 批量处理器
type BatchProcessor struct {
	db        *sql.DB
	sqlDriver drivers.SQLDriver
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(db *sql.DB, sqlDriver drivers.SQLDriver) *BatchProcessor {
	return &BatchProcessor{
		db:        db,
		sqlDriver: sqlDriver,
	}
}

// ExecuteBatch 执行批量操作
func (bp *BatchProcessor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// 使用SQLDriver生成批量插入SQL
	sql, args, err := bp.sqlDriver.GenerateInsertSQL(schema, data)
	if err != nil {
		return err
	}

	// 执行 SQL
	_, err = bp.db.ExecContext(ctx, sql, args...)
	return err
}

// Executor MySQL批量执行器
type Executor struct {
	processor *BatchProcessor
}

// NewBatchExecutor 创建MySQL批量执行器（使用默认Driver）
func NewBatchExecutor(db *sql.DB) *Executor {
	return &Executor{
		processor: NewBatchProcessor(db, DefaultDriver),
	}
}

// NewBatchExecutorWithDriver 创建MySQL批量执行器（使用自定义Driver）
func NewBatchExecutorWithDriver(db *sql.DB, driver drivers.SQLDriver) *Executor {
	return &Executor{
		processor: NewBatchProcessor(db, driver),
	}
}

// ExecuteBatch 执行批量操作
func (e *Executor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
	return e.processor.ExecuteBatch(ctx, schema, data)
}
