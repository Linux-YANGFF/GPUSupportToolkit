package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// MainWindow 主窗口结构
type MainWindow struct {
	window fyne.Window
	nav    *NavigationBar
}

// NewMainWindow 创建主窗口
func NewMainWindow() *MainWindow {
	a := app.New()
	w := a.NewWindow("GST - GPU Support Toolkit")

	mw := &MainWindow{window: w}
	mw.setupUI()
	return mw
}

// setupUI 设置UI布局
func (mw *MainWindow) setupUI() {
	// 创建导航栏
	mw.nav = NewNavigationBar(mw.onNavSelected)

	// 创建内容区
	content := widget.NewLabel("Welcome to GST")

	// 左侧导航 + 右侧内容
	split := container.NewHSplit(mw.nav.Build(), content)
	split.SetOffset(0.2)

	mw.window.SetContent(split)
	mw.window.Resize(fyne.NewSize(1200, 800))
	mw.window.SetMaster()
}

// onNavSelected 导航选中回调
func (mw *MainWindow) onNavSelected(page string) {
	// 切换页面
}

// Show 显示窗口
func (mw *MainWindow) Show() {
	mw.window.Show()
}
