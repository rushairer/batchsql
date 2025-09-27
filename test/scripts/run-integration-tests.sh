#!/bin/bash

set -e

echo "🚀 开始BatchSQL集成测试..."

# 等待服务启动
echo "⏳ 等待数据库服务启动..."
sleep 10

# 检查数据库连接
echo "🔍 检查数据库连接..."

# MySQL连接测试
echo "测试MySQL连接..."
mysql -h$MYSQL_HOST -P$MYSQL_PORT -u$MYSQL_USER -p$MYSQL_PASSWORD -e "SELECT 1;" $MYSQL_DATABASE

# PostgreSQL连接测试
echo "测试PostgreSQL连接..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DATABASE -c "SELECT 1;"

# Redis连接测试
echo "测试Redis连接..."
redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD ping

# MongoDB连接测试
echo "测试MongoDB连接..."
mongosh --host $MONGODB_HOST:$MONGODB_PORT -u $MONGODB_USER -p $MONGODB_PASSWORD --authenticationDatabase admin --eval "db.adminCommand('ping')"

echo "✅ 所有数据库连接正常"

# 运行单元测试
echo "🧪 运行单元测试..."
go test -v -race -coverprofile=coverage.out ./...

# 运行集成测试
echo "🔗 运行集成测试..."
go test -v -tags=integration ./test/integration/...

# 运行性能测试
echo "⚡ 运行性能测试..."
go test -v -bench=. -benchmem ./...

# 运行压力测试
echo "💪 运行压力测试..."
./test/scripts/stress-test.sh

# 生成测试报告
echo "📊 生成测试报告..."
go tool cover -html=coverage.out -o coverage.html

echo "🎉 所有测试完成！"
echo "📈 测试覆盖率报告: coverage.html"
echo "📊 Prometheus监控: http://localhost:9090"
echo "📈 Grafana仪表板: http://localhost:3000 (admin/admin)"