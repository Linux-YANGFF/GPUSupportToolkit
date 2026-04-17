package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// navItem 导航项
type navItem struct {
	ID    string
	Label string
	Icon  string // 图标名称
}

// NavigationBar 左侧导航栏
type NavigationBar struct {
	onSelect func(page string)
	selected string
	buttons  []*widget.Button
	items    []navItem
}

// NewNavigationBar 创建导航栏
func NewNavigationBar(onSelect func(string)) *NavigationBar {
	nb := &NavigationBar{
		onSelect: onSelect,
		selected: "home",
		items: []navItem{
			{ID: "home", Label: "首页", Icon: "home"},
			{ID: "search", Label: "关键字检索", Icon: "search"},
			{ID: "timerange", Label: "时间段检索", Icon: "clock"},
			{ID: "frame", Label: "帧分析", Icon: "document"},
			{ID: "shader", Label: "Shader统计", Icon: "color"},
			{ID: "export", Label: "导出结果", Icon: "upload"},
		},
	}
	return nb
}

// Build 构建导航栏UI
func (nb *NavigationBar) Build() fyne.CanvasObject {
	itemContainer := container.NewVBox()

	for i := range nb.items {
		item := &nb.items[i]
		btn := nb.createNavButton(item)
		nb.buttons = append(nb.buttons, btn)
		itemContainer.Add(btn)
	}

	// 添加弹性空间
	itemContainer.Add(layout.NewSpacer())

	return itemContainer
}

// createNavButton 创建导航按钮
func (nb *NavigationBar) createNavButton(item *navItem) *widget.Button {
	isSelected := item.ID == nb.selected

	btn := widget.NewButton(item.Label, func() {
		nb.select(item.ID)
		if nb.onSelect != nil {
			nb.onSelect(item.ID)
		}
	})

	if isSelected {
		btn.Importance = widget.HighImportance
	}

	return btn
}

// select 选择导航项
func (nb *NavigationBar) select(id string) {
	nb.selected = id

	// 更新所有按钮状态
	for i, btn := range nb.buttons {
		item := &nb.items[i]
		if item.ID == id {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.MediumImportance
		}
	}
}

// SelectedID 返回当前选中的ID
func (nb *NavigationBar) SelectedID() string {
	return nb.selected
}

// SetSelected 设置选中项
func (nb *NavigationBar) SetSelected(id string) {
	nb.select(id)
}
