// Package main provides advanced BatchSQL usage examples
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers"
)

func main() {
	// 创建客户端
	client := batchsql.NewClient()
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("关闭客户端时出错: %v", err)
		}
	}()

	// 获取执行器并添加数据库连接
	executor := client.GetExecutor()

	// 添加MySQL连接
	if batchExecutor, ok := executor.(*batchsql.BatchExecutor); ok {
		err := batchExecutor.AddSQLConnection("mysql", "user:password@tcp(localhost:3306)/testdb")
		if err != nil {
			log.Printf("添加MySQL连接失败: %v", err)
		}

		// 添加Redis连接
		err = batchExecutor.AddRedisConnection("localhost:6379", "", 0)
		if err != nil {
			log.Printf("添加Redis连接失败: %v", err)
		}

		// 添加MongoDB连接
		err = batchExecutor.AddMongoConnection("mongodb://localhost:27017", "testdb")
		if err != nil {
			log.Printf("添加MongoDB连接失败: %v", err)
		}
	}

	ctx := context.Background()

	// 示例1: 基本批量操作
	fmt.Println("=== 示例1: 基本批量操作 ===")
	if err := basicBatchExample(ctx, client); err != nil {
		log.Printf("基本批量操作失败: %v", err)
	}

	// 示例2: 大批量数据处理
	fmt.Println("\n=== 示例2: 大批量数据处理 ===")
	if err := largeBatchExample(ctx, client); err != nil {
		log.Printf("大批量数据处理失败: %v", err)
	}

	// 示例3: 流式批处理
	fmt.Println("\n=== 示例3: 流式批处理 ===")
	if err := streamBatchExample(ctx, client); err != nil {
		log.Printf("流式批处理失败: %v", err)
	}

	// 示例4: 多数据库批处理
	fmt.Println("\n=== 示例4: 多数据库批处理 ===")
	if err := multiDatabaseExample(ctx, client); err != nil {
		log.Printf("多数据库批处理失败: %v", err)
	}
}

// basicBatchExample 基本批量操作示例
func basicBatchExample(ctx context.Context, client *batchsql.Client) error {
	// 创建MySQL Schema
	mysqlDriver := drivers.NewMySQLDriver()
	schema := client.CreateSchema("users", batchsql.ConflictUpdate, mysqlDriver, "id", "name", "email", "created_at")

	// 准备数据
	data := []map[string]interface{}{
		{"id": 1, "name": "张三", "email": "zhangsan@example.com", "created_at": time.Now()},
		{"id": 2, "name": "李四", "email": "lisi@example.com", "created_at": time.Now()},
		{"id": 3, "name": "王五", "email": "wangwu@example.com", "created_at": time.Now()},
	}

	// 执行批量操作
	if err := client.ExecuteWithSchema(ctx, schema, data); err != nil {
		return fmt.Errorf("执行批量操作失败: %w", err)
	}

	fmt.Printf("成功插入 %d 条用户记录\n", len(data))
	return nil
}

