package ui

import (
	"fmt"
	"fridare-gui/internal/assets"
	"fridare-gui/internal/config"
	"fridare-gui/internal/utils"
	"log"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	WindowMinWidth  = 1200
	WindowMinHeight = 800
)

// MainWindow 主窗口结构
type MainWindow struct {
	app    fyne.App
	window fyne.Window
	config *config.Config

	// UI 组件
	content      *fyne.Container
	tabContainer *container.AppTabs
	statusBar    *widget.Label
	logText      *widget.Entry

	// 全局配置控件
	proxyEntry *widget.Entry
	nameEntry  *widget.Entry
	portEntry  *widget.Entry

	// 功能模块
	downloadTab *DownloadTab
	modifyTab   *ModifyTab
	packageTab  *PackageTab
	createTab   *CreateTab // 新增创建标签页
	toolsTab    *ToolsTab
	settingsTab *SettingsTab
}

// NewMainWindow 创建主窗口
func NewMainWindow(app fyne.App, cfg *config.Config) *MainWindow {
	// 创建窗口
	window := app.NewWindow("Fridare GUI - Frida 魔改工具")
	window.SetMaster()
	window.SetIcon(assets.AppIcon) // 设置窗口图标

	// 设置窗口大小
	window.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
	window.SetFixedSize(false)

	// 设置最小尺寸
	window.SetContent(widget.NewLabel("Loading..."))

	// 设置窗口最小尺寸
	if cfg.WindowWidth < WindowMinWidth {
		cfg.WindowWidth = WindowMinWidth
	}
	if cfg.WindowHeight < WindowMinHeight {
		cfg.WindowHeight = WindowMinHeight
	}

	mw := &MainWindow{
		app:    app,
		window: window,
		config: cfg,
	}

	// 初始化UI
	mw.setupUI()

	// 应用主题
	mw.applyTheme()

	return mw
}

// setupUI 设置UI
func (mw *MainWindow) setupUI() {
	// 创建左侧边栏 - 全局配置
	leftSidebar := mw.createLeftSidebar()

	// 创建功能标签页
	mw.tabContainer = container.NewAppTabs()

	// 创建各个功能模块
	mw.downloadTab = NewDownloadTab(mw.app, mw.config, mw.updateStatus)
	mw.modifyTab = NewModifyTab(mw.app, mw.config, mw.updateStatus, mw.addLog)
	mw.packageTab = NewPackageTab(mw.app, mw.config, mw.updateStatus, mw.addLog)
	mw.createTab = NewCreateTab(mw.app, mw.config, mw.updateStatus, mw.addLog) // 新增创建标签页
	mw.toolsTab = NewToolsTab(mw.config, mw.updateStatus)
	mw.settingsTab = NewSettingsTab(mw.config, mw.updateStatus, mw.applyTheme)

	// 添加标签页（与原型保持一致）
	mw.tabContainer.Append(container.NewTabItem("📥 下载", mw.downloadTab.Content()))
	mw.tabContainer.Append(container.NewTabItem("🔧 魔改", mw.modifyTab.Content()))
	mw.tabContainer.Append(container.NewTabItem("📦 iOS魔改+打包", mw.packageTab.Content()))
	mw.tabContainer.Append(container.NewTabItem("🆕 创建DEB包", mw.createTab.Content())) // 新增创建标签页
	mw.tabContainer.Append(container.NewTabItem("🛠️ frida-tools 魔改", mw.toolsTab.Content()))

	// 创建底部状态区域（包含日志和按钮）
	bottomArea := mw.createBottomArea()

	// 创建顶部工具栏
	toolbar := mw.createToolbar()

	// 创建主内容区域（右侧的标签页区域）
	mainContentArea := container.NewBorder(
		nil,             // top
		bottomArea,      // bottom
		nil,             // left
		nil,             // right
		mw.tabContainer, // center
	)

	// 创建主布局 - 使用水平分割支持拖动调整大小
	splitContainer := container.NewHSplit(leftSidebar, mainContentArea)
	splitContainer.Offset = 0.22 // 设置左侧占比22%

	// 设置分割容器的最小尺寸
	if scroll, ok := leftSidebar.(*container.Scroll); ok {
		scroll.SetMinSize(fyne.NewSize(200, 0))
	}

	mw.content = container.NewBorder(
		toolbar,        // top
		nil,            // bottom
		nil,            // left
		nil,            // right
		splitContainer, // center
	)

	// 设置窗口内容
	mw.window.SetContent(mw.content)
}

