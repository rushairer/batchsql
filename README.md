# BatchSQL

一个高性能的 Go 批量 SQL 处理库，基于 `go-pipeline` 实现，支持多种数据库类型和冲突处理策略。

## 🏗️ 架构设计

### 核心组件
```
Application
    ↓
BatchSQL (绑定特定数据库类型)
    ↓
gopipeline (按Schema指针分组)
    ↓
BatchExecutor (数据库特定的执行器)
    ↓
BatchProcessor + SQLDriver (SQL生成和执行)
    ↓
Database Connection
```

### 设计原则
- **一个BatchSQL绑定一个数据库类型** - 避免混合数据库的复杂性
- **Schema专注表结构定义** - 职责单一，可复用性强
- **SQLDriver处理SQL生成** - 数据库特定逻辑集中管理
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
    "time"
    "github.com/rushairer/batchsql"
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
    
    // 2. 创建MySQL BatchSQL实例（默认方式）
    config := batchsql.PipelineConfig{
        BufferSize:    1000,        // 缓冲区大小
        FlushSize:     100,         // 批量刷新大小
        FlushInterval: 5 * time.Second, // 刷新间隔
    }
    batch := batchsql.NewMySQLBatchSQL(ctx, db, config)
    defer batch.Close()

    // 3. 定义 schema（不再需要指定数据库类型）
    userSchema := batchsql.NewSchema(
        "users",                    // 表名
        batchsql.ConflictIgnore,   // 冲突策略
        "id", "name", "email",     // 列名
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

### 测试使用

```go
func TestBatchSQL(t *testing.T) {
    ctx := context.Background()
    
    // 使用模拟执行器进行测试（默认MySQL Driver）
    config := batchsql.PipelineConfig{
        BufferSize:    100,
        FlushSize:     10,
        FlushInterval: time.Second,
    }
    batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)
    defer batch.Close()
    
    // 测试逻辑...
}
```

## 📋 详细功能

### API 设计模式

#### 默认方式（推荐）
```go
// 简单易用，使用全局默认Driver
mysqlBatch := batchsql.NewMySQLBatchSQL(ctx, db, config)
postgresBatch := batchsql.NewPostgreSQLBatchSQL(ctx, db, config)
sqliteBatch := batchsql.NewSQLiteBatchSQL(ctx, db, config)

// 测试时也很简单
batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)
```

#### 自定义方式（扩展支持）
```go
// 支持第三方Driver扩展
customDriver := &MyCustomSQLDriver{}
mysqlBatch := batchsql.NewMySQLBatchSQLWithDriver(ctx, db, config, customDriver)

// 测试时使用特定Driver
batch, mockExecutor := batchsql.NewBatchSQLWithMockDriver(ctx, config, customDriver)
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
userSchema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name", "email")
productSchema := batchsql.NewSchema("products", batchsql.ConflictUpdate, "id", "name", "price")

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
    
    // MySQL
    mysqlDB, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
    mysqlBatch := batchsql.NewMySQLBatchSQL(ctx, mysqlDB, config)
    
    // PostgreSQL
    postgresDB, _ := sql.Open("postgres", "postgres://user:password@localhost/testdb?sslmode=disable")
    postgresBatch := batchsql.NewPostgreSQLBatchSQL(ctx, postgresDB, config)
    
    // SQLite
    sqliteDB, _ := sql.Open("sqlite3", "./test.db")
    sqliteBatch := batchsql.NewSQLiteBatchSQL(ctx, sqliteDB, config)
    
    // 每个BatchSQL处理对应数据库的多个表
    userSchema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name")
    productSchema := batchsql.NewSchema("products", batchsql.ConflictUpdate, "id", "name", "price")
    
    // MySQL处理用户和产品表
    mysqlBatch.Submit(ctx, batchsql.NewRequest(userSchema).SetInt64("id", 1).SetString("name", "User1"))
    mysqlBatch.Submit(ctx, batchsql.NewRequest(productSchema).SetInt64("id", 1).SetString("name", "Product1").SetFloat64("price", 99.99))
}
```

### 第三方扩展示例

```go
// 扩展支持TiDB
type TiDBDriver struct{}

func (d *TiDBDriver) GenerateInsertSQL(schema *batchsql.Schema, batchSize int) string {
    // TiDB特定的批量插入优化
    // 实现SQLDriver接口
}

// 使用自定义Driver
tidbDriver := &TiDBDriver{}
batch := batchsql.NewMySQLBatchSQLWithDriver(ctx, tidbDB, config, tidbDriver)
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

## 测试

运行测试：

```bash
go test -v
```

测试覆盖：
- 基本批量处理功能
- Schema 分组逻辑
- SQL 生成正确性
- 不同数据库类型和冲突策略

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
├── drivers/                 # 数据库驱动目录
│   ├── interfaces.go        # 驱动接口定义
│   ├── mock/                # 模拟驱动（用于测试）
│   │   ├── driver.go        # Mock SQL驱动实现
│   │   └── executor.go      # Mock批量执行器实现
│   ├── mysql/               # MySQL驱动
│   │   ├── driver.go        # MySQL SQL驱动实现
│   │   └── executor.go      # MySQL批量执行器实现
│   ├── postgresql/          # PostgreSQL驱动
│   │   ├── driver.go        # PostgreSQL SQL驱动实现
│   │   └── executor.go      # PostgreSQL批量执行器实现
│   └── sqlite/              # SQLite驱动
│       ├── driver.go        # SQLite SQL驱动实现
│       └── executor.go      # SQLite批量执行器实现
└── test/                    # 测试目录
    └── integration/         # 集成测试
```

## 🔧 架构图

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Application   │───▶│    BatchSQL      │───▶│  gopipeline     │
│                 │    │ (MySQL/PG/SQLite)│    │ (异步批量处理)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ BatchExecutor    │    │  Flush Function │
                       │ (数据库特定)      │    │ (批量刷新逻辑)   │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ BatchProcessor   │    │ Schema Grouping │
                       │ (处理核心逻辑)    │    │ (按表分组聚合)   │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │   SQLDriver      │    │   SQL Generation│
                       │ (数据库特定SQL)   │    │ (批量INSERT语句) │
                       └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   Database       │
                       │ (用户管理连接池)  │
                       └──────────────────┘
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License