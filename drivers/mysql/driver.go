package mysql

import (
	"fmt"
	"strings"

	"github.com/rushairer/batchsql/drivers"
)

// Driver MySQL数据库SQL生成器
type Driver struct{}

// NewDriver 创建MySQL驱动（用于自定义需求）
func NewDriver() *Driver {
	return &Driver{}
}

// GenerateInsertSQL 生成MySQL批量插入SQL
func (d *Driver) GenerateInsertSQL(schema *drivers.Schema, data []map[string]interface{}) (string, []interface{}, error) {
	if len(data) == 0 {
		return "", nil, nil
	}

	columns := schema.Columns
	if len(columns) == 0 {
		return "", nil, fmt.Errorf("no columns defined in schema")
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
		sql := fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)
		return sql, args, nil
	case drivers.ConflictReplace:
		sql := fmt.Sprintf("REPLACE INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)
		return sql, args, nil
	case drivers.ConflictUpdate:
		updatePairs := make([]string, len(columns))
		for i, col := range columns {
			updatePairs[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
		}
		sql := fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s", baseSQL, strings.Join(updatePairs, ", "))
		return sql, args, nil
	default:
		return baseSQL, args, nil
	}
}

func (d *Driver) generatePlaceholders(columnCount, batchSize int) string {
	singleRow := "(" + strings.Repeat("?, ", columnCount-1) + "?)"
	rows := make([]string, batchSize)
	for i := range rows {
		rows[i] = singleRow
	}
	return strings.Join(rows, ", ")
}

// DefaultDriver 全局默认MySQL驱动实例
var DefaultDriver = &Driver{}
