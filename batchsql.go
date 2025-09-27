package batchsql

import (
	"context"
	"database/sql"
	"time"

	gopipeline "github.com/rushairer/go-pipeline/v2"
)

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
		return batchSQL.executor.ExecuteBatch(ctx, batchData)
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
	go pipeline.AsyncPerform(ctx)

	return batchSQL
}

// NewBatchSQLWithDB 使用数据库连接创建 BatchSQL 实例
func NewBatchSQLWithDB(ctx context.Context, db *sql.DB, buffSize uint32, flushSize uint32, flushInterval time.Duration) *BatchSQL {
	executor := NewDatabaseBatchExecutor(db)
	return NewBatchSQL(ctx, buffSize, flushSize, flushInterval, executor)
}

// NewBatchSQLWithMock 使用模拟执行器创建 BatchSQL 实例（用于测试）
func NewBatchSQLWithMock(ctx context.Context, buffSize uint32, flushSize uint32, flushInterval time.Duration) (*BatchSQL, *MockBatchExecutor) {
	mockExecutor := NewMockBatchExecutor()
	batchSQL := NewBatchSQL(ctx, buffSize, flushSize, flushInterval, mockExecutor)
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
