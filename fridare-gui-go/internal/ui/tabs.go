package ui

import (
	"fridare-gui/internal/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ModifyTab 修改标签页
type ModifyTab struct {
	config       *config.Config
	updateStatus StatusUpdater
	content      *fyne.Container
}

// NewModifyTab 创建修改标签页
func NewModifyTab(cfg *config.Config, statusUpdater StatusUpdater) *ModifyTab {
	mt := &ModifyTab{
		config:       cfg,
		updateStatus: statusUpdater,
	}

	mt.setupUI()
	return mt
}

func (mt *ModifyTab) setupUI() {
	// 文件选择区域
	filePathEntry := widget.NewEntry()
	filePathEntry.SetPlaceHolder("选择要修改的 Frida 文件...")

	browseBtn := widget.NewButton("浏览", func() {
		mt.updateStatus("文件选择功能待实现")
	})

	fileSelectArea := container.NewBorder(
		nil, nil, nil, browseBtn, filePathEntry,
	)

	// 修改选项
	magicNameEntry := widget.NewEntry()
	magicNameEntry.SetPlaceHolder("frida")
	magicNameEntry.SetText("fridare")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("27042")

	optionsForm := widget.NewForm(
		widget.NewFormItem("魔改名称", magicNameEntry),
		widget.NewFormItem("默认端口", portEntry),
	)

	// 修改按钮
	patchBtn := widget.NewButton("开始魔改", func() {
		mt.updateStatus("二进制修改功能待实现")
	})
	patchBtn.Importance = widget.HighImportance

	// 进度显示
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	progressLabel := widget.NewLabel("")
	progressLabel.Hide()

	mt.content = container.NewVBox(
		widget.NewCard("二进制魔改器", "修改 Frida 二进制文件的特征字符串", container.NewVBox(
			container.NewVBox(
				widget.NewLabel("选择文件:"),
				fileSelectArea,
			),
			widget.NewSeparator(),
			optionsForm,
			widget.NewSeparator(),
			patchBtn,
			progressBar,
			progressLabel,
		)),
	)
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
