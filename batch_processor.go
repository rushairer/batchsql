package batchsql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// BatchProcessor 批量处理器
type BatchProcessor struct {
	db        *sql.DB
	sqlDriver SQLDriver
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(db *sql.DB, sqlDriver SQLDriver) *BatchProcessor {
	return &BatchProcessor{
		db:        db,
		sqlDriver: sqlDriver,
	}
}

// ProcessBatch 处理批量数据
func (bp *BatchProcessor) ProcessBatch(ctx context.Context, batchData []*Request) error {
	if len(batchData) == 0 {
		return nil
	}

	// 按 schema 指针聚合请求
	schemaGroups := bp.groupBySchema(batchData)

	// 处理每个 schema 组
	for schema, requests := range schemaGroups {
		if err := bp.processSchemaGroup(ctx, schema, requests); err != nil {
			log.Printf("Failed to process schema group for table %s: %v", schema.TableName(), err)
			return err
		}
	}

	return nil
}

// groupBySchema 按 schema 指针聚合请求
func (bp *BatchProcessor) groupBySchema(batchData []*Request) map[*Schema][]*Request {
	groups := make(map[*Schema][]*Request)

	for _, request := range batchData {
		schema := request.Schema()
		groups[schema] = append(groups[schema], request)
	}

	return groups
}

// processSchemaGroup 处理同一 schema 的请求组
func (bp *BatchProcessor) processSchemaGroup(ctx context.Context, schema *Schema, requests []*Request) error {
	if len(requests) == 0 {
		return nil
	}

	// 验证所有请求
	for i, request := range requests {
		if err := request.Validate(); err != nil {
			return fmt.Errorf("request %d validation failed: %w", i, err)
		}
	}

	// 使用SQLDriver生成批量插入SQL
	sql := bp.sqlDriver.GenerateInsertSQL(schema, len(requests))
	if sql == "" {
		return fmt.Errorf("failed to generate SQL for schema %s", schema.TableName())
	}

	log.Printf("Generated SQL for table %s: %s", schema.TableName(), sql)

	// 准备参数
	args := bp.prepareArgs(schema, requests)

	// 执行 SQL
	result, err := bp.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute batch insert for table %s: %w", schema.TableName(), err)
	}

	// 记录执行结果
	rowsAffected, _ := result.RowsAffected()
	log.Printf("Batch insert completed for table %s: %d rows affected, %d requests processed",
		schema.TableName(), rowsAffected, len(requests))

	return nil
}

// prepareArgs 准备 SQL 参数
func (bp *BatchProcessor) prepareArgs(schema *Schema, requests []*Request) []any {
	columnCount := len(schema.Columns())
	args := make([]any, 0, len(requests)*columnCount)

	for _, request := range requests {
		values := request.GetOrderedValues()
		args = append(args, values...)
	}

	return args
}
