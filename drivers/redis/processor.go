package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rushairer/batchsql/drivers"
)

// RedisBatchProcessor Redis批量处理器
// 实现 BatchProcessor 接口，专注于Redis的核心处理逻辑
// 架构位置：CommonExecutor -> RedisBatchProcessor -> RedisDriver -> Redis Client
//
// 职责：
// - 调用RedisDriver生成Redis操作
// - 执行批量Redis操作
// - 处理Redis连接和Pipeline
type RedisBatchProcessor struct {
	client      *redis.Client // Redis客户端连接
	redisDriver RedisDriver   // Redis操作生成器
}

// NewRedisBatchProcessor 创建Redis批量处理器
// 参数：
// - client: Redis客户端连接
// - redisDriver: Redis操作生成器
func NewRedisBatchProcessor(client *redis.Client, redisDriver RedisDriver) *RedisBatchProcessor {
	return &RedisBatchProcessor{
		client:      client,
		redisDriver: redisDriver,
	}
}

// ExecuteBatch 执行批量操作
func (rp *RedisBatchProcessor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	// 使用RedisDriver生成Redis操作
	operations, err := rp.redisDriver.GenerateOperations(schema, data)
	if err != nil {
		return fmt.Errorf("生成Redis操作失败: %w", err)
	}

	// 执行Redis批量操作
	err = rp.redisDriver.ExecuteOperations(ctx, rp.client, operations)
	if err != nil {
		return err
	}

	return nil
}
