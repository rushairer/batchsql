# BatchSQL Makefile

.PHONY: help build test test-unit docker-mysql-test docker-postgres-test docker-sqlite-test docker-all-tests clean lint fmt vet

# 默认目标
help:
	@echo "BatchSQL - High-Performance Batch Processing Library"
	@echo ""
	@echo "Available targets:"
	@echo "  build                 - Build the library"
	@echo "  test                  - Run all tests"
	@echo "  test-unit            - Run unit tests only"
	@echo "  docker-mysql-test    - Run MySQL pressure test (30min)"
	@echo "  docker-postgres-test - Run PostgreSQL pressure test (30min)"
	@echo "  docker-sqlite-test   - Run SQLite pressure test (30min)"
	@echo "  docker-redis-test   - Run Redis pressure test (30min)"
	@echo "  docker-all-tests     - Run all database pressure tests"
	@echo "  lint                 - Run linter"
	@echo "  fmt                  - Format code"
	@echo "  vet                  - Run go vet"
	@echo "  clean                - Clean build artifacts and test data"
	@echo "  benchmark            - Run performance benchmarks"
	@echo "  coverage             - Generate test coverage report"
	@echo "  show-reports         - Show available test reports"

# 构建
build:
	@echo "🔨 Building BatchSQL..."
	go build -v ./...

# 单元测试
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -race -cover ./...

# 所有测试
test: test-unit
	@echo "✅ All tests completed"

# Docker 集成测试 - 单数据库高性能压力测试
docker-mysql-test:
	@echo "🐳 Starting MySQL pressure test..."
	docker-compose -f ./test/docker-compose.mysql.yml down -v --remove-orphans
	docker-compose -f ./test/docker-compose.mysql.yml build --no-cache
	docker-compose -f ./test/docker-compose.mysql.yml up --abort-on-container-exit --exit-code-from mysql-test

docker-postgres-test:
	@echo "🐳 Starting PostgreSQL pressure test..."
	docker-compose -f ./test/docker-compose.postgres.yml down -v --remove-orphans
	docker-compose -f ./test/docker-compose.postgres.yml build --no-cache
	docker-compose -f ./test/docker-compose.postgres.yml up --abort-on-container-exit --exit-code-from postgres-test

docker-sqlite-test:
	@echo "🐳 Starting SQLite pressure test..."
	docker-compose -f ./test/docker-compose.sqlite.yml down -v --remove-orphans
	docker-compose -f ./test/docker-compose.sqlite.yml build --no-cache
	docker-compose -f ./test/docker-compose.sqlite.yml up --abort-on-container-exit --exit-code-from sqlite-test

docker-redis-test:
	@echo "🐳 Starting Redis pressure test..."
	docker-compose -f ./test/docker-compose.redis.yml down -v --remove-orphans
	docker-compose -f ./test/docker-compose.redis.yml build --no-cache
	docker-compose -f ./test/docker-compose.redis.yml up --abort-on-container-exit --exit-code-from redis-test

docker-all-tests: docker-mysql-test docker-postgres-test docker-sqlite-test docker-redis-test
	@echo "🎉 All pressure tests completed!"
	@echo "📊 Check ./test/reports/ for detailed performance reports"

# 代码质量检查
lint:
	@echo "🔍 Running linter..."
	golangci-lint run

fmt:
	@echo "📝 Formatting code..."
	gofumpt -w .

vet:
	@echo "🔍 Running go vet..."
	go vet ./...

# 性能基准测试
benchmark:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./...

# 测试覆盖率
coverage:
	@echo "📊 Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# 清理
clean:
	@echo "🧹 Cleaning up..."
	go clean -testcache
	rm -f coverage.out coverage.html
	rm -rf test/reports/*
	docker-compose -f ./test/docker-compose.mysql.yml down -v --remove-orphans 2>/dev/null || true
	docker-compose -f ./test/docker-compose.postgres.yml down -v --remove-orphans 2>/dev/null || true
	docker-compose -f ./test/docker-compose.sqlite.yml down -v --remove-orphans 2>/dev/null || true
	docker-compose -f ./test/docker-compose.redis.yml down -v --remove-orphans 2>/dev/null || true

	docker system prune -f

# 完整的 CI/CD 流程
ci: fmt vet lint test-unit docker-mysql-test
	@echo "🎉 All CI checks passed!"

# 开发环境设置
dev-setup:
	@echo "🛠️ Setting up development environment..."
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install mvdan.cc/gofumpt@latest
	@echo "✅ Development environment ready!"

# 查看集成测试报告
show-reports:
	@echo "📊 Available test reports:"
	@ls -la test/reports/ 2>/dev/null || echo "No reports found. Run 'make docker-mysql-test' first."

# 性能分析
profile:
	@echo "🔬 Running performance profiling..."
	go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@echo "📊 Profiles generated: cpu.prof, mem.prof"
	@echo "Use 'go tool pprof cpu.prof' or 'go tool pprof mem.prof' to analyze"