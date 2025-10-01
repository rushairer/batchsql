#!/bin/bash

# BatchSQL Prometheus 监控启动脚本
# 使用方法: ./scripts/start-monitoring.sh [options]

set -e

# 默认配置
COMPOSE_FILE="docker-compose.integration.yml"
DETACH_MODE=true
SHOW_LOGS=false
CLEANUP=false

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 帮助信息
show_help() {
    echo "BatchSQL Prometheus 监控启动脚本"
    echo ""
    echo "使用方法:"
    echo "  $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help          显示此帮助信息"
    echo "  -f, --foreground    前台运行（不使用 -d 参数）"
    echo "  -l, --logs          启动后显示日志"
    echo "  -c, --cleanup       清理并重新启动"
    echo "  --stop              停止监控服务"
    echo "  --status            查看服务状态"
    echo ""
    echo "示例:"
    echo "  $0                  # 后台启动监控服务"
    echo "  $0 -f -l            # 前台启动并显示日志"
    echo "  $0 -c               # 清理并重新启动"
    echo "  $0 --stop           # 停止服务"
    echo "  $0 --status         # 查看状态"
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -f|--foreground)
            DETACH_MODE=false
            shift
            ;;
        -l|--logs)
            SHOW_LOGS=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP=true
            shift
            ;;
        --stop)
            echo -e "${YELLOW}🛑 停止监控服务...${NC}"
            docker compose -f $COMPOSE_FILE down prometheus grafana
            echo -e "${GREEN}✅ 监控服务已停止${NC}"
            exit 0
            ;;
        --status)
            echo -e "${BLUE}📊 监控服务状态:${NC}"
            docker compose -f $COMPOSE_FILE ps prometheus grafana
            exit 0
            ;;
        *)
            echo -e "${RED}❌ 未知选项: $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 检查 Docker 和 Docker Compose
check_dependencies() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker 未安装或不在 PATH 中${NC}"
        exit 1
    fi

    if ! command -v docker compose &> /dev/null; then
        echo -e "${RED}❌ Docker Compose 未安装或不在 PATH 中${NC}"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo -e "${RED}❌ Docker 服务未运行${NC}"
        exit 1
    fi
}

# 检查配置文件
check_config_files() {
    if [[ ! -f $COMPOSE_FILE ]]; then
        echo -e "${RED}❌ Docker Compose 文件不存在: $COMPOSE_FILE${NC}"
        exit 1
    fi

    local required_files=(
        "test/integration/prometheus.yml"
        "test/integration/grafana/provisioning/datasources/prometheus.yml"
        "test/integration/grafana/provisioning/dashboards/dashboard.yml"
        "test/integration/grafana/dashboards/batchsql-performance.json"
    )

    for file in "${required_files[@]}"; do
        if [[ ! -f $file ]]; then
            echo -e "${RED}❌ 配置文件不存在: $file${NC}"
            exit 1
        fi
    done
}

# 清理旧服务
cleanup_services() {
    if [[ $CLEANUP == true ]]; then
        echo -e "${YELLOW}🧹 清理旧服务...${NC}"
        docker compose -f $COMPOSE_FILE down prometheus grafana -v --remove-orphans
        docker system prune -f
        echo -e "${GREEN}✅ 清理完成${NC}"
    fi
}

# 启动服务
start_services() {
    echo -e "${BLUE}🚀 启动 BatchSQL 监控服务...${NC}"
    
    local compose_args=()
    if [[ $DETACH_MODE == true ]]; then
        compose_args+=("-d")
    fi

    docker compose -f $COMPOSE_FILE up prometheus grafana "${compose_args[@]}"
    
    if [[ $DETACH_MODE == true ]]; then
        echo -e "${GREEN}✅ 监控服务已启动${NC}"
        
        # 等待服务启动
        echo -e "${YELLOW}⏳ 等待服务启动...${NC}"
        sleep 10
        
        # 检查服务状态
        echo -e "${BLUE}📊 服务状态:${NC}"
        docker compose -f $COMPOSE_FILE ps prometheus grafana
        
        # 显示访问信息
        echo ""
        echo -e "${GREEN}🎉 监控服务已就绪！${NC}"
        echo ""
        echo -e "${BLUE}📊 访问地址:${NC}"
        echo "  • Grafana 仪表板:    http://localhost:3000 (admin/admin)"
        echo "  • Prometheus 控制台: http://localhost:9091"
        echo "  • BatchSQL 指标:     http://localhost:9090/metrics"
        echo "  • 健康检查:          http://localhost:9090/health"
        echo ""
        echo -e "${YELLOW}💡 提示:${NC}"
        echo "  • 运行集成测试以开始收集指标"
        echo "  • 使用 '$0 --logs' 查看实时日志"
        echo "  • 使用 '$0 --stop' 停止服务"
        echo ""
        
        # 显示日志（如果请求）
        if [[ $SHOW_LOGS == true ]]; then
            echo -e "${BLUE}📋 实时日志 (Ctrl+C 退出):${NC}"
            docker compose -f $COMPOSE_FILE logs -f prometheus grafana
        fi
    fi
}

# 主函数
main() {
    echo -e "${GREEN}🔧 BatchSQL Prometheus 监控启动器${NC}"
    echo ""
    
    check_dependencies
    check_config_files
    cleanup_services
    start_services
}

# 错误处理
trap 'echo -e "${RED}❌ 脚本执行失败${NC}"; exit 1' ERR

# 执行主函数
main "$@"