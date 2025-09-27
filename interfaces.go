package batchsql

import (
	"context"
	"database/sql"

	"github.com/rushairer/batchsql/drivers"
)

// 重新导出drivers包的类型，保持向后兼容
type ConflictStrategy = drivers.ConflictStrategy
type Schema = drivers.Schema
type SQLDriver = drivers.SQLDriver
type BatchExecutor = drivers.BatchExecutor

// 重新导出常量
const (
	ConflictIgnore  = drivers.ConflictIgnore
	ConflictReplace = drivers.ConflictReplace
	ConflictUpdate  = drivers.ConflictUpdate
)

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
