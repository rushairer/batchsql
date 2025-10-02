# BatchSQL

一个高性能的 Go 批量 SQL 处理库，基于 `go-pipeline` 实现，支持多种数据库类型和冲突处理策略。

*最后更新：2025年10月2日 | 版本：v1.2.1*

## 🏗️ 架构设计

延伸阅读
- [架构设计详解](docs/development/architecture.md)

### 核心组件
```mermaid
flowchart TB

  %% 子图1：系统级数据流
  subgraph A0[系统级数据流]
    A1[Application] --> A2["BatchSQL<br/>(MySQL/PG/SQLite/Redis)"] --> A3["gopipeline<br/>(异步批量处理)"]

    A2 --> A4["BatchExecutor<br/>(统一执行接口)"]
    A3 --> A5["Flush Function<br/>(批量刷新逻辑)"]

    A4 --> A6[数据库驱动层]
    A5 --> A7["Schema Grouping<br/>(按表分组聚合)"]

    A6 --> A8["SQL数据库<br/>(MySQL/PG/SQLite)"]
    A6 --> A9[Redis数据库]

    A8 --> A10["Database<br/>(SQL连接池)"]
    A9 --> A11["Redis Client<br/>(Redis连接)"]
  end

  %% 子图2：组件分层与驱动路径
  subgraph B0[组件分层与驱动路径]
    B1["BatchExecutor"] --> B2["CommonExecutor<br/>(通用执行器)"]
    B2 --> B3["BatchProcessor + Driver<br/>(操作生成和执行)"]
    B3 --> B4["Database Connection"]

    %% 数据库类型分支
    B4 --> B5[SQL数据库]
    B4 --> B6[NoSQL数据库]

    B5 --> B51[MySQL]
    B5 --> B52[PostgreSQL]
    B5 --> B53[SQLite]

    B6 --> B61[Redis]
  end

  %% 视图之间的对应关系（虚线）
  A4 -. 同一执行器 .-> B1
  A6 -. 连接/驱动 .-> B4
```

### 设计原则
- **一个BatchSQL绑定一个数据库类型** - 避免混合数据库的复杂性
- **Schema专注表结构定义** - 职责单一，可复用性强
- **BatchExecutor统一接口** - 所有数据库驱动的统一入口
- **模块化设计** - 清晰的组件分工，便于维护和扩展
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

延伸阅读
- [API 参考](docs/api/reference.md)
- [配置指南](docs/api/configuration.md)
- [使用示例](docs/guides/examples.md)

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
    // 内部架构：ThrottledBatchExecutor -> SQLBatchProcessor -> MySQLDriver
    config := batchsql.PipelineConfig{
        BufferSize:    1000,        // 缓冲区大小
        FlushSize:     100,         // 批量刷新大小
        FlushInterval: 5 * time.Second, // 刷新间隔
    }
    batch := batchsql.NewMySQLBatchSQL(ctx, db, config)

    // 3. 定义 schema（表结构定义，与数据库类型解耦）
    userSchema := batchsql.NewSchema(
        "users",                    // 表名
        batchsql.ConflictIgnore,     // 冲突策略
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

注意：
- 自 v1.1.1 起，Submit 会在尝试入队前优先检查 ctx.Err()（取消/超时将立即返回，不会进入内部批处理通道）。请在提交前妥善管理 context 生命周期，避免无效提交。

延伸阅读
- [使用示例](docs/guides/examples.md)
- [配置指南](docs/api/configuration.md)

### Redis 使用示例

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/rushairer/batchsql"

)

