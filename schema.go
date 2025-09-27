package batchsql

import (
	"fmt"
	"strings"
)

type ConflictStrategy int

const (
	ConflictIgnore  ConflictStrategy = iota // 跳过冲突
	ConflictReplace                         // 覆盖冲突
	ConflictUpdate                          // 更新冲突
)

type DatabaseType int

const (
	MySQL      DatabaseType = iota // MySQL
	PostgreSQL                     // PostgreSQL
	SQLite                         // SQLite
)

type Schema struct {
	tableName        string
	conflictStrategy ConflictStrategy
	databaseType     DatabaseType
	columns          []string // 列名顺序
}

func NewSchema(
	tableName string,
	conflictStrategy ConflictStrategy,
	databaseType DatabaseType,
	columns ...string,
) *Schema {
	return &Schema{
		tableName:        tableName,
		conflictStrategy: conflictStrategy,
		databaseType:     databaseType,
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

func (s *Schema) DatabaseType() DatabaseType {
	return s.databaseType
}

func (s *Schema) Columns() []string {
	return s.columns
}

// GenerateInsertSQL 根据数据库类型和冲突策略生成批量插入SQL
func (s *Schema) GenerateInsertSQL(batchSize int) string {
	if len(s.columns) == 0 {
		return ""
	}

	columnsStr := strings.Join(s.columns, ", ")
	placeholders := s.generatePlaceholders(batchSize)

	switch s.databaseType {
	case MySQL:
		return s.generateMySQLInsert(columnsStr, placeholders)
	case PostgreSQL:
		return s.generatePostgreSQLInsert(columnsStr, placeholders)
	case SQLite:
		return s.generateSQLiteInsert(columnsStr, placeholders)
	default:
		return ""
	}
}

func (s *Schema) generatePlaceholders(batchSize int) string {
	singleRow := "(" + strings.Repeat("?, ", len(s.columns)-1) + "?)"
	rows := make([]string, batchSize)
	for i := range rows {
		rows[i] = singleRow
	}
	return strings.Join(rows, ", ")
}

func (s *Schema) generateMySQLInsert(columnsStr, placeholders string) string {
	baseSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)

	switch s.conflictStrategy {
	case ConflictIgnore:
		return fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)
	case ConflictReplace:
		return fmt.Sprintf("REPLACE INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)
	case ConflictUpdate:
		updatePairs := make([]string, len(s.columns))
		for i, col := range s.columns {
			updatePairs[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
		}
		return fmt.Sprintf("%s ON DUPLICATE KEY UPDATE %s", baseSQL, strings.Join(updatePairs, ", "))
	default:
		return baseSQL
	}
}

func (s *Schema) generatePostgreSQLInsert(columnsStr, placeholders string) string {
	baseSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)

	switch s.conflictStrategy {
	case ConflictIgnore:
		return fmt.Sprintf("%s ON CONFLICT DO NOTHING", baseSQL)
	case ConflictReplace, ConflictUpdate:
		updatePairs := make([]string, len(s.columns))
		for i, col := range s.columns {
			updatePairs[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
		}
		// 假设第一列是主键，实际使用中可能需要更灵活的配置
		return fmt.Sprintf("%s ON CONFLICT (%s) DO UPDATE SET %s", baseSQL, s.columns[0], strings.Join(updatePairs, ", "))
	default:
		return baseSQL
	}
}

func (s *Schema) generateSQLiteInsert(columnsStr, placeholders string) string {
	baseSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)

	switch s.conflictStrategy {
	case ConflictIgnore:
		return fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)
	case ConflictReplace:
		return fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES %s", s.tableName, columnsStr, placeholders)
	case ConflictUpdate:
		updatePairs := make([]string, len(s.columns))
		for i, col := range s.columns {
			updatePairs[i] = fmt.Sprintf("%s = excluded.%s", col, col)
		}
		return fmt.Sprintf("%s ON CONFLICT DO UPDATE SET %s", baseSQL, strings.Join(updatePairs, ", "))
	default:
		return baseSQL
	}
}
