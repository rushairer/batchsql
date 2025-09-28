package drivers

import (
	"context"
)

// ConflictStrategy 冲突处理策略
type ConflictStrategy int

const (
	ConflictIgnore ConflictStrategy = iota
	ConflictReplace
	ConflictUpdate
)

// Schema 表结构定义
type Schema struct {
	TableName        string
	Columns          []string
	ConflictStrategy ConflictStrategy
}

// SQLDriver 数据库特定的SQL生成器接口
type SQLDriver interface {
	GenerateInsertSQL(schema *Schema, data []map[string]any) (string, []any, error)
}

// BatchExecutor 批量执行器接口 - 所有数据库驱动的统一入口
// 实现方式：
// - SQL数据库：通过 CommonExecutor + BatchProcessor + SQLDriver 组合实现
// - NoSQL数据库：直接实现此接口（如 Redis）
// - 测试环境：通过 MockExecutor 直接实现
type BatchExecutor interface {
	ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error
	WithMetricsReporter(metricsReporter MetricsReporter) BatchExecutor
}

// BatchProcessor 批量处理器接口 - SQL数据库的核心处理逻辑
// 注意：此接口不是必须的，仅用于SQL数据库的代码复用
// - 与 CommonExecutor 配合使用，提供SQL数据库的通用处理逻辑
// - NoSQL数据库可以跳过此层，直接实现 BatchExecutor
type BatchProcessor interface {
	ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error
}

// MetricsReporter 性能监控报告器接口
type MetricsReporter interface {
	RecordBatchExecution(driver string, table string, batchSize int, duration int64, status string)
}
