package drivers

import (
	"fmt"

	"github.com/rushairer/batchsql"
)

// MongoCommand MongoDB 命令实现
type MongoCommand struct {
	commandType string
	operations  []interface{} // MongoDB 批量操作
	metadata    map[string]interface{}
}

func (c *MongoCommand) GetCommandType() string {
	return c.commandType
}

func (c *MongoCommand) GetCommand() interface{} {
	return c.operations
}

func (c *MongoCommand) GetParameters() []interface{} {
	return nil // MongoDB 操作参数已包含在 operations 中
}

func (c *MongoCommand) GetMetadata() map[string]interface{} {
	return c.metadata
}

// MongoOperation MongoDB 操作结构
type MongoOperation struct {
	Type       string                 `json:"type"`       // insert, update, replace
	Filter     map[string]interface{} `json:"filter"`     // 查询条件
	Document   map[string]interface{} `json:"document"`   // 文档数据
	Update     map[string]interface{} `json:"update"`     // 更新操作
	Upsert     bool                   `json:"upsert"`     // 是否 upsert
	Collection string                 `json:"collection"` // 集合名
}

// MongoDBDriver MongoDB 驱动
type MongoDBDriver struct {
	name string
}

func NewMongoDBDriver() *MongoDBDriver {
	return &MongoDBDriver{name: "mongodb"}
}

func (d *MongoDBDriver) GetName() string {
	return d.name
}

func (d *MongoDBDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty requests")
	}

	columns := schema.GetColumns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns defined")
	}

	collection := schema.GetIdentifier()
	operations := make([]interface{}, 0, len(requests))

	for _, request := range requests {
		values := request.GetOrderedValues()
		if len(values) != len(columns) {
			return nil, fmt.Errorf("column count mismatch")
		}

		// 构建文档
		document := make(map[string]interface{})
		for i, col := range columns {
			document[col] = values[i]
		}

		// 假设第一列是主键 (_id 或唯一标识)
		filter := map[string]interface{}{
			columns[0]: values[0],
		}

		var operation *MongoOperation

		switch schema.GetConflictStrategy() {
		case batchsql.ConflictIgnore:
			// 只在文档不存在时插入
			operation = &MongoOperation{
				Type:       "insert",
				Document:   document,
				Collection: collection,
			}
		case batchsql.ConflictReplace:
			// 替换整个文档
			operation = &MongoOperation{
				Type:       "replace",
				Filter:     filter,
				Document:   document,
				Upsert:     true,
				Collection: collection,
			}
		case batchsql.ConflictUpdate:
			// 更新文档
			updateDoc := make(map[string]interface{})
			for i := 1; i < len(columns); i++ { // 跳过主键
				updateDoc[columns[i]] = values[i]
			}
			operation = &MongoOperation{
				Type:   "update",
				Filter: filter,
				Update: map[string]interface{}{
					"$set": updateDoc,
				},
				Upsert:     true,
				Collection: collection,
			}
		default:
			// 默认插入
			operation = &MongoOperation{
				Type:       "insert",
				Document:   document,
				Collection: collection,
			}
		}

		operations = append(operations, operation)
	}

	return &MongoCommand{
		commandType: "MONGODB",
		operations:  operations,
		metadata: map[string]interface{}{
			"collection": collection,
			"batch_size": len(requests),
			"driver":     d.name,
		},
	}, nil
}

func (d *MongoDBDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictReplace,
		batchsql.ConflictUpdate,
	}
}

func (d *MongoDBDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if schema.GetIdentifier() == "" {
		return fmt.Errorf("collection name cannot be empty")
	}
	columns := schema.GetColumns()
	if len(columns) == 0 {
		return fmt.Errorf("columns cannot be empty")
	}

	supported := d.SupportedConflictStrategies()
	strategy := schema.GetConflictStrategy()
	for _, s := range supported {
		if s == strategy {
			return nil
		}
	}
	return fmt.Errorf("unsupported conflict strategy: %v", strategy)
}

// MongoTimeSeriesDriver MongoDB 时间序列集合驱动
type MongoTimeSeriesDriver struct {
	MongoDBDriver
	timeField   string
	metaField   string
	granularity string
}

func NewMongoTimeSeriesDriver(timeField, metaField, granularity string) *MongoTimeSeriesDriver {
	return &MongoTimeSeriesDriver{
		MongoDBDriver: MongoDBDriver{name: "mongodb-timeseries"},
		timeField:     timeField,
		metaField:     metaField,
		granularity:   granularity,
	}
}

func (d *MongoTimeSeriesDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty requests")
	}

	columns := schema.GetColumns()
	collection := schema.GetIdentifier()
	operations := make([]interface{}, 0, len(requests))

	for _, request := range requests {
		values := request.GetOrderedValues()
		if len(values) != len(columns) {
			return nil, fmt.Errorf("column count mismatch")
		}

		// 构建时间序列文档
		document := make(map[string]interface{})
		for i, col := range columns {
			document[col] = values[i]
		}

		// 时间序列集合通常只支持插入
		operation := &MongoOperation{
			Type:       "insert",
			Document:   document,
			Collection: collection,
		}

		operations = append(operations, operation)
	}

	return &MongoCommand{
		commandType: "MONGODB",
		operations:  operations,
		metadata: map[string]interface{}{
			"collection":    collection,
			"batch_size":    len(requests),
			"driver":        d.name,
			"time_field":    d.timeField,
			"meta_field":    d.metaField,
			"granularity":   d.granularity,
			"is_timeseries": true,
		},
	}, nil
}

func (d *MongoTimeSeriesDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	// 时间序列集合通常只支持插入
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
	}
}

func (d *MongoTimeSeriesDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if err := d.MongoDBDriver.ValidateSchema(schema); err != nil {
		return err
	}

	columns := schema.GetColumns()

	// 验证时间字段是否存在
	timeFieldExists := false
	for _, col := range columns {
		if col == d.timeField {
			timeFieldExists = true
			break
		}
	}
	if !timeFieldExists {
		return fmt.Errorf("time field '%s' not found in columns", d.timeField)
	}

	// 验证元数据字段是否存在（如果指定了）
	if d.metaField != "" {
		metaFieldExists := false
		for _, col := range columns {
			if col == d.metaField {
				metaFieldExists = true
				break
			}
		}
		if !metaFieldExists {
			return fmt.Errorf("meta field '%s' not found in columns", d.metaField)
		}
	}

	return nil
}
