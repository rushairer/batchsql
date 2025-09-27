package batchsql

import (
	"sync"
	"time"
)

// BatchExecutionMetrics 批量执行指标
type BatchExecutionMetrics struct {
	DriverName   string        `json:"driver_name"`
	BatchSize    int           `json:"batch_size"`
	Duration     time.Duration `json:"duration"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
}

// DriverMetrics 驱动指标
type DriverMetrics struct {
	DriverName      string        `json:"driver_name"`
	TotalExecutions int64         `json:"total_executions"`
	SuccessfulExecs int64         `json:"successful_executions"`
	FailedExecs     int64         `json:"failed_executions"`
	TotalRequests   int64         `json:"total_requests"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	LastExecution   time.Time     `json:"last_execution"`
	LastError       string        `json:"last_error,omitempty"`
}

// DefaultMetricsCollector 默认指标收集器实现
type DefaultMetricsCollector struct {
	driverMetrics    map[string]*DriverMetrics
	executionHistory []BatchExecutionMetrics
	maxHistorySize   int
	mutex            sync.RWMutex
	startTime        time.Time
}

// NewDefaultMetricsCollector 创建默认指标收集器
func NewDefaultMetricsCollector() *DefaultMetricsCollector {
	return &DefaultMetricsCollector{
		driverMetrics:    make(map[string]*DriverMetrics),
		executionHistory: make([]BatchExecutionMetrics, 0),
		maxHistorySize:   1000, // 保留最近1000次执行记录
		startTime:        time.Now(),
	}
}

// SetMaxHistorySize 设置最大历史记录数量
func (mc *DefaultMetricsCollector) SetMaxHistorySize(size int) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.maxHistorySize = size
}

// RecordBatchExecution 记录批量执行
func (mc *DefaultMetricsCollector) RecordBatchExecution(driverName string, batchSize int, duration int64, err error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	now := time.Now()
	durationTime := time.Duration(duration) * time.Millisecond

	// 更新驱动指标
	if mc.driverMetrics[driverName] == nil {
		mc.driverMetrics[driverName] = &DriverMetrics{
			DriverName: driverName,
		}
	}

	metrics := mc.driverMetrics[driverName]
	metrics.TotalExecutions++
	metrics.TotalRequests += int64(batchSize)
	metrics.TotalDuration += durationTime
	metrics.LastExecution = now

	if err != nil {
		metrics.FailedExecs++
		metrics.LastError = err.Error()
	} else {
		metrics.SuccessfulExecs++
	}

	// 计算平均执行时间
	if metrics.TotalExecutions > 0 {
		metrics.AverageDuration = time.Duration(int64(metrics.TotalDuration) / metrics.TotalExecutions)
	}

	// 记录执行历史
	execution := BatchExecutionMetrics{
		DriverName: driverName,
		BatchSize:  batchSize,
		Duration:   durationTime,
		Success:    err == nil,
		Timestamp:  now,
	}

	if err != nil {
		execution.ErrorMessage = err.Error()
	}

	mc.executionHistory = append(mc.executionHistory, execution)

	// 限制历史记录大小
	if len(mc.executionHistory) > mc.maxHistorySize {
		// 删除最旧的记录
		copy(mc.executionHistory, mc.executionHistory[1:])
		mc.executionHistory = mc.executionHistory[:mc.maxHistorySize]
	}
}

// RecordRequestCount 记录请求数量
func (mc *DefaultMetricsCollector) RecordRequestCount(driverName string, count int) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.driverMetrics[driverName] == nil {
		mc.driverMetrics[driverName] = &DriverMetrics{
			DriverName: driverName,
		}
	}

	mc.driverMetrics[driverName].TotalRequests += int64(count)
}

