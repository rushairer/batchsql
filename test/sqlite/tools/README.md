# SQLite 测试工具使用指南

## 快速开始

每个工具都在独立的目录中，可以直接运行：

```bash
# 配置分析工具
cd test/sqlite/tools/config-analysis
go run main.go

# 性能基准测试
cd test/sqlite/tools/benchmark  
go run main.go

# 清理方式测试
cd test/sqlite/tools/clear-test
go run main.go

# 路径兼容性测试
cd test/sqlite/tools/path-compatibility
go run main.go
```

## 工具说明

| 工具 | 目录 | 功能 |
|------|------|------|
| 配置分析 | `config-analysis/` | 分析 SQLite 配置方案 |
| 基准测试 | `benchmark/` | 性能测试和 RPS 分析 |
| 清理测试 | `clear-test/` | 对比不同清理方式 |
| 路径测试 | `path-compatibility/` | 环境兼容性检查 |

## 数据目录

测试数据保存在 `test/data/` 目录下，会自动创建。