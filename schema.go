package batchsql

// ConflictStrategy 冲突处理策略
type ConflictStrategy uint8

const (
	ConflictIgnore ConflictStrategy = iota
	ConflictReplace
	ConflictUpdate
)

// Schema 表结构定义
type Schema struct {
	Name             string
	Columns          []string
	ConflictStrategy ConflictStrategy
}

// NewSchema 创建新的Schema实例
func NewSchema(
	name string,
	conflictStrategy ConflictStrategy,
	columns ...string,
) *Schema {
	return &Schema{
		Name:             name,
		ConflictStrategy: conflictStrategy,
		Columns:          columns,
	}
}
