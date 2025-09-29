package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers"
)

// runRedisTests 运行 Redis 数据库测试
func runRedisTests(dsn string, config TestConfig) []TestResult {
	var results []TestResult

	// 解析 Redis DSN
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		log.Printf("❌ 解析 Redis DSN 失败：%v", err)
		return results
	}

	// 连接 Redis
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	// 测试连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("❌ Ping Redis 失败：%v", err)
		return results
	}

	// 运行不同的测试场景
	testCases := []struct {
		name     string
		testFunc func(*redis.Client, TestConfig) TestResult
	}{
		{"高吞吐量测试", runRedisHighThroughputTest},
		{"并发工作线程测试", runRedisConcurrentWorkersTest},
		{"大批次测试", runRedisLargeBatchTest},
		{"内存压力测试", runRedisMemoryPressureTest},
		{"长时间运行测试", runRedisLongDurationTest},
	}

	for _, tc := range testCases {
		// 每个测试前清理 Redis 数据
		log.Printf("  🧹 在运行 %s 前清理 Redis 数据...", tc.name)
		if err := rdb.FlushDB(ctx).Err(); err != nil {
			log.Printf("❌ Failed to flush Redis DB before %s: %v", tc.name, err)
		}

		log.Printf("  🔄 在 Redis 上运行 %s...", tc.name)
		result := tc.testFunc(rdb, config)
		result.TestName = tc.name
		result.Database = "redis"
		results = append(results, result)

		// 测试间隔，让系统恢复
		time.Sleep(5 * time.Second)
	}

	return results
}

// runRedisHighThroughputTest Redis 高吞吐量测试
func runRedisHighThroughputTest(rdb *redis.Client, config TestConfig) TestResult {
	ctx := context.Background()

	batchSQL := batchsql.NewRedisBatchSQL(ctx, rdb, batchsql.PipelineConfig{
		BufferSize:    config.BufferSize,
		FlushSize:     config.BatchSize,
		FlushInterval: config.FlushInterval,
	})

	// Redis 使用简单的 key-value schema
	schema := batchsql.NewSchema("redis_test", drivers.ConflictIgnore,
		"key", "value", "ttl")

	startTime := time.Now()
	var recordCount int64
	var errors []string

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 高吞吐量测试
	testCtx, cancel := context.WithTimeout(ctx, config.TestDuration)
	defer cancel()

	maxRecords := int64(config.ConcurrentWorkers * config.RecordsPerWorker)

	for i := int64(0); i < maxRecords; i++ {
		select {
		case <-testCtx.Done():
			goto TestComplete
		default:
			request := batchsql.NewRequest(schema).
				SetString("key", fmt.Sprintf("test:user:%d", i)).
				SetString("value", fmt.Sprintf(`{"id":%d,"name":"User_%d","email":"user_%d@example.com","active":%t}`, i, i, i, i%2 == 0)).
				SetInt64("ttl", 3600000) // 1小时 TTL (毫秒)

			if err := batchSQL.Submit(testCtx, request); err != nil {
				errors = append(errors, err.Error())
				if len(errors) > 100 {
					break
				}
			} else {
				recordCount++
			}

			// 定期强制GC
			if i%1000 == 0 {
				runtime.GC()
			}
		}
	}

TestComplete:
	duration := time.Since(startTime)

	// 等待处理完成
	time.Sleep(5 * time.Second)

	// 记录最终内存
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 查询 Redis 中的实际记录数
	actualRecords, countErr := getRedisRecordCount(rdb, ctx)
	if countErr != nil {
		errors = append(errors, fmt.Sprintf("统计实际记录数失败：%v", countErr))
		actualRecords = -1
	}

	// 计算数据完整性
	dataIntegrityRate, integrityStatus, rpsValid, rpsNote := calculateDataIntegrity(recordCount, actualRecords)

	// 只有在数据完整性100%时才计算有效的RPS
	rps := 0.0
	if rpsValid && duration.Seconds() > 0 {
		rps = float64(recordCount) / duration.Seconds()
	}

	return TestResult{
		Duration:            duration,
		TotalRecords:        recordCount,
		ActualRecords:       actualRecords,
		DataIntegrityRate:   dataIntegrityRate,
		DataIntegrityStatus: integrityStatus,
		RecordsPerSecond:    rps,
		RPSValid:            rpsValid,
		RPSNote:             rpsNote,
		ConcurrentWorkers:   1,
		TestParameters: TestParams{
			BatchSize:       config.BatchSize,
			BufferSize:      config.BufferSize,
			FlushInterval:   config.FlushInterval,
			ExpectedRecords: int64(config.ConcurrentWorkers * config.RecordsPerWorker),
			TestDuration:    config.TestDuration,
		},
		MemoryUsage: MemoryStats{
			AllocMB:      calculateMemoryDiffMB(m2.Alloc, m1.Alloc),
			TotalAllocMB: calculateMemoryDiffMB(m2.TotalAlloc, m1.TotalAlloc),
			SysMB:        calculateMemoryDiffMB(m2.Sys, m1.Sys),
			NumGC:        m2.NumGC - m1.NumGC,
		},
		Errors:  errors,
		Success: len(errors) == 0 && rpsValid,
	}
}

