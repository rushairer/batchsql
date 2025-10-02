package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics Prometheus 指标收集器
type PrometheusMetrics struct {
	// 计数器指标
	totalRecordsProcessed *prometheus.CounterVec
	totalTestsRun         *prometheus.CounterVec
	totalErrors           *prometheus.CounterVec

	// 直方图指标
	testDuration     *prometheus.HistogramVec
	recordsPerSecond *prometheus.HistogramVec
	batchProcessTime *prometheus.HistogramVec

	// 仪表盘指标
	currentRPS        *prometheus.GaugeVec
	memoryUsage       *prometheus.GaugeVec
	dataIntegrityRate *prometheus.GaugeVec
	concurrentWorkers *prometheus.GaugeVec
	activeConnections *prometheus.GaugeVec

	// 新增：与核心库对齐的 Gauge
	executorConcurrency *prometheus.GaugeVec
	queueLength         *prometheus.GaugeVec
	inflightBatches     *prometheus.GaugeVec

	// 新增：与核心库对齐的 Histogram
	enqueueLatency   *prometheus.HistogramVec
	assembleDuration *prometheus.HistogramVec
	executeDuration  *prometheus.HistogramVec
	batchSize        *prometheus.HistogramVec

	// 摘要指标
	responseTime *prometheus.SummaryVec

	registry *prometheus.Registry
	server   *http.Server
	mutex    sync.RWMutex
}

// NewPrometheusMetrics 创建 Prometheus 指标收集器
func NewPrometheusMetrics() *PrometheusMetrics {
	registry := prometheus.NewRegistry()

	pm := &PrometheusMetrics{
		// 计数器指标
		totalRecordsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "batchsql_records_processed_total",
				Help: "Total number of records processed by BatchSQL",
			},
			[]string{"database", "test_name", "status"},
		),

		totalTestsRun: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "batchsql_tests_run_total",
				Help: "Total number of tests run",
			},
			[]string{"database", "test_name", "result"},
		),

		totalErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "batchsql_errors_total",
				Help: "Total number of errors encountered",
			},
			[]string{"database", "test_name", "error_type"},
		),

		// 直方图指标
		testDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_test_duration_seconds",
				Help:    "Duration of BatchSQL tests in seconds",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~100s
			},
			[]string{"database", "test_name"},
		),

		recordsPerSecond: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_records_per_second",
				Help:    "Records processed per second",
				Buckets: prometheus.ExponentialBuckets(1000, 2, 12), // 1K to ~4M RPS
			},
			[]string{"database", "test_name"},
		),

		batchProcessTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_batch_process_duration_seconds",
				Help:    "Time taken to process a batch",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
			},
			[]string{"database", "batch_size"},
		),

		// 仪表盘指标
		currentRPS: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_current_rps",
				Help: "Current records per second",
			},
			[]string{"database", "test_name"},
		),

		memoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_memory_usage_mb",
				Help: "Memory usage in MB",
			},
			[]string{"database", "test_name", "type"}, // type: alloc, total_alloc, sys
		),

		dataIntegrityRate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_data_integrity_rate",
				Help: "Data integrity rate (0-1 range, multiply by 100 for percentage display)",
			},
			[]string{"database", "test_name"},
		),

		concurrentWorkers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_concurrent_workers",
				Help: "Number of concurrent workers",
			},
			[]string{"database", "test_name"},
		),

		activeConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_active_connections",
				Help: "Number of active database connections",
			},
			[]string{"database"},
		),

		// 新增：核心库对齐的 Gauge
		executorConcurrency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_executor_concurrency",
				Help: "Current executor concurrency",
			},
			[]string{"database"},
		),
		queueLength: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_pipeline_queue_length",
				Help: "Current pipeline queue length",
			},
			[]string{"database"},
		),

		inflightBatches: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_inflight_batches",
				Help: "Current in-flight batch count (executing now)",
			},
			[]string{"database"},
		),

		// 新增：核心库对齐的 Histogram
		enqueueLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_enqueue_latency_seconds",
				Help:    "Latency from submit to enqueue",
				Buckets: prometheus.ExponentialBuckets(0.0005, 2, 18),
			},
			[]string{"database"},
		),
		assembleDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_batch_assemble_duration_seconds",
				Help:    "Duration to assemble a batch",
				Buckets: prometheus.ExponentialBuckets(0.0005, 2, 18),
			},
			[]string{"database"},
		),
		executeDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_execute_duration_seconds",
				Help:    "Execute duration for a batch",
				Buckets: prometheus.ExponentialBuckets(0.0005, 2, 18),
			},
			[]string{"database", "test_name"}, // 保守复用现有标签集，若需 table/status 可后续扩展
		),
		batchSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_batch_size",
				Help:    "Batch size distribution",
				Buckets: prometheus.ExponentialBuckets(1, 2, 12),
			},
			[]string{"database"},
		),

		// 摘要指标
		responseTime: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       "batchsql_response_time_seconds",
				Help:       "Response time for batch operations",
				Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
			},
			[]string{"database", "operation"},
		),

		registry: registry,
	}

	// 注册所有指标
	registry.MustRegister(
		pm.totalRecordsProcessed,
		pm.totalTestsRun,
		pm.totalErrors,
		pm.testDuration,
		pm.recordsPerSecond,
		pm.batchProcessTime,
		// 注册新增直方图
		pm.enqueueLatency,
		pm.assembleDuration,
		pm.executeDuration,
		pm.batchSize,
		// 既有与新增 Gauge
		pm.currentRPS,
		pm.memoryUsage,
		pm.dataIntegrityRate,
		pm.concurrentWorkers,
		pm.activeConnections,
		pm.executorConcurrency,
		pm.queueLength,
		pm.inflightBatches,
		pm.responseTime,
	)

	// 初始化基础指标，确保端点始终返回有效数据
	pm.initializeBaseMetrics()

	return pm
}

