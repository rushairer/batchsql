package drivers

import (
	"context"
	"database/sql"
	"time"
)

// CommonExecutor SQL数据库通用批量执行器
// 实现 BatchExecutor 接口，为SQL数据库提供统一的执行逻辑
// 架构：CommonExecutor -> BatchProcessor -> SQLDriver -> Database
//
// 设计优势：
// - 代码复用：所有SQL数据库共享相同的执行逻辑和指标收集
// - 职责分离：执行控制与具体处理逻辑分离
// - 易于扩展：新增SQL数据库只需实现SQLDriver接口
type CommonExecutor struct {
	processor       BatchProcessor  // 具体的批量处理逻辑
	metricsReporter MetricsReporter // 性能指标报告器
}

// NewCommonExecutor 创建通用执行器（使用自定义BatchProcessor）
func NewCommonExecutor(processor BatchProcessor) *CommonExecutor {
	return &CommonExecutor{
		processor: processor,
	}
}

// NewSQLExecutor 创建SQL数据库执行器（推荐方式）
// 内部使用 SQLBatchProcessor + SQLDriver 组合
func NewSQLExecutor(db *sql.DB, driver SQLDriver) *CommonExecutor {
	return NewCommonExecutor(NewSQLBatchProcessor(db, driver))
}

// ExecuteBatch 执行批量操作
func (e *CommonExecutor) ExecuteBatch(ctx context.Context, schema *Schema, data []map[string]any) error {
	startTime := time.Now()

	err := e.processor.ExecuteBatch(ctx, schema, data)
	status := "success"
	if err != nil {
		status = "fail"
	}
	if e.metricsReporter != nil {
		e.metricsReporter.RecordBatchExecution(
			schema.TableName,
			len(data),
			time.Since(startTime).Milliseconds(),
			status,
		)
	}
	return err
}

// WithMetricsReporter 设置指标报告器
func (e *CommonExecutor) WithMetricsReporter(metricsReporter MetricsReporter) BatchExecutor {
	e.metricsReporter = metricsReporter
	return e
}
