package batchsql

import "fmt"

// Schema 数据结构定义
type Schema struct {
	identifier       string
	conflictStrategy ConflictStrategy
	columns          []string
	driver           DatabaseDriver
	metadata         map[string]interface{}
}

// NewSchema 创建 Schema
func NewSchema(
	identifier string,
	conflictStrategy ConflictStrategy,
	driver DatabaseDriver,
	columns ...string,
) *Schema {
	return &Schema{
		identifier:       identifier,
		conflictStrategy: conflictStrategy,
		columns:          columns,
		driver:           driver,
		metadata:         make(map[string]interface{}),
	}
}

// GetIdentifier 获取标识符（表名/集合名/键前缀等）
func (s *Schema) GetIdentifier() string {
	return s.identifier
}

// GetConflictStrategy 获取冲突策略
func (s *Schema) GetConflictStrategy() ConflictStrategy {
	return s.conflictStrategy
}

// GetColumns 获取列名
func (s *Schema) GetColumns() []string {
	return s.columns
}

// GetDatabaseDriver 获取数据库驱动
func (s *Schema) GetDatabaseDriver() DatabaseDriver {
	return s.driver
}

// SetMetadata 设置元数据
func (s *Schema) SetMetadata(key string, value interface{}) {
	s.metadata[key] = value
}

// GetMetadata 获取元数据
func (s *Schema) GetMetadata(key string) (interface{}, bool) {
	value, exists := s.metadata[key]
	return value, exists
}

// GetAllMetadata 获取所有元数据
func (s *Schema) GetAllMetadata() map[string]interface{} {
	return s.metadata
}

// Validate 验证 Schema
func (s *Schema) Validate() error {
	if s.identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if len(s.columns) == 0 {
		return fmt.Errorf("columns cannot be empty")
	}
	if s.driver == nil {
		return fmt.Errorf("database driver cannot be nil")
	}

	// 使用驱动验证
	return s.driver.ValidateSchema(s)
}

// Clone 克隆 Schema
func (s *Schema) Clone() SchemaInterface {
	newSchema := &Schema{
		identifier:       s.identifier,
		conflictStrategy: s.conflictStrategy,
		columns:          make([]string, len(s.columns)),
		driver:           s.driver,
		metadata:         make(map[string]interface{}),
	}

	copy(newSchema.columns, s.columns)
	for k, v := range s.metadata {
		newSchema.metadata[k] = v
	}

	return newSchema
}

// WithConflictStrategy 设置冲突策略（链式调用）
func (s *Schema) WithConflictStrategy(strategy ConflictStrategy) *Schema {
	s.conflictStrategy = strategy
	return s
}

// WithColumns 设置列名（链式调用）
func (s *Schema) WithColumns(columns ...string) *Schema {
	s.columns = columns
	return s
}

// WithMetadata 设置元数据（链式调用）
func (s *Schema) WithMetadata(key string, value interface{}) *Schema {
	s.SetMetadata(key, value)
	return s
}

// AddColumn 添加列
func (s *Schema) AddColumn(column string) *Schema {
	s.columns = append(s.columns, column)
	return s
}

// RemoveColumn 移除列
func (s *Schema) RemoveColumn(column string) *Schema {
	for i, col := range s.columns {
		if col == column {
			s.columns = append(s.columns[:i], s.columns[i+1:]...)
			break
		}
	}
	return s
}

// HasColumn 检查是否包含指定列
func (s *Schema) HasColumn(column string) bool {
	for _, col := range s.columns {
		if col == column {
			return true
		}
	}
	return false
}

// GetColumnIndex 获取列的索引
func (s *Schema) GetColumnIndex(column string) int {
	for i, col := range s.columns {
		if col == column {
			return i
		}
	}
	return -1
}

// String 字符串表示
func (s *Schema) String() string {
	return fmt.Sprintf("Schema{driver=%s, identifier=%s, strategy=%v, columns=%v}",
		s.driver.GetName(), s.identifier, s.conflictStrategy, s.columns)
}
