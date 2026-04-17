package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// setupMenu 设置菜单
func (mw *MainWindow) setupMenu() {
	// 文件菜单
	fileMenu := menu.NewMenu("文件",
		menu.NewItem("打开文件...", mw.onOpenFile),
		menu.NewSeparator(),
		menu.NewItem("退出", func() {
			mw.window.Close()
		}),
	)

	// 视图菜单
	viewMenu := menu.NewMenu("视图",
		menu.NewItem("首页", func() { mw.showPage("home") }),
		menu.NewItem("关键字检索", func() { mw.showPage("search") }),
		menu.NewItem("时间段检索", func() { mw.showPage("timerange") }),
		menu.NewItem("帧分析", func() { mw.showPage("frame") }),
		menu.NewItem("Shader统计", func() { mw.showPage("shader") }),
		menu.NewSeparator(),
		menu.NewItem("深色主题", func() {
			// 切换主题
		}),
	)

	// 帮助菜单
	helpMenu := menu.NewMenu("帮助",
		menu.NewItem("关于", mw.onAbout),
	)

	mainMenu := menu.NewMenuBar()
	mainMenu.Append(fileMenu)
	mainMenu.Append(viewMenu)
	mainMenu.Append(helpMenu)

	mw.window.SetMainMenu(mainMenu)

	// 设置快捷键
	mw.setupShortcuts()
}

// setupShortcuts 设置快捷键
func (mw *MainWindow) setupShortcuts() {
	// Ctrl+O 打开文件
	shortcutOpen := &fyne.Shortcut{
		Advanced: fyne.NewShortcutKey(fyne.KeyO, fyne.KeyModifierControl),
	}
	mw.window.AddShortcut(shortcutOpen, func(shortcut fyne.Shortcut) {
		mw.onOpenFile()
	})
}

// onOpenFile 打开文件处理
func (mw *MainWindow) onOpenFile() {
	// TODO: 实现文件选择对话框
}

// onAbout 关于对话框
func (mw *MainWindow) onAbout() {
	dialog.ShowCustom("关于", "关闭",
		widget.NewVBox(
			widget.NewLabel("GST - GPU Support Toolkit"),
			widget.NewLabel("版本: 0.1.0"),
			widget.NewLabel(""),
			widget.NewLabel("GPU日志分析工具"),
		),
		mw.window,
	)
}

// showPage 显示指定页面
func (mw *MainWindow) showPage(page string) {
	// TODO: 根据page切换内容区域
}
