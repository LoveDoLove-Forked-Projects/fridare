package ui

import (
	"encoding/json"
	"fmt"
	"fridare-gui/internal/config"
	"fridare-gui/internal/core"
	"fridare-gui/internal/utils"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// fixedWidthEntry 创建固定宽度的Entry
func fixedWidthEntry(width float32, placeholder string) *FixedWidthEntry {
	entry := NewFixedWidthEntry(width)
	entry.SetPlaceHolder(placeholder)
	entry.Resize(fyne.NewSize(width, entry.MinSize().Height))
	return entry
}

// ModifyTab 修改标签页
type ModifyTab struct {
	app          fyne.App
	config       *config.Config
	updateStatus StatusUpdater
	addLog       func(string) // 添加日志记录功能
	content      *fyne.Container

	// UI 组件
	filePathEntry  *widget.Entry
	magicNameEntry *widget.Entry
	fileInfoText   *widget.RichText
	progressBar    *widget.ProgressBar
	progressLabel  *widget.Label
	patchBtn       *widget.Button

	// 核心功能
	hexReplacer *core.HexReplacer
}

// NewModifyTab 创建修改标签页
func NewModifyTab(app fyne.App, cfg *config.Config, statusUpdater StatusUpdater, logFunc func(string)) *ModifyTab {
	mt := &ModifyTab{
		app:          app,
		config:       cfg,
		updateStatus: statusUpdater,
		addLog:       logFunc,
		hexReplacer:  core.NewHexReplacer(),
	}

	mt.setupUI()
	return mt
}

func (mt *ModifyTab) setupUI() {
	// 输入文件选择区域
	mt.filePathEntry = widget.NewEntry()
	mt.filePathEntry.SetPlaceHolder("选择要修改的 Frida 二进制文件...")
	mt.filePathEntry.OnChanged = func(path string) {
		if path != "" {
			mt.analyzeFile(path)
		} else {
			mt.fileInfoText.ParseMarkdown("")
		}
		// 立即验证输入
		mt.validateInput(mt.magicNameEntry.Text, mt.filePathEntry.Text)
	}

	browseInputBtn := widget.NewButton("浏览", func() {
		mt.selectInputFile()
	})

	fileSelectArea := container.NewBorder(
		nil, nil, nil, browseInputBtn, mt.filePathEntry,
	)

	// 魔改选项
	mt.magicNameEntry = widget.NewEntry()
	mt.magicNameEntry.SetPlaceHolder("输入5个小写字母")
	if mt.config.MagicName != "" && len(mt.config.MagicName) == 5 {
		mt.magicNameEntry.SetText(mt.config.MagicName)
	} else {
		mt.magicNameEntry.SetText("frida")
	}
	// 验证输入
	mt.magicNameEntry.OnChanged = func(text string) {
		mt.validateInput(text, mt.filePathEntry.Text)
	}

	// 随机生成按钮
	randomBtn := widget.NewButton("随机", func() {
		randomName := mt.generateRandomName()
		mt.magicNameEntry.SetText(randomName)
		mt.validateInput(randomName, mt.filePathEntry.Text)
	})

	magicNameArea := container.NewBorder(
		nil, nil, nil, randomBtn, mt.magicNameEntry,
	)

	optionsForm := container.NewVBox(
		widget.NewLabel("魔改名称 (必须5个小写字母):"),
		magicNameArea,
	)

	// 文件信息显示区域
	mt.fileInfoText = widget.NewRichText()
	mt.fileInfoText.Resize(fyne.NewSize(0, 200))

	fileInfoScroll := container.NewScroll(mt.fileInfoText)
	fileInfoScroll.SetMinSize(fyne.NewSize(0, 200))

	fileInfoCard := widget.NewCard("文件信息", "二进制文件格式和架构信息", fileInfoScroll)

	// 修改按钮
	mt.patchBtn = widget.NewButton("开始魔改", func() {
		mt.startPatching()
	})
	mt.patchBtn.Importance = widget.HighImportance
	mt.patchBtn.Disable() // 初始状态禁用

	// 进度显示
	mt.progressBar = widget.NewProgressBar()
	mt.progressBar.Hide()

	mt.progressLabel = widget.NewLabel("")
	mt.progressLabel.Hide()

	// 主布局
	mainContent := container.NewVBox(
		container.NewVBox(
			widget.NewLabel("输入文件:"),
			fileSelectArea,
		),
		widget.NewSeparator(),
		optionsForm,
		widget.NewSeparator(),
		mt.patchBtn,
		mt.progressBar,
		mt.progressLabel,
	)

	// 使用水平分割布局
	splitContainer := container.NewHSplit(
		widget.NewCard("二进制魔改器", "修改 Frida 二进制文件的特征字符串", mainContent),
		fileInfoCard,
	)
	splitContainer.Offset = 0.6 // 左侧占60%

	mt.content = container.NewPadded(splitContainer)

	// 初始验证状态
	mt.validateInput(mt.magicNameEntry.Text, mt.filePathEntry.Text)
}

// selectInputFile 选择输入文件
func (mt *ModifyTab) selectInputFile() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		path := reader.URI().Path()
		mt.filePathEntry.SetText(path)
		mt.validateInput(mt.magicNameEntry.Text, path)
	}, fyne.CurrentApp().Driver().AllWindows()[0])

	// 移除文件类型过滤，支持所有文件类型
	fileDialog.Show()
}

// analyzeFile 分析文件
func (mt *ModifyTab) analyzeFile(filePath string) {
	go func() {
		mt.updateStatus("正在分析文件...")

		description, err := mt.hexReplacer.DescribeFile(filePath)
		if err != nil {
			mt.fileInfoText.ParseMarkdown(fmt.Sprintf("**错误:** %s", err.Error()))
			mt.updateStatus("文件分析失败: " + err.Error())
			return
		}

		// 格式化显示信息
		markdown := fmt.Sprintf("**文件路径:** %s\n\n**文件信息:**\n```\n%s\n```", filePath, description)
		mt.fileInfoText.ParseMarkdown(markdown)
		fyne.Do(func() {
			mt.updateStatus("文件分析完成")
		})
	}()
}

// validateInput 验证输入
func (mt *ModifyTab) validateInput(name string, filePath string) {
	inputValid := name != ""
	nameValid := len(name) == 5 && utils.IsFridaNewName(name)
	filePathValid := utils.FileExists(filePath)

	if inputValid && nameValid && filePathValid {
		mt.patchBtn.Enable()
	} else {
		mt.patchBtn.Disable()
	}

}

// generateRandomName 生成随机名称
func (mt *ModifyTab) generateRandomName() string {
	return utils.GenerateRandomName()
}

