# BatchSQL Prometheus 监控指南

*最后更新：2025年1月28日 | 版本：v2.0.0*

## 🎯 概述

BatchSQL 集成了 Prometheus 监控系统，可以实时收集和展示批量处理的性能指标。通过 Grafana 仪表板，你可以直观地观察不同测试用例的性能曲线，包括 RPS、延迟、内存使用、数据完整性等关键指标。

## 🏗️ 监控架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  BatchSQL App   │───▶│   Prometheus    │───▶│     Grafana     │
│  (集成测试)      │    │   (指标收集)     │    │   (可视化展示)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
    :9091/metrics           :9090/api              :3000/dashboards
```

### 核心组件

1. **PrometheusMetrics**: 实现 `MetricsReporter` 接口，收集批量处理指标
2. **Prometheus Server**: 抓取和存储时间序列数据
3. **Grafana Dashboard**: 提供丰富的可视化图表

## 🚀 快速开始

### 1. 启动监控环境

```bash
# 使用一键启动脚本
./scripts/run-integration-tests-with-monitoring.sh

# 或手动启动
docker-compose -f docker-compose.monitoring.yml up -d
```

### 2. 访问监控界面

- **Prometheus**: http://localhost:9092
- **Grafana**: http://localhost:3002 (admin/admin123)
- **测试指标**: http://localhost:9091/metrics

### 3. 查看性能仪表板

在 Grafana 中打开 "BatchSQL Performance Dashboard"，可以看到：

- **RPS 曲线**: 不同测试用例的每秒记录处理数
- **执行延迟**: 批量操作的响应时间分布
- **内存使用**: 实时内存分配和系统内存使用
- **数据完整性**: 数据一致性百分比
- **成功率**: 测试用例的成功/失败状态

## 📊 监控指标详解

### 批量执行指标

| 指标名称 | 类型 | 描述 | 标签 |
|---------|------|------|------|
| `batchsql_batch_execution_duration_seconds` | Histogram | 批量执行耗时分布 | database, table, status, test_case |
| `batchsql_batch_execution_total` | Counter | 批量执行总次数 | database, table, status, test_case |
| `batchsql_batch_size` | Histogram | 批次大小分布 | database, table, test_case |
| `batchsql_records_processed_total` | Counter | 处理记录总数 | database, table, test_case |

### 测试相关指标

| 指标名称 | 类型 | 描述 | 标签 |
|---------|------|------|------|
| `batchsql_test_duration_seconds` | Gauge | 测试执行时长 | database, test_case |
| `batchsql_test_records_total` | Gauge | 测试记录数量 | database, test_case, type |
| `batchsql_test_rps` | Gauge | 测试 RPS | database, test_case |
| `batchsql_test_success` | Gauge | 测试成功状态 | database, test_case |
| `batchsql_test_data_integrity_rate` | Gauge | 数据完整性百分比 | database, test_case |

### 系统资源指标

| 指标名称 | 类型 | 描述 | 标签 |
|---------|------|------|------|
| `batchsql_memory_usage_mb` | Gauge | 内存使用量(MB) | database, test_case, type |
| `batchsql_gc_count` | Gauge | GC 运行次数 | database, test_case |
| `batchsql_errors_total` | Counter | 错误总数 | database, test_case, error_type |

## 🔧 自定义监控

### 在应用中集成 Prometheus

```go
package main

import (
    "context"
    "database/sql"
    
    "github.com/rushairer/batchsql"
    "github.com/rushairer/batchsql/monitoring"
)

func main() {
    // 创建 Prometheus 指标收集器
    prometheusMetrics := monitoring.NewPrometheusMetrics()
    
    // 启动 HTTP 服务器暴露指标
    if err := prometheusMetrics.StartServer(9090); err != nil {
        panic(err)
    }
    defer prometheusMetrics.StopServer()
    
    // 创建 BatchSQL 实例并配置监控
    db, _ := sql.Open("mysql", "dsn")
    batchSQL := batchsql.NewMySQLBatchSQL(context.Background(), db, 
        batchsql.PipelineConfig{
            BufferSize:    10000,
            FlushSize:     500,
            FlushInterval: 100 * time.Millisecond,
        }).WithMetricsReporter(prometheusMetrics)
    
    // 设置当前测试用例标签
    prometheusMetrics.SetCurrentTestCase("my-test-case")
    
    // 使用 BatchSQL 进行批量处理...
}
```

### 记录自定义测试结果

```go
// 记录测试结果到 Prometheus
prometheusMetrics.RecordTestResult("mysql", "performance-test", monitoring.TestResult{
    Database:            "mysql",
    TestName:            "performance-test",
    Duration:            time.Minute * 5,
    TotalRecords:        100000,
    ActualRecords:       100000,
    DataIntegrityRate:   100.0,
    RecordsPerSecond:    2000.0,
    RPSValid:            true,
    Success:             true,
    // ... 其他字段
})

