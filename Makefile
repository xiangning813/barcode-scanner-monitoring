# Makefile for Barcode Scanner Project

# 变量定义
APP_NAME=barcode-scanner
MAIN_PATH=./cmd/scanner
BUILD_DIR=./build
CONFIG_DIR=./configs
LOGS_DIR=./logs

# Go 相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# 构建标志
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty) -X main.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"

# 默认目标
.PHONY: all
all: clean deps fmt vet test build

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 格式化代码
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# 代码检查
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 构建应用程序
.PHONY: build
build: create-dirs
	@echo "Building application..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME).exe $(MAIN_PATH)

# 构建 Linux 版本
.PHONY: build-linux
build-linux: create-dirs
	@echo "Building for Linux..."
	set GOOS=linux&& set GOARCH=amd64&& $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux $(MAIN_PATH)

# 构建 macOS 版本
.PHONY: build-darwin
build-darwin: create-dirs
	@echo "Building for macOS..."
	set GOOS=darwin&& set GOARCH=amd64&& $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin $(MAIN_PATH)

# 构建所有平台版本
.PHONY: build-all
build-all: build build-linux build-darwin

# 创建必要的目录
.PHONY: create-dirs
create-dirs:
	@if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
	@if not exist "$(LOGS_DIR)" mkdir "$(LOGS_DIR)"

# 运行应用程序
.PHONY: run
run: build
	@echo "Running application..."
	$(BUILD_DIR)\$(APP_NAME).exe

# 开发模式运行（直接运行源码）
.PHONY: dev
dev:
	@echo "Running in development mode..."
	$(GOCMD) run $(MAIN_PATH)

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@if exist "$(BUILD_DIR)" rmdir /s /q "$(BUILD_DIR)"
	@if exist "coverage.out" del "coverage.out"
	@if exist "coverage.html" del "coverage.html"

# 安装应用程序到系统
.PHONY: install
install: build
	@echo "Installing application..."
	copy "$(BUILD_DIR)\$(APP_NAME).exe" "C:\Program Files\$(APP_NAME)\$(APP_NAME).exe"

# 生成文档
.PHONY: docs
docs:
	@echo "Generating documentation..."
	$(GOCMD) doc -all ./... > docs/api.md

# 代码质量检查
.PHONY: lint
lint:
	@echo "Running linter..."
	@where golangci-lint >nul 2>&1 && golangci-lint run || echo "golangci-lint not found, skipping..."

# 安全检查
.PHONY: security
security:
	@echo "Running security check..."
	@where gosec >nul 2>&1 && gosec ./... || echo "gosec not found, skipping..."

# 性能测试
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# 初始化项目（首次运行）
.PHONY: init
init: create-dirs deps
	@echo "Initializing project..."
	@if not exist "$(CONFIG_DIR)\config.yaml" (
		echo Creating default config file...
		copy "$(CONFIG_DIR)\config.yaml.example" "$(CONFIG_DIR)\config.yaml" 2>nul || echo Config file already exists
	)
	@echo "Project initialized successfully!"
	@echo "You can now run: make dev"

# 数据库相关命令
.PHONY: db-migrate
db-migrate:
	@echo "Running database migration..."
	$(GOCMD) run $(MAIN_PATH) -migrate

.PHONY: db-seed
db-seed:
	@echo "Seeding database..."
	$(GOCMD) run $(MAIN_PATH) -seed

.PHONY: db-reset
db-reset:
	@echo "Resetting database..."
	@if exist "barcode_scanner.db" del "barcode_scanner.db"
	$(MAKE) db-migrate
	$(MAKE) db-seed

# Docker 相关命令
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 8081:8081 $(APP_NAME):latest

# 发布相关
.PHONY: release
release: clean test build-all
	@echo "Creating release package..."
	@if not exist "release" mkdir "release"
	tar -czf release/$(APP_NAME)-windows.tar.gz -C $(BUILD_DIR) $(APP_NAME).exe
	tar -czf release/$(APP_NAME)-linux.tar.gz -C $(BUILD_DIR) $(APP_NAME)-linux
	tar -czf release/$(APP_NAME)-darwin.tar.gz -C $(BUILD_DIR) $(APP_NAME)-darwin
	@echo "Release packages created in release/ directory"

# 帮助信息
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make all          - Run full build pipeline (clean, deps, fmt, vet, test, build)"
	@echo "  make deps         - Install dependencies"
	@echo "  make fmt          - Format code"
	@echo "  make vet          - Run go vet"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage- Run tests with coverage"
	@echo "  make build        - Build application"
	@echo "  make build-all    - Build for all platforms"
	@echo "  make run          - Build and run application"
	@echo "  make dev          - Run in development mode"
	@echo "  make clean        - Clean build files"
	@echo "  make init         - Initialize project"
	@echo "  make lint         - Run linter (requires golangci-lint)"
	@echo "  make security     - Run security check (requires gosec)"
	@echo "  make bench        - Run benchmarks"
	@echo "  make docs         - Generate documentation"
	@echo "  make db-migrate   - Run database migration"
	@echo "  make db-seed      - Seed database"
	@echo "  make db-reset     - Reset database"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"
	@echo "  make release      - Create release packages"
	@echo "  make help         - Show this help message"

# 快速开始
.PHONY: quick-start
quick-start: init dev

# 检查环境
.PHONY: check-env
check-env:
	@echo "Checking environment..."
	@$(GOCMD) version
	@echo "Go environment OK"
	@where git >nul 2>&1 && echo "Git found" || echo "Git not found"
	@where make >nul 2>&1 && echo "Make found" || echo "Make not found"

# 开发者工具安装
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOCMD) install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "Development tools installed"

# 项目状态检查
.PHONY: status
status:
	@echo "Project Status:"
	@echo "==============="
	@echo "Go version: $(shell $(GOCMD) version)"
	@echo "Module: $(shell $(GOCMD) list -m)"
	@echo "Dependencies: $(shell $(GOCMD) list -m all | wc -l) modules"
	@echo "Source files: $(shell find . -name '*.go' | wc -l) files"
	@if exist "$(BUILD_DIR)\$(APP_NAME).exe" (echo "Build status: Built") else (echo "Build status: Not built")
	@echo "==============="