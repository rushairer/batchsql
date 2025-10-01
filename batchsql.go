package batchsql

import (
	"context"
	"database/sql"
	"time"

	redisV9 "github.com/redis/go-redis/v9"
	gopipeline "github.com/rushairer/go-pipeline/v2"
)

// BatchSQL 批量处理管道
// 核心组件，整合 go-pipeline 和 BatchExecutor，提供统一的批量处理接口
//
// 架构层次：
// Application -> BatchSQL -> gopipeline -> BatchExecutor -> Database
//
// 支持的BatchExecutor实现：
// - SQL数据库：CommonExecutor + BatchProcessor + SQLDriver
// - NoSQL数据库：直接实现BatchExecutor（如Redis）
// - 测试环境：MockExecutor
type BatchSQL struct {
	pipeline *gopipeline.StandardPipeline[*Request] // 异步批量处理管道
	executor BatchExecutor                          // 批量执行器（数据库特定）
}

// NewBatchSQL 创建 BatchSQL 实例
// 这是最底层的构造函数，接受任何实现了BatchExecutor接口的执行器
// 通常不直接使用，而是通过具体数据库的工厂方法创建
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
			// 在开始耗时操作前快速检查
			if err := ctx.Err(); err != nil {
				return err
			}

			// 转换为数据格式
			data := make([]map[string]any, len(requests))
			for i, request := range requests {
				// 如果单个schema的数据量很大，可以定期检查
				if len(requests) > 10000 && i%1000 == 0 {
					if err := ctx.Err(); err != nil {
						return err
					}
				}
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
// 内部架构：BatchSQL -> CommonExecutor -> SQLBatchProcessor -> MySQLDriver -> MySQL
// 这是推荐的使用方式，使用MySQL优化的默认配置
func NewMySQLBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, DefaultMySQLDriver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewMySQLBatchSQLWithDriver 创建MySQL BatchSQL实例（使用自定义Driver）
// 内部架构：BatchSQL -> CommonExecutor -> SQLBatchProcessor -> CustomDriver -> MySQL
// 适用于需要自定义SQL生成逻辑的场景（如TiDB优化）
func NewMySQLBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, driver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewPostgreSQLBatchSQL 创建PostgreSQL BatchSQL实例（使用默认Driver）
func NewPostgreSQLBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, DefaultPostgreSQLDriver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewPostgreSQLBatchSQLWithDriver 创建PostgreSQL BatchSQL实例（使用自定义Driver）
func NewPostgreSQLBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, driver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewSQLiteBatchSQL 创建SQLite BatchSQL实例（使用默认Driver）
func NewSQLiteBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, DefaultSQLiteDriver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewSQLiteBatchSQLWithDriver 创建SQLite BatchSQL实例（使用自定义Driver）
func NewSQLiteBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, driver)
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewRedisBatchSQL 创建Redis BatchSQL实例
// 内部架构：BatchSQL -> RedisExecutor -> Redis Client（直接实现，无BatchProcessor层）
// Redis作为NoSQL数据库，跳过SQL相关的抽象层，直接实现BatchExecutor接口
func NewRedisBatchSQL(ctx context.Context, db *redisV9.Client, config PipelineConfig) *BatchSQL {
	executor := NewThrottledBatchExecutor(NewRedisBatchProcessor(db, DefaultRedisPipelineDriver))
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

func NewRedisBatchSQLWithDriver(ctx context.Context, db *redisV9.Client, config PipelineConfig, driver RedisDriver) *BatchSQL {
	executor := NewThrottledBatchExecutor(NewRedisBatchProcessor(db, driver))
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewBatchSQLWithMock 使用模拟执行器创建 BatchSQL 实例（用于测试）
// 内部架构：BatchSQL -> MockExecutor（直接实现BatchExecutor，无真实数据库操作）
// 适用于单元测试，不依赖真实数据库连接
func NewBatchSQLWithMock(ctx context.Context, config PipelineConfig) (*BatchSQL, *MockExecutor) {
	mockExecutor := NewMockExecutor()
	batchSQL := NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, mockExecutor)
	return batchSQL, mockExecutor
}

// NewBatchSQLWithMockDriver 使用模拟执行器创建 BatchSQL 实例（测试特定SQLDriver）
// 内部架构：BatchSQL -> MockExecutor（模拟CommonExecutor行为，测试SQLDriver逻辑）
// 适用于测试自定义SQLDriver的SQL生成逻辑
func NewBatchSQLWithMockDriver(ctx context.Context, config PipelineConfig, sqlDriver SQLDriver) (*BatchSQL, *MockExecutor) {
	mockExecutor := NewMockExecutorWithDriver(sqlDriver)
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
		return ctx.Err()
	}
}
