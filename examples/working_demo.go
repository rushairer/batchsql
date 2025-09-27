package main

import (
	"context"
	"log"
	"time"

	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers"
)

// 第三阶段可工作的演示
func main() {
	log.Println("=== BatchSQL 第三阶段可工作演示 ===")

	// 1. 创建简化客户端配置
	config := batchsql.DefaultClientConfig()

	// 添加数据库连接配置
	config.Connections["mysql"] = &batchsql.ConnectionConfig{
		DriverName:    "mysql",
		ConnectionURL: "mock://localhost",
	}

	// 2. 创建简化客户端
	client, err := batchsql.NewSimpleBatchSQLClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 3. 演示新架构的核心功能
	demonstrateNewArchitecture(ctx, client)

	// 4. 演示多数据库支持
	demonstrateMultiDatabase(ctx, client)

	// 5. 演示指标和监控
	demonstrateMonitoring(client)

	log.Println("\n=== 第三阶段演示完成 ===")
	log.Println("✅ 新架构基于接口设计，支持多种数据库")
	log.Println("✅ 统一的API和使用方式")
	log.Println("✅ 内置指标收集和健康检查")
	log.Println("✅ 可扩展的驱动架构")
}

func demonstrateNewArchitecture(ctx context.Context, client *batchsql.SimpleBatchSQLClient) {
	log.Println("\n--- 新架构核心功能演示 ---")

	// 创建不同类型的驱动
	mysqlDriver := drivers.NewMySQLDriver()
	redisDriver := drivers.NewRedisDriver()
	mongoDriver := drivers.NewMongoDBDriver()

	log.Printf("MySQL驱动: %s", mysqlDriver.GetName())
	log.Printf("  支持的冲突策略: %v", mysqlDriver.SupportedConflictStrategies())

	log.Printf("Redis驱动: %s", redisDriver.GetName())
	log.Printf("  支持的冲突策略: %v", redisDriver.SupportedConflictStrategies())

	log.Printf("MongoDB驱动: %s", mongoDriver.GetName())
	log.Printf("  支持的冲突策略: %v", mongoDriver.SupportedConflictStrategies())

	// 创建统一的Schema
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

func demonstrateMultiDatabase(ctx context.Context, client *batchsql.SimpleBatchSQLClient) {
	log.Println("\n--- 多数据库支持演示 ---")

	// 演示不同数据库的Schema创建
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
			map[string]interface{}{
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

func demonstrateMonitoring(client *batchsql.SimpleBatchSQLClient) {
	log.Println("\n--- 指标和监控演示 ---")

	// 获取指标
	metrics := client.GetMetrics()

	log.Println("系统指标:")
	if uptime, ok := metrics["uptime"].(time.Duration); ok {
		log.Printf("  运行时间: %v", uptime)
	}

	if totalExecs, ok := metrics["total_executions"].(int64); ok {
		log.Printf("  总执行次数: %d", totalExecs)
	}

	if successRate, ok := metrics["success_rate"].(float64); ok {
		log.Printf("  成功率: %.2f%%", successRate)
	}

	// 健康检查
	ctx := context.Background()
	health := client.HealthCheck(ctx)

	log.Println("\n健康检查:")
	log.Printf("  系统状态: %v", health["status"])
	log.Printf("  检查时间: %v", health["timestamp"])

	if connections, ok := health["connections"].(map[string]interface{}); ok {
		log.Println("  连接状态:")
		for name, status := range connections {
			log.Printf("    %s: %+v", name, status)
		}
	}
}

// 演示扩展性
func demonstrateExtensibility() {
	log.Println("\n--- 扩展性演示 ---")

	log.Println("添加新数据库支持只需要:")
	log.Println("1. 实现 DatabaseDriver 接口")
	log.Println("2. 实现 BatchCommand 接口")
	log.Println("3. 注册到连接管理器")

	log.Println("\n示例代码:")
	log.Println(`
type CustomDriver struct{}

func (d *CustomDriver) GetName() string {
    return "custom"
}

func (d *CustomDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    // 自定义命令生成逻辑
    return &CustomCommand{...}, nil
}

func (d *CustomDriver) SupportedConflictStrategies() []ConflictStrategy {
    return []ConflictStrategy{ConflictIgnore, ConflictUpdate}
}

func (d *CustomDriver) ValidateSchema(schema SchemaInterface) error {
    // 自定义验证逻辑
    return nil
}
`)

	log.Println("然后就可以像使用其他数据库一样使用自定义数据库！")
}