// StartServer 启动 Prometheus HTTP 服务器
func (pm *PrometheusMetrics) StartServer(port int) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.server != nil {
		return fmt.Errorf("prometheus server already running")
	}

	// 设置 Gin 为发布模式，减少日志输出
	gin.SetMode(gin.ReleaseMode)

	// 创建 Gin 路由器
	router := gin.Default()

	// 添加 Go 运行时指标到我们的自定义 registry
	pm.registry.MustRegister(collectors.NewBuildInfoCollector())
	pm.registry.MustRegister(collectors.NewGoCollector())
	pm.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// 创建使用我们自定义 registry 的 handler
	metricsHandler := promhttp.HandlerFor(pm.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: false,
	})

	// 添加 /metrics 端点
	router.GET("/metrics", gin.WrapH(metricsHandler))

	// 添加健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	pm.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		log.Printf("📊 Prometheus metrics server starting on port %d", port)
		log.Printf("📊 Metrics endpoint: http://localhost:%d/metrics", port)
		if err := pm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Prometheus server error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	return nil
}

// StopServer 停止 Prometheus HTTP 服务器
func (pm *PrometheusMetrics) StopServer() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pm.server.Shutdown(ctx)
	pm.server = nil

	if err == nil {
		log.Println("📊 Prometheus metrics server stopped")
	}

	return err
}

// RecordTestResult 记录测试结果
func (pm *PrometheusMetrics) RecordTestResult(result TestResult) {
	database := result.Database
	testName := result.TestName

	// 记录测试运行
	if result.Success {
		pm.totalTestsRun.WithLabelValues(database, testName, "success").Inc()
	} else {
		pm.totalTestsRun.WithLabelValues(database, testName, "failure").Inc()
	}

	// 记录处理的记录数 (确保值为正数)
	if result.TotalRecords > 0 {
		pm.totalRecordsProcessed.WithLabelValues(database, testName, "processed").Add(float64(result.TotalRecords))
	}
	if result.ActualRecords > 0 {
		pm.totalRecordsProcessed.WithLabelValues(database, testName, "actual").Add(float64(result.ActualRecords))
	}

	// 记录错误
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			pm.totalErrors.WithLabelValues(database, testName, "general").Inc()
			log.Printf("📊 Recorded error for %s/%s: %s", database, testName, err)
		}
	}

	// 记录测试持续时间
	pm.testDuration.WithLabelValues(database, testName).Observe(result.Duration.Seconds())

	// 记录 RPS（如果有效）
	if result.RPSValid && result.RecordsPerSecond > 0 {
		pm.recordsPerSecond.WithLabelValues(database, testName).Observe(result.RecordsPerSecond)
		pm.currentRPS.WithLabelValues(database, testName).Set(result.RecordsPerSecond)
	}

	// 记录内存使用情况
	pm.memoryUsage.WithLabelValues(database, testName, "alloc").Set(result.MemoryUsage.AllocMB)
	pm.memoryUsage.WithLabelValues(database, testName, "total_alloc").Set(result.MemoryUsage.TotalAllocMB)
	pm.memoryUsage.WithLabelValues(database, testName, "sys").Set(result.MemoryUsage.SysMB)

	// 记录数据完整性 (确保值在 0-1 范围内)
	integrityRate := result.DataIntegrityRate / 100.0 // 将百分比转换为 0-1 范围
	pm.dataIntegrityRate.WithLabelValues(database, testName).Set(integrityRate)

	// 记录并发工作线程数
	pm.concurrentWorkers.WithLabelValues(database, testName).Set(float64(result.ConcurrentWorkers))

	log.Printf("📊 Recorded metrics for %s/%s: RPS=%.2f, Integrity=%.2f%%, Duration=%v",
		database, testName, result.RecordsPerSecond, result.DataIntegrityRate, result.Duration)
}

// RecordBatchProcessTime 记录批处理时间
func (pm *PrometheusMetrics) RecordBatchProcessTime(database string, batchSize uint32, duration time.Duration) {
	pm.batchProcessTime.WithLabelValues(database, fmt.Sprintf("%d", batchSize)).Observe(duration.Seconds())
}

