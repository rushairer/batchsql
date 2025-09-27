package batchsql

import (
	"context"
	"database/sql"
)

// BatchExecutor 批量执行器接口
type BatchExecutor interface {
	ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]interface{}) error
}

// SQLDriver 数据库特定的SQL生成器接口
type SQLDriver interface {
	GenerateInsertSQL(schema *Schema, data []map[string]interface{}) (string, []interface{}, error)
}

// TransactionExecutor 支持事务的批量执行器接口（可选扩展）
type TransactionExecutor interface {
	BatchExecutor
	ExecuteBatchWithTx(ctx context.Context, tx *sql.Tx, schema *Schema, data []map[string]interface{}) error
}

// MetricsReporter 性能监控报告器接口（未来扩展）
type MetricsReporter interface {
	RecordBatchExecution(driver string, table string, batchSize int, duration int64, err error)
	RecordRetry(driver string, table string, attempt int, err error)
}

// MetricsAware 支持监控的执行器接口（未来扩展）
type MetricsAware interface {
	WithMetricsReporter(reporter MetricsReporter) BatchExecutor
}
