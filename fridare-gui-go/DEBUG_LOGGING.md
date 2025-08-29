# DEB包修改详细日志功能说明

## 概述
已为fridare DEB包修改功能添加了完整的调试日志系统，便于分析DEB文件生成和解压过程中的问题。

## 新增日志功能

### 1. 主要修改流程日志
- **ModifyDebPackage()**: 完整的7步修改流程日志
- 输入/输出文件大小对比
- 每个步骤的开始和完成状态
- 临时目录创建和清理过程

### 2. DEB文件解压日志 (extractDebWithGoAr)
- AR文件头部验证
- 每个AR条目的详细信息 (名称、大小)
- control.tar.xz 和 data.tar.xz 的提取过程
- 文件大小统计

### 3. TAR档案解压日志 (extractTarArchive)
- 压缩格式检测 (gzip/xz/未压缩)
- 每个TAR条目的处理过程
- 文件/目录创建详情
- 文件大小验证和统计

### 4. TAR档案创建日志 (createDataTarData/createControlTarData)
- 遍历源目录过程
- 每个文件/目录的添加详情
- 路径分隔符转换 (Windows -> Unix)
- 文件权限和时间戳处理
- 最终统计信息

### 5. XZ压缩日志 (compressWithXz)
- 原始数据大小
- 压缩后大小
- 压缩率计算
- 压缩过程中的错误检测

### 6. AR文件写入日志 (arWriter.writeFile)
- 每个AR条目的写入详情
- 头部和数据字节数验证
- 对齐填充处理
- 写入完成确认

### 7. DEB文件验证日志 (validateGeneratedDeb)
- 生成的DEB文件完整性检查
- AR格式验证
- XZ压缩格式验证
- TAR内容结构验证

## 调试工具

### 1. GUI版本 (fridare-gui-debug.exe)
```bash
go build -o build/fridare-gui-debug.exe cmd/main.go
./build/fridare-gui-debug.exe
```

### 2. 命令行调试版本 (debug.exe)
```bash
go build -o build/debug.exe cmd/debug/main.go
./build/debug.exe input.deb output.deb magic-name [port]
```

示例:
```bash
./build/debug.exe frida_17.2.17_iphoneos-arm64.deb frida_modified.deb test-frida 27042
```

## 日志级别说明

### DEBUG
- 详细的处理步骤
- 文件路径转换
- 格式检测结果
- 字节对齐处理

### INFO
- 主要流程节点
- 文件大小统计
- 压缩/解压完成
- 成功操作确认

### WARNING
- 非致命错误
- 大小不匹配
- 格式异常
- 兼容性问题

### ERROR
- 致命错误
- 文件操作失败
- 格式错误
- 权限问题

### SUCCESS
- 操作成功完成
- 最终结果确认

## 问题诊断指南

### 1. DEB文件大小异常
检查日志中的:
- 输入文件大小 vs 输出文件大小
- TAR档案大小统计
- XZ压缩率
- AR条目大小

### 2. data.tar解压失败
查看日志:
- XZ格式检测结果
- TAR条目结构
- 文件权限设置
- 路径处理过程

### 3. 文件结构问题
监控日志:
- 目录创建过程
- 文件复制详情
- 相对路径计算
- 符号链接处理

### 4. 压缩问题
观察日志:
- 压缩前后大小对比
- 压缩率计算
- XZ写入过程
- 格式验证结果

## 下一步调试建议

1. **运行命令行版本**获取详细日志
2. **对比原始和生成的DEB文件**结构
3. **检查TAR档案**的具体内容
4. **验证文件权限**和路径格式
5. **分析压缩过程**中的异常

这个日志系统将帮助精确定位DEB包修改过程中data.tar解压失败的具体原因。
