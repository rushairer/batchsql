# BatchSQL 集成测试文档

## 🎯 概述

BatchSQL 集成测试使用 Docker 容器在真实数据库环境中进行高并发压力测试，验证库的性能和稳定性。

## 🚀 快速开始

### 运行单个数据库测试

```bash
# MySQL 30分钟压力测试
make docker-mysql-test

# PostgreSQL 30分钟压力测试  
make docker-postgres-test

# SQLite 30分钟压力测试
make docker-sqlite-test

# 运行所有数据库测试
make docker-all-tests
```

## ⚙️ 配置说明

### 统一配置源

所有测试配置都在各自的 `docker-compose.{database}.yml` 文件中统一管理：

```yaml
# docker-compose.mysql.yml 中的配置
environment:
  - TEST_DURATION=1800s        # 30分钟测试时间
  - CONCURRENT_WORKERS=10      # 10个并发worker
  - RECORDS_PER_WORKER=2000    # 每个worker处理2000条记录
  - BATCH_SIZE=200            # 批次大小200条
  - BUFFER_SIZE=5000          # 缓冲区大小5000条
  - FLUSH_INTERVAL=100ms      # 刷新间隔100毫秒
```

### 修改测试参数

要修改测试参数，编辑对应的 docker-compose 文件：

- **MySQL**: `docker-compose.mysql.yml`
- **PostgreSQL**: `docker-compose.postgres.yml`
- **SQLite**: 在 `Makefile` 中的 `docker-sqlite-test` 目标

## 📊 测试报告

测试完成后，报告会保存在 `./test/reports/` 目录：

```bash
# 查看可用报告
make show-reports

# 报告文件格式
test/reports/
├── integration_test_report_2024-01-15_14-30-25.json  # JSON格式
└── integration_test_report_2024-01-15_14-30-25.html  # HTML格式
```

### 报告内容

- **性能指标**: 吞吐量 (RPS)、延迟、内存使用
- **并发测试**: 多worker并发写入性能
- **稳定性测试**: 长时间运行稳定性
- **内存分析**: GC次数、内存分配统计
- **错误统计**: 详细的错误信息和统计

## 🐳 Docker 架构

### 单容器架构

为了最大化内存利用率，每个数据库使用独立的容器：

```bash
# MySQL 测试 - 只运行 MySQL 容器
docker-compose -f docker-compose.mysql.yml up

# PostgreSQL 测试 - 只运行 PostgreSQL 容器  
docker-compose -f docker-compose.postgres.yml up

# SQLite 测试 - 无需外部容器
./test/integration/run-single-db-test.sh sqlite
```

### 容器配置

- **MySQL**: 使用 `mysql:8.0-oracle` 镜像，优化配置
- **PostgreSQL**: 使用 `postgres:16-alpine` 镜像
- **测试容器**: 基于 `golang:1.20-alpine` 构建

## 🔧 开发和调试

### 本地开发

```bash
# 设置开发环境
make dev-setup

# 运行单元测试
make test-unit

# 代码格式化和检查
make fmt vet lint
```

### 故障排除

```bash
# 清理所有容器和数据
make clean

# 查看容器状态
docker-compose -f docker-compose.mysql.yml ps

# 查看容器日志
docker-compose -f docker-compose.mysql.yml logs mysql-test
```

## 📈 性能基准

### 典型性能指标

基于默认配置 (10 workers × 2000 records × 30min)：

- **MySQL**: ~5000-15000 RPS
- **PostgreSQL**: ~3000-10000 RPS  
- **SQLite**: ~2000-8000 RPS

*实际性能取决于硬件配置和系统负载*

### 压力测试场景

1. **高吞吐量测试**: 单线程最大吞吐量
2. **并发压力测试**: 多worker并发写入
3. **大批次测试**: 大批次数据处理
4. **内存压力测试**: 大数据量内存使用
5. **长时间稳定性测试**: 30分钟持续运行

## 🛠️ 扩展测试

### 添加新的测试场景

在 `test/integration/main.go` 中添加新的测试函数：

```go
func runCustomTest(db *sql.DB, dbType string, config TestConfig) TestResult {
    // 自定义测试逻辑
}
```

### 支持新的数据库

1. 创建新的 `docker-compose.{database}.yml` 文件
2. 在 `test/sql/` 目录下创建初始化脚本
3. 在 `Makefile` 中添加新的测试目标
4. 更新 `test/integration/main.go` 中的数据库支持

## 📋 CI/CD 集成

```bash
# 完整的 CI 流程
make ci

# 包含: 格式化 → 静态检查 → 单元测试 → 集成测试
```

## 🔍 监控和分析

### 性能分析

```bash
# 生成性能分析报告
make profile

# 查看 CPU 分析
go tool pprof cpu.prof

# 查看内存分析  
go tool pprof mem.prof
```

### 测试覆盖率

```bash
# 生成覆盖率报告
make coverage

# 查看 HTML 报告
open coverage.html
```

## 📚 相关文档

- [主要 README](README.md) - 库的基本使用
- [配置说明](CONFIG.md) - 详细的配置参数说明
- [发布检查清单](RELEASE_CHECKLIST.md) - 发布前的检查项目

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支
3. 运行完整测试: `make ci`
4. 提交 Pull Request

确保所有测试通过并且性能指标在合理范围内。