// startPatching 开始修改
func (mt *ModifyTab) startPatching() {
	inputPath := mt.filePathEntry.Text
	magicName := mt.magicNameEntry.Text

	// 自动生成输出路径
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// 将文件名中的 frida 替换为魔改名称
	if strings.Contains(name, "frida") {
		name = strings.ReplaceAll(name, "frida", magicName)
	} else {
		name = magicName + "_" + name
	}

	outputPath := filepath.Join(dir, name+ext)

	// 显示进度
	mt.progressBar.Show()
	mt.progressLabel.Show()
	mt.progressBar.SetValue(0)
	mt.progressLabel.SetText("正在初始化...")
	mt.patchBtn.Disable()

	go func() {
		defer func() {
			mt.progressBar.Hide()
			mt.progressLabel.Hide()
			mt.patchBtn.Enable()
		}()

		mt.updateStatus("开始魔改二进制文件...")
		mt.addLog("INFO: 开始魔改二进制文件")
		mt.addLog(fmt.Sprintf("INFO: 输入文件: %s", inputPath))
		mt.addLog(fmt.Sprintf("INFO: 输出文件: %s", outputPath))
		mt.addLog(fmt.Sprintf("INFO: 魔改名称: %s", magicName))

		// 进度回调函数
		progressCallback := func(progress float64, message string) {
			mt.progressBar.SetValue(progress)
			mt.progressLabel.SetText(message)
			mt.updateStatus(message)
			mt.addLog(fmt.Sprintf("INFO: %s (%.1f%%)", message, progress*100))
		}

		// 执行修改
		err := mt.hexReplacer.PatchFile(inputPath, magicName, outputPath, progressCallback)
		if err != nil {
			errorMsg := "魔改失败: " + err.Error()
			mt.updateStatus(errorMsg)
			mt.progressLabel.SetText("魔改失败!")
			mt.addLog("ERROR: " + errorMsg)

			// 只显示最终错误结果的弹窗
			dialog.ShowError(fmt.Errorf("魔改失败: %v", err), fyne.CurrentApp().Driver().AllWindows()[0])
			return
		}

		mt.progressBar.SetValue(1.0)
		mt.progressLabel.SetText("魔改完成!")
		successMsg := fmt.Sprintf("魔改完成! 输出文件: %s", outputPath)
		mt.updateStatus(successMsg)
		mt.addLog("SUCCESS: " + successMsg)

		// 更新配置
		mt.config.MagicName = magicName
		mt.config.Save()
		mt.addLog("INFO: 配置已保存")

		// 只显示最终成功结果的弹窗
		// 缩短路径显示以避免宽度问题
		inputBaseName := filepath.Base(inputPath)
		outputBaseName := filepath.Base(outputPath)

		contentText := fmt.Sprintf("魔改完成!\n\n输入文件: %s\n输出文件: %s\n魔改名称: %s\n\n文件已保存到与输入文件相同的目录",
			inputBaseName, outputBaseName, magicName)

		// 使用简单的信息弹窗，内容会自动换行
		dialog.ShowInformation("魔改完成", contentText, fyne.CurrentApp().Driver().AllWindows()[0])
	}()
}

func (mt *ModifyTab) Content() *fyne.Container {
	return mt.content
}

func (mt *ModifyTab) Refresh() {
	// 刷新逻辑
}

// PackageTab 打包标签页
type PackageTab struct {
	app          fyne.App
	config       *config.Config
	updateStatus StatusUpdater
	addLog       func(string)
	content      *fyne.Container

	// UI 组件
	debFileEntry    *widget.Entry // DEB文件选择
	outputPathEntry *widget.Entry
	portEntry       *widget.Entry
	magicNameEntry  *widget.Entry
	packageBtn      *widget.Button
	progressBar     *widget.ProgressBar
	progressLabel   *widget.Label

	// 核心功能
	debPackager *core.DebPackager
}

func NewPackageTab(app fyne.App, cfg *config.Config, statusUpdater StatusUpdater, logFunc func(string)) *PackageTab {
	pt := &PackageTab{
		app:          app,
		config:       cfg,
		updateStatus: statusUpdater,
		addLog:       logFunc,
		debPackager:  core.NewDebPackager(),
	}

	pt.setupUI()
	return pt
}

func (pt *PackageTab) setupUI() {
	// 直接设置为修改现有DEB包模式，不需要模式选择
	pt.debFileEntry = widget.NewEntry()
	pt.debFileEntry.SetPlaceHolder("选择要修改的 DEB 包文件...")
	pt.debFileEntry.OnChanged = func(path string) {
		pt.validateInput()
	}

	// 设置DEB文件选择区域
	browseDebBtn := widget.NewButton("浏览", func() {
		pt.selectDebFile()
	})

	debFileArea := container.NewBorder(
		nil, nil, widget.NewLabel("DEB文件:"), browseDebBtn, pt.debFileEntry,
	)

	// 包信息显示区域
	infoText := widget.NewRichText()
	infoText.ParseMarkdown("**DEB包修改器**\n\n" +
		"• 选择现有的Frida DEB包文件\n" +
		"• 自动读取包元数据\n" +
		"• 使用指定的魔改名称和端口进行修改\n" +
		"• 生成修改后的DEB包\n\n" +
		"**支持的修改：**\n" +
		"• 修改Frida服务名称\n" +
		"• 修改默认监听端口\n" +
		"• 保持原包的所有其他设置")

	packageInfoCard := widget.NewCard("操作说明", "", infoText)

	// 输出路径选择
	pt.outputPathEntry = widget.NewEntry()
	pt.outputPathEntry.SetPlaceHolder("DEB 包输出路径...")
	pt.outputPathEntry.OnChanged = func(path string) {
		pt.validateInput()
	}

	browseOutputBtn := widget.NewButton("浏览", func() {
		pt.selectOutputPath()
	})

	outputArea := container.NewBorder(
		nil, nil, nil, browseOutputBtn, pt.outputPathEntry,
	)

	// 魔改配置
	pt.portEntry = widget.NewEntry()
	if pt.config.DefaultPort != 0 {
		pt.portEntry.SetText(fmt.Sprintf("%d", pt.config.DefaultPort))
	} else {
		pt.portEntry.SetText("27042")
	}
	pt.portEntry.SetPlaceHolder("Frida 服务器端口")

	pt.magicNameEntry = widget.NewEntry()
	if pt.config.MagicName != "" && len(pt.config.MagicName) == 5 {
		pt.magicNameEntry.SetText(pt.config.MagicName)
	} else {
		pt.magicNameEntry.SetText("frida")
	}
	pt.magicNameEntry.SetPlaceHolder("魔改名称 (5个字符)")

	// 验证输入
	pt.magicNameEntry.OnChanged = func(text string) {
		pt.validateInput()
	}
	pt.portEntry.OnChanged = func(text string) {
		pt.validateInput()
	}

	// 随机生成魔改名称按钮
	randomMagicBtn := widget.NewButton("随机", func() {
		randomName := utils.GenerateRandomName()
		pt.magicNameEntry.SetText(randomName)
	})

	magicNameArea := container.NewBorder(
		nil, nil, nil, randomMagicBtn, pt.magicNameEntry,
	)

	// 进度条和状态
	pt.progressBar = widget.NewProgressBar()
	pt.progressBar.Hide()
	pt.progressLabel = widget.NewLabel("")
	pt.progressLabel.Hide()

	// 操作按钮
	pt.packageBtn = widget.NewButton("修改 DEB 包", func() {
		pt.startPackaging()
	})
	pt.packageBtn.Disable()

	progressArea := container.NewVBox(
		pt.progressBar,
		pt.progressLabel,
	)

	actionArea := container.NewBorder(
		nil, nil, nil, pt.packageBtn, progressArea,
	)

	// 主布局
	pt.content = container.NewVBox(
		debFileArea,
		widget.NewSeparator(),
		container.NewBorder(
			nil, nil, widget.NewLabel("输出路径:"), nil, outputArea,
		),
		widget.NewSeparator(),
		container.NewBorder(
			nil, nil, widget.NewLabel("魔改名称:"), nil, magicNameArea,
		),
		container.NewBorder(
			nil, nil, widget.NewLabel("服务端口:"), nil, pt.portEntry,
		),
		widget.NewSeparator(),
		packageInfoCard,
		widget.NewSeparator(),
		actionArea,
	)
}

func (pt *PackageTab) Content() *fyne.Container {
	return pt.content
}

func (pt *PackageTab) Refresh() {
	// 刷新逻辑
	pt.validateInput()
}

// selectDebFile 选择DEB文件
func (pt *PackageTab) selectDebFile() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		path := reader.URI().Path()
		pt.debFileEntry.SetText(path)

		// 自动设置输出路径
		if pt.outputPathEntry.Text == "" {
			dir := filepath.Dir(path)
			base := filepath.Base(path)
			nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))

			// 生成修改后的文件名
			magicName := pt.magicNameEntry.Text
			if magicName == "" {
				magicName = "frida"
			}

			outputName := fmt.Sprintf("%s_%s_modified.deb", nameWithoutExt, magicName)
			outputPath := filepath.Join(dir, outputName)
			pt.outputPathEntry.SetText(outputPath)
		}
	}, fyne.CurrentApp().Driver().AllWindows()[0])

	// 设置文件过滤器
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".deb"}))
	fileDialog.Show()
}

