# BatchSQL Makefile

.PHONY: test test-unit test-integration test-stress docker-test clean help

# 默认目标
all: test

# 单元测试
test-unit:
	@echo "🧪 运行单元测试..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 单元测试完成，覆盖率报告: coverage.html"

# 快速测试（无覆盖率）
test:
	@echo "🚀 运行快速测试..."
	go test -v ./...

# 基准测试
bench:
	@echo "⚡ 运行性能基准测试..."
	go test -bench=. -benchmem ./...

# Docker集成测试
test-integration:
	@echo "🐳 启动Docker集成测试..."
	docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	docker-compose -f docker-compose.test.yml down

# 压力测试
test-stress:
	@echo "💪 运行压力测试..."
	chmod +x test/scripts/stress-test.sh
	./test/scripts/stress-test.sh

# 清理
clean:
	@echo "🧹 清理测试文件..."
	rm -f coverage.out coverage.html
	docker-compose -f docker-compose.test.yml down -v
	go clean -testcache

# 代码格式化
fmt:
	@echo "🎨 格式化代码..."
	go fmt ./...
	go mod tidy

# 代码检查
lint:
	@echo "🔍 代码检查..."
	go vet ./...
	golangci-lint run

# 构建示例
build-examples:
	@echo "🔨 构建示例程序..."
	go build -o bin/simple_demo examples/simple_demo.go
	go build -o bin/prometheus_example examples/prometheus_example.go
	go build -o bin/stress_test test/cmd/stress/main.go

# 运行示例
run-demo:
	@echo "🎯 运行演示程序..."
	go run examples/simple_demo.go

# 帮助信息
help:
	@echo "BatchSQL 测试命令:"
	@echo "  make test          - 运行快速测试"
	@echo "  make test-unit     - 运行单元测试（含覆盖率）"
	@echo "  make test-integration - 运行Docker集成测试"
	@echo "  make test-stress   - 运行压力测试"
	@echo "  make bench         - 运行性能基准测试"
	@echo "  make fmt           - 格式化代码"
	@echo "  make lint          - 代码检查"
	@echo "  make clean         - 清理测试文件"
	@echo "  make run-demo      - 运行演示程序"
	@echo "  make help          - 显示帮助信息"