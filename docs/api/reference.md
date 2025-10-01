# BatchSQL API 参考手册

## 📖 概述

BatchSQL 提供了简洁而强大的API，支持多种数据库的高性能批量操作。本文档提供完整的API参考和最佳实践。

## 🏗️ 核心组件

### BatchSQL 主类

```go
type BatchSQL struct {
    // 内部字段（不直接访问）
}

// 创建BatchSQL实例
func NewBatchSQL(
    ctx context.Context,
    bufferSize int,
    batchSize int,
    flushInterval time.Duration,
    executor batchsql.BatchExecutor,
) *BatchSQL
```

**参数说明**：
- `ctx`: 上下文，用于控制生命周期
- `bufferSize`: 内存缓冲区大小（推荐：1000-10000）
- `batchSize`: 批次大小（推荐：100-1000）
- `flushInterval`: 刷新间隔（推荐：100ms-1s）
- `executor`: 批量执行器实现

### Submit 取消语义（v1.1.1 起）

- 当传入的 ctx 已被取消或超时，Submit 会在尝试入队之前立即返回 ctx.Err()（context.Canceled 或 context.DeadlineExceeded）
- 对提交通道的选择发生前即检查 ctx，避免“已入队但外部随后取消”的不确定性
- 调用方应在提交前管理好 context 生命周期，避免无效提交

最小示例：
```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
cancel() // 或自然超时

if err := batch.Submit(ctx, req); err != nil {
    // 立即返回 context.Canceled 或 context.DeadlineExceeded，不会入队
    log.Printf("submit cancelled: %v", err)
}
```

### Schema 定义

```go
type Schema struct {
    Name     string
    ConflictMode  ConflictMode
    Fields        []string
}
```

### 可选并发限流（WithConcurrencyLimit）

```go
// 直接在执行器上启用限流（示例：MySQL）
executor := batchsql.NewSQLThrottledBatchExecutorWithDriver(db, batchsql.DefaultMySQLDriver).
    WithConcurrencyLimit(8)

// 创建 BatchSQL
batch := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, executor)
```

说明：
- limit <= 0 不启用限流（默认行为）
- 限流在 ExecuteBatch 入口，避免攒批后同时触发高并发
- 指标上报与错误处理与不限流路径一致

// 创建Schema
func NewSchema(tableName string, conflictMode ConflictMode, fields ...string) *Schema
```

**冲突处理模式**：
```go
const (
    ConflictIgnore  ConflictMode = "IGNORE"   // 忽略冲突
    ConflictReplace ConflictMode = "REPLACE"  // 替换冲突
    ConflictUpdate  ConflictMode = "UPDATE"   // 更新冲突
)
```

### Request 构建

```go
type Request struct {
    schema *Schema
    data   map[string]any
}

// 创建请求
func NewRequest(schema *Schema) *Request

// 设置字段值
func (r *Request) SetString(field, value string) *Request
func (r *Request) SetInt64(field string, value int64) *Request
func (r *Request) SetFloat64(field string, value float64) *Request
func (r *Request) SetBool(field string, value bool) *Request
func (r *Request) SetTime(field string, value time.Time) *Request
func (r *Request) SetBytes(field string, value []byte) *Request
func (r *Request) SetAny(field string, value any) *Request
```

## 🔌 数据库驱动

### MySQL 驱动

```go
import "github.com/rushairer/batchsql/drivers/mysql"

// 创建MySQL执行器
func NewBatchExecutor(db *sql.DB) batchsql.BatchExecutor

// 使用示例
db, _ := sql.Open("mysql", dsn)
executor := mysql.NewBatchExecutor(db)
batchSQL := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, executor)
```

### PostgreSQL 驱动

```go
import "github.com/rushairer/batchsql/drivers/postgresql"

// 创建PostgreSQL执行器
func NewBatchExecutor(db *sql.DB) batchsql.BatchExecutor

// 使用示例
db, _ := sql.Open("postgres", dsn)
executor := postgresql.NewBatchExecutor(db)
batchSQL := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, executor)
```

### SQLite 驱动

```go
import "github.com/rushairer/batchsql/drivers/sqlite"

// 创建SQLite执行器
func NewBatchExecutor(db *sql.DB) batchsql.BatchExecutor

