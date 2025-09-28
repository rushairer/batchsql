package sqlite

import (
	"database/sql"

	"github.com/rushairer/batchsql/drivers"
)

// NewBatchExecutor 创建SQLite批量执行器（使用默认Driver）
func NewBatchExecutor(db *sql.DB) *drivers.CommonExecutor {
	return drivers.NewSQLExecutor(db, DefaultDriver)
}

// NewBatchExecutorWithDriver 创建SQLite批量执行器（使用自定义Driver）
func NewBatchExecutorWithDriver(db *sql.DB, driver drivers.SQLDriver) *drivers.CommonExecutor {
	return drivers.NewSQLExecutor(db, driver)
}
