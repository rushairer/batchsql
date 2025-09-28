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
	"github.com/rushairer/batchsql/drivers"
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
	Database            string        `json:"database"`
	TestName            string        `json:"test_name"`
	Duration            time.Duration `json:"duration"`
	TotalRecords        int64         `json:"total_records"`         // 成功提交的记录数
	ActualRecords       int64         `json:"actual_records"`        // 数据库中实际的记录数
	DataIntegrityRate   float64       `json:"data_integrity_rate"`   // 数据完整性百分比
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
		// 每个测试前清理表数据，确保测试独立性
		log.Printf("  🧹 Clearing table before %s...", tc.name)
		if err := clearTestTable(db, dbType); err != nil {
			log.Printf("❌ Failed to clear table before %s: %v", tc.name, err)
			// 继续执行测试，但记录错误
		}

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

// 安全计算内存差值，避免负数溢出
func calculateMemoryDiffMB(after, before uint64) float64 {
	if after >= before {
		return float64(after-before) / 1024 / 1024
	}
	// 如果 after < before（GC回收了内存），返回0而不是负数
	return 0.0
}

// calculateDataIntegrity 计算数据完整性状态
func calculateDataIntegrity(submitted, actual int64) (rate float64, status string, rpsValid bool, rpsNote string) {
	if actual < 0 {
		return 0.0, "❓ 无法验证", false, "无法获取实际记录数，RPS无效"
	}

	if submitted == 0 {
		return 0.0, "❌ 无提交记录", false, "无提交记录，RPS无效"
	}

	rate = float64(actual) / float64(submitted) * 100.0

	if actual == submitted {
		return 100.0, "✅ 完全一致", true, "数据完整性100%，RPS有效"
	} else if actual > submitted {
		return rate, fmt.Sprintf("⚠️ 超出预期 (+%d条)", actual-submitted), false, "数据超出预期，RPS无效"
	} else {
		lossCount := submitted - actual
		lossRate := float64(lossCount) / float64(submitted) * 100.0
		return rate, fmt.Sprintf("❌ 数据丢失 (-%d条, %.1f%%)", lossCount, lossRate), false, fmt.Sprintf("数据丢失%.1f%%，RPS无效", lossRate)
	}
}

// 清理测试表数据 - 使用高性能的清理方式
func clearTestTable(db *sql.DB, dbType string) error {
	switch dbType {
	case "mysql":
		// MySQL 使用 TRUNCATE，性能最佳
		_, err := db.Exec("TRUNCATE TABLE integration_test")
		return err
	case "postgres":
		// PostgreSQL 使用 TRUNCATE，支持级联
		_, err := db.Exec("TRUNCATE TABLE integration_test RESTART IDENTITY")
		return err
	case "sqlite3":
		// SQLite 使用重建表方式，避免锁定问题
		return clearSQLiteTableByRecreate(db)
	default:
		// 兜底方案
		_, err := db.Exec("DELETE FROM integration_test")
		return err
	}
}

