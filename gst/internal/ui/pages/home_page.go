package pages

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// HomePage 首页
type HomePage struct {
	dropZone     *widget.Label
	fileLabel    *widget.Label
	window       fyne.Window
	currentFile  string
	onFileOpen   func(string)
}

// NewHomePage 创建首页
func NewHomePage() *HomePage {
	hp := &HomePage{
		fileLabel: widget.NewLabel("未打开文件"),
	}

	// 拖拽提示区域
	hp.dropZone = widget.NewLabel("拖拽日志文件到此处\n或点击下方按钮选择文件")
	hp.dropZone.Alignment = fyne.TextAlignCenter

	return hp
}

// SetWindow 设置所属窗口
func (hp *HomePage) SetWindow(w fyne.Window) {
	hp.window = w
}

// SetOnFileOpen 设置文件打开回调
func (hp *HomePage) SetOnFileOpen(cb func(string)) {
	hp.onFileOpen = cb
}

// Build 构建UI
func (hp *HomePage) Build() fyne.CanvasObject {
	// 标题
	title := canvas.NewText("欢迎使用 GST - GPU日志分析工具", theme.TextColor())
	title.TextSize = 24
	title.Alignment = fyne.TextAlignCenter

	// 副标题
	subtitle := canvas.NewText("支持 apitrace 日志和 profile 日志分析", theme.TextColor())
	subtitle.TextSize = 14
	subtitle.Alignment = fyne.TextAlignCenter

	// 文件信息
	fileInfoContainer := container.NewVBox(
		widget.NewSeparator(),
		hp.fileLabel,
		widget.NewSeparator(),
	)

	// 按钮
	openBtn := widget.NewButton("选择文件...", hp.onOpenFile)

	// 拖拽区域 (使用Label模拟，实际拖拽需要更复杂实现)
	dropContainer := container.NewVBox(
		layout.NewSpacer(),
		hp.dropZone,
		layout.NewSpacer(),
	)

	return container.NewBorder(
		nil, nil, nil, nil,
		container.NewVBox(
			layout.NewSpacer(),
			title,
			container.NewPaddings(10, 10, 10, 10, subtitle),
			container.NewPaddings(20, 20, 50, 50, dropContainer),
			openBtn,
			container.NewPaddings(10, 10, 10, 10, fileInfoContainer),
			layout.NewSpacer(),
		),
	)
}

// onOpenFile 打开文件
func (hp *HomePage) onOpenFile() {
	if hp.window == nil {
		return
	}
	dialog.ShowFileOpen(func(f fyne.URICloser, err error) {
		if err != nil || f == nil {
			return
		}
		uri := f.URI()
		if uri != nil {
			hp.SetFile(uri.Path())
		}
	}, hp.window)
}

// SetFile 设置当前文件
func (hp *HomePage) SetFile(path string) {
	hp.currentFile = path
	if path == "" {
		hp.fileLabel.SetText("未打开文件")
	} else {
		hp.fileLabel.SetText("当前文件: " + path)
	}
	if hp.onFileOpen != nil {
		hp.onFileOpen(path)
	}
}
