package pages

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"gst/internal/core"
)

// SearchPage 关键字检索页面
type SearchPage struct {
	keywordEntry   *widget.Entry
	resultTable    *widget.Table
	pageLabel      *widget.Label
	exportBtn      *widget.Button

	currentResults []core.SearchResult
	currentPage    int
	totalPages     int
	onSearch       func(keywords string)
	onExport       func()
}

func NewSearchPage(onSearch func(keywords string), onExport func()) *SearchPage {
	sp := &SearchPage{
		keywordEntry: widget.NewEntry(),
		pageLabel:    widget.NewLabel("Page 1 / 1"),
		exportBtn:    widget.NewButton("Export", onExport),
		onSearch:     onSearch,
		onExport:     onExport,
	}
	sp.keywordEntry.PlaceHolder = "Enter keywords (space separated)..."

	// 结果表格 - 2列: LineNum, Content
	sp.resultTable = widget.NewTable(
		func() (int, int) { return len(sp.currentResults), 2 },
		func(i widget.TableCellID) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Col == 0 {
				label.SetText(fmt.Sprintf("%d", sp.currentResults[i.Row].LineNum))
			} else {
				label.SetText(sp.currentResults[i.Row].Content)
			}
		},
	)

	searchBtn := widget.NewButton("Search", func() {
		if sp.onSearch != nil {
			sp.onSearch(sp.keywordEntry.Text)
		}
	})

	// 布局
	content := container.NewVBox(
		widget.NewLabel("Keyword Search"),
		container.NewHBox(sp.keywordEntry, searchBtn),
		sp.resultTable,
		container.NewHBox(sp.pageLabel, sp.exportBtn),
	)

	_ = content // suppress unused variable warning
	return sp
}

func (sp *SearchPage) SetResults(results []core.SearchResult) {
	sp.currentResults = results
	sp.currentPage = 0
	sp.totalPages = (len(results) + 999) / 1000
	if sp.totalPages == 0 {
		sp.totalPages = 1
	}
	sp.resultTable.Refresh()
	sp.pageLabel.SetText(fmt.Sprintf("Page %d / %d", sp.currentPage+1, sp.totalPages))
}
