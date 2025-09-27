package drivers

import "context"

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

// BatchExecutor 批量执行器接口
type BatchExecutor interface {
	ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error
}
