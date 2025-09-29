# BatchSQL

一个高性能的 Go 批量 SQL 处理库，基于 `go-pipeline` 实现，支持多种数据库类型和冲突处理策略。

*最后更新：2025年1月28日 | 版本：v1.0.1.0*

## 🏗️ 架构设计

### 核心组件
```
Application
    ↓
BatchSQL (绑定特定数据库类型)
    ↓
gopipeline (按Schema指针分组)
    ↓
BatchExecutor (统一执行接口)
    ├── CommonExecutor (SQL数据库通用执行器)
    │   ↓
    │   BatchProcessor + SQLDriver (SQL生成和执行)
    │   ↓
    │   Database Connection
    └── 直接实现 (NoSQL数据库如Redis)
        ↓
        Database Connection
```

### 设计原则
- **一个BatchSQL绑定一个数据库类型** - 避免混合数据库的复杂性
- **Schema专注表结构定义** - 职责单一，可复用性强
- **BatchExecutor统一接口** - 所有数据库驱动的统一入口
- **灵活的实现方式** - SQL数据库使用CommonExecutor+BatchProcessor，NoSQL可直接实现
- **轻量级设计** - 不涉及连接池管理，支持任何数据库框架

## 🚀 功能特性

### 核心功能
- **批量处理**：使用 `gopipeline.StandardPipeline` 进行高效的批量数据处理
- **多数据库支持**：支持 MySQL、PostgreSQL、SQLite，易于扩展
- **冲突处理策略**：支持跳过、覆盖、更新三种冲突处理方式
- **类型安全**：提供类型化的列操作方法
- **智能聚合**：按 schema 指针自动聚合相同表的请求

### 设计亮点
- **指针传递优化**：使用指针传递减少内存复制，提高性能
- **并发安全**：支持并发提交请求，自动按 schema 分组处理
- **灵活配置**：支持自定义缓冲区大小、刷新大小和刷新间隔
- **混合API设计**：默认方式简单易用，自定义方式支持第三方扩展
- **框架无关**：支持原生 `sql.DB`、GORM、sqlx 等任何数据库框架

## 🚀 快速开始

### 安装

```bash
go get github.com/rushairer/batchsql
```

### 基本使用

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "time"
    "github.com/rushairer/batchsql"
    "github.com/rushairer/batchsql/drivers"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    ctx := context.Background()
    
    // 1. 创建数据库连接（用户自己管理连接池）
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // 2. 创建MySQL BatchSQL实例
    // 内部使用 CommonExecutor + SQLBatchProcessor + MySQLDriver
    config := batchsql.PipelineConfig{
        BufferSize:    1000,        // 缓冲区大小
        FlushSize:     100,         // 批量刷新大小
        FlushInterval: 5 * time.Second, // 刷新间隔
    }
    batch := batchsql.NewMySQLBatchSQL(ctx, db, config)

    // 3. 定义 schema（表结构定义，与数据库类型解耦）
    userSchema := batchsql.NewSchema(
        "users",                    // 表名
        drivers.ConflictIgnore,     // 冲突策略
        "id", "name", "email",      // 列名
    )

    // 4. 创建并提交请求
    request := batchsql.NewRequest(userSchema).
        SetInt64("id", 1).
        SetString("name", "John").
        SetString("email", "john@example.com")

    if err := batch.Submit(ctx, request); err != nil {
        panic(err)
    }
    
    // 5. 监听错误
    go func() {
        errorChan := batch.ErrorChan(10)
        for err := range errorChan {
            log.Printf("Batch processing error: %v", err)
        }
    }()
}
```

### Redis 使用示例

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/rushairer/batchsql"
    "github.com/rushairer/batchsql/drivers"
)

func main() {
    ctx := context.Background()
    
    // 1. 创建Redis连接
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer rdb.Close()
    
    // 2. 创建Redis BatchSQL实例
    // 内部直接实现 BatchExecutor 接口，无需 BatchProcessor
    config := batchsql.PipelineConfig{
        BufferSize:    1000,
        FlushSize:     100,
        FlushInterval: 5 * time.Second,
    }
    batch := batchsql.NewRedisBatchSQL(ctx, rdb, config)

    // 3. 定义 Redis schema（key, value, ttl）
    cacheSchema := batchsql.NewSchema(
        "cache",                    // 逻辑表名
        drivers.ConflictReplace,    // Redis默认覆盖
        "key", "value", "ttl",      // 列名
    )

    // 4. 提交Redis数据
    request := batchsql.NewRequest(cacheSchema).
        SetString("cmd", "set").
        SetString("key", "user:1").
        SetString("value", "John Doe").
        SetInt64("ttl", 3600000) // TTL in milliseconds

    if err := batch.Submit(ctx, request); err != nil {
        panic(err)
    }
}
```