// createToolbar 创建工具栏
func (mw *MainWindow) createToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			log.Println("新建操作")
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			log.Println("打开文件夹操作")
		}),
		widget.NewToolbarAction(theme.DocumentSaveIcon(), func() {
			mw.saveConfig()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			mw.refreshContent()
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.InfoIcon(), func() {
			mw.showAbout()
		}),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			mw.tabContainer.SelectTabIndex(4) // 选择设置标签页
		}),
	)

	return toolbar
}

// updateStatus 更新状态栏
func (mw *MainWindow) updateStatus(message string) {
	if mw.statusBar != nil {
		fyne.Do(func() {
			mw.statusBar.SetText(message)
		})
	}
	// 记录日志但不立即更新UI
	log.Println("STATUS:", message)
}

// saveConfig 保存配置
func (mw *MainWindow) saveConfig() {
	if err := mw.config.Save(); err != nil {
		mw.updateStatus("保存配置失败: " + err.Error())
		log.Printf("保存配置失败: %v", err)
	} else {
		mw.updateStatus("配置已保存")
	}
}

// refreshContent 刷新内容
func (mw *MainWindow) refreshContent() {
	mw.updateStatus("刷新中...")

	// 刷新各个标签页的内容
	if mw.downloadTab != nil {
		mw.downloadTab.Refresh()
	}
	if mw.modifyTab != nil {
		mw.modifyTab.Refresh()
	}
	if mw.packageTab != nil {
		mw.packageTab.Refresh()
	}
	if mw.toolsTab != nil {
		mw.toolsTab.Refresh()
	}

	mw.updateStatus("刷新完成")
}

// applyTheme 应用主题
func (mw *MainWindow) applyTheme() {
	switch mw.config.Theme {
	case "dark":
		mw.app.Settings().SetTheme(theme.DarkTheme())
	case "light":
		mw.app.Settings().SetTheme(theme.LightTheme())
	default:
		// auto - 使用系统默认
		mw.app.Settings().SetTheme(theme.DefaultTheme())
	}
}

// showAbout 显示关于对话框
func (mw *MainWindow) showAbout() {
	// 创建简单的对话框内容
	content := widget.NewLabel(`Fridare GUI v1.0.0

Frida 重打包和修补工具的图形界面版本

特性: 下载发行版, 二进制修补, DEB包生成, 工具集成

作者: suifei@gmail.com
项目: https://github.com/suifei/fridare`)

	content.Alignment = fyne.TextAlignCenter
	content.Wrapping = fyne.TextWrapWord

	// 创建对话框
	dialog := dialog.NewCustom("关于 Fridare GUI", "确定", content, mw.window)
	dialog.Resize(fyne.NewSize(400, 250))
	dialog.Show()
}

// ShowAndRun 显示窗口并运行应用
func (mw *MainWindow) ShowAndRun() {
	// 设置关闭回调
	mw.window.SetCloseIntercept(func() {
		mw.saveConfig()
		mw.app.Quit()
	})

	// 显示窗口
	mw.window.Show()

	// 运行应用
	mw.app.Run()
}

// StatusUpdater 状态更新接口
type StatusUpdater func(message string)

