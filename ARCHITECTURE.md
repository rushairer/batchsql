# BatchSQL 架构重构：基于接口的可扩展设计

## 🎯 重构目标

将原有的硬编码数据库支持重构为基于接口的可扩展架构，支持 SQL 数据库、Redis、MongoDB 等多种数据存储类型。

## 🏗️ 新架构设计

### 核心接口层

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
    Clone() SchemaInterface
}

// 批量命令接口
type BatchCommand interface {
    GetCommandType() string
    GetCommand() interface{}
    GetParameters() []interface{}
    GetMetadata() map[string]interface{}
}
```

### 驱动实现层

#### 1. SQL 数据库驱动
- **MySQLDriver**: 支持 `INSERT IGNORE`、`REPLACE INTO`、`ON DUPLICATE KEY UPDATE`
- **PostgreSQLDriver**: 支持 `ON CONFLICT DO NOTHING`、`ON CONFLICT DO UPDATE`
- **SQLiteDriver**: 支持 `INSERT OR IGNORE`、`INSERT OR REPLACE`

#### 2. Redis 驱动
- **RedisDriver**: 基础 Redis 操作（HSET、SET）
- **RedisHashDriver**: Redis Hash 操作
- **RedisSetDriver**: Redis Set 操作

#### 3. MongoDB 驱动
- **MongoDBDriver**: 标准 MongoDB 集合操作
- **MongoTimeSeriesDriver**: MongoDB 时间序列集合

## 🚀 架构优势

### 1. **高度可扩展**
```go
// 添加新数据库只需实现 DatabaseDriver 接口
type ElasticsearchDriver struct{}

func (d *ElasticsearchDriver) GetName() string {
    return "elasticsearch"
}

func (d *ElasticsearchDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    // 生成 Elasticsearch bulk API 命令
}
```

### 2. **统一的使用方式**
```go
// 不同数据库使用相同的 API
mysqlSchema := batchsql.NewUniversalSchema("users", batchsql.ConflictUpdate, mysqlDriver, "id", "name")
redisSchema := batchsql.NewUniversalSchema("sessions", batchsql.ConflictReplace, redisDriver, "key", "value")
mongoSchema := batchsql.NewUniversalSchema("products", batchsql.ConflictIgnore, mongoDriver, "_id", "name")

// 统一的请求创建方式
request := batchsql.NewRequestFromInterface(schema).SetString("name", "value")
```

### 3. **灵活的冲突策略**
每个驱动可以定义自己支持的冲突策略：
```go
// MySQL 支持所有策略
mysql.SupportedConflictStrategies() // [ConflictIgnore, ConflictReplace, ConflictUpdate]

// Redis 主要支持替换
redis.SupportedConflictStrategies() // [ConflictReplace, ConflictIgnore]

// MongoDB 支持忽略和更新
mongo.SupportedConflictStrategies() // [ConflictIgnore, ConflictUpdate]
```

### 4. **类型安全的命令生成**
```go
// SQL 命令
type SQLCommand struct {
    sql        string
    parameters []interface{}
    metadata   map[string]interface{}
}

// Redis 命令
type RedisCommand struct {
    commands [][]interface{} // 多个 Redis 命令
    metadata map[string]interface{}
}

// MongoDB 命令
type MongoCommand struct {
    operations []interface{} // MongoDB 操作数组
    metadata   map[string]interface{}
}
```

## 📊 性能对比

### 旧架构 vs 新架构

| 特性 | 旧架构 | 新架构 |
|------|--------|--------|
| 数据库支持 | 硬编码 3 种 | 接口化，无限扩展 |
| 添加新数据库 | 修改核心代码 | 实现接口即可 |
| 代码复用 | 低 | 高 |
| 测试复杂度 | 高 | 低（Mock 接口） |
| 维护成本 | 高 | 低 |

### 内存使用优化
```go
// 旧架构：每种数据库类型都有独立的处理逻辑
switch dbType {
case MySQL: // 重复的处理逻辑
case PostgreSQL: // 重复的处理逻辑
case SQLite: // 重复的处理逻辑
}

