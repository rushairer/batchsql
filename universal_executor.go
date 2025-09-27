package batchsql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// RedisClient Redis 客户端接口（避免直接依赖）
type RedisClient interface {
	Pipeline() RedisPipeline
}

// RedisPipeline Redis 管道接口
type RedisPipeline interface {
	Do(ctx context.Context, args ...interface{}) error
	Exec(ctx context.Context) ([]interface{}, error)
}

// MongoClient MongoDB 客户端接口（避免直接依赖）
type MongoClient interface {
	Database(name string) MongoDatabase
}

// MongoDatabase MongoDB 数据库接口
type MongoDatabase interface {
	Collection(name string) MongoCollection
}

// MongoCollection MongoDB 集合接口
type MongoCollection interface {
	InsertOne(ctx context.Context, document interface{}) (interface{}, error)
	UpdateOne(ctx context.Context, filter, update interface{}, opts ...interface{}) (interface{}, error)
	ReplaceOne(ctx context.Context, filter, replacement interface{}, opts ...interface{}) (interface{}, error)
}

// UniversalBatchExecutor 通用批量执行器
type UniversalBatchExecutor struct {
	connectionManager ConnectionManager
	metricsCollector  MetricsCollector
	supportedDrivers  map[string]bool
}

// NewUniversalBatchExecutor 创建通用批量执行器
func NewUniversalBatchExecutor(connectionManager ConnectionManager) *UniversalBatchExecutor {
	return &UniversalBatchExecutor{
		connectionManager: connectionManager,
		supportedDrivers: map[string]bool{
			"mysql":              true,
			"postgresql":         true,
			"sqlite":             true,
			"redis":              true,
			"redis-hash":         true,
			"redis-set":          true,
			"mongodb":            true,
			"mongodb-timeseries": true,
		},
	}
}

// SetMetricsCollector 设置指标收集器
func (e *UniversalBatchExecutor) SetMetricsCollector(collector MetricsCollector) {
	e.metricsCollector = collector
}

// ExecuteBatch 执行批量操作
func (e *UniversalBatchExecutor) ExecuteBatch(ctx context.Context, commands []BatchCommand) error {
	if len(commands) == 0 {
		return nil
	}

	startTime := time.Now()

	// 处理每个命令
	for _, command := range commands {
		if err := e.executeCommand(ctx, command.GetMetadata()["driver"].(string), command); err != nil {
			duration := time.Since(startTime).Milliseconds()
			if e.metricsCollector != nil {
				e.metricsCollector.RecordBatchExecution(
					command.GetMetadata()["driver"].(string),
					1, // 单个命令
					duration,
					err,
				)
			}
			return fmt.Errorf("failed to execute command: %w", err)
		}
	}

	duration := time.Since(startTime).Milliseconds()
	if e.metricsCollector != nil {
		for _, command := range commands {
			e.metricsCollector.RecordBatchExecution(
				command.GetMetadata()["driver"].(string),
				1,
				duration,
				nil,
			)
		}
	}

	return nil
}

// ExecuteBatchRequests 执行批量请求（保持向后兼容）
func (e *UniversalBatchExecutor) ExecuteBatchRequests(ctx context.Context, batchData []*Request) error {
	if len(batchData) == 0 {
		return nil
	}

	startTime := time.Now()

	// 按 schema 分组
	schemaGroups := e.groupBySchema(batchData)

	// 处理每个 schema 组
	for schema, requests := range schemaGroups {
		if err := e.processSchemaGroup(ctx, schema, requests); err != nil {
			duration := time.Since(startTime).Milliseconds()
			if e.metricsCollector != nil {
				e.metricsCollector.RecordBatchExecution(
					schema.GetDatabaseDriver().GetName(),
					len(requests),
					duration,
					err,
				)
			}
			return fmt.Errorf("failed to process schema group %s: %w", schema.GetIdentifier(), err)
		}
	}

	duration := time.Since(startTime).Milliseconds()
	if e.metricsCollector != nil {
		for schema, requests := range schemaGroups {
			e.metricsCollector.RecordBatchExecution(
				schema.GetDatabaseDriver().GetName(),
				len(requests),
				duration,
				nil,
			)
		}
	}

	return nil
}

// groupBySchema 按 schema 分组
func (e *UniversalBatchExecutor) groupBySchema(batchData []*Request) map[SchemaInterface][]*Request {
	groups := make(map[SchemaInterface][]*Request)

	for _, request := range batchData {
		// 暂时跳过类型转换，这需要修改 Request 结构
		// 这里假设我们有一个方法可以获取 SchemaInterface
		if request.schema != nil {
			// 创建一个临时的 UniversalSchema 来包装现有的 Schema
			universalSchema := &UniversalSchema{
				identifier:       request.schema.TableName(),
				conflictStrategy: request.schema.ConflictStrategy(),
				columns:          request.schema.Columns(),
				// driver 需要根据 DatabaseType 创建
				metadata: make(map[string]interface{}),
			}
			groups[universalSchema] = append(groups[universalSchema], request)
		}
	}

	return groups
}

// processSchemaGroup 处理 schema 组
func (e *UniversalBatchExecutor) processSchemaGroup(ctx context.Context, schema SchemaInterface, requests []*Request) error {
	driver := schema.GetDatabaseDriver()
	driverName := driver.GetName()

	// 检查驱动支持
	if !e.supportedDrivers[driverName] {
		return fmt.Errorf("unsupported driver: %s", driverName)
	}

	// 生成批量命令
	command, err := driver.GenerateBatchCommand(schema, requests)
	if err != nil {
		return fmt.Errorf("failed to generate batch command: %w", err)
	}

	// 执行命令
	return e.executeCommand(ctx, driverName, command)
}

