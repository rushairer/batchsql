package batchsql

type ConflictStrategy int

const (
	ConflictIgnore  ConflictStrategy = iota // 跳过冲突
	ConflictReplace                         // 覆盖冲突
	ConflictUpdate                          // 更新冲突
)

// Schema 定义了批量操作的表结构，专注于表结构定义
type Schema struct {
	tableName        string
	conflictStrategy ConflictStrategy
	columns          []string // 列名顺序
}

func NewSchema(
	tableName string,
	conflictStrategy ConflictStrategy,
	columns ...string,
) *Schema {
	return &Schema{
		tableName:        tableName,
		conflictStrategy: conflictStrategy,
		columns:          columns,
	}
}

// Getters
func (s *Schema) TableName() string {
	return s.tableName
}

func (s *Schema) ConflictStrategy() ConflictStrategy {
	return s.conflictStrategy
}

func (s *Schema) Columns() []string {
	return s.columns
}