// selectOutputPath 选择输出路径
func (pt *PackageTab) selectOutputPath() {
	fileDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()

		filePath := writer.URI().Path()
		if !strings.HasSuffix(strings.ToLower(filePath), ".deb") {
			filePath += ".deb"
		}
		pt.outputPathEntry.SetText(filePath)
	}, fyne.CurrentApp().Driver().AllWindows()[0])

	// 设置默认文件名
	magicName := pt.magicNameEntry.Text
	if magicName == "" {
		magicName = "frida"
	}

	defaultName := fmt.Sprintf("%s_modified.deb", magicName)
	fileDialog.SetFileName(defaultName)

	fileDialog.Show()
}

// validateInput 验证输入
func (pt *PackageTab) validateInput() {
	outputPathValid := pt.outputPathEntry.Text != ""
	magicNameValid := len(pt.magicNameEntry.Text) == 5 && utils.IsFridaNewName(pt.magicNameEntry.Text)
	portValid := pt.isValidPort(pt.portEntry.Text)
	fileValid := pt.debFileEntry.Text != ""

	if fileValid && outputPathValid && magicNameValid && portValid {
		pt.packageBtn.Enable()
	} else {
		pt.packageBtn.Disable()
	}
}

// isValidPort 检查端口是否有效
func (pt *PackageTab) isValidPort(portStr string) bool {
	if portStr == "" {
		return false
	}
	port, err := strconv.Atoi(portStr)
	return err == nil && port > 0 && port <= 65535
}

// startPackaging 开始修改DEB包
func (pt *PackageTab) startPackaging() {
	outputPath := pt.outputPathEntry.Text
	debFile := pt.debFileEntry.Text

	// 解析端口
	port, err := strconv.Atoi(pt.portEntry.Text)
	if err != nil {
		pt.updateStatus("端口号无效")
		pt.addLog("ERROR: 端口号无效: " + pt.portEntry.Text)
		return
	}

	magicName := pt.magicNameEntry.Text

	// 显示进度
	pt.progressBar.Show()
	pt.progressLabel.Show()
	pt.progressBar.SetValue(0)
	pt.progressLabel.SetText("正在初始化...")
	pt.packageBtn.Disable()

	go func() {
		defer func() {
			fyne.Do(pt.progressBar.Hide)
			fyne.Do(pt.progressLabel.Hide)
			fyne.Do(pt.packageBtn.Enable)
		}()

		pt.modifyExistingDebPackage(outputPath, port, magicName, debFile)
	}()
}

// modifyExistingDebPackage 修改现有DEB包
func (pt *PackageTab) modifyExistingDebPackage(outputPath string, port int, magicName string, debFile string) {

	pt.updateStatus("开始修改DEB包...")
	pt.addLog("INFO: 开始修改现有DEB包")
	pt.addLog(fmt.Sprintf("INFO: 输入DEB文件: %s", debFile))
	pt.addLog(fmt.Sprintf("INFO: 输出路径: %s", outputPath))
	pt.addLog(fmt.Sprintf("INFO: 魔改名称: %s", magicName))
	pt.addLog(fmt.Sprintf("INFO: 端口: %d", port))

	// 创建DEB修改器
	debModifier := core.NewDebModifier(debFile, outputPath, magicName, port)

	// 进度回调函数
	progressCallback := func(progress float64, message string) {
		fyne.Do(func() {
			pt.progressBar.SetValue(progress)
			pt.progressLabel.SetText(message)
			pt.updateStatus(message)
			pt.addLog(fmt.Sprintf("INFO: %s (%.1f%%)", message, progress*100))
		})
	}

	// 执行修改
	err := debModifier.ModifyDebPackage(progressCallback)
	if err != nil {
		errorMsg := "DEB包修改失败: " + err.Error()
		pt.updateStatus(errorMsg)
		pt.progressLabel.SetText("修改失败!")
		pt.addLog("ERROR: " + errorMsg)

		// 显示错误弹窗
		dialog.ShowError(fmt.Errorf("DEB包修改失败: %v", err), fyne.CurrentApp().Driver().AllWindows()[0])
		return
	}

	pt.progressBar.SetValue(1.0)
	pt.progressLabel.SetText("DEB包修改完成!")
	successMsg := fmt.Sprintf("DEB包修改完成! 输出文件: %s", outputPath)
	pt.updateStatus(successMsg)
	pt.addLog("SUCCESS: " + successMsg)

	// 更新配置
	pt.config.MagicName = magicName
	pt.config.DefaultPort = port
	pt.config.Save()
	pt.addLog("INFO: 配置已保存")

	// 显示成功弹窗
	outputBaseName := filepath.Base(outputPath)
	inputBaseName := filepath.Base(debFile)
	contentText := fmt.Sprintf("DEB包修改完成!\n\n原始文件: %s\n输出文件: %s\n魔改名称: %s\n端口: %d\n\n修改后的文件已保存到指定位置",
		inputBaseName, outputBaseName, magicName, port)

	dialog.ShowInformation("DEB包修改完成", contentText, fyne.CurrentApp().Driver().AllWindows()[0])
}

// PythonEnv Python环境信息
type PythonEnv struct {
	Name    string // 环境名称 (conda env name 或 "System Python")
	Path    string // Python可执行文件路径
	Version string // Python版本
	Type    string // 环境类型 (conda, venv, system)
}

// FridaInfo Frida工具信息
type FridaInfo struct {
	Version     string // frida版本
	InstallPath string // 安装路径
	PatchStatus string // 补丁状态 (original, patched, unknown)
	BackupPath  string // 备份路径
}

// ToolsTab 工具标签页
type ToolsTab struct {
	app          fyne.App
	config       *config.Config
	updateStatus StatusUpdater
	addLog       func(string)
	content      *fyne.Container

	// UI组件
	pythonEnvSelect   *widget.Select
	refreshEnvBtn     *widget.Button
	envInfoLabel      *widget.Label
	fridaInfoLabel    *widget.Label
	fridaVersionLabel *widget.Label
	fridaPathLabel    *widget.Label
	patchStatusLabel  *widget.Label

	magicNameEntry *FixedWidthEntry
	portEntry      *FixedWidthEntry

	patchBtn   *widget.Button
	restoreBtn *widget.Button
	backupBtn  *widget.Button

	progressBar   *widget.ProgressBar
	progressLabel *widget.Label

	// 数据
	pythonEnvs  []PythonEnv
	currentEnv  *PythonEnv
	fridaInfo   *FridaInfo
	hexReplacer *core.HexReplacer
}

func NewToolsTab(cfg *config.Config, statusUpdater StatusUpdater) *ToolsTab {
	tt := &ToolsTab{
		config:       cfg,
		updateStatus: statusUpdater,
		addLog:       func(msg string) {}, // 默认空实现
		pythonEnvs:   []PythonEnv{},
		hexReplacer:  core.NewHexReplacer(),
	}

	tt.setupUI()
	return tt
}

// SetLogFunction 设置日志函数
func (tt *ToolsTab) SetLogFunction(addLog func(string)) {
	tt.addLog = addLog
}