// executeCommand 执行命令
func (e *UniversalBatchExecutor) executeCommand(ctx context.Context, driverName string, command BatchCommand) error {
	conn, err := e.connectionManager.GetConnection(driverName)
	if err != nil {
		return fmt.Errorf("failed to get connection for %s: %w", driverName, err)
	}
	defer e.connectionManager.ReleaseConnection(driverName, conn)

	switch command.GetCommandType() {
	case "SQL":
		return e.executeSQLCommand(ctx, conn, command)
	case "REDIS":
		return e.executeRedisCommand(ctx, conn, command)
	case "MONGODB":
		return e.executeMongoCommand(ctx, conn, command)
	default:
		return fmt.Errorf("unsupported command type: %s", command.GetCommandType())
	}
}

// executeSQLCommand 执行 SQL 命令
func (e *UniversalBatchExecutor) executeSQLCommand(ctx context.Context, conn interface{}, command BatchCommand) error {
	db, ok := conn.(*sql.DB)
	if !ok {
		return fmt.Errorf("invalid SQL connection type")
	}

	sqlStr, ok := command.GetCommand().(string)
	if !ok {
		return fmt.Errorf("invalid SQL command format")
	}

	parameters := command.GetParameters()
	result, err := db.ExecContext(ctx, sqlStr, parameters...)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	metadata := command.GetMetadata()
	log.Printf("SQL execution completed - Driver: %s, Table: %s, Rows affected: %d",
		metadata["driver"], metadata["table"], rowsAffected)

	return nil
}

// executeRedisCommand 执行 Redis 命令
func (e *UniversalBatchExecutor) executeRedisCommand(ctx context.Context, conn interface{}, command BatchCommand) error {
	client, ok := conn.(RedisClient)
	if !ok {
		return fmt.Errorf("invalid Redis connection type")
	}

	commands, ok := command.GetCommand().([][]interface{})
	if !ok {
		return fmt.Errorf("invalid Redis command format")
	}

	// 使用 Pipeline 批量执行
	pipe := client.Pipeline()
	for _, cmd := range commands {
		if len(cmd) < 2 {
			continue
		}
		pipe.Do(ctx, cmd...)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute Redis pipeline: %w", err)
	}

	metadata := command.GetMetadata()
	log.Printf("Redis execution completed - Driver: %s, Key prefix: %s, Commands: %d",
		metadata["driver"], metadata["key_prefix"], len(commands))

	return nil
}

// executeMongoCommand 执行 MongoDB 命令
func (e *UniversalBatchExecutor) executeMongoCommand(ctx context.Context, conn interface{}, command BatchCommand) error {
	client, ok := conn.(MongoClient)
	if !ok {
		return fmt.Errorf("invalid MongoDB connection type")
	}

	operations, ok := command.GetCommand().([]interface{})
	if !ok {
		return fmt.Errorf("invalid MongoDB command format")
	}

	metadata := command.GetMetadata()
	collectionName, ok := metadata["collection"].(string)
	if !ok {
		return fmt.Errorf("missing collection name in metadata")
	}

	// 假设使用默认数据库，实际使用中应该从配置获取
	database := client.Database("default")
	collection := database.Collection(collectionName)

	// 执行批量操作
	for _, op := range operations {
		mongoOp, ok := op.(*MongoOperation)
		if !ok {
			continue
		}

		switch mongoOp.Type {
		case "insert":
			_, err := collection.InsertOne(ctx, mongoOp.Document)
			if err != nil {
				return fmt.Errorf("failed to insert document: %w", err)
			}
		case "update":
			// 简化版本，不使用 options
			_, err := collection.UpdateOne(ctx, mongoOp.Filter, mongoOp.Update)
			if err != nil {
				return fmt.Errorf("failed to update document: %w", err)
			}
		case "replace":
			// 简化版本，不使用 options
			_, err := collection.ReplaceOne(ctx, mongoOp.Filter, mongoOp.Document)
			if err != nil {
				return fmt.Errorf("failed to replace document: %w", err)
			}
		}
	}

	log.Printf("MongoDB execution completed - Driver: %s, Collection: %s, Operations: %d",
		metadata["driver"], collectionName, len(operations))

	return nil
}

// GetSupportedDrivers 获取支持的驱动
func (e *UniversalBatchExecutor) GetSupportedDrivers() []string {
	drivers := make([]string, 0, len(e.supportedDrivers))
	for driver := range e.supportedDrivers {
		drivers = append(drivers, driver)
	}
	return drivers
}

// Close 关闭执行器
func (e *UniversalBatchExecutor) Close() error {
	if e.connectionManager != nil {
		return e.connectionManager.Close()
	}
	return nil
}

// MongoOperation MongoDB 操作结构（从 mongodb_driver.go 移动到这里以避免循环依赖）
type MongoOperation struct {
	Type       string                 `json:"type"`
	Filter     map[string]interface{} `json:"filter"`
	Document   map[string]interface{} `json:"document"`
	Update     map[string]interface{} `json:"update"`
	Upsert     bool                   `json:"upsert"`
	Collection string                 `json:"collection"`
}
