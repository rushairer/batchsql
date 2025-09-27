package batchsql

import "context"

// DatabaseDriver 数据库驱动接口
type DatabaseDriver interface {
	// GetName 获取数据库类型名称
	GetName() string

	// GenerateBatchCommand 生成批量操作命令
	GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error)

	// SupportedConflictStrategies 获取支持的冲突策略
	SupportedConflictStrategies() []ConflictStrategy

	// ValidateSchema 验证 schema 配置
	ValidateSchema(schema SchemaInterface) error
}

// SchemaInterface schema 接口
type SchemaInterface interface {
	// 基本信息
	GetIdentifier() string // 表名/集合名/键前缀等
	GetConflictStrategy() ConflictStrategy
	GetColumns() []string
	GetDatabaseDriver() DatabaseDriver

	// 验证
	Validate() error

	// 克隆
	Clone() SchemaInterface
}

// BatchCommand 批量操作命令接口
type BatchCommand interface {
	// GetCommandType 获取命令类型（SQL、Redis命令、MongoDB操作等）
	GetCommandType() string

	// GetCommand 获取具体命令内容
	GetCommand() interface{}

	// GetParameters 获取参数
	GetParameters() []interface{}

	// GetMetadata 获取元数据
	GetMetadata() map[string]interface{}
}

// BatchExecutorInterface 批量执行器接口
type BatchExecutorInterface interface {
	// ExecuteBatch 执行批量操作
	ExecuteBatch(ctx context.Context, commands []BatchCommand) error

	// GetSupportedDrivers 获取支持的驱动类型
	GetSupportedDrivers() []string

	// Close 关闭连接
	Close() error
}

// ConnectionManager 连接管理器接口
type ConnectionManager interface {
	// GetConnection 获取连接
	GetConnection(driverName string) (interface{}, error)

	// ReleaseConnection 释放连接
	ReleaseConnection(driverName string, conn interface{}) error

	// AddConnection 添加连接配置
	AddConnection(driverName string, config *ConnectionConfig) error

	// Close 关闭所有连接
	Close() error
}

// DataTransformer 数据转换器接口
type DataTransformer interface {
	// TransformRequest 转换请求数据
	TransformRequest(request *Request, schema SchemaInterface) (interface{}, error)

	// TransformBatch 转换批量数据
	TransformBatch(requests []*Request, schema SchemaInterface) ([]interface{}, error)
}

// ConflictResolver 冲突解决器接口
type ConflictResolver interface {
	// ResolveConflict 解决冲突
	ResolveConflict(strategy ConflictStrategy, existing, new interface{}) (interface{}, error)

	// SupportedStrategies 支持的策略
	SupportedStrategies() []ConflictStrategy
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordBatchExecution 记录批量执行
	RecordBatchExecution(driverName string, batchSize int, duration int64, err error)

	// RecordRequestCount 记录请求数量
	RecordRequestCount(driverName string, count int)

	// GetMetrics 获取指标
	GetMetrics() map[string]interface{}
}
