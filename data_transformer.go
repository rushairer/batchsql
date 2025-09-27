package batchsql

import (
	"fmt"
	"reflect"
	"time"
)

// DefaultDataTransformer 默认数据转换器实现
type DefaultDataTransformer struct {
	typeConverters map[string]TypeConverter
}

// TypeConverter 类型转换器接口
type TypeConverter interface {
	Convert(value interface{}, targetType string) (interface{}, error)
	SupportedTypes() []string
}

// NewDefaultDataTransformer 创建默认数据转换器
func NewDefaultDataTransformer() *DefaultDataTransformer {
	transformer := &DefaultDataTransformer{
		typeConverters: make(map[string]TypeConverter),
	}

	// 注册默认转换器
	transformer.RegisterConverter("sql", NewSQLTypeConverter())
	transformer.RegisterConverter("redis", NewRedisTypeConverter())
	transformer.RegisterConverter("mongodb", NewMongoDBTypeConverter())

	return transformer
}

// RegisterConverter 注册类型转换器
func (dt *DefaultDataTransformer) RegisterConverter(driverType string, converter TypeConverter) {
	dt.typeConverters[driverType] = converter
}

// TransformRequest 转换请求数据
func (dt *DefaultDataTransformer) TransformRequest(request *Request, schema SchemaInterface) (interface{}, error) {
	driverName := schema.GetDatabaseDriver().GetName()
	converter := dt.getConverterForDriver(driverName)

	if converter == nil {
		return request.Columns(), nil // 返回原始数据
	}

	transformed := make(map[string]interface{})
	columns := request.Columns()

	for key, value := range columns {
		// 根据列名和驱动类型进行转换
		convertedValue, err := converter.Convert(value, dt.inferTargetType(key, value, driverName))
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}
		transformed[key] = convertedValue
	}

	return transformed, nil
}

// TransformBatch 转换批量数据
func (dt *DefaultDataTransformer) TransformBatch(requests []*Request, schema SchemaInterface) ([]interface{}, error) {
	if len(requests) == 0 {
		return nil, nil
	}

	transformed := make([]interface{}, len(requests))
	for i, request := range requests {
		transformedRequest, err := dt.TransformRequest(request, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to transform request %d: %w", i, err)
		}
		transformed[i] = transformedRequest
	}

	return transformed, nil
}

// getConverterForDriver 根据驱动获取转换器
func (dt *DefaultDataTransformer) getConverterForDriver(driverName string) TypeConverter {
	switch {
	case driverName == "mysql" || driverName == "postgresql" || driverName == "sqlite":
		return dt.typeConverters["sql"]
	case driverName == "redis" || driverName == "redis-hash" || driverName == "redis-set":
		return dt.typeConverters["redis"]
	case driverName == "mongodb" || driverName == "mongodb-timeseries":
		return dt.typeConverters["mongodb"]
	default:
		return nil
	}
}

// inferTargetType 推断目标类型
func (dt *DefaultDataTransformer) inferTargetType(fieldName string, value interface{}, driverName string) string {
	if value == nil {
		return "null"
	}

	valueType := reflect.TypeOf(value)
	switch valueType.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "boolean"
	default:
		if _, ok := value.(time.Time); ok {
			return "datetime"
		}
		return "string" // 默认转换为字符串
	}
}

// SQLTypeConverter SQL类型转换器
type SQLTypeConverter struct{}

func NewSQLTypeConverter() *SQLTypeConverter {
	return &SQLTypeConverter{}
}

func (c *SQLTypeConverter) Convert(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch targetType {
	case "string":
		return fmt.Sprintf("%v", value), nil
	case "integer":
		return c.convertToInt64(value)
	case "float":
		return c.convertToFloat64(value)
	case "boolean":
		return c.convertToBool(value)
	case "datetime":
		return c.convertToTime(value)
	default:
		return value, nil
	}
}

func (c *SQLTypeConverter) SupportedTypes() []string {
	return []string{"string", "integer", "float", "boolean", "datetime", "null"}
}

func (c *SQLTypeConverter) convertToInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		// 尝试解析字符串
		return 0, fmt.Errorf("string to int64 conversion not implemented")
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

func (c *SQLTypeConverter) convertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func (c *SQLTypeConverter) convertToBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case int:
		return v != 0, nil
	case string:
		return v == "true" || v == "1", nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

func (c *SQLTypeConverter) convertToTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		// 尝试解析常见的时间格式
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("cannot parse time string: %s", v)
	default:
		return time.Time{}, fmt.Errorf("cannot convert %T to time.Time", value)
	}
}

