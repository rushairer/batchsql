package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rushairer/batchsql"
)

// 环境变量解析辅助函数
func parseIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// TestConfig 测试配置
type TestConfig struct {
	TestDuration      time.Duration `json:"test_duration"`
	ConcurrentWorkers int           `json:"concurrent_workers"`
	RecordsPerWorker  int           `json:"records_per_worker"`
	BatchSize         uint32        `json:"batch_size"`
	BufferSize        uint32        `json:"buffer_size"`
	FlushInterval     time.Duration `json:"flush_interval"`
}

// TestResult 测试结果
type TestResult struct {
	Database          string        `json:"database"`
	TestName          string        `json:"test_name"`
	Duration          time.Duration `json:"duration"`
	TotalRecords      int64         `json:"total_records"`  // 成功提交的记录数
	ActualRecords     int64         `json:"actual_records"` // 数据库中实际的记录数
	RecordsPerSecond  float64       `json:"records_per_second"`
	ConcurrentWorkers int           `json:"concurrent_workers"`
	MemoryUsage       MemoryStats   `json:"memory_usage"`
	Errors            []string      `json:"errors"`
	Success           bool          `json:"success"`
}

// MemoryStats 内存统计
type MemoryStats struct {
	AllocMB      uint64 `json:"alloc_mb"`
	TotalAllocMB uint64 `json:"total_alloc_mb"`
	SysMB        uint64 `json:"sys_mb"`
	NumGC        uint32 `json:"num_gc"`
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

func main() {
	log.Println("🚀 Starting BatchSQL Integration Tests...")

	// 加载配置
	config := loadConfig()

	// 创建测试报告
	report := &TestReport{
		Timestamp:   time.Now(),
		Environment: "Docker Integration",
		GoVersion:   runtime.Version(),
		TestConfig:  config,
		Results:     []TestResult{},
	}

	startTime := time.Now()

	// 运行 MySQL 测试
	if mysqlDSN := os.Getenv("MYSQL_DSN"); mysqlDSN != "" {
		log.Println("📊 Running MySQL integration tests...")
		mysqlResults := runDatabaseTests("mysql", mysqlDSN, config)
		report.Results = append(report.Results, mysqlResults...)
	}

	// 运行 PostgreSQL 测试
	if postgresDSN := os.Getenv("POSTGRES_DSN"); postgresDSN != "" {
		log.Println("📊 Running PostgreSQL integration tests...")
		postgresResults := runDatabaseTests("postgres", postgresDSN, config)
		report.Results = append(report.Results, postgresResults...)
	}

	// 运行 SQLite 测试
	if sqliteDSN := os.Getenv("SQLITE_DSN"); sqliteDSN != "" {
		log.Println("📊 Running SQLite integration tests...")
		sqliteResults := runDatabaseTests("sqlite3", sqliteDSN, config)
		report.Results = append(report.Results, sqliteResults...)
	}

	// 生成摘要
	report.Summary = generateSummary(report.Results, time.Since(startTime))

	// 保存报告
	saveReport(report)

	// 输出结果
	printSummary(report)

	// 如果有失败的测试，退出码为 1
	if report.Summary.FailedTests > 0 {
		os.Exit(1)
	}
}

func loadConfig() TestConfig {
	// 统一从环境变量读取配置，docker-compose为唯一配置源
	config := TestConfig{
		TestDuration:      parseDurationEnv("TEST_DURATION", 1800*time.Second), // 30分钟默认
		ConcurrentWorkers: parseIntEnv("CONCURRENT_WORKERS", 10),
		RecordsPerWorker:  parseIntEnv("RECORDS_PER_WORKER", 2000),
		BatchSize:         uint32(parseIntEnv("BATCH_SIZE", 200)),
		BufferSize:        uint32(parseIntEnv("BUFFER_SIZE", 5000)),
		FlushInterval:     parseDurationEnv("FLUSH_INTERVAL", 100*time.Millisecond),
	}

	log.Printf("📋 Loaded Test Configuration:")
	log.Printf("   Test Duration: %v", config.TestDuration)
	log.Printf("   Concurrent Workers: %d", config.ConcurrentWorkers)
	log.Printf("   Records Per Worker: %d", config.RecordsPerWorker)
	log.Printf("   Batch Size: %d", config.BatchSize)
	log.Printf("   Buffer Size: %d", config.BufferSize)
	log.Printf("   Flush Interval: %v", config.FlushInterval)

	return config
}

func runDatabaseTests(dbType, dsn string, config TestConfig) []TestResult {
	var results []TestResult

	// 连接数据库
	db, err := sql.Open(dbType, dsn)
	if err != nil {
		log.Printf("❌ Failed to connect to %s: %v", dbType, err)
		return results
	}
	defer db.Close()

	// 设置连接池
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Printf("❌ Failed to ping %s: %v", dbType, err)
		return results
	}

	// 创建测试表
	if err := createTestTables(db, dbType); err != nil {
		log.Printf("❌ Failed to create test tables for %s: %v", dbType, err)
		return results
	}

	// 运行不同的测试场景
	testCases := []struct {
		name     string
		testFunc func(*sql.DB, string, TestConfig) TestResult
	}{
		{"High Throughput Test", runHighThroughputTest},
		{"Concurrent Workers Test", runConcurrentWorkersTest},
		{"Large Batch Test", runLargeBatchTest},
		{"Memory Pressure Test", runMemoryPressureTest},
		{"Long Duration Test", runLongDurationTest},
	}

	for _, tc := range testCases {
		log.Printf("  🔄 Running %s on %s...", tc.name, dbType)
		result := tc.testFunc(db, dbType, config)
		result.TestName = tc.name
		result.Database = dbType
		results = append(results, result)

		// 测试间隔，让系统恢复
		time.Sleep(5 * time.Second)
	}

	return results
}

