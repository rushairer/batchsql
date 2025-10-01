package batchsql

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// BatchExecutor 批量执行器接口 - 所有数据库驱动的统一入口
type BatchExecutor interface {
	// ExecuteBatch 执行批量操作
	ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error

	// WithMetricsReporter 设置性能监控报告器
	WithMetricsReporter(metricsReporter MetricsReporter) BatchExecutor
}

// ThrottledBatchExecutor SQL数据库通用批量执行器
// 实现 ThrottledBatchExecutor 接口，为SQL数据库提供统一的执行逻辑
// 架构：ThrottledBatchExecutor -> BatchProcessor -> SQLDriver -> Database
//
// 设计优势：
// - 代码复用：所有SQL数据库共享相同的执行逻辑和指标收集
// - 职责分离：执行控制与具体处理逻辑分离
// - 易于扩展：新增SQL数据库只需实现SQLDriver接口
type ThrottledBatchExecutor struct {
	processor       BatchProcessor  // 具体的批量处理逻辑
	metricsReporter MetricsReporter // 性能指标报告器
	semaphore       chan struct{}   // 可选信号量，用于限制 ExecuteBatch 并发
}

// NewThrottledBatchExecutor 创建通用执行器（使用自定义BatchProcessor）
func NewThrottledBatchExecutor(processor BatchProcessor) *ThrottledBatchExecutor {
	return &ThrottledBatchExecutor{
		processor: processor,
	}
}

// NewThrottledBatchExecutorWithDriver 创建SQL数据库执行器（推荐方式）
// 内部使用 SQLBatchProcessor + SQLDriver 组合
func NewSQLThrottledBatchExecutorWithDriver(db *sql.DB, driver SQLDriver) *ThrottledBatchExecutor {
	return NewThrottledBatchExecutor(NewSQLBatchProcessor(db, driver))
}

func NewRedisThrottledBatchExecutor(client *redis.Client) *ThrottledBatchExecutor {
	return NewThrottledBatchExecutor(NewRedisBatchProcessor(client, DefaultRedisPipelineDriver))
}

func NewRedisThrottledBatchExecutorWithDriver(client *redis.Client, driver RedisDriver) *ThrottledBatchExecutor {
	return NewThrottledBatchExecutor(NewRedisBatchProcessor(client, driver))
}

// ExecuteBatch 执行批量操作
func (e *ThrottledBatchExecutor) ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// 可选并发限流：当设置了信号量时，进入前需占用一个令牌
	if e.semaphore != nil {
		select {
		case e.semaphore <- struct{}{}:
			defer func() { <-e.semaphore }()
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	startTime := time.Now()

	status := "success"

	operations, err := e.processor.GenerateOperations(ctx, schema, data)
	if err != nil {
		return err
	}
	err = e.processor.ExecuteOperations(ctx, operations)
	if err != nil {
		status = "fail"
	}
	if e.metricsReporter != nil {
		e.metricsReporter.RecordBatchExecution(
			schema.Name,
			len(data),
			time.Since(startTime).Milliseconds(),
			status,
		)
	}
	return err
}

// WithMetricsReporter 设置指标报告器
func (e *ThrottledBatchExecutor) WithMetricsReporter(metricsReporter MetricsReporter) BatchExecutor {
	e.metricsReporter = metricsReporter
	return e
}

// WithConcurrencyLimit 设置并发上限（limit <= 0 表示不启用限流）
func (e *ThrottledBatchExecutor) WithConcurrencyLimit(limit int) BatchExecutor {
	if limit > 0 {
		e.semaphore = make(chan struct{}, limit)
	} else {
		e.semaphore = nil
	}
	return e
}

// Executor 模拟批量执行器（用于测试）
type MockExecutor struct {
	ExecutedBatches [][]map[string]any
	driver          SQLDriver
	metricsReporter MetricsReporter
	mu              sync.RWMutex
}

// NewMockExecutor 创建模拟批量执行器（使用默认Driver）
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		ExecutedBatches: make([][]map[string]any, 0),
		driver:          DefaultMySQLDriver,
	}
}

// NewMockExecutorWithDriver 创建模拟批量执行器（使用自定义Driver）
func NewMockExecutorWithDriver(driver SQLDriver) *MockExecutor {
	if driver == nil {
		driver = DefaultMySQLDriver
	}
	return &MockExecutor{
		ExecutedBatches: make([][]map[string]any, 0),
		driver:          driver,
	}
}

// ExecuteBatch 模拟执行批量操作
func (e *MockExecutor) ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error {
	e.mu.Lock()
	e.ExecutedBatches = append(e.ExecutedBatches, data)
	e.mu.Unlock()

	// 生成SQL信息（不输出大参数）
	_, args, err := e.driver.GenerateInsertSQL(ctx, schema, data)
	if err != nil {
		return err
	}

	// 只显示参数数量，避免输出大字符串
	log.Printf("Mock execution - Table: %s, Data count: %d, Args count: %d",
		schema.Name, len(data), len(args))

	return nil
}

// WithMetricsReporter 设置指标报告器
func (e *MockExecutor) WithMetricsReporter(metricsReporter MetricsReporter) BatchExecutor {
	e.metricsReporter = metricsReporter
	return e
}

// SnapshotExecutedBatches 返回一次性快照，避免并发读写竞态
func (e *MockExecutor) SnapshotExecutedBatches() [][]map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([][]map[string]any, len(e.ExecutedBatches))
	copy(out, e.ExecutedBatches)
	return out
}
