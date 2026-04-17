package exporter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"gst/internal/core"
)

// Exporter 导出接口
type Exporter interface {
	Export(w io.Writer) error
}

// TXTExporter TXT格式导出
type TXTExporter struct {
	Results []core.SearchResult
}

func (e TXTExporter) Export(w io.Writer) error {
	// 按行号排序输出
	sort.Slice(e.Results, func(i, j int) bool {
		return e.Results[i].LineNum < e.Results[j].LineNum
	})
	for _, r := range e.Results {
		if _, err := fmt.Fprintf(w, "[%d] %s\n", r.LineNum, r.Content); err != nil {
			return err
		}
	}
	return nil
}

// CSVExporter CSV格式导出
type CSVExporter struct {
	Data interface{} // SearchResult, FrameInfo, FuncStats, ShaderInfo 等
}

func (e CSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	switch v := e.Data.(type) {
	case []core.SearchResult:
		if err := csvWriter.Write([]string{"LineNum", "Content", "PageNum"}); err != nil {
			return err
		}
		for _, r := range v {
			if err := csvWriter.Write([]string{
				fmt.Sprintf("%d", r.LineNum),
				r.Content,
				fmt.Sprintf("%d", r.PageNum),
			}); err != nil {
				return err
			}
		}
	case []core.FuncStats:
		if err := csvWriter.Write([]string{"FuncName", "CallCount", "TotalTimeUs", "AvgTimeUs"}); err != nil {
			return err
		}
		for _, s := range v {
			if err := csvWriter.Write([]string{
				s.FuncName,
				fmt.Sprintf("%d", s.CallCount),
				fmt.Sprintf("%d", s.TotalTimeUs),
				fmt.Sprintf("%d", s.AvgTimeUs),
			}); err != nil {
				return err
			}
		}
	case []core.FrameInfo:
		if err := csvWriter.Write([]string{"FrameNum", "StartLine", "EndLine", "TotalTimeUs", "APICallCount"}); err != nil {
			return err
		}
		for _, f := range v {
			if err := csvWriter.Write([]string{
				fmt.Sprintf("%d", f.FrameNum),
				fmt.Sprintf("%d", f.StartLine),
				fmt.Sprintf("%d", f.EndLine),
				fmt.Sprintf("%d", f.TotalTimeUs),
				fmt.Sprintf("%d", len(f.APICalls)),
			}); err != nil {
				return err
			}
		}
	case []core.ShaderInfo:
		if err := csvWriter.Write([]string{"Type", "CompileCount", "TotalCompileTimeUs"}); err != nil {
			return err
		}
		for _, s := range v {
			if err := csvWriter.Write([]string{
				s.Type,
				fmt.Sprintf("%d", s.CompileCount),
				fmt.Sprintf("%d", s.TotalCompileTimeUs),
			}); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported data type for CSV export")
	}
	return nil
}

// JSONExporter JSON格式导出
type JSONExporter struct {
	Data interface{}
}

func (e JSONExporter) Export(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(e.Data)
}

// ExportSearchResults 导出检索结果
func ExportSearchResults(results []core.SearchResult, format string, w io.Writer) error {
	switch format {
	case "txt":
		return TXTExporter{Results: results}.Export(w)
	case "csv":
		return CSVExporter{Data: results}.Export(w)
	case "json":
		return JSONExporter{Data: results}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportFrameDetail 导出帧详情
func ExportFrameDetail(frame *core.FrameInfo, format string, w io.Writer) error {
	if frame == nil {
		return fmt.Errorf("frame is nil")
	}

	switch format {
	case "txt", "csv", "json":
		// txt/csv format uses FrameInfo slice for consistent handling
		frames := []core.FrameInfo{*frame}
		switch format {
		case "txt":
			return exportFrameDetailTxt(frames, w)
		case "csv":
			return CSVExporter{Data: frames}.Export(w)
		case "json":
			return JSONExporter{Data: frame}.Export(w)
		}
	}
	return fmt.Errorf("unsupported format: %s", format)
}

// exportFrameDetailTxt 以TXT格式导出帧详情
func exportFrameDetailTxt(frames []core.FrameInfo, w io.Writer) error {
	for _, frame := range frames {
		if _, err := fmt.Fprintf(w, "Frame #%d\n", frame.FrameNum); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Lines: %d - %d\n", frame.StartLine, frame.EndLine); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Total Time: %d us (%.2f ms)\n", frame.TotalTimeUs, float64(frame.TotalTimeUs)/1000.0); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "API Calls: %d\n\n", len(frame.APICalls)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%-30s %10s %10s\n", "API Name", "Count", "Time(us)"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s\n", "--------------------------------------------------"); err != nil {
			return err
		}
		for _, call := range frame.APICalls {
			if _, err := fmt.Fprintf(w, "%-30s %10d %10d\n", call.APIName, call.Count, call.TimeUs); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

// ExportFuncStats 导出函数统计
func ExportFuncStats(stats []core.FuncStats, format string, w io.Writer) error {
	switch format {
	case "txt":
		return exportFuncStatsTxt(stats, w)
	case "csv":
		return CSVExporter{Data: stats}.Export(w)
	case "json":
		return JSONExporter{Data: stats}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportFuncStatsTxt 以TXT格式导出函数统计
func exportFuncStatsTxt(stats []core.FuncStats, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%-30s %10s %15s %15s\n", "Function", "CallCount", "TotalTime(us)", "AvgTime(us)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", "--------------------------------------------------"); err != nil {
		return err
	}
	for _, s := range stats {
		if _, err := fmt.Fprintf(w, "%-30s %10d %15d %15d\n", s.FuncName, s.CallCount, s.TotalTimeUs, s.AvgTimeUs); err != nil {
			return err
		}
	}
	return nil
}

// ExportShaderStats 导出Shader统计
func ExportShaderStats(infos []core.ShaderInfo, format string, w io.Writer) error {
	switch format {
	case "txt":
		return exportShaderStatsTxt(infos, w)
	case "csv":
		return CSVExporter{Data: infos}.Export(w)
	case "json":
		return JSONExporter{Data: infos}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// exportShaderStatsTxt 以TXT格式导出Shader统计
func exportShaderStatsTxt(infos []core.ShaderInfo, w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%-20s %15s %20s\n", "Type", "CompileCount", "TotalCompileTime(us)"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", "--------------------------------------------------"); err != nil {
		return err
	}
	for _, s := range infos {
		if _, err := fmt.Fprintf(w, "%-20s %15d %20d\n", s.Type, s.CompileCount, s.TotalCompileTimeUs); err != nil {
			return err
		}
	}
	return nil
}
