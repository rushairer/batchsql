package postgresql

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rushairer/batchsql/drivers"
)

// Driver PostgreSQL数据库SQL生成器
type Driver struct{}

// NewDriver 创建PostgreSQL驱动（用于自定义需求）
func NewDriver() *Driver {
	return &Driver{}
}

// GenerateInsertSQL 生成PostgreSQL批量插入SQL
func (d *Driver) GenerateInsertSQL(schema *drivers.Schema, data []map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, nil
	}

	columns := schema.Columns
	if len(columns) == 0 {
		return "", nil, errors.New("no columns defined in schema")
	}

	columnsStr := strings.Join(columns, ", ")
	placeholders := d.generatePlaceholders(len(columns), len(data))

	// 构建参数数组
	args := make([]interface{}, 0, len(data)*len(columns))
	for _, row := range data {
		for _, col := range columns {
			args = append(args, row[col])
		}
	}

	baseSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)

	switch schema.ConflictStrategy {
	case drivers.ConflictIgnore:
		sql := baseSQL + " ON CONFLICT DO NOTHING"
		return sql, args, nil
	case drivers.ConflictReplace, drivers.ConflictUpdate:
		updatePairs := make([]string, len(columns))
		for i, col := range columns {
			updatePairs[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
		}
		// 假设第一个列是主键
		sql := fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s", baseSQL, columns[0], strings.Join(updatePairs, ", "))
		return sql, args, nil
	default:
		return baseSQL, args, nil
	}
}

func (d *Driver) generatePlaceholders(columnCount, batchSize int) string {
	rows := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		placeholders := make([]string, columnCount)
		for j := 0; j < columnCount; j++ {
			placeholders[j] = fmt.Sprintf("$%d", i*columnCount+j+1)
		}
		rows[i] = "(" + strings.Join(placeholders, ", ") + ")"
	}
	return strings.Join(rows, ", ")
}

// DefaultDriver 全局默认PostgreSQL驱动实例
var DefaultDriver = &Driver{}