// 使用示例
db, _ := sql.Open("sqlite3", dsn)
executor := sqlite.NewBatchExecutor(db)
batchSQL := batchsql.NewBatchSQL(ctx, 1000, 100, 200*time.Millisecond, executor)
```

### Redis 驱动

```go
import "github.com/rushairer/batchsql/drivers/redis"

// 创建Redis执行器
func NewBatchExecutor(rdb *redis.Client) batchsql.BatchExecutor

// 使用示例
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
executor := redis.NewBatchExecutor(rdb)
batchSQL := batchsql.NewBatchSQL(ctx, 5000, 500, 50*time.Millisecond, executor)
```

## 📊 指标监控

### MetricsReporter 接口

```go
type MetricsReporter interface {
    RecordBatchExecution(tableName string, batchSize int, duration int64, status string)
}
```

### WithMetricsReporter 最佳实践

#### 1. 基本用法

```go
// 创建指标报告器
metricsReporter := NewCustomMetricsReporter()

// 为执行器添加指标监控
executor := mysql.NewBatchExecutor(db)
executor = executor.WithMetricsReporter(metricsReporter)

// 创建BatchSQL实例
batchSQL := batchsql.NewBatchSQL(ctx, bufferSize, batchSize, flushInterval, executor)
```

#### 2. Prometheus 集成示例

```go
// 创建Prometheus指标报告器
prometheusMetrics := NewPrometheusMetrics()
metricsReporter := NewPrometheusMetricsReporter(prometheusMetrics, "mysql", "batch_insert")

// 应用到执行器
executor := mysql.NewBatchExecutor(db)
if prometheusMetrics != nil {
    executor = executor.WithMetricsReporter(metricsReporter).(*batchsql.CommonExecutor)
}

batchSQL := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, executor)
```

#### 3. 自定义指标报告器

```go
type CustomMetricsReporter struct {
    logger *log.Logger
    stats  *Stats
}

func (r *CustomMetricsReporter) RecordBatchExecution(tableName string, batchSize int, duration int64, status string) {
    r.logger.Printf("Batch executed: table=%s, size=%d, duration=%dms, status=%s", 
        tableName, batchSize, duration, status)
    
    r.stats.RecordBatch(batchSize, duration, status == "success")
}

// 使用自定义报告器
metricsReporter := &CustomMetricsReporter{
    logger: log.New(os.Stdout, "[METRICS] ", log.LstdFlags),
    stats:  NewStats(),
}

executor = executor.WithMetricsReporter(metricsReporter)
```

#### 4. 多数据库监控模式

```go
func setupExecutorWithMetrics(dbType string, db interface{}, prometheusMetrics *PrometheusMetrics, testName string) batchsql.BatchExecutor {
    var executor batchsql.BatchExecutor
    
    switch dbType {
    case "mysql":
        executor = mysql.NewBatchExecutor(db.(*sql.DB))
    case "postgres":
        executor = postgresql.NewBatchExecutor(db.(*sql.DB))
    case "sqlite3":
        executor = sqlite.NewBatchExecutor(db.(*sql.DB))
    case "redis":
        executor = redis.NewBatchExecutor(db.(*redis.Client))
    }
    
    // 统一添加指标监控
    if prometheusMetrics != nil {
        metricsReporter := NewPrometheusMetricsReporter(prometheusMetrics, dbType, testName)
        executor = executor.WithMetricsReporter(metricsReporter)
    }
    
    return executor
}
```

## 🚀 完整使用示例

### 基础批量插入

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/rushairer/batchsql"
    "github.com/rushairer/batchsql/drivers"
    "github.com/rushairer/batchsql/drivers/mysql"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // 1. 连接数据库
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // 2. 创建执行器
    executor := mysql.NewBatchExecutor(db)
    
    // 3. 创建BatchSQL实例
    ctx := context.Background()
    batchSQL := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, executor)
    defer batchSQL.Close()
    
    // 4. 定义Schema
    schema := batchsql.NewSchema("users", batchsql.ConflictIgnore,
        "id", "name", "email", "created_at")
    
    // 5. 批量提交数据
    for i := 0; i < 10000; i++ {
        request := batchsql.NewRequest(schema).
            SetInt64("id", int64(i)).
            SetString("name", fmt.Sprintf("User_%d", i)).
            SetString("email", fmt.Sprintf("user_%d@example.com", i)).
            SetTime("created_at", time.Now())
        
        if err := batchSQL.Submit(ctx, request); err != nil {
            fmt.Printf("Submit error: %v\n", err)
        }
    }
    
    fmt.Println("Batch insert completed!")
}
```

