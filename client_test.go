package batchsql

import (
	"context"
	"errors"
	"testing"
	"time"
)

// 这个文件包含白盒测试，用于测试内部实现细节
// 公开API的测试请参考 client_integration_test.go

// MockDriver 用于测试的模拟驱动
type MockDriver struct {
	name       string
	shouldFail bool
	commands   []BatchCommand
	strategies []ConflictStrategy
}

func NewMockDriver(name string) *MockDriver {
	return &MockDriver{
		name:       name,
		shouldFail: false,
		commands:   make([]BatchCommand, 0),
		strategies: []ConflictStrategy{ConflictIgnore, ConflictReplace, ConflictUpdate},
	}
}

func (m *MockDriver) GetName() string {
	return m.name
}

func (m *MockDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
	if m.shouldFail {
		return nil, errors.New("mock driver error")
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

func (m *MockDriver) SupportedConflictStrategies() []ConflictStrategy {
	return m.strategies
}

func (m *MockDriver) ValidateSchema(schema SchemaInterface) error {
	if m.shouldFail {
		return errors.New("schema validation failed")
	}
	return nil
}

// MockBatchCommand 用于测试的模拟批量命令
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

// MockMetricsReporter 用于测试的模拟监控报告器
type MockMetricsReporter struct {
	reported []BatchMetrics
}

func (m *MockMetricsReporter) ReportBatchExecution(ctx context.Context, metrics BatchMetrics) {
	m.reported = append(m.reported, metrics)
}

// TestNewClient 测试客户端创建
func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient() returned nil")
	}
}

// TestWithMetricsReporter 测试监控报告器设置
func TestWithMetricsReporter(t *testing.T) {
	client := NewClient()
	reporter := &MockMetricsReporter{}

	clientWithReporter := client.WithMetricsReporter(reporter)
	if clientWithReporter == nil {
		t.Fatal("WithMetricsReporter() returned nil")
	}

	if clientWithReporter.reporter != reporter {
		t.Fatal("MetricsReporter was not set correctly")
	}
}

// TestExecuteWithSchema 测试使用Schema执行
func TestExecuteWithSchema(t *testing.T) {
	// 创建模拟驱动
	mockDriver := NewMockDriver("mysql")

	// 创建客户端
	client := NewClient()

	// 创建测试Schema
	schema := NewSchema("test_table", ConflictIgnore, mockDriver, "id", "name")

	// 测试数据
	data := []map[string]interface{}{
		{"id": 1, "name": "test1"},
		{"id": 2, "name": "test2"},
	}

	// 执行测试
	err := client.ExecuteWithSchema(context.Background(), schema, data)
	if err != nil {
		t.Fatalf("ExecuteWithSchema() failed: %v", err)
	}

	// 验证命令被生成
	if len(mockDriver.commands) == 0 {
		t.Fatal("No commands were generated")
	}
}

// TestExecuteWithSchemaError 测试错误处理
func TestExecuteWithSchemaError(t *testing.T) {
	// 创建会失败的模拟驱动
	mockDriver := NewMockDriver("mysql")
	mockDriver.shouldFail = true

	// 创建客户端
	client := NewClient()

	// 创建测试Schema
	schema := NewSchema("test_table", ConflictIgnore, mockDriver, "id")

	// 测试数据
	data := []map[string]interface{}{
		{"id": 1},
	}

	// 执行测试，应该返回错误
	err := client.ExecuteWithSchema(context.Background(), schema, data)
	if err == nil {
		t.Fatal("Expected error but got nil")
	}
}

