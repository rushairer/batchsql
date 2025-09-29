package redis

import (
	"github.com/redis/go-redis/v9"
	"github.com/rushairer/batchsql/drivers"
)

// NewBatchExecutor 创建Redis批量执行器（使用默认驱动）
// 返回 CommonExecutor，内部架构：CommonExecutor -> RedisBatchProcessor -> RedisDriver
// 这是推荐的使用方式，使用Redis优化的默认操作生成器
func NewBatchExecutor(client *redis.Client) drivers.BatchExecutor {
	processor := NewRedisBatchProcessor(client, DefaultDriver)
	return drivers.NewCommonExecutor(processor)
}

// NewBatchExecutorWithDriver 创建Redis批量执行器（使用自定义驱动）
// 返回 CommonExecutor，内部架构：CommonExecutor -> RedisBatchProcessor -> RedisDriver
func NewBatchExecutorWithDriver(client *redis.Client, driver RedisDriver) drivers.BatchExecutor {
	processor := NewRedisBatchProcessor(client, driver)
	return drivers.NewCommonExecutor(processor)
}
