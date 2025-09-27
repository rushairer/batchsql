package batchsql

import (
	"context"
	"testing"
	"time"

	"github.com/rushairer/batchsql/drivers"
)

// TestPhase3BasicFunctionality 测试第三阶段基本功能
func TestPhase3BasicFunctionality(t *testing.T) {
	// 创建客户端配置
	config := DefaultClientConfig()
	config.Connections["mysql"] = &ConnectionConfig{
		DriverName:    "mysql",
		ConnectionURL: "test://localhost",
	}

	// 创建客户端
	client, err := NewSimpleBatchSQLClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// 测试MySQL操作
	t.Run("MySQL Operations", func(t *testing.T) {
		testMySQLOperations(t, client)
	})

	// 测试Redis操作
	t.Run("Redis Operations", func(t *testing.T) {
		testRedisOperations(t, client)
	})

	// 测试MongoDB操作
	t.Run("MongoDB Operations", func(t *testing.T) {
		testMongoDBOperations(t, client)
	})

	// 测试指标收集
	t.Run("Metrics Collection", func(t *testing.T) {
		testMetricsCollection(t, client)
	})

	// 测试健康检查
	t.Run("Health Check", func(t *testing.T) {
		testHealthCheck(t, client)
	})
}

func testMySQLOperations(t *testing.T, client *SimpleBatchSQLClient) {
	ctx := context.Background()

	// 创建MySQL驱动和Schema
	mysqlDriver := drivers.NewMySQLDriver()
	userSchema := client.CreateSchema(
		"users",
		ConflictUpdate,
		mysqlDriver,
		"id", "name", "email",
	)

	// 验证Schema
	if err := userSchema.Validate(); err != nil {
		t.Errorf("Schema validation failed: %v", err)
	}

	// 准备测试数据
	userData := []map[string]interface{}{
		{
			"id":    1,
			"name":  "Alice",
			"email": "alice@example.com",
		},
		{
			"id":    2,
			"name":  "Bob",
			"email": "bob@example.com",
		},
	}

	// 执行批量操作
	err := client.ExecuteWithSchema(ctx, userSchema, userData)
	if err != nil {
		t.Errorf("MySQL operation failed: %v", err)
	}

	// 验证支持的冲突策略
	strategies := mysqlDriver.SupportedConflictStrategies()
	if len(strategies) == 0 {
		t.Error("MySQL driver should support conflict strategies")
	}

	expectedStrategies := []ConflictStrategy{ConflictIgnore, ConflictReplace, ConflictUpdate}
	for _, expected := range expectedStrategies {
		found := false
		for _, actual := range strategies {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("MySQL driver should support strategy: %v", expected)
		}
	}
}

func testRedisOperations(t *testing.T, client *SimpleBatchSQLClient) {
	ctx := context.Background()

	// 测试基础Redis操作
	redisDriver := drivers.NewRedisDriver()
	sessionSchema := client.CreateSchema(
		"session",
		ConflictReplace,
		redisDriver,
		"user_id", "token",
	)

	sessionData := []map[string]interface{}{
		{
			"user_id": "user_1",
			"token":   "token_123",
		},
	}

	err := client.ExecuteWithSchema(ctx, sessionSchema, sessionData)
	if err != nil {
		t.Errorf("Redis operation failed: %v", err)
	}

	// 测试Redis Set操作
	redisSetDriver := drivers.NewRedisSetDriver()
	setSchema := client.CreateSchema(
		"online_users",
		ConflictIgnore,
		redisSetDriver,
		"room_id", "user_id",
	)

	setData := []map[string]interface{}{
		{
			"room_id": "room_1",
			"user_id": "user_1",
		},
	}

	err = client.ExecuteWithSchema(ctx, setSchema, setData)
	if err != nil {
		t.Errorf("Redis Set operation failed: %v", err)
	}
}

func testMongoDBOperations(t *testing.T, client *SimpleBatchSQLClient) {
	ctx := context.Background()

	// 测试标准MongoDB操作
	mongoDriver := drivers.NewMongoDBDriver()
	productSchema := client.CreateSchema(
		"products",
		ConflictUpdate,
		mongoDriver,
		"_id", "name", "price",
	)

	productData := []map[string]interface{}{
		{
			"_id":   "product_1",
			"name":  "Laptop",
			"price": 999.99,
		},
	}

	err := client.ExecuteWithSchema(ctx, productSchema, productData)
	if err != nil {
		t.Errorf("MongoDB operation failed: %v", err)
	}

	// 测试时间序列集合
	timeSeriesDriver := drivers.NewMongoTimeSeriesDriver("timestamp", "device_id", "seconds")
	metricsSchema := client.CreateSchema(
		"device_metrics",
		ConflictIgnore,
		timeSeriesDriver,
		"device_id", "timestamp", "temperature",
	)

	metricsData := []map[string]interface{}{
		{
			"device_id":   "sensor_001",
			"timestamp":   time.Now(),
			"temperature": 23.5,
		},
	}

	err = client.ExecuteWithSchema(ctx, metricsSchema, metricsData)
	if err != nil {
		t.Errorf("MongoDB TimeSeries operation failed: %v", err)
	}
}

