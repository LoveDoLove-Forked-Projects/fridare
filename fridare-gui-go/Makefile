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

# 本地构建 - 构建所有版本
.PHONY: build
build: build-gui build-create build-patch

# 构建GUI程序
.PHONY: build-gui
build-gui:
	@if [ ! -d $(BUILD_DIR) ]; then mkdir $(BUILD_DIR); fi
	@echo "构建GUI应用程序..."
	@cd cmd/gui && fyne build -o ../../$(BUILD_DIR)/$(APP_NAME).exe
	@echo "✅ GUI构建完成: $(BUILD_DIR)/$(APP_NAME).exe"

# 构建创建DEB包命令行工具
.PHONY: build-create
build-create:
	@if [ ! -d $(BUILD_DIR) ]; then mkdir $(BUILD_DIR); fi
	@echo "构建创建DEB包工具..."
	@go build $(GO_FLAGS) -o $(BUILD_DIR)/fridare-create.exe cmd/create/main.go
	@echo "✅ 创建工具构建完成: $(BUILD_DIR)/fridare-create.exe"

# 构建DEB包修改/补丁工具
.PHONY: build-patch
build-patch:
	@if [ ! -d $(BUILD_DIR) ]; then mkdir $(BUILD_DIR); fi
	@echo "构建DEB包修改工具..."
	@go build $(GO_FLAGS) -o $(BUILD_DIR)/fridare-patch.exe cmd/patch/main.go
	@echo "✅ 修改工具构建完成: $(BUILD_DIR)/fridare-patch.exe"

# 兼容性目标 (保持向后兼容)
.PHONY: build-debug
build-debug: build-patch

# 运行GUI程序
.PHONY: run
run: build-gui
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
	
	# Windows AMD64 - GUI
	@echo "Building GUI for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/gui
	
	# Windows AMD64 - CLI Tools  
	@echo "Building CLI Tools for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-create-windows-amd64.exe ./cmd/create
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-patch-windows-amd64.exe ./cmd/patch
	
	# Windows ARM64 - GUI
	@echo "Building GUI for Windows ARM64..."
	@GOOS=windows GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-arm64.exe ./cmd/gui
	
	# Windows ARM64 - CLI Tools
	@echo "Building CLI Tools for Windows ARM64..."
	@GOOS=windows GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-create-windows-arm64.exe ./cmd/create
	@GOOS=windows GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-patch-windows-arm64.exe ./cmd/patch
	
	# macOS AMD64 - GUI
	@echo "Building GUI for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-macos-amd64 ./cmd/gui
	
	# macOS AMD64 - CLI Tools
	@echo "Building CLI Tools for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-create-macos-amd64 ./cmd/create
	@GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-patch-macos-amd64 ./cmd/patch
	
	# macOS ARM64 - GUI
	@echo "Building GUI for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-macos-arm64 ./cmd/gui
	
	# macOS ARM64 - CLI Tools
	@echo "Building CLI Tools for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-create-macos-arm64 ./cmd/create
	@GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-patch-macos-arm64 ./cmd/patch
	
	# Linux AMD64 - GUI
	@echo "Building GUI for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./cmd/gui
	
	# Linux AMD64 - CLI Tools
	@echo "Building CLI Tools for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-create-linux-amd64 ./cmd/create
	@GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-patch-linux-amd64 ./cmd/patch
	
	# Linux ARM64 - GUI
	@echo "Building GUI for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 ./cmd/gui
	
	# Linux ARM64 - CLI Tools  
	@echo "Building CLI Tools for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-create-linux-arm64 ./cmd/create
	@GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $(DIST_DIR)/fridare-patch-linux-arm64 ./cmd/patch

# 使用 fyne package 打包
.PHONY: package
package: deps
	@mkdir -p $(DIST_DIR)
	
	# 安装 fyne 命令行工具
	go install fyne.io/fyne/v2/cmd/fyne@latest
	
	# Windows
	fyne package -os windows -name $(APP_NAME) -src ./cmd/gui -output $(DIST_DIR)/$(APP_NAME).exe
	
	# macOS
	fyne package -os darwin -name $(APP_NAME) -src ./cmd/gui -output $(DIST_DIR)/$(APP_NAME).app
	
	# Linux
	fyne package -os linux -name $(APP_NAME) -src ./cmd/gui -output $(DIST_DIR)/$(APP_NAME)

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
	@echo "Fridare 构建系统 - 可用的目标:"
	@echo ""
	@echo "主要构建目标:"
	@echo "  build       - 构建所有版本 (GUI + CLI工具)"
	@echo "  build-gui   - 仅构建GUI版本"
	@echo "  build-create- 仅构建DEB包创建工具"
	@echo "  build-patch - 仅构建DEB包修改工具"
	@echo "  run         - 构建并运行GUI版本"
	@echo ""
	@echo "高级构建目标:"
	@echo "  build-all   - 跨平台构建所有版本"
	@echo "  package     - 使用fyne package打包GUI版本"
	@echo ""
	@echo "开发工具:"
	@echo "  deps        - 安装依赖"
	@echo "  test        - 运行测试"
	@echo "  clean       - 清理构建文件"
	@echo "  fmt         - 格式化代码"
	@echo "  vet         - 代码检查"
	@echo ""
	@echo "设置工具:"
	@echo "  mod-init    - 初始化Go模块"
	@echo "  dev-setup   - 设置开发环境"
	@echo "  help        - 显示帮助"
	@echo ""
	@echo "项目结构:"
	@echo "  cmd/gui/     - GUI版本主程序"
	@echo "  cmd/create/  - DEB包创建CLI工具"
	@echo "  cmd/patch/   - DEB包修改CLI工具"
