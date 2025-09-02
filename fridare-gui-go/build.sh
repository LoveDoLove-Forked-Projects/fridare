#!/bin/bash
# Fridare GUI 构建脚本

set -e

echo "=== Fridare GUI 构建工具 ==="
echo ""

# 进入项目目录
cd "$(dirname "$0")"

# 清理旧的构建文件
echo "清理旧的构建文件..."
rm -f build/fridare-gui.exe
rm -f build/fridare-create.exe
rm -f build/fridare-patch.exe

# 使用 fyne build 构建（包含更好的图标和资源打包）
echo "构建应用程序..."
fyne build --src cmd/gui -o ../../build/fridare-gui.exe
go build -o build/fridare-create.exe cmd/create/main.go
go build -o build/fridare-patch.exe cmd/patch/main.go

echo ""
echo "✅ 构建完成！"
echo ""
echo "生成的文件："
ls -la build/fridare-gui.exe
ls -la build/fridare-create.exe
ls -la build/fridare-patch.exe

echo ""
echo "运行应用程序："
echo "  ./build/fridare-gui.exe"

./build/fridare-gui.exe