// createLeftSidebar 创建左侧边栏
func (mw *MainWindow) createLeftSidebar() fyne.CanvasObject {
	// 全局配置标题
	configTitle := widget.NewCard("全局配置", "应用程序全局设置", nil)

	// 网络代理配置
	mw.proxyEntry = widget.NewEntry()
	mw.proxyEntry.SetPlaceHolder("http://proxy:port")
	if mw.config.Proxy != "" {
		mw.proxyEntry.SetText(mw.config.Proxy)
	}

	// 代理测试按钮
	proxyTestBtn := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		mw.testProxy()
	})
	proxyTestBtn.Resize(fyne.NewSize(32, 32))
	proxyTestBtn.SetText("")

	// 代理输入框和测试按钮的容器
	proxyContainer := container.NewBorder(nil, nil, nil, proxyTestBtn, mw.proxyEntry)

	proxyForm := container.NewVBox(
		widget.NewRichTextFromMarkdown("**网络代理:**"),
		proxyContainer,
		widget.NewSeparator(),
	)

	// 魔改名称配置
	mw.nameEntry = widget.NewEntry()
	mw.nameEntry.SetPlaceHolder("fridare")
	if mw.config.MagicName != "" {
		mw.nameEntry.SetText(mw.config.MagicName)
	}

	// 随机名称生成按钮
	nameGenBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		mw.generateRandomName()
	})
	nameGenBtn.Resize(fyne.NewSize(32, 32))
	nameGenBtn.SetText("")

	// 名称输入框和生成按钮的容器
	nameContainer := container.NewBorder(nil, nil, nil, nameGenBtn, mw.nameEntry)

	nameForm := container.NewVBox(
		widget.NewRichTextFromMarkdown("**魔改名称:**"),
		nameContainer,
		widget.NewSeparator(),
	)

	// 端口配置
	mw.portEntry = widget.NewEntry()
	mw.portEntry.SetPlaceHolder("27042")
	if mw.config.DefaultPort != 0 {
		mw.portEntry.SetText(fmt.Sprintf("%d", mw.config.DefaultPort))
	}

	// 保存配置按钮
	saveButton := widget.NewButton("保存配置", func() {
		mw.saveGlobalConfig()
	})
	saveButton.Importance = widget.HighImportance

	portForm := container.NewVBox(
		widget.NewRichTextFromMarkdown("**端口号:**"),
		mw.portEntry,
		widget.NewSeparator(),
		saveButton,
	)

	// 组装左侧边栏内容
	sidebarContent := container.NewVBox(
		configTitle,
		widget.NewSeparator(),
		proxyForm,
		nameForm,
		portForm,
	)

	// 使用 Padded 容器添加内边距
	paddedContent := container.NewPadded(sidebarContent)

	// 添加滚动支持
	scrollSidebar := container.NewScroll(paddedContent)
	scrollSidebar.SetMinSize(fyne.NewSize(200, 0)) // 增加最小宽度

	return scrollSidebar
}

// createBottomArea 创建底部区域
func (mw *MainWindow) createBottomArea() *fyne.Container {
	// 创建状态栏
	mw.statusBar = widget.NewLabel("等待操作...")
	mw.statusBar.TextStyle = fyne.TextStyle{Italic: true}

	// 创建日志区域
	mw.logText = widget.NewMultiLineEntry()
	mw.logText.SetPlaceHolder("执行日志将显示在这里...")
	mw.logText.Disable()                    // 只读
	mw.logText.Resize(fyne.NewSize(0, 150)) // 设置高度

	// 创建日志控制按钮
	clearBtn := widget.NewButton("清空", func() {
		mw.logText.SetText("")
		mw.updateStatus("日志已清空")
	})

	historyBtn := widget.NewButton("历史", func() {
		mw.updateStatus("历史功能待实现")
	})

	logControls := container.NewHBox(
		mw.statusBar,
		widget.NewSeparator(),
		clearBtn,
		historyBtn,
	)

	// 创建带滚动的日志区域
	logScroll := container.NewScroll(mw.logText)
	logScroll.SetMinSize(fyne.NewSize(0, 150))

	// 组装底部区域
	bottomArea := container.NewBorder(
		logControls, // top
		nil,         // bottom
		nil,         // left
		nil,         // right
		logScroll,   // center
	)

	return bottomArea
}

