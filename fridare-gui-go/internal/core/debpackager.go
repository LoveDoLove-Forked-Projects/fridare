package core

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DebPackager DEB包构建器
type DebPackager struct {
	TempDir string
}

// PackageInfo 包信息
type PackageInfo struct {
	Name         string
	Version      string
	Architecture string
	Maintainer   string
	Description  string
	Depends      string
	Section      string
	Priority     string
	Homepage     string
	Port         int
	MagicName    string
}

// NewDebPackager 创建新的DEB包构建器
func NewDebPackager() *DebPackager {
	return &DebPackager{}
}

// CreateDebPackage 创建DEB包
func (dp *DebPackager) CreateDebPackage(fridaFile, outputPath string, info *PackageInfo, progressCallback func(float64, string)) error {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "fridare_deb_*")
	if err != nil {
		return fmt.Errorf("创建临时目录失败: %v", err)
	}
	dp.TempDir = tempDir
	defer os.RemoveAll(tempDir)

	progressCallback(0.1, "创建临时目录...")

	// 检测Frida文件架构
	arch, err := dp.detectArchitecture(fridaFile)
	if err != nil {
		return fmt.Errorf("检测文件架构失败: %v", err)
	}
	info.Architecture = arch

	progressCallback(0.2, "检测文件架构: "+arch)

	// 创建包目录结构
	err = dp.createPackageStructure(tempDir, arch, info)
	if err != nil {
		return fmt.Errorf("创建包结构失败: %v", err)
	}

	progressCallback(0.3, "创建包目录结构...")

	// 复制Frida文件到正确位置
	err = dp.copyFridaFile(fridaFile, tempDir, arch, info)
	if err != nil {
		return fmt.Errorf("复制Frida文件失败: %v", err)
	}

	progressCallback(0.5, "复制Frida文件...")

	// 创建控制文件
	err = dp.createControlFile(tempDir, info)
	if err != nil {
		return fmt.Errorf("创建控制文件失败: %v", err)
	}

	progressCallback(0.6, "创建控制文件...")

	// 创建启动脚本
	err = dp.createLaunchDaemon(tempDir, arch, info)
	if err != nil {
		return fmt.Errorf("创建启动脚本失败: %v", err)
	}

	progressCallback(0.7, "创建启动脚本...")

	// 创建安装后脚本
	err = dp.createPostInstScript(tempDir, info)
	if err != nil {
		return fmt.Errorf("创建安装脚本失败: %v", err)
	}

	progressCallback(0.8, "创建安装脚本...")

	// 构建DEB包
	err = dp.buildDebPackage(tempDir, outputPath, info)
	if err != nil {
		return fmt.Errorf("构建DEB包失败: %v", err)
	}

	progressCallback(1.0, "DEB包构建完成")

	return nil
}

// detectArchitecture 检测文件架构
func (dp *DebPackager) detectArchitecture(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 读取文件头部信息
	header := make([]byte, 16)
	_, err = file.Read(header)
	if err != nil {
		return "", err
	}

	// 检测Mach-O格式 (iOS)
	if len(header) >= 4 {
		magic := string(header[:4])
		switch magic {
		case "\xca\xfe\xba\xbe": // FAT binary
			return "iphoneos-arm", nil
		case "\xcf\xfa\xed\xfe": // 64-bit little endian
			return "iphoneos-arm64", nil
		case "\xce\xfa\xed\xfe": // 32-bit little endian
			return "iphoneos-arm", nil
		case "\xfe\xed\xfa\xce": // 32-bit big endian
			return "iphoneos-arm", nil
		case "\xfe\xed\xfa\xcf": // 64-bit big endian
			return "iphoneos-arm64", nil
		}
	}

	// 默认返回arm64
	return "iphoneos-arm64", nil
}

