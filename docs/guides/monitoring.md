# BatchSQL 监控系统指南

## 📊 监控系统概览

BatchSQL 提供完整的监控解决方案，基于 Prometheus + Grafana 技术栈，实现实时性能监控、数据完整性验证和系统健康检查。

### 🏗️ 监控架构

```
BatchSQL Application
        ↓ (metrics)
   Prometheus Server  
        ↓ (query)
    Grafana Dashboard
        ↓ (alerts)
   Alert Manager
```

## 🚀 快速开始

### 1. 启动监控栈

```bash
# 使用 Docker Compose 启动完整监控环境
docker-compose -f docker-compose.integration.yml up -d

# 验证服务状态
docker-compose ps

# 访问服务
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
```

### 2. 配置应用监控

```go
package main

import (
    "github.com/rushairer/batchsql"
    "github.com/rushairer/batchsql/drivers/mysql"
    "github.com/rushairer/batchsql/test/integration"
)

func main() {
    // 1. 创建 Prometheus 指标收集器
    prometheusMetrics := integration.NewPrometheusMetrics()
    
    // 2. 启动指标服务器
    go prometheusMetrics.StartServer(9090)
    defer prometheusMetrics.StopServer()
    
    // 3. 创建带监控的执行器
    executor := mysql.NewBatchExecutor(db)
    metricsReporter := integration.NewPrometheusMetricsReporter(
        prometheusMetrics, "mysql", "production")
    executor = executor.WithMetricsReporter(metricsReporter)
    
    // 4. 创建 BatchSQL 实例
    batchSQL := batchsql.NewBatchSQL(ctx, 5000, 200, 100*time.Millisecond, executor)
    defer batchSQL.Close()
    
    // 5. 正常使用，指标自动收集
    // ...
}
```

## 📈 核心指标

### 性能指标

| 指标名称 | 类型 | 描述 | 标签 |
|---------|------|------|------|
| `batchsql_records_processed_total` | Counter | 已处理记录总数 | `database`, `table`, `test_name` |
| `batchsql_current_rps` | Gauge | 当前每秒处理记录数 | `database`, `table`, `test_name` |
| `batchsql_batch_execution_duration_ms` | Histogram | 批次执行耗时分布 | `database`, `table`, `test_name` |
| `batchsql_batch_size` | Histogram | 批次大小分布 | `database`, `table`, `test_name` |

### 质量指标

| 指标名称 | 类型 | 描述 | 标签 |
|---------|------|------|------|
| `batchsql_data_integrity_rate` | Gauge | 数据完整性率 (0-1) | `database`, `test_name` |
| `batchsql_error_rate` | Gauge | 错误率 (0-1) | `database`, `test_name` |
| `batchsql_batch_success_total` | Counter | 成功批次总数 | `database`, `table`, `test_name` |
| `batchsql_batch_failed_total` | Counter | 失败批次总数 | `database`, `table`, `test_name` |

### 系统指标

| 指标名称 | 类型 | 描述 | 标签 |
|---------|------|------|------|
| `batchsql_memory_usage_bytes` | Gauge | 内存使用量 | `database`, `test_name` |
| `batchsql_active_connections` | Gauge | 活跃连接数 | `database` |
| `batchsql_buffer_utilization` | Gauge | 缓冲区利用率 (0-1) | `database`, `test_name` |

## 🎛️ Grafana 面板配置

### 主要面板

#### 1. 性能概览面板

```json
{
  "title": "BatchSQL 性能概览",
  "panels": [
    {
      "title": "实时 RPS",
      "type": "stat",
      "targets": [
        {
          "expr": "sum(batchsql_current_rps) by (database)",
          "legendFormat": "{{database}}"
        }
      ]
    },
    {
      "title": "累计处理记录数",
      "type": "stat", 
      "targets": [
        {
          "expr": "sum(batchsql_records_processed_total)",
          "legendFormat": "总记录数"
        }
      ]
    }
  ]
}
```

#### 2. 数据完整性面板

```json
{
  "title": "数据完整性监控",
  "panels": [
    {
      "title": "数据完整性率",
      "type": "table",
      "targets": [
        {
          "expr": "batchsql_data_integrity_rate * 100",
          "format": "table",
          "legendFormat": "{{database}} - {{test_name}}"
        }
      ],
      "fieldConfig": {
        "defaults": {
          "unit": "percent",
          "thresholds": {
            "steps": [
              {"color": "red", "value": 0},
              {"color": "yellow", "value": 95},
              {"color": "green", "value": 99}
            ]
          }
        }
      }
    }
  ]
}
```

#### 3. 性能趋势面板

```json
{
  "title": "性能趋势分析",
  "panels": [
    {
      "title": "RPS 趋势",
      "type": "timeseries",
      "targets": [
        {
          "expr": "batchsql_current_rps",
          "legendFormat": "{{database}} - {{test_name}}"
        }
      ]
    },
    {
      "title": "批次执行耗时",
      "type": "timeseries", 
      "targets": [
        {
          "expr": "histogram_quantile(0.95, batchsql_batch_execution_duration_ms_bucket)",
          "legendFormat": "P95 - {{database}}"
        },
        {
          "expr": "histogram_quantile(0.50, batchsql_batch_execution_duration_ms_bucket)",
          "legendFormat": "P50 - {{database}}"
        }
      ]
    }
  ]
}
```