// largeBatchExample 大批量数据处理示例
func largeBatchExample(ctx context.Context, client *batchsql.Client) error {
	// 创建PostgreSQL Schema
	pgDriver := drivers.NewPostgreSQLDriver()
	schema := client.CreateSchema("orders", batchsql.ConflictIgnore, pgDriver, "id", "user_id", "amount", "status")

	// 生成大量测试数据
	const totalRecords = 100000
	data := make([]map[string]interface{}, totalRecords)

	for i := 0; i < totalRecords; i++ {
		data[i] = map[string]interface{}{
			"id":      i + 1,
			"user_id": (i % 1000) + 1,
			"amount":  float64((i%10000)+1) * 0.01,
			"status":  "pending",
		}
	}

	fmt.Printf("开始处理 %d 条订单记录...\n", totalRecords)
	startTime := time.Now()

	// 使用大批量处理，自动分批
	if err := client.ExecuteLargeBatch(ctx, schema, data, 5000); err != nil {
		return fmt.Errorf("大批量处理失败: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("成功处理 %d 条记录，耗时: %v\n", totalRecords, duration)
	fmt.Printf("平均处理速度: %.2f 记录/秒\n", float64(totalRecords)/duration.Seconds())

	return nil
}

// streamBatchExample 流式批处理示例
func streamBatchExample(ctx context.Context, client *batchsql.Client) error {
	// 创建Redis Schema（用于缓存）
	redisDriver := drivers.NewRedisDriver()
	schema := client.CreateSchema("user_cache", batchsql.ConflictReplace, redisDriver, "key", "value", "ttl")

	// 创建数据流
	dataStream := make(chan map[string]interface{}, 100)

	// 启动数据生产者
	go func() {
		defer close(dataStream)

		for i := 0; i < 50000; i++ {
			select {
			case dataStream <- map[string]interface{}{
				"key":   fmt.Sprintf("user:%d", i),
				"value": fmt.Sprintf(`{"id":%d,"name":"用户%d","last_active":"%s"}`, i, i, time.Now().Format(time.RFC3339)),
				"ttl":   3600, // 1小时过期
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	fmt.Println("开始流式批处理...")
	startTime := time.Now()

	// 执行流式批处理
	if err := client.ExecuteStreamBatch(ctx, schema, dataStream, 1000); err != nil {
		return fmt.Errorf("流式批处理失败: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("流式批处理完成，耗时: %v\n", duration)

	return nil
}

// multiDatabaseExample 多数据库批处理示例
func multiDatabaseExample(ctx context.Context, client *batchsql.Client) error {
	// 同时操作多个数据库

	// MySQL: 用户数据
	mysqlDriver := drivers.NewMySQLDriver()
	userSchema := client.CreateSchema("users", batchsql.ConflictUpdate, mysqlDriver, "id", "name", "email")

	// MongoDB: 用户行为日志
	mongoDriver := drivers.NewMongoDBDriver()
	logSchema := client.CreateSchema("user_logs", batchsql.ConflictIgnore, mongoDriver, "user_id", "action", "timestamp", "metadata")

	// Redis: 用户会话
	redisDriver := drivers.NewRedisDriver()
	sessionSchema := client.CreateSchema("user_sessions", batchsql.ConflictReplace, redisDriver, "session_id", "user_id", "expires_at")

	// 准备用户数据
	userData := []map[string]interface{}{
		{"id": 1, "name": "张三", "email": "zhangsan@example.com"},
		{"id": 2, "name": "李四", "email": "lisi@example.com"},
	}

	// 准备日志数据
	logData := []map[string]interface{}{
		{"user_id": 1, "action": "login", "timestamp": time.Now(), "metadata": map[string]interface{}{"ip": "192.168.1.1"}},
		{"user_id": 2, "action": "login", "timestamp": time.Now(), "metadata": map[string]interface{}{"ip": "192.168.1.2"}},
	}

	// 准备会话数据
	sessionData := []map[string]interface{}{
		{"session_id": "sess_001", "user_id": 1, "expires_at": time.Now().Add(24 * time.Hour)},
		{"session_id": "sess_002", "user_id": 2, "expires_at": time.Now().Add(24 * time.Hour)},
	}

	// 并发执行多个数据库操作
	errChan := make(chan error, 3)

	// MySQL操作
	go func() {
		errChan <- client.ExecuteWithSchema(ctx, userSchema, userData)
	}()

	// MongoDB操作
	go func() {
		errChan <- client.ExecuteWithSchema(ctx, logSchema, logData)
	}()

	// Redis操作
	go func() {
		errChan <- client.ExecuteWithSchema(ctx, sessionSchema, sessionData)
	}()

	// 等待所有操作完成
	for i := 0; i < 3; i++ {
		if err := <-errChan; err != nil {
			return fmt.Errorf("多数据库操作失败: %w", err)
		}
	}

	fmt.Println("多数据库批处理完成")
	return nil
}
