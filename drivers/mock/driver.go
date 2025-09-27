package mock

import (
	"fmt"
	"strings"

	"github.com/rushairer/batchsql/drivers"
)

// Driver 模拟SQL生成器（默认使用MySQL语法）
type Driver struct {
	databaseType string
}

// NewDriver 创建模拟驱动
func NewDriver() *Driver {
	return &Driver{databaseType: "mysql"}
}

// NewDriverWithType 创建指定数据库类型的模拟驱动
func NewDriverWithType(dbType string) *Driver {
	return &Driver{databaseType: dbType}
}

// GenerateInsertSQL 生成模拟SQL（默认MySQL语法）
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

	// 根据数据库类型生成不同的SQL
	switch d.databaseType {
	case "mysql":
		return d.generateMySQLSQL(schema, baseSQL, columnsStr, placeholders, args)
	case "postgresql":
		return d.generatePostgreSQLSQL(schema, baseSQL, columnsStr, placeholders, args)
	case "sqlite":
		return d.generateSQLiteSQL(schema, baseSQL, columnsStr, placeholders, args)
	default:
		return baseSQL, args, nil
	}
}

func (d *Driver) generateMySQLSQL(schema *drivers.Schema, baseSQL, columnsStr, placeholders string, args []interface{}) (string, []interface{}, error) {
	switch schema.ConflictStrategy {
	case drivers.ConflictIgnore:
		sql := fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)
		return sql, args, nil
	case drivers.ConflictReplace:
		sql := fmt.Sprintf("REPLACE INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)
		return sql, args, nil
	case drivers.ConflictUpdate:
		updatePairs := make([]string, len(schema.Columns))
		for i, col := range schema.Columns {
			updatePairs[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
		}
		sql := fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s", baseSQL, strings.Join(updatePairs, ", "))
		return sql, args, nil
	default:
		return baseSQL, args, nil
	}
}

func (d *Driver) generatePostgreSQLSQL(schema *drivers.Schema, baseSQL, columnsStr, placeholders string, args []interface{}) (string, []interface{}, error) {
	switch schema.ConflictStrategy {
	case drivers.ConflictIgnore:
		sql := fmt.Sprintf("%s ON CONFLICT DO NOTHING", baseSQL)
		return sql, args, nil
	case drivers.ConflictReplace, drivers.ConflictUpdate:
		updatePairs := make([]string, len(schema.Columns))
		for i, col := range schema.Columns {
			updatePairs[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
		}
		sql := fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s", baseSQL, schema.Columns[0], strings.Join(updatePairs, ", "))
		return sql, args, nil
	default:
		return baseSQL, args, nil
	}
}

func (d *Driver) generateSQLiteSQL(schema *drivers.Schema, baseSQL, columnsStr, placeholders string, args []interface{}) (string, []interface{}, error) {
	switch schema.ConflictStrategy {
	case drivers.ConflictIgnore:
		sql := fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)
		return sql, args, nil
	case drivers.ConflictReplace:
		sql := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES %s", schema.TableName, columnsStr, placeholders)
		return sql, args, nil
	case drivers.ConflictUpdate:
		updatePairs := make([]string, len(schema.Columns))
		for i, col := range schema.Columns {
			updatePairs[i] = fmt.Sprintf("%s = excluded.%s", col, col)
		}
		sql := fmt.Sprintf("%s ON CONFLICT DO UPDATE SET %s", baseSQL, strings.Join(updatePairs, ", "))
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

// DefaultDriver 全局默认模拟驱动实例
var DefaultDriver = NewDriver()