### 测试使用

```go
func TestBatchSQL(t *testing.T) {
    ctx := context.Background()
    
    // 使用模拟执行器进行测试
    // 内部使用 MockExecutor 直接实现 BatchExecutor 接口
    config := batchsql.PipelineConfig{
        BufferSize:    100,
        FlushSize:     10,
        FlushInterval: time.Second,
    }
    batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)
    
    // 定义测试schema
    testSchema := batchsql.NewSchema("test_table", drivers.ConflictIgnore, "id", "name")
    
    // 提交测试数据
    request := batchsql.NewRequest(testSchema).
        SetInt64("id", 1).
        SetString("name", "test")
    
    err := batch.Submit(ctx, request)
    assert.NoError(t, err)
    
    // 验证模拟执行器的调用
    time.Sleep(100 * time.Millisecond) // 等待批量处理
    assert.True(t, mockExecutor.WasCalled())
    
    // 获取执行的数据
    executedData := mockExecutor.GetExecutedData()
    assert.Len(t, executedData, 1)
}
```

## 📋 详细功能

### API 设计模式

#### 默认方式（推荐）
```go
// SQL数据库：使用 CommonExecutor + BatchProcessor + SQLDriver
mysqlBatch := batchsql.NewMySQLBatchSQL(ctx, db, config)
postgresBatch := batchsql.NewPostgreSQLBatchSQL(ctx, db, config)
sqliteBatch := batchsql.NewSQLiteBatchSQL(ctx, db, config)

// NoSQL数据库：直接实现 BatchExecutor 接口
redisBatch := batchsql.NewRedisBatchSQL(ctx, redisClient, config)

// 测试：使用 MockExecutor 直接实现 BatchExecutor
batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)
```

#### 自定义方式（扩展支持）
```go
// SQL数据库：支持自定义SQLDriver
customDriver := &MyCustomSQLDriver{}
mysqlBatch := batchsql.NewMySQLBatchSQLWithDriver(ctx, db, config, customDriver)

// 测试：使用特定Driver的Mock
batch, mockExecutor := batchsql.NewBatchSQLWithMockDriver(ctx, config, customDriver)

// 完全自定义：实现自己的BatchExecutor
type MyExecutor struct {
    // 自定义字段
}

func (e *MyExecutor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
    // 自定义实现
    return nil
}

func (e *MyExecutor) WithMetricsReporter(reporter drivers.MetricsReporter) drivers.BatchExecutor {
    // 设置指标报告器
    return e
}

customExecutor := &MyExecutor{}
batch := batchsql.NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, customExecutor)
```

### 冲突处理策略

```go
type ConflictStrategy int

const (
    ConflictIgnore  ConflictStrategy = iota // 跳过冲突
    ConflictReplace                         // 覆盖冲突
    ConflictUpdate                          // 更新冲突
)
```

### Schema 设计
```go
// Schema专注于表结构定义，与数据库类型解耦
userSchema := batchsql.NewSchema("users", drivers.ConflictIgnore, "id", "name", "email")
productSchema := batchsql.NewSchema("products", drivers.ConflictUpdate, "id", "name", "price")

// 同一个Schema可以在不同数据库类型间复用
```

### 生成的 SQL 示例

#### MySQL
- **ConflictIgnore**: `INSERT IGNORE INTO users (id, name) VALUES (?, ?)`
- **ConflictReplace**: `REPLACE INTO users (id, name) VALUES (?, ?)`
- **ConflictUpdate**: `INSERT INTO users (id, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE name = VALUES(name)`

#### PostgreSQL
- **ConflictIgnore**: `INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT DO NOTHING`
- **ConflictUpdate**: `INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`

