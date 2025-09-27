package batchsql

import "fmt"

// UniversalSchema 通用 Schema 实现
type UniversalSchema struct {
	identifier       string
	conflictStrategy ConflictStrategy
	columns          []string
	driver           DatabaseDriver
	metadata         map[string]interface{}
}

// NewUniversalSchema 创建通用 Schema
func NewUniversalSchema(
	identifier string,
	conflictStrategy ConflictStrategy,
	driver DatabaseDriver,
	columns ...string,
) *UniversalSchema {
	return &UniversalSchema{
		identifier:       identifier,
		conflictStrategy: conflictStrategy,
		columns:          columns,
		driver:           driver,
		metadata:         make(map[string]interface{}),
	}
}

// GetIdentifier 获取标识符（表名/集合名/键前缀等）
func (s *UniversalSchema) GetIdentifier() string {
	return s.identifier
}

// GetConflictStrategy 获取冲突策略
func (s *UniversalSchema) GetConflictStrategy() ConflictStrategy {
	return s.conflictStrategy
}

// GetColumns 获取列名
func (s *UniversalSchema) GetColumns() []string {
	return s.columns
}

// GetDatabaseDriver 获取数据库驱动
func (s *UniversalSchema) GetDatabaseDriver() DatabaseDriver {
	return s.driver
}

// SetMetadata 设置元数据
func (s *UniversalSchema) SetMetadata(key string, value interface{}) {
	s.metadata[key] = value
}

// GetMetadata 获取元数据
func (s *UniversalSchema) GetMetadata(key string) (interface{}, bool) {
	value, exists := s.metadata[key]
	return value, exists
}

// GetAllMetadata 获取所有元数据
func (s *UniversalSchema) GetAllMetadata() map[string]interface{} {
	return s.metadata
}

// Validate 验证 Schema
func (s *UniversalSchema) Validate() error {
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
func (s *UniversalSchema) Clone() SchemaInterface {
	newSchema := &UniversalSchema{
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
func (s *UniversalSchema) WithConflictStrategy(strategy ConflictStrategy) *UniversalSchema {
	s.conflictStrategy = strategy
	return s
}

// WithColumns 设置列名（链式调用）
func (s *UniversalSchema) WithColumns(columns ...string) *UniversalSchema {
	s.columns = columns
	return s
}

// WithMetadata 设置元数据（链式调用）
func (s *UniversalSchema) WithMetadata(key string, value interface{}) *UniversalSchema {
	s.SetMetadata(key, value)
	return s
}

// AddColumn 添加列
func (s *UniversalSchema) AddColumn(column string) *UniversalSchema {
	s.columns = append(s.columns, column)
	return s
}

// RemoveColumn 移除列
func (s *UniversalSchema) RemoveColumn(column string) *UniversalSchema {
	for i, col := range s.columns {
		if col == column {
			s.columns = append(s.columns[:i], s.columns[i+1:]...)
			break
		}
	}
	return s
}

// HasColumn 检查是否包含指定列
func (s *UniversalSchema) HasColumn(column string) bool {
	for _, col := range s.columns {
		if col == column {
			return true
		}
	}
	return false
}

// GetColumnIndex 获取列的索引
func (s *UniversalSchema) GetColumnIndex(column string) int {
	for i, col := range s.columns {
		if col == column {
			return i
		}
	}
	return -1
}

// String 字符串表示
func (s *UniversalSchema) String() string {
	return fmt.Sprintf("Schema{driver=%s, identifier=%s, strategy=%v, columns=%v}",
		s.driver.GetName(), s.identifier, s.conflictStrategy, s.columns)
}

// NewSchema 创建Schema的便捷函数
func NewSchema(identifier string, strategy ConflictStrategy, driver DatabaseDriver, columns ...string) SchemaInterface {
	return NewUniversalSchema(identifier, strategy, driver, columns...)
}
