# Prometheus集成指南

## 概述

BatchSQL支持通过`MetricsReporter`接口集成Prometheus监控。本指南展示如何在生产环境中集成Prometheus。

## 依赖安装

首先添加Prometheus客户端依赖：

```bash
go mod tidy
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promauto
```

## 实现Prometheus报告器

```go
package main

import (
    "context"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/rushairer/batchsql"
)

type PrometheusReporter struct {
    duration *prometheus.HistogramVec
    total    *prometheus.CounterVec
    errors   *prometheus.CounterVec
}

func NewPrometheusReporter() *PrometheusReporter {
    return &PrometheusReporter{
        duration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "batchsql_execution_duration_seconds",
                Help:    "BatchSQL execution duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"driver", "table"},
        ),
        total: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "batchsql_executions_total",
                Help: "Total number of BatchSQL executions",
            },
            []string{"driver", "table", "status"},
        ),
        errors: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "batchsql_errors_total",
                Help: "Total number of BatchSQL errors",
            },
            []string{"driver", "table", "error_type"},
        ),
    }
}

func (p *PrometheusReporter) ReportBatchExecution(ctx context.Context, metrics batchsql.BatchMetrics) {
    // 记录执行时长
    p.duration.WithLabelValues(metrics.Driver, metrics.Table).Observe(metrics.Duration.Seconds())

    // 记录执行状态
    status := "success"
    if metrics.Error != nil {
        status = "error"
        errorType := "execution_error"
        p.errors.WithLabelValues(metrics.Driver, metrics.Table, errorType).Inc()
    }

    p.total.WithLabelValues(metrics.Driver, metrics.Table, status).Inc()
}
```

## 使用方式

```go
func main() {
    // 创建Prometheus监控报告器
    prometheusReporter := NewPrometheusReporter()

    // 创建带监控的BatchSQL客户端
    client := batchsql.NewClient().WithMetricsReporter(prometheusReporter)

    // 正常使用BatchSQL，监控数据会自动收集
    err := client.ExecuteWithSchema(ctx, schema, data)
}
```

## 监控指标

### 核心指标

1. **batchsql_execution_duration_seconds**
   - 类型: Histogram
   - 标签: `driver`, `table`
   - 描述: BatchSQL执行时长分布

2. **batchsql_executions_total**
   - 类型: Counter
   - 标签: `driver`, `table`, `status`
   - 描述: BatchSQL执行总数

3. **batchsql_errors_total**
   - 类型: Counter
   - 标签: `driver`, `table`, `error_type`
   - 描述: BatchSQL错误总数

### 有用的Prometheus查询

```promql
# 每秒执行率
rate(batchsql_executions_total[5m])

# 95%分位数延迟
histogram_quantile(0.95, rate(batchsql_execution_duration_seconds_bucket[5m]))

# 错误率
rate(batchsql_errors_total[5m]) / rate(batchsql_executions_total[5m])

# 按驱动分组的执行量
sum(rate(batchsql_executions_total[5m])) by (driver)

# 按表分组的平均执行时间
rate(batchsql_execution_duration_seconds_sum[5m]) / rate(batchsql_execution_duration_seconds_count[5m])
```

## Grafana仪表板

可以创建Grafana仪表板来可视化这些指标：

### 推荐面板

1. **执行率时间序列图**
   ```promql
   sum(rate(batchsql_executions_total[5m])) by (driver)
   ```

2. **延迟分位数图**
   ```promql
   histogram_quantile(0.50, rate(batchsql_execution_duration_seconds_bucket[5m]))
   histogram_quantile(0.95, rate(batchsql_execution_duration_seconds_bucket[5m]))
   histogram_quantile(0.99, rate(batchsql_execution_duration_seconds_bucket[5m]))
   ```

3. **错误率图**
   ```promql
   rate(batchsql_errors_total[5m]) / rate(batchsql_executions_total[5m])
   ```

4. **热力图**
   ```promql
   rate(batchsql_execution_duration_seconds_bucket[5m])
   ```

## 告警规则

```yaml
groups:
- name: batchsql
  rules:
  - alert: BatchSQLHighErrorRate
    expr: rate(batchsql_errors_total[5m]) / rate(batchsql_executions_total[5m]) > 0.05
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "BatchSQL error rate is high"
      description: "BatchSQL error rate is {{ $value | humanizePercentage }} for {{ $labels.driver }}/{{ $labels.table }}"

  - alert: BatchSQLHighLatency
    expr: histogram_quantile(0.95, rate(batchsql_execution_duration_seconds_bucket[5m])) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "BatchSQL latency is high"
      description: "BatchSQL 95th percentile latency is {{ $value }}s for {{ $labels.driver }}/{{ $labels.table }}"
```

## 最佳实践

1. **标签使用**: 合理使用标签，避免高基数标签
2. **采样率**: 对于高频操作，考虑采样以减少监控开销
3. **告警设置**: 设置合理的告警阈值
4. **仪表板**: 创建清晰的可视化仪表板
5. **数据保留**: 配置合适的数据保留策略