// RecordResponseTime 记录响应时间
func (pm *PrometheusMetrics) RecordResponseTime(database, operation string, duration time.Duration) {
	pm.responseTime.WithLabelValues(database, operation).Observe(duration.Seconds())
}

// 新增：与 MetricsReporter 对齐的方法
func (pm *PrometheusMetrics) RecordEnqueueLatency(database string, d time.Duration) {
	pm.enqueueLatency.WithLabelValues(database).Observe(d.Seconds())
}

func (pm *PrometheusMetrics) RecordAssembleDuration(database string, d time.Duration) {
	pm.assembleDuration.WithLabelValues(database).Observe(d.Seconds())
}

func (pm *PrometheusMetrics) RecordExecuteDuration(database, tableOrTest, status string, d time.Duration) {
	// 目前 prometheus.go 中 executeDuration 仅有 database,test_name 两个标签
	// 为不破坏现有集成测试结构，这里将 tableOrTest 作为 test_name 使用；status 暂不入标签
	pm.executeDuration.WithLabelValues(database, tableOrTest).Observe(d.Seconds())
}

func (pm *PrometheusMetrics) RecordBatchSize(database string, n int) {
	pm.batchSize.WithLabelValues(database).Observe(float64(n))
}

func (pm *PrometheusMetrics) SetExecutorConcurrency(database string, n int) {
	pm.executorConcurrency.WithLabelValues(database).Set(float64(n))
}

func (pm *PrometheusMetrics) SetQueueLength(database string, n int) {
	pm.queueLength.WithLabelValues(database).Set(float64(n))
}

func (pm *PrometheusMetrics) IncInflight(database string) {
	pm.inflightBatches.WithLabelValues(database).Inc()
}

func (pm *PrometheusMetrics) DecInflight(database string) {
	pm.inflightBatches.WithLabelValues(database).Dec()
}

// initializeBaseMetrics 初始化基础指标，确保端点始终返回有效数据
func (pm *PrometheusMetrics) initializeBaseMetrics() {
	// 初始化计数器指标为 0
	databases := []string{"mysql", "postgres", "sqlite", "redis"}
	testTypes := []string{"batch_insert", "concurrent_workers", "large_batch", "stress_test"}

	for _, db := range databases {
		for _, testType := range testTypes {
			// 初始化计数器 - 3个标签: database, test_name, status/result/error_type
			// 使用与 RecordTestResult 中相同的标签值
			pm.totalRecordsProcessed.WithLabelValues(db, testType, "processed").Add(0)
			pm.totalRecordsProcessed.WithLabelValues(db, testType, "actual").Add(0)
			pm.totalTestsRun.WithLabelValues(db, testType, "success").Add(0)
			pm.totalTestsRun.WithLabelValues(db, testType, "failure").Add(0)
			pm.totalErrors.WithLabelValues(db, testType, "general").Add(0)

			// 初始化仪表盘指标 - 标签数量要匹配定义
			// currentRPS: 2个标签 (database, test_name)
			pm.currentRPS.WithLabelValues(db, testType).Set(0)

			// memoryUsage: 3个标签 (database, test_name, type)
			pm.memoryUsage.WithLabelValues(db, testType, "alloc").Set(0)
			pm.memoryUsage.WithLabelValues(db, testType, "total_alloc").Set(0)
			pm.memoryUsage.WithLabelValues(db, testType, "sys").Set(0)

			// dataIntegrityRate: 2个标签 (database, test_name) - 范围 0-1
			pm.dataIntegrityRate.WithLabelValues(db, testType).Set(1.0)

			// concurrentWorkers: 2个标签 (database, test_name)
			pm.concurrentWorkers.WithLabelValues(db, testType).Set(0)
		}

		// activeConnections: 1个标签 (database)
		pm.activeConnections.WithLabelValues(db).Set(0)
	}
}

// UpdateActiveConnections 更新活跃连接数
func (pm *PrometheusMetrics) UpdateActiveConnections(database string, count int) {
	pm.activeConnections.WithLabelValues(database).Set(float64(count))
}

// UpdateCurrentRPS 更新当前 RPS
func (pm *PrometheusMetrics) UpdateCurrentRPS(database, testName string, rps float64) {
	pm.currentRPS.WithLabelValues(database, testName).Set(rps)
}

// UpdateMemoryUsage 更新内存使用情况
func (pm *PrometheusMetrics) UpdateMemoryUsage(database, testName string, allocMB, totalAllocMB, sysMB float64) {
	pm.memoryUsage.WithLabelValues(database, testName, "alloc").Set(allocMB)
	pm.memoryUsage.WithLabelValues(database, testName, "total_alloc").Set(totalAllocMB)
	pm.memoryUsage.WithLabelValues(database, testName, "sys").Set(sysMB)
}

// GetMetricsURL 获取指标 URL
func (pm *PrometheusMetrics) GetMetricsURL(port int) string {
	return fmt.Sprintf("http://localhost:%d/metrics", port)
}