func main() {
    ctx := context.Background()
    
    // 1. 创建Redis连接
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer rdb.Close()
    
    // 2. 创建Redis BatchSQL实例
    // 内部架构：ThrottledBatchExecutor -> RedisBatchProcessor -> RedisDriver
    config := batchsql.PipelineConfig{
        BufferSize:    1000,
        FlushSize:     100,
        FlushInterval: 5 * time.Second,
    }
    batch := batchsql.NewRedisBatchSQL(ctx, rdb, config)

    // 3. 定义 Redis schema（使用 SETEX 命令格式）
    cacheSchema := batchsql.NewSchema(
        "cache",                    // 逻辑表名
        batchsql.ConflictReplace,    // Redis默认覆盖
        "cmd", "key", "ttl", "value", // SETEX 命令参数顺序
    )

    // 4. 提交Redis数据（SETEX 命令）
    request := batchsql.NewRequest(cacheSchema).
        SetString("cmd", "SETEX").
        SetString("key", "user:1").
        SetInt64("ttl", 3600).      // TTL in seconds
        SetString("value", `{"name":"John Doe","email":"john@example.com"}`)

    if err := batch.Submit(ctx, request); err != nil {
        panic(err)
    }
    
    // 5. 监听错误
    go func() {
        errorChan := batch.ErrorChan(10)
        for err := range errorChan {
            log.Printf("Redis batch processing error: %v", err)
        }
    }()
}
```

### 测试使用

延伸阅读
- [测试指南](docs/guides/testing.md)
- [集成测试指南](docs/guides/integration-tests.md)

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
    testSchema := batchsql.NewSchema("test_table", batchsql.ConflictIgnore, "id", "name")
    
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

## 📡 监控与指标（MetricsReporter）

- 功能：统一上报入队延迟、攒批耗时、执行耗时、批大小、错误计数、执行并发、队列长度、在途批次等关键阶段与状态
- 使用场景：
  - 开箱即用观测（Prometheus + Grafana）
  - 接入自有监控体系（实现自定义 Reporter）
- 配置要点：
  - 默认 NoopMetricsReporter（零开销，未注入时不产生任何观测）
  - 务必在 NewBatchSQL 之前对执行器注入 Reporter（WithMetricsReporter）
  - NewBatchSQL 会尊重已注入的 Reporter，不会覆盖为 Noop

最小示例（Prometheus 快速上手）
```go
pm := integration.NewPrometheusMetrics()
go pm.StartServer(9090)
defer pm.StopServer()

exec := batchsql.NewSQLThrottledBatchExecutorWithDriver(db, driver)
reporter := integration.NewPrometheusMetricsReporter(pm, "postgres", "user_batch")
exec = exec.WithMetricsReporter(reporter).(batchsql.BatchExecutor)

bs := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, exec)
defer bs.Close()
```

延伸阅读
- [监控快速上手（Prometheus + Grafana）](docs/guides/monitoring-quickstart.md)
- [自定义 MetricsReporter 指南](docs/guides/custom-metrics-reporter.md)
- [API 参考（MetricsReporter 小节）](docs/api/reference.md)

#### Prometheus + Grafana 快速监控

BatchSQL 支持 Prometheus 指标收集和 Grafana 可视化，让你能够实时监控性能曲线变化。

- 快速启动监控
```bash
# 使用 Make 命令（推荐）
make monitoring                           # 启动监控环境
make test-integration-with-monitoring     # 启动监控后运行测试
```

- 访问监控界面
  - Grafana 仪表板: http://localhost:3000 (admin/admin)
  - Prometheus 控制台: http://localhost:9091
  - BatchSQL 指标: http://localhost:9090/metrics

- 监控指标
  - 性能指标: RPS、响应时间、批处理时间
  - 资源指标: 内存使用、并发工作线程、活跃连接
  - 质量指标: 数据完整性率、错误率

详细使用说明请参考：[监控指南](docs/guides/monitoring.md)

## 📋 详细功能

延伸阅读
- [API 参考](docs/api/reference.md)
- [使用示例](docs/guides/examples.md)

### API 设计模式

#### 默认方式（推荐）
```go
// SQL数据库
mysqlBatch := batchsql.NewMySQLBatchSQL(ctx, db, config)
postgresBatch := batchsql.NewPostgreSQLBatchSQL(ctx, db, config)
sqliteBatch := batchsql.NewSQLiteBatchSQL(ctx, db, config)

// NoSQL数据库
redisBatch := batchsql.NewRedisBatchSQL(ctx, redisClient, config)

// 测试
batch, mockExecutor := batchsql.NewBatchSQLWithMock(ctx, config)
```

#### 自定义方式（扩展支持）
```go
// SQL数据库：支持自定义SQLDriver
customSQLDriver := &MyCustomSQLDriver{}
mysqlBatch := batchsql.NewMySQLBatchSQLWithDriver(ctx, db, config, customSQLDriver)

// Redis数据库：支持自定义RedisDriver
customRedisDriver := &MyCustomRedisDriver{}
redisBatch := batchsql.NewRedisBatchSQLWithDriver(ctx, redisClient, config, customRedisDriver)

// 测试：使用特定Driver的Mock
batch, mockExecutor := batchsql.NewBatchSQLWithMockDriver(ctx, config, customSQLDriver)

