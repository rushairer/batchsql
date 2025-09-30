package mock

import (
	"context"
	"log"

	"github.com/rushairer/batchsql/drivers"
)

// Executor 模拟批量执行器（用于测试）
type MockExecutor struct {
	ExecutedBatches [][]map[string]any
	driver          drivers.SQLDriver
	metricsReporter drivers.MetricsReporter
}

// NewBatchExecutor 创建模拟批量执行器（使用默认Driver）
func NewBatchExecutor() *MockExecutor {
	return &MockExecutor{
		ExecutedBatches: make([][]map[string]any, 0),
		driver:          DefaultDriver,
	}
}

// NewBatchExecutorWithDriver 创建模拟批量执行器（使用自定义Driver）
func NewBatchExecutorWithDriver(driver drivers.SQLDriver) *MockExecutor {
	if driver == nil {
		driver = DefaultDriver
	}
	return &MockExecutor{
		ExecutedBatches: make([][]map[string]any, 0),
		driver:          driver,
	}
}

// ExecuteBatch 模拟执行批量操作
func (e *MockExecutor) ExecuteBatch(_ context.Context, schema *drivers.Schema, data []map[string]any) error {
	e.ExecutedBatches = append(e.ExecutedBatches, data)

	// 生成SQL信息（不输出大参数）
	_, args, err := e.driver.GenerateInsertSQL(schema, data)
	if err != nil {
		return err
	}

	// 只显示参数数量，避免输出大字符串
	log.Printf("Mock execution - Table: %s, Data count: %d, Args count: %d",
		schema.TableName, len(data), len(args))

	return nil
}

// WithMetricsReporter 设置指标报告器
func (e *MockExecutor) WithMetricsReporter(metricsReporter drivers.MetricsReporter) drivers.BatchExecutor {
	e.metricsReporter = metricsReporter
	return e
}