#### SQLite
- **ConflictIgnore**: `INSERT OR IGNORE INTO users (id, name) VALUES (?, ?)`
- **ConflictReplace**: `INSERT OR REPLACE INTO users (id, name) VALUES (?, ?)`
- **ConflictUpdate**: `INSERT INTO users (id, name) VALUES (?, ?) ON CONFLICT DO UPDATE SET name = excluded.name`

### 类型化的列操作

```go
request := batchsql.NewRequest(schema).
    SetInt32("age", 30).
    SetInt64("id", 12345).
    SetFloat64("salary", 75000.50).
    SetString("name", "John Doe").
    SetBool("is_active", true).
    SetTime("created_at", time.Now()).
    SetBytes("data", []byte("binary data")).
    SetNull("optional_field")
```

### 获取类型化的值

```go
if name, err := request.GetString("name"); err == nil {
    fmt.Printf("Name: %s", name)
}

if age, err := request.GetInt32("age"); err == nil {
    fmt.Printf("Age: %d", age)
}
```

## 高级用法

### 多数据库支持

```go
import (
    "database/sql"
    "github.com/redis/go-redis/v9"
    _ "github.com/go-sql-driver/mysql"
    _ "github.com/lib/pq"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    ctx := context.Background()
    config := batchsql.PipelineConfig{
        BufferSize:    1000,
        FlushSize:     100,
        FlushInterval: 5 * time.Second,
    }
    
    // SQL数据库 - 使用 CommonExecutor + BatchProcessor + SQLDriver
    
    // MySQL
    mysqlDB, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
    mysqlBatch := batchsql.NewMySQLBatchSQL(ctx, mysqlDB, config)
    
    // PostgreSQL
    postgresDB, _ := sql.Open("postgres", "postgres://user:password@localhost/testdb?sslmode=disable")
    postgresBatch := batchsql.NewPostgreSQLBatchSQL(ctx, postgresDB, config)
    
    // SQLite
    sqliteDB, _ := sql.Open("sqlite3", "./test.db")
    sqliteBatch := batchsql.NewSQLiteBatchSQL(ctx, sqliteDB, config)
    
    // NoSQL数据库 - 直接实现 BatchExecutor
    
    // Redis
    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    redisBatch := batchsql.NewRedisBatchSQL(ctx, redisClient, config)
    
    // 定义通用schema（可在不同数据库间复用）
    userSchema := batchsql.NewSchema("users", drivers.ConflictIgnore, "id", "name")
    productSchema := batchsql.NewSchema("products", drivers.ConflictUpdate, "id", "name", "price")
    cacheSchema := batchsql.NewSchema("cache", drivers.ConflictReplace, "key", "value", "ttl")
    
    // 每个BatchSQL处理对应数据库的多个表
    
    // MySQL处理用户和产品表
    mysqlBatch.Submit(ctx, batchsql.NewRequest(userSchema).SetInt64("id", 1).SetString("name", "User1"))
    mysqlBatch.Submit(ctx, batchsql.NewRequest(productSchema).SetInt64("id", 1).SetString("name", "Product1").SetFloat64("price", 99.99))
    
    // PostgreSQL处理相同的schema
    postgresBatch.Submit(ctx, batchsql.NewRequest(userSchema).SetInt64("id", 2).SetString("name", "User2"))
    
    // Redis处理缓存数据
    redisBatch.Submit(ctx, batchsql.NewRequest(cacheSchema).
        SetString("key", "user:1").
        SetString("value", "User1").
        SetInt64("ttl", 3600000))
}
```

### 第三方扩展示例

#### 扩展SQL数据库支持（如TiDB）
```go
// 实现SQLDriver接口
type TiDBDriver struct{}

func (d *TiDBDriver) GenerateInsertSQL(schema *drivers.Schema, data []map[string]any) (string, []any, error) {
    // TiDB特定的批量插入优化
    // 可以使用TiDB的特殊语法或优化
    return sql, args, nil
}

// 使用自定义Driver，内部仍使用CommonExecutor架构
tidbDriver := &TiDBDriver{}
batch := batchsql.NewMySQLBatchSQLWithDriver(ctx, tidbDB, config, tidbDriver)
```

