package batchsql

import (
	"context"
	"fmt"
	"time"
)

// SimpleBatchSQLClient 简化的BatchSQL客户端
type SimpleBatchSQLClient struct {
	connectionManager *DefaultConnectionManager
	metricsCollector  *DefaultMetricsCollector
	config            *ClientConfig
}

// NewSimpleBatchSQLClient 创建简化的BatchSQL客户端
func NewSimpleBatchSQLClient(config *ClientConfig) (*SimpleBatchSQLClient, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	// 创建连接管理器
	connectionManager := NewDefaultConnectionManager()

	// 添加连接配置
	for driverName, connConfig := range config.Connections {
		if err := connectionManager.AddConnection(driverName, connConfig); err != nil {
			return nil, fmt.Errorf("failed to add connection for %s: %w", driverName, err)
		}
	}

	// 创建指标收集器
	var metricsCollector *DefaultMetricsCollector
	if config.EnableMetrics {
		collector := NewDefaultMetricsCollector()
		collector.SetMaxHistorySize(config.MetricsHistorySize)
		metricsCollector = collector
	}

	return &SimpleBatchSQLClient{
		connectionManager: connectionManager,
		metricsCollector:  metricsCollector,
		config:            config,
	}, nil
}

// ExecuteWithSchema 使用指定Schema执行批量操作
func (c *SimpleBatchSQLClient) ExecuteWithSchema(ctx context.Context, schema SchemaInterface, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	startTime := time.Now()

	// 转换数据为Request
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

	// 执行命令
	err = c.executeCommand(ctx, driver.GetName(), command)

	// 记录指标
	duration := time.Since(startTime).Milliseconds()
	if c.metricsCollector != nil {
		c.metricsCollector.RecordBatchExecution(
			driver.GetName(),
			len(requests),
			duration,
			err,
		)
	}

	return err
}

// executeCommand 执行命令
func (c *SimpleBatchSQLClient) executeCommand(ctx context.Context, driverName string, command BatchCommand) error {
	conn, err := c.connectionManager.GetConnection(driverName)
	if err != nil {
		return fmt.Errorf("failed to get connection for %s: %w", driverName, err)
	}
	defer c.connectionManager.ReleaseConnection(driverName, conn)

	switch command.GetCommandType() {
	case "SQL":
		return c.executeSQLCommand(ctx, conn, command)
	case "REDIS":
		return c.executeRedisCommand(ctx, conn, command)
	case "MONGODB":
		return c.executeMongoCommand(ctx, conn, command)
	default:
		return fmt.Errorf("unsupported command type: %s", command.GetCommandType())
	}
}

// executeSQLCommand 执行SQL命令
func (c *SimpleBatchSQLClient) executeSQLCommand(ctx context.Context, conn interface{}, command BatchCommand) error {
	// 模拟SQL执行
	fmt.Printf("Executing SQL: %v\n", command.GetCommand())
	fmt.Printf("Parameters: %d\n", len(command.GetParameters()))
	return nil
}

// executeRedisCommand 执行Redis命令
func (c *SimpleBatchSQLClient) executeRedisCommand(ctx context.Context, conn interface{}, command BatchCommand) error {
	// 模拟Redis执行
	fmt.Printf("Executing Redis commands: %v\n", command.GetCommand())
	return nil
}

// executeMongoCommand 执行MongoDB命令
func (c *SimpleBatchSQLClient) executeMongoCommand(ctx context.Context, conn interface{}, command BatchCommand) error {
	// 模拟MongoDB执行
	fmt.Printf("Executing MongoDB operations: %v\n", command.GetCommand())
	return nil
}

// CreateSchema 创建Schema的便捷方法
func (c *SimpleBatchSQLClient) CreateSchema(identifier string, conflictStrategy ConflictStrategy, driver DatabaseDriver, columns ...string) SchemaInterface {
	return NewUniversalSchema(identifier, conflictStrategy, driver, columns...)
}

// GetMetrics 获取执行指标
func (c *SimpleBatchSQLClient) GetMetrics() map[string]interface{} {
	if c.metricsCollector == nil {
		return map[string]interface{}{
			"metrics_enabled": false,
		}
	}

	return c.metricsCollector.GetMetrics()
}

// Close 关闭客户端
func (c *SimpleBatchSQLClient) Close() error {
	return c.connectionManager.Close()
}

// AddConnection 添加数据库连接
func (c *SimpleBatchSQLClient) AddConnection(driverName string, config *ConnectionConfig) error {
	if err := c.connectionManager.AddConnection(driverName, config); err != nil {
		return fmt.Errorf("failed to add connection: %w", err)
	}

	// 更新客户端配置
	c.config.Connections[driverName] = config
	return nil
}

// HealthCheck 健康检查
func (c *SimpleBatchSQLClient) HealthCheck(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	// 检查连接状态
	connectionHealth := make(map[string]interface{})
	for driverName := range c.config.Connections {
		conn, err := c.connectionManager.GetConnection(driverName)
		if err != nil {
			connectionHealth[driverName] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
			health["status"] = "degraded"
		} else {
			connectionHealth[driverName] = map[string]interface{}{
				"status":    "healthy",
				"connected": conn != nil,
			}
		}
	}

	health["connections"] = connectionHealth

	// 添加指标摘要
	if c.metricsCollector != nil {
		health["metrics_summary"] = c.metricsCollector.GetSummary()
	}

	return health
}