// runRedisConcurrentWorkersTest Redis 并发工作线程测试
func runRedisConcurrentWorkersTest(rdb *redis.Client, config TestConfig) TestResult {
	ctx := context.Background()

	batchSQL := batchsql.NewRedisBatchSQL(ctx, rdb, batchsql.PipelineConfig{
		BufferSize:    config.BufferSize,
		FlushSize:     config.BatchSize,
		FlushInterval: config.FlushInterval,
	})

	schema := batchsql.NewSchema("redis_test", drivers.ConflictIgnore,
		"cmd", "key", "value", "ttl")

	startTime := time.Now()
	var totalRecords int64
	var mu sync.Mutex
	var errors []string

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 并发工作者测试
	var wg sync.WaitGroup
	batchSize := 100

	for workerID := 0; workerID < config.ConcurrentWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			workerRecords := 0
			baseID := int64(id * config.RecordsPerWorker)

			for batch := 0; batch < config.RecordsPerWorker; batch += batchSize {
				endIdx := batch + batchSize
				if endIdx > config.RecordsPerWorker {
					endIdx = config.RecordsPerWorker
				}

				for i := batch; i < endIdx; i++ {
					request := batchsql.NewRequest(schema).
						SetString("cmd", "set").
						SetString("key", fmt.Sprintf("test:worker:%d:user:%d", id, baseID+int64(i))).
						SetString("value", fmt.Sprintf(`{"worker_id":%d,"user_id":%d,"name":"W%d_U%d","active":%t}`, id, baseID+int64(i), id, i, (id+i)%2 == 0)).
						SetInt64("ttl", 3600000) // 1小时 TTL

					if err := batchSQL.Submit(ctx, request); err != nil {
						mu.Lock()
						errors = append(errors, fmt.Sprintf("Worker %d: %v", id, err))
						mu.Unlock()
					} else {
						workerRecords++
					}
				}

				runtime.GC()
				time.Sleep(10 * time.Millisecond)
			}

			mu.Lock()
			totalRecords += int64(workerRecords)
			mu.Unlock()
		}(workerID)
	}

	wg.Wait()
	duration := time.Since(startTime)

	// 等待处理完成
	time.Sleep(5 * time.Second)

	// 记录最终内存
	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 查询 Redis 中的实际记录数
	actualRecords, countErr := getRedisRecordCount(rdb, ctx)
	if countErr != nil {
		mu.Lock()
		errors = append(errors, fmt.Sprintf("统计实际记录数失败：%v", countErr))
		mu.Unlock()
		actualRecords = -1
	}

	// 计算数据完整性
	dataIntegrityRate, integrityStatus, rpsValid, rpsNote := calculateDataIntegrity(totalRecords, actualRecords)

	// 只有在数据完整性100%时才计算有效的RPS
	rps := 0.0
	if rpsValid && duration.Seconds() > 0 {
		rps = float64(totalRecords) / duration.Seconds()
	}

	return TestResult{
		Duration:            duration,
		TotalRecords:        totalRecords,
		ActualRecords:       actualRecords,
		DataIntegrityRate:   dataIntegrityRate,
		DataIntegrityStatus: integrityStatus,
		RecordsPerSecond:    rps,
		RPSValid:            rpsValid,
		RPSNote:             rpsNote,
		ConcurrentWorkers:   config.ConcurrentWorkers,
		TestParameters: TestParams{
			BatchSize:       config.BatchSize,
			BufferSize:      config.BufferSize,
			FlushInterval:   config.FlushInterval,
			ExpectedRecords: int64(config.ConcurrentWorkers * config.RecordsPerWorker),
			TestDuration:    config.TestDuration,
		},
		MemoryUsage: MemoryStats{
			AllocMB:      calculateMemoryDiffMB(m2.Alloc, m1.Alloc),
			TotalAllocMB: calculateMemoryDiffMB(m2.TotalAlloc, m1.TotalAlloc),
			SysMB:        calculateMemoryDiffMB(m2.Sys, m1.Sys),
			NumGC:        m2.NumGC - m1.NumGC,
		},
		Errors:  errors,
		Success: len(errors) == 0 && rpsValid,
	}
}

// runRedisLargeBatchTest Redis 大批次测试
func runRedisLargeBatchTest(rdb *redis.Client, config TestConfig) TestResult {
	largeConfig := config
	largeConfig.BatchSize = 5000
	largeConfig.BufferSize = 50000

	result := runRedisHighThroughputTest(rdb, largeConfig)
	result.TestName = "Large Batch Test"
	return result
}

// runRedisMemoryPressureTest Redis 内存压力测试
func runRedisMemoryPressureTest(rdb *redis.Client, config TestConfig) TestResult {
	memConfig := config
	memConfig.BatchSize = 100
	memConfig.BufferSize = 1000
	memConfig.RecordsPerWorker = 50000

	result := runRedisConcurrentWorkersTest(rdb, memConfig)
	result.TestName = "Memory Pressure Test"
	return result
}

// runRedisLongDurationTest Redis 长时间运行测试
func runRedisLongDurationTest(rdb *redis.Client, config TestConfig) TestResult {
	longConfig := config
	longConfig.TestDuration = 10 * time.Minute

	result := runRedisHighThroughputTest(rdb, longConfig)
	result.TestName = "Long Duration Test"
	return result
}

// getRedisRecordCount 获取 Redis 中的记录数量
func getRedisRecordCount(rdb *redis.Client, ctx context.Context) (int64, error) {
	// 使用 DBSIZE 命令获取当前数据库中的 key 数量
	count, err := rdb.DBSize(ctx).Result()
	return count, err
}