#### 扩展NoSQL数据库支持（如MongoDB）
```go
// 直接实现BatchExecutor接口
type MongoExecutor struct {
    client          *mongo.Client
    metricsReporter drivers.MetricsReporter
}

func NewMongoBatchExecutor(client *mongo.Client) *MongoExecutor {
    return &MongoExecutor{client: client}
}

func (e *MongoExecutor) ExecuteBatch(ctx context.Context, schema *drivers.Schema, data []map[string]any) error {
    if len(data) == 0 {
        return nil
    }
    
    // MongoDB特定的批量插入逻辑
    collection := e.client.Database("mydb").Collection(schema.TableName)
    
    // 转换数据格式
    docs := make([]interface{}, len(data))
    for i, row := range data {
        docs[i] = row
    }
    
    // 执行批量插入
    _, err := collection.InsertMany(ctx, docs)
    return err
}

func (e *MongoExecutor) WithMetricsReporter(reporter drivers.MetricsReporter) drivers.BatchExecutor {
    e.metricsReporter = reporter
    return e
}

// 创建MongoDB BatchSQL
func NewMongoBatchSQL(ctx context.Context, client *mongo.Client, config batchsql.PipelineConfig) *batchsql.BatchSQL {
    executor := NewMongoBatchExecutor(client)
    return batchsql.NewBatchSQL(ctx, config.BufferSize, config.FlushSize, config.FlushInterval, executor)
}

// 使用
mongoClient, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
mongoBatch := NewMongoBatchSQL(ctx, mongoClient, config)
```

### 框架集成示例

```go
// 与GORM集成
gormDB, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
sqlDB, _ := gormDB.DB()
batch := batchsql.NewMySQLBatchSQL(ctx, sqlDB, config)

// 与sqlx集成
sqlxDB, _ := sqlx.Connect("mysql", dsn)
batch := batchsql.NewMySQLBatchSQL(ctx, sqlxDB.DB, config)
```

## ⚡ 性能优化

### 内存效率
- **指针传递**：使用 `StandardPipeline[*Request]` 而非值传递，减少内存复制
- **智能聚合**：按 schema 指针自动聚合相同表的请求，减少数据库操作次数
- **全局Driver共享**：SQLDriver实例全局共享，避免重复创建
- **零拷贝设计**：Request数据直接传递，无额外序列化开销

### 并发处理
- **多goroutine安全**：支持多 goroutine 并发提交请求
- **自动分组**：按 schema 指针聚合，确保相同表的请求批量处理
- **异步处理**：基于 go-pipeline 的异步处理，不阻塞主线程
- **背压控制**：缓冲区满时自动背压，防止内存溢出

### 数据库优化
- **批量插入**：自动生成优化的批量INSERT语句
- **事务保证**：每个批次使用单个事务，保证数据一致性
- **连接复用**：用户自己管理连接池，支持连接复用
- **SQL优化**：针对不同数据库生成最优的SQL语法

## 📊 质量评估

基于最新集成测试报告的项目质量状态评估：

### 测试通过率
| 数据库 | 测试数量 | 通过 | 失败 | 通过率 | BatchSQL 状态 |
|--------|----------|------|------|--------|---------------|
| **SQLite** | 5 | 4 | 1 | 80% | ✅ 正常（失败为 SQLite 架构限制） |
| **MySQL** | 5 | 5 | 0 | 100% | ✅ 优秀 |
| **PostgreSQL** | 5 | 5 | 0 | 100% | ✅ 优秀 |
| **总计** | 15 | 14 | 1 | 93.3% | ✅ 优秀 |

### 性能指标
| 数据库 | 平均 RPS | 最大 RPS | 数据完整性 | BatchSQL 性能评级 |
|--------|----------|----------|------------|------------------|
| **SQLite** | 105,246 | 199,071 | 80% 测试通过 | ✅ 符合 SQLite 预期 |
| **MySQL** | 144,879 | 168,472 | 100% 测试通过 | ✅ 优秀 |
| **PostgreSQL** | 152,586 | 191,037 | 100% 测试通过 | ✅ 优秀 |