func (tt *ToolsTab) setupUI() {
	// Python环境选择区域
	tt.pythonEnvSelect = widget.NewSelect([]string{"点击刷新扫描Python环境..."}, func(selected string) {
		tt.onPythonEnvSelected(selected)
	})
	tt.pythonEnvSelect.Resize(fyne.NewSize(300, 0))

	tt.refreshEnvBtn = widget.NewButton("刷新环境", func() {
		tt.scanPythonEnvironments()
	})

	tt.envInfoLabel = widget.NewLabel("未选择Python环境")
	tt.envInfoLabel.Wrapping = fyne.TextWrapWord

	environmentArea := widget.NewCard("Python环境", "", container.NewVBox(
		container.NewHBox(
			widget.NewLabel("选择Python环境:"),
			tt.pythonEnvSelect,
			tt.refreshEnvBtn,
		),
		tt.envInfoLabel,
	))

	// frida-tools信息区域
	tt.fridaInfoLabel = widget.NewLabel("请先选择Python环境")
	tt.fridaVersionLabel = widget.NewLabel("版本: 未知")
	tt.fridaPathLabel = widget.NewLabel("路径: 未知")
	tt.patchStatusLabel = widget.NewLabel("状态: 未检测")

	detectBtn := widget.NewButton("检测frida-tools", func() {
		tt.detectFridaTools()
	})

	fridaInfoArea := widget.NewCard("frida-tools信息", "", container.NewVBox(
		container.NewHBox(detectBtn, tt.fridaInfoLabel),
		tt.fridaVersionLabel,
		tt.fridaPathLabel,
		tt.patchStatusLabel,
	))

	// 魔改配置区域
	tt.magicNameEntry = fixedWidthEntry(180, "魔改名称")
	tt.magicNameEntry.SetText("fridare")

	// 魔改名称验证器
	tt.magicNameEntry.Validator = func(text string) error {
		if len(text) == 0 {
			return fmt.Errorf("魔改名称不能为空")
		}
		if len(text) > 10 {
			return fmt.Errorf("魔改名称不能超过10个字符")
		}
		// 检查字符合法性
		for i, c := range text {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
				return fmt.Errorf("第%d个字符'%c'无效，只能包含字母、数字、下划线和横线", i+1, c)
			}
		}
		return nil
	}

	tt.portEntry = fixedWidthEntry(160, "端口")
	tt.portEntry.SetText("27042")
	tt.portEntry.Validator = func(text string) error {
		if port, err := strconv.Atoi(text); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("端口必须在1-65535范围内")
		}
		return nil
	}

	// 操作按钮区域
	tt.patchBtn = widget.NewButton("执行魔改", func() {
		tt.patchFridaTools()
	})
	tt.patchBtn.Importance = widget.HighImportance
	tt.patchBtn.Disable()

	tt.restoreBtn = widget.NewButton("恢复原版", func() {
		tt.restoreFridaTools()
	})
	tt.restoreBtn.Disable()

	tt.backupBtn = widget.NewButton("手动备份", func() {
		tt.backupFridaTools()
	})
	tt.backupBtn.Disable()

	configArea := widget.NewCard("魔改配置", "", container.NewVBox(
		container.NewHBox(
			widget.NewLabel("魔改名称:"), tt.magicNameEntry,
			widget.NewLabel("默认端口:"), tt.portEntry,
			tt.patchBtn,
			tt.restoreBtn,
			tt.backupBtn),
	))
	// 进度显示
	tt.progressBar = widget.NewProgressBar()
	tt.progressBar.Hide()

	tt.progressLabel = widget.NewLabel("")
	tt.progressLabel.Hide()

	progressArea := container.NewVBox(
		tt.progressBar,
		tt.progressLabel,
	)

	// 主布局
	tt.content = container.NewVBox(
		container.NewGridWithColumns(2,
			environmentArea,
			fridaInfoArea),
		configArea,
		progressArea,
	)

	// 初始扫描Python环境
	go tt.scanPythonEnvironments()
}

// scanPythonEnvironments 扫描Python环境
func (tt *ToolsTab) scanPythonEnvironments() {
	tt.updateStatus("正在扫描Python环境...")
	tt.addLog("INFO: 开始扫描Python环境")

	var envs []PythonEnv

	// 扫描conda环境
	condaEnvs := tt.scanCondaEnvironments()
	envs = append(envs, condaEnvs...)

	// 扫描系统Python
	systemPython := tt.scanSystemPython()
	if systemPython != nil {
		envs = append(envs, *systemPython)
	}

	// 扫描venv环境 (可选，先不实现)

	tt.pythonEnvs = envs

	// 更新UI
	fyne.Do(func() {
		if len(envs) == 0 {
			tt.pythonEnvSelect.Options = []string{"未找到Python环境"}
			tt.envInfoLabel.SetText("未找到可用的Python环境")
			tt.updateStatus("未找到Python环境")
		} else {
			options := make([]string, len(envs))
			for i, env := range envs {
				options[i] = fmt.Sprintf("%s (%s)", env.Name, env.Type)
			}
			tt.pythonEnvSelect.Options = options
			tt.pythonEnvSelect.Refresh()
			tt.updateStatus(fmt.Sprintf("找到 %d 个Python环境", len(envs)))
			tt.addLog(fmt.Sprintf("INFO: 找到 %d 个Python环境", len(envs)))
		}
	})
}

// scanCondaEnvironments 扫描conda环境
func (tt *ToolsTab) scanCondaEnvironments() []PythonEnv {
	var envs []PythonEnv

	// 执行conda env list命令
	cmd := exec.Command("conda", "env", "list", "--json")
	hideConsoleCmd(cmd)
	output, err := cmd.Output()
	if err != nil {
		tt.addLog("INFO: 未找到conda环境")
		return envs
	}

	// 解析JSON输出
	var condaInfo struct {
		Envs []string `json:"envs"`
	}

	if err := json.Unmarshal(output, &condaInfo); err != nil {
		tt.addLog("ERROR: 解析conda环境信息失败: " + err.Error())
		return envs
	}

	// 获取每个环境的详细信息
	for _, envPath := range condaInfo.Envs {
		pythonPath := filepath.Join(envPath, "python")
		if runtime.GOOS == "windows" {
			pythonPath = filepath.Join(envPath, "python.exe")
		}

		// 检查python可执行文件是否存在
		if _, err := os.Stat(pythonPath); os.IsNotExist(err) {
			continue
		}

		// 获取环境名称
		envName := filepath.Base(envPath)
		if envName == "." {
			envName = "base"
		}

		// 获取Python版本
		versionCmd := exec.Command(pythonPath, "--version")
		hideConsoleCmd(versionCmd)
		versionOutput, err := versionCmd.Output()
		version := "未知"
		if err == nil {
			version = strings.TrimSpace(string(versionOutput))
		}

		env := PythonEnv{
			Name:    envName,
			Path:    pythonPath,
			Version: version,
			Type:    "conda",
		}
		envs = append(envs, env)
		tt.addLog(fmt.Sprintf("INFO: 找到conda环境: %s (%s)", envName, version))
	}

	return envs
}

// scanSystemPython 扫描系统Python
func (tt *ToolsTab) scanSystemPython() *PythonEnv {
	// 尝试找到系统Python
	pythonCmds := []string{"python", "python3"}

	for _, cmd := range pythonCmds {
		pythonPath, err := exec.LookPath(cmd)
		if err != nil {
			continue
		}

		// 获取Python版本
		versionCmd := exec.Command(pythonPath, "--version")
		hideConsoleCmd(versionCmd)
		versionOutput, err := versionCmd.Output()
		version := "未知"
		if err == nil {
			version = strings.TrimSpace(string(versionOutput))
		}

		env := &PythonEnv{
			Name:    "System Python",
			Path:    pythonPath,
			Version: version,
			Type:    "system",
		}

		tt.addLog(fmt.Sprintf("INFO: 找到系统Python: %s", version))
		return env
	}

	return nil
}

// onPythonEnvSelected Python环境选择回调
func (tt *ToolsTab) onPythonEnvSelected(selected string) {
	if selected == "" || selected == "未找到Python环境" {
		return
	}

	// 从选择的字符串中找到对应的环境
	for _, env := range tt.pythonEnvs {
		expectedText := fmt.Sprintf("%s (%s)", env.Name, env.Type)
		if expectedText == selected {
			tt.currentEnv = &env

			// 更新环境信息显示
			tt.envInfoLabel.SetText(fmt.Sprintf("环境: %s\n版本: %s\n路径: %s",
				env.Name, env.Version, env.Path))

			tt.updateStatus(fmt.Sprintf("已选择Python环境: %s", env.Name))
			tt.addLog(fmt.Sprintf("INFO: 切换到Python环境: %s", env.Name))

			// 启用检测按钮
			// 自动检测frida-tools
			go tt.detectFridaTools()
			break
		}
	}
}

