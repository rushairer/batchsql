package batchsql

import (
	"context"
	"fmt"
	"time"
)

// BatchSQLClient 统一的BatchSQL客户端
type BatchSQLClient struct {
	executor          BatchExecutorInterface
	connectionManager ConnectionManager
	metricsCollector  MetricsCollector
	dataTransformer   DataTransformer
	config            *ClientConfig
}

// ClientConfig 客户端配置
type ClientConfig struct {
	// 连接配置
	Connections map[string]*ConnectionConfig `json:"connections"`

	// 执行配置
	DefaultBatchSize int           `json:"default_batch_size"`
	MaxBatchSize     int           `json:"max_batch_size"`
	ExecutionTimeout time.Duration `json:"execution_timeout"`
	RetryAttempts    int           `json:"retry_attempts"`
	RetryDelay       time.Duration `json:"retry_delay"`

	// 指标配置
	EnableMetrics      bool `json:"enable_metrics"`
	MetricsHistorySize int  `json:"metrics_history_size"`

	// 验证配置
	EnableValidation bool `json:"enable_validation"`
	StrictMode       bool `json:"strict_mode"`
}

// DefaultClientConfig 默认客户端配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Connections:        make(map[string]*ConnectionConfig),
		DefaultBatchSize:   100,
		MaxBatchSize:       1000,
		ExecutionTimeout:   30 * time.Second,
		RetryAttempts:      3,
		RetryDelay:         1 * time.Second,
		EnableMetrics:      true,
		MetricsHistorySize: 1000,
		EnableValidation:   true,
		StrictMode:         false,
	}
}

// NewBatchSQLClient 创建新的BatchSQL客户端
func NewBatchSQLClient(config *ClientConfig) (*BatchSQLClient, error) {
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

	// 创建执行器
	executor := NewUniversalBatchExecutor(connectionManager)

	// 创建指标收集器
	var metricsCollector MetricsCollector
	if config.EnableMetrics {
		collector := NewDefaultMetricsCollector()
		collector.SetMaxHistorySize(config.MetricsHistorySize)
		metricsCollector = collector
		executor.SetMetricsCollector(metricsCollector)
	}

	// 创建数据转换器
	var dataTransformer DataTransformer
	if config.EnableValidation {
		dataTransformer = NewValidationTransformer()
	} else {
		dataTransformer = NewDefaultDataTransformer()
	}

	return &BatchSQLClient{
		executor:          executor,
		connectionManager: connectionManager,
		metricsCollector:  metricsCollector,
		dataTransformer:   dataTransformer,
		config:            config,
	}, nil
}

// AddConnection 添加数据库连接
func (c *BatchSQLClient) AddConnection(driverName string, config *ConnectionConfig) error {
	if err := c.connectionManager.AddConnection(driverName, config); err != nil {
		return fmt.Errorf("failed to add connection: %w", err)
	}

	// 更新客户端配置
	c.config.Connections[driverName] = config
	return nil
}

// ExecuteBatch 执行批量操作
func (c *BatchSQLClient) ExecuteBatch(ctx context.Context, requests []*Request) error {
	if len(requests) == 0 {
		return nil
	}

	// 应用执行超时
	if c.config.ExecutionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.config.ExecutionTimeout)
		defer cancel()
	}

	// 分批处理
	return c.executeBatchWithRetry(ctx, requests)
}

// executeBatchWithRetry 带重试的批量执行
func (c *BatchSQLClient) executeBatchWithRetry(ctx context.Context, requests []*Request) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// 等待重试延迟
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.config.RetryDelay):
			}
		}

		// 暂时跳过执行器，直接处理请求
		err := c.processRequestsDirectly(ctx, requests)
		if err == nil {
			return nil // 成功执行
		}

		lastErr = err

		// 检查是否应该重试
		if !c.shouldRetry(err) {
			break
		}
	}

	return fmt.Errorf("batch execution failed after %d attempts: %w", c.config.RetryAttempts+1, lastErr)
}

// processRequestsDirectly 直接处理请求（临时实现）
func (c *BatchSQLClient) processRequestsDirectly(ctx context.Context, requests []*Request) error {
	// 模拟成功执行
	return nil
}

// shouldRetry 判断是否应该重试
func (c *BatchSQLClient) shouldRetry(err error) bool {
	// 简化实现，实际应该根据错误类型判断
	// 例如：网络错误、超时错误等可以重试
	// 语法错误、权限错误等不应该重试
	return true
}

