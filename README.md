# BatchSQL - 通用批量数据库操作框架

## 🚀 项目简介

BatchSQL 是一个现代化的 Go 语言批量数据库操作框架，支持多种数据库类型的统一操作接口。

## ✨ 核心特性

- **多数据库支持**: MySQL、PostgreSQL、Redis、MongoDB
- **统一接口**: 所有数据库使用相同的操作方式
- **类型安全**: 强类型的 Schema 和 Request 系统
- **可扩展架构**: 基于接口的插件化设计
- **内置监控**: 指标收集和健康检查
- **冲突处理**: 支持多种数据冲突策略

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
    
    // 创建 MySQL 驱动
    mysqlDriver := drivers.NewSQLDriver("mysql")
    
    // 创建 Schema
    schema := batchsql.NewSchema("users", batchsql.ConflictStrategyReplace, mysqlDriver, "id", "name", "email")
    
    // 创建请求
    request := batchsql.NewRequestFromInterface(schema)
    request.Set("id", 1)
    request.Set("name", "John Doe")
    request.Set("email", "john@example.com")
    
    // 执行操作
    ctx := context.Background()
    if err := client.ExecuteWithSchema(ctx, schema, []map[string]interface{}{
        {"id": 1, "name": "John", "email": "john@example.com"},
    }); err != nil {
        log.Fatal(err)
    }
    
    log.Println("操作成功完成!")
}
```

## 🏗️ 架构设计

### 核心组件

1. **SchemaInterface**: 定义数据结构和约束
2. **DatabaseDriver**: 数据库驱动接口
3. **BatchCommand**: 批量操作命令
4. **Request**: 数据请求对象
5. **Client**: 统一客户端接口

### 支持的数据库

- **MySQL/PostgreSQL**: 通过 SQL 驱动
- **Redis**: 通过 Redis 驱动  
- **MongoDB**: 通过 MongoDB 驱动

### 冲突策略

- `ConflictStrategyIgnore`: 忽略冲突
- `ConflictStrategyReplace`: 替换冲突数据
- `ConflictStrategyUpdate`: 更新冲突数据

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
    └── demo.go
```

## 🎯 设计原则

1. **接口驱动**: 所有组件基于接口设计，高度可扩展
2. **类型安全**: 编译时错误检查，运行时类型验证
3. **统一API**: 不同数据库使用相同的操作方式
4. **性能优化**: 批量操作，连接池管理
5. **可观测性**: 内置指标收集和健康检查

## 🔮 扩展示例

添加新数据库支持只需实现 `DatabaseDriver` 接口：

```go
type CustomDriver struct{}

func (d *CustomDriver) GetName() string {
    return "custom"
}

func (d *CustomDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    // 实现自定义命令生成逻辑
    return &CustomCommand{}, nil
}

func (d *CustomDriver) SupportedConflictStrategies() []ConflictStrategy {
    return []ConflictStrategy{ConflictStrategyIgnore, ConflictStrategyReplace}
}

func (d *CustomDriver) ValidateSchema(schema SchemaInterface) error {
    // 实现自定义验证逻辑
    return nil
}
```

## 📊 性能特性

- **批量处理**: 支持大批量数据操作
- **连接池**: 自动管理数据库连接
- **并发安全**: 线程安全的操作
- **内存优化**: 流式处理大数据集

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

---

**BatchSQL - 让批量数据库操作变得简单而强大！** 🎉