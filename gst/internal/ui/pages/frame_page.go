package pages

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"gst/internal/core"
)

// FramePage 帧分析页面
type FramePage struct {
	topFramesList *widget.List
	detailView    *fyne.Container

	frames          []core.FrameInfo
	onFrameSelected func(frame *core.FrameInfo)
}

func NewFramePage(onFrameSelected func(frame *core.FrameInfo)) *FramePage {
	fp := &FramePage{
		onFrameSelected: onFrameSelected,
	}

	// 帧列表 - 显示: 帧号, 总耗时
	fp.topFramesList = widget.NewList(
		func() int { return len(fp.frames) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(li widget.ListItemID, o fyne.CanvasObject) {
			frame := fp.frames[li]
			o.(*widget.Label).SetText(
				fmt.Sprintf("Frame %d: %d us", frame.FrameNum, frame.TotalTimeUs))
		},
	)

	fp.topFramesList.OnSelected = func(id widget.ListItemID) {
		if fp.onFrameSelected != nil {
			fp.onFrameSelected(&fp.frames[id])
		}
	}

	// 右侧详情区
	fp.detailView = container.NewVBox(
		widget.NewLabel("Select a frame to view details"),
	)

	split := container.NewHSplit(fp.topFramesList, fp.detailView)
	split.SetOffset(0.3)

	return fp
}

func (fp *FramePage) SetFrames(frames []core.FrameInfo) {
	fp.frames = frames
	fp.topFramesList.Refresh()
}