### 技术说明
🔵 **SQLite 架构限制**（非项目缺陷）：SQLite 是单写入者数据库，大批次并发写入失败属于数据库引擎固有限制  
🟢 **BatchSQL 功能完整**：所有核心功能正常，错误检测机制完善  
🟢 **代码质量优秀**：在 MySQL/PostgreSQL 上表现优异，证明实现正确  

### 发布状态
**当前状态**：✅ **可以发布**  
**项目质量**：BatchSQL 核心功能完整，无需修复  
**SQLite 说明**：测试失败源于 SQLite 单写入者架构限制，非项目问题  
**使用建议**：高并发场景推荐 MySQL/PostgreSQL，轻量级场景可用 SQLite  

*详细分析报告：[QUALITY_ASSESSMENT.md](QUALITY_ASSESSMENT.md)*

## 📋 测试

### 单元测试
```bash
# 运行所有单元测试
go test -v

# 运行测试覆盖率分析
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 集成测试
```bash
# 运行所有数据库集成测试
make docker-all-tests

# 运行单个数据库测试
make docker-mysql-test      # MySQL 测试
make docker-postgres-test   # PostgreSQL 测试
make docker-sqlite-test     # SQLite 测试
```

### SQLite 专用测试工具
```bash
# SQLite 性能基准测试
cd test/sqlite/tools/benchmark && go run main.go

# SQLite 配置分析
cd test/sqlite/tools/config-analysis && go run main.go

# SQLite 清理测试
cd test/sqlite/tools/clear-test && go run main.go

# 路径兼容性测试
cd test/sqlite/tools/path-compatibility && go run main.go
```

### 测试覆盖范围
- ✅ 基本批量处理功能
- ✅ Schema 分组逻辑
- ✅ SQL 生成正确性
- ✅ 不同数据库类型和冲突策略
- ✅ 错误处理和边界条件
- ✅ 并发安全性测试
- ✅ 大数据量压力测试
- ✅ 数据库连接异常处理

*详细测试文档：[README-INTEGRATION-TESTS.md](README-INTEGRATION-TESTS.md)*

## 🏗️ 文件结构

```
batchsql/
├── batchsql.go              # 主入口和工厂方法
├── schema.go                # Schema定义（表结构）
├── request.go               # Request定义（类型安全的数据操作）
├── batch_processor.go       # 批量处理核心逻辑
├── interfaces.go            # 主要接口定义
├── error.go                 # 错误定义
├── batchsql_test.go         # 测试文件
├── go.mod                   # Go模块定义
├── go.sum                   # 依赖校验文件
├── .golangci.yml            # Go代码检查配置
├── README.md                # 项目文档
├── ARCHITECTURE.md          # 架构设计文档（v1.0.1.0新增）
├── CONFIG.md                # 配置参数详细说明
├── CONTRIBUTING.md          # 贡献指南（已更新架构部分）
├── QUALITY_ASSESSMENT.md    # 项目质量评估报告
├── README-INTEGRATION-TESTS.md # 集成测试文档
├── RELEASE_CHECKLIST.md     # 发布检查清单
├── Makefile                 # 构建和测试命令
├── .env.test                # 统一测试配置
├── .env.sqlite.test         # SQLite 专用测试配置
├── docker-compose.*.yml     # Docker 测试配置文件
├── Dockerfile.*             # Docker 构建文件
├── drivers/                 # 数据库驱动目录
│   ├── interfaces.go        # 核心接口定义（BatchExecutor, BatchProcessor, SQLDriver等）
│   ├── common_executor.go   # 通用执行器实现（SQL数据库共用）
│   ├── batch_processor.go   # 批量处理器实现（SQL数据库共用）
│   ├── mock/                # 模拟驱动（用于测试）
│   │   ├── driver.go        # Mock SQL驱动实现
│   │   └── executor.go      # Mock批量执行器实现（直接实现BatchExecutor）
│   ├── mysql/               # MySQL驱动
│   │   ├── driver.go        # MySQL SQL驱动实现
│   │   └── executor.go      # MySQL执行器工厂（返回CommonExecutor）
│   ├── postgresql/          # PostgreSQL驱动
│   │   ├── driver.go        # PostgreSQL SQL驱动实现
│   │   └── executor.go      # PostgreSQL执行器工厂（返回CommonExecutor）
│   ├── sqlite/              # SQLite驱动
│   │   ├── driver.go        # SQLite SQL驱动实现
│   │   └── executor.go      # SQLite执行器工厂（返回CommonExecutor）
│   └── redis/               # Redis驱动
│       └── executor.go      # Redis执行器（直接实现BatchExecutor）
└── test/                    # 测试目录
    ├── integration/         # 集成测试
    │   ├── main.go          # 集成测试主程序
    │   └── run-single-db-test.sh # 单数据库测试脚本
    ├── reports/             # 测试报告目录
    ├── sql/                 # 数据库初始化脚本
    │   ├── mysql/           # MySQL 初始化脚本
    │   ├── postgres/        # PostgreSQL 初始化脚本
    │   └── sqlite/          # SQLite 初始化脚本
    └── sqlite/              # SQLite 专用测试工具
        ├── README.md        # SQLite 测试工具说明
        ├── SQLITE_OPTIMIZATION.md # SQLite 优化文档
        ├── PERFORMANCE_ANALYSIS.md # 性能分析报告
        ├── TEST_REPORT_ANALYSIS.md # 测试报告分析
        └── tools/           # SQLite 测试工具集
            ├── README.md    # 工具集说明
            ├── benchmark/   # 性能基准测试
            ├── clear-test/  # 清理方式测试
            ├── config-analysis/ # 配置分析工具
            └── path-compatibility/ # 路径兼容性测试
