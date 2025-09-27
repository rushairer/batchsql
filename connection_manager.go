package batchsql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	DriverName      string                 `json:"driver_name"`
	ConnectionURL   string                 `json:"connection_url"`
	MaxOpenConns    int                    `json:"max_open_conns"`
	MaxIdleConns    int                    `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration          `json:"conn_max_lifetime"`
	Options         map[string]interface{} `json:"options"`
}

// DefaultConnectionManager 默认连接管理器实现
type DefaultConnectionManager struct {
	connections map[string]interface{}
	configs     map[string]*ConnectionConfig
	mutex       sync.RWMutex
}

// NewDefaultConnectionManager 创建默认连接管理器
func NewDefaultConnectionManager() *DefaultConnectionManager {
	return &DefaultConnectionManager{
		connections: make(map[string]interface{}),
		configs:     make(map[string]*ConnectionConfig),
	}
}

// AddConnection 添加连接配置
func (cm *DefaultConnectionManager) AddConnection(driverName string, config *ConnectionConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.configs[driverName] = config
	return nil
}

// GetConnection 获取连接
func (cm *DefaultConnectionManager) GetConnection(driverName string) (interface{}, error) {
	cm.mutex.RLock()
	if conn, exists := cm.connections[driverName]; exists {
		cm.mutex.RUnlock()
		return conn, nil
	}
	cm.mutex.RUnlock()

	// 创建新连接
	return cm.createConnection(driverName)
}

// createConnection 创建新连接
func (cm *DefaultConnectionManager) createConnection(driverName string) (interface{}, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 双重检查
	if conn, exists := cm.connections[driverName]; exists {
		return conn, nil
	}

	config, exists := cm.configs[driverName]
	if !exists {
		return nil, fmt.Errorf("no configuration found for driver: %s", driverName)
	}

	var conn interface{}
	var err error

	switch {
	case driverName == "mysql" || driverName == "postgresql" || driverName == "sqlite":
		conn, err = cm.createSQLConnection(config)
	case driverName == "redis" || driverName == "redis-hash" || driverName == "redis-set":
		conn, err = cm.createRedisConnection(config)
	case driverName == "mongodb" || driverName == "mongodb-timeseries":
		conn, err = cm.createMongoConnection(config)
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverName)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create connection for %s: %w", driverName, err)
	}

	cm.connections[driverName] = conn
	return conn, nil
}

// createSQLConnection 创建SQL连接
func (cm *DefaultConnectionManager) createSQLConnection(config *ConnectionConfig) (*sql.DB, error) {
	db, err := sql.Open(config.DriverName, config.ConnectionURL)
	if err != nil {
		return nil, err
	}

	// 配置连接池
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// createRedisConnection 创建Redis连接（模拟实现）
func (cm *DefaultConnectionManager) createRedisConnection(config *ConnectionConfig) (RedisClient, error) {
	// 这里应该使用实际的Redis客户端，如 go-redis
	// 为了避免依赖，这里返回一个模拟实现
	return &MockRedisClient{
		connectionURL: config.ConnectionURL,
		options:       config.Options,
	}, nil
}

// createMongoConnection 创建MongoDB连接（模拟实现）
func (cm *DefaultConnectionManager) createMongoConnection(config *ConnectionConfig) (MongoClient, error) {
	// 这里应该使用实际的MongoDB客户端，如 mongo-driver
	// 为了避免依赖，这里返回一个模拟实现
	return &MockMongoClient{
		connectionURL: config.ConnectionURL,
		options:       config.Options,
	}, nil
}

// ReleaseConnection 释放连接（在连接池模式下通常不需要显式释放）
func (cm *DefaultConnectionManager) ReleaseConnection(driverName string, conn interface{}) error {
	// 在连接池模式下，连接会自动管理
	// 这里可以添加一些清理逻辑
	return nil
}

// Close 关闭所有连接
func (cm *DefaultConnectionManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var lastErr error

	for driverName, conn := range cm.connections {
		switch c := conn.(type) {
		case *sql.DB:
			if err := c.Close(); err != nil {
				lastErr = fmt.Errorf("failed to close SQL connection %s: %w", driverName, err)
			}
		case RedisClient:
			// Redis客户端通常有Close方法
			if closer, ok := c.(interface{ Close() error }); ok {
				if err := closer.Close(); err != nil {
					lastErr = fmt.Errorf("failed to close Redis connection %s: %w", driverName, err)
				}
			}
		case MongoClient:
			// MongoDB客户端通常有Disconnect方法
			if disconnector, ok := c.(interface{ Disconnect() error }); ok {
				if err := disconnector.Disconnect(); err != nil {
					lastErr = fmt.Errorf("failed to close MongoDB connection %s: %w", driverName, err)
				}
			}
		}
	}

	// 清空连接映射
	cm.connections = make(map[string]interface{})
	cm.configs = make(map[string]*ConnectionConfig)

	return lastErr
}

// GetConnectionInfo 获取连接信息
func (cm *DefaultConnectionManager) GetConnectionInfo() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	info := make(map[string]interface{})
	for driverName, config := range cm.configs {
		info[driverName] = map[string]interface{}{
			"driver_name":       config.DriverName,
			"max_open_conns":    config.MaxOpenConns,
			"max_idle_conns":    config.MaxIdleConns,
			"conn_max_lifetime": config.ConnMaxLifetime,
			"connected":         cm.connections[driverName] != nil,
		}
	}

	return info
}

// MockRedisClient Redis客户端模拟实现
type MockRedisClient struct {
	connectionURL string
	options       map[string]interface{}
}

func (c *MockRedisClient) Pipeline() RedisPipeline {
	return &MockRedisPipeline{
		commands: make([][]interface{}, 0),
	}
}

// MockRedisPipeline Redis管道模拟实现
type MockRedisPipeline struct {
	commands [][]interface{}
}

func (p *MockRedisPipeline) Do(ctx context.Context, args ...interface{}) error {
	p.commands = append(p.commands, args)
	return nil
}

func (p *MockRedisPipeline) Exec(ctx context.Context) ([]interface{}, error) {
	// 模拟执行结果
	results := make([]interface{}, len(p.commands))
	for i := range p.commands {
		results[i] = "OK"
	}
	return results, nil
}

// MockMongoClient MongoDB客户端模拟实现
type MockMongoClient struct {
	connectionURL string
	options       map[string]interface{}
}

func (c *MockMongoClient) Database(name string) MongoDatabase {
	return &MockMongoDatabase{
		name:   name,
		client: c,
	}
}

func (c *MockMongoClient) Disconnect() error {
	return nil
}

// MockMongoDatabase MongoDB数据库模拟实现
type MockMongoDatabase struct {
	name   string
	client *MockMongoClient
}

func (db *MockMongoDatabase) Collection(name string) MongoCollection {
	return &MockMongoCollection{
		name:     name,
		database: db,
	}
}

// MockMongoCollection MongoDB集合模拟实现
type MockMongoCollection struct {
	name     string
	database *MockMongoDatabase
}

func (c *MockMongoCollection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	return map[string]interface{}{"InsertedID": "mock_id"}, nil
}

func (c *MockMongoCollection) UpdateOne(ctx context.Context, filter, update interface{}, opts ...interface{}) (interface{}, error) {
	return map[string]interface{}{"ModifiedCount": 1}, nil
}

func (c *MockMongoCollection) ReplaceOne(ctx context.Context, filter, replacement interface{}, opts ...interface{}) (interface{}, error) {
	return map[string]interface{}{"ModifiedCount": 1}, nil
}
