package main

import (
	"context"
	"log"
	"time"

	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers"
)

// 演示新架构的使用方式
func main() {
	ctx := context.Background()

	// 示例1：使用 MySQL 驱动
	demonstrateMySQLUsage(ctx)

	// 示例2：使用 Redis 驱动
	demonstrateRedisUsage(ctx)

	// 示例3：使用 MongoDB 驱动
	demonstrateMongoDBUsage(ctx)

	// 示例4：混合使用多种数据库
	demonstrateMixedUsage(ctx)
}

func demonstrateMySQLUsage(ctx context.Context) {
	log.Println("=== MySQL 驱动示例 ===")

	// 创建 MySQL 驱动
	mysqlDriver := drivers.NewMySQLDriver()

	// 创建 Schema
	userSchema := batchsql.NewUniversalSchema(
		"users",                             // 表名
		batchsql.ConflictUpdate,             // 冲突策略
		mysqlDriver,                         // 驱动
		"id", "name", "email", "created_at", // 列名
	)

	// 验证 Schema
	if err := userSchema.Validate(); err != nil {
		log.Printf("Schema validation failed: %v", err)
		return
	}

	// 创建请求
	requests := make([]*batchsql.Request, 0, 10)
	for i := 0; i < 10; i++ {
		request := batchsql.NewRequestFromInterface(userSchema).
			SetInt64("id", int64(i)).
			SetString("name", "User"+string(rune('A'+i%26))).
			SetString("email", "user"+string(rune('0'+i%10))+"@example.com").
			SetTime("created_at", time.Now())

		requests = append(requests, request)
	}

	// 生成批量命令
	command, err := mysqlDriver.GenerateBatchCommand(userSchema, requests)
	if err != nil {
		log.Printf("Failed to generate command: %v", err)
		return
	}

	// 打印生成的 SQL
	log.Printf("Generated SQL: %s", command.GetCommand())
	log.Printf("Parameters count: %d", len(command.GetParameters()))
	log.Printf("Metadata: %+v", command.GetMetadata())
}

func demonstrateRedisUsage(ctx context.Context) {
	log.Println("\n=== Redis 驱动示例 ===")

	// 创建 Redis 驱动
	redisDriver := drivers.NewRedisDriver()

	// 创建 Schema
	sessionSchema := batchsql.NewUniversalSchema(
		"session",                        // 键前缀
		batchsql.ConflictReplace,         // 冲突策略
		redisDriver,                      // 驱动
		"user_id", "token", "expires_at", // 列名
	)

	// 创建请求
	requests := make([]*batchsql.Request, 0, 5)
	for i := 0; i < 5; i++ {
		request := batchsql.NewRequestFromInterface(sessionSchema).
			SetInt64("user_id", int64(i)).
			SetString("token", "token_"+string(rune('A'+i%26))).
			SetTime("expires_at", time.Now().Add(24*time.Hour))

		requests = append(requests, request)
	}

	// 生成批量命令
	command, err := redisDriver.GenerateBatchCommand(sessionSchema, requests)
	if err != nil {
		log.Printf("Failed to generate command: %v", err)
		return
	}

	// 打印生成的 Redis 命令
	commands := command.GetCommand().([][]interface{})
	log.Printf("Generated Redis commands:")
	for i, cmd := range commands {
		log.Printf("  Command %d: %v", i+1, cmd)
	}
	log.Printf("Metadata: %+v", command.GetMetadata())
}

