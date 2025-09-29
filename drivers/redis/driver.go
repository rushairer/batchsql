package redis

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"github.com/rushairer/batchsql/drivers"
)

type RedisCmd []any

type RedisBatchCmd []RedisCmd

// RedisDriver Redis数据库操作生成器
// 与SQL驱动不同，Redis驱动生成的是Redis操作而不是SQL语句
type RedisDriver interface {
	GenerateOperations(schema *drivers.Schema, data []map[string]any) (RedisBatchCmd, error)
	ExecuteOperations(ctx context.Context, client *redis.Client, batchCmd RedisBatchCmd) error
}

// Driver Redis驱动实现
type Driver struct{}

// NewDriver 创建Redis驱动
func NewDriver() *Driver {
	return &Driver{}
}

// GenerateOperations 根据schema和data生成Redis操作
func (d *Driver) GenerateOperations(schema *drivers.Schema, data []map[string]any) (RedisBatchCmd, error) {
	if len(data) == 0 {
		return nil, nil
	}

	columns := schema.Columns

	if len(columns) < 2 {
		return nil, errors.New("redis schema must have at least 2 columns: cmd and key")
	}

	// 构建参数数组
	batchCmd := make(RedisBatchCmd, len(data))
	for i, row := range data {
		batchCmd[i] = make(RedisCmd, len(columns))
		for j, col := range columns {
			batchCmd[i][j] = row[col]
		}
	}
	return batchCmd, nil
}

// ExecuteOperations 执行Redis操作
func (d *Driver) ExecuteOperations(ctx context.Context, client *redis.Client, batchCmd RedisBatchCmd) (err error) {
	if len(batchCmd) == 0 {
		return nil
	}

	// 使用Pipeline批量执行
	pipeline := client.Pipeline()

	for _, cmd := range batchCmd {
		pipeline.Do(ctx, cmd...)
	}

	// 执行Pipeline
	cmds, err := pipeline.Exec(ctx)
	if err != nil {
		return err
	}

	// 检查每个命令的执行结果
	for _, cmd := range cmds {
		if cmd.Err() != nil {
			err = errors.Join(err, cmd.Err())
		}
	}

	return err
}

// DefaultDriver 全局默认Redis驱动实例
var DefaultDriver = &Driver{}
