package mysql

import (
	"database/sql"

	"github.com/rushairer/batchsql/drivers"
)

// NewBatchExecutor 创建MySQL批量执行器（使用默认Driver）
// 返回 CommonExecutor，内部架构：CommonExecutor -> SQLBatchProcessor -> MySQLDriver
// 这是推荐的使用方式，使用MySQL优化的默认SQL生成器
func NewBatchExecutor(db *sql.DB) *drivers.CommonExecutor {
	return drivers.NewSQLExecutor(db, DefaultDriver, "mysql")
}

// NewBatchExecutorWithDriver 创建MySQL批量执行器（使用自定义Driver）
// 返回 CommonExecutor，但使用自定义的SQLDriver实现
// 适用于需要特殊SQL优化或支持MySQL变种（如TiDB）的场景
func NewBatchExecutorWithDriver(db *sql.DB, driver drivers.SQLDriver) *drivers.CommonExecutor {
	return drivers.NewSQLExecutor(db, driver, "mysql")
}
