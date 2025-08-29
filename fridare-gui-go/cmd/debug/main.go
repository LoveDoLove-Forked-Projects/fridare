package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"fridare-gui/internal/core"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("用法: debug.exe <输入DEB文件> <输出DEB文件> <魔改名称> [端口]")
		fmt.Println("示例: debug.exe frida_17.2.17_iphoneos-arm64.deb frida_modified.deb test-frida 27042")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]
	magicName := os.Args[3]
	port := 27042
	if len(os.Args) > 4 {
		fmt.Sscanf(os.Args[4], "%d", &port)
	}

	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Printf("开始DEB包修改过程:\n")
	fmt.Printf("输入文件: %s\n", inputPath)
	fmt.Printf("输出文件: %s\n", outputPath)
	fmt.Printf("魔改名称: %s\n", magicName)
	fmt.Printf("端口: %d\n", port)
	fmt.Println()

	// 检查输入文件是否存在
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Fatalf("输入文件不存在: %s", inputPath)
	}

	// 创建输出目录
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	// 创建DEB修改器
	modifier := core.NewDebModifier(inputPath, outputPath, magicName, port)

	// 进度回调函数
	progressCallback := func(progress float64, message string) {
		fmt.Printf("[%.0f%%] %s\n", progress*100, message)
	}

	// 执行修改
	err := modifier.ModifyDebPackage(progressCallback)
	if err != nil {
		log.Fatalf("DEB包修改失败: %v", err)
	}

	fmt.Println("\nDEB包修改成功完成!")

	// 显示输出文件信息
	if stat, err := os.Stat(outputPath); err == nil {
		fmt.Printf("输出文件大小: %d 字节 (%.2f KB)\n", stat.Size(), float64(stat.Size())/1024)
	}
}
