package ui

import (
	"fmt"
	"fridare-gui/internal/config"
	"fridare-gui/internal/core"
	"fridare-gui/internal/utils"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

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

// ToolsTab 工具标签页
type ToolsTab struct {
	config       *config.Config
	updateStatus StatusUpdater
	content      *fyne.Container
}

func NewToolsTab(cfg *config.Config, statusUpdater StatusUpdater) *ToolsTab {
	tt := &ToolsTab{
		config:       cfg,
		updateStatus: statusUpdater,
	}

	tt.setupUI()
	return tt
}

func (tt *ToolsTab) setupUI() {
	// 环境检测区域
	checkStatusLabel := widget.NewLabel("点击检查按钮开始环境检测...")

	checkBtn := widget.NewButton("检查依赖", func() {
		tt.updateStatus("开始环境检测...")
		checkStatusLabel.SetText("正在检查 Python 环境...")
		// TODO: 实际的环境检测逻辑
	})

	environmentArea := container.NewVBox(
		widget.NewLabel("环境检测:"),
		container.NewHBox(checkBtn, checkStatusLabel),
	)

	// frida-tools 路径选择
	toolsPathEntry := widget.NewEntry()
	toolsPathEntry.SetPlaceHolder("选择 frida-tools 安装路径...")

	browseToolsBtn := widget.NewButton("浏览", func() {
		tt.updateStatus("工具路径选择功能待实现")
	})

	toolsPathArea := container.NewBorder(
		nil, nil, nil, browseToolsBtn, toolsPathEntry,
	)

	// 魔改选项
	magicNameEntry := widget.NewEntry()
	magicNameEntry.SetText("fridare")

	portEntry := widget.NewEntry()
	portEntry.SetText("27042")

	optionsForm := widget.NewForm(
		widget.NewFormItem("魔改名称", magicNameEntry),
		widget.NewFormItem("默认端口", portEntry),
	)

	// 魔改按钮
	patchToolsBtn := widget.NewButton("魔改 frida-tools", func() {
		tt.updateStatus("frida-tools 魔改功能待实现")
	})
	patchToolsBtn.Importance = widget.HighImportance

	// 还原按钮
	restoreBtn := widget.NewButton("还原原版", func() {
		tt.updateStatus("还原功能待实现")
	})

	buttonsArea := container.NewHBox(patchToolsBtn, restoreBtn)

	// 进度显示
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	progressLabel := widget.NewLabel("")
	progressLabel.Hide()

	tt.content = container.NewVBox(
		widget.NewCard("frida-tools 魔改器", "修改 Python frida-tools 库的默认配置", container.NewVBox(
			environmentArea,
			widget.NewSeparator(),
			container.NewVBox(
				widget.NewLabel("frida-tools 路径:"),
				toolsPathArea,
			),
			widget.NewSeparator(),
			optionsForm,
			widget.NewSeparator(),
			buttonsArea,
			progressBar,
			progressLabel,
		)),
	)
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

	// UI 组件
	fridaServerEntry   *widget.Entry
	fridaAgentEntry    *widget.Entry
	outputPathEntry    *widget.Entry
	magicNameEntry     *widget.Entry
	portEntry          *widget.Entry
	packageNameEntry   *widget.Entry
	versionEntry       *widget.Entry
	architectureSelect *widget.Select
	maintainerEntry    *widget.Entry
	descriptionEntry   *widget.Entry
	dependsEntry       *widget.Entry
	sectionEntry       *widget.Entry
	prioritySelect     *widget.Select
	homepageEntry      *widget.Entry
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
	// 文件选择部分
	ct.fridaServerEntry = widget.NewEntry()
	ct.fridaServerEntry.SetPlaceHolder("选择frida-server文件...")
	serverSelectBtn := widget.NewButton("选择frida-server", ct.selectFridaServer)

	ct.fridaAgentEntry = widget.NewEntry()
	ct.fridaAgentEntry.SetPlaceHolder("选择frida-agent.dylib文件 (可选)...")
	agentSelectBtn := widget.NewButton("选择frida-agent", ct.selectFridaAgent)

	ct.outputPathEntry = widget.NewEntry()
	ct.outputPathEntry.SetPlaceHolder("选择输出DEB文件路径...")
	outputSelectBtn := widget.NewButton("选择输出路径", ct.selectOutputPath)

	// 基本配置
	ct.magicNameEntry = widget.NewEntry()
	ct.magicNameEntry.SetPlaceHolder("5个字符的魔改名称")
	ct.magicNameEntry.Validator = func(text string) error {
		if len(text) != 5 {
			return fmt.Errorf("必须是5个字符")
		}
		if len(text) > 0 {
			first := text[0]
			if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z')) {
				return fmt.Errorf("必须以字母开头")
			}
		}
		for _, c := range text {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
				return fmt.Errorf("只能包含字母和数字")
			}
		}
		return nil
	}

	ct.portEntry = widget.NewEntry()
	ct.portEntry.SetText("27042")
	ct.portEntry.Validator = func(text string) error {
		if port, err := strconv.Atoi(text); err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("端口必须在1-65535范围内")
		}
		return nil
	}

	ct.isRootlessCheck = widget.NewCheck("Rootless结构", nil)

	// 包信息配置
	ct.packageNameEntry = widget.NewEntry()
	ct.packageNameEntry.SetPlaceHolder("包名 (自动生成)")

	ct.versionEntry = widget.NewEntry()
	ct.versionEntry.SetText("17.2.17")

	ct.architectureSelect = widget.NewSelect([]string{
		"iphoneos-arm64",
		"iphoneos-arm",
		"all",
	}, nil)
	ct.architectureSelect.SetSelected("iphoneos-arm64")

	ct.maintainerEntry = widget.NewEntry()
	ct.maintainerEntry.SetText("Fridare Team <support@fridare.com>")

	ct.descriptionEntry = widget.NewEntry()
	ct.descriptionEntry.SetPlaceHolder("包描述 (自动生成)")

	ct.dependsEntry = widget.NewEntry()
	ct.dependsEntry.SetText("firmware (>= 12.0)")

	ct.sectionEntry = widget.NewEntry()
	ct.sectionEntry.SetText("Development")

	ct.prioritySelect = widget.NewSelect([]string{
		"optional",
		"important",
		"required",
		"standard",
	}, nil)
	ct.prioritySelect.SetSelected("optional")

	ct.homepageEntry = widget.NewEntry()
	ct.homepageEntry.SetText("https://frida.re/")

	// 进度显示
	ct.progressBar = widget.NewProgressBar()
	ct.progressLabel = widget.NewLabel("准备就绪")

	// 创建按钮
	ct.createBtn = widget.NewButton("创建DEB包", ct.createDebPackage)
	ct.createBtn.Importance = widget.HighImportance

	// 布局
	fileSection := container.NewVBox(
		widget.NewCard("文件选择", "",
			container.NewVBox(
				container.NewHBox(ct.fridaServerEntry, serverSelectBtn),
				container.NewHBox(ct.fridaAgentEntry, agentSelectBtn),
				container.NewHBox(ct.outputPathEntry, outputSelectBtn),
			),
		),
	)

	basicSection := container.NewVBox(
		widget.NewCard("基本配置", "",
			container.NewVBox(
				container.NewGridWithColumns(2,
					widget.NewLabel("魔改名称:"), ct.magicNameEntry,
					widget.NewLabel("端口:"), ct.portEntry,
				),
				ct.isRootlessCheck,
			),
		),
	)

	packageSection := container.NewVBox(
		widget.NewCard("包信息", "",
			container.NewVBox(
				container.NewGridWithColumns(2,
					widget.NewLabel("包名:"), ct.packageNameEntry,
					widget.NewLabel("版本:"), ct.versionEntry,
					widget.NewLabel("架构:"), ct.architectureSelect,
					widget.NewLabel("维护者:"), ct.maintainerEntry,
				),
				container.NewVBox(
					widget.NewLabel("描述:"),
					ct.descriptionEntry,
					widget.NewLabel("依赖:"),
					ct.dependsEntry,
				),
				container.NewGridWithColumns(2,
					widget.NewLabel("分类:"), ct.sectionEntry,
					widget.NewLabel("优先级:"), ct.prioritySelect,
				),
				container.NewVBox(
					widget.NewLabel("主页:"),
					ct.homepageEntry,
				),
			),
		),
	)

	actionSection := container.NewVBox(
		widget.NewCard("操作", "",
			container.NewVBox(
				ct.progressLabel,
				ct.progressBar,
				ct.createBtn,
			),
		),
	)

	ct.content = container.NewVBox(
		fileSection,
		basicSection,
		packageSection,
		actionSection,
	)

	// 设置监听器
	ct.magicNameEntry.OnChanged = ct.updatePackageName
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
		return
	}

	packageName := fmt.Sprintf("re.frida.server.%s", magicName)
	if ct.isRootlessCheck.Checked {
		packageName += ".rootless"
	}

	ct.packageNameEntry.SetText(packageName)

	// 同时更新描述
	description := fmt.Sprintf("Dynamic instrumentation toolkit for developers, security researchers, and reverse engineers (Modified: %s)", magicName)
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
	dialog.ShowError(fmt.Errorf(message), ct.app.Driver().AllWindows()[0])
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
