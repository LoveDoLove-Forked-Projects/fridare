# Fridare DEB包创建功能

## 概述

Fridare现在支持创建新的Frida DEB包，包括GUI界面和命令行工具两种方式。

## 主要功能

### 1. 创建新DEB包
- ✅ 支持从原始frida-server文件创建DEB包
- ✅ 支持二进制文件hex替换 (魔改功能)
- ✅ 支持Root和Rootless两种结构
- ✅ 支持frida-agent.dylib文件处理
- ✅ 自动生成LaunchDaemon配置文件
- ✅ 自动生成安装/卸载脚本

### 2. 结构类型支持

#### Root结构 (传统越狱)
- 文件位置: `/usr/sbin/`, `/usr/lib/`, `/Library/LaunchDaemons/`
- 适用于: 传统越狱环境
- plist文件包含: `_MSSafeMode`, `LimitLoadToSessionType`

#### Rootless结构 (现代越狱)  
- 文件位置: `/var/jb/usr/sbin/`, `/var/jb/usr/lib/`, `/var/jb/Library/LaunchDaemons/`
- 适用于: checkra1n, unc0ver等现代越狱
- plist文件简化配置，去除`_MSSafeMode`等

### 3. 二进制修改功能
- 使用HexReplacer进行字符串替换
- 支持frida-server和frida-agent.dylib文件
- 保持文件权限和结构完整性
- 实时进度显示

## 使用方法

### GUI界面

1. 打开Fridare GUI程序
2. 点击 "🆕 创建DEB包" 标签页
3. 选择所需文件和配置:
   - **frida-server文件**: 必需，原始frida-server二进制文件
   - **frida-agent文件**: 可选，frida-agent.dylib库文件
   - **输出路径**: DEB文件保存位置
   - **魔改名称**: 5个字符的替换名称 (如: `agent`, `myapp`)
   - **端口**: 服务端口 (默认: 27042)
   - **结构类型**: 选择Root或Rootless
4. 配置包信息 (可选，有默认值)
5. 点击 "创建DEB包" 按钮

### 命令行工具

```bash
# 基本用法
fridare-create.exe -server frida-server -magic agent -output agent.deb

# 创建Rootless结构
fridare-create.exe -server frida-server -magic agent -rootless -output agent-rootless.deb

# 包含agent库文件
fridare-create.exe -server frida-server -agent frida-agent.dylib -magic agent -output agent.deb

# 自定义端口和包信息
fridare-create.exe -server frida-server -magic myapp -port 27043 -name com.example.myapp -version 1.0.0 -output myapp.deb

# 查看所有选项
fridare-create.exe -help
```

### 参数说明

| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `-server` | ✅ | - | frida-server文件路径 |
| `-magic` | ✅ | - | 5字符魔改名称 (字母开头，字母数字组合) |
| `-output` | ✅ | - | 输出DEB文件路径 |
| `-agent` | ❌ | - | frida-agent.dylib文件路径 |
| `-rootless` | ❌ | false | 是否为rootless结构 |
| `-port` | ❌ | 27042 | 服务端口 |
| `-name` | ❌ | 自动生成 | 包名 |
| `-version` | ❌ | 17.2.17 | 版本号 |
| `-arch` | ❌ | iphoneos-arm64 | 架构 |
| `-maintainer` | ❌ | Fridare Team | 维护者 |
| `-desc` | ❌ | 自动生成 | 包描述 |
| `-depends` | ❌ | firmware (>= 12.0) | 依赖 |

## 生成的文件结构

### Root结构
```
usr/
  sbin/
    {magic_name}              # 修改后的frida-server
  lib/
    {magic_name}/
      {magic_name}-agent.dylib # 修改后的agent库 (如果提供)
Library/
  LaunchDaemons/
    re.{magic_name}.server.plist
```

### Rootless结构  
```
var/
  re/                         # 使用安全路径避免检测
    usr/
      sbin/
        {magic_name}          # 修改后的frida-server
      lib/
        {magic_name}/
          {magic_name}-agent.dylib # 修改后的agent库 (如果提供)
    Library/
      LaunchDaemons/
        re.{magic_name}.server.plist
```

## 安装和使用

### 安装DEB包
```bash
dpkg -i your_package.deb
```

### 控制服务

#### Root结构
```bash
# 启动服务
launchctl load /Library/LaunchDaemons/re.{magic_name}.server.plist

# 停止服务  
launchctl unload /Library/LaunchDaemons/re.{magic_name}.server.plist
```

#### Rootless结构
```bash
# 启动服务
launchctl load /var/jb/Library/LaunchDaemons/re.{magic_name}.server.plist

# 停止服务
launchctl unload /var/jb/Library/LaunchDaemons/re.{magic_name}.server.plist
```

### 连接使用
```bash
# 使用自定义端口连接
frida -H <设备IP>:<端口> <进程名>

# 例如：端口27043
frida -H 192.168.1.100:27043 SpringBoard
```

## 技术特性

### 二进制修改
- 使用HexReplacer技术替换字符串
- 支持MachO、ELF、PE格式
- 保持文件结构和权限
- 实时进度反馈

### DEB包构建
- 纯Go实现，无需外部dpkg工具
- AR格式写入器
- XZ压缩支持  
- 完整的control、postinst、prerm脚本

### 兼容性
- 支持iOS 12.0+
- ARM64和ARM架构
- Root和Rootless越狱环境
- 自动路径映射和权限设置

## 注意事项

1. **魔改名称要求**: 
   - 必须是5个字符
   - 以字母开头
   - 只能包含字母和数字

2. **文件权限**: 
   - 自动设置正确的文件权限
   - frida-server: 755 (可执行)
   - agent库: 755 (可执行)  
   - plist文件: 644 (只读)

3. **端口配置**:
   - 默认端口27042
   - 自定义端口会自动添加到启动参数
   - 确保端口未被占用

4. **路径安全**:
   - Rootless结构使用`/var/re`避免敏感词检测
   - 自动处理路径映射和转换

## 错误修复

### Bug修复记录

#### 1. ✅ TAR结构修复
**问题**: rootless结构TAR打包时根目录条目处理错误
**修复**: 所有结构都使用`./`前缀，包括rootless的`./var/re/...`

#### 2. ✅ createPostInstScript功能改进  
**问题**: 原版本路径和权限设置不准确
**修复**: 
- 分离Root和Rootless脚本逻辑
- 修正二进制文件路径（usr/sbin而非usr/bin）
- 添加dylib文件权限设置
- 改进错误处理（使用`|| true`避免脚本失败）

#### 3. ✅ 包信息结构扩展
**问题**: PackageInfo缺少IsRootless字段
**修复**: 添加IsRootless布尔字段支持结构类型判断

## 构建说明

```bash
# 构建所有程序
make build

# 单独构建
make build-gui      # GUI程序
make build-create   # 创建工具
make build-debug    # Debug程序

# 运行测试
./build/fridare-create.exe -help
./build/fridare-gui.exe
```

---

**作者**: Fridare Team  
**版本**: 1.0.0  
**更新**: 2025年8月29日
