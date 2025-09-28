package monitoring

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rushairer/batchsql/drivers"
)

// PrometheusMetrics Prometheus指标收集器，实现MetricsReporter接口
type PrometheusMetrics struct {
	// 批量执行指标
	batchExecutionDuration *prometheus.HistogramVec
	batchExecutionTotal    *prometheus.CounterVec
	batchSize              *prometheus.HistogramVec
	recordsProcessed       *prometheus.CounterVec

	// 测试相关指标
	testDuration      *prometheus.GaugeVec
	testRecordsTotal  *prometheus.GaugeVec
	testRPS           *prometheus.GaugeVec
	testSuccess       *prometheus.GaugeVec
	testDataIntegrity *prometheus.GaugeVec

	// 系统资源指标
	memoryUsage *prometheus.GaugeVec
	gcCount     *prometheus.GaugeVec

	// 错误指标
	errorTotal *prometheus.CounterVec

	registry        *prometheus.Registry
	server          *http.Server
	mu              sync.RWMutex
	currentTestCase string
}

// NewPrometheusMetrics 创建Prometheus指标收集器
func NewPrometheusMetrics() *PrometheusMetrics {
	registry := prometheus.NewRegistry()

	pm := &PrometheusMetrics{
		// 批量执行指标
		batchExecutionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_batch_execution_duration_seconds",
				Help:    "Duration of batch execution in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
			},
			[]string{"database", "table", "status", "test_case"},
		),

		batchExecutionTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "batchsql_batch_execution_total",
				Help: "Total number of batch executions",
			},
			[]string{"database", "table", "status", "test_case"},
		),

		batchSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "batchsql_batch_size",
				Help:    "Size of batches processed",
				Buckets: prometheus.ExponentialBuckets(1, 2, 15), // 1 to ~32k
			},
			[]string{"database", "table", "test_case"},
		),

		recordsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "batchsql_records_processed_total",
				Help: "Total number of records processed",
			},
			[]string{"database", "table", "test_case"},
		),

		// 测试相关指标
		testDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_test_duration_seconds",
				Help: "Duration of test execution in seconds",
			},
			[]string{"database", "test_case"},
		),

		testRecordsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_test_records_total",
				Help: "Total records processed in test",
			},
			[]string{"database", "test_case", "type"}, // type: submitted, actual
		),

		testRPS: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_test_rps",
				Help: "Records per second achieved in test",
			},
			[]string{"database", "test_case"},
		),

		testSuccess: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_test_success",
				Help: "Test success status (1 for success, 0 for failure)",
			},
			[]string{"database", "test_case"},
		),

		testDataIntegrity: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_test_data_integrity_rate",
				Help: "Data integrity rate as percentage (0-100)",
			},
			[]string{"database", "test_case"},
		),

		// 系统资源指标
		memoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_memory_usage_mb",
				Help: "Memory usage in MB",
			},
			[]string{"database", "test_case", "type"}, // type: alloc, total_alloc, sys
		),

		gcCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "batchsql_gc_count",
				Help: "Number of GC runs",
			},
			[]string{"database", "test_case"},
		),

		// 错误指标
		errorTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "batchsql_errors_total",
				Help: "Total number of errors",
			},
			[]string{"database", "test_case", "error_type"},
		),

		registry: registry,
	}

	// 注册所有指标
	registry.MustRegister(
		pm.batchExecutionDuration,
		pm.batchExecutionTotal,
		pm.batchSize,
		pm.recordsProcessed,
		pm.testDuration,
		pm.testRecordsTotal,
		pm.testRPS,
		pm.testSuccess,
		pm.testDataIntegrity,
		pm.memoryUsage,
		pm.gcCount,
		pm.errorTotal,
	)

	return pm
}

// StartServer 启动Prometheus HTTP服务器
func (pm *PrometheusMetrics) StartServer(port int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.server != nil {
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(pm.registry, promhttp.HandlerOpts{}))

	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	pm.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		if err := pm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Prometheus server error: %v\n", err)
		}
	}()

	return nil
}

// StopServer 停止Prometheus HTTP服务器
func (pm *PrometheusMetrics) StopServer() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.server == nil {
		return nil
	}

	err := pm.server.Close()
	pm.server = nil
	return err
}

