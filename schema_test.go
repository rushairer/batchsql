package batchsql

import (
	"testing"
)

// TestNewSchema 测试Schema创建
func TestNewSchema(t *testing.T) {
	mockDriver := NewMockDriver("mysql")
	columns := []string{"id", "name", "age"}

	schema := NewSchema("users", ConflictIgnore, mockDriver, columns...)

	if schema.GetIdentifier() != "users" {
		t.Errorf("Expected identifier 'users', got '%s'", schema.GetIdentifier())
	}

	if schema.GetConflictStrategy() != ConflictIgnore {
		t.Errorf("Expected conflict strategy ConflictIgnore, got %v", schema.GetConflictStrategy())
	}

	schemaColumns := schema.GetColumns()
	if len(schemaColumns) != len(columns) {
		t.Errorf("Expected %d columns, got %d", len(columns), len(schemaColumns))
	}

	for i, col := range columns {
		if schemaColumns[i] != col {
			t.Errorf("Expected column '%s' at index %d, got '%s'", col, i, schemaColumns[i])
		}
	}

	if schema.GetDatabaseDriver() != mockDriver {
		t.Error("Expected database driver to match")
	}
}

// TestSchemaValidation 测试Schema验证
func TestSchemaValidation(t *testing.T) {
	mockDriver := NewMockDriver("mysql")

	// 测试有效Schema
	validSchema := NewSchema("users", ConflictIgnore, mockDriver, "id", "name")
	err := validSchema.Validate()
	if err != nil {
		t.Errorf("Valid schema should not return error, got: %v", err)
	}

	// 测试空标识符
	invalidSchema1 := NewSchema("", ConflictIgnore, mockDriver, "id")
	err = invalidSchema1.Validate()
	if err == nil {
		t.Error("Schema with empty identifier should return error")
	}

	// 测试空列
	invalidSchema2 := NewSchema("users", ConflictIgnore, mockDriver)
	err = invalidSchema2.Validate()
	if err == nil {
		t.Error("Schema with no columns should return error")
	}

	// 测试空驱动
	invalidSchema3 := NewSchema("users", ConflictIgnore, nil, "id")
	err = invalidSchema3.Validate()
	if err == nil {
		t.Error("Schema with nil driver should return error")
	}
}

// TestSchemaClone 测试Schema克隆
func TestSchemaClone(t *testing.T) {
	mockDriver := NewMockDriver("mysql")
	original := NewSchema("users", ConflictReplace, mockDriver, "id", "name")
	original.SetMetadata("test_key", "test_value")

	cloned := original.Clone()

	// 验证基本属性
	if cloned.GetIdentifier() != original.GetIdentifier() {
		t.Error("Cloned schema identifier should match original")
	}

	if cloned.GetConflictStrategy() != original.GetConflictStrategy() {
		t.Error("Cloned schema conflict strategy should match original")
	}

	if cloned.GetDatabaseDriver() != original.GetDatabaseDriver() {
		t.Error("Cloned schema driver should match original")
	}

	// 验证列
	originalColumns := original.GetColumns()
	clonedColumns := cloned.GetColumns()
	if len(originalColumns) != len(clonedColumns) {
		t.Error("Cloned schema should have same number of columns")
	}

	for i, col := range originalColumns {
		if clonedColumns[i] != col {
			t.Errorf("Cloned column at index %d should be '%s', got '%s'", i, col, clonedColumns[i])
		}
	}

	// 验证元数据
	clonedSchema := cloned.(*Schema)
	value, exists := clonedSchema.GetMetadata("test_key")
	if !exists {
		t.Error("Cloned schema should have metadata")
	}
	if value != "test_value" {
		t.Errorf("Expected metadata value 'test_value', got '%v'", value)
	}
}

// TestSchemaChainMethods 测试链式调用方法
func TestSchemaChainMethods(t *testing.T) {
	mockDriver := NewMockDriver("mysql")

	schema := NewSchema("users", ConflictIgnore, mockDriver, "id").
		WithConflictStrategy(ConflictReplace).
		WithColumns("id", "name", "email").
		WithMetadata("version", "1.0")

	if schema.GetConflictStrategy() != ConflictReplace {
		t.Error("WithConflictStrategy should update conflict strategy")
	}

	columns := schema.GetColumns()
	expectedColumns := []string{"id", "name", "email"}
	if len(columns) != len(expectedColumns) {
		t.Errorf("Expected %d columns, got %d", len(expectedColumns), len(columns))
	}

	value, exists := schema.GetMetadata("version")
	if !exists || value != "1.0" {
		t.Error("WithMetadata should set metadata")
	}
}

// TestSchemaColumnOperations 测试列操作
func TestSchemaColumnOperations(t *testing.T) {
	mockDriver := NewMockDriver("mysql")
	schema := NewSchema("users", ConflictIgnore, mockDriver, "id", "name")

	// 测试添加列
	schema.AddColumn("email")
	if !schema.HasColumn("email") {
		t.Error("AddColumn should add the column")
	}

	// 测试列索引
	index := schema.GetColumnIndex("name")
	if index != 1 {
		t.Errorf("Expected column 'name' at index 1, got %d", index)
	}

	// 测试移除列
	schema.RemoveColumn("name")
	if schema.HasColumn("name") {
		t.Error("RemoveColumn should remove the column")
	}

	// 测试不存在的列
	if schema.HasColumn("nonexistent") {
		t.Error("HasColumn should return false for non-existent column")
	}

	index = schema.GetColumnIndex("nonexistent")
	if index != -1 {
		t.Errorf("Expected -1 for non-existent column, got %d", index)
	}
}

// TestSchemaMetadata 测试元数据操作
func TestSchemaMetadata(t *testing.T) {
	mockDriver := NewMockDriver("mysql")
	schema := NewSchema("users", ConflictIgnore, mockDriver, "id")

	// 设置元数据
	schema.SetMetadata("key1", "value1")
	schema.SetMetadata("key2", 42)

	// 获取元数据
	value1, exists1 := schema.GetMetadata("key1")
	if !exists1 || value1 != "value1" {
		t.Error("Should be able to get string metadata")
	}

	value2, exists2 := schema.GetMetadata("key2")
	if !exists2 || value2 != 42 {
		t.Error("Should be able to get integer metadata")
	}

	// 获取不存在的元数据
	_, exists3 := schema.GetMetadata("nonexistent")
	if exists3 {
		t.Error("Should return false for non-existent metadata")
	}

	// 获取所有元数据
	allMetadata := schema.GetAllMetadata()
	if len(allMetadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(allMetadata))
	}
}

// TestSchemaString 测试字符串表示
func TestSchemaString(t *testing.T) {
	mockDriver := NewMockDriver("mysql")
	schema := NewSchema("users", ConflictReplace, mockDriver, "id", "name")

	str := schema.String()
	if str == "" {
		t.Error("String() should return non-empty string")
	}

	// 验证字符串包含关键信息
	if !contains(str, "mysql") {
		t.Error("String should contain driver name")
	}
	if !contains(str, "users") {
		t.Error("String should contain identifier")
	}
}

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