// 新架构：统一的处理流程
command, err := driver.GenerateBatchCommand(schema, requests)
executor.ExecuteBatch(ctx, []BatchCommand{command})
```

## 🔧 扩展示例

### 添加 Elasticsearch 支持

```go
type ElasticsearchDriver struct {
    indexName string
}

func (d *ElasticsearchDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    var operations []interface{}
    
    for _, request := range requests {
        // 生成 Elasticsearch bulk 操作
        switch schema.GetConflictStrategy() {
        case ConflictIgnore:
            operations = append(operations, map[string]interface{}{
                "create": map[string]interface{}{
                    "_index": schema.GetIdentifier(),
                    "_id":    request.GetString("id"),
                },
            })
        case ConflictUpdate:
            operations = append(operations, map[string]interface{}{
                "index": map[string]interface{}{
                    "_index": schema.GetIdentifier(),
                    "_id":    request.GetString("id"),
                },
            })
        }
        operations = append(operations, request.Columns())
    }
    
    return &ElasticsearchCommand{
        operations: operations,
        metadata: map[string]interface{}{
            "index":      schema.GetIdentifier(),
            "batch_size": len(requests),
        },
    }, nil
}
```

### 添加 ClickHouse 支持

```go
type ClickHouseDriver struct{}

func (d *ClickHouseDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    // 生成 ClickHouse INSERT 语句
    columns := strings.Join(schema.GetColumns(), ", ")
    
    var values []string
    var parameters []interface{}
    
    for _, request := range requests {
        placeholders := make([]string, len(schema.GetColumns()))
        for i := range placeholders {
            placeholders[i] = "?"
            parameters = append(parameters, request.GetOrderedValues()[i])
        }
        values = append(values, "("+strings.Join(placeholders, ", ")+")")
    }
    
    sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", 
        schema.GetIdentifier(), columns, strings.Join(values, ", "))
    
    return &SQLCommand{
        sql:        sql,
        parameters: parameters,
        metadata: map[string]interface{}{
            "table":      schema.GetIdentifier(),
            "batch_size": len(requests),
            "driver":     "clickhouse",
        },
    }, nil
}
```

## 🧪 测试策略

### 接口测试
```go
func TestDatabaseDriver(t *testing.T) {
    drivers := []DatabaseDriver{
        drivers.NewMySQLDriver(),
        drivers.NewRedisDriver(),
        drivers.NewMongoDBDriver(),
    }
    
    for _, driver := range drivers {
        t.Run(driver.GetName(), func(t *testing.T) {
            // 统一的接口测试
            testDriverInterface(t, driver)
        })
    }
}
```

### Mock 驱动
```go
type MockDriver struct {
    name       string
    strategies []ConflictStrategy
}

func (d *MockDriver) GenerateBatchCommand(schema SchemaInterface, requests []*Request) (BatchCommand, error) {
    return &MockCommand{
        commandType: "MOCK",
        command:     fmt.Sprintf("MOCK_%s", strings.ToUpper(d.name)),
    }, nil
}
```

## 📈 迁移指南

### 从旧架构迁移

```go
// 旧代码
schema := batchsql.NewSchema("users", batchsql.ConflictUpdate)
request := batchsql.NewRequest(schema)

// 新代码
driver := drivers.NewMySQLDriver()
schema := batchsql.NewUniversalSchema("users", batchsql.ConflictUpdate, driver, "id", "name")
request := batchsql.NewRequestFromInterface(schema)
```

### 渐进式迁移
1. **第一阶段**: 保持旧 API 兼容，内部使用新架构
2. **第二阶段**: 提供新 API，标记旧 API 为 deprecated
3. **第三阶段**: 移除旧 API，完全使用新架构

## 🎉 总结

新架构通过接口抽象实现了：

✅ **可扩展性**: 轻松添加新数据库支持  
✅ **一致性**: 统一的 API 和使用方式  
✅ **可测试性**: 接口化设计便于 Mock 和测试  
✅ **可维护性**: 清晰的职责分离  
✅ **性能**: 优化的内存使用和批处理逻辑  

这个重构为 BatchSQL 库的长期发展奠定了坚实的基础，使其能够适应不断变化的数据存储需求。