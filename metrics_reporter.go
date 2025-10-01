package batchsql

// MetricsReporter 性能监控报告器接口
type MetricsReporter interface {
	RecordBatchExecution(tableName string, batchSize int, duration int64, status string)
}
