package batchsql_test

import (
	"context"
	"testing"

	"github.com/rushairer/batchsql"
)

// TestClientPublicAPI 测试客户端公开API
func TestClientPublicAPI(t *testing.T) {
	// 创建客户端
	client := batchsql.NewClient()
	if client == nil {
		t.Fatal("NewClient() should not return nil")
	}

	// 测试链式调用
	reporter := &MockMetricsReporter{}
	clientWithReporter := client.WithMetricsReporter(reporter)
	if clientWithReporter == nil {
		t.Fatal("WithMetricsReporter() should not return nil")
	}
}

// TestSchemaPublicAPI 测试Schema公开API
func TestSchemaPublicAPI(t *testing.T) {
	driver := &MockDriver{name: "test"}

	// 测试Schema创建
	schema := batchsql.NewSchema("users", batchsql.ConflictIgnore, driver, "id", "name")

	// 验证公开方法
	if schema.GetIdentifier() != "users" {
		t.Errorf("Expected identifier 'users', got '%s'", schema.GetIdentifier())
	}

	if schema.GetConflictStrategy() != batchsql.ConflictIgnore {
		t.Errorf("Expected ConflictIgnore strategy")
	}

	columns := schema.GetColumns()
	if len(columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(columns))
	}

	if schema.GetDatabaseDriver() != driver {
		t.Error("Driver should match")
	}
}

// TestEndToEndWorkflow 端到端工作流测试
func TestEndToEndWorkflow(t *testing.T) {
	// 创建模拟驱动
	driver := &MockDriver{name: "mysql"}

	// 创建客户端和监控
	reporter := &MockMetricsReporter{}
	client := batchsql.NewClient().WithMetricsReporter(reporter)

	// 创建Schema
	schema := batchsql.NewSchema("users", batchsql.ConflictReplace, driver, "id", "name", "email")

	// 准备测试数据
	data := []map[string]interface{}{
		{"id": 1, "name": "Alice", "email": "alice@test.com"},
		{"id": 2, "name": "Bob", "email": "bob@test.com"},
	}

	// 执行批量操作
	err := client.ExecuteWithSchema(context.Background(), schema, data)
	if err != nil {
		t.Fatalf("ExecuteWithSchema failed: %v", err)
	}

	// 验证监控数据
	if len(reporter.metrics) == 0 {
		t.Fatal("No metrics reported")
	}

	metrics := reporter.metrics[0]
	if metrics.Driver != "mysql" {
		t.Errorf("Expected driver 'mysql', got '%s'", metrics.Driver)
	}
	if metrics.BatchSize != 2 {
		t.Errorf("Expected batch size 2, got %d", metrics.BatchSize)
	}
}

// TestErrorHandling 错误处理测试
func TestErrorHandling(t *testing.T) {
	client := batchsql.NewClient()

	// 测试nil schema
	err := client.ExecuteWithSchema(context.Background(), nil, []map[string]interface{}{{"id": 1}})
	if err == nil {
		t.Fatal("Expected error for nil schema")
	}

	// 测试空数据
	driver := &MockDriver{name: "test"}
	schema := batchsql.NewSchema("test", batchsql.ConflictIgnore, driver, "id")
	err = client.ExecuteWithSchema(context.Background(), schema, []map[string]interface{}{})
	if err != nil {
		t.Errorf("Should not error on empty data: %v", err)
	}
}

// TestConflictStrategies 冲突策略测试
func TestConflictStrategies(t *testing.T) {
	driver := &MockDriver{name: "test"}

	strategies := []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictReplace,
		batchsql.ConflictUpdate,
	}

	for _, strategy := range strategies {
		schema := batchsql.NewSchema("test", strategy, driver, "id")
		if schema.GetConflictStrategy() != strategy {
			t.Errorf("Strategy not set correctly: expected %v, got %v",
				strategy, schema.GetConflictStrategy())
		}
	}
}

// Mock implementations for black-box testing

type MockDriver struct {
	name       string
	shouldFail bool
	commands   []batchsql.BatchCommand
}

func (m *MockDriver) GetName() string {
	return m.name
}

func (m *MockDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if m.shouldFail {
		return nil, &MockError{message: "mock driver error"}
	}

	command := &MockBatchCommand{
		commandType: "INSERT",
		command:     "INSERT INTO " + schema.GetIdentifier(),
		parameters:  make([]interface{}, len(requests)),
		metadata:    map[string]interface{}{"table": schema.GetIdentifier()},
	}

	m.commands = append(m.commands, command)
	return command, nil
}

func (m *MockDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictReplace,
		batchsql.ConflictUpdate,
	}
}

func (m *MockDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if m.shouldFail {
		return &MockError{message: "schema validation failed"}
	}
	return nil
}

type MockBatchCommand struct {
	commandType string
	command     interface{}
	parameters  []interface{}
	metadata    map[string]interface{}
}

func (m *MockBatchCommand) GetCommandType() string {
	return m.commandType
}

func (m *MockBatchCommand) GetCommand() interface{} {
	return m.command
}

func (m *MockBatchCommand) GetParameters() []interface{} {
	return m.parameters
}

func (m *MockBatchCommand) GetMetadata() map[string]interface{} {
	return m.metadata
}

type MockMetricsReporter struct {
	metrics []batchsql.BatchMetrics
}

func (m *MockMetricsReporter) ReportBatchExecution(ctx context.Context, metrics batchsql.BatchMetrics) {
	m.metrics = append(m.metrics, metrics)
}

type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}