// clearSQLiteTableByRecreate SQLite专用的重建表清理方式
func clearSQLiteTableByRecreate(db *sql.DB) error {
	// 1. 删除表
	if _, err := db.Exec("DROP TABLE IF EXISTS integration_test"); err != nil {
		return fmt.Errorf("failed to drop table: %v", err)
	}

	// 2. 重新创建表
	createSQL := `
	CREATE TABLE integration_test (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		data TEXT,
		value REAL,
		is_active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// 3. 重新创建索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_name ON integration_test(name)",
		"CREATE INDEX IF NOT EXISTS idx_email ON integration_test(email)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}

	log.Printf("  ✅ SQLite table recreated successfully")
	return nil
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

	schema := batchsql.NewSchema("integration_test", drivers.ConflictIgnore,
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
		Success: len(errors) == 0 && rpsValid, // 只有数据完整性100%才算成功
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

	schema := batchsql.NewSchema("integration_test", drivers.ConflictIgnore,
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
		Success: len(errors) == 0 && rpsValid, // 只有数据完整性100%才算成功
	}
}

func runLargeBatchTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	// 大批次测试 - 使用更大的批次大小
	largeConfig := config
	largeConfig.BatchSize = 5000
	largeConfig.BufferSize = 50000

	result := runHighThroughputTest(db, dbType, largeConfig)
	result.TestName = "Large Batch Test"
	return result
}

func runMemoryPressureTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	// 内存压力测试 - 使用大数据量和小批次
	memConfig := config
	memConfig.BatchSize = 100
	memConfig.BufferSize = 1000
	memConfig.RecordsPerWorker = 50000

	result := runConcurrentWorkersTest(db, dbType, memConfig)
	result.TestName = "Memory Pressure Test"
	return result
}

func runLongDurationTest(db *sql.DB, dbType string, config TestConfig) TestResult {
	// 长时间运行测试
	longConfig := config
	longConfig.TestDuration = 10 * time.Minute

	result := runHighThroughputTest(db, dbType, longConfig)
	result.TestName = "Long Duration Test"
	return result
}

// getReportsDirectory 智能检测报告目录，兼容本地和Docker环境
func getReportsDirectory() string {
	// 检查是否在Docker环境中（通过检查/app目录是否存在且可写）
	if info, err := os.Stat("/app"); err == nil && info.IsDir() {
		// 尝试在/app目录创建测试文件来检查写权限
		testFile := "/app/.write_test"
		if file, err := os.Create(testFile); err == nil {
			file.Close()
			os.Remove(testFile)
			return "/app/reports" // Docker环境，使用/app/reports
		}
	}

	// 本地环境或Docker环境无写权限，使用相对路径
	return "reports"
}

func generateSummary(results []TestResult, totalDuration time.Duration) TestSummary {
	summary := TestSummary{
		TotalTests:    len(results),
		TotalDuration: totalDuration.String(),
	}

	var totalRecords int64
	var validRPSCount int
	var totalValidRPS float64
	maxRPS := 0.0

	for _, result := range results {
		if result.Success {
			summary.PassedTests++
		} else {
			summary.FailedTests++
		}

		totalRecords += result.TotalRecords

		// 只统计有效的RPS
		if result.RPSValid {
			totalValidRPS += result.RecordsPerSecond
			validRPSCount++

			if result.RecordsPerSecond > maxRPS {
				maxRPS = result.RecordsPerSecond
			}
		}
	}

	summary.TotalRecords = totalRecords
	summary.MaxRPS = maxRPS
	if validRPSCount > 0 {
		summary.AverageRPS = totalValidRPS / float64(validRPSCount)
	} else {
		summary.AverageRPS = 0.0 // 没有有效的RPS数据
	}

	return summary
}

func saveReport(report *TestReport) {
	// 智能检测报告目录 - 兼容本地和Docker环境
	reportsDir := getReportsDirectory()
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		log.Printf("❌ Failed to create reports directory: %v", err)
		return
	}

	// 生成文件名
	timestamp := report.Timestamp.Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s/integration_test_report_%s.json", reportsDir, timestamp)

	// 保存 JSON 报告
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("❌ Failed to marshal report: %v", err)
		return
	}

	if err := os.WriteFile(filename, data, 0o644); err != nil {
		log.Printf("❌ Failed to save report: %v", err)
		return
	}

	log.Printf("📊 Test report saved to: %s", filename)

	// 生成 HTML 报告
	generateHTMLReport(report, timestamp, reportsDir)
}

func generateHTMLReport(report *TestReport, timestamp string, reportsDir string) {
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>BatchSQL 集成测试报告</title>
    <style>
        body { font-family: "Microsoft YaHei", "SimHei", Arial, sans-serif; margin: 20px; }
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
        <h1>🚀 BatchSQL 集成测试报告</h1>
        <p><strong>测试时间:</strong> %s</p>
        <p><strong>测试环境:</strong> %s</p>
        <p><strong>Go 版本:</strong> %s</p>
    </div>

    <div class="summary %s">
        <h2>📊 测试摘要</h2>
        <div class="metric"><strong>总测试数:</strong> %d</div>
        <div class="metric"><strong>通过:</strong> %d</div>
        <div class="metric"><strong>失败:</strong> %d</div>
        <div class="metric"><strong>总记录数:</strong> %d</div>
        <div class="metric"><strong>平均 RPS:</strong> %.2f</div>
        <div class="metric"><strong>最大 RPS:</strong> %.2f</div>
        <div class="metric"><strong>总耗时:</strong> %s</div>
    </div>

    <h2>📋 测试结果</h2>
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

		// 使用新的数据完整性状态
		consistencyStatus := result.DataIntegrityStatus

		actualRecordsDisplay := "N/A"
		if result.ActualRecords >= 0 {
			actualRecordsDisplay = fmt.Sprintf("%d", result.ActualRecords)
		}

		// RPS显示逻辑
		rpsDisplay := ""
		if result.RPSValid {
			rpsDisplay = fmt.Sprintf("%.2f", result.RecordsPerSecond)
		} else {
			rpsDisplay = fmt.Sprintf("<s>%.2f</s> (无效)", result.RecordsPerSecond)
		}

		htmlContent += fmt.Sprintf(`
    <div class="result %s">
        <h3>%s %s - %s</h3>
        <table>
            <tr><th>指标</th><th>数值</th></tr>
            <tr><td>测试耗时</td><td>%s</td></tr>
            <tr><td>提交记录数</td><td>%d</td></tr>
            <tr><td>数据库实际记录数</td><td>%s</td></tr>
            <tr><td>数据完整性</td><td>%s (%.1f%%)</td></tr>
            <tr><td>每秒记录数 (RPS)</td><td>%s</td></tr>
            <tr><td>RPS有效性</td><td>%s</td></tr>
            <tr><td>并发工作者数</td><td>%d</td></tr>
            <tr><td>批次大小</td><td>%d</td></tr>
            <tr><td>缓冲区大小</td><td>%d</td></tr>
            <tr><td>刷新间隔</td><td>%s</td></tr>
            <tr><td>内存分配 (MB)</td><td>%.2f</td></tr>
            <tr><td>总内存分配 (MB)</td><td>%.2f</td></tr>
            <tr><td>系统内存 (MB)</td><td>%.2f</td></tr>
            <tr><td>GC 运行次数</td><td>%d</td></tr>
            <tr><td>错误数量</td><td>%d</td></tr>
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
			result.DataIntegrityRate,
			rpsDisplay,
			result.RPSNote,
			result.ConcurrentWorkers,
			result.TestParameters.BatchSize,
			result.TestParameters.BufferSize,
			result.TestParameters.FlushInterval.String(),
			result.MemoryUsage.AllocMB,
			result.MemoryUsage.TotalAllocMB,
			result.MemoryUsage.SysMB,
			result.MemoryUsage.NumGC,
			len(result.Errors),
		)

		if len(result.Errors) > 0 {
			htmlContent += "<h4>错误信息:</h4><ul>"
			for i, err := range result.Errors {
				if i >= 10 { // 只显示前10个错误
					htmlContent += fmt.Sprintf("<li>... 还有 %d 个错误</li>", len(result.Errors)-10)
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

	htmlFilename := fmt.Sprintf("%s/integration_test_report_%s.html", reportsDir, timestamp)
	if err := os.WriteFile(htmlFilename, []byte(htmlContent), 0o644); err != nil {
		log.Printf("❌ Failed to save HTML report: %v", err)
		return
	}

	log.Printf("📊 HTML report saved to: %s", htmlFilename)
}

func printSummary(report *TestReport) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🚀 BATCHSQL 集成测试总结")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("📅 测试时间: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("🌍 测试环境: %s\n", report.Environment)
	fmt.Printf("🔧 Go 版本: %s\n", report.GoVersion)

	fmt.Println("\n📊 总体结果:")
	fmt.Printf("   总测试数: %d\n", report.Summary.TotalTests)
	fmt.Printf("   ✅ 通过: %d\n", report.Summary.PassedTests)
	fmt.Printf("   ❌ 失败: %d\n", report.Summary.FailedTests)
	fmt.Printf("   📈 总记录数: %d\n", report.Summary.TotalRecords)
	fmt.Printf("   ⚡ 平均 RPS: %.2f\n", report.Summary.AverageRPS)
	fmt.Printf("   🚀 最大 RPS: %.2f\n", report.Summary.MaxRPS)
	fmt.Printf("   ⏱️  总耗时: %s\n", report.Summary.TotalDuration)

	fmt.Println("\n📋 详细结果:")
	for _, result := range report.Results {
		status := "✅"
		if !result.Success {
			status = "❌"
		}

		// 使用新的数据完整性信息
		consistencyInfo := fmt.Sprintf(" | %s (%.1f%%)", result.DataIntegrityStatus, result.DataIntegrityRate)

		// RPS显示
		rpsInfo := ""
		if result.RPSValid {
			rpsInfo = fmt.Sprintf("RPS: %.2f", result.RecordsPerSecond)
		} else {
			rpsInfo = fmt.Sprintf("RPS: ~~%.2f~~ (无效)", result.RecordsPerSecond)
		}

		fmt.Printf("   %s %s - %s\n", status, result.Database, result.TestName)
		fmt.Printf("      耗时: %s | 提交: %d | %s | 工作者: %d | 错误: %d%s\n",
			result.Duration.String(),
			result.TotalRecords,
			rpsInfo,
			result.ConcurrentWorkers,
			len(result.Errors),
			consistencyInfo,
		)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))

	if report.Summary.FailedTests > 0 {
		fmt.Println("❌ 部分测试失败 - 请查看详细报告获取更多信息")
	} else {
		fmt.Println("🎉 所有测试通过 - BatchSQL 运行状态优秀！")
	}

	fmt.Println(strings.Repeat("=", 80))
}
