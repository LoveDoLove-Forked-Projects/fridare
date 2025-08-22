package ui

import (
	"fmt"
	"fridare-gui/internal/config"
	"fridare-gui/internal/core"
	"fridare-gui/internal/utils"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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
	config       *config.Config
	updateStatus StatusUpdater
	content      *fyne.Container
}

func NewPackageTab(cfg *config.Config, statusUpdater StatusUpdater) *PackageTab {
	pt := &PackageTab{
		config:       cfg,
		updateStatus: statusUpdater,
	}

	pt.setupUI()
	return pt
}

func (pt *PackageTab) setupUI() {
	// Frida 文件选择
	fridaFileEntry := widget.NewEntry()
	fridaFileEntry.SetPlaceHolder("选择魔改后的 frida-server...")

	browseFridaBtn := widget.NewButton("浏览", func() {
		pt.updateStatus("Frida 文件选择功能待实现")
	})

	fridaFileArea := container.NewBorder(
		nil, nil, nil, browseFridaBtn, fridaFileEntry,
	)

	// DEB 包信息
	packageNameEntry := widget.NewEntry()
	packageNameEntry.SetText("com.example.fridare")

	versionEntry := widget.NewEntry()
	versionEntry.SetText("1.0.0")

	authorEntry := widget.NewEntry()
	authorEntry.SetText("Unknown")

	packageForm := widget.NewForm(
		widget.NewFormItem("包名", packageNameEntry),
		widget.NewFormItem("版本", versionEntry),
		widget.NewFormItem("作者", authorEntry),
	)

	// 输出路径
	outputPathEntry := widget.NewEntry()
	outputPathEntry.SetPlaceHolder("DEB 包输出路径...")

	browseOutputBtn := widget.NewButton("浏览", func() {
		pt.updateStatus("输出路径选择功能待实现")
	})

	outputArea := container.NewBorder(
		nil, nil, nil, browseOutputBtn, outputPathEntry,
	)

	// 打包按钮
	packageBtn := widget.NewButton("生成 DEB 包", func() {
		pt.updateStatus("DEB 打包功能待实现")
	})
	packageBtn.Importance = widget.HighImportance

	// 进度显示
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	progressLabel := widget.NewLabel("")
	progressLabel.Hide()

	pt.content = container.NewVBox(
		widget.NewCard("iOS DEB 打包器", "将魔改的 Frida 文件打包为 Cydia DEB 安装包", container.NewVBox(
			container.NewVBox(
				widget.NewLabel("Frida 文件:"),
				fridaFileArea,
			),
			widget.NewSeparator(),
			packageForm,
			widget.NewSeparator(),
			container.NewVBox(
				widget.NewLabel("输出路径:"),
				outputArea,
			),
			widget.NewSeparator(),
			packageBtn,
			progressBar,
			progressLabel,
		)),
	)
}

func (pt *PackageTab) Content() *fyne.Container {
	return pt.content
}

func (pt *PackageTab) Refresh() {
	// 刷新逻辑
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
