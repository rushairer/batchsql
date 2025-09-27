package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/rushairer/batchsql"
)

// StressTestConfig 压力测试配置
type StressTestConfig struct {
	Batches     int
	BatchSize   int
	Concurrent  int
	Duration    time.Duration
	ShowMetrics bool
}

// TestMetricsReporter 测试用的监控报告器
type TestMetricsReporter struct {
	mu      sync.Mutex
	metrics []batchsql.BatchMetrics
}

func (r *TestMetricsReporter) ReportBatchExecution(ctx context.Context, metrics batchsql.BatchMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = append(r.metrics, metrics)
}

func (r *TestMetricsReporter) GetMetrics() []batchsql.BatchMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]batchsql.BatchMetrics{}, r.metrics...)
}

// MockStressDriver 压力测试用的模拟驱动
type MockStressDriver struct {
	name     string
	delay    time.Duration
	executed int64
	mu       sync.Mutex
}

func NewMockStressDriver(name string, delay time.Duration) *MockStressDriver {
	return &MockStressDriver{
		name:  name,
		delay: delay,
	}
}

func (d *MockStressDriver) GetName() string {
	return d.name
}

func (d *MockStressDriver) GenerateBatchCommand(schema batchsql.SchemaInterface, requests []*batchsql.Request) (batchsql.BatchCommand, error) {
	// 模拟处理延迟
	if d.delay > 0 {
		time.Sleep(d.delay)
	}

	d.mu.Lock()
	d.executed += int64(len(requests))
	d.mu.Unlock()

	return &MockBatchCommand{
		commandType: "INSERT",
		command:     fmt.Sprintf("INSERT INTO %s", schema.GetIdentifier()),
		parameters:  make([]interface{}, len(requests)),
	}, nil
}

func (d *MockStressDriver) SupportedConflictStrategies() []batchsql.ConflictStrategy {
	return []batchsql.ConflictStrategy{batchsql.ConflictIgnore, batchsql.ConflictReplace}
}

func (d *MockStressDriver) ValidateSchema(schema batchsql.SchemaInterface) error {
	return nil
}

func (d *MockStressDriver) GetExecutedCount() int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.executed
}

// MockBatchCommand 模拟批量命令
type MockBatchCommand struct {
	commandType string
	command     interface{}
	parameters  []interface{}
}

func (c *MockBatchCommand) GetCommandType() string {
	return c.commandType
}

func (c *MockBatchCommand) GetCommand() interface{} {
	return c.command
}

func (c *MockBatchCommand) GetParameters() []interface{} {
	return c.parameters
}

func (c *MockBatchCommand) GetMetadata() map[string]interface{} {
	return map[string]interface{}{"test": true}
}