### 完整面板导入

```bash
# 导入预配置的 Grafana 面板
curl -X POST \
  http://admin:admin@localhost:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -d @test/integration/grafana/provisioning/dashboards/batchsql-performance.json
```

## 🔍 监控查询示例

### Prometheus 查询语句

#### 性能分析查询

```promql
# 各数据库的平均 RPS
avg(batchsql_current_rps) by (database)

# 最近5分钟的记录处理速率
rate(batchsql_records_processed_total[5m])

# 批次执行耗时的95分位数
histogram_quantile(0.95, rate(batchsql_batch_execution_duration_ms_bucket[5m]))

# 错误率趋势
rate(batchsql_batch_failed_total[5m]) / rate(batchsql_batch_success_total[5m] + batchsql_batch_failed_total[5m])
```

#### 容量规划查询

```promql
# 内存使用趋势
batchsql_memory_usage_bytes / 1024 / 1024  # 转换为 MB

# 缓冲区利用率
avg(batchsql_buffer_utilization) by (database, test_name)

# 连接池使用情况
batchsql_active_connections / on(database) group_left() max_connections
```

#### 数据质量查询

```promql
# 数据完整性低于99%的测试
batchsql_data_integrity_rate < 0.99

# 各数据库的数据完整性对比
batchsql_data_integrity_rate * 100

# 数据完整性变化趋势
delta(batchsql_data_integrity_rate[1h])
```

## 🚨 告警配置

### Prometheus 告警规则

```yaml
# prometheus/alert_rules.yml
groups:
  - name: batchsql_alerts
    rules:
      - alert: BatchSQLHighErrorRate
        expr: rate(batchsql_batch_failed_total[5m]) / rate(batchsql_batch_success_total[5m] + batchsql_batch_failed_total[5m]) > 0.05
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "BatchSQL 错误率过高"
          description: "数据库 {{ $labels.database }} 的错误率为 {{ $value | humanizePercentage }}"
      
      - alert: BatchSQLLowDataIntegrity
        expr: batchsql_data_integrity_rate < 0.95
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "BatchSQL 数据完整性异常"
          description: "测试 {{ $labels.test_name }} 的数据完整性仅为 {{ $value | humanizePercentage }}"
      
      - alert: BatchSQLLowPerformance
        expr: batchsql_current_rps < 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "BatchSQL 性能下降"
          description: "数据库 {{ $labels.database }} 的 RPS 降至 {{ $value }}"
      
      - alert: BatchSQLHighMemoryUsage
        expr: batchsql_memory_usage_bytes > 1073741824  # 1GB
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: "BatchSQL 内存使用过高"
          description: "内存使用量达到 {{ $value | humanizeBytes }}"
```

### Grafana 告警配置

```json
{
  "alert": {
    "name": "数据完整性告警",
    "message": "BatchSQL 数据完整性低于阈值",
    "frequency": "10s",
    "conditions": [
      {
        "query": {
          "queryType": "",
          "refId": "A",
          "model": {
            "expr": "batchsql_data_integrity_rate * 100",
            "interval": "",
            "legendFormat": "",
            "refId": "A"
          }
        },
        "reducer": {
          "type": "last",
          "params": []
        },
        "evaluator": {
          "params": [95],
          "type": "lt"
        }
      }
    ],
    "executionErrorState": "alerting",
    "noDataState": "no_data",
    "for": "1m"
  }
}
```

## 🔧 高级配置

### 自定义指标收集器

```go
type CustomMetricsCollector struct {
    prometheus *PrometheusMetrics
    database   string
    testName   string
    
    // 自定义指标
    customCounter   prometheus.Counter
    customHistogram prometheus.Histogram
}

func NewCustomMetricsCollector(pm *PrometheusMetrics, database, testName string) *CustomMetricsCollector {
    collector := &CustomMetricsCollector{
        prometheus: pm,
        database:   database,
        testName:   testName,
    }
    
    // 注册自定义指标
    collector.customCounter = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "batchsql_custom_operations_total",
        Help: "Total number of custom operations",
    })
    
    collector.customHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "batchsql_custom_duration_seconds",
        Help:    "Duration of custom operations",
        Buckets: prometheus.DefBuckets,
    })
    
    prometheus.MustRegister(collector.customCounter, collector.customHistogram)
    
    return collector
}

func (c *CustomMetricsCollector) RecordCustomOperation(duration time.Duration) {
    c.customCounter.Inc()
    c.customHistogram.Observe(duration.Seconds())
}
```

### 多环境监控配置

