package drivers

import (
	"fmt"

	"github.com/rushairer/batchsql"
)

// RedisCommand Redis 命令实现
type RedisCommand struct {
	commandType string
	commands    [][]interface{} // Redis 支持管道操作
	metadata    map[string]interface{}
}

func (c *RedisCommand) GetCommandType() string {
	return c.commandType
}

func (c *RedisCommand) GetCommand() interface{} {
	return c.commands
}

func (c *RedisCommand) GetParameters() []interface{} {
	// Redis 命令参数已经包含在 commands 中
	return nil
}

func (c *RedisCommand) GetMetadata() map[string]interface{} {
	return c.metadata
}

// RedisDriver Redis 驱动
type RedisDriver struct {
	name string
}

func NewRedisDriver() *RedisDriver {
	return &RedisDriver{name: "redis"}
}

func (d *RedisDriver) GetName() string {
	return d.name
}

func (d *RedisDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty requests")
	}

	columns := schema.GetColumns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns defined")
	}

	keyPrefix := schema.GetIdentifier() // Redis 中用作键前缀
	commands := make([][]interface{}, 0, len(requests))

	for _, request := range requests {
		values := request.GetOrderedValues()
		if len(values) != len(columns) {
			return nil, fmt.Errorf("column count mismatch")
		}

		// 构建 Redis 键，假设第一列是主键
		key := fmt.Sprintf("%s:%v", keyPrefix, values[0])

		switch schema.GetConflictStrategy() {
		case batchsql.ConflictIgnore:
			// 使用 HSETNX (只在字段不存在时设置)
			for i := 1; i < len(columns); i++ {
				cmd := []interface{}{"HSETNX", key, columns[i], values[i]}
				commands = append(commands, cmd)
			}
		case batchsql.ConflictReplace, batchsql.ConflictUpdate:
			// 使用 HSET (覆盖或更新)
			if len(columns) > 1 {
				cmd := []interface{}{"HSET", key}
				for i := 1; i < len(columns); i++ {
					cmd = append(cmd, columns[i], values[i])
				}
				commands = append(commands, cmd)
			}
		default:
			// 默认使用 HSET
			if len(columns) > 1 {
				cmd := []interface{}{"HSET", key}
				for i := 1; i < len(columns); i++ {
					cmd = append(cmd, columns[i], values[i])
				}
				commands = append(commands, cmd)
			}
		}
	}

	return &RedisCommand{
		commandType: "REDIS",
		commands:    commands,
		metadata: map[string]interface{}{
			"key_prefix": keyPrefix,
			"batch_size": len(requests),
			"driver":     d.name,
		},
	}, nil
}

func (d *RedisDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{
		batchsql.ConflictIgnore,
		batchsql.ConflictReplace,
		batchsql.ConflictUpdate,
	}
}

func (d *RedisDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if schema.GetIdentifier() == "" {
		return fmt.Errorf("key prefix cannot be empty")
	}
	columns := schema.GetColumns()
	if len(columns) == 0 {
		return fmt.Errorf("columns cannot be empty")
	}
	if len(columns) < 2 {
		return fmt.Errorf("redis driver requires at least 2 columns (key + fields)")
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

// RedisHashDriver Redis Hash 专用驱动
type RedisHashDriver struct {
	RedisDriver
}

func NewRedisHashDriver() *RedisHashDriver {
	return &RedisHashDriver{
		RedisDriver: RedisDriver{name: "redis-hash"},
	}
}

// RedisSetDriver Redis Set 专用驱动
type RedisSetDriver struct {
	RedisDriver
}

func NewRedisSetDriver() *RedisSetDriver {
	return &RedisSetDriver{
		RedisDriver: RedisDriver{name: "redis-set"},
	}
}

func (d *RedisSetDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("empty requests")
	}

	columns := schema.GetColumns()
	if len(columns) != 2 {
		return nil, fmt.Errorf("redis set driver requires exactly 2 columns (set_key, member)")
	}

	commands := make([][]interface{}, 0, len(requests))

	for _, request := range requests {
		values := request.GetOrderedValues()
		if len(values) != 2 {
			return nil, fmt.Errorf("column count mismatch")
		}

		setKey := fmt.Sprintf("%s:%v", schema.GetIdentifier(), values[0])
		member := values[1]

		switch schema.GetConflictStrategy() {
		case batchsql.ConflictIgnore:
			// Redis Set 天然去重，SADD 就是 ignore 模式
			cmd := []interface{}{"SADD", setKey, member}
			commands = append(commands, cmd)
		case batchsql.ConflictReplace, batchsql.ConflictUpdate:
			// 对于 Set，replace 和 update 都等同于 add
			cmd := []interface{}{"SADD", setKey, member}
			commands = append(commands, cmd)
		default:
			cmd := []interface{}{"SADD", setKey, member}
			commands = append(commands, cmd)
		}
	}

	return &RedisCommand{
		commandType: "REDIS",
		commands:    commands,
		metadata: map[string]interface{}{
			"key_prefix": schema.GetIdentifier(),
			"batch_size": len(requests),
			"driver":     d.name,
			"data_type":  "set",
		},
	}, nil
}

func (d *RedisSetDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	if schema.GetIdentifier() == "" {
		return fmt.Errorf("key prefix cannot be empty")
	}
	columns := schema.GetColumns()
	if len(columns) != 2 {
		return fmt.Errorf("redis set driver requires exactly 2 columns (set_key, member)")
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
