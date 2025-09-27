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
	reporter MetricsReporter // 可选的监控报告器
}

// NewClient 创建新的客户端
func NewClient() *Client {
	return &Client{
		reporter: nil, // 默认不启用监控
	}
}

// WithMetricsReporter 设置监控报告器
func (c *Client) WithMetricsReporter(reporter MetricsReporter) *Client {
	c.reporter = reporter
	return c
}

// ExecuteWithSchema 使用Schema执行批量操作
func (c *Client) ExecuteWithSchema(ctx context.Context, schema SchemaInterface, data []map[string]interface{}) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if len(data) == 0 {
		return nil
	}

	// 记录开始时间（用于监控）
	startTime := time.Now()

	// 验证Schema
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
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

	// 生成批量命令
	driver := schema.GetDatabaseDriver()
	command, err := driver.GenerateBatchCommand(schema, requests)
	if err != nil {
		return fmt.Errorf("failed to generate batch command: %w", err)
	}

	// 执行操作
	execErr := c.simulateExecution(ctx, driver.GetName(), command)

	// 报告监控指标（如果启用了监控）
	if c.reporter != nil {
		metrics := BatchMetrics{
			Driver:    driver.GetName(),
			Table:     schema.GetIdentifier(),
			BatchSize: len(data),
			Duration:  time.Since(startTime),
			Error:     execErr,
			StartTime: startTime,
		}
		c.reporter.ReportBatchExecution(ctx, metrics)
	}

	return execErr
}

// simulateExecution 模拟执行命令
func (c *Client) simulateExecution(ctx context.Context, driverName string, command BatchCommand) error {
	// 在实际实现中，这里会连接真实的数据库并执行命令
	// 现在只是模拟执行过程
	return nil
}

// CreateSchema 创建Schema的便捷方法
func (c *Client) CreateSchema(identifier string, strategy ConflictStrategy, driver DatabaseDriver, columns ...string) SchemaInterface {
	return NewSchema(identifier, strategy, driver, columns...)
}

// Close 关闭客户端
func (c *Client) Close() error {
	// 清理资源
	return nil
}