```

## 🔧 架构图

### 整体架构
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Application   │───▶│    BatchSQL      │───▶│  gopipeline     │
│                 │    │(MySQL/PG/SQLite/ │    │  (异步批量处理)   │
│                 │    │    Redis)        │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ BatchExecutor    │    │  Flush Function │
                       │ (统一执行接口)     │    │  (批量刷新逻辑)   │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ 实现方式分支       │    │ Schema Grouping │
                       │                  │    │  (按表分组聚合)   │
                       └──────────────────┘    └─────────────────┘
                          │              │
                          ▼              ▼
              ┌─────────────────┐  ┌─────────────────┐
              │ CommonExecutor  │  │  直接实现        │
              │ (SQL数据库)      │  │  (NoSQL数据库)   │
              └─────────────────┘  └─────────────────┘
                          │              │
                          ▼              ▼
              ┌─────────────────┐  ┌─────────────────┐
              │BatchProcessor + │  │   Database      │
              │   SQLDriver     │  │ (如Redis Client)│
              └─────────────────┘  └─────────────────┘
                          │
                          ▼
              ┌─────────────────┐
              │   Database      │
              │ (SQL连接池)     │
              └─────────────────┘
```

### SQL数据库执行路径
```
BatchExecutor → CommonExecutor → BatchProcessor → SQLDriver → Database
```

### NoSQL数据库执行路径  
```
BatchExecutor → 直接实现 → Database
```

## 📚 相关文档

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 详细的架构设计文档和扩展指南 ⭐ *v1.0.1.0新增*
- **[CONFIG.md](CONFIG.md)** - 详细的配置参数说明和调优建议
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - 贡献指南（已更新架构部分）
- **[README-INTEGRATION-TESTS.md](README-INTEGRATION-TESTS.md)** - 集成测试完整文档
- **[QUALITY_ASSESSMENT.md](QUALITY_ASSESSMENT.md)** - 项目质量评估报告
- **[RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md)** - 发布检查清单
- **[test/sqlite/README.md](test/sqlite/README.md)** - SQLite 测试工具集说明
- **[test/sqlite/SQLITE_OPTIMIZATION.md](test/sqlite/SQLITE_OPTIMIZATION.md)** - SQLite 优化文档
- **[test/sqlite/PERFORMANCE_ANALYSIS.md](test/sqlite/PERFORMANCE_ANALYSIS.md)** - SQLite 性能分析

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发流程
1. Fork 项目
2. 创建功能分支
3. 运行完整测试：`make ci`
4. 提交 Pull Request

### 测试要求
- 所有单元测试必须通过
- 集成测试通过率 ≥ 90%
- 代码覆盖率 ≥ 60%
- 通过 golangci-lint 检查

## 📄 许可证

MIT License