// detectFridaTools 检测frida-tools信息
func (tt *ToolsTab) detectFridaTools() {
	if tt.currentEnv == nil {
		tt.updateStatus("请先选择Python环境")
		return
	}

	fyne.Do(func() {
		tt.fridaInfoLabel.SetText("未安装frida-tools")
		tt.fridaVersionLabel.SetText("版本: 未安装")
		tt.fridaPathLabel.SetText("路径: 无")
		tt.patchStatusLabel.SetText("状态: 未安装")
		tt.patchBtn.Disable()
		tt.restoreBtn.Disable()
		tt.backupBtn.Disable()
	})
	tt.updateStatus("正在检测frida-tools...")
	tt.addLog("INFO: 开始检测frida-tools")

	// 使用选定的Python环境执行pip show frida
	var cmd *exec.Cmd
	if tt.currentEnv.Type == "conda" {
		// conda环境需要激活
		envName := tt.currentEnv.Name
		if envName == "base" {
			cmd = exec.Command("conda", "run", "-n", "base", "pip", "show", "frida")
		} else {
			cmd = exec.Command("conda", "run", "-n", envName, "pip", "show", "frida")
		}
	} else {
		// 系统Python直接使用pip
		cmd = exec.Command(tt.currentEnv.Path, "-m", "pip", "show", "frida")
	}
	hideConsoleCmd(cmd)

	output, err := cmd.Output()
	if err != nil {
		tt.updateStatus("未检测到frida-tools")
		tt.addLog("ERROR: 未检测到frida-tools: " + err.Error())
		return
	}

	// 解析pip show输出
	lines := strings.Split(string(output), "\n")
	var version, location string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Location:") {
			location = strings.TrimSpace(strings.TrimPrefix(line, "Location:"))
		}
	}

	if version == "" || location == "" {
		tt.updateStatus("无法获取frida-tools信息")
		tt.addLog("ERROR: 无法解析frida-tools信息")
		return
	}

	// 检测patch状态
	fridaPath := filepath.Join(location, "frida")
	patchStatus := tt.checkPatchStatus(fridaPath)
	backupPath := filepath.Join(fridaPath, "_original_backup")

	tt.fridaInfo = &FridaInfo{
		Version:     version,
		InstallPath: fridaPath,
		PatchStatus: patchStatus,
		BackupPath:  backupPath,
	}

	// 更新UI
	fyne.Do(func() {
		tt.fridaInfoLabel.SetText("已检测到frida-tools")
		tt.fridaVersionLabel.SetText("版本: " + version)
		tt.fridaPathLabel.SetText("路径: " + fridaPath)
		tt.patchStatusLabel.SetText("状态: " + patchStatus)

		// 根据状态启用/禁用按钮
		if patchStatus == "original" {
			tt.patchBtn.Enable()
			tt.restoreBtn.Disable()
			tt.backupBtn.Enable()
		} else if patchStatus == "patched" {
			tt.patchBtn.Disable()
			tt.restoreBtn.Enable()
			tt.backupBtn.Disable()
		} else {
			tt.patchBtn.Enable()
			tt.restoreBtn.Disable()
			tt.backupBtn.Enable()
		}
	})

	tt.updateStatus("frida-tools检测完成")
	tt.addLog(fmt.Sprintf("INFO: frida-tools v%s 检测完成，状态: %s", version, patchStatus))
}

// checkPatchStatus 检查patch状态
func (tt *ToolsTab) checkPatchStatus(fridaPath string) string {
	// 检查备份目录是否存在
	backupPath := filepath.Join(fridaPath, "_original_backup")
	if _, err := os.Stat(backupPath); err == nil {
		return "patched"
	}

	// 检查关键文件是否包含frida字符串 (简单检测)
	coreFile := filepath.Join(fridaPath, "_frida.py")
	if _, err := os.Stat(coreFile); err == nil {
		content, err := os.ReadFile(coreFile)
		if err == nil {
			contentStr := string(content)
			// 如果包含默认的frida字符串，认为是原版
			if strings.Contains(contentStr, "frida-server") && !strings.Contains(contentStr, "fridare") {
				return "original"
			} else if strings.Contains(contentStr, "fridare") {
				return "patched"
			}
		}
	}

	return "unknown"
}

// patchFridaTools 执行frida-tools魔改
func (tt *ToolsTab) patchFridaTools() {
	if tt.currentEnv == nil || tt.fridaInfo == nil {
		tt.updateStatus("请先选择Python环境并检测frida-tools")
		return
	}

	magicName := strings.TrimSpace(tt.magicNameEntry.Text)
	port := strings.TrimSpace(tt.portEntry.Text)

	// 验证输入
	if err := tt.magicNameEntry.Validator(magicName); err != nil {
		tt.updateStatus("魔改名称错误: " + err.Error())
		return
	}
	if err := tt.portEntry.Validator(port); err != nil {
		tt.updateStatus("端口错误: " + err.Error())
		return
	}

	tt.updateStatus("开始执行frida-tools魔改...")
	tt.addLog(fmt.Sprintf("INFO: 开始魔改frida-tools，魔改名称: %s, 端口: %s", magicName, port))

	// 显示进度
	fyne.Do(func() {
		tt.progressBar.Show()
		tt.progressLabel.Show()
		tt.progressLabel.SetText("正在创建备份...")
		tt.progressBar.SetValue(0.1)
		tt.patchBtn.Disable()
	})

	go func() {
		// 1. 创建备份
		if err := tt.createBackup(); err != nil {
			fyne.Do(func() {
				tt.progressBar.Hide()
				tt.progressLabel.Hide()
				tt.patchBtn.Enable()
			})
			tt.updateStatus("创建备份失败: " + err.Error())
			tt.addLog("ERROR: 创建备份失败: " + err.Error())
			return
		}

		fyne.Do(func() {
			tt.progressLabel.SetText("正在执行魔改...")
			tt.progressBar.SetValue(0.5)
		})

		// 2. 执行魔改
		if err := tt.performPatch(magicName, port); err != nil {
			fyne.Do(func() {
				tt.progressBar.Hide()
				tt.progressLabel.Hide()
				tt.patchBtn.Enable()
			})
			tt.updateStatus("魔改失败: " + err.Error())
			tt.addLog("ERROR: 魔改失败: " + err.Error())
			return
		}

		fyne.Do(func() {
			tt.progressLabel.SetText("魔改完成!")
			tt.progressBar.SetValue(1.0)

			// 延迟隐藏进度条
			time.AfterFunc(2*time.Second, func() {
				fyne.Do(func() {
					tt.progressBar.Hide()
					tt.progressLabel.Hide()
				})
			})

			// 更新按钮状态
			tt.patchBtn.Disable()
			tt.restoreBtn.Enable()
			tt.backupBtn.Disable()

			// 更新状态显示
			tt.patchStatusLabel.SetText("状态: patched")
		})

		tt.updateStatus("frida-tools魔改完成!")
		tt.addLog("SUCCESS: frida-tools魔改完成")
	}()
}

// createBackup 创建备份
func (tt *ToolsTab) createBackup() error {
	backupPath := tt.fridaInfo.BackupPath

	// 如果备份已存在，跳过
	if _, err := os.Stat(backupPath); err == nil {
		tt.addLog("INFO: 备份已存在，跳过创建备份")
		return nil
	}

	// 创建备份目录
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}

	// 复制关键文件
	filesToBackup := []string{
		"_frida.py",
		"core.py",
		"__init__.py",
	}

	for _, file := range filesToBackup {
		srcPath := filepath.Join(tt.fridaInfo.InstallPath, file)
		dstPath := filepath.Join(backupPath, file)

		if _, err := os.Stat(srcPath); err == nil {
			if err := tt.copyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("备份文件 %s 失败: %v", file, err)
			}
			tt.addLog(fmt.Sprintf("INFO: 已备份文件: %s", file))
		}
	}

	tt.addLog("INFO: 备份创建完成")
	return nil
}

