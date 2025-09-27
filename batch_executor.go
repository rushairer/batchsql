package batchsql

import (
	"context"
	"database/sql"
	"log"
)

// MySQLBatchExecutor MySQL批量执行器
type MySQLBatchExecutor struct {
	processor *BatchProcessor
}

// NewMySQLBatchExecutor 创建MySQL批量执行器（使用默认Driver）
func NewMySQLBatchExecutor(db *sql.DB) *MySQLBatchExecutor {
	return &MySQLBatchExecutor{
		processor: NewBatchProcessor(db, DefaultMySQLDriver),
	}
}

// NewMySQLBatchExecutorWithDriver 创建MySQL批量执行器（使用自定义Driver）
func NewMySQLBatchExecutorWithDriver(db *sql.DB, driver SQLDriver) *MySQLBatchExecutor {
	return &MySQLBatchExecutor{
		processor: NewBatchProcessor(db, driver),
	}
}

// ExecuteBatch 执行批量操作
func (e *MySQLBatchExecutor) ExecuteBatch(ctx context.Context, batchData []*Request) error {
	return e.processor.ProcessBatch(ctx, batchData)
}

// PostgreSQLBatchExecutor PostgreSQL批量执行器
type PostgreSQLBatchExecutor struct {
	processor *BatchProcessor
}

// NewPostgreSQLBatchExecutor 创建PostgreSQL批量执行器（使用默认Driver）
func NewPostgreSQLBatchExecutor(db *sql.DB) *PostgreSQLBatchExecutor {
	return &PostgreSQLBatchExecutor{
		processor: NewBatchProcessor(db, DefaultPostgreSQLDriver),
	}
}

// NewPostgreSQLBatchExecutorWithDriver 创建PostgreSQL批量执行器（使用自定义Driver）
func NewPostgreSQLBatchExecutorWithDriver(db *sql.DB, driver SQLDriver) *PostgreSQLBatchExecutor {
	return &PostgreSQLBatchExecutor{
		processor: NewBatchProcessor(db, driver),
	}
}

// ExecuteBatch 执行批量操作
func (e *PostgreSQLBatchExecutor) ExecuteBatch(ctx context.Context, batchData []*Request) error {
	return e.processor.ProcessBatch(ctx, batchData)
}

// SQLiteBatchExecutor SQLite批量执行器
type SQLiteBatchExecutor struct {
	processor *BatchProcessor
}

// NewSQLiteBatchExecutor 创建SQLite批量执行器（使用默认Driver）
func NewSQLiteBatchExecutor(db *sql.DB) *SQLiteBatchExecutor {
	return &SQLiteBatchExecutor{
		processor: NewBatchProcessor(db, DefaultSQLiteDriver),
	}
}

// NewSQLiteBatchExecutorWithDriver 创建SQLite批量执行器（使用自定义Driver）
func NewSQLiteBatchExecutorWithDriver(db *sql.DB, driver SQLDriver) *SQLiteBatchExecutor {
	return &SQLiteBatchExecutor{
		processor: NewBatchProcessor(db, driver),
	}
}

// ExecuteBatch 执行批量操作
func (e *SQLiteBatchExecutor) ExecuteBatch(ctx context.Context, batchData []*Request) error {
	return e.processor.ProcessBatch(ctx, batchData)
}

// MockBatchExecutor 模拟批量执行器（用于测试）
type MockBatchExecutor struct {
	ExecutedBatches [][]*Request
	sqlDriver       SQLDriver
}

// NewMockBatchExecutor 创建模拟批量执行器（使用默认MySQL Driver）
func NewMockBatchExecutor() *MockBatchExecutor {
	return &MockBatchExecutor{
		ExecutedBatches: make([][]*Request, 0),
		sqlDriver:       DefaultMySQLDriver,
	}
}

// NewMockBatchExecutorWithDriver 创建模拟批量执行器（使用自定义Driver）
func NewMockBatchExecutorWithDriver(sqlDriver SQLDriver) *MockBatchExecutor {
	if sqlDriver == nil {
		sqlDriver = DefaultMySQLDriver // 默认使用MySQL驱动
	}
	return &MockBatchExecutor{
		ExecutedBatches: make([][]*Request, 0),
		sqlDriver:       sqlDriver,
	}
}

// ExecuteBatch 模拟执行批量操作
func (mbe *MockBatchExecutor) ExecuteBatch(ctx context.Context, batchData []*Request) error {
	mbe.ExecutedBatches = append(mbe.ExecutedBatches, batchData)

	// 按 schema 分组并打印信息
	processor := &BatchProcessor{sqlDriver: mbe.sqlDriver}
	groups := processor.groupBySchema(batchData)

	for schema, requests := range groups {
		sql := mbe.sqlDriver.GenerateInsertSQL(schema, len(requests))
		log.Printf("Mock execution - Table: %s, Requests: %d, SQL: %s",
			schema.TableName(), len(requests), sql)
	}

	return nil
}
