package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// StatusBar 状态栏
type StatusBar struct {
	currentFile *widget.Label
	progress    *widget.ProgressBar
	osInfo      *widget.Label
	statusText  *widget.Label
}

// NewStatusBar 创建状态栏
func NewStatusBar() *StatusBar {
	return &StatusBar{
		currentFile: widget.NewLabel("No file open"),
		progress:    widget.NewProgressBar(),
		osInfo:      widget.NewLabel(""),
		statusText:  widget.NewLabel("就绪"),
	}
}

// SetFile 设置当前文件
func (sb *StatusBar) SetFile(path string) {
	if path == "" {
		sb.currentFile.SetText("文件: 未打开")
	} else {
		sb.currentFile.SetText("文件: " + path)
	}
}

// SetProgress 设置进度
func (sb *StatusBar) SetProgress(p float64) {
	if p < 0 || p > 1 {
		sb.progress.Hide()
	} else {
		sb.progress.Show()
		sb.progress.SetValue(p)
	}
}

// SetOSInfo 设置OS信息
func (sb *StatusBar) SetOSInfo(info string) {
	sb.osInfo.SetText("系统: " + info)
}

// SetStatus 设置状态文本
func (sb *StatusBar) SetStatus(status string) {
	sb.statusText.SetText(status)
}

// Build 构建状态栏UI
func (sb *StatusBar) Build() fyne.CanvasObject {
	statusLabel := canvas.NewText("状态:", theme.TextColor())
	statusLabel.TextSize = 12

	fileLabel := canvas.NewText("文件:", theme.TextColor())
	fileLabel.TextSize = 12

	osLabel := canvas.NewText("系统:", theme.TextColor())
	osLabel.TextSize = 12

	return container.NewHBox(
		layout.NewSpacer(),
		statusLabel,
		sb.statusText,
		layout.NewSpacer(),
		fileLabel,
		sb.currentFile,
		layout.NewSpacer(),
		osLabel,
		sb.osInfo,
	)
}
