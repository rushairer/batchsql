package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/rushairer/batchsql/drivers"
)

// Executor Redis批量执行器
// 直接实现 BatchExecutor 接口，无需 BatchProcessor 层
// 架构：BatchExecutor -> Redis Client（跳过BatchProcessor层）
//
// 设计说明：
// - Redis作为NoSQL数据库，有自己的数据模型和操作方式
// - 直接实现BatchExecutor避免了不必要的抽象层
// - 使用Redis Pipeline特性实现高效的批量操作
type Executor struct {
	db              *redis.Client           // Redis客户端连接
	metricsReporter drivers.MetricsReporter // 性能指标报告器
}

// NewBatchExecutor 创建Redis批量执行器
// 返回直接实现BatchExecutor接口的执行器，无需额外的处理层
func NewBatchExecutor(db *redis.Client) *Executor {
	return &Executor{
		db: db,
	}
}

// ExecuteBatch 执行批量操作
func (e *Executor) ExecuteBatch(ctx context.Context, _ *drivers.Schema, data []map[string]any) error {
	log.Println("ExecuteBatch")
	if len(data) == 0 {
		return nil
	}

	// 使用Redis的Pipeline特性实现批量插入
	pipeline := e.db.Pipeline()
	for _, row := range data {
		log.Println(row)
		cmds := []string{}
		for _, v := range row {
			cmds = append(cmds, v.(string))
		}
		cmd := pipeline.Do(ctx, cmds)
		log.Println(cmd)
	}
	cmds, err := pipeline.Exec(ctx)
	log.Println(cmds)
	return err
}

// WithMetricsReporter 设置指标报告器
func (e *Executor) WithMetricsReporter(metricsReporter drivers.MetricsReporter) drivers.BatchExecutor {
	e.metricsReporter = metricsReporter
	return e
}
