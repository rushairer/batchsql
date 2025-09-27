package batchsql

import (
	"context"
	"database/sql"
	"time"

	"github.com/rushairer/batchsql/drivers/mock"
	"github.com/rushairer/batchsql/drivers/mysql"
	"github.com/rushairer/batchsql/drivers/postgresql"
	"github.com/rushairer/batchsql/drivers/sqlite"
	gopipeline "github.com/rushairer/go-pipeline/v2"
)

// BatchSQL 批量处理管道
type BatchSQL struct {
	pipeline *gopipeline.StandardPipeline[*Request]
	executor BatchExecutor
}

// NewBatchSQL 创建 BatchSQL 实例
func NewBatchSQL(ctx context.Context, buffSize uint32, flushSize uint32, flushInterval time.Duration, executor BatchExecutor) *BatchSQL {
	batchSQL := &BatchSQL{
		executor: executor,
	}

	// 创建 flush 函数，使用批量执行器处理数据
	flushFunc := func(ctx context.Context, batchData []*Request) error {
		// 按schema分组处理
		schemaGroups := make(map[*Schema][]*Request)
		for _, request := range batchData {
			schema := request.Schema()
			schemaGroups[schema] = append(schemaGroups[schema], request)
		}

		// 处理每个schema组
		for schema, requests := range schemaGroups {
			// 转换为数据格式
			data := make([]map[string]any, len(requests))
			for i, request := range requests {
				rowData := make(map[string]any)
				values := request.GetOrderedValues()
				columns := schema.Columns

				for j, col := range columns {
					if j < len(values) {
						rowData[col] = values[j]
					}
				}
				data[i] = rowData
			}

			// 执行批量操作
			if err := batchSQL.executor.ExecuteBatch(ctx, schema, data); err != nil {
				return err
			}
		}
		return nil
	}

	pipeline := gopipeline.NewStandardPipeline(
		gopipeline.PipelineConfig{
			BufferSize:    buffSize,
			FlushSize:     flushSize,
			FlushInterval: flushInterval,
		},
		flushFunc,
	)

	batchSQL.pipeline = pipeline
	go func() {
		_ = pipeline.AsyncPerform(ctx)
	}()

	return batchSQL
}

// PipelineConfig 管道配置
type PipelineConfig struct {
	BufferSize    uint32
	FlushSize     uint32
	FlushInterval time.Duration
}

// NewMySQLBatchSQL 创建MySQL BatchSQL实例（使用默认Driver）
func NewMySQLBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := mysql.NewBatchExecutor(db)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewMySQLBatchSQLWithDriver 创建MySQL BatchSQL实例（使用自定义Driver）
func NewMySQLBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := mysql.NewBatchExecutorWithDriver(db, driver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewPostgreSQLBatchSQL 创建PostgreSQL BatchSQL实例（使用默认Driver）
func NewPostgreSQLBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := postgresql.NewBatchExecutor(db)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewPostgreSQLBatchSQLWithDriver 创建PostgreSQL BatchSQL实例（使用自定义Driver）
func NewPostgreSQLBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := postgresql.NewBatchExecutorWithDriver(db, driver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewSQLiteBatchSQL 创建SQLite BatchSQL实例（使用默认Driver）
func NewSQLiteBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := sqlite.NewBatchExecutor(db)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewSQLiteBatchSQLWithDriver 创建SQLite BatchSQL实例（使用自定义Driver）
func NewSQLiteBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := sqlite.NewBatchExecutorWithDriver(db, driver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewBatchSQLWithMock 使用模拟执行器创建 BatchSQL 实例（用于测试，使用默认MySQL Driver）
func NewBatchSQLWithMock(ctx context.Context, config PipelineConfig) (*BatchSQL, *mock.Executor) {
	mockExecutor := mock.NewBatchExecutor()
	batchSQL := NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, mockExecutor)
	return batchSQL, mockExecutor
}

// NewBatchSQLWithMockDriver 使用模拟执行器创建 BatchSQL 实例（用于测试，使用自定义Driver）
func NewBatchSQLWithMockDriver(ctx context.Context, config PipelineConfig, sqlDriver SQLDriver) (*BatchSQL, *mock.Executor) {
	mockExecutor := mock.NewBatchExecutorWithDriver(sqlDriver)
	batchSQL := NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, mockExecutor)
	return batchSQL, mockExecutor
}

// ErrorChan 获取错误通道
func (b *BatchSQL) ErrorChan(size int) <-chan error {
	return b.pipeline.ErrorChan(size)
}

// Submit 提交请求到批量处理管道
func (b *BatchSQL) Submit(ctx context.Context, request *Request) error {
	dataChan := b.pipeline.DataChan()

	select {
	case dataChan <- request:
		return nil
	case <-ctx.Done():
		return ErrContextCanceled
	}
}

// Close 关闭批量处理管道
func (b *BatchSQL) Close() error {
	// gopipeline.StandardPipeline 可能没有 Close 方法
	// 这里可以添加其他清理逻辑
	return nil
}
