package batchsql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// BatchProcessor 批量处理器
type BatchProcessor struct {
	db *sql.DB
}

// NewBatchProcessor 创建批量处理器
func NewBatchProcessor(db *sql.DB) *BatchProcessor {
	return &BatchProcessor{db: db}
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

	// 生成批量插入 SQL
	sql := schema.GenerateInsertSQL(len(requests))
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

// BatchExecutor 批量执行器接口
type BatchExecutor interface {
	ExecuteBatch(ctx context.Context, batchData []*Request) error
}

// DatabaseBatchExecutor 数据库批量执行器
type DatabaseBatchExecutor struct {
	processor *BatchProcessor
}

// NewDatabaseBatchExecutor 创建数据库批量执行器
func NewDatabaseBatchExecutor(db *sql.DB) *DatabaseBatchExecutor {
	return &DatabaseBatchExecutor{
		processor: NewBatchProcessor(db),
	}
}

// ExecuteBatch 执行批量操作
func (dbe *DatabaseBatchExecutor) ExecuteBatch(ctx context.Context, batchData []*Request) error {
	return dbe.processor.ProcessBatch(ctx, batchData)
}

// MockBatchExecutor 模拟批量执行器（用于测试）
type MockBatchExecutor struct {
	ExecutedBatches [][]*Request
}

// NewMockBatchExecutor 创建模拟批量执行器
func NewMockBatchExecutor() *MockBatchExecutor {
	return &MockBatchExecutor{
		ExecutedBatches: make([][]*Request, 0),
	}
}

// ExecuteBatch 模拟执行批量操作
func (mbe *MockBatchExecutor) ExecuteBatch(ctx context.Context, batchData []*Request) error {
	mbe.ExecutedBatches = append(mbe.ExecutedBatches, batchData)

	// 按 schema 分组并打印信息
	processor := &BatchProcessor{}
	groups := processor.groupBySchema(batchData)

	for schema, requests := range groups {
		sql := schema.GenerateInsertSQL(len(requests))
		log.Printf("Mock execution - Table: %s, Requests: %d, SQL: %s",
			schema.TableName(), len(requests), sql)
	}

	return nil
}