// TestMetricsReporting 测试监控报告
func TestMetricsReporting(t *testing.T) {
	// 创建模拟驱动和监控报告器
	mockDriver := NewMockDriver("mysql")
	mockReporter := &MockMetricsReporter{}

	// 创建客户端并设置监控
	client := NewClient().WithMetricsReporter(mockReporter)

	// 创建测试Schema
	schema := NewSchema("test_table", ConflictIgnore, mockDriver, "id")

	// 测试数据
	data := []map[string]interface{}{
		{"id": 1},
	}

	// 执行测试
	err := client.ExecuteWithSchema(context.Background(), schema, data)
	if err != nil {
		t.Fatalf("ExecuteWithSchema() failed: %v", err)
	}

	// 验证监控数据被报告
	if len(mockReporter.reported) == 0 {
		t.Fatal("No metrics were reported")
	}

	metrics := mockReporter.reported[0]
	if metrics.Driver != "mysql" {
		t.Errorf("Expected driver 'mysql', got '%s'", metrics.Driver)
	}
	if metrics.Table != "test_table" {
		t.Errorf("Expected table 'test_table', got '%s'", metrics.Table)
	}
	if metrics.BatchSize != 1 {
		t.Errorf("Expected batch size 1, got %d", metrics.BatchSize)
	}
	if metrics.Error != nil {
		t.Errorf("Expected no error, got %v", metrics.Error)
	}
}

// TestCreateSchema 测试Schema创建便捷方法
func TestCreateSchema(t *testing.T) {
	client := NewClient()
	mockDriver := NewMockDriver("mysql")

	schema := client.CreateSchema("users", ConflictReplace, mockDriver, "id", "name", "email")

	if schema.GetIdentifier() != "users" {
		t.Errorf("Expected identifier 'users', got '%s'", schema.GetIdentifier())
	}

	if schema.GetConflictStrategy() != ConflictReplace {
		t.Errorf("Expected conflict strategy ConflictReplace, got %v", schema.GetConflictStrategy())
	}

	columns := schema.GetColumns()
	expectedColumns := []string{"id", "name", "email"}
	if len(columns) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(columns))
	}

	for i, col := range expectedColumns {
		if columns[i] != col {
			t.Errorf("Expected column '%s' at index %d, got '%s'", col, i, columns[i])
		}
	}
}

// TestClientClose 测试客户端关闭
func TestClientClose(t *testing.T) {
	client := NewClient()
	err := client.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

// TestExecuteWithNilSchema 测试空Schema处理
func TestExecuteWithNilSchema(t *testing.T) {
	client := NewClient()

	err := client.ExecuteWithSchema(context.Background(), nil, []map[string]interface{}{{"id": 1}})
	if err == nil {
		t.Fatal("Expected error for nil schema")
	}

	if err.Error() != "schema cannot be nil" {
		t.Errorf("Expected 'schema cannot be nil', got '%s'", err.Error())
	}
}

// TestExecuteWithEmptyData 测试空数据处理
func TestExecuteWithEmptyData(t *testing.T) {
	client := NewClient()
	mockDriver := NewMockDriver("mysql")
	schema := NewSchema("test", ConflictIgnore, mockDriver, "id")

	err := client.ExecuteWithSchema(context.Background(), schema, []map[string]interface{}{})
	if err != nil {
		t.Errorf("Expected no error for empty data, got %v", err)
	}
}

// TestBatchMetrics 测试BatchMetrics结构
func TestBatchMetrics(t *testing.T) {
	now := time.Now()
	duration := time.Second
	testError := errors.New("test error")

	metrics := BatchMetrics{
		Driver:    "mysql",
		Table:     "users",
		BatchSize: 100,
		Duration:  duration,
		Error:     testError,
		StartTime: now,
	}

	if metrics.Driver != "mysql" {
		t.Errorf("Expected driver 'mysql', got '%s'", metrics.Driver)
	}
	if metrics.Table != "users" {
		t.Errorf("Expected table 'users', got '%s'", metrics.Table)
	}
	if metrics.BatchSize != 100 {
		t.Errorf("Expected batch size 100, got %d", metrics.BatchSize)
	}
	if metrics.Duration != duration {
		t.Errorf("Expected duration %v, got %v", duration, metrics.Duration)
	}
	if metrics.Error != testError {
		t.Errorf("Expected error %v, got %v", testError, metrics.Error)
	}
	if metrics.StartTime != now {
		t.Errorf("Expected start time %v, got %v", now, metrics.StartTime)
	}
}