// GetMetrics 获取指标
func (mc *DefaultMetricsCollector) GetMetrics() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// 复制驱动指标以避免并发问题
	driverMetricsCopy := make(map[string]*DriverMetrics)
	for name, metrics := range mc.driverMetrics {
		metricsCopy := *metrics
		driverMetricsCopy[name] = &metricsCopy
	}

	// 复制执行历史
	historyCopy := make([]BatchExecutionMetrics, len(mc.executionHistory))
	copy(historyCopy, mc.executionHistory)

	// 计算总体统计
	totalExecutions := int64(0)
	totalRequests := int64(0)
	totalDuration := time.Duration(0)
	successfulExecs := int64(0)
	failedExecs := int64(0)

	for _, metrics := range driverMetricsCopy {
		totalExecutions += metrics.TotalExecutions
		totalRequests += metrics.TotalRequests
		totalDuration += metrics.TotalDuration
		successfulExecs += metrics.SuccessfulExecs
		failedExecs += metrics.FailedExecs
	}

	var averageDuration time.Duration
	if totalExecutions > 0 {
		averageDuration = time.Duration(int64(totalDuration) / totalExecutions)
	}

	var successRate float64
	if totalExecutions > 0 {
		successRate = float64(successfulExecs) / float64(totalExecutions) * 100
	}

	return map[string]interface{}{
		"start_time":        mc.startTime,
		"uptime":            time.Since(mc.startTime),
		"total_executions":  totalExecutions,
		"successful_execs":  successfulExecs,
		"failed_execs":      failedExecs,
		"success_rate":      successRate,
		"total_requests":    totalRequests,
		"total_duration":    totalDuration,
		"average_duration":  averageDuration,
		"driver_metrics":    driverMetricsCopy,
		"execution_history": historyCopy,
		"history_size":      len(historyCopy),
		"max_history_size":  mc.maxHistorySize,
	}
}

// GetDriverMetrics 获取特定驱动的指标
func (mc *DefaultMetricsCollector) GetDriverMetrics(driverName string) *DriverMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if metrics, exists := mc.driverMetrics[driverName]; exists {
		// 返回副本以避免并发问题
		metricsCopy := *metrics
		return &metricsCopy
	}

	return nil
}

// GetRecentExecutions 获取最近的执行记录
func (mc *DefaultMetricsCollector) GetRecentExecutions(limit int) []BatchExecutionMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if limit <= 0 || limit > len(mc.executionHistory) {
		limit = len(mc.executionHistory)
	}

	// 返回最近的记录（从末尾开始）
	start := len(mc.executionHistory) - limit
	recent := make([]BatchExecutionMetrics, limit)
	copy(recent, mc.executionHistory[start:])

	return recent
}

// GetExecutionsByDriver 获取特定驱动的执行记录
func (mc *DefaultMetricsCollector) GetExecutionsByDriver(driverName string, limit int) []BatchExecutionMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	var driverExecutions []BatchExecutionMetrics
	for _, execution := range mc.executionHistory {
		if execution.DriverName == driverName {
			driverExecutions = append(driverExecutions, execution)
		}
	}

	if limit > 0 && limit < len(driverExecutions) {
		// 返回最近的记录
		start := len(driverExecutions) - limit
		return driverExecutions[start:]
	}

	return driverExecutions
}

// Reset 重置所有指标
func (mc *DefaultMetricsCollector) Reset() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.driverMetrics = make(map[string]*DriverMetrics)
	mc.executionHistory = make([]BatchExecutionMetrics, 0)
	mc.startTime = time.Now()
}

// GetSummary 获取指标摘要
func (mc *DefaultMetricsCollector) GetSummary() map[string]interface{} {
	metrics := mc.GetMetrics()

	summary := map[string]interface{}{
		"uptime":           metrics["uptime"],
		"total_executions": metrics["total_executions"],
		"success_rate":     metrics["success_rate"],
		"total_requests":   metrics["total_requests"],
		"average_duration": metrics["average_duration"],
	}

	// 添加每个驱动的简要信息
	driverSummary := make(map[string]interface{})
	if driverMetrics, ok := metrics["driver_metrics"].(map[string]*DriverMetrics); ok {
		for name, dm := range driverMetrics {
			var successRate float64
			if dm.TotalExecutions > 0 {
				successRate = float64(dm.SuccessfulExecs) / float64(dm.TotalExecutions) * 100
			}

			driverSummary[name] = map[string]interface{}{
				"executions":   dm.TotalExecutions,
				"success_rate": successRate,
				"avg_duration": dm.AverageDuration,
				"last_exec":    dm.LastExecution,
			}
		}
	}

	summary["drivers"] = driverSummary
	return summary
}