func main() {
	// 解析命令行参数
	config := &StressTestConfig{}
	flag.IntVar(&config.Batches, "batches", 50, "批次数量")
	flag.IntVar(&config.BatchSize, "batch-size", 100, "每批记录数")
	flag.IntVar(&config.Concurrent, "concurrent", 5, "并发数")
	flag.DurationVar(&config.Duration, "duration", 0, "测试持续时间 (0表示基于批次数)")
	flag.BoolVar(&config.ShowMetrics, "metrics", true, "显示详细监控指标")
	flag.Parse()

	fmt.Printf("🚀 BatchSQL 压力测试开始\n")
	fmt.Printf("📊 配置: 批次=%d, 每批大小=%d, 并发=%d\n",
		config.Batches, config.BatchSize, config.Concurrent)

	// 创建监控报告器
	reporter := &TestMetricsReporter{}

	// 创建模拟驱动 (添加小延迟模拟真实数据库操作)
	driver := NewMockStressDriver("stress-test", time.Microsecond*100)

	// 创建客户端
	client := batchsql.NewClient().WithMetricsReporter(reporter)

	// 创建Schema
	schema := batchsql.NewSchema("stress_test_table", batchsql.ConflictIgnore, driver,
		"id", "name", "email", "created_at")

	// 记录开始时间和内存
	startTime := time.Now()
	var startMemStats, endMemStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMemStats)

	// 运行压力测试
	if config.Duration > 0 {
		runDurationBasedTest(client, schema, config)
	} else {
		runBatchBasedTest(client, schema, config)
	}

	// 记录结束时间和内存
	endTime := time.Now()
	runtime.GC()
	runtime.ReadMemStats(&endMemStats)

	// 获取监控指标
	metrics := reporter.GetMetrics()
	executedRecords := driver.GetExecutedCount()

	// 计算统计信息
	totalDuration := endTime.Sub(startTime)
	totalBatches := len(metrics)

	var totalErrors int
	var totalDurationSum time.Duration
	for _, m := range metrics {
		if m.Error != nil {
			totalErrors++
		}
		totalDurationSum += m.Duration
	}

	// 输出结果
	fmt.Printf("\n✅ 压力测试完成\n")
	fmt.Printf("⏱️  总耗时: %v\n", totalDuration)
	fmt.Printf("📦 执行批次: %d\n", totalBatches)
	fmt.Printf("📝 处理记录: %d\n", executedRecords)
	fmt.Printf("❌ 错误数量: %d\n", totalErrors)
	fmt.Printf("📈 成功率: %.2f%%\n", float64(totalBatches-totalErrors)/float64(totalBatches)*100)

	if totalBatches > 0 {
		fmt.Printf("⚡ 平均批次耗时: %v\n", totalDurationSum/time.Duration(totalBatches))
		fmt.Printf("🚀 吞吐量: %.2f 记录/秒\n", float64(executedRecords)/totalDuration.Seconds())
		fmt.Printf("📊 批次吞吐量: %.2f 批次/秒\n", float64(totalBatches)/totalDuration.Seconds())
	}

	// 内存使用情况
	memUsed := endMemStats.Alloc - startMemStats.Alloc
	fmt.Printf("💾 内存使用: %d KB\n", memUsed/1024)
	fmt.Printf("🗑️  GC 次数: %d\n", endMemStats.NumGC-startMemStats.NumGC)

	// 显示详细监控指标
	if config.ShowMetrics && len(metrics) > 0 {
		fmt.Printf("\n📋 详细监控指标:\n")
		fmt.Printf("%-10s %-15s %-10s %-12s %-8s\n", "批次", "表名", "大小", "耗时", "状态")
		fmt.Printf("%-10s %-15s %-10s %-12s %-8s\n", "----", "----", "----", "----", "----")

		for i, m := range metrics {
			status := "✅"
			if m.Error != nil {
				status = "❌"
			}
			fmt.Printf("%-10d %-15s %-10d %-12v %-8s\n",
				i+1, m.Table, m.BatchSize, m.Duration, status)

			// 只显示前10个和后5个，中间的用...表示
			if i == 9 && len(metrics) > 15 {
				fmt.Printf("%-10s %-15s %-10s %-12s %-8s\n", "...", "...", "...", "...", "...")
				_ = len(metrics) - 6
			}
		}
	}

	fmt.Printf("\n🎯 压力测试报告已生成\n")
}

// runBatchBasedTest 基于批次数的测试
func runBatchBasedTest(client *batchsql.Client, schema batchsql.SchemaInterface, config *StressTestConfig) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrent)

	for i := 0; i < config.Batches; i++ {
		wg.Add(1)
		go func(batchNum int) {
			defer wg.Done()

			// 控制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 生成测试数据
			data := generateTestData(config.BatchSize, batchNum)

			// 执行批量操作
			ctx := context.Background()
			err := client.ExecuteWithSchema(ctx, schema, data)
			if err != nil {
				log.Printf("批次 %d 执行失败: %v", batchNum, err)
			}
		}(i)
	}

	wg.Wait()
}

// runDurationBasedTest 基于持续时间的测试
func runDurationBasedTest(client *batchsql.Client, schema batchsql.SchemaInterface, config *StressTestConfig) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrent)
	stopChan := make(chan struct{})

	// 启动定时器
	go func() {
		time.Sleep(config.Duration)
		close(stopChan)
	}()

	batchNum := 0
	for {
		select {
		case <-stopChan:
			wg.Wait()
			return
		default:
			wg.Add(1)
			go func(bn int) {
				defer wg.Done()

				// 控制并发数
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// 生成测试数据
				data := generateTestData(config.BatchSize, bn)

				// 执行批量操作
				ctx := context.Background()
				err := client.ExecuteWithSchema(ctx, schema, data)
				if err != nil {
					log.Printf("批次 %d 执行失败: %v", bn, err)
				}
			}(batchNum)
			batchNum++
		}
	}
}

// generateTestData 生成测试数据
func generateTestData(size int, batchNum int) []map[string]interface{} {
	data := make([]map[string]interface{}, size)
	for i := 0; i < size; i++ {
		data[i] = map[string]interface{}{
			"id":         batchNum*size + i + 1,
			"name":       fmt.Sprintf("user_%d_%d", batchNum, i),
			"email":      fmt.Sprintf("user_%d_%d@test.com", batchNum, i),
			"created_at": time.Now().Format("2006-01-02 15:04:05"),
		}
	}
	return data
}
