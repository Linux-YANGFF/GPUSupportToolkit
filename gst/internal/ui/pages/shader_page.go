package pages

import (
	"fmt"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"gst/internal/core"
)

// ShaderPage Shader统计页面
type ShaderPage struct {
	shaderList *widget.List
	summary    *widget.Label

	shaders []core.ShaderInfo
}

func NewShaderPage() *ShaderPage {
	sp := &ShaderPage{}

	sp.shaderList = widget.NewList(
		func() int { return len(sp.shaders) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(li widget.ListItemID, o fyne.CanvasObject) {
			shader := sp.shaders[li]
			o.(*widget.Label).SetText(
				fmt.Sprintf("%s: %d calls, %d us",
					shader.Type, shader.CompileCount, shader.TotalCompileTimeUs))
		},
	)

	content := container.NewVBox(
		widget.NewLabel("Shader Statistics"),
		sp.shaderList,
	)

	return sp
}

func (sp *ShaderPage) SetShaders(shaders []core.ShaderInfo) {
	sp.shaders = shaders
	sp.shaderList.Refresh()
}
