# BatchSQL - 通用批量数据库操作框架

[![CI](https://github.com/rushairer/batchsql/workflows/CI/badge.svg)](https://github.com/rushairer/batchsql/actions/workflows/ci.yml)
[![Integration Tests](https://github.com/rushairer/batchsql/workflows/Integration%20Tests/badge.svg)](https://github.com/rushairer/batchsql/actions/workflows/integration.yml)
[![CodeQL](https://github.com/rushairer/batchsql/workflows/CodeQL/badge.svg)](https://github.com/rushairer/batchsql/actions/workflows/codeql.yml)
[![codecov](https://codecov.io/gh/rushairer/batchsql/branch/master/graph/badge.svg)](https://codecov.io/gh/rushairer/batchsql)
[![Go Report Card](https://goreportcard.com/badge/github.com/rushairer/batchsql)](https://goreportcard.com/report/github.com/rushairer/batchsql)
[![GoDoc](https://godoc.org/github.com/rushairer/batchsql?status.svg)](https://godoc.org/github.com/rushairer/batchsql)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rushairer/batchsql)](https://github.com/rushairer/batchsql)
[![Release](https://img.shields.io/github/v/release/rushairer/batchsql)](https://github.com/rushairer/batchsql/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GitHub issues](https://img.shields.io/github/issues/rushairer/batchsql)](https://github.com/rushairer/batchsql/issues)
[![GitHub stars](https://img.shields.io/github/stars/rushairer/batchsql)](https://github.com/rushairer/batchsql/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/rushairer/batchsql)](https://github.com/rushairer/batchsql/network)

## 🚀 项目简介

BatchSQL 是一个现代化的 Go 语言批量数据库操作框架，基于接口驱动设计，支持多种数据库类型的统一操作。

## ✨ 核心特性

- **🔌 多数据库支持**: MySQL、PostgreSQL、Redis、MongoDB
- **🎯 统一接口**: 所有数据库使用相同的操作方式
- **🛡️ 类型安全**: 强类型的 Schema 和 Request 系统
- **🔧 可扩展架构**: 基于接口的插件化设计
- **📊 内置监控**: 指标收集和健康检查
- **⚡ 高性能**: 批量处理和连接池管理
- **🔄 冲突处理**: 支持多种数据冲突策略

## 📦 快速开始

### 安装

```bash
go get github.com/rushairer/batchsql
```

### 基本使用

```go
package main

import (
    "context"
    "log"
    
    "github.com/rushairer/batchsql"
    "github.com/rushairer/batchsql/drivers"
)

func main() {
    // 创建客户端
    client := batchsql.NewClient()
    defer client.Close()
    
    // 创建 MySQL 驱动
    mysqlDriver := drivers.NewMySQLDriver()
    
    // 创建 Schema
    schema := client.CreateSchema(
        "users",                        // 表名
        batchsql.ConflictUpdate,        // 冲突策略
        mysqlDriver,                    // 驱动
        "id", "name", "email",          // 列名
    )
    
    // 准备数据
    data := []map[string]interface{}{
        {"id": 1, "name": "Alice", "email": "alice@example.com"},
        {"id": 2, "name": "Bob", "email": "bob@example.com"},
    }
    
    // 执行批量操作
    ctx := context.Background()
    if err := client.ExecuteWithSchema(ctx, schema, data); err != nil {
        log.Fatal(err)
    }
    
    log.Println("操作成功完成!")
}
```

## 🏗️ 架构设计

### 核心组件

```go
// 数据库驱动接口
type DatabaseDriver interface {
    GetName() string
    GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error)
    SupportedConflictStrategies() []ConflictStrategy
    ValidateSchema(schema SchemaInterface) error
}

// Schema 接口
type SchemaInterface interface {
    GetIdentifier() string
    GetConflictStrategy() ConflictStrategy
    GetColumns() []string
    GetDatabaseDriver() DatabaseDriver
    Validate() error
}
```

### 支持的数据库

| 数据库 | 驱动 | 冲突策略支持 |
|--------|------|-------------|
| **MySQL** | `MySQLDriver` | IGNORE, REPLACE, UPDATE |
| **PostgreSQL** | `PostgreSQLDriver` | IGNORE, UPDATE |
| **Redis** | `RedisDriver` | IGNORE, REPLACE |
| **MongoDB** | `MongoDBDriver` | IGNORE, UPDATE |

### 冲突策略

- `ConflictIgnore`: 忽略冲突数据
- `ConflictReplace`: 替换冲突数据  
- `ConflictUpdate`: 更新冲突数据

## 📊 多数据库示例

### MySQL 操作

```go
mysqlDriver := drivers.NewMySQLDriver()
userSchema := client.CreateSchema("users", batchsql.ConflictUpdate, mysqlDriver, 
    "id", "name", "email", "created_at")

userData := []map[string]interface{}{
    {
        "id":         1,
        "name":       "Alice",
        "email":      "alice@example.com",
        "created_at": time.Now(),
    },
}

client.ExecuteWithSchema(ctx, userSchema, userData)
```

### Redis 操作

```go
redisDriver := drivers.NewRedisDriver()
sessionSchema := client.CreateSchema("session", batchsql.ConflictReplace, redisDriver,
    "user_id", "token", "expires_at")

sessionData := []map[string]interface{}{
    {
        "user_id":    "user_1",
        "token":      "token_abc123",
        "expires_at": time.Now().Add(24 * time.Hour),
    },
}

client.ExecuteWithSchema(ctx, sessionSchema, sessionData)
```

### MongoDB 操作

```go
mongoDriver := drivers.NewMongoDBDriver()
productSchema := client.CreateSchema("products", batchsql.ConflictUpdate, mongoDriver,
    "_id", "name", "price", "category")

productData := []map[string]interface{}{
    {
        "_id":      "product_1",
        "name":     "Laptop",
        "price":    999.99,
        "category": "electronics",
    },
}

client.ExecuteWithSchema(ctx, productSchema, productData)
```

## 🔧 高级功能

### 监控集成

BatchSQL支持可选的监控功能，可以轻松集成Prometheus等监控系统：

```go
// 实现监控报告器接口
type PrometheusReporter struct {
    duration *prometheus.HistogramVec
    total    *prometheus.CounterVec
}

func (p *PrometheusReporter) ReportBatchExecution(ctx context.Context, metrics batchsql.BatchMetrics) {
    status := "success"
    if metrics.Error != nil {
        status = "error"
    }
    
    p.duration.WithLabelValues(metrics.Driver, metrics.Table).Observe(metrics.Duration.Seconds())
    p.total.WithLabelValues(metrics.Driver, metrics.Table, status).Inc()
}

// 使用监控
client := batchsql.NewClient().WithMetricsReporter(prometheusReporter)
```

### 监控数据

`BatchMetrics` 包含以下监控数据：
- `Driver`: 数据库驱动名称
- `Table`: 表名/集合名
- `BatchSize`: 批量大小
- `Duration`: 执行时长
- `Error`: 错误信息（如果有）
- `StartTime`: 开始时间

## 🎨 扩展新数据库

添加新数据库支持只需实现 `DatabaseDriver` 接口：

```go
type ElasticsearchDriver struct{}

func (d *ElasticsearchDriver) GetName() string {
    return "elasticsearch"
}

func (d *ElasticsearchDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    // 实现 Elasticsearch bulk API 命令生成
    var operations []interface{}
    
    for _, request := range requests {
        switch schema.GetConflictStrategy() {
        case ConflictIgnore:
            operations = append(operations, map[string]interface{}{
                "create": map[string]interface{}{
                    "_index": schema.GetIdentifier(),
                    "_id":    request.Get("id"),
                },
            })
        case ConflictUpdate:
            operations = append(operations, map[string]interface{}{
                "index": map[string]interface{}{
                    "_index": schema.GetIdentifier(),
                    "_id":    request.Get("id"),
                },
            })
        }
        operations = append(operations, request.GetOrderedValues())
    }
    
    return &ElasticsearchCommand{
        operations: operations,
        metadata: map[string]interface{}{
            "driver": "elasticsearch",
            "index":  schema.GetIdentifier(),
        },
    }, nil
}

func (d *ElasticsearchDriver) SupportedConflictStrategies() []ConflictStrategy {
    return []ConflictStrategy{ConflictIgnore, ConflictUpdate}
}

func (d *ElasticsearchDriver) ValidateSchema(schema SchemaInterface) error {
    return nil
}
```

## 📁 项目结构

```
batchsql/
├── batchsql.go          # 主客户端
├── interfaces.go        # 核心接口定义
├── universal_schema.go  # Schema 实现
├── request.go          # 请求对象
├── types.go            # 类型定义
├── drivers/            # 数据库驱动
│   ├── sql_driver.go
│   ├── redis_driver.go
│   └── mongodb_driver.go
└── examples/           # 使用示例
    └── simple_demo.go
```

## 🎯 设计原则

1. **接口驱动**: 所有组件基于接口设计，高度可扩展
2. **类型安全**: 编译时错误检查，运行时类型验证
3. **统一API**: 不同数据库使用相同的操作方式
4. **性能优化**: 批量操作，连接池管理
5. **可观测性**: 内置指标收集和健康检查

## 🧪 运行示例

```bash
# 运行基础示例
go run examples/simple_demo.go

# 运行Prometheus集成示例
go run examples/prometheus_example.go

# 运行测试
go test ./...
```

## 📈 性能特性

- **批量处理**: 支持大批量数据操作
- **连接池**: 自动管理数据库连接
- **并发安全**: 线程安全的操作
- **内存优化**: 流式处理大数据集
- **高性能**: 微秒级响应时间

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

**BatchSQL - 让批量数据库操作变得简单而强大！** 🎉