func createTestTables(db *sql.DB, dbType string) error {
	var createSQL string

	switch dbType {
	case "mysql":
		createSQL = `
		DROP TABLE IF EXISTS integration_test;
		CREATE TABLE integration_test (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			data TEXT,
			value DECIMAL(10,2),
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_name (name),
			INDEX idx_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
		`
	case "postgres":
		createSQL = `
		DROP TABLE IF EXISTS integration_test;
		CREATE TABLE integration_test (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			data TEXT,
			value DECIMAL(10,2),
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_name ON integration_test(name);
		CREATE INDEX IF NOT EXISTS idx_email ON integration_test(email);
		`
	case "sqlite3":
		createSQL = `
		DROP TABLE IF EXISTS integration_test;
		CREATE TABLE integration_test (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			data TEXT,
			value REAL,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE INDEX IF NOT EXISTS idx_name ON integration_test(name);
		CREATE INDEX IF NOT EXISTS idx_email ON integration_test(email);
		`
	}

	_, err := db.Exec(createSQL)
	return err
}

// 验证数据库中的实际记录数
func getActualRecordCount(db *sql.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM integration_test").Scan(&count)
	return count, err
}

func runHighThroughputTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	ctx := context.Background()

	var batchSQL *batchsql.BatchSQL
	switch dbType {
	case "mysql":
		batchSQL = batchsql.NewMySQLBatchSQL(ctx, db, batchsql.PipelineConfig{
			BufferSize:    config.BufferSize,
			FlushSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	case "postgres":
		batchSQL = batchsql.NewPostgreSQLBatchSQL(ctx, db, batchsql.PipelineConfig{
			BufferSize:    config.BufferSize,
			FlushSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	case "sqlite3":
		batchSQL = batchsql.NewSQLiteBatchSQL(ctx, db, batchsql.PipelineConfig{
			BufferSize:    config.BufferSize,
			FlushSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	}

	schema := batchsql.NewSchema("integration_test", batchsql.ConflictIgnore,
		"id", "name", "email", "data", "value", "is_active", "created_at")

	startTime := time.Now()
	var recordCount int64
	var errors []string

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 高吞吐量测试 - 限制记录数量避免内存泄漏
	testCtx, cancel := context.WithTimeout(ctx, config.TestDuration)
	defer cancel()

	maxRecords := int64(config.ConcurrentWorkers * config.RecordsPerWorker) // 限制最大记录数

	for i := int64(0); i < maxRecords; i++ {
		select {
		case <-testCtx.Done():
			goto TestComplete
		default:
			request := batchsql.NewRequest(schema).
				SetInt64("id", i).
				SetString("name", fmt.Sprintf("User_%d", i)).
				SetString("email", fmt.Sprintf("user_%d@example.com", i)).
				SetString("data", fmt.Sprintf("Data_%d", i)). // 减少字符串长度
				SetFloat64("value", float64(i%10000)/100.0).
				SetBool("is_active", i%2 == 0).
				SetTime("created_at", time.Now())

			if err := batchSQL.Submit(testCtx, request); err != nil {
				errors = append(errors, err.Error())
				if len(errors) > 100 { // 限制错误数量
					break
				}
			} else {
				recordCount++
			}

			// 定期强制GC，避免内存积累
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

	// 查询数据库中的实际记录数
	actualRecords, countErr := getActualRecordCount(db)
	if countErr != nil {
		errors = append(errors, fmt.Sprintf("Failed to count actual records: %v", countErr))
		actualRecords = -1 // 标记为无法获取
	}

	return TestResult{
		Duration:          duration,
		TotalRecords:      recordCount,
		ActualRecords:     actualRecords,
		RecordsPerSecond:  float64(recordCount) / duration.Seconds(),
		ConcurrentWorkers: 1,
		MemoryUsage: MemoryStats{
			AllocMB:      (m2.Alloc - m1.Alloc) / 1024 / 1024,
			TotalAllocMB: (m2.TotalAlloc - m1.TotalAlloc) / 1024 / 1024,
			SysMB:        (m2.Sys - m1.Sys) / 1024 / 1024,
			NumGC:        m2.NumGC - m1.NumGC,
		},
		Errors:  errors,
		Success: len(errors) == 0,
	}
}

func runConcurrentWorkersTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	ctx := context.Background()

	var batchSQL *batchsql.BatchSQL
	switch dbType {
	case "mysql":
		batchSQL = batchsql.NewMySQLBatchSQL(ctx, db, batchsql.PipelineConfig{
			BufferSize:    config.BufferSize,
			FlushSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	case "postgres":
		batchSQL = batchsql.NewPostgreSQLBatchSQL(ctx, db, batchsql.PipelineConfig{
			BufferSize:    config.BufferSize,
			FlushSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	case "sqlite3":
		batchSQL = batchsql.NewSQLiteBatchSQL(ctx, db, batchsql.PipelineConfig{
			BufferSize:    config.BufferSize,
			FlushSize:     config.BatchSize,
			FlushInterval: config.FlushInterval,
		})
	}

	schema := batchsql.NewSchema("integration_test", batchsql.ConflictIgnore,
		"id", "name", "email", "data", "value", "is_active", "created_at")

	startTime := time.Now()
	var totalRecords int64
	var mu sync.Mutex
	var errors []string

	// 记录初始内存
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// 并发工作者测试 - 批次处理避免内存峰值
	var wg sync.WaitGroup
	batchSize := 100 // 每批处理100条记录

	for workerID := 0; workerID < config.ConcurrentWorkers; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			workerRecords := 0
			baseID := int64(id * config.RecordsPerWorker)

			// 分批处理，避免内存峰值
			for batch := 0; batch < config.RecordsPerWorker; batch += batchSize {
				endIdx := batch + batchSize
				if endIdx > config.RecordsPerWorker {
					endIdx = config.RecordsPerWorker
				}

				for i := batch; i < endIdx; i++ {
					request := batchsql.NewRequest(schema).
						SetInt64("id", baseID+int64(i)).
						SetString("name", fmt.Sprintf("W%d_U%d", id, i)).          // 缩短字符串
						SetString("email", fmt.Sprintf("u%d_%d@test.com", id, i)). // 缩短字符串
						SetString("data", fmt.Sprintf("D%d_%d", id, i)).           // 大幅缩短数据
						SetFloat64("value", float64((id*100+i)%1000)/10.0).
						SetBool("is_active", (id+i)%2 == 0).
						SetTime("created_at", time.Now())

					if err := batchSQL.Submit(ctx, request); err != nil {
						mu.Lock()
						errors = append(errors, fmt.Sprintf("Worker %d: %v", id, err))
						mu.Unlock()
					} else {
						workerRecords++
					}
				}

				// 每批处理完后强制GC
				runtime.GC()

				// 添加小延迟，避免过度竞争
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

	// 查询数据库中的实际记录数
	actualRecords, countErr := getActualRecordCount(db)
	if countErr != nil {
		mu.Lock()
		errors = append(errors, fmt.Sprintf("Failed to count actual records: %v", countErr))
		mu.Unlock()
		actualRecords = -1 // 标记为无法获取
	}

	return TestResult{
		Duration:          duration,
		TotalRecords:      totalRecords,
		ActualRecords:     actualRecords,
		RecordsPerSecond:  float64(totalRecords) / duration.Seconds(),
		ConcurrentWorkers: config.ConcurrentWorkers,
		MemoryUsage: MemoryStats{
			AllocMB:      (m2.Alloc - m1.Alloc) / 1024 / 1024,
			TotalAllocMB: (m2.TotalAlloc - m1.TotalAlloc) / 1024 / 1024,
			SysMB:        (m2.Sys - m1.Sys) / 1024 / 1024,
			NumGC:        m2.NumGC - m1.NumGC,
		},
		Errors:  errors,
		Success: len(errors) == 0,
	}
}

func runLargeBatchTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	// 大批次测试 - 使用更大的批次大小
	largeConfig := config
	largeConfig.BatchSize = 5000
	largeConfig.BufferSize = 50000

	return runHighThroughputTest(db, dbType, largeConfig)
}

func runMemoryPressureTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	// 内存压力测试 - 使用大数据量和小批次
	memConfig := config
	memConfig.BatchSize = 100
	memConfig.BufferSize = 1000
	memConfig.RecordsPerWorker = 50000

	return runConcurrentWorkersTest(db, dbType, memConfig)
}

func runLongDurationTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	// 长时间运行测试
	longConfig := config
	longConfig.TestDuration = 10 * time.Minute

	return runHighThroughputTest(db, dbType, longConfig)
}

func generateSummary(results []TestResult, totalDuration time.Duration) TestSummary {
	summary := TestSummary{
		TotalTests:    len(results),
		TotalDuration: totalDuration.String(),
	}

	var totalRecords int64
	var totalRPS float64
	maxRPS := 0.0

	for _, result := range results {
		if result.Success {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}

		totalRecords += result.TotalRecords
		totalRPS += result.RecordsPerSecond

		if result.RecordsPerSecond > maxRPS {
			maxRPS = result.RecordsPerSecond
		}
	}

	summary.TotalRecords = totalRecords
	summary.MaxRPS = maxRPS
	if len(results) > 0 {
		summary.AverageRPS = totalRPS / float64(len(results))
	}

	return summary
}

func saveReport(report *TestReport) {
	// 确保报告目录存在
	if err := os.MkdirAll("/app/reports", 0755); err != nil {
		log.Printf("❌ Failed to create reports directory: %v", err)
		return
	}

	// 生成文件名
	timestamp := report.Timestamp.Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("/app/reports/integration_test_report_%s.json", timestamp)

	// 保存 JSON 报告
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("❌ Failed to marshal report: %v", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("❌ Failed to save report: %v", err)
		return
	}

	log.Printf("📊 Test report saved to: %s", filename)

	// 生成 HTML 报告
	generateHTMLReport(report, timestamp)
}

func generateHTMLReport(report *TestReport, timestamp string) {
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>BatchSQL Integration Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background: #f4f4f4; padding: 20px; border-radius: 5px; }
        .summary { background: #e8f5e8; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .failed { background: #ffe8e8; }
        .result { margin: 10px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .success { border-left: 5px solid #4CAF50; }
        .error { border-left: 5px solid #f44336; }
        table { width: 100%%; border-collapse: collapse; margin: 10px 0; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
        .metric { display: inline-block; margin: 10px; padding: 10px; background: #f9f9f9; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 BatchSQL Integration Test Report</h1>
        <p><strong>Timestamp:</strong> %s</p>
        <p><strong>Environment:</strong> %s</p>
        <p><strong>Go Version:</strong> %s</p>
    </div>

    <div class="summary %s">
        <h2>📊 Test Summary</h2>
        <div class="metric"><strong>Total Tests:</strong> %d</div>
        <div class="metric"><strong>Passed:</strong> %d</div>
        <div class="metric"><strong>Failed:</strong> %d</div>
        <div class="metric"><strong>Total Records:</strong> %d</div>
        <div class="metric"><strong>Average RPS:</strong> %.2f</div>
        <div class="metric"><strong>Max RPS:</strong> %.2f</div>
        <div class="metric"><strong>Total Duration:</strong> %s</div>
    </div>

    <h2>📋 Test Results</h2>
`,
		report.Timestamp.Format("2006-01-02 15:04:05"),
		report.Environment,
		report.GoVersion,
		func() string {
			if report.Summary.FailedTests > 0 {
				return "failed"
			}
			return ""
		}(),
		report.Summary.TotalTests,
		report.Summary.PassedTests,
		report.Summary.FailedTests,
		report.Summary.TotalRecords,
		report.Summary.AverageRPS,
		report.Summary.MaxRPS,
		report.Summary.TotalDuration,
	)

	for _, result := range report.Results {
		status := "success"
		statusIcon := "✅"
		if !result.Success {
			status = "error"
			statusIcon = "❌"
		}

		// 计算数据一致性状态
		consistencyStatus := ""
		if result.ActualRecords >= 0 {
			if result.ActualRecords == result.TotalRecords {
				consistencyStatus = "✅ 一致"
			} else {
				consistencyStatus = fmt.Sprintf("⚠️ 不一致 (差异: %d)", result.ActualRecords-result.TotalRecords)
			}
		} else {
			consistencyStatus = "❓ 无法验证"
		}

		actualRecordsDisplay := "N/A"
		if result.ActualRecords >= 0 {
			actualRecordsDisplay = fmt.Sprintf("%d", result.ActualRecords)
		}

		htmlContent += fmt.Sprintf(`
    <div class="result %s">
        <h3>%s %s - %s</h3>
        <table>
            <tr><th>Metric</th><th>Value</th></tr>
            <tr><td>Duration</td><td>%s</td></tr>
            <tr><td>提交记录数</td><td>%d</td></tr>
            <tr><td>数据库实际记录数</td><td>%s</td></tr>
            <tr><td>数据一致性</td><td>%s</td></tr>
            <tr><td>Records/Second</td><td>%.2f</td></tr>
            <tr><td>Concurrent Workers</td><td>%d</td></tr>
            <tr><td>Memory Alloc (MB)</td><td>%d</td></tr>
            <tr><td>Total Alloc (MB)</td><td>%d</td></tr>
            <tr><td>System Memory (MB)</td><td>%d</td></tr>
            <tr><td>GC Runs</td><td>%d</td></tr>
            <tr><td>Errors</td><td>%d</td></tr>
        </table>
`,
			status,
			statusIcon,
			result.Database,
			result.TestName,
			result.Duration.String(),
			result.TotalRecords,
			actualRecordsDisplay,
			consistencyStatus,
			result.RecordsPerSecond,
			result.ConcurrentWorkers,
			result.MemoryUsage.AllocMB,
			result.MemoryUsage.TotalAllocMB,
			result.MemoryUsage.SysMB,
			result.MemoryUsage.NumGC,
			len(result.Errors),
		)

		if len(result.Errors) > 0 {
			htmlContent += "<h4>Errors:</h4><ul>"
			for i, err := range result.Errors {
				if i >= 10 { // 只显示前10个错误
					htmlContent += fmt.Sprintf("<li>... and %d more errors</li>", len(result.Errors)-10)
					break
				}
				htmlContent += fmt.Sprintf("<li>%s</li>", err)
			}
			htmlContent += "</ul>"
		}

		htmlContent += "</div>"
	}

	htmlContent += `
</body>
</html>`

	htmlFilename := fmt.Sprintf("/app/reports/integration_test_report_%s.html", timestamp)
	if err := os.WriteFile(htmlFilename, []byte(htmlContent), 0644); err != nil {
		log.Printf("❌ Failed to save HTML report: %v", err)
		return
	}

	log.Printf("📊 HTML report saved to: %s", htmlFilename)
}

func printSummary(report *TestReport) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🚀 BATCHSQL INTEGRATION TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("📅 Timestamp: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("🌍 Environment: %s\n", report.Environment)
	fmt.Printf("🔧 Go Version: %s\n", report.GoVersion)

	fmt.Println("\n📊 OVERALL RESULTS:")
	fmt.Printf("   Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Printf("   ✅ Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("   ❌ Failed: %d\n", report.Summary.FailedTests)
	fmt.Printf("   📈 Total Records: %d\n", report.Summary.TotalRecords)
	fmt.Printf("   ⚡ Average RPS: %.2f\n", report.Summary.AverageRPS)
	fmt.Printf("   🚀 Max RPS: %.2f\n", report.Summary.MaxRPS)
	fmt.Printf("   ⏱️  Total Duration: %s\n", report.Summary.TotalDuration)

	fmt.Println("\n📋 DETAILED RESULTS:")
	for _, result := range report.Results {
		status := "✅"
		if !result.Success {
			status = "❌"
		}

		// 计算数据一致性状态
		consistencyInfo := ""
		if result.ActualRecords >= 0 {
			if result.ActualRecords == result.TotalRecords {
				consistencyInfo = " | 数据一致 ✅"
			} else {
				consistencyInfo = fmt.Sprintf(" | 数据不一致 ⚠️ (实际:%d vs 提交:%d)", result.ActualRecords, result.TotalRecords)
			}
		} else {
			consistencyInfo = " | 数据验证失败 ❓"
		}

		fmt.Printf("   %s %s - %s\n", status, result.Database, result.TestName)
		fmt.Printf("      Duration: %s | 提交: %d | RPS: %.2f | Workers: %d | Errors: %d%s\n",
			result.Duration.String(),
			result.TotalRecords,
			result.RecordsPerSecond,
			result.ConcurrentWorkers,
			len(result.Errors),
			consistencyInfo,
		)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))

	if report.Summary.FailedTests > 0 {
		fmt.Println("❌ SOME TESTS FAILED - Check the detailed report for more information")
	} else {
		fmt.Println("🎉 ALL TESTS PASSED - BatchSQL is performing excellently!")
	}

	fmt.Println(strings.Repeat("=", 80))
}