// copyFile 复制文件
func (tt *ToolsTab) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// performPatch 执行魔改
func (tt *ToolsTab) performPatch(magicName, port string) error {
	// 1. Python代码字符串魔改
	if err := tt.patchPythonFiles(magicName, port); err != nil {
		return fmt.Errorf("Python文件魔改失败: %v", err)
	}

	// 2. SO文件二进制魔改
	if err := tt.patchSOFiles(magicName, port); err != nil {
		return fmt.Errorf("SO文件魔改失败: %v", err)
	}

	return nil
}

// patchPythonFiles 魔改Python文件
func (tt *ToolsTab) patchPythonFiles(magicName, port string) error {
	// 定义要魔改的文件和替换规则
	patchRules := map[string]map[string]string{
		"_frida.py": {
			"frida-server": magicName + "-server",
			"frida-agent":  magicName + "-agent",
			"27042":        port,
			"frida":        magicName,
		},
		"core.py": {
			"frida-server": magicName + "-server",
			"frida-agent":  magicName + "-agent",
			"frida":        magicName,
			"27042":        port,
		},
		"__init__.py": {
			"frida": magicName,
		},
	}

	for file, rules := range patchRules {
		filePath := filepath.Join(tt.fridaInfo.InstallPath, file)

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			tt.addLog(fmt.Sprintf("WARN: Python文件不存在，跳过: %s", file))
			continue
		}

		// 读取文件内容
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("读取文件 %s 失败: %v", file, err)
		}

		contentStr := string(content)
		originalContent := contentStr

		// 应用替换规则
		for oldStr, newStr := range rules {
			if strings.Contains(contentStr, oldStr) {
				contentStr = strings.ReplaceAll(contentStr, oldStr, newStr)
				tt.addLog(fmt.Sprintf("INFO: 替换 '%s' -> '%s' 在文件 %s", oldStr, newStr, file))
			}
		}

		// 如果内容有变化，写回文件
		if contentStr != originalContent {
			if err := os.WriteFile(filePath, []byte(contentStr), 0644); err != nil {
				return fmt.Errorf("写入文件 %s 失败: %v", file, err)
			}
			tt.addLog(fmt.Sprintf("SUCCESS: 已魔改Python文件: %s", file))
		} else {
			tt.addLog(fmt.Sprintf("INFO: Python文件无需修改: %s", file))
		}
	}

	return nil
}

// patchSOFiles 魔改SO文件
func (tt *ToolsTab) patchSOFiles(magicName, port string) error {
	// 查找SO文件
	soFiles, err := tt.findSOFiles()
	if err != nil {
		return fmt.Errorf("查找SO文件失败: %v", err)
	}

	if len(soFiles) == 0 {
		tt.addLog("INFO: 未找到SO文件，跳过二进制魔改")
		return nil
	}

	// 使用hexreplace工具进行二进制替换
	for _, soFile := range soFiles {
		if err := tt.patchSingleSOFile(soFile, magicName, port); err != nil {
			tt.addLog(fmt.Sprintf("WARN: SO文件魔改失败: %s, 错误: %v", soFile, err))
			continue
		}
		tt.addLog(fmt.Sprintf("SUCCESS: 已魔改SO文件: %s", soFile))
	}

	return nil
}

// findSOFiles 查找SO文件
func (tt *ToolsTab) findSOFiles() ([]string, error) {
	var soFiles []string

	// 遍历frida安装目录
	err := filepath.Walk(tt.fridaInfo.InstallPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续遍历
		}

		// 查找.so、.dll、.dylib文件
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".so" || ext == ".dll" || ext == ".dylib" || ext == ".pyd" {
			soFiles = append(soFiles, path)
		}

		return nil
	})

	return soFiles, err
}

// patchSingleSOFile 魔改单个SO文件
func (tt *ToolsTab) patchSingleSOFile(soFile, magicName, port string) error {
	// 使用HexReplacer进行专业的二进制魔改
	// HexReplacer会自动处理所有frida相关的字符串替换
	if err := tt.hexReplace(soFile, "", magicName); err != nil {
		return fmt.Errorf("魔改SO文件失败: %v", err)
	}

	tt.addLog(fmt.Sprintf("SUCCESS: 已魔改SO文件: %s", soFile))
	return nil
} // hexReplace 执行十六进制替换 - 使用HexReplacer进行专业的二进制魔改
func (tt *ToolsTab) hexReplace(filePath, oldStr, newStr string) error {
	// 检查新字符串长度（魔改名称必须是5个字符）
	if len(newStr) != 5 {
		return fmt.Errorf("魔改名称必须是5个字符，当前为: %s (%d字符)", newStr, len(newStr))
	}

	// 创建临时输出文件
	tempFile := filePath + ".tmp"

	// 使用HexReplacer进行专业的二进制魔改
	err := tt.hexReplacer.PatchFile(filePath, newStr, tempFile, func(progress float64, status string) {
		// 可以在这里添加进度回调，但对于SO文件魔改我们简化处理
		tt.addLog(fmt.Sprintf("INFO: %s (%.1f%%)", status, progress*100))
	})

	if err != nil {
		// 清理临时文件
		os.Remove(tempFile)
		return fmt.Errorf("HexReplacer魔改失败: %v", err)
	}

	// 替换原文件
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("替换原文件失败: %v", err)
	}

	return nil
}

// restoreFridaTools 恢复原版frida-tools
func (tt *ToolsTab) restoreFridaTools() {
	if tt.fridaInfo == nil {
		tt.updateStatus("请先检测frida-tools")
		return
	}

	tt.updateStatus("开始恢复原版frida-tools...")
	tt.addLog("INFO: 开始恢复原版frida-tools")

	// 显示进度
	fyne.Do(func() {
		tt.progressBar.Show()
		tt.progressLabel.Show()
		tt.progressLabel.SetText("正在恢复原版...")
		tt.progressBar.SetValue(0.2)
		tt.restoreBtn.Disable()
	})

	go func() {
		if err := tt.performRestore(); err != nil {
			fyne.Do(func() {
				tt.progressBar.Hide()
				tt.progressLabel.Hide()
				tt.restoreBtn.Enable()
			})
			tt.updateStatus("恢复失败: " + err.Error())
			tt.addLog("ERROR: 恢复失败: " + err.Error())
			return
		}

		fyne.Do(func() {
			tt.progressLabel.SetText("恢复完成!")
			tt.progressBar.SetValue(1.0)

			// 延迟隐藏进度条
			time.AfterFunc(2*time.Second, func() {
				fyne.Do(func() {
					tt.progressBar.Hide()
					tt.progressLabel.Hide()
				})
			})

			// 更新按钮状态
			tt.patchBtn.Enable()
			tt.restoreBtn.Disable()
			tt.backupBtn.Enable()

			// 更新状态显示
			tt.patchStatusLabel.SetText("状态: original")
		})

		tt.updateStatus("frida-tools恢复完成!")
		tt.addLog("SUCCESS: frida-tools恢复完成")
	}()
}

// performRestore 执行恢复
func (tt *ToolsTab) performRestore() error {
	backupPath := tt.fridaInfo.BackupPath

	// 检查备份是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("备份不存在: %s", backupPath)
	}

	// 恢复备份的文件
	files, err := os.ReadDir(backupPath)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %v", err)
	}

	fyne.Do(func() {
		tt.progressBar.SetValue(0.5)
	})

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		srcPath := filepath.Join(backupPath, file.Name())
		dstPath := filepath.Join(tt.fridaInfo.InstallPath, file.Name())

		if err := tt.copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("恢复文件 %s 失败: %v", file.Name(), err)
		}
		tt.addLog(fmt.Sprintf("INFO: 已恢复文件: %s", file.Name()))
	}

	// 删除备份目录
	if err := os.RemoveAll(backupPath); err != nil {
		tt.addLog(fmt.Sprintf("WARN: 删除备份目录失败: %v", err))
	} else {
		tt.addLog("INFO: 已删除备份目录")
	}

	return nil
}

