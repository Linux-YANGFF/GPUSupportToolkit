package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// VirtualTable 虚拟表格 - 用于大数据量展示
// 使用组合模式，将 widget.Table 作为命名字段而非嵌入，避免直接依赖 Fyne 具体类型结构
type VirtualTable struct {
	Table        *widget.Table // 组合而非继承，更符合 Go 的组合设计原则
	TotalRows    int
	VisibleRows   int
	ScrollOffset  int
	RenderRow     func(i int) []string
	onRowSelected func(i int)
}

func NewVirtualTable(totalRows int, renderRow func(i int) []string) *VirtualTable {
	vt := &VirtualTable{
		TotalRows:    totalRows,
		VisibleRows:  20,
		ScrollOffset: 0,
		RenderRow:    renderRow,
	}

	// 只渲染可见区域
	vt.Table = widget.NewTable(
		func() (int, int) {
			if vt.RenderRow != nil && vt.TotalRows > 0 {
				return vt.TotalRows, len(vt.RenderRow(0))
			}
			return vt.TotalRows, 1
		},
		func(i widget.TableCellID) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			if vt.RenderRow != nil {
				rowData := vt.RenderRow(i.Row)
				if i.Col < len(rowData) {
					o.(*widget.Label).SetText(rowData[i.Col])
				}
			}
		},
	)

	return vt
}

func (vt *VirtualTable) SetOnRowSelected(cb func(i int)) {
	vt.onRowSelected = cb
}

func (vt *VirtualTable) ScrollToRow(row int) {
	vt.ScrollOffset = row
	vt.Table.Refresh()
}
