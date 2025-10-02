package batchsql

import (
	"context"
	"database/sql"
	"sync/atomic"
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
/*
支持的 BatchExecutor 实现：
- SQL 数据库：ThrottledBatchExecutor + SQLBatchProcessor + SQLDriver
- NoSQL 数据库：ThrottledBatchExecutor + RedisBatchProcessor + RedisDriver（直接生成/执行命令）
- 测试环境：MockExecutor（直接实现 BatchExecutor）
可选能力：
- WithConcurrencyLimit：通过信号量限制 ExecuteBatch 并发，避免攒批后同时冲击数据库（limit <= 0 等价于不限流）
*/
type BatchSQL struct {
	pipeline        *gopipeline.StandardPipeline[*Request] // 异步批量处理管道
	executor        BatchExecutor                          // 批量执行器（数据库特定）
	metricsReporter MetricsReporter                        // 指标上报器（默认 Noop）
	closed          atomic.Bool                            // 当创建时上下文被取消后置为 true，拒绝后续提交
}

// NewBatchSQL 创建 BatchSQL 实例
// 这是最底层的构造函数，接受任何实现了BatchExecutor接口的执行器
// 通常不直接使用，而是通过具体数据库的工厂方法创建
func NewBatchSQL(ctx context.Context, buffSize uint32, flushSize uint32, flushInterval time.Duration, executor BatchExecutor) *BatchSQL {
	// 确保 BatchSQL 始终拥有可用 reporter，但不误覆盖自定义执行器的已有配置
	var reporter MetricsReporter
	if mp, ok := executor.(MetricsProvider); ok {
		// 已实现可选能力接口，可安全探测
		if mp.MetricsReporter() != nil {
			reporter = mp.MetricsReporter()
		} else {
			// 明确为空时才注入 Noop，并回写到执行器中
			reporter = NewNoopMetricsReporter()
			executor = executor.WithMetricsReporter(reporter)
		}
	} else {
		// 未实现可选接口：不覆盖对方内部配置，仅在 BatchSQL 内部使用本地 Noop
		reporter = NewNoopMetricsReporter()
		// 注意：不调用 executor.WithMetricsReporter(reporter)
	}

	batchSQL := &BatchSQL{
		executor:        executor,
		metricsReporter: reporter,
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
			assembleStart := time.Now()
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

			// 组装完成指标（批大小 + 组装耗时）
			batchSQL.metricsReporter.ObserveBatchSize(len(requests))
			batchSQL.metricsReporter.ObserveBatchAssemble(time.Since(assembleStart))

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
	// 标记管道生命周期：创建时 ctx 一旦取消，后续 Submit 均应拒绝
	go func() {
		<-ctx.Done()
		batchSQL.closed.Store(true)
	}()

	return batchSQL
}

// PipelineConfig 管道配置
type PipelineConfig struct {
	BufferSize    uint32
	FlushSize     uint32
	FlushInterval time.Duration

	// Step 2: 可选重试配置（零值=关闭，向后兼容）
	Retry RetryConfig
}

// NewMySQLBatchSQL 创建MySQL BatchSQL实例（使用默认Driver）
/*
内部架构：BatchSQL -> ThrottledBatchExecutor -> SQLBatchProcessor -> MySQLDriver -> MySQL
*/
// 这是推荐的使用方式，使用MySQL优化的默认配置
func NewMySQLBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, DefaultMySQLDriver)
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewMySQLBatchSQLWithDriver 创建MySQL BatchSQL实例（使用自定义Driver）
/*
内部架构：BatchSQL -> ThrottledBatchExecutor -> SQLBatchProcessor -> CustomDriver -> MySQL
*/
// 适用于需要自定义SQL生成逻辑的场景（如TiDB优化）
func NewMySQLBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, driver)
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewPostgreSQLBatchSQL 创建PostgreSQL BatchSQL实例（使用默认Driver）
func NewPostgreSQLBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, DefaultPostgreSQLDriver)
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewPostgreSQLBatchSQLWithDriver 创建PostgreSQL BatchSQL实例（使用自定义Driver）
func NewPostgreSQLBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, driver)
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewSQLiteBatchSQL 创建SQLite BatchSQL实例（使用默认Driver）
func NewSQLiteBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, DefaultSQLiteDriver)
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewSQLiteBatchSQLWithDriver 创建SQLite BatchSQL实例（使用自定义Driver）
func NewSQLiteBatchSQLWithDriver(ctx context.Context, db *sql.DB, config PipelineConfig, driver SQLDriver) *BatchSQL {
	executor := NewSQLThrottledBatchExecutorWithDriver(db, driver)
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// NewRedisBatchSQL 创建Redis BatchSQL实例
/*
内部架构（NoSQL）：BatchSQL -> ThrottledBatchExecutor -> RedisBatchProcessor -> RedisDriver -> Redis
说明：NoSQL 路径不使用 SQL 抽象层，直接生成并执行 Redis 命令；仍可启用 WithConcurrencyLimit 控制批次并发。
*/
func NewRedisBatchSQL(ctx context.Context, db *redisV9.Client, config PipelineConfig) *BatchSQL {
	executor := NewThrottledBatchExecutor(NewRedisBatchProcessor(db, DefaultRedisPipelineDriver))
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
	return NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

func NewRedisBatchSQLWithDriver(ctx context.Context, db *redisV9.Client, config PipelineConfig, driver RedisDriver) *BatchSQL {
	executor := NewThrottledBatchExecutor(NewRedisBatchProcessor(db, driver))
	if config.Retry.Enabled {
		executor.WithRetryConfig(config.Retry)
	}
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
	// 优先尊重取消，避免 select 在多就绪时随机选择发送路径
	if err := ctx.Err(); err != nil {
		return err
	}
	// 若 BatchSQL 所属生命周期已结束（创建时的 ctx 已取消），直接拒绝提交
	if b.closed.Load() {
		return context.Canceled
	}

	if request == nil {
		return ErrEmptyRequest
	}

	schema := request.Schema()
	if schema == nil {
		return ErrInvalidSchema
	}
	if schema.Columns == nil {
		return ErrMissingColumn
	}
	if len(schema.Name) == 0 {
		return ErrEmptySchemaName
	}

	dataChan := b.pipeline.DataChan()
	enqueueStart := time.Now()

	select {
	case dataChan <- request:
		// 入队成功后记录入队耗时与队列长度
		// 注意：len(dataChan) 是近似观测，仅用于指标参考
		// 这里将耗时统计放在调用方路径内，默认 Noop 不引入开销
		b.metricsReporter.ObserveEnqueueLatency(time.Since(enqueueStart))
		b.metricsReporter.SetQueueLength(len(dataChan))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
