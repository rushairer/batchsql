// Package batchsql provides a unified batch database operation framework
package batchsql

import (
	"context"
	"fmt"
	"time"
)

// MetricsReporter 监控报告接口
type MetricsReporter interface {
	ReportBatchExecution(ctx context.Context, metrics BatchMetrics)
}

// BatchMetrics 批量操作指标
type BatchMetrics struct {
	Driver    string        // 数据库驱动名称
	Table     string        // 表名/集合名
	BatchSize int           // 批量大小
	Duration  time.Duration // 执行时长
	Error     error         // 错误信息（如果有）
	StartTime time.Time     // 开始时间
}

// Client 批量SQL客户端
type Client struct {
	executor BatchExecutorInterface // 批量执行器
	reporter MetricsReporter        // 可选的监控报告器
}

// NewClient 创建新的客户端
func NewClient() *Client {
	return &Client{
		executor: NewBatchExecutor(),
		reporter: nil, // 默认不启用监控
	}
}

// WithExecutor 设置批量执行器
func (c *Client) WithExecutor(executor BatchExecutorInterface) *Client {
	c.executor = executor
	return c
}

// WithMetricsReporter 设置监控报告器
func (c *Client) WithMetricsReporter(reporter MetricsReporter) *Client {
	c.reporter = reporter
	return c
}

// GetExecutor 获取执行器（用于添加数据库连接）
func (c *Client) GetExecutor() BatchExecutorInterface {
	return c.executor
}

// ExecuteBatch 执行批量操作（核心方法）
func (c *Client) ExecuteBatch(ctx context.Context, schema SchemaInterface, requests []*Request) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if len(requests) == 0 {
		return nil
	}

	// 记录开始时间（用于监控）
	startTime := time.Now()

	// 验证Schema
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// 生成批量命令
	driver := schema.GetDatabaseDriver()
	command, err := driver.GenerateBatchCommand(schema, requests)
	if err != nil {
		return fmt.Errorf("failed to generate batch command: %w", err)
	}

	// 执行操作
	execErr := c.executor.ExecuteBatch(ctx, []BatchCommand{command})

	// 报告监控指标（如果启用了监控）
	if c.reporter != nil {
		metrics := BatchMetrics{
			Driver:    driver.GetName(),
			Table:     schema.GetIdentifier(),
			BatchSize: len(requests),
			Duration:  time.Since(startTime),
			Error:     execErr,
			StartTime: startTime,
		}
		c.reporter.ReportBatchExecution(ctx, metrics)
	}

	return execErr
}

// ExecuteWithSchema 使用Schema执行批量操作（便捷方法）
func (c *Client) ExecuteWithSchema(ctx context.Context, schema SchemaInterface, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// 转换数据为请求
	requests := make([]*Request, len(data))
	for i, item := range data {
		request := NewRequestFromInterface(schema)
		for key, value := range item {
			request.Set(key, value)
		}
		requests[i] = request
	}

	return c.ExecuteBatch(ctx, schema, requests)
}

// ExecuteStreamBatch 流式批处理（处理海量数据）
func (c *Client) ExecuteStreamBatch(ctx context.Context, schema SchemaInterface, dataStream <-chan map[string]interface{}, batchSize int) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if batchSize <= 0 {
		batchSize = 1000 // 默认批大小
	}

	batch := make([]*Request, 0, batchSize)

	for {
		select {
		case data, ok := <-dataStream:
			if !ok {
				// 处理最后一批数据
				if len(batch) > 0 {
					if err := c.ExecuteBatch(ctx, schema, batch); err != nil {
						return err
					}
				}
				return nil
			}

			// 转换数据为请求
			request := NewRequestFromInterface(schema)
			for key, value := range data {
				request.Set(key, value)
			}
			batch = append(batch, request)

			// 当批次满了时执行
			if len(batch) >= batchSize {
				if err := c.ExecuteBatch(ctx, schema, batch); err != nil {
					return err
				}
				batch = batch[:0] // 重置批次
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ExecuteLargeBatch 处理大批量数据（自动分批）
func (c *Client) ExecuteLargeBatch(ctx context.Context, schema SchemaInterface, data []map[string]interface{}, batchSize int) error {
	if len(data) == 0 {
		return nil
	}

	if batchSize <= 0 {
		batchSize = 1000 // 默认批大小
	}

	// 分批处理
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		if err := c.ExecuteWithSchema(ctx, schema, batch); err != nil {
			return fmt.Errorf("failed to execute batch %d-%d: %w", i, end-1, err)
		}

		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// CreateSchema 创建Schema的便捷方法
func (c *Client) CreateSchema(identifier string, strategy ConflictStrategy, driver DatabaseDriver, columns ...string) SchemaInterface {
	return NewSchema(identifier, strategy, driver, columns...)
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.executor != nil {
		return c.executor.Close()
	}
	return nil
}
