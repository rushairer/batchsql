package drivers

import (
	"fmt"
	"strings"

	"github.com/rushairer/batchsql"
)

// SQLDriver SQL 数据库驱动基类
type SQLDriver struct {
	name string
}

// SQLCommand SQL 命令实现
type SQLCommand struct {
	commandType string
	sql         string
	parameters  []interface{}
	metadata    map[string]interface{}
}

func (c *SQLCommand) GetCommandType() string {
	return c.commandType
}

func (c *SQLCommand) GetCommand() interface{} {
	return c.sql
}

func (c *SQLCommand) GetParameters() []interface{} {
	return c.parameters
}

func (c *SQLCommand) GetMetadata() map[string]interface{} {
	return c.metadata
}

// MySQLDriver MySQL 驱动
type MySQLDriver struct {
	SQLDriver
}

func NewMySQLDriver() *MySQLDriver {
	return &MySQLDriver{
		SQLDriver: SQLDriver{name: "mysql"},
	}
}

func (d *MySQLDriver) GetName() string {
	return d.name
}

func (d *MySQLDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty requests")
	}

	columns := schema.GetColumns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns defined")
	}

	// 生成 SQL
	columnsStr := strings.Join(columns, ", ")
	placeholders := d.generatePlaceholders(len(columns), len(requests))

	var sql string
	switch schema.GetConflictStrategy() {
	case batchsql.ConflictIgnore:
		sql = fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s", schema.GetIdentifier(), columnsStr, placeholders)
	case batchsql.ConflictReplace:
		sql = fmt.Sprintf("REPLACE INTO %s (%s) VALUES %s", schema.GetIdentifier(), columnsStr, placeholders)
	case batchsql.ConflictUpdate:
		baseSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", schema.GetIdentifier(), columnsStr, placeholders)
		updatePairs := make([]string, len(columns))
		for i, col := range columns {
			updatePairs[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
		}
		sql = fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s", baseSQL, strings.Join(updatePairs, ", "))
	default:
		sql = fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", schema.GetIdentifier(), columnsStr, placeholders)
	}

	// 准备参数
	parameters := make([]interface{}, 0, len(requests)*len(columns))
	for _, request := range requests {
		values := request.GetOrderedValues()
		parameters = append(parameters, values...)
	}

	return &SQLCommand{
		commandType: "SQL",
		sql:         sql,
		parameters:  parameters,
		metadata: map[string]interface{}{
			"table":      schema.GetIdentifier(),
			"batch_size": len(requests),
			"driver":     d.name,
		},
	}, nil
}

func (d *MySQLDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictReplace,
		batchsql.ConflictUpdate,
	}
}

func (d *MySQLDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if schema.GetIdentifier() == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	if len(schema.GetColumns()) == 0 {
		return fmt.Errorf("columns cannot be empty")
	}

	// 验证冲突策略是否支持
	supported := d.SupportedConflictStrategies()
	strategy := schema.GetConflictStrategy()
	for _, s := range supported {
		if s == strategy {
			return nil
		}
	}
	return fmt.Errorf("unsupported conflict strategy: %v", strategy)
}

func (d *MySQLDriver) generatePlaceholders(columnCount, batchSize int) string {
	singleRow := "(" + strings.Repeat("?, ", columnCount-1) + "?)"
	rows := make([]string, batchSize)
	for i := range rows {
		rows[i] = singleRow
	}
	return strings.Join(rows, ", ")
}

// PostgreSQLDriver PostgreSQL 驱动
type PostgreSQLDriver struct {
	SQLDriver
}

func NewPostgreSQLDriver() *PostgreSQLDriver {
	return &PostgreSQLDriver{
		SQLDriver: SQLDriver{name: "postgresql"},
	}
}

func (d *PostgreSQLDriver) GetName() string {
	return d.name
}

func (d *PostgreSQLDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty requests")
	}

	columns := schema.GetColumns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns defined")
	}

	columnsStr := strings.Join(columns, ", ")
	placeholders := d.generatePlaceholders(len(columns), len(requests))
	baseSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", schema.GetIdentifier(), columnsStr, placeholders)

	var sql string
	switch schema.GetConflictStrategy() {
	case batchsql.ConflictIgnore:
		sql = fmt.Sprintf("%s ON CONFLICT DO NOTHING", baseSQL)
	case batchsql.ConflictReplace, batchsql.ConflictUpdate:
		updatePairs := make([]string, len(columns))
		for i, col := range columns {
			updatePairs[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
		}
		// 假设第一列是主键
		sql = fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s", baseSQL, columns[0], strings.Join(updatePairs, ", "))
	default:
		sql = baseSQL
	}

	parameters := make([]interface{}, 0, len(requests)*len(columns))
	for _, request := range requests {
		values := request.GetOrderedValues()
		parameters = append(parameters, values...)
	}

	return &SQLCommand{
		commandType: "SQL",
		sql:         sql,
		parameters:  parameters,
		metadata: map[string]interface{}{
			"table":      schema.GetIdentifier(),
			"batch_size": len(requests),
			"driver":     d.name,
		},
	}, nil
}

func (d *PostgreSQLDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictUpdate,
	}
}

func (d *PostgreSQLDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if schema.GetIdentifier() == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	if len(schema.GetColumns()) == 0 {
		return fmt.Errorf("columns cannot be empty")
	}

	supported := d.SupportedConflictStrategies()
	strategy := schema.GetConflictStrategy()
	for _, s := range supported {
		if s == strategy {
			return nil
		}
	}
	return fmt.Errorf("unsupported conflict strategy: %v", strategy)
}

func (d *PostgreSQLDriver) generatePlaceholders(columnCount, batchSize int) string {
	singleRow := "(" + strings.Repeat("?, ", columnCount-1) + "?)"
	rows := make([]string, batchSize)
	for i := range rows {
		rows[i] = singleRow
	}
	return strings.Join(rows, ", ")
}
