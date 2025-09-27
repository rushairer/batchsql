#!/bin/bash

# BatchSQL 压力测试脚本

set -e

echo "🚀 开始 BatchSQL 压力测试..."

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装"
    exit 1
fi

# 构建压力测试工具
echo "🔨 构建压力测试工具..."
go build -o bin/stress_test test/cmd/stress/main.go

# 创建测试结果目录
mkdir -p test/results

# 运行不同场景的压力测试
echo "📊 运行压力测试场景..."

# 场景1: 小批量高频率
echo "测试场景1: 小批量高频率 (100 批次, 每批 10 条记录)"
./bin/stress_test -batches=100 -batch-size=10 -concurrent=5 > test/results/small_batch_high_freq.log 2>&1

# 场景2: 大批量低频率
echo "测试场景2: 大批量低频率 (10 批次, 每批 1000 条记录)"
./bin/stress_test -batches=10 -batch-size=1000 -concurrent=2 > test/results/large_batch_low_freq.log 2>&1

# 场景3: 高并发
echo "测试场景3: 高并发 (50 批次, 每批 100 条记录, 10 并发)"
./bin/stress_test -batches=50 -batch-size=100 -concurrent=10 > test/results/high_concurrency.log 2>&1

# 场景4: 极限测试
echo "测试场景4: 极限测试 (100 批次, 每批 500 条记录, 20 并发)"
./bin/stress_test -batches=100 -batch-size=500 -concurrent=20 > test/results/extreme_test.log 2>&1

echo "✅ 压力测试完成！"
echo "📋 测试结果保存在 test/results/ 目录中"

# 生成测试报告
echo "📈 生成测试报告..."
cat > test/results/summary.md << EOF
# BatchSQL 压力测试报告

## 测试环境
- 时间: $(date)
- Go 版本: $(go version)
- 系统: $(uname -a)

## 测试场景

### 场景1: 小批量高频率
- 批次数: 100
- 每批记录数: 10
- 并发数: 5
- 总记录数: 1,000

### 场景2: 大批量低频率
- 批次数: 10
- 每批记录数: 1,000
- 并发数: 2
- 总记录数: 10,000

### 场景3: 高并发
- 批次数: 50
- 每批记录数: 100
- 并发数: 10
- 总记录数: 5,000

### 场景4: 极限测试
- 批次数: 100
- 每批记录数: 500
- 并发数: 20
- 总记录数: 50,000

## 详细结果

查看各场景的详细日志:
- 场景1: small_batch_high_freq.log
- 场景2: large_batch_low_freq.log
- 场景3: high_concurrency.log
- 场景4: extreme_test.log
EOF

echo "📊 测试报告已生成: test/results/summary.md"