// backupFridaTools 手动备份
func (tt *ToolsTab) backupFridaTools() {
	if tt.fridaInfo == nil {
		tt.updateStatus("请先检测frida-tools")
		return
	}

	tt.updateStatus("开始手动备份...")
	tt.addLog("INFO: 开始手动备份")

	go func() {
		if err := tt.createBackup(); err != nil {
			tt.updateStatus("手动备份失败: " + err.Error())
			tt.addLog("ERROR: 手动备份失败: " + err.Error())
		} else {
			tt.updateStatus("手动备份完成")
			tt.addLog("SUCCESS: 手动备份完成")
		}
	}()
}

func (tt *ToolsTab) Content() *fyne.Container {
	return tt.content
}

func (tt *ToolsTab) Refresh() {
	// 刷新逻辑
}

// SettingsTab 设置标签页
type SettingsTab struct {
	config       *config.Config
	updateStatus StatusUpdater
	applyTheme   func()
	content      *fyne.Container
}

func NewSettingsTab(cfg *config.Config, statusUpdater StatusUpdater, themeApplier func()) *SettingsTab {
	st := &SettingsTab{
		config:       cfg,
		updateStatus: statusUpdater,
		applyTheme:   themeApplier,
	}

	st.setupUI()
	return st
}

func (st *SettingsTab) setupUI() {
	st.content = container.NewVBox(
		widget.NewCard("应用设置", "", container.NewVBox(
			widget.NewLabel("配置应用程序设置..."),
			widget.NewButton("保存设置", func() {
				st.updateStatus("设置保存功能待实现")
			}),
		)),
	)
}

func (st *SettingsTab) Content() *fyne.Container {
	return st.content
}

func (st *SettingsTab) Refresh() {
	// 刷新逻辑
}

// CreateTab 创建DEB包标签页
type CreateTab struct {
	app          fyne.App
	config       *config.Config
	updateStatus StatusUpdater
	addLog       func(string)
	content      *fyne.Container

	// UI 组件 - 使用widget.Entry改善所有输入框宽度
	fridaServerEntry   *FixedWidthEntry
	fridaAgentEntry    *FixedWidthEntry
	outputPathEntry    *FixedWidthEntry
	magicNameEntry     *FixedWidthEntry
	portEntry          *FixedWidthEntry
	packageNameEntry   *FixedWidthEntry
	versionEntry       *FixedWidthEntry
	architectureSelect *widget.Select
	maintainerEntry    *FixedWidthEntry
	descriptionEntry   *FixedWidthEntry
	dependsEntry       *FixedWidthEntry
	sectionEntry       *FixedWidthEntry
	prioritySelect     *widget.Select
	homepageEntry      *FixedWidthEntry
	isRootlessCheck    *widget.Check
	progressBar        *widget.ProgressBar
	progressLabel      *widget.Label
	createBtn          *widget.Button

	// 核心功能
	creator *core.CreateFridaDeb
}

// NewCreateTab 创建新的创建标签页
func NewCreateTab(app fyne.App, cfg *config.Config, statusUpdater StatusUpdater, logFunc func(string)) *CreateTab {
	ct := &CreateTab{
		app:          app,
		config:       cfg,
		updateStatus: statusUpdater,
		addLog:       logFunc,
	}

	ct.setupUI()
	return ct
}

// setupUI 设置UI界面
func (ct *CreateTab) setupUI() {
	// 使用固定宽度Entry组件 - 增加宽度
	ct.fridaServerEntry = fixedWidthEntry(200, "选择frida-server文件...")
	ct.fridaAgentEntry = fixedWidthEntry(200, "选择frida-agent.dylib文件 (可选)...")
	ct.outputPathEntry = fixedWidthEntry(180, "选择输出DEB文件路径...")

	ct.magicNameEntry = fixedWidthEntry(100, "5字符")
	ct.portEntry = fixedWidthEntry(100, "端口")
	ct.packageNameEntry = fixedWidthEntry(300, "包名 (自动生成)")
	ct.versionEntry = fixedWidthEntry(200, "版本")
	ct.maintainerEntry = fixedWidthEntry(300, "维护者")
	ct.descriptionEntry = fixedWidthEntry(300, "包描述")
	ct.dependsEntry = fixedWidthEntry(200, "依赖")
	ct.sectionEntry = fixedWidthEntry(200, "分类")
	ct.homepageEntry = fixedWidthEntry(300, "主页")

	// 设置按钮
	serverSelectBtn := widget.NewButton("选择", ct.selectFridaServer)
	agentSelectBtn := widget.NewButton("选择", ct.selectFridaAgent)
	outputSelectBtn := widget.NewButton("选择", ct.selectOutputPath)

	// 基本配置验证器和事件
	ct.magicNameEntry.Validator = func(text string) error {
		if len(text) != 5 {
			return fmt.Errorf("魔改名称必须是5个字符")
		}
		if len(text) == 0 {
			return fmt.Errorf("魔改名称不能为空")
		}

		// 检查首字符必须是字母
		first := text[0]
		if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z')) {
			return fmt.Errorf("魔改名称必须以字母开头")
		}

		// 检查所有字符必须是字母或数字
		for i, c := range text {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
				return fmt.Errorf("第%d个字符'%c'无效，只能包含字母和数字", i+1, c)
			}
		}

		// 检查是否为保留名称
		lowerText := strings.ToLower(text)
		reservedNames := []string{"frida", "admin", "root", "user", "guest"}
		for _, reserved := range reservedNames {
			if lowerText == reserved {
				return fmt.Errorf("'%s'是保留名称，请使用其他名称", text)
			}
		}

		return nil
	}

	// 添加实时验证和字符长度限制
	ct.magicNameEntry.OnChanged = func(text string) {
		// 限制输入长度为5个字符
		if len(text) > 5 {
			ct.magicNameEntry.SetText(text[:5])
			return
		}

		// 实时更新包名
		ct.updatePackageName(text)

		// 实时验证显示
		if err := ct.magicNameEntry.Validator(text); err != nil {
			ct.updateStatus(fmt.Sprintf("魔改名称错误: %v", err))
		} else if len(text) == 5 {
			ct.updateStatus("魔改名称验证通过")
		}
	}

	ct.portEntry.SetText("27042")
	ct.portEntry.Validator = func(text string) error {
		if port, err := strconv.Atoi(text); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("端口必须在1-65535范围内")
		}
		return nil
	}

	ct.isRootlessCheck = widget.NewCheck("Rootless结构", nil)

	// 包信息配置
	ct.packageNameEntry.Disable() // 设置为只读

	ct.versionEntry.SetText("17.2.17")

	ct.architectureSelect = widget.NewSelect([]string{
		"iphoneos-arm64",
		"iphoneos-arm",
		"all",
	}, nil)
	ct.architectureSelect.SetSelected("iphoneos-arm64")
	ct.architectureSelect.Resize(fyne.NewSize(200, 0))

	ct.maintainerEntry.SetText("Fridare Team <support@fridare.com>")
	ct.descriptionEntry.SetPlaceHolder("包描述 (自动生成)")
	ct.dependsEntry.SetText("firmware (>= 12.0)")
	ct.sectionEntry.SetText("Development")

	ct.prioritySelect = widget.NewSelect([]string{
		"optional",
		"important",
		"required",
		"standard",
	}, nil)
	ct.prioritySelect.SetSelected("optional")
	ct.prioritySelect.Resize(fyne.NewSize(200, 0))

	ct.homepageEntry.SetText("https://frida.re/")

	// 进度显示
	ct.progressBar = widget.NewProgressBar()
	ct.progressLabel = widget.NewLabel("准备就绪")

	// 创建按钮
	ct.createBtn = widget.NewButton("创建DEB包", ct.createDebPackage)
	ct.createBtn.Importance = widget.HighImportance

	// 简化的紧凑布局 - 使用Border布局避免重叠
	// 文件选择区域
	serverRow := container.NewBorder(nil, nil,
		widget.NewLabel("frida-server:"),
		serverSelectBtn,
		ct.fridaServerEntry)

	agentRow := container.NewBorder(nil, nil,
		widget.NewLabel(" frida-agent:"),
		agentSelectBtn,
		ct.fridaAgentEntry)

	outputRow := container.NewBorder(nil, nil,
		widget.NewLabel("     输出路径:"),
		outputSelectBtn,
		ct.outputPathEntry)

	fileSection := widget.NewCard("文件选择", "", container.NewVBox(
		serverRow,
		agentRow,
		outputRow,
	))

	// 基本配置区域 - 使用HBox横向排列
	configSection := widget.NewCard("基本配置", "", container.NewHBox(
		widget.NewLabel("魔改名称:"), ct.magicNameEntry,
		widget.NewLabel("　　端口:"), ct.portEntry,
		widget.NewLabel("　　　　"), ct.isRootlessCheck,
	))

	// 包信息区域 - 分两行显示
	packageRow1 := container.NewHBox(
		widget.NewLabel("　　包名:"), ct.packageNameEntry,
		widget.NewLabel("　　版本:"), ct.versionEntry,
		widget.NewLabel("　　架构:"), ct.architectureSelect,
	)

	packageRow2 := container.NewHBox(
		widget.NewLabel("　维护者:"), ct.maintainerEntry,
		widget.NewLabel("　　分类:"), ct.sectionEntry,
		widget.NewLabel("　优先级:"), ct.prioritySelect,
	)

	packageSection := widget.NewCard("　包信息", "", container.NewVBox(
		packageRow1,
		packageRow2,
	))

	// 详细信息区域 - 一行显示
	detailRow := container.NewHBox(
		widget.NewLabel("　　描述:"), ct.descriptionEntry,
		widget.NewLabel("　　依赖:"), ct.dependsEntry,
		widget.NewLabel("　　主页:"), ct.homepageEntry,
	)

	detailSection := widget.NewCard("详细信息", "", detailRow)

	// 操作区域 - 使用Border布局
	actionSection := container.NewBorder(nil, nil,
		container.NewHBox(ct.progressLabel, ct.progressBar),
		ct.createBtn,
		nil,
	)

	// 主布局
	ct.content = container.NewVBox(
		fileSection,
		configSection,
		packageSection,
		detailSection,
		actionSection,
	)

	// 设置监听器 - 魔改名称的OnChanged已在Entry定义时设置
	ct.isRootlessCheck.OnChanged = func(checked bool) {
		ct.updatePackageName(ct.magicNameEntry.Text)
	}
}