func demonstrateMongoDBUsage(ctx context.Context) {
	log.Println("\n=== MongoDB 驱动示例 ===")

	// 创建 MongoDB 驱动
	mongoDriver := drivers.NewMongoDBDriver()

	// 创建 Schema
	productSchema := batchsql.NewUniversalSchema(
		"products",                         // 集合名
		batchsql.ConflictUpdate,            // 冲突策略
		mongoDriver,                        // 驱动
		"_id", "name", "price", "category", // 列名
	)

	// 创建请求
	requests := make([]*batchsql.Request, 0, 3)
	for i := 0; i < 3; i++ {
		request := batchsql.NewRequestFromInterface(productSchema).
			SetString("_id", "product_"+string(rune('A'+i%26))).
			SetString("name", "Product "+string(rune('A'+i%26))).
			SetFloat64("price", float64(i+1)*99.99).
			SetString("category", "electronics")

		requests = append(requests, request)
	}

	// 生成批量命令
	command, err := mongoDriver.GenerateBatchCommand(productSchema, requests)
	if err != nil {
		log.Printf("Failed to generate command: %v", err)
		return
	}

	// 打印生成的 MongoDB 操作
	operations := command.GetCommand().([]interface{})
	log.Printf("Generated MongoDB operations:")
	for i, op := range operations {
		log.Printf("  Operation %d: %+v", i+1, op)
	}
	log.Printf("Metadata: %+v", command.GetMetadata())
}

func demonstrateMixedUsage(ctx context.Context) {
	log.Println("\n=== 混合数据库使用示例 ===")

	// 创建不同的驱动和 Schema
	mysqlDriver := drivers.NewMySQLDriver()
	redisSetDriver := drivers.NewRedisSetDriver()
	mongoTimeSeriesDriver := drivers.NewMongoTimeSeriesDriver("timestamp", "device_id", "seconds")

	// MySQL 用户表
	userSchema := batchsql.NewUniversalSchema(
		"users",
		batchsql.ConflictIgnore,
		mysqlDriver,
		"id", "name", "email",
	)

	// Redis 在线用户集合
	onlineUsersSchema := batchsql.NewUniversalSchema(
		"online_users",
		batchsql.ConflictIgnore,
		redisSetDriver,
		"room_id", "user_id",
	)

	// MongoDB 时间序列数据
	metricsSchema := batchsql.NewUniversalSchema(
		"device_metrics",
		batchsql.ConflictIgnore,
		mongoTimeSeriesDriver,
		"device_id", "timestamp", "temperature", "humidity",
	)

	// 展示不同 Schema 的特性
	schemas := []batchsql.SchemaInterface{userSchema, onlineUsersSchema, metricsSchema}

	for i, schema := range schemas {
		log.Printf("Schema %d:", i+1)
		log.Printf("  Driver: %s", schema.GetDatabaseDriver().GetName())
		log.Printf("  Identifier: %s", schema.GetIdentifier())
		log.Printf("  Conflict Strategy: %v", schema.GetConflictStrategy())
		log.Printf("  Columns: %v", schema.GetColumns())
		log.Printf("  Supported Strategies: %v", schema.GetDatabaseDriver().SupportedConflictStrategies())

		if err := schema.Validate(); err != nil {
			log.Printf("  Validation Error: %v", err)
		} else {
			log.Printf("  Validation: ✓ Passed")
		}
		log.Println()
	}
}

// 演示 Schema 的高级功能
func demonstrateSchemaFeatures() {
	log.Println("\n=== Schema 高级功能示例 ===")

	mysqlDriver := drivers.NewMySQLDriver()

	// 创建基础 Schema
	baseSchema := batchsql.NewUniversalSchema(
		"base_table",
		batchsql.ConflictIgnore,
		mysqlDriver,
		"id", "name",
	)

	// 使用链式调用扩展 Schema
	extendedSchema := baseSchema.
		WithConflictStrategy(batchsql.ConflictUpdate).
		WithColumns("id", "name", "email", "created_at", "updated_at").
		WithMetadata("version", "1.0").
		WithMetadata("description", "Extended user schema")

	log.Printf("Base Schema: %s", baseSchema.GetIdentifier())
	log.Printf("Extended Schema: %s", extendedSchema.GetIdentifier())

	// 克隆 Schema
	clonedSchema := extendedSchema.Clone().(*batchsql.UniversalSchema)
	clonedSchema.WithMetadata("cloned", true)

	log.Printf("Cloned Schema: %s", clonedSchema.GetIdentifier())

	// 检查列操作
	log.Printf("Has 'email' column: %t", extendedSchema.HasColumn("email"))
	log.Printf("Email column index: %d", extendedSchema.GetColumnIndex("email"))

	// 获取元数据
	if version, exists := extendedSchema.GetMetadata("version"); exists {
		log.Printf("Schema version: %s", version)
	}
}
