# BatchSQL 文档索引

欢迎使用 BatchSQL 文档！本文档提供了完整的使用指南、API 参考和开发文档。

## 📚 文档结构

### 🚀 快速开始
- [项目概览](../README.md) - 项目介绍和快速开始
- [使用示例](guides/examples.md) - 丰富的代码示例和最佳实践

### 📖 API 文档
- [API 参考](api/reference.md) - 完整的 API 文档和使用指南
- [配置指南](api/configuration.md) - 详细的配置参数说明

### 📖 用户指南
- [测试指南](guides/testing.md) - 完整的测试文档和 Redis 测试报告
- [监控指南](guides/monitoring.md) - Prometheus + Grafana 监控系统
- [监控快速上手](guides/monitoring-quickstart.md) - 三步完成 Prometheus + Grafana 指标可视化
- [自定义 MetricsReporter 指南](guides/custom-metrics-reporter.md) - 接口语义、实现建议与示例代码
- [故障排除](guides/troubleshooting.md) - 完整的问题诊断和解决方案
- [集成测试](guides/integration-tests.md) - 集成测试详细说明

### 🔧 开发文档
- [架构设计](development/architecture.md) - 系统架构和设计理念
- [贡献指南](development/contributing.md) - 如何参与项目开发
- [发布清单](development/release.md) - 版本发布流程
- [质量评估](development/quality.md) - 代码质量分析报告
- [修复记录](development/changelog.md) - 重要修复和改进记录

### 📊 测试报告
- [性能分析](reports/PERFORMANCE_ANALYSIS.md) - SQLite 性能测试分析
- [优化建议](reports/SQLITE_OPTIMIZATION.md) - SQLite 优化策略
- [测试报告](reports/TEST_REPORT_ANALYSIS.md) - 详细测试结果分析
- [工具说明](reports/sqlite-tools.md) - SQLite 测试工具文档

## 🔍 按主题浏览

### 数据库支持
- **MySQL**: [API 参考](api/reference.md#mysql) | [配置](api/configuration.md#mysql)
- **PostgreSQL**: [API 参考](api/reference.md#postgresql) | [配置](api/configuration.md#postgresql)
- **SQLite**: [API 参考](api/reference.md#sqlite) | [优化指南](reports/SQLITE_OPTIMIZATION.md)
- **Redis**: [API 参考](api/reference.md#redis) | [测试报告](guides/testing.md#redis-测试)

### 功能特性
- **批量处理**: [使用示例](guides/examples.md#批量处理)
- **冲突处理**: [API 参考](api/reference.md#冲突处理策略)
- **监控集成**: [监控指南](guides/monitoring.md)
- **性能优化**: [架构设计](development/architecture.md#性能优化)

### 开发和测试
- **单元测试**: [测试指南](guides/testing.md#单元测试)
- **集成测试**: [集成测试](guides/integration-tests.md)
- **性能测试**: [性能分析](reports/PERFORMANCE_ANALYSIS.md)
- **故障排除**: [故障排除](guides/troubleshooting.md)

## 📋 快速链接

- 🚀 [快速开始](../README.md#🚀-快速开始)
- 📖 [API 参考](api/reference.md)
- 💡 [使用示例](guides/examples.md)
- 🧪 [测试指南](guides/testing.md)
- 📊 [监控指南](guides/monitoring.md)
- 📈 [监控快速上手](guides/monitoring-quickstart.md)
- 🧩 [自定义 MetricsReporter 指南](guides/custom-metrics-reporter.md)
- 🔧 [故障排除](guides/troubleshooting.md)
- 🏗️ [架构设计](development/architecture.md)

## 📡 MetricsReporter 快速了解

- 功能说明：统一的指标上报接口，覆盖入队延迟、攒批耗时、执行耗时、批大小、错误计数、执行并发、队列长度、在途批次等关键阶段与状态。
- 使用场景：
  - 开箱即用观测（Prometheus + Grafana）
  - 接入自有监控体系（实现自定义 Reporter）
  - 压测/调优时对各阶段瓶颈进行定位
- 配置方法：
  - 默认 NoopMetricsReporter（零开销，未注入时不产生观测）
  - 在 NewBatchSQL 之前，先对执行器注入 Reporter（WithMetricsReporter）
  - NewBatchSQL 会尊重已注入 Reporter，不会覆盖为 Noop
- 最小示例（Prometheus 快速上手）：
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
- 延伸阅读：
  - 监控快速上手：docs/guides/monitoring-quickstart.md
  - 自定义 Reporter：docs/guides/custom-metrics-reporter.md
  - API 接口定义：docs/api/reference.md（MetricsReporter 小节）

## 📞 获取帮助

如果您在使用过程中遇到问题：

1. 查看 [故障排除指南](guides/troubleshooting.md)
2. 阅读 [API 参考文档](api/reference.md)
3. 查看 [使用示例](guides/examples.md)
4. 提交 [GitHub Issue](https://github.com/rushairer/batchsql/issues)

---

*最后更新：2025年9月30日*