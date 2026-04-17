package main

import (
	"gst/internal/platform"
	"gst/internal/ui"
)

func main() {
	// 检测 OS
	if !platform.IsSupportedOS() {
		println("Warning: Unsupported OS")
	}

	// 启动 UI
	mw := ui.NewMainWindow()
	mw.Show()
}
