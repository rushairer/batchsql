// Package main provides simple BatchSQL usage examples
package main

import (
	"context"
	"log"
	"time"

	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers"
)

func main() {
	log.Println("=== BatchSQL 简化演示 ===")

	// 创建客户端
	client := batchsql.NewClient()
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("关闭客户端时出错: %v", err)
		}
	}()

	ctx := context.Background()

	// 演示核心功能
	demonstrateCore(ctx, client)

	// 演示多数据库支持
	demonstrateMultiDatabase(ctx, client)

	// 演示监控功能
	demonstrateMonitoring(client)

	log.Println("\n=== 演示完成 ===")
	log.Println("✅ 统一架构基于接口设计")
	log.Println("✅ 支持多种数据库类型")
	log.Println("✅ 内置监控和健康检查")
	log.Println("✅ 高度可扩展的设计")
}

func demonstrateCore(ctx context.Context, client *batchsql.Client) {
	log.Println("\n--- 核心功能演示 ---")

	// 创建驱动
	mysqlDriver := drivers.NewMySQLDriver()
	log.Printf("MySQL驱动: %s", mysqlDriver.GetName())
	log.Printf("  支持的冲突策略: %v", mysqlDriver.SupportedConflictStrategies())

	// 创建Schema
	userSchema := client.CreateSchema(
		"users",
		batchsql.ConflictUpdate,
		mysqlDriver,
		"id", "name", "email", "created_at",
	)

	// 验证Schema
	if err := userSchema.Validate(); err != nil {
		log.Printf("Schema验证失败: %v", err)
	} else {
		log.Printf("✅ Schema验证通过: %s", userSchema.GetIdentifier())
	}

	// 准备数据
	userData := []map[string]interface{}{
		{
			"id":         1,
			"name":       "Alice",
			"email":      "alice@example.com",
			"created_at": time.Now(),
		},
		{
			"id":         2,
			"name":       "Bob",
			"email":      "bob@example.com",
			"created_at": time.Now(),
		},
	}

	// 执行批量操作
	if err := client.ExecuteWithSchema(ctx, userSchema, userData); err != nil {
		log.Printf("执行失败: %v", err)
	} else {
		log.Printf("✅ 批量操作执行成功，处理了 %d 条记录", len(userData))
	}
}

func demonstrateMultiDatabase(ctx context.Context, client *batchsql.Client) {
	log.Println("\n--- 多数据库支持演示 ---")

	databases := []struct {
		name       string
		driver     batchsql.DatabaseDriver
		identifier string
		strategy   batchsql.ConflictStrategy
		columns    []string
	}{
		{
			name:       "MySQL",
			driver:     drivers.NewMySQLDriver(),
			identifier: "users",
			strategy:   batchsql.ConflictUpdate,
			columns:    []string{"id", "name", "email"},
		},
		{
			name:       "PostgreSQL",
			driver:     drivers.NewPostgreSQLDriver(),
			identifier: "products",
			strategy:   batchsql.ConflictIgnore,
			columns:    []string{"id", "name", "price"},
		},
		{
			name:       "Redis",
			driver:     drivers.NewRedisDriver(),
			identifier: "sessions",
			strategy:   batchsql.ConflictReplace,
			columns:    []string{"user_id", "token"},
		},
		{
			name:       "MongoDB",
			driver:     drivers.NewMongoDBDriver(),
			identifier: "logs",
			strategy:   batchsql.ConflictUpdate,
			columns:    []string{"_id", "timestamp", "message"},
		},
	}

	for _, db := range databases {
		log.Printf("\n%s 数据库:", db.name)

		// 创建Schema
		schema := client.CreateSchema(db.identifier, db.strategy, db.driver, db.columns...)

		// 验证Schema
		if err := schema.Validate(); err != nil {
			log.Printf("  ❌ Schema验证失败: %v", err)
			continue
		}

		log.Printf("  ✅ Schema: %s", schema.GetIdentifier())
		log.Printf("  ✅ 冲突策略: %v", schema.GetConflictStrategy())
		log.Printf("  ✅ 列: %v", schema.GetColumns())

		// 生成示例命令
		request := batchsql.NewRequestFromInterface(schema)
		for i, col := range db.columns {
			request.Set(col, "value_"+string(rune('A'+i)))
		}

		command, err := db.driver.GenerateBatchCommand(schema, []*batchsql.Request{request})
		if err != nil {
			log.Printf("  ❌ 命令生成失败: %v", err)
			continue
		}

		log.Printf("  ✅ 命令类型: %s", command.GetCommandType())
		log.Printf("  ✅ 参数数量: %d", len(command.GetParameters()))

		// 执行操作
		data := []map[string]interface{}{
			{
				db.columns[0]: "test_value",
			},
		}

		if err := client.ExecuteWithSchema(ctx, schema, data); err != nil {
			log.Printf("  ❌ 执行失败: %v", err)
		} else {
			log.Printf("  ✅ 执行成功")
		}
	}
}

func demonstrateMonitoring(client *batchsql.Client) {
	log.Println("\n--- 监控功能演示 ---")

	// 创建一个简单的监控报告器
	reporter := &SimpleMetricsReporter{}

	// 设置监控报告器
	clientWithMetrics := client.WithMetricsReporter(reporter)

	// 执行一些操作来触发监控
	mysqlDriver := drivers.NewMySQLDriver()
	schema := clientWithMetrics.CreateSchema("test_table", batchsql.ConflictUpdate, mysqlDriver, "id", "name")

	testData := []map[string]interface{}{
		{"id": 1, "name": "Test User"},
	}

	ctx := context.Background()
	err := clientWithMetrics.ExecuteWithSchema(ctx, schema, testData)
	if err != nil {
		log.Printf("执行失败: %v", err)
	}

	// 显示收集到的监控数据
	log.Println("监控数据:")
	for _, metric := range reporter.GetMetrics() {
		log.Printf("  驱动: %s, 表: %s, 批量大小: %d, 耗时: %v",
			metric.Driver, metric.Table, metric.BatchSize, metric.Duration)
	}
}

// SimpleMetricsReporter 简单的监控报告器示例
type SimpleMetricsReporter struct {
	metrics []batchsql.BatchMetrics
}

func (r *SimpleMetricsReporter) ReportBatchExecution(ctx context.Context, metrics batchsql.BatchMetrics) {
	r.metrics = append(r.metrics, metrics)
}

func (r *SimpleMetricsReporter) GetMetrics() []batchsql.BatchMetrics {
	return r.metrics
}
