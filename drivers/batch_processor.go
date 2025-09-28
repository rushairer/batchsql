package drivers

import (
	"context"
	"database/sql"
)

// SQLBatchProcessor SQL数据库批量处理器
// 实现 BatchProcessor 接口，专注于SQL数据库的核心处理逻辑
// 架构位置：CommonExecutor -> SQLBatchProcessor -> SQLDriver -> Database
//
// 职责：
// - 调用SQLDriver生成数据库特定的SQL语句
// - 执行批量SQL操作
// - 处理数据库连接和事务
//
// 注意：此组件仅用于SQL数据库，NoSQL数据库可直接实现BatchExecutor
type SQLBatchProcessor struct {
	db        *sql.DB   // 数据库连接
	sqlDriver SQLDriver // SQL生成器（数据库特定）
}

// NewSQLBatchProcessor 创建SQL批量处理器
// 参数：
// - db: 数据库连接（用户管理连接池）
// - sqlDriver: 数据库特定的SQL生成器
func NewSQLBatchProcessor(db *sql.DB, sqlDriver SQLDriver) *SQLBatchProcessor {
	return &SQLBatchProcessor{
		db:        db,
		sqlDriver: sqlDriver,
	}
}

// ExecuteBatch 执行批量操作
func (bp *SQLBatchProcessor) ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// 使用SQLDriver生成批量插入SQL
	sql, args, err := bp.sqlDriver.GenerateInsertSQL(schema, data)
	if err != nil {
		return err
	}

	// 执行 SQL
	_, err = bp.db.ExecContext(ctx, sql, args...)
	return err
}
