# BatchSQL

一个高性能的 Go 批量 SQL 处理库，支持多种数据库类型和冲突处理策略。

## 功能特性

### 🚀 核心功能
- **批量处理**：使用 `gopipeline.StandardPipeline` 进行高效的批量数据处理
- **多数据库支持**：支持 MySQL、PostgreSQL、SQLite
- **冲突处理策略**：支持跳过、覆盖、更新三种冲突处理方式
- **类型安全**：提供类型化的列操作方法
- **智能聚合**：按 schema 指针自动聚合相同类型的请求

### 🎯 设计亮点
- **指针传递优化**：使用指针传递减少内存复制，提高性能
- **并发安全**：支持并发提交请求，自动按 schema 分组处理
- **灵活配置**：支持自定义缓冲区大小、刷新大小和刷新间隔
- **测试友好**：提供模拟执行器用于测试

## 快速开始

### 安装

```bash
go get github.com/rushairer/batchsql
```

### 基本使用

```go
package main

import (
    "context"
    "time"
    "github.com/rushairer/batchsql"
)

func main() {
    ctx := context.Background()
    
    // 创建带模拟执行器的 BatchSQL 实例
    batch, _ := batchsql.NewBatchSQLWithMock(ctx, 100, 10, time.Second)
    defer batch.Close()

    // 定义 schema
    schema := batchsql.NewSchema(
        "users",                    // 表名
        batchsql.ConflictIgnore,   // 冲突策略
        batchsql.MySQL,            // 数据库类型
        "id", "name", "email",     // 列名
    )

    // 创建并提交请求
    request := batchsql.NewRequest(schema).
        SetInt64("id", 1).
        SetString("name", "John").
        SetString("email", "john@example.com")

    if err := batch.Submit(ctx, request); err != nil {
        panic(err)
    }
}
```

## 详细功能

### 支持的数据库类型

```go
type DatabaseType int

const (
    MySQL      DatabaseType = iota // MySQL
    PostgreSQL                     // PostgreSQL
    SQLite                         // SQLite
)
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

### 使用真实数据库连接

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    ctx := context.Background()
    batch := batchsql.NewBatchSQLWithDB(ctx, db, 1000, 100, 5*time.Second)
    defer batch.Close()

    // 监听错误
    go func() {
        errorChan := batch.ErrorChan(10)
        for err := range errorChan {
            log.Printf("Batch processing error: %v", err)
        }
    }()

    // 使用 batch...
}
```

### 批量处理不同类型的数据

```go
// 创建不同的 schema
mysqlSchema := batchsql.NewSchema("users", batchsql.ConflictIgnore, batchsql.MySQL, "id", "name")
postgresSchema := batchsql.NewSchema("products", batchsql.ConflictUpdate, batchsql.PostgreSQL, "id", "name", "price")

// 提交不同类型的请求
userRequest := batchsql.NewRequest(mysqlSchema).SetInt64("id", 1).SetString("name", "User1")
productRequest := batchsql.NewRequest(postgresSchema).SetInt64("id", 1).SetString("name", "Product1").SetFloat64("price", 99.99)

batch.Submit(ctx, userRequest)
batch.Submit(ctx, productRequest)

// 系统会自动按 schema 分组处理
```

## 性能优化

### 内存效率
- 使用指针传递 `StandardPipeline[*Request]` 而非值传递，减少内存复制
- 智能聚合相同 schema 的请求，减少数据库连接次数
- 支持对象池模式（可扩展）

### 并发处理
- 支持多 goroutine 并发提交请求
- 自动按 schema 指针聚合，确保相同配置的请求批量处理
- 异步处理，不阻塞主线程

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

## 架构设计

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Application   │───▶│    BatchSQL      │───▶│  gopipeline     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ BatchExecutor    │    │  Flush Function │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ BatchProcessor   │    │ Schema Grouping │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │   Database       │    │   SQL Generation│
                       └──────────────────┘    └─────────────────┘
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License