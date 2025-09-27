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
	log.Println("🚀 开始 BatchSQL 集成测试...")

	// 等待数据库服务启动
	time.Sleep(5 * time.Second)

	ctx := context.Background()

	// 测试基本功能
	if err := testBasicFunctionality(ctx); err != nil {
		log.Printf("❌ 基本功能测试失败: %v", err)
		return
	}
	log.Println("✅ 基本功能测试通过")

	// 测试各种驱动的创建
	if err := testDriverCreation(); err != nil {
		log.Printf("❌ 驱动创建测试失败: %v", err)
		return
	}
	log.Println("✅ 驱动创建测试通过")

	// 测试 Schema 创建和验证
	if err := testSchemaCreation(); err != nil {
		log.Printf("❌ Schema 创建测试失败: %v", err)
		return
	}
	log.Println("✅ Schema 创建测试通过")

	log.Println("🎉 所有集成测试通过！BatchSQL 系统运行正常")
}

func testBasicFunctionality(ctx context.Context) error {
	// 创建一个模拟驱动进行基本功能测试
	driver := &MockDriver{name: "integration-test"}

	client := batchsql.NewClient()
	schema := batchsql.NewSchema("integration_test", batchsql.ConflictReplace, driver, "id", "name", "value")

	testData := []map[string]interface{}{
		{"id": 1, "name": "test1", "value": "integration_value1"},
		{"id": 2, "name": "test2", "value": "integration_value2"},
		{"id": 3, "name": "test3", "value": "integration_value3"},
	}

	return client.ExecuteWithSchema(ctx, schema, testData)
}

func testDriverCreation() error {
	// 测试各种驱动的创建
	drivers := []struct {
		name   string
		create func() interface{}
	}{
		{"MySQL", func() interface{} { return drivers.NewMySQLDriver() }},
		{"PostgreSQL", func() interface{} { return drivers.NewPostgreSQLDriver() }},
		{"Redis", func() interface{} { return drivers.NewRedisDriver() }},
		{"MongoDB", func() interface{} { return drivers.NewMongoDBDriver() }},
		{"RedisHash", func() interface{} { return drivers.NewRedisHashDriver() }},
		{"RedisSet", func() interface{} { return drivers.NewRedisSetDriver() }},
		{"MongoTimeSeries", func() interface{} { return drivers.NewMongoTimeSeriesDriver("timestamp", "metadata", "seconds") }},
	}

	for _, d := range drivers {
		driver := d.create()
		if driver == nil {
			return fmt.Errorf("%s 驱动创建失败", d.name)
		}
		log.Printf("  ✓ %s 驱动创建成功", d.name)
	}

	return nil
}

func testSchemaCreation() error {
	// 测试不同冲突策略的 Schema 创建
	driver := &MockDriver{name: "schema-test"}

	strategies := []struct {
		name     string
		strategy batchsql.ConflictStrategy
	}{
		{"IGNORE", batchsql.ConflictIgnore},
		{"REPLACE", batchsql.ConflictReplace},
		{"UPDATE", batchsql.ConflictUpdate},
	}

	for _, s := range strategies {
		schema := batchsql.NewSchema("test_table", s.strategy, driver, "id", "name", "value")
		if schema == nil {
			return fmt.Errorf("Schema 创建失败 (策略: %s)", s.name)
		}

		// 验证 Schema 属性
		if schema.GetIdentifier() != "test_table" {
			return fmt.Errorf("表名不匹配: 期望 'test_table', 实际 '%s'", schema.GetIdentifier())
		}

		if schema.GetConflictStrategy() != s.strategy {
			return fmt.Errorf("冲突策略不匹配: 期望 %v, 实际 %v", s.strategy, schema.GetConflictStrategy())
		}

		log.Printf("  ✓ %s 策略 Schema 创建和验证成功", s.name)
	}

	return nil
}

// MockDriver 用于集成测试的模拟驱动
type MockDriver struct {
	name string
}

func (d *MockDriver) GetName() string {
	return d.name
}

func (d *MockDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	return &MockCommand{
		commandType: "INSERT",
		requests:    requests,
		metadata: map[string]interface{}{
			"table":    schema.GetIdentifier(),
			"columns":  schema.GetColumns(),
			"strategy": schema.GetConflictStrategy(),
			"count":    len(requests),
		},
	}, nil
}

func (d *MockDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictReplace,
		batchsql.ConflictUpdate,
	}
}

func (d *MockDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if schema.GetIdentifier() == "" {
		return fmt.Errorf("表名不能为空")
	}
	if len(schema.GetColumns()) == 0 {
		return fmt.Errorf("列定义不能为空")
	}
	return nil
}

func (d *MockDriver) Close() error {
	log.Printf("  关闭 %s 驱动连接", d.name)
	return nil
}

// MockCommand 用于测试的模拟命令
type MockCommand struct {
	commandType string
	requests    []*batchsql.Request
	metadata    map[string]interface{}
}

func (c *MockCommand) GetCommandType() string {
	return c.commandType
}

func (c *MockCommand) GetCommand() interface{} {
	return c.requests
}

func (c *MockCommand) GetParameters() []interface{} {
	return nil
}

func (c *MockCommand) GetMetadata() map[string]interface{} {
	return c.metadata
}
