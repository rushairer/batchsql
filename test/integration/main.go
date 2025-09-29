package main

import (
	"log"
	"os"
	"runtime"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.Println("🚀 启动 BatchSQL 集成测试...")

	// 加载配置
	config := loadConfig()

	// 创建测试报告
	report := &TestReport{
		Timestamp:   time.Now(),
		Environment: "Docker 集成环境",
		GoVersion:   runtime.Version(),
		TestConfig:  config,
		Results:     []TestResult{},
	}

	startTime := time.Now()

	// 运行 MySQL 测试
	if mysqlDSN := os.Getenv("MYSQL_DSN"); mysqlDSN != "" {
		log.Println("📊 正在运行 MySQL 集成测试...")
		mysqlResults := runDatabaseTests("mysql", mysqlDSN, config)
		report.Results = append(report.Results, mysqlResults...)
	}

	// 运行 PostgreSQL 测试
	if postgresDSN := os.Getenv("POSTGRES_DSN"); postgresDSN != "" {
		log.Println("📊 正在运行 PostgreSQL 集成测试...")
		postgresResults := runDatabaseTests("postgres", postgresDSN, config)
		report.Results = append(report.Results, postgresResults...)
	}

	// 运行 SQLite 测试
	if sqliteDSN := os.Getenv("SQLITE_DSN"); sqliteDSN != "" {
		log.Println("📊 正在运行 SQLite 集成测试...")
		sqliteResults := runDatabaseTests("sqlite3", sqliteDSN, config)
		report.Results = append(report.Results, sqliteResults...)
	}

	// 运行 Redis 测试
	if redisDSN := os.Getenv("REDIS_DSN"); redisDSN != "" {
		log.Println("📊 正在运行 Redis 集成测试...")
		redisResults := runRedisTests(redisDSN, config)
		report.Results = append(report.Results, redisResults...)
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