// RedisTypeConverter Redis类型转换器
type RedisTypeConverter struct{}

func NewRedisTypeConverter() *RedisTypeConverter {
	return &RedisTypeConverter{}
}

func (c *RedisTypeConverter) Convert(value interface{}, targetType string) (interface{}, error) {
	// Redis主要存储字符串，所以大部分值都转换为字符串
	if value == nil {
		return "", nil
	}

	switch targetType {
	case "string":
		return fmt.Sprintf("%v", value), nil
	case "datetime":
		if t, ok := value.(time.Time); ok {
			return t.Unix(), nil // 转换为Unix时间戳
		}
		return fmt.Sprintf("%v", value), nil
	default:
		return fmt.Sprintf("%v", value), nil
	}
}

func (c *RedisTypeConverter) SupportedTypes() []string {
	return []string{"string", "datetime"}
}

// MongoDBTypeConverter MongoDB类型转换器
type MongoDBTypeConverter struct{}

func NewMongoDBTypeConverter() *MongoDBTypeConverter {
	return &MongoDBTypeConverter{}
}

func (c *MongoDBTypeConverter) Convert(value interface{}, targetType string) (interface{}, error) {
	// MongoDB支持丰富的数据类型，大部分可以直接存储
	if value == nil {
		return nil, nil
	}

	switch targetType {
	case "datetime":
		if t, ok := value.(time.Time); ok {
			return t, nil // MongoDB原生支持时间类型
		}
		// 尝试解析字符串时间
		if s, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t, nil
			}
		}
		return value, nil
	default:
		return value, nil // MongoDB可以存储大部分Go类型
	}
}

func (c *MongoDBTypeConverter) SupportedTypes() []string {
	return []string{"string", "integer", "float", "boolean", "datetime", "object", "array", "null"}
}

// ValidationTransformer 验证转换器
type ValidationTransformer struct {
	*DefaultDataTransformer
	validators map[string]FieldValidator
}

// FieldValidator 字段验证器接口
type FieldValidator interface {
	Validate(fieldName string, value interface{}) error
}

// NewValidationTransformer 创建验证转换器
func NewValidationTransformer() *ValidationTransformer {
	return &ValidationTransformer{
		DefaultDataTransformer: NewDefaultDataTransformer(),
		validators:             make(map[string]FieldValidator),
	}
}

// RegisterValidator 注册字段验证器
func (vt *ValidationTransformer) RegisterValidator(fieldName string, validator FieldValidator) {
	vt.validators[fieldName] = validator
}

// TransformRequest 转换并验证请求数据
func (vt *ValidationTransformer) TransformRequest(request *Request, schema SchemaInterface) (interface{}, error) {
	// 先进行验证
	columns := request.Columns()
	for fieldName, value := range columns {
		if validator, exists := vt.validators[fieldName]; exists {
			if err := validator.Validate(fieldName, value); err != nil {
				return nil, fmt.Errorf("validation failed for field %s: %w", fieldName, err)
			}
		}
	}

	// 然后进行转换
	return vt.DefaultDataTransformer.TransformRequest(request, schema)
}

// StringLengthValidator 字符串长度验证器
type StringLengthValidator struct {
	MinLength int
	MaxLength int
}

func (v *StringLengthValidator) Validate(fieldName string, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("field %s must be a string", fieldName)
	}

	if len(str) < v.MinLength {
		return fmt.Errorf("field %s must be at least %d characters", fieldName, v.MinLength)
	}

	if v.MaxLength > 0 && len(str) > v.MaxLength {
		return fmt.Errorf("field %s must be at most %d characters", fieldName, v.MaxLength)
	}

	return nil
}

// RangeValidator 数值范围验证器
type RangeValidator struct {
	Min interface{}
	Max interface{}
}

func (v *RangeValidator) Validate(fieldName string, value interface{}) error {
	// 简化实现，实际应该支持更多数值类型
	switch val := value.(type) {
	case int:
		if v.Min != nil {
			if min, ok := v.Min.(int); ok && val < min {
				return fmt.Errorf("field %s must be at least %d", fieldName, min)
			}
		}
		if v.Max != nil {
			if max, ok := v.Max.(int); ok && val > max {
				return fmt.Errorf("field %s must be at most %d", fieldName, max)
			}
		}
	case float64:
		if v.Min != nil {
			if min, ok := v.Min.(float64); ok && val < min {
				return fmt.Errorf("field %s must be at least %f", fieldName, min)
			}
		}
		if v.Max != nil {
			if max, ok := v.Max.(float64); ok && val > max {
				return fmt.Errorf("field %s must be at most %f", fieldName, max)
			}
		}
	}

	return nil
}
