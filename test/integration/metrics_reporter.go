package main

import (
	"time"

	"github.com/rushairer/batchsql/drivers"
)

// PrometheusMetricsReporter 实现 MetricsReporter 接口，将批处理指标报告给 Prometheus
type PrometheusMetricsReporter struct {
	prometheusMetrics *PrometheusMetrics
	database          string
	testName          string
}

// NewPrometheusMetricsReporter 创建 Prometheus 指标报告器
func NewPrometheusMetricsReporter(prometheusMetrics *PrometheusMetrics, database, testName string) drivers.MetricsReporter {
	return &PrometheusMetricsReporter{
		prometheusMetrics: prometheusMetrics,
		database:          database,
		testName:          testName,
	}
}

// RecordBatchExecution 记录批处理执行指标
// 这个方法会在每次批处理执行时被 CommonExecutor 调用
func (r *PrometheusMetricsReporter) RecordBatchExecution(tableName string, batchSize int, duration int64, status string) {
	if r.prometheusMetrics == nil {
		return
	}

	// 记录批处理时间（转换毫秒到秒）
	durationSeconds := time.Duration(duration) * time.Millisecond
	r.prometheusMetrics.RecordBatchProcessTime(r.database, uint32(batchSize), durationSeconds)

	// 记录响应时间
	r.prometheusMetrics.RecordResponseTime(r.database, "batch_insert", durationSeconds)

	// 记录处理的记录数
	if status == "success" {
		r.prometheusMetrics.totalRecordsProcessed.WithLabelValues(r.database, r.testName, "success").Add(float64(batchSize))

		// 计算并更新当前 RPS（基于批次大小和持续时间）
		if durationSeconds.Seconds() > 0 {
			currentRPS := float64(batchSize) / durationSeconds.Seconds()
			r.prometheusMetrics.currentRPS.WithLabelValues(r.database, r.testName).Set(currentRPS)
		}

		// 假设数据完整性为 100%（成功的批次）
		r.prometheusMetrics.dataIntegrityRate.WithLabelValues(r.database, r.testName).Set(1.0)
	} else {
		r.prometheusMetrics.totalErrors.WithLabelValues(r.database, r.testName, "batch_execution").Inc()

		// 失败时设置数据完整性为 0
		r.prometheusMetrics.dataIntegrityRate.WithLabelValues(r.database, r.testName).Set(0.0)
	}

	// 记录批处理操作计数
	r.prometheusMetrics.totalTestsRun.WithLabelValues(r.database, r.testName, status).Inc()

	// 记录测试持续时间到直方图
	r.prometheusMetrics.testDuration.WithLabelValues(r.database, r.testName).Observe(durationSeconds.Seconds())

	// 记录每秒记录数到直方图
	if durationSeconds.Seconds() > 0 {
		rps := float64(batchSize) / durationSeconds.Seconds()
		r.prometheusMetrics.recordsPerSecond.WithLabelValues(r.database, r.testName).Observe(rps)
	}
}
