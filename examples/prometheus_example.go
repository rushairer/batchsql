// Package main demonstrates Prometheus integration with BatchSQL
// This is a separate example that requires prometheus dependencies
// Run: go mod tidy && go run examples/prometheus_example.go
package main

import (
	"context"
	"log"
	"time"

	// Note: These imports require adding prometheus dependencies to go.mod
	// "github.com/prometheus/client_golang/prometheus"
	// "github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/rushairer/batchsql"
	"github.com/rushairer/batchsql/drivers"
)

// MockPrometheusHistogramVec 模拟Prometheus HistogramVec
type MockPrometheusHistogramVec struct{}

func (m *MockPrometheusHistogramVec) WithLabelValues(labels ...string) MockPrometheusHistogram {
	return MockPrometheusHistogram{}
}

type MockPrometheusHistogram struct{}

func (m MockPrometheusHistogram) Observe(value float64) {
	log.Printf("Prometheus Histogram - Labels: %v, Value: %f", "mock", value)
}

// MockPrometheusCounterVec 模拟Prometheus CounterVec
type MockPrometheusCounterVec struct{}

func (m *MockPrometheusCounterVec) WithLabelValues(labels ...string) MockPrometheusCounter {
	return MockPrometheusCounter{}
}

type MockPrometheusCounter struct{}

func (m MockPrometheusCounter) Inc() {
	log.Printf("Prometheus Counter - Increment")
}

// PrometheusReporter Prometheus监控报告器（使用模拟实现）
type PrometheusReporter struct {
	duration *MockPrometheusHistogramVec
	total    *MockPrometheusCounterVec
	errors   *MockPrometheusCounterVec
}

// NewPrometheusReporter 创建Prometheus监控报告器
func NewPrometheusReporter() *PrometheusReporter {
	log.Println("创建Prometheus监控报告器（模拟实现）")
	return &PrometheusReporter{
		duration: &MockPrometheusHistogramVec{},
		total:    &MockPrometheusCounterVec{},
		errors:   &MockPrometheusCounterVec{},
	}
}

// ReportBatchExecution 实现MetricsReporter接口
func (p *PrometheusReporter) ReportBatchExecution(ctx context.Context, metrics batchsql.BatchMetrics) {
	// 记录执行时长
	p.duration.WithLabelValues(metrics.Driver, metrics.Table).Observe(metrics.Duration.Seconds())

	// 记录执行状态
	status := "success"
	if metrics.Error != nil {
		status = "error"
		// 记录错误类型
		errorType := "unknown"
		if metrics.Error != nil {
			errorType = "execution_error"
		}
		p.errors.WithLabelValues(metrics.Driver, metrics.Table, errorType).Inc()
	}

	p.total.WithLabelValues(metrics.Driver, metrics.Table, status).Inc()
}

func prometheusMain() {
	log.Println("=== BatchSQL Prometheus集成示例 ===")

	// 创建Prometheus监控报告器
	prometheusReporter := NewPrometheusReporter()

	// 创建带监控的BatchSQL客户端
	client := batchsql.NewClient().WithMetricsReporter(prometheusReporter)

	// 创建不同的数据库驱动和Schema
	mysqlDriver := drivers.NewMySQLDriver()
	redisDriver := drivers.NewRedisDriver()

	userSchema := client.CreateSchema("users", batchsql.ConflictUpdate, mysqlDriver, "id", "name", "email")
	sessionSchema := client.CreateSchema("sessions", batchsql.ConflictReplace, redisDriver, "user_id", "token")

	ctx := context.Background()

	// 执行多个批量操作
	operations := []struct {
		name   string
		schema batchsql.SchemaInterface
		data   []map[string]interface{}
	}{
		{
			name:   "用户数据",
			schema: userSchema,
			data: []map[string]interface{}{
				{"id": 1, "name": "Alice", "email": "alice@example.com"},
				{"id": 2, "name": "Bob", "email": "bob@example.com"},
			},
		},
		{
			name:   "会话数据",
			schema: sessionSchema,
			data: []map[string]interface{}{
				{"user_id": "user_1", "token": "token_abc123"},
				{"user_id": "user_2", "token": "token_def456"},
			},
		},
	}

	for _, op := range operations {
		log.Printf("\n执行 %s 批量操作...", op.name)

		start := time.Now()
		err := client.ExecuteWithSchema(ctx, op.schema, op.data)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ %s 执行失败: %v", op.name, err)
		} else {
			log.Printf("✅ %s 执行成功，耗时: %v", op.name, duration)
		}
	}

	log.Println("\n=== Prometheus指标已记录 ===")
	log.Println("可以通过以下指标查看监控数据:")
	log.Println("- batchsql_execution_duration_seconds: 执行时长")
	log.Println("- batchsql_executions_total: 执行总数")
	log.Println("- batchsql_errors_total: 错误总数")

	log.Println("\n示例Prometheus查询:")
	log.Println("- rate(batchsql_executions_total[5m]): 每秒执行率")
	log.Println("- histogram_quantile(0.95, batchsql_execution_duration_seconds): 95%分位数延迟")
	log.Println("- batchsql_errors_total / batchsql_executions_total: 错误率")
}
