package batchsql

import (
	"fmt"
	"time"

	"github.com/rushairer/batchsql/drivers"
)

// 用来存储请求的数据的各种字段信息和对应的schema
type Request struct {
	schema  *drivers.Schema
	columns map[string]any // 使用 map 存储列名到值的映射
}

func NewRequest(schema *drivers.Schema) *Request {
	return &Request{
		schema:  schema,
		columns: make(map[string]any),
	}
}

// Schema 获取请求的 schema
func (r *Request) Schema() *drivers.Schema {
	return r.schema
}

// Columns 获取所有列数据
func (r *Request) Columns() map[string]any {
	return r.columns
}

// GetOrderedValues 按照 schema 中定义的列顺序返回值
func (r *Request) GetOrderedValues() []any {
	values := make([]any, len(r.schema.Columns))
	for i, colName := range r.schema.Columns {
		values[i] = r.columns[colName]
	}
	return values
}

// 类型化的设置方法
func (r *Request) SetInt32(colName string, value int32) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetInt64(colName string, value int64) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetFloat32(colName string, value float32) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetFloat64(colName string, value float64) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetString(colName string, value string) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetBool(colName string, value bool) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetTime(colName string, value time.Time) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetBytes(colName string, value []byte) *Request {
	r.columns[colName] = value
	return r
}

func (r *Request) SetNull(colName string) *Request {
	r.columns[colName] = nil
	return r
}

// 通用设置方法
func (r *Request) Set(colName string, value any) *Request {
	r.columns[colName] = value
	return r
}

// 类型化的获取方法
func (r *Request) GetInt32(colName string) (int32, error) {
	value, exists := r.columns[colName]
	if !exists {
		return 0, fmt.Errorf("column %s not found", colName)
	}
	if v, ok := value.(int32); ok {
		return v, nil
	}
	return 0, fmt.Errorf("column %s is not int32", colName)
}

func (r *Request) GetInt64(colName string) (int64, error) {
	value, exists := r.columns[colName]
	if !exists {
		return 0, fmt.Errorf("column %s not found", colName)
	}
	if v, ok := value.(int64); ok {
		return v, nil
	}
	return 0, fmt.Errorf("column %s is not int64", colName)
}

func (r *Request) GetString(colName string) (string, error) {
	value, exists := r.columns[colName]
	if !exists {
		return "", fmt.Errorf("column %s not found", colName)
	}
	if v, ok := value.(string); ok {
		return v, nil
	}
	return "", fmt.Errorf("column %s is not string", colName)
}

func (r *Request) GetFloat64(colName string) (float64, error) {
	value, exists := r.columns[colName]
	if !exists {
		return 0, fmt.Errorf("column %s not found", colName)
	}
	if v, ok := value.(float64); ok {
		return v, nil
	}
	return 0, fmt.Errorf("column %s is not float64", colName)
}

func (r *Request) GetBool(colName string) (bool, error) {
	value, exists := r.columns[colName]
	if !exists {
		return false, fmt.Errorf("column %s not found", colName)
	}
	if v, ok := value.(bool); ok {
		return v, nil
	}
	return false, fmt.Errorf("column %s is not bool", colName)
}

func (r *Request) GetTime(colName string) (time.Time, error) {
	value, exists := r.columns[colName]
	if !exists {
		return time.Time{}, fmt.Errorf("column %s not found", colName)
	}
	if v, ok := value.(time.Time); ok {
		return v, nil
	}
	return time.Time{}, fmt.Errorf("column %s is not time.Time", colName)
}

// 验证请求是否包含所有必需的列
func (r *Request) Validate() error {
	for _, colName := range r.schema.Columns {
		if _, exists := r.columns[colName]; !exists {
			return fmt.Errorf("missing required column: %s", colName)
		}
	}
	return nil
}
