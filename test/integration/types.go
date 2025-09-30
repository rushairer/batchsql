package main

import "time"

// TestConfig 测试配置
type TestConfig struct {
	TestDuration      time.Duration `json:"test_duration"`
	ConcurrentWorkers int           `json:"concurrent_workers"`
	RecordsPerWorker  int           `json:"records_per_worker"`
	BatchSize         uint32        `json:"batch_size"`
	BufferSize        uint32        `json:"buffer_size"`
	FlushInterval     time.Duration `json:"flush_interval"`
	// Prometheus 配置
	PrometheusEnabled bool `json:"prometheus_enabled"`
	PrometheusPort    int  `json:"prometheus_port"`
}

// TestResult 测试结果
type TestResult struct {
	Database            string        `json:"database"`
	TestName            string        `json:"test_name"`
	Duration            time.Duration `json:"duration"`
	TotalRecords        int64         `json:"total_records"`         // 成功提交的记录数
	ActualRecords       int64         `json:"actual_records"`        // 数据库中实际的记录数
	DataIntegrityRate   float64       `json:"data_integrity_rate"`   // 数据完整性百分比 (0-100)
	DataIntegrityStatus string        `json:"data_integrity_status"` // 数据完整性状态描述
	RecordsPerSecond    float64       `json:"records_per_second"`    // RPS (仅在数据完整性100%时有效)
	RPSValid            bool          `json:"rps_valid"`             // RPS是否有效
	RPSNote             string        `json:"rps_note"`              // RPS说明
	ConcurrentWorkers   int           `json:"concurrent_workers"`
	TestParameters      TestParams    `json:"test_parameters"` // 测试参数
	MemoryUsage         MemoryStats   `json:"memory_usage"`
	Errors              []string      `json:"errors"`
	Success             bool          `json:"success"`
}

// TestParams 测试参数
type TestParams struct {
	BatchSize       uint32        `json:"batch_size"`
	BufferSize      uint32        `json:"buffer_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	ExpectedRecords int64         `json:"expected_records"`
	TestDuration    time.Duration `json:"test_duration"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	NumGC        uint32  `json:"num_gc"`
}

// TestReport 测试报告
type TestReport struct {
	Timestamp   time.Time    `json:"timestamp"`
	Environment string       `json:"environment"`
	GoVersion   string       `json:"go_version"`
	TestConfig  TestConfig   `json:"test_config"`
	Results     []TestResult `json:"results"`
	Summary     TestSummary  `json:"summary"`
}

// TestSummary 测试摘要
type TestSummary struct {
	TotalTests    int     `json:"total_tests"`
	PassedTests   int     `json:"passed_tests"`
	FailedTests   int     `json:"failed_tests"`
	TotalRecords  int64   `json:"total_records"`
	AverageRPS    float64 `json:"average_rps"`
	MaxRPS        float64 `json:"max_rps"`
	TotalDuration string  `json:"total_duration"`
}