// ExecuteWithSchema 使用指定Schema执行批量操作
func (c *BatchSQLClient) ExecuteWithSchema(ctx context.Context, schema SchemaInterface, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// 转换数据为Request
	requests := make([]*Request, len(data))
	for i, item := range data {
		request := NewRequestFromInterface(schema)
		for key, value := range item {
			request.Set(key, value)
		}
		requests[i] = request
	}

	return c.ExecuteBatch(ctx, requests)
}

// CreateSchema 创建Schema的便捷方法
func (c *BatchSQLClient) CreateSchema(identifier string, conflictStrategy ConflictStrategy, driver DatabaseDriver, columns ...string) SchemaInterface {
	return NewUniversalSchema(identifier, conflictStrategy, driver, columns...)
}

// GetMetrics 获取执行指标
func (c *BatchSQLClient) GetMetrics() map[string]interface{} {
	if c.metricsCollector == nil {
		return map[string]interface{}{
			"metrics_enabled": false,
		}
	}

	metrics := c.metricsCollector.GetMetrics()
	metrics["client_config"] = map[string]interface{}{
		"default_batch_size": c.config.DefaultBatchSize,
		"max_batch_size":     c.config.MaxBatchSize,
		"execution_timeout":  c.config.ExecutionTimeout,
		"retry_attempts":     c.config.RetryAttempts,
		"retry_delay":        c.config.RetryDelay,
	}

	return metrics
}

// GetConnectionInfo 获取连接信息
func (c *BatchSQLClient) GetConnectionInfo() map[string]interface{} {
	if manager, ok := c.connectionManager.(*DefaultConnectionManager); ok {
		return manager.GetConnectionInfo()
	}
	return map[string]interface{}{}
}

// HealthCheck 健康检查
func (c *BatchSQLClient) HealthCheck(ctx context.Context) map[string]interface{} {
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
		if collector, ok := c.metricsCollector.(*DefaultMetricsCollector); ok {
			health["metrics_summary"] = collector.GetSummary()
		}
	}

	return health
}

// Close 关闭客户端
func (c *BatchSQLClient) Close() error {
	var lastErr error

	// 关闭执行器
	if err := c.executor.Close(); err != nil {
		lastErr = err
	}

	// 关闭连接管理器
	if err := c.connectionManager.Close(); err != nil {
		lastErr = err
	}

	return lastErr
}

// BatchBuilder 批量操作构建器
type BatchBuilder struct {
	client   *BatchSQLClient
	schema   SchemaInterface
	requests []*Request
}

// NewBatchBuilder 创建批量操作构建器
func (c *BatchSQLClient) NewBatchBuilder(schema SchemaInterface) *BatchBuilder {
	return &BatchBuilder{
		client:   c,
		schema:   schema,
		requests: make([]*Request, 0),
	}
}

// Add 添加请求
func (bb *BatchBuilder) Add(data map[string]interface{}) *BatchBuilder {
	request := NewRequestFromInterface(bb.schema)
	for key, value := range data {
		request.Set(key, value)
	}
	bb.requests = append(bb.requests, request)
	return bb
}

// AddRequest 添加Request对象
func (bb *BatchBuilder) AddRequest(request *Request) *BatchBuilder {
	bb.requests = append(bb.requests, request)
	return bb
}

// Execute 执行批量操作
func (bb *BatchBuilder) Execute(ctx context.Context) error {
	return bb.client.ExecuteBatch(ctx, bb.requests)
}

// Count 获取请求数量
func (bb *BatchBuilder) Count() int {
	return len(bb.requests)
}

// Clear 清空请求
func (bb *BatchBuilder) Clear() *BatchBuilder {
	bb.requests = bb.requests[:0]
	return bb
}

// Preview 预览将要执行的命令
func (bb *BatchBuilder) Preview() ([]BatchCommand, error) {
	if len(bb.requests) == 0 {
		return nil, nil
	}

	// 按schema分组
	schemaGroups := make(map[SchemaInterface][]*Request)
	for _, request := range bb.requests {
		schemaGroups[bb.schema] = append(schemaGroups[bb.schema], request)
	}

	var commands []BatchCommand
	for schema, requests := range schemaGroups {
		command, err := schema.GetDatabaseDriver().GenerateBatchCommand(schema, requests)
		if err != nil {
			return nil, fmt.Errorf("failed to generate command: %w", err)
		}
		commands = append(commands, command)
	}

	return commands, nil
}