### 高级配置示例

```go
func advancedBatchInsert() {
    // 高性能配置
    config := BatchConfig{
        BufferSize:    10000,  // 大缓冲区
        BatchSize:     500,    // 中等批次
        FlushInterval: 50 * time.Millisecond, // 快速刷新
    }
    
    // 创建带监控的执行器
    executor := mysql.NewBatchExecutor(db)
    
    // 添加Prometheus监控
    if prometheusEnabled {
        metricsReporter := NewPrometheusMetricsReporter(prometheusMetrics, "mysql", "high_performance")
        executor = executor.WithMetricsReporter(metricsReporter).(*batchsql.CommonExecutor)
    }
    
    batchSQL := batchsql.NewBatchSQL(ctx, config.BufferSize, config.BatchSize, config.FlushInterval, executor)
    
    // 使用事务控制
    tx, _ := db.Begin()
    defer tx.Rollback()
    
    // 批量操作...
    
    tx.Commit()
}
```

### Redis 批量操作示例

```go
func redisBatchExample() {
    // 连接Redis
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 创建Redis执行器
    executor := redis.NewBatchExecutor(rdb)
    batchSQL := batchsql.NewBatchSQL(ctx, 5000, 500, 50*time.Millisecond, executor)
    
    // Redis Schema（使用命令格式）
    schema := batchsql.NewSchema("redis_cache", batchsql.ConflictReplace,
        "cmd", "key", "value", "ex_flag", "ttl")
    
    // 批量SET操作
    for i := 0; i < 1000; i++ {
        request := batchsql.NewRequest(schema).
            SetString("cmd", "SET").
            SetString("key", fmt.Sprintf("user:%d", i)).
            SetString("value", fmt.Sprintf(`{"id":%d,"name":"User_%d"}`, i, i)).
            SetString("ex_flag", "EX").
            SetInt64("ttl", 3600) // 1小时TTL
        
        batchSQL.Submit(ctx, request)
    }
}
```

## ⚙️ 配置参数指南

### 性能调优参数

| 参数 | 推荐值 | 说明 |
|------|--------|------|
| **BufferSize** | 1000-10000 | 内存缓冲区大小，影响内存使用 |
| **BatchSize** | 100-1000 | 单次批处理大小，影响网络效率 |
| **FlushInterval** | 50ms-1s | 刷新间隔，影响延迟 |

### 数据库特定建议

#### MySQL
- BufferSize: 5000-10000
- BatchSize: 200-500
- FlushInterval: 100ms

#### PostgreSQL  
- BufferSize: 5000-10000
- BatchSize: 200-500
- FlushInterval: 100ms

#### SQLite
- BufferSize: 1000-2000
- BatchSize: 50-200
- FlushInterval: 200ms

#### Redis
- BufferSize: 5000-20000
- BatchSize: 500-2000
- FlushInterval: 50ms

## 🔍 错误处理

### 常见错误类型

```go
// 连接错误
if err := db.Ping(); err != nil {
    log.Fatal("Database connection failed:", err)
}

// 提交错误
if err := batchSQL.Submit(ctx, request); err != nil {
    log.Printf("Submit failed: %v", err)
    // 实现重试逻辑
}

// 批处理错误
// 通过MetricsReporter监控失败率
```

### 最佳实践

1. **连接池配置**
```go
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(50)
db.SetConnMaxLifetime(time.Hour)
```

2. **上下文控制**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

3. **优雅关闭**
```go
defer batchSQL.Close() // 确保所有数据都被刷新
```

## 📚 相关文档

- [EXAMPLES.md](EXAMPLES.md) - 更多使用示例
- [MONITORING_GUIDE.md](MONITORING_GUIDE.md) - 监控配置
- [PERFORMANCE_TUNING.md](PERFORMANCE_TUNING.md) - 性能优化
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - 问题排查

---

💡 **提示**：API设计遵循链式调用模式，支持流畅的代码编写。建议结合具体使用场景选择合适的配置参数。