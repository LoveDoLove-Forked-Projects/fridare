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

# 使用 fyne build 构建（包含更好的图标和资源打包）
echo "构建应用程序..."
cd cmd && fyne build -o ../build/fridare-gui.exe && cd ..

echo ""
echo "✅ 构建完成！"
echo ""
echo "生成的文件："
ls -la build/fridare-gui.exe

echo ""
echo "运行应用程序："
echo "  ./build/fridare-gui.exe"

./build/fridare-gui.exe