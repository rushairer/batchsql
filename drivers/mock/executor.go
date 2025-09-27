package mock

import (
	"context"
	"log"

	"github.com/rushairer/batchsql/drivers"
)

// Executor 模拟批量执行器（用于测试）
type Executor struct {
	ExecutedBatches [][]map[string]interface{}
	driver          drivers.SQLDriver
}

// NewBatchExecutor 创建模拟批量执行器（使用默认Driver）
func NewBatchExecutor() *Executor {
	return &Executor{
		ExecutedBatches: make([][]map[string]interface{}, 0),
		driver:          DefaultDriver,
	}
}

// NewBatchExecutorWithDriver 创建模拟批量执行器（使用自定义Driver）
func NewBatchExecutorWithDriver(driver drivers.SQLDriver) *Executor {
	if driver == nil {
		driver = DefaultDriver
	}
	return &Executor{
		ExecutedBatches: make([][]map[string]interface{}, 0),
		driver:          driver,
	}
}

// ExecuteBatch 模拟执行批量操作
func (e *Executor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]interface{}) error {
	e.ExecutedBatches = append(e.ExecutedBatches, data)

	// 生成并打印SQL信息
	sql, args, err := e.driver.GenerateInsertSQL(schema, data)
	if err != nil {
		return err
	}
	log.Printf("Mock execution - Table: %s, Data count: %d, SQL: %s, Args: %v",
		schema.TableName, len(data), sql, args)

	return nil
}