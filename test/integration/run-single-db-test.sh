#!/bin/bash

set -e

echo "🚀 Starting BatchSQL Single Database Pressure Test"
echo "============================================================"

# 显示环境信息
echo "📋 Environment Information:"
echo "   Container OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "   Available Memory: $(free -h | grep Mem | awk '{print $2}')"
echo "   CPU Cores: $(nproc)"
echo ""

# 显示配置
echo "⚙️  Optimized High-Performance Test Configuration:"
echo "   Test Type: ${TEST_TYPE:-unknown}"
echo "   MySQL DSN: ${MYSQL_DSN:-Not configured}"
echo "   PostgreSQL DSN: ${POSTGRES_DSN:-Not configured}"
echo "   SQLite DSN: ${SQLITE_DSN:-/app/data/test.db}"
echo "   Test Duration: ${TEST_DURATION:-1800s} (NO TIMEOUT LIMIT)"
echo "   Concurrent Workers: ${CONCURRENT_WORKERS:-10}"
echo "   Records Per Worker: ${RECORDS_PER_WORKER:-2000}"
echo "   Batch Size: ${BATCH_SIZE:-200}"
echo "   Total Expected Records: $((${CONCURRENT_WORKERS:-10} * ${RECORDS_PER_WORKER:-2000}))"
echo "   Memory Optimized: Balanced for sustained high performance"
echo ""

# 等待数据库服务就绪
echo "⏳ Waiting for database service to be ready..."

if [ "$TEST_TYPE" = "mysql" ] && [ ! -z "$MYSQL_DSN" ]; then
    echo "   Checking MySQL connection..."
    MYSQL_READY=false
    for i in {1..30}; do
        if timeout 5 mysql -h mysql -u testuser -ptestpass123 -e "SELECT 1" >/dev/null 2>&1; then
            echo "   ✅ MySQL is ready for high-performance testing"
            MYSQL_READY=true
            break
        fi
        echo "   ⏳ Waiting for MySQL... (attempt $i/30)"
        sleep 5
    done
    
    if [ "$MYSQL_READY" = false ]; then
        echo "   ❌ MySQL connection failed after 150 seconds"
        exit 1
    fi
fi

if [ "$TEST_TYPE" = "postgres" ] && [ ! -z "$POSTGRES_DSN" ]; then
    echo "   Checking PostgreSQL connection..."
    POSTGRES_READY=false
    for i in {1..30}; do
        if timeout 5 pg_isready -h postgres -U testuser >/dev/null 2>&1; then
            echo "   ✅ PostgreSQL is ready for high-performance testing"
            POSTGRES_READY=true
            break
        fi
        echo "   ⏳ Waiting for PostgreSQL... (attempt $i/30)"
        sleep 5
    done
    
    if [ "$POSTGRES_READY" = false ]; then
        echo "   ❌ PostgreSQL connection failed after 150 seconds"
        exit 1
    fi
fi

echo ""

# 创建报告目录
mkdir -p /app/reports

# 启动健康检查服务器（后台运行）
echo "🏥 Starting health check server..."
(
    while true; do
        echo -e "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK" | nc -l -p 8080
        sleep 1
    done
) &
HEALTH_PID=$!

# 运行压力测试 - 无超时限制
echo "🧪 Running OPTIMIZED HIGH-PERFORMANCE pressure test..."
echo "============================================================"
echo "🚨 NO TIMEOUT LIMIT - Test will run until completion"
echo "💪 Optimized concurrency: ${CONCURRENT_WORKERS:-10} workers"
echo "📊 Substantial dataset: $((${CONCURRENT_WORKERS:-10} * ${RECORDS_PER_WORKER:-2000})) total records"
echo "⚡ Balanced throughput: ${BATCH_SIZE:-200} records per batch"
echo "🧠 Memory optimized for sustained performance"
echo ""

# 运行测试 - 移除所有超时限制
echo "🚀 Starting pressure test binary..."
./integration-test

TEST_EXIT_CODE=$?

# 停止健康检查服务器
kill $HEALTH_PID 2>/dev/null || true

echo ""
echo "============================================================"

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo "🎉 High-performance pressure test completed successfully!"
    echo ""
    echo "📊 Performance Reports generated in /app/reports/"
    ls -la /app/reports/
else
    echo "❌ Pressure test failed with exit code: $TEST_EXIT_CODE"
    echo ""
    echo "📊 Check reports for performance analysis:"
    ls -la /app/reports/ 2>/dev/null || echo "No reports generated"
fi

echo ""
echo "🔍 Final System Resource Usage:"
echo "   Memory Usage: $(free -h | grep Mem | awk '{print $3 "/" $2}')"
echo "   Disk Usage: $(df -h /app | tail -1 | awk '{print $3 "/" $2 " (" $5 ")"}')"

exit $TEST_EXIT_CODE