// 记录实时指标
prometheusMetrics.RecordRealTimeMetrics("mysql", "performance-test", 1500.0, 256.5)
```

## 📈 Grafana 仪表板配置

### 预置仪表板

项目包含一个预配置的 Grafana 仪表板 (`monitoring/grafana/dashboards/batchsql-performance.json`)，包含以下面板：

1. **Records Per Second (RPS)**: 显示不同测试用例的 RPS 趋势
2. **Batch Execution Duration**: 批量操作延迟的百分位数分布
3. **Memory Usage**: 内存分配和系统内存使用情况
4. **Data Integrity Rate**: 数据完整性百分比
5. **Batch Execution Rate**: 批量执行频率
6. **Records Processing Rate**: 记录处理速率
7. **Test Success Status**: 测试成功/失败状态

### 自定义面板

你可以基于收集的指标创建自定义面板：

```promql
# 平均 RPS（按数据库类型分组）
avg by (database) (batchsql_test_rps)

# 99th 百分位延迟
histogram_quantile(0.99, rate(batchsql_batch_execution_duration_seconds_bucket[5m]))

# 错误率
rate(batchsql_errors_total[5m]) / rate(batchsql_batch_execution_total[5m]) * 100

# 内存增长率
rate(batchsql_memory_usage_mb{type="alloc"}[5m])
```

## 🔍 故障排查

### 常见问题

1. **指标未显示**
   - 检查 Prometheus 配置文件中的 scrape 配置
   - 确认应用的 `/metrics` 端点可访问
   - 验证防火墙设置

2. **Grafana 无法连接 Prometheus**
   - 检查 Docker 网络配置
   - 确认 Prometheus 服务正常运行
   - 验证数据源配置

3. **仪表板显示异常**
   - 检查时间范围设置
   - 确认查询语句正确
   - 验证标签匹配

### 调试命令

```bash
# 检查 Prometheus 目标状态
curl http://localhost:9092/api/v1/targets

# 查看可用指标
curl http://localhost:9091/metrics

# 检查容器日志
docker-compose -f docker-compose.monitoring.yml logs prometheus
docker-compose -f docker-compose.monitoring.yml logs grafana
```

## 🎛️ 配置选项

### 环境变量

| 变量名 | 默认值 | 描述 |
|--------|--------|------|
| `PROMETHEUS_PORT` | 9090 | Prometheus HTTP 服务端口 |
| `TEST_DURATION` | 30m | 集成测试持续时间 |
| `CONCURRENT_WORKERS` | 10 | 并发工作者数量 |
| `BATCH_SIZE` | 500 | 批次大小 |
| `BUFFER_SIZE` | 10000 | 缓冲区大小 |

### Prometheus 配置

编辑 `monitoring/prometheus.yml` 来调整抓取配置：

```yaml
scrape_configs:
  - job_name: 'batchsql-app'
    static_configs:
      - targets: ['app:9090']
    scrape_interval: 5s  # 抓取间隔
    metrics_path: /metrics
```

## 🚀 生产环境部署

### 安全考虑

1. **认证和授权**: 为 Grafana 配置 LDAP 或 OAuth
2. **网络安全**: 使用 TLS 加密通信
3. **访问控制**: 限制 Prometheus 和 Grafana 的访问权限

### 性能优化

1. **存储配置**: 调整 Prometheus 的存储保留策略
2. **查询优化**: 使用合适的时间范围和聚合函数
3. **资源限制**: 为容器设置适当的 CPU 和内存限制

### 高可用部署

```yaml
# docker-compose.prod.yml 示例
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    deploy:
      replicas: 2
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 1G
```

## 📚 相关资源

- [Prometheus 官方文档](https://prometheus.io/docs/)
- [Grafana 官方文档](https://grafana.com/docs/)
- [PromQL 查询语言](https://prometheus.io/docs/prometheus/latest/querying/)
- [BatchSQL 架构文档](ARCHITECTURE.md)

---

通过这套监控系统，你可以深入了解 BatchSQL 的性能特征，识别瓶颈，优化配置，确保在生产环境中达到最佳性能！🎉