// selectFridaServer 选择frida-server文件
func (ct *CreateTab) selectFridaServer() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		filePath := reader.URI().Path()
		ct.fridaServerEntry.SetText(filePath)
		ct.addLog(fmt.Sprintf("选择frida-server文件: %s", filePath))
	}, ct.app.Driver().AllWindows()[0])
}

// selectFridaAgent 选择frida-agent文件
func (ct *CreateTab) selectFridaAgent() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		filePath := reader.URI().Path()
		ct.fridaAgentEntry.SetText(filePath)
		ct.addLog(fmt.Sprintf("选择frida-agent文件: %s", filePath))
	}, ct.app.Driver().AllWindows()[0])
}

// selectOutputPath 选择输出路径
func (ct *CreateTab) selectOutputPath() {
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()

		filePath := writer.URI().Path()
		if !strings.HasSuffix(strings.ToLower(filePath), ".deb") {
			filePath += ".deb"
		}
		ct.outputPathEntry.SetText(filePath)
		ct.addLog(fmt.Sprintf("选择输出路径: %s", filePath))
	}, ct.app.Driver().AllWindows()[0])
}

// updatePackageName 更新包名
func (ct *CreateTab) updatePackageName(magicName string) {
	if magicName == "" {
		ct.packageNameEntry.SetText("")
		ct.descriptionEntry.SetText("")
		return
	}

	// 生成规则：将魔改字符替换其中的frida字符
	basePackageName := "re.frida.server"

	// 将frida替换为魔改名称
	packageName := strings.ReplaceAll(basePackageName, "frida", magicName)

	// 添加rootless后缀（如果选中）
	if ct.isRootlessCheck.Checked {
		packageName += ".rootless"
	}

	ct.packageNameEntry.SetText(packageName)

	// 同时更新描述，也替换其中的frida
	baseDescription := "Dynamic instrumentation toolkit for developers, security researchers, and reverse engineers based on Frida"
	description := strings.ReplaceAll(baseDescription, "Frida", strings.Title(magicName))
	description += fmt.Sprintf(" (Modified: %s)", magicName)
	ct.descriptionEntry.SetText(description)
}

// createDebPackage 创建DEB包
func (ct *CreateTab) createDebPackage() {
	// 验证输入
	if ct.fridaServerEntry.Text == "" {
		ct.showError("请选择frida-server文件")
		return
	}

	if ct.outputPathEntry.Text == "" {
		ct.showError("请选择输出路径")
		return
	}

	if ct.magicNameEntry.Text == "" {
		ct.showError("请输入魔改名称")
		return
	}

	if err := ct.magicNameEntry.Validator(ct.magicNameEntry.Text); err != nil {
		ct.showError(fmt.Sprintf("魔改名称格式错误: %v", err))
		return
	}

	port, err := strconv.Atoi(ct.portEntry.Text)
	if err != nil {
		ct.showError("端口格式错误")
		return
	}

	// 禁用按钮
	ct.createBtn.Disable()
	ct.progressBar.SetValue(0)
	ct.progressLabel.SetText("开始创建...")

	// 异步执行
	go ct.performCreate(port)
}

// performCreate 执行创建过程
func (ct *CreateTab) performCreate(port int) {
	defer func() {
		ct.createBtn.Enable()
	}()

	// 创建包信息
	packageInfo := &core.PackageInfo{
		Name:         ct.packageNameEntry.Text,
		Version:      ct.versionEntry.Text,
		Architecture: ct.architectureSelect.Selected,
		Maintainer:   ct.maintainerEntry.Text,
		Description:  ct.descriptionEntry.Text,
		Depends:      ct.dependsEntry.Text,
		Section:      ct.sectionEntry.Text,
		Priority:     ct.prioritySelect.Selected,
		Homepage:     ct.homepageEntry.Text,
		Port:         port,
		MagicName:    ct.magicNameEntry.Text,
		IsRootless:   ct.isRootlessCheck.Checked,
	}

	// 创建DEB构建器
	creator := core.NewCreateFridaDeb(ct.fridaServerEntry.Text, ct.outputPathEntry.Text, packageInfo)
	if ct.fridaAgentEntry.Text != "" {
		creator.FridaAgentPath = ct.fridaAgentEntry.Text
	}

	ct.addLog("开始创建DEB包...")
	ct.addLog(fmt.Sprintf("魔改名称: %s, 端口: %d, 结构: %s",
		packageInfo.MagicName, packageInfo.Port,
		map[bool]string{true: "Rootless", false: "Root"}[packageInfo.IsRootless]))

	// 执行创建
	err := creator.CreateDebPackage()
	if err != nil {
		ct.progressLabel.SetText("创建失败")
		ct.showError(fmt.Sprintf("创建DEB包失败: %v", err))
		ct.addLog(fmt.Sprintf("错误: %v", err))
		return
	}

	ct.progressBar.SetValue(1.0)
	ct.progressLabel.SetText("创建完成")
	ct.addLog("DEB包创建成功!")

	// 显示成功信息
	ct.showSuccess("DEB包创建成功!", fmt.Sprintf("输出文件: %s", ct.outputPathEntry.Text))

	ct.updateStatus("DEB包创建完成")
}

// showError 显示错误信息
func (ct *CreateTab) showError(message string) {
	dialog.ShowError(fmt.Errorf("%s", message), ct.app.Driver().AllWindows()[0])
}

// showSuccess 显示成功信息
func (ct *CreateTab) showSuccess(title, message string) {
	dialog.ShowInformation(title, message, ct.app.Driver().AllWindows()[0])
}

// Content 返回标签页内容
func (ct *CreateTab) Content() *fyne.Container {
	return ct.content
}

// Refresh 刷新标签页
func (ct *CreateTab) Refresh() {
	// 刷新逻辑
}

func hideConsoleCmd(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		}
	}
}