func testMetricsCollection(t *testing.T, client *SimpleBatchSQLClient) {
	// 获取指标
	metrics := client.GetMetrics()

	// 验证基本指标字段
	expectedFields := []string{
		"start_time", "uptime", "total_executions",
		"successful_execs", "failed_execs", "success_rate",
		"total_requests", "total_duration", "average_duration",
	}

	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Metrics should contain field: %s", field)
		}
	}

	// 验证驱动指标
	if driverMetrics, ok := metrics["driver_metrics"].(map[string]*DriverMetrics); ok {
		// 应该有一些驱动指标（从之前的测试中）
		if len(driverMetrics) == 0 {
			t.Log("No driver metrics found (this is expected for mock operations)")
		}
	} else {
		t.Error("driver_metrics should be of type map[string]*DriverMetrics")
	}
}

func testHealthCheck(t *testing.T, client *SimpleBatchSQLClient) {
	ctx := context.Background()

	health := client.HealthCheck(ctx)

	// 验证健康检查基本字段
	if status, ok := health["status"]; !ok {
		t.Error("Health check should contain status field")
	} else if status != "healthy" && status != "degraded" {
		t.Errorf("Health status should be 'healthy' or 'degraded', got: %v", status)
	}

	if _, ok := health["timestamp"]; !ok {
		t.Error("Health check should contain timestamp field")
	}

	// 验证连接健康状态
	if connections, ok := health["connections"].(map[string]interface{}); ok {
		if len(connections) == 0 {
			t.Error("Health check should contain connection information")
		}
	} else {
		t.Error("connections should be of type map[string]interface{}")
	}
}

// TestSchemaValidation 测试Schema验证
func TestSchemaValidation(t *testing.T) {
	mysqlDriver := drivers.NewMySQLDriver()

	// 测试有效Schema
	validSchema := NewUniversalSchema(
		"users",
		ConflictUpdate,
		mysqlDriver,
		"id", "name", "email",
	)

	if err := validSchema.Validate(); err != nil {
		t.Errorf("Valid schema should pass validation: %v", err)
	}

	// 测试无效Schema（空表名）
	invalidSchema := NewUniversalSchema(
		"",
		ConflictUpdate,
		mysqlDriver,
		"id", "name",
	)

	if err := invalidSchema.Validate(); err == nil {
		t.Error("Invalid schema (empty identifier) should fail validation")
	}

	// 测试无效Schema（无列）
	noColumnsSchema := NewUniversalSchema(
		"users",
		ConflictUpdate,
		mysqlDriver,
	)

	if err := noColumnsSchema.Validate(); err == nil {
		t.Error("Invalid schema (no columns) should fail validation")
	}
}

// TestDriverCompatibility 测试驱动兼容性
func TestDriverCompatibility(t *testing.T) {
	drivers := []struct {
		name   string
		driver DatabaseDriver
	}{
		{"MySQL", drivers.NewMySQLDriver()},
		{"PostgreSQL", drivers.NewPostgreSQLDriver()},
		{"SQLite", drivers.NewSQLiteDriver()},
		{"Redis", drivers.NewRedisDriver()},
		{"RedisHash", drivers.NewRedisHashDriver()},
		{"RedisSet", drivers.NewRedisSetDriver()},
		{"MongoDB", drivers.NewMongoDBDriver()},
		{"MongoTimeSeries", drivers.NewMongoTimeSeriesDriver("timestamp", "device_id", "seconds")},
	}

	for _, d := range drivers {
		t.Run(d.name, func(t *testing.T) {
			// 验证驱动名称
			if d.driver.GetName() == "" {
				t.Errorf("%s driver should have a name", d.name)
			}

			// 验证支持的冲突策略
			strategies := d.driver.SupportedConflictStrategies()
			if len(strategies) == 0 {
				t.Errorf("%s driver should support at least one conflict strategy", d.name)
			}

			// 创建测试Schema
			schema := NewUniversalSchema(
				"test_table",
				strategies[0], // 使用第一个支持的策略
				d.driver,
				"id", "name",
			)

			// 验证Schema
			if err := d.driver.ValidateSchema(schema); err != nil {
				t.Errorf("%s driver schema validation failed: %v", d.name, err)
			}

			// 创建测试请求
			request := NewRequestFromInterface(schema)
			request.SetInt64("id", 1)
			request.SetString("name", "test")

			// 生成批量命令
			command, err := d.driver.GenerateBatchCommand(schema, []*Request{request})
			if err != nil {
				t.Errorf("%s driver failed to generate batch command: %v", d.name, err)
			}

			// 验证命令基本属性
			if command.GetCommandType() == "" {
				t.Errorf("%s driver should return command with type", d.name)
			}

			if command.GetCommand() == nil {
				t.Errorf("%s driver should return command with content", d.name)
			}

			if command.GetMetadata() == nil {
				t.Errorf("%s driver should return command with metadata", d.name)
			}
		})
	}
}

// BenchmarkPhase3Performance 性能基准测试
func BenchmarkPhase3Performance(b *testing.B) {
	config := DefaultClientConfig()
	config.Connections["mysql"] = &ConnectionConfig{
		DriverName:    "mysql",
		ConnectionURL: "test://localhost",
	}

	client, err := NewSimpleBatchSQLClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	mysqlDriver := drivers.NewMySQLDriver()
	schema := client.CreateSchema("users", ConflictUpdate, mysqlDriver, "id", "name", "email")

	data := []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@example.com"},
		{"id": 2, "name": "Bob", "email": "bob@example.com"},
		{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.ExecuteWithSchema(ctx, schema, data)
		if err != nil {
			b.Errorf("Execution failed: %v", err)
		}
	}
}
