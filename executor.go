package batchsql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BatchExecutor 批量执行器实现
type BatchExecutor struct {
	connections map[string]interface{}
	mutex       sync.RWMutex
}

// NewBatchExecutor 创建新的批量执行器
func NewBatchExecutor() *BatchExecutor {
	return &BatchExecutor{
		connections: make(map[string]interface{}),
	}
}

// AddSQLConnection 添加SQL数据库连接
func (e *BatchExecutor) AddSQLConnection(driverName, dsn string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var db *sql.DB
	var err error

	switch driverName {
	case "mysql":
		db, err = sql.Open("mysql", dsn)
	case "postgresql", "postgres":
		db, err = sql.Open("postgres", dsn)
	default:
		return fmt.Errorf("unsupported SQL driver: %s", driverName)
	}

	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	e.connections[driverName] = db
	return nil
}

// AddRedisConnection 添加Redis连接
func (e *BatchExecutor) AddRedisConnection(addr, password string, db int) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	e.connections["redis"] = client
	return nil
}

// AddMongoConnection 添加MongoDB连接
func (e *BatchExecutor) AddMongoConnection(uri, database string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	mongoConn := &MongoConnection{
		Client:   client,
		Database: database,
	}

	e.connections["mongodb"] = mongoConn
	return nil
}

// MongoConnection MongoDB连接封装
type MongoConnection struct {
	Client   *mongo.Client
	Database string
}

// ExecuteBatch 执行批量操作
func (e *BatchExecutor) ExecuteBatch(ctx context.Context, commands []BatchCommand) error {
	if len(commands) == 0 {
		return nil
	}

	// 按驱动类型分组命令
	commandsByDriver := make(map[string][]BatchCommand)
	for _, cmd := range commands {
		metadata := cmd.GetMetadata()
		if driver, ok := metadata["driver"].(string); ok {
			commandsByDriver[driver] = append(commandsByDriver[driver], cmd)
		}
	}

	// 并发执行不同驱动的命令
	var wg sync.WaitGroup
	errChan := make(chan error, len(commandsByDriver))

	for driverName, driverCommands := range commandsByDriver {
		wg.Add(1)
		go func(driver string, cmds []BatchCommand) {
			defer wg.Done()
			if err := e.executeDriverCommands(ctx, driver, cmds); err != nil {
				errChan <- fmt.Errorf("driver %s: %w", driver, err)
			}
		}(driverName, driverCommands)
	}

	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch execution failed: %v", errors)
	}

	return nil
}

// executeDriverCommands 执行特定驱动的命令
func (e *BatchExecutor) executeDriverCommands(ctx context.Context, driverName string, commands []BatchCommand) error {
	e.mutex.RLock()
	conn, exists := e.connections[driverName]
	e.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("no connection found for driver: %s", driverName)
	}

	switch driverName {
	case "mysql", "postgresql", "postgres":
		return e.executeSQLCommands(ctx, conn.(*sql.DB), commands)
	case "redis":
		return e.executeRedisCommands(ctx, conn.(*redis.Client), commands)
	case "mongodb":
		return e.executeMongoCommands(ctx, conn.(*MongoConnection), commands)
	default:
		return fmt.Errorf("unsupported driver: %s", driverName)
	}
}

// executeSQLCommands 执行SQL命令
func (e *BatchExecutor) executeSQLCommands(ctx context.Context, db *sql.DB, commands []BatchCommand) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, cmd := range commands {
		sqlStr, ok := cmd.GetCommand().(string)
		if !ok {
			return fmt.Errorf("invalid SQL command type")
		}

		params := cmd.GetParameters()
		if _, err := tx.ExecContext(ctx, sqlStr, params...); err != nil {
			return fmt.Errorf("failed to execute SQL: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// executeRedisCommands 执行Redis命令
func (e *BatchExecutor) executeRedisCommands(ctx context.Context, client *redis.Client, commands []BatchCommand) error {
	pipe := client.Pipeline()

	for _, cmd := range commands {
		redisCmd, ok := cmd.GetCommand().([]interface{})
		if !ok {
			return fmt.Errorf("invalid Redis command type")
		}

		pipe.Do(ctx, redisCmd...)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// executeMongoCommands 执行MongoDB命令
func (e *BatchExecutor) executeMongoCommands(ctx context.Context, conn *MongoConnection, commands []BatchCommand) error {
	db := conn.Client.Database(conn.Database)

	for _, cmd := range commands {
		mongoOp, ok := cmd.GetCommand().(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid MongoDB command type")
		}

		collection := db.Collection(mongoOp["collection"].(string))
		operation := mongoOp["operation"].(string)
		documents := mongoOp["documents"].([]interface{})

		switch operation {
		case "insertMany":
			if _, err := collection.InsertMany(ctx, documents); err != nil {
				return fmt.Errorf("failed to insert documents: %w", err)
			}
		case "updateMany":
			// 实现更新逻辑
		case "replaceMany":
			// 实现替换逻辑
		default:
			return fmt.Errorf("unsupported MongoDB operation: %s", operation)
		}
	}

	return nil
}

// GetSupportedDrivers 获取支持的驱动
func (e *BatchExecutor) GetSupportedDrivers() []string {
	return []string{"mysql", "postgresql", "postgres", "redis", "mongodb"}
}

// Close 关闭所有连接
func (e *BatchExecutor) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var errors []error

	for driverName, conn := range e.connections {
		switch driverName {
		case "mysql", "postgresql", "postgres":
			if db, ok := conn.(*sql.DB); ok {
				if err := db.Close(); err != nil {
					errors = append(errors, fmt.Errorf("failed to close %s: %w", driverName, err))
				}
			}
		case "redis":
			if client, ok := conn.(*redis.Client); ok {
				if err := client.Close(); err != nil {
					errors = append(errors, fmt.Errorf("failed to close Redis: %w", err))
				}
			}
		case "mongodb":
			if mongoConn, ok := conn.(*MongoConnection); ok {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := mongoConn.Client.Disconnect(ctx); err != nil {
					errors = append(errors, fmt.Errorf("failed to close MongoDB: %w", err))
				}
				cancel()
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}