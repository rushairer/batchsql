package batchsql

import "github.com/rushairer/batchsql/drivers"

// NewSchema 创建新的Schema实例，保持向后兼容
func NewSchema(
	tableName string,
	conflictStrategy ConflictStrategy,
	columns ...string,
) *Schema {
	return &drivers.Schema{
		TableName:        tableName,
		ConflictStrategy: conflictStrategy,
		Columns:          columns,
	}
}
