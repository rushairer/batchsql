package batchsql

import (
	"fmt"
	"reflect"
)

// Request represents a single database operation request
type Request struct {
	schema SchemaInterface
	data   map[string]interface{}
}

// NewRequest creates a new request with the given schema
func NewRequest(schema SchemaInterface) *Request {
	return &Request{
		schema: schema,
		data:   make(map[string]interface{}),
	}
}

// NewRequestFromInterface creates a new request from a schema interface
func NewRequestFromInterface(schema SchemaInterface) *Request {
	return NewRequest(schema)
}

// Set sets a value for the given key
func (r *Request) Set(key string, value interface{}) {
	r.data[key] = value
}

// Get gets a value for the given key
func (r *Request) Get(key string) (interface{}, bool) {
	value, exists := r.data[key]
	return value, exists
}

// GetString gets a string value for the given key
func (r *Request) GetString(key string) string {
	if value, exists := r.data[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// GetInt gets an int value for the given key
func (r *Request) GetInt(key string) int {
	if value, exists := r.data[key]; exists {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetInt64 gets an int64 value for the given key
func (r *Request) GetInt64(key string) int64 {
	if value, exists := r.data[key]; exists {
		switch v := value.(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}

// GetFloat64 gets a float64 value for the given key
func (r *Request) GetFloat64(key string) float64 {
	if value, exists := r.data[key]; exists {
		switch v := value.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0.0
}

// GetBool gets a bool value for the given key
func (r *Request) GetBool(key string) bool {
	if value, exists := r.data[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// GetData returns all data as a map
func (r *Request) GetData() map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range r.data {
		result[k] = v
	}
	return result
}

// GetOrderedValues returns values in the order of schema columns
func (r *Request) GetOrderedValues() []interface{} {
	columns := r.schema.GetColumns()
	values := make([]interface{}, len(columns))

	for i, column := range columns {
		if value, exists := r.data[column]; exists {
			values[i] = value
		} else {
			values[i] = nil
		}
	}

	return values
}

// Schema returns the associated schema
func (r *Request) Schema() SchemaInterface {
	return r.schema
}

// Validate validates the request against its schema
func (r *Request) Validate() error {
	if r.schema == nil {
		return fmt.Errorf("request has no associated schema")
	}

	// Validate schema first
	if err := r.schema.Validate(); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	// Check required columns
	columns := r.schema.GetColumns()
	for _, column := range columns {
		if _, exists := r.data[column]; !exists {
			// For now, we don't enforce required fields
			// In a real implementation, you might have required field metadata
		}
	}

	return nil
}

// Clone creates a copy of the request
func (r *Request) Clone() *Request {
	clone := &Request{
		schema: r.schema,
		data:   make(map[string]interface{}),
	}

	for k, v := range r.data {
		clone.data[k] = cloneValue(v)
	}

	return clone
}

// cloneValue creates a deep copy of a value
func cloneValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Slice:
		slice := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			slice[i] = cloneValue(v.Index(i).Interface())
		}
		return slice
	case reflect.Map:
		m := make(map[string]interface{})
		for _, key := range v.MapKeys() {
			m[key.String()] = cloneValue(v.MapIndex(key).Interface())
		}
		return m
	default:
		return value
	}
}

// String returns a string representation of the request
func (r *Request) String() string {
	return fmt.Sprintf("Request{schema: %s, data: %+v}", r.schema.GetIdentifier(), r.data)
}
