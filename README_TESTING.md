# BatchSQL 测试指南

## 测试架构

BatchSQL 采用混合测试策略，结合白盒测试和黑盒测试的优势：

### 🔍 测试文件分类

#### 白盒测试 (`package batchsql`)
- `client_test.go` - 客户端内部逻辑测试
- `schema_test.go` - Schema内部实现测试

**用途：**
- 测试私有方法和字段
- 验证内部状态变化
- 边界条件和错误处理

#### 黑盒测试 (`package batchsql_test`)
- `client_integration_test.go` - 公开API集成测试

**用途：**
- 端到端工作流测试
- 用户使用场景验证
- API设计合理性检查

## 🧪 测试命令

```bash
# 运行所有测试
make test

# 单元测试 + 覆盖率
make test-unit

# 压力测试
make test-stress

# 性能基准测试
make bench

# Docker集成测试
make test-integration
```

## 📊 测试覆盖率

当前测试覆盖率：**47.2%**

### 覆盖情况
- ✅ 客户端核心功能
- ✅ Schema操作
- ✅ 监控系统
- ✅ 错误处理
- ✅ 并发安全

### 待提升领域
- 🔄 驱动实现
- 🔄 连接管理
- 🔄 数据转换

## 🚀 压力测试场景

### 场景1: 小批量高频率
- 批次数: 100
- 每批记录数: 10
- 并发数: 5
- **结果**: 35万记录/秒

### 场景2: 大批量低频率
- 批次数: 10
- 每批记录数: 1,000
- 并发数: 2

### 场景3: 高并发
- 批次数: 50
- 每批记录数: 100
- 并发数: 10

### 场景4: 极限测试
- 批次数: 100
- 每批记录数: 500
- 并发数: 20

## 🐳 Docker集成测试

支持的数据库：
- MySQL 8.0
- PostgreSQL 15
- Redis 7
- MongoDB 6

监控组件：
- Prometheus
- Grafana (可选)

## 📝 编写测试的最佳实践

### 1. 选择合适的测试类型

**白盒测试适用于：**
```go
// 测试内部状态
func TestInternalState(t *testing.T) {
    client := NewClient()
    // 访问私有字段进行验证
    if client.reporter != nil {
        t.Error("Default reporter should be nil")
    }
}
```

**黑盒测试适用于：**
```go
// 测试公开API
func TestPublicAPI(t *testing.T) {
    client := batchsql.NewClient()
    // 只使用公开方法
    result := client.WithMetricsReporter(reporter)
    // 验证行为而非实现
}
```

### 2. Mock设计原则

- 实现真实接口
- 提供可控的行为
- 记录调用历史
- 支持错误注入

### 3. 测试数据管理

- 使用确定性数据
- 避免外部依赖
- 清理测试状态
- 隔离测试用例

## 🔧 故障排除

### 常见问题

1. **导入循环**
   - 使用 `package xxx_test` 避免
   - 将共享mock移到独立包

2. **竞态条件**
   - 使用 `go test -race`
   - 正确使用同步原语

3. **测试超时**
   - 设置合理的context超时
   - 避免无限等待

### 调试技巧

```bash
# 详细输出
go test -v ./...

# 竞态检测
go test -race ./...

# 覆盖率分析
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📈 持续改进

### 测试质量指标
- 代码覆盖率 > 80%
- 所有测试通过率 100%
- 压力测试稳定性 > 99.9%
- 平均测试执行时间 < 10s

### 定期检查
- 每周运行完整测试套件
- 每月更新压力测试基准
- 季度评估测试架构