// 完全自定义：实现自己的BatchExecutor
type MyExecutor struct {
    // 自定义字段
}

func (e *MyExecutor) ExecuteBatch(ctx context.Context, schema *batchsql.Schema, data []map[string]any) error {
    // 自定义实现
    return nil
}

func (e *MyExecutor) WithMetricsReporter(reporter batchsql.MetricsReporter) batchsql.BatchExecutor {
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
    
    // SQL数据库
    
    // MySQL
    mysqlDB, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/testdb")
    mysqlBatch := batchsql.NewMySQLBatchSQL(ctx, mysqlDB, config)
    
    // PostgreSQL
    postgresDB, _ := sql.Open("postgres", "postgres://user:password@localhost/testdb?sslmode=disable")
    postgresBatch := batchsql.NewPostgreSQLBatchSQL(ctx, postgresDB, config)
    
    // SQLite
    sqliteDB, _ := sql.Open("sqlite3", "./test.db")
    sqliteBatch := batchsql.NewSQLiteBatchSQL(ctx, sqliteDB, config)
    
    // NoSQL数据库
    
    // Redis
    redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    redisBatch := batchsql.NewRedisBatchSQL(ctx, redisClient, config)
    
    // 定义通用schema（可在不同数据库间复用）
    userSchema := batchsql.NewSchema("users", batchsql.ConflictIgnore, "id", "name")
    productSchema := batchsql.NewSchema("products", batchsql.ConflictUpdate, "id", "name", "price")
    
    // Redis专用schema（SETEX命令格式）
    cacheSchema := batchsql.NewSchema("cache", batchsql.ConflictReplace, "cmd", "key", "ttl", "value")
    
    // 每个BatchSQL处理对应数据库的多个表
    
    // MySQL处理用户和产品表
    mysqlBatch.Submit(ctx, batchsql.NewRequest(userSchema).SetInt64("id", 1).SetString("name", "User1"))
    mysqlBatch.Submit(ctx, batchsql.NewRequest(productSchema).SetInt64("id", 1).SetString("name", "Product1").SetFloat64("price", 99.99))
    
    // PostgreSQL处理相同的schema
    postgresBatch.Submit(ctx, batchsql.NewRequest(userSchema).SetInt64("id", 2).SetString("name", "User2"))
    
    // Redis处理缓存数据（使用SETEX命令）
    redisBatch.Submit(ctx, batchsql.NewRequest(cacheSchema).
        SetString("cmd", "SETEX").
        SetString("key", "user:1").
        SetInt64("ttl", 3600).
        SetString("value", `{"name":"User1","active":true}`))
}
```

### 第三方扩展示例

#### 扩展SQL数据库支持（如TiDB）
```go
// 实现SQLDriver接口
type TiDBDriver struct{}

func (d *TiDBDriver) GenerateInsertSQL(schema *batchsql.Schema, data []map[string]any) (string, []any, error) {
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
    metricsReporter batchsql.MetricsReporter
}

func NewMongoBatchExecutor(client *mongo.Client) *MongoExecutor {
    return &MongoExecutor{client: client}
}

func (e *MongoExecutor) ExecuteBatch(ctx context.Context, schema *batchsql.Schema, data []map[string]any) error {
    if len(data) == 0 {
        return nil
    }
    
    // MongoDB特定的批量插入逻辑
    collection := e.client.Database("mydb").Collection(schema.Name)
    
    // 转换数据格式
    docs := make([]interface{}, len(data))
    for i, row := range data {
        docs[i] = row
    }
    
    // 执行批量插入
    _, err := collection.InsertMany(ctx, docs)
    return err
}