// createPackageStructure 创建包目录结构
func (dp *DebPackager) createPackageStructure(tempDir, arch string, info *PackageInfo) error {
	dirs := []string{
		"DEBIAN",
		"usr/bin",
		"Library/LaunchDaemons",
	}

	// 根据架构创建不同的目录结构
	if strings.Contains(arch, "arm64") {
		dirs = append(dirs, "var/jb/usr/bin", "var/jb/Library/LaunchDaemons")
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// copyFridaFile 复制Frida文件到正确位置
func (dp *DebPackager) copyFridaFile(srcFile, tempDir, arch string, info *PackageInfo) error {
	var destPaths []string

	// 根据架构确定目标路径
	if strings.Contains(arch, "arm64") {
		destPaths = []string{
			filepath.Join(tempDir, "var/jb/usr/bin", info.MagicName),
			filepath.Join(tempDir, "usr/bin", info.MagicName),
		}
	} else {
		destPaths = []string{
			filepath.Join(tempDir, "usr/bin", info.MagicName),
		}
	}

	// 复制文件到所有目标路径
	for _, destPath := range destPaths {
		err := dp.copyFile(srcFile, destPath)
		if err != nil {
			return err
		}

		// 设置可执行权限
		err = os.Chmod(destPath, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

// copyFile 复制文件
func (dp *DebPackager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 确保目录存在
	err = os.MkdirAll(filepath.Dir(dst), 0755)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// createControlFile 创建控制文件
func (dp *DebPackager) createControlFile(tempDir string, info *PackageInfo) error {
	controlPath := filepath.Join(tempDir, "DEBIAN", "control")

	// 计算安装大小
	installedSize, err := dp.calculateInstalledSize(tempDir)
	if err != nil {
		installedSize = 100 // 默认值
	}

	content := fmt.Sprintf(`Package: %s
Version: %s
Architecture: %s
Maintainer: %s
Installed-Size: %d
Depends: %s
Section: %s
Priority: %s
Homepage: %s
Description: %s
 Magic name: %s
 Default port: %d
`,
		info.Name,
		info.Version,
		info.Architecture,
		info.Maintainer,
		installedSize,
		info.Depends,
		info.Section,
		info.Priority,
		info.Homepage,
		info.Description,
		info.MagicName,
		info.Port,
	)

	return os.WriteFile(controlPath, []byte(content), 0644)
}

// createLaunchDaemon 创建启动守护程序配置
func (dp *DebPackager) createLaunchDaemon(tempDir, arch string, info *PackageInfo) error {
	var plistPaths []string

	// 根据架构确定plist文件路径
	if strings.Contains(arch, "arm64") {
		plistPaths = []string{
			filepath.Join(tempDir, "var/jb/Library/LaunchDaemons", fmt.Sprintf("re.%s.server.plist", info.MagicName)),
			filepath.Join(tempDir, "Library/LaunchDaemons", fmt.Sprintf("re.%s.server.plist", info.MagicName)),
		}
	} else {
		plistPaths = []string{
			filepath.Join(tempDir, "Library/LaunchDaemons", fmt.Sprintf("re.%s.server.plist", info.MagicName)),
		}
	}

	// plist内容
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>re.%s.server</string>
	<key>ProgramArguments</key>
	<array>
		<string>/usr/bin/%s</string>
		<string>-l</string>
		<string>0.0.0.0:%d</string>
	</array>
	<key>UserName</key>
	<string>root</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`, info.MagicName, info.MagicName, info.Port)

	// 写入所有plist文件
	for _, plistPath := range plistPaths {
		err := os.WriteFile(plistPath, []byte(plistContent), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// createPostInstScript 创建安装后脚本
func (dp *DebPackager) createPostInstScript(tempDir string, info *PackageInfo) error {
	postinstPath := filepath.Join(tempDir, "DEBIAN", "postinst")

	content := fmt.Sprintf(`#!/bin/bash
set -e

# 设置可执行权限
chmod 755 /usr/bin/%s
chown root:wheel /usr/bin/%s

# 如果是rootless环境，也设置jailbreak目录权限
if [ -f /var/jb/usr/bin/%s ]; then
    chmod 755 /var/jb/usr/bin/%s
    chown root:wheel /var/jb/usr/bin/%s
fi

# 加载守护程序
if [ -f /Library/LaunchDaemons/re.%s.server.plist ]; then
    launchctl load /Library/LaunchDaemons/re.%s.server.plist
fi

if [ -f /var/jb/Library/LaunchDaemons/re.%s.server.plist ]; then
    launchctl load /var/jb/Library/LaunchDaemons/re.%s.server.plist
fi

echo "Fridare %s 安装完成"
echo "服务已启动在端口 %d"
`,
		info.MagicName, info.MagicName,
		info.MagicName, info.MagicName, info.MagicName,
		info.MagicName, info.MagicName,
		info.MagicName, info.MagicName,
		info.MagicName, info.Port,
	)

	err := os.WriteFile(postinstPath, []byte(content), 0755)
	if err != nil {
		return err
	}

	// 创建卸载前脚本
	prermPath := filepath.Join(tempDir, "DEBIAN", "prerm")
	prermContent := fmt.Sprintf(`#!/bin/bash
set -e

# 停止守护程序
if [ -f /Library/LaunchDaemons/re.%s.server.plist ]; then
    launchctl unload /Library/LaunchDaemons/re.%s.server.plist
fi

if [ -f /var/jb/Library/LaunchDaemons/re.%s.server.plist ]; then
    launchctl unload /var/jb/Library/LaunchDaemons/re.%s.server.plist
fi

echo "Fridare %s 服务已停止"
`,
		info.MagicName, info.MagicName,
		info.MagicName, info.MagicName,
		info.MagicName,
	)

	return os.WriteFile(prermPath, []byte(prermContent), 0755)
}

// calculateInstalledSize 计算安装大小（KB）
func (dp *DebPackager) calculateInstalledSize(dir string) (int, error) {
	var totalSize int64

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.Contains(path, "DEBIAN") {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	// 转换为KB并向上取整
	sizeKB := int((totalSize + 1023) / 1024)
	return sizeKB, nil
}

// buildDebPackage 构建DEB包
func (dp *DebPackager) buildDebPackage(tempDir, outputPath string, info *PackageInfo) error {
	// 检查是否有dpkg-deb命令
	if _, err := exec.LookPath("dpkg-deb"); err == nil {
		return dp.buildWithDpkgDeb(tempDir, outputPath)
	}

	// 如果没有dpkg-deb，使用内置方法
	return dp.buildWithTar(tempDir, outputPath, info)
}

// buildWithDpkgDeb 使用dpkg-deb构建
func (dp *DebPackager) buildWithDpkgDeb(tempDir, outputPath string) error {
	cmd := exec.Command("dpkg-deb", "--build", tempDir, outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dpkg-deb 执行失败: %v, 输出: %s", err, string(output))
	}
	return nil
}

// buildWithTar 使用tar构建（内置方法）
func (dp *DebPackager) buildWithTar(tempDir, outputPath string, info *PackageInfo) error {
	// 创建data.tar.gz
	dataPath := filepath.Join(tempDir, "data.tar.gz")
	err := dp.createDataTar(tempDir, dataPath)
	if err != nil {
		return err
	}

	// 创建control.tar.gz
	controlPath := filepath.Join(tempDir, "control.tar.gz")
	err = dp.createControlTar(tempDir, controlPath)
	if err != nil {
		return err
	}

	// 创建debian-binary文件
	debianBinaryPath := filepath.Join(tempDir, "debian-binary")
	err = os.WriteFile(debianBinaryPath, []byte("2.0\n"), 0644)
	if err != nil {
		return err
	}

	// 创建最终的.deb文件
	return dp.createDebFile(tempDir, outputPath)
}

// createDataTar 创建data.tar.gz
func (dp *DebPackager) createDataTar(tempDir, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 遍历并添加非DEBIAN目录的所有文件
	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		// 跳过DEBIAN目录和临时文件
		if strings.HasPrefix(relPath, "DEBIAN") || strings.Contains(relPath, ".tar.gz") || relPath == "debian-binary" {
			return nil
		}

		if relPath == "." {
			return nil
		}

		// 创建tar头部
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = "./" + relPath

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// createControlTar 创建control.tar.gz
func (dp *DebPackager) createControlTar(tempDir, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	debianDir := filepath.Join(tempDir, "DEBIAN")

	// 遍历DEBIAN目录
	return filepath.Walk(debianDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(debianDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		// 创建tar头部
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = "./" + relPath

		err = tarWriter.WriteHeader(header)
		if err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// createDebFile 创建最终的.deb文件
func (dp *DebPackager) createDebFile(tempDir, outputPath string) error {
	debFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer debFile.Close()

	// 写入debian-binary
	debianBinaryPath := filepath.Join(tempDir, "debian-binary")
	err = dp.appendFile(debFile, debianBinaryPath)
	if err != nil {
		return err
	}

	// 写入control.tar.gz
	controlPath := filepath.Join(tempDir, "control.tar.gz")
	err = dp.appendFile(debFile, controlPath)
	if err != nil {
		return err
	}

	// 写入data.tar.gz
	dataPath := filepath.Join(tempDir, "data.tar.gz")
	return dp.appendFile(debFile, dataPath)
}

// appendFile 将文件内容追加到另一个文件
func (dp *DebPackager) appendFile(dst *os.File, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(dst, src)
	return err
}

// ValidatePackageInfo 验证包信息
func (dp *DebPackager) ValidatePackageInfo(info *PackageInfo) error {
	if info.Name == "" {
		return fmt.Errorf("包名不能为空")
	}
	if info.Version == "" {
		return fmt.Errorf("版本不能为空")
	}
	if info.Maintainer == "" {
		return fmt.Errorf("维护者不能为空")
	}
	if info.MagicName == "" {
		return fmt.Errorf("魔改名称不能为空")
	}
	if len(info.MagicName) != 5 {
		return fmt.Errorf("魔改名称必须为5个字符")
	}
	if info.Port <= 0 || info.Port > 65535 {
		return fmt.Errorf("端口号必须在1-65535之间")
	}
	return nil
}

// GetDefaultPackageInfo 获取默认包信息
func (dp *DebPackager) GetDefaultPackageInfo() *PackageInfo {
	return &PackageInfo{
		Name:         "com.fridare.server",
		Version:      "1.0.0",
		Architecture: "iphoneos-arm64",
		Maintainer:   "Fridare Team <suifei@gmail.com>",
		Description:  "Modified Frida Server for iOS",
		Depends:      "",
		Section:      "Development",
		Priority:     "optional",
		Homepage:     "https://github.com/suifei/fridare",
		Port:         27042,
		MagicName:    "frida",
	}
}