```go
type EnvironmentConfig struct {
    Name              string
    PrometheusPort    int
    GrafanaURL        string
    AlertManagerURL   string
    MetricsPrefix     string
}

func SetupMonitoringForEnvironment(env EnvironmentConfig) *PrometheusMetrics {
    // 创建带环境标识的指标收集器
    prometheusMetrics := NewPrometheusMetrics()
    prometheusMetrics.SetEnvironment(env.Name)
    prometheusMetrics.SetMetricsPrefix(env.MetricsPrefix)
    
    // 启动指标服务器
    go prometheusMetrics.StartServer(env.PrometheusPort)
    
    // 配置告警
    if env.AlertManagerURL != "" {
        prometheusMetrics.ConfigureAlertManager(env.AlertManagerURL)
    }
    
    return prometheusMetrics
}

// 使用示例
func main() {
    envs := []EnvironmentConfig{
        {Name: "production", PrometheusPort: 9090, MetricsPrefix: "prod_"},
        {Name: "staging", PrometheusPort: 9091, MetricsPrefix: "staging_"},
        {Name: "development", PrometheusPort: 9092, MetricsPrefix: "dev_"},
    }
    
    for _, env := range envs {
        metrics := SetupMonitoringForEnvironment(env)
        defer metrics.StopServer()
    }
}
```

## 📊 监控最佳实践

### 1. 指标命名规范

```go
// ✅ 好的命名
batchsql_records_processed_total
batchsql_batch_execution_duration_ms
batchsql_data_integrity_rate

// ❌ 避免的命名
records_count
duration
integrity
```

### 2. 标签使用原则

```go
// ✅ 合理的标签
labels := map[string]string{
    "database":  "mysql",      // 数据库类型
    "table":     "users",      // 表名
    "test_name": "batch_insert", // 测试名称
    "env":       "production", // 环境
}

// ❌ 避免高基数标签
labels := map[string]string{
    "record_id": "12345",     // 会产生大量时间序列
    "timestamp": "1609459200", // 时间戳不应作为标签
}
```

### 3. 监控数据保留策略

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "alert_rules.yml"

scrape_configs:
  - job_name: 'batchsql'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 5s
    metrics_path: /metrics

# 数据保留配置
storage:
  tsdb:
    retention.time: 30d
    retention.size: 10GB
```

### 4. 性能优化建议

```go
// 批量更新指标，减少锁竞争
func (pm *PrometheusMetrics) BatchUpdateMetrics(updates []MetricUpdate) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    
    for _, update := range updates {
        switch update.Type {
        case "counter":
            pm.counters[update.Name].Add(update.Value)
        case "gauge":
            pm.gauges[update.Name].Set(update.Value)
        case "histogram":
            pm.histograms[update.Name].Observe(update.Value)
        }
    }
}

// 使用缓冲区减少指标更新频率
type MetricsBuffer struct {
    updates []MetricUpdate
    size    int
    mu      sync.Mutex
}

func (mb *MetricsBuffer) Add(update MetricUpdate) {
    mb.mu.Lock()
    defer mb.mu.Unlock()
    
    mb.updates = append(mb.updates, update)
    
    if len(mb.updates) >= mb.size {
        mb.flush()
    }
}
```

## 🔍 故障排查

### 常见问题及解决方案

#### 1. 指标数据缺失

**症状**：Grafana 面板显示 "No data"

**排查步骤**：
```bash
# 检查 Prometheus 目标状态
curl http://localhost:9090/api/v1/targets

# 检查指标是否存在
curl http://localhost:9090/api/v1/label/__name__/values | grep batchsql

# 检查应用指标端点
curl http://localhost:9090/metrics | grep batchsql
```

**解决方案**：
- 确认应用正确启动指标服务器
- 检查防火墙和网络连接
- 验证 Prometheus 配置文件

#### 2. 数据完整性指标异常

**症状**：显示 10000% 或其他异常值

**排查步骤**：
```bash
# 检查原始指标值
curl -s http://localhost:9090/api/v1/query?query=batchsql_data_integrity_rate

# 检查 Grafana 查询表达式
# 应该是：batchsql_data_integrity_rate * 100
# 而不是：batchsql_data_integrity_rate * 10000
```

**解决方案**：
- 确认指标范围为 0-1
- 修正 Grafana 查询表达式
- 检查应用中的指标计算逻辑

#### 3. 性能指标不准确

**症状**：RPS 显示异常高或异常低

**排查步骤**：
```promql
# 检查计数器增长率
rate(batchsql_records_processed_total[1m])

# 检查时间窗口设置
increase(batchsql_records_processed_total[5m])
```

**解决方案**：
- 调整 PromQL 查询的时间窗口
- 确认计数器正确递增
- 检查系统时钟同步

## 📚 相关文档

- [API_REFERENCE.md](API_REFERENCE.md) - WithMetricsReporter 详细用法
- [EXAMPLES.md](EXAMPLES.md) - 监控集成示例
- [TESTING_GUIDE.md](TESTING_GUIDE.md) - 监控测试方法
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - 详细故障排查

---

💡 **监控建议**：
1. 从核心指标开始，逐步扩展监控范围
2. 设置合理的告警阈值，避免告警疲劳
3. 定期审查和优化监控配置
4. 建立监控数据的备份和恢复机制