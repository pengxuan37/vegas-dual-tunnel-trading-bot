# Vegas Dual Tunnel Trading Bot Makefile

.PHONY: build run test clean deps fmt vet lint install-tools

# 变量定义
APP_NAME=vegas-trading-bot
BUILD_DIR=./bin
MAIN_FILE=./main.go
COVER_FILE=coverage.out

# 默认目标
all: deps fmt vet build

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# 构建应用
build:
	@echo "Building application..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# 运行应用
run:
	@echo "Running application..."
	go run $(MAIN_FILE)

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=$(COVER_FILE) ./...
	go tool cover -html=$(COVER_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 静态分析
vet:
	@echo "Running go vet..."
	go vet ./...

# 代码检查（需要安装golangci-lint）
lint:
	@echo "Running linter..."
	golangci-lint run

# 安装开发工具
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 清理构建文件
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f $(COVER_FILE)
	rm -f coverage.html

# 创建必要的目录
init-dirs:
	@echo "Creating necessary directories..."
	mkdir -p data
	mkdir -p logs

# 初始化配置文件
init-config:
	@echo "Initializing config file..."
	@if [ ! -f config.yaml ]; then \
		cp config.yaml.example config.yaml; \
		echo "Config file created: config.yaml"; \
		echo "Please edit config.yaml with your actual configuration"; \
	else \
		echo "Config file already exists"; \
	fi

# 完整初始化
init: init-dirs init-config deps
	@echo "Project initialization completed"

# 开发模式运行（监听文件变化）
dev:
	@echo "Starting development mode..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without hot reload..."; \
		go run $(MAIN_FILE); \
	fi

# 安装air（热重载工具）
install-air:
	@echo "Installing air for hot reload..."
	go install github.com/cosmtrek/air@latest

# 显示帮助信息
help:
	@echo "Available targets:"
	@echo "  all          - Run deps, fmt, vet, and build"
	@echo "  deps         - Install dependencies"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run linter (requires golangci-lint)"
	@echo "  clean        - Clean build files"
	@echo "  init         - Initialize project (dirs, config, deps)"
	@echo "  dev          - Run in development mode with hot reload"
	@echo "  install-tools- Install development tools"
	@echo "  install-air  - Install air for hot reload"
	@echo "  help         - Show this help message"