func (e *MongoExecutor) WithMetricsReporter(reporter batchsql.MetricsReporter) batchsql.BatchExecutor {
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

### 可选并发限流示例

```go
// 高级用法：自行构造可限流的执行器，再创建 BatchSQL
db, _ := sql.Open("mysql", dsn)
// 构造 SQL 执行器，并限制同时执行的批次数为 8
executor := batchsql.NewSQLThrottledBatchExecutorWithDriver(db, batchsql.DefaultMySQLDriver).
    WithConcurrencyLimit(8)

// 创建 BatchSQL（管道配置）
cfg := batchsql.PipelineConfig{BufferSize: 5000, FlushSize: 200, FlushInterval: 100 * time.Millisecond}
batch := batchsql.NewBatchSQL(ctx, cfg.BufferSize, cfg.FlushSize, cfg.FlushInterval, executor)
```

说明：
- limit <= 0 时不启用限流，行为等价于默认
- 限流发生在 ExecuteBatch 入口，避免攒批后过度并发压垮数据库
- 指标上报与错误处理逻辑保持一致

### 框架集成示例

延伸阅读
- [使用示例](docs/guides/examples.md)
- [API 参考](docs/api/reference.md)

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

延伸阅读
- [调优最佳实践](docs/guides/tuning.md)
- [性能分析报告](docs/reports/PERFORMANCE_ANALYSIS.md)
- [SQLite 优化建议](docs/reports/SQLITE_OPTIMIZATION.md)


## 📊 质量评估

延伸阅读
- [质量评估方法](docs/development/quality.md)
- [测试报告分析](docs/reports/TEST_REPORT_ANALYSIS.md)

基于最新集成测试报告的项目质量状态评估：

### 测试通过率
| 数据库 | 测试数量 | 通过 | 失败 | 通过率 | BatchSQL 状态 |
|--------|----------|------|------|--------|---------------|
| **SQLite** | 5 | 4 | 1 | 80% | ✅ 正常（失败为 SQLite 架构限制） |
| **MySQL** | 5 | 5 | 0 | 100% | ✅ 优秀 |
| **PostgreSQL** | 5 | 5 | 0 | 100% | ✅ 优秀 |
| **Redis** | 5 | 5 | 0 | 100% | ✅ 优秀（三层架构重构完成） |
| **总计** | 20 | 19 | 1 | 95% | ✅ 优秀 |

### 性能指标
| 数据库 | 平均 RPS | 最大 RPS | 数据完整性 | BatchSQL 性能评级 |
|--------|----------|----------|------------|------------------|
| **SQLite** | 105,246 | 199,071 | 80% 测试通过 | ✅ 符合 SQLite 预期 |
| **MySQL** | 144,879 | 168,472 | 100% 测试通过 | ✅ 优秀 |
| **PostgreSQL** | 152,586 | 191,037 | 100% 测试通过 | ✅ 优秀 |
| **Redis** | 180,000+ | 250,000+ | 100% 测试通过 | ✅ 优秀（三层架构优化） |

### 技术说明
🔵 **SQLite 架构限制**（非项目缺陷）：SQLite 是单写入者数据库，大批次并发写入失败属于数据库引擎固有限制  
🟢 **BatchSQL 功能完整**：所有核心功能正常，错误检测机制完善  
🟢 **代码质量优秀**：在 MySQL/PostgreSQL/Redis 上表现优异，证明实现正确  

### 发布状态
**当前状态**：✅ **可以发布**  
**项目质量**：BatchSQL 核心功能完整，所有数据库驱动稳定可用  
**SQLite 说明**：测试失败源于 SQLite 单写入者架构限制，非项目问题  
**使用建议**：
- 高并发场景推荐 MySQL/PostgreSQL/Redis
- 轻量级场景可用 SQLite
- 缓存场景推荐 Redis（性能优异）

## 📚 文档导航

BatchSQL 提供完整的文档体系，按使用场景分类：

### 🚀 快速开始

- [📖 文档索引](docs/index.md) - 完整的文档导航和索引
- [💡 使用示例](docs/guides/examples.md) - 丰富的代码示例和最佳实践

### 📖 API 文档
- [🚀 API 参考](docs/api/reference.md) - 完整的 API 文档和使用指南
- [⚙️ 配置指南](docs/api/configuration.md) - 详细的配置参数说明

### 📖 用户指南
- [🧪 测试指南](docs/guides/testing.md) - 完整的测试文档和 Redis 测试报告
- [📊 监控指南](docs/guides/monitoring.md) - Prometheus + Grafana 监控系统
- [🔧 故障排除](docs/guides/troubleshooting.md) - 完整的问题诊断和解决方案
- [🔗 集成测试](docs/guides/integration-tests.md) - 集成测试详细说明
- [🛠 调优最佳实践](docs/guides/tuning.md) - 指标细化 + 自适应策略 + 基准/压力流程

### 🔧 开发文档
- [🏗️ 架构设计](docs/development/architecture.md) - 系统架构和设计理念
- [🤝 贡献指南](docs/development/contributing.md) - 如何参与项目开发
- [📋 发布清单](docs/development/release.md) - 版本发布流程
- [📈 质量评估](docs/development/quality.md) - 代码质量分析报告
- [🐛 修复记录](docs/development/changelog.md) - 重要修复和改进记录

### 📊 测试报告
- [📈 性能分析](docs/reports/PERFORMANCE_ANALYSIS.md) - SQLite 性能测试分析
- [🔧 优化建议](docs/reports/SQLITE_OPTIMIZATION.md) - SQLite 优化策略
- [📋 测试报告](docs/reports/TEST_REPORT_ANALYSIS.md) - 详细测试结果分析
- [🛠️ 工具说明](docs/reports/sqlite-tools.md) - SQLite 测试工具文档

## 🐛 重要修复记录

延伸阅读
- [修复与改进记录](docs/development/changelog.md)

### 数据完整性监控指标修复 (2025-09-30)
- **问题**：Grafana 监控面板显示数据完整性为 10000% 而非正常的 100%
- **原因**：Prometheus 指标范围定义不一致（0-1 vs 0-100）
- **修复**：统一指标范围为 0-1，修复初始化和记录逻辑
- **影响**：✅ 监控面板现在正确显示数据完整性百分比
- **详情**：[修复日志](docs/development/changelog.md)

## 📋 测试

延伸阅读
- [测试指南](docs/guides/testing.md)
- [集成测试指南](docs/guides/integration-tests.md)

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
make docker-redis-test      # Redis 测试
```

> 提示：性能观测请参见上文“📡 监控与指标（MetricsReporter）”章节的「Prometheus + Grafana 快速监控」小节。

### 测试覆盖范围
- ✅ 基本批量处理功能
- ✅ Schema 分组逻辑
- ✅ SQL 生成正确性
- ✅ Redis 操作生成正确性
- ✅ 不同数据库类型和冲突策略
- ✅ 错误处理和边界条件
- ✅ 并发安全性测试
- ✅ 大数据量压力测试
- ✅ 数据库连接异常处理
- ✅ Redis Pipeline 批量执行

*详细测试文档：[集成测试指南](docs/guides/integration-tests.md)*

## 🏗️ 文件结构

```
batchsql/
├── README.md
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
├── .env.test
├── .env.sqlite.test
├── docker-compose.integration.yml
├── Dockerfile.integration
├── Dockerfile.sqlite.integration
├── batchsql.go              # 主入口与管道工厂
├── driver.go                # 驱动接口与实现入口（SQL/Redis等）
├── executor.go              # 执行器（含可选并发限流：WithConcurrencyLimit）
├── processor.go             # 处理器（SQL/Redis等各自批处理实现）
├── metrics_reporter.go      # 指标上报接口与默认实现
├── schema.go                # Schema 定义
├── request.go               # Request 定义
├── error.go                 # 错误定义
├── batchsql_test.go
├── benchmark_test.go
├── boundary_test.go
├── concurrency_test.go
├── db_connection_test.go
├── error_handling_test.go
├── large_data_test.go
├── docs/
│   ├── index.md
│   ├── api/
│   │   ├── reference.md
│   │   └── configuration.md
│   ├── guides/
│   │   ├── examples.md
│   │   ├── testing.md
│   │   ├── monitoring.md
│   │   ├── monitoring-quickstart.md
│   │   ├── custom-metrics-reporter.md
│   │   ├── tuning.md
│   │   ├── troubleshooting.md
│   │   └── integration-tests.md
│   ├── development/
│   │   ├── architecture.md
│   │   ├── contributing.md
│   │   ├── changelog.md
│   │   ├── quality.md
│   │   └── release.md
│   └── reports/
│       ├── PERFORMANCE_ANALYSIS.md
│       ├── SQLITE_OPTIMIZATION.md
│       ├── TEST_REPORT_ANALYSIS.md
│       └── sqlite-tools.md
├── scripts/

└── test/
    ├── integration/
    │   ├── config.go
    │   ├── main.go
    │   ├── metrics_reporter.go
    │   ├── prometheus.go
    │   ├── prometheus.yml
    │   ├── redis_tests.go
    │   ├── reports.go
    │   ├── run-single-db-test.sh
    │   ├── sql_tests.go
    │   ├── types.go
    │   ├── utils.go
    │   └── grafana/
    │       └── provisioning/...
    ├── sql/
    │   ├── mysql/init.sql
    │   ├── postgres/init.sql
    │   └── sqlite/init.sql
    └── sqlite/
        └── tools/...
```




## 🤝 贡献

延伸阅读
- [贡献指南](docs/development/contributing.md)
- [发布流程](docs/development/release.md)

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