# 构建问题解决方案

## 问题诊断

原始错误：
```
$ fyne build
# fridare-gui
runtime.main_main·f: function main is undeclared in the main package

exit status 1
```

## 根本原因

1. **资源文件冲突**：手动创建的 `internal/assets/icon.go` 文件与自动生成的 `appicon.go` 文件发生变量名冲突
2. **构建目录问题**：`fyne build` 命令默认在当前目录查找 `main.go`，但我们的 `main.go` 位于 `cmd/` 目录

## 解决步骤

### 1. 删除冲突的资源文件
```bash
rm internal/assets/icon.go
```

**原因**：手动创建的 `icon.go` 文件重新定义了 `AppIcon` 变量，但数据为空，与自动生成的 `appicon.go` 冲突。

### 2. 使用正确的构建方法

#### 方法一：标准 Go 构建
```bash
go build -o build/fridare-gui.exe ./cmd
```

#### 方法二：Fyne 优化构建
```bash
cd cmd && fyne build -o ../build/fridare-gui-fyne.exe
```

### 3. 自动化构建脚本
创建了 `build.sh` 脚本来简化构建过程：
```bash
./build.sh
```

## 文件结构说明

### 正确的资源文件结构
```
internal/assets/
├── appicon.go     # 自动生成的应用图标（通过 fyne bundle）
└── logo.go        # 自动生成的Logo资源（通过 fyne bundle）
```

### 项目结构
```
fridare-gui-go/
├── cmd/
│   └── main.go           # 应用程序入口点
├── internal/
│   ├── assets/           # 嵌入式资源
│   ├── config/           # 配置管理
│   └── ui/               # 用户界面
├── build/                # 构建输出目录
└── build.sh              # 构建脚本
```

## 构建输出

成功构建后会生成两个版本：

1. **fridare-gui.exe**：标准 Go 构建版本
2. **fridare-gui-fyne.exe**：Fyne 优化版本（推荐使用）

## 经验总结

1. **避免手动创建资源文件**：使用 `fyne bundle` 命令自动生成
2. **注意构建目录**：`fyne build` 需要在包含 `main.go` 的目录中运行
3. **使用构建脚本**：自动化构建过程，避免重复的错误
4. **版本管理**：保持自动生成的文件在版本控制中，避免手动修改

## 下次构建

推荐使用构建脚本：
```bash
./build.sh
```

这将生成单一的 `build/fridare-gui.exe` 文件。

手动构建（如果需要）：
```bash
cd cmd && fyne build -o ../build/fridare-gui.exe
```
