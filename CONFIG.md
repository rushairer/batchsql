# BatchSQL 集成测试配置说明

## 🎯 统一配置原则

**唯一配置源**: `docker-compose.mysql.yml` 中的环境变量

所有测试参数都从这个文件的环境变量中读取，避免配置分散和不一致问题。

## 📋 当前配置参数

```yaml
environment:
  # 数据库连接
  - MYSQL_DSN=testuser:testpass123@tcp(mysql:3306)/batchsql_test?parseTime=true&multiStatements=true
  
  # 测试参数 - 30分钟压力测试配置
  - TEST_DURATION=1800s        # 测试持续时间：30分钟
  - CONCURRENT_WORKERS=10      # 并发工作者数量：10个
  - RECORDS_PER_WORKER=2000    # 每个工作者处理记录数：2000条
  - BATCH_SIZE=200            # 批次大小：200条记录
  - BUFFER_SIZE=5000          # 缓冲区大小：5000条记录
  - FLUSH_INTERVAL=100ms      # 刷新间隔：100毫秒
  
  # 测试类型
  - TEST_TYPE=mysql           # 当前测试的数据库类型
```

## 🔧 修改配置

要修改测试参数，**只需要编辑 `docker-compose.mysql.yml` 文件**：

1. 打开 `docker-compose.mysql.yml`
2. 找到 `mysql-test` 服务的 `environment` 部分
3. 修改相应的环境变量值
4. 重新运行测试：`make docker-mysql-test`

## 📊 配置说明

### 性能相关参数
- `CONCURRENT_WORKERS`: 并发工作线程数，影响CPU和内存使用
- `RECORDS_PER_WORKER`: 每个工作者处理的记录数，影响总测试数据量
- `BATCH_SIZE`: 批量插入的记录数，影响内存使用和数据库性能
- `BUFFER_SIZE`: 内存缓冲区大小，影响内存使用
- `FLUSH_INTERVAL`: 数据刷新间隔，影响实时性和性能

### 时间相关参数
- `TEST_DURATION`: 测试持续时间，支持格式：`30s`, `5m`, `1h`, `1800s`

## ⚠️ 重要提醒

1. **不要在其他文件中修改配置参数**
2. **所有默认值都已统一到与docker-compose一致**
3. **如果测试失败，首先检查内存使用情况**
4. **建议的配置范围**：
   - CONCURRENT_WORKERS: 5-20
   - RECORDS_PER_WORKER: 1000-5000
   - BATCH_SIZE: 100-500
   - 总记录数建议不超过100,000条

## 🚀 快速测试命令

```bash
# MySQL 压力测试
make docker-mysql-test

# PostgreSQL 压力测试（需要创建对应的docker-compose文件）
make docker-postgres-test

# SQLite 压力测试（需要创建对应的docker-compose文件）
make docker-sqlite-test
```

## 📈 性能调优建议

### 高性能配置（需要充足内存）
```yaml
- CONCURRENT_WORKERS=20
- RECORDS_PER_WORKER=5000
- BATCH_SIZE=500
```

### 稳定配置（当前使用）
```yaml
- CONCURRENT_WORKERS=10
- RECORDS_PER_WORKER=2000
- BATCH_SIZE=200
```

### 保守配置（内存受限环境）
```yaml
- CONCURRENT_WORKERS=5
- RECORDS_PER_WORKER=1000
- BATCH_SIZE=100