// RecordBatchExecution 实现MetricsReporter接口
func (pm *PrometheusMetrics) RecordBatchExecution(database string, table string, batchSize int, duration int64, status string) {
	testCase := pm.getCurrentTestCase()

	// 记录执行时长
	pm.batchExecutionDuration.WithLabelValues(database, table, status, testCase).
		Observe(float64(duration) / 1000.0) // 转换为秒

	// 记录执行次数
	pm.batchExecutionTotal.WithLabelValues(database, table, status, testCase).Inc()

	// 记录批次大小
	pm.batchSize.WithLabelValues(database, table, testCase).Observe(float64(batchSize))

	// 记录处理的记录数
	if status == "success" {
		pm.recordsProcessed.WithLabelValues(database, table, testCase).Add(float64(batchSize))
	}
}

// SetCurrentTestCase 设置当前测试用例（用于标签）
func (pm *PrometheusMetrics) SetCurrentTestCase(testCase string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.currentTestCase = testCase
}

// getCurrentTestCase 获取当前测试用例
func (pm *PrometheusMetrics) getCurrentTestCase() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.currentTestCase == "" {
		return "unknown"
	}
	return pm.currentTestCase
}

// RecordTestResult 记录测试结果
func (pm *PrometheusMetrics) RecordTestResult(database, testCase string, result TestResult) {
	// 测试持续时间
	pm.testDuration.WithLabelValues(database, testCase).Set(result.Duration.Seconds())

	// 记录数量
	pm.testRecordsTotal.WithLabelValues(database, testCase, "submitted").Set(float64(result.TotalRecords))
	if result.ActualRecords >= 0 {
		pm.testRecordsTotal.WithLabelValues(database, testCase, "actual").Set(float64(result.ActualRecords))
	}

	// RPS
	if result.RPSValid {
		pm.testRPS.WithLabelValues(database, testCase).Set(result.RecordsPerSecond)
	}

	// 成功状态
	successValue := 0.0
	if result.Success {
		successValue = 1.0
	}
	pm.testSuccess.WithLabelValues(database, testCase).Set(successValue)

	// 数据完整性
	pm.testDataIntegrity.WithLabelValues(database, testCase).Set(result.DataIntegrityRate)

	// 内存使用
	pm.memoryUsage.WithLabelValues(database, testCase, "alloc").Set(result.MemoryUsage.AllocMB)
	pm.memoryUsage.WithLabelValues(database, testCase, "total_alloc").Set(result.MemoryUsage.TotalAllocMB)
	pm.memoryUsage.WithLabelValues(database, testCase, "sys").Set(result.MemoryUsage.SysMB)

	// GC次数
	pm.gcCount.WithLabelValues(database, testCase).Set(float64(result.MemoryUsage.NumGC))

	// 错误计数
	for _, err := range result.Errors {
		errorType := "unknown"
		if len(err) > 50 {
			errorType = err[:50] // 截断长错误消息
		} else {
			errorType = err
		}
		pm.errorTotal.WithLabelValues(database, testCase, errorType).Inc()
	}
}

// RecordRealTimeMetrics 记录实时指标
func (pm *PrometheusMetrics) RecordRealTimeMetrics(database, testCase string, rps float64, memoryMB float64) {
	pm.testRPS.WithLabelValues(database, testCase).Set(rps)
	pm.memoryUsage.WithLabelValues(database, testCase, "current").Set(memoryMB)
}

// TestResult 测试结果结构
type TestResult struct {
	Database            string        `json:"database"`
	TestName            string        `json:"test_name"`
	Duration            time.Duration `json:"duration"`
	TotalRecords        int64         `json:"total_records"`
	ActualRecords       int64         `json:"actual_records"`
	DataIntegrityRate   float64       `json:"data_integrity_rate"`
	DataIntegrityStatus string        `json:"data_integrity_status"`
	RecordsPerSecond    float64       `json:"records_per_second"`
	RPSValid            bool          `json:"rps_valid"`
	RPSNote             string        `json:"rps_note"`
	ConcurrentWorkers   int           `json:"concurrent_workers"`
	MemoryUsage         MemoryStats   `json:"memory_usage"`
	Errors              []string      `json:"errors"`
	Success             bool          `json:"success"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
}

// 确保PrometheusMetrics实现了MetricsReporter接口
var _ drivers.MetricsReporter = (*PrometheusMetrics)(nil)