// saveGlobalConfig 保存全局配置
func (mw *MainWindow) saveGlobalConfig() {
	// 更新配置
	mw.config.Proxy = mw.proxyEntry.Text
	mw.config.MagicName = mw.nameEntry.Text

	if portStr := mw.portEntry.Text; portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			mw.config.DefaultPort = port
		}
	}

	// 保存配置
	if err := mw.config.Save(); err != nil {
		mw.updateStatus("保存配置失败: " + err.Error())
		mw.addLog("ERROR: 保存配置失败: " + err.Error())
	} else {
		mw.updateStatus("配置已保存")
		mw.addLog("INFO: 全局配置已保存")
	}
}

// addLog 添加日志
func (mw *MainWindow) addLog(message string) {
	if mw.logText != nil {
		currentText := mw.logText.Text
		timestamp := time.Now().Format("15:04:05")
		newText := fmt.Sprintf("%s [%s] %s\n", currentText, timestamp, message)
		fyne.Do(func() {
			mw.logText.SetText(newText)

			// 滚动到底部
			mw.logText.CursorRow = len(strings.Split(newText, "\n"))
		})
	}
}

// testProxy 测试代理连接
func (mw *MainWindow) testProxy() {
	proxyURL := strings.TrimSpace(mw.proxyEntry.Text)

	// 如果代理为空，测试直连
	if proxyURL == "" {
		mw.updateStatus("正在测试直连...")
		mw.addLog("INFO: 开始测试直连")
	} else {
		mw.updateStatus("正在测试代理连接...")
		mw.addLog("INFO: 开始测试代理连接: " + proxyURL)
	}

	// 异步执行测试
	go func() {
		// 测试多个URL
		testURLs := []struct {
			name string
			url  string
		}{
			{"GitHub Frida API", "https://api.github.com/repos/frida/frida/releases/latest"},
			{"Google", "https://www.google.com"},
		}

		var results []string
		var successCount int

		for _, test := range testURLs {
			success, message, err := utils.TestProxy(proxyURL, test.url, mw.config.Timeout)

			if success {
				results = append(results, fmt.Sprintf("✓ %s: %s", test.name, message))
				successCount++
				mw.addLog(fmt.Sprintf("SUCCESS: %s - %s", test.name, message))
			} else {
				results = append(results, fmt.Sprintf("✗ %s: %s", test.name, message))
				mw.addLog(fmt.Sprintf("ERROR: %s - %s", test.name, message))
				if err != nil {
					mw.addLog("ERROR: " + err.Error())
				}
			}
		}

		// 更新UI
		if successCount > 0 {
			if successCount == len(testURLs) {
				mw.updateStatus("代理测试完全成功")
				dialog.ShowInformation("代理测试结果",
					fmt.Sprintf("测试成功！(%d/%d)\n\n%s",
						successCount, len(testURLs), strings.Join(results, "\n")),
					mw.window)
			} else {
				mw.updateStatus(fmt.Sprintf("代理测试部分成功 (%d/%d)", successCount, len(testURLs)))
				dialog.ShowInformation("代理测试结果",
					fmt.Sprintf("部分成功 (%d/%d)\n\n%s",
						successCount, len(testURLs), strings.Join(results, "\n")),
					mw.window)
			}
		} else {
			mw.updateStatus("代理测试失败")
			dialog.ShowError(
				fmt.Errorf("代理测试失败\n\n%s", strings.Join(results, "\n")),
				mw.window)
		}
	}()
}

// generateRandomName 生成随机名称
func (mw *MainWindow) generateRandomName() {
	randomName := utils.GenerateRandomName()
	mw.nameEntry.SetText(randomName)
	mw.updateStatus("已生成随机名称: " + randomName)
	mw.addLog("INFO: 生成随机名称: " + randomName)
}
