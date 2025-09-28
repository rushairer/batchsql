#!/bin/bash

# BatchSQL 集成测试 + Prometheus监控启动脚本
# 使用方法: ./scripts/run-integration-tests-with-monitoring.sh

set -e

echo "🚀 启动 BatchSQL 集成测试 + Prometheus 监控环境"

# 检查Docker和Docker Compose是否可用
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装或不可用"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose 未安装或不可用"
    exit 1
fi

# 创建必要的目录
echo "📁 创建必要的目录..."
mkdir -p reports
mkdir -p monitoring/grafana/provisioning/datasources
mkdir -p monitoring/grafana/provisioning/dashboards
mkdir -p monitoring/grafana/dashboards

# 停止并清理现有容器
echo "🧹 清理现有容器..."
docker-compose -f docker-compose.monitoring.yml down -v --remove-orphans || true

# 构建并启动服务
echo "🔨 构建并启动监控服务..."
docker-compose -f docker-compose.monitoring.yml up -d prometheus grafana mysql postgres

# 等待数据库服务启动
echo "⏳ 等待数据库服务启动..."
sleep 30

# 检查数据库连接
echo "🔍 检查数据库连接..."
docker-compose -f docker-compose.monitoring.yml exec mysql mysqladmin ping -h localhost -u root -ppassword || {
    echo "❌ MySQL 连接失败"
    exit 1
}

docker-compose -f docker-compose.monitoring.yml exec postgres pg_isready -U postgres || {
    echo "❌ PostgreSQL 连接失败"
    exit 1
}

echo "✅ 数据库服务已就绪"

# 启动集成测试
echo "🧪 启动集成测试..."
docker-compose -f docker-compose.monitoring.yml up --build integration-test

# 显示访问信息
echo ""
echo "🎉 监控环境已启动！"
echo ""
echo "📊 访问地址："
echo "   Prometheus: http://localhost:9090"
echo "   Grafana:    http://localhost:3000 (admin/admin123)"
echo "   测试指标:   http://localhost:9091/metrics"
echo ""
echo "📈 Grafana 仪表板："
echo "   - BatchSQL Performance Dashboard"
echo ""
echo "📁 测试报告目录: ./reports/"
echo ""
echo "🛑 停止监控环境: docker-compose -f docker-compose.monitoring.yml down"
echo ""

# 可选：保持服务运行
read -p "是否保持监控服务运行？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "✅ 监控服务将继续运行"
    echo "💡 使用以下命令停止: docker-compose -f docker-compose.monitoring.yml down"
else
    echo "🛑 停止监控服务..."
    docker-compose -f docker-compose.monitoring.yml down
fi