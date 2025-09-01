# Fridare GUI Makefile

APP_NAME := fridare-gui
VERSION := 1.0.0
BUILD_DIR := build
DIST_DIR := dist

# Go编译参数
GO_FLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

# 默认目标
.PHONY: all
all: deps build

# 安装依赖
.PHONY: deps
deps:
	go mod download
	go mod tidy

# 本地构建
.PHONY: build
build: build-gui build-create build-debug

# 构建GUI程序
.PHONY: build-gui
build-gui:
	@mkdir -p $(BUILD_DIR)
	@echo "构建GUI应用程序..."
	@cd cmd && fyne build -o ../$(BUILD_DIR)/$(APP_NAME).exe
	@echo "✅ GUI构建完成: $(BUILD_DIR)/$(APP_NAME).exe"

# 构建创建DEB包命令行工具
.PHONY: build-create
build-create:
	@mkdir -p $(BUILD_DIR)
	@echo "构建创建DEB包工具..."
	@go build $(GO_FLAGS) -o $(BUILD_DIR)/fridare-create.exe cmd/create/main.go
	@echo "✅ 创建工具构建完成: $(BUILD_DIR)/fridare-create.exe"

# 构建debug程序
.PHONY: build-debug
build-debug:
	@mkdir -p $(BUILD_DIR)
	@echo "构建debug程序..."
	@go build $(GO_FLAGS) -o $(BUILD_DIR)/debug.exe cmd/debug/main.go
	@echo "✅ Debug程序构建完成: $(BUILD_DIR)/debug.exe"

# 运行程序
.PHONY: run
run: build
	./$(BUILD_DIR)/$(APP_NAME).exe

# 清理
.PHONY: clean
clean:
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)

# 跨平台构建
.PHONY: build-all
build-all: clean deps
	@mkdir -p $(DIST_DIR)
	
	# Windows AMD64
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd
	
	# Windows ARM64
	@echo "Building for Windows ARM64..."
	@GOOS=windows GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-arm64.exe ./cmd
	
	# macOS AMD64
	@echo "Building for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-macos-amd64 ./cmd
	
	# macOS ARM64
	@echo "Building for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-macos-arm64 ./cmd
	
	# Linux AMD64
	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./cmd
	
	# Linux ARM64
	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 ./cmd

# 使用 fyne package 打包
.PHONY: package
package: deps
	@mkdir -p $(DIST_DIR)
	
	# 安装 fyne 命令行工具
	go install fyne.io/fyne/v2/cmd/fyne@latest
	
	# Windows
	fyne package -os windows -name $(APP_NAME) -src ./cmd -output $(DIST_DIR)/$(APP_NAME).exe
	
	# macOS
	fyne package -os darwin -name $(APP_NAME) -src ./cmd -output $(DIST_DIR)/$(APP_NAME).app
	
	# Linux
	fyne package -os linux -name $(APP_NAME) -src ./cmd -output $(DIST_DIR)/$(APP_NAME)

# 测试
.PHONY: test
test:
	go test -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	go fmt ./...

# 代码检查
.PHONY: vet
vet:
	go vet ./...

# 初始化Go模块
.PHONY: mod-init
mod-init:
	go mod init fridare-gui
	go get fyne.io/fyne/v2/app
	go get fyne.io/fyne/v2/widget
	go get fyne.io/fyne/v2/container
	go get fyne.io/fyne/v2/theme
	go get github.com/go-resty/resty/v2
	go get gopkg.in/yaml.v3

# 开发环境设置
.PHONY: dev-setup
dev-setup: mod-init deps
	@echo "开发环境设置完成"

# 帮助
.PHONY: help
help:
	@echo "可用的目标:"
	@echo "  deps        - 安装依赖"
	@echo "  build       - 本地构建"
	@echo "  run         - 构建并运行"
	@echo "  build-all   - 跨平台构建"
	@echo "  package     - 使用fyne package打包"
	@echo "  test        - 运行测试"
	@echo "  clean       - 清理构建文件"
	@echo "  fmt         - 格式化代码"
	@echo "  vet         - 代码检查"
	@echo "  mod-init    - 初始化Go模块"
	@echo "  dev-setup   - 设置开发环境"
	@echo "  help        - 显示帮助"
