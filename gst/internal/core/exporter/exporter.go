package exporter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"gst/internal/core"
)

type Exporter interface {
	Export(w io.Writer) error
}

type TXTExporter struct {
	Results []core.SearchResult
}

func (e TXTExporter) Export(w io.Writer) error {
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

type SearchResultCSVExporter struct {
	Results []core.SearchResult
}

func (e SearchResultCSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	if err := csvWriter.Write([]string{"LineNum", "Content", "PageNum"}); err != nil {
		return err
	}
	for _, r := range e.Results {
		if err := csvWriter.Write([]string{
			fmt.Sprintf("%d", r.LineNum),
			r.Content,
			fmt.Sprintf("%d", r.PageNum),
		}); err != nil {
			return err
		}
	}
	return nil
}

type FuncStatsCSVExporter struct {
	Stats []core.FuncStats
}

func (e FuncStatsCSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	if err := csvWriter.Write([]string{"FuncName", "CallCount", "TotalTimeUs", "AvgTimeUs"}); err != nil {
		return err
	}
	for _, s := range e.Stats {
		if err := csvWriter.Write([]string{
			s.FuncName,
			fmt.Sprintf("%d", s.CallCount),
			fmt.Sprintf("%d", s.TotalTimeUs),
			fmt.Sprintf("%d", s.AvgTimeUs),
		}); err != nil {
			return err
		}
	}
	return nil
}

type FramesCSVExporter struct {
	Frames []core.FrameInfo
}

func (e FramesCSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	if err := csvWriter.Write([]string{"FrameNum", "StartLine", "EndLine", "TotalTimeUs", "APICallCount"}); err != nil {
		return err
	}
	for _, f := range e.Frames {
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
	return nil
}

type SingleFrameCSVExporter struct {
	Frame core.FrameInfo
}

func (e SingleFrameCSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	if err := csvWriter.Write([]string{"FrameNum", "StartLine", "EndLine", "TotalTimeUs", "SwapBufferTimeUs", "APITotalTimeUs", "APICallCount"}); err != nil {
		return err
	}
	f := e.Frame
	if err := csvWriter.Write([]string{
		fmt.Sprintf("%d", f.FrameNum),
		fmt.Sprintf("%d", f.StartLine),
		fmt.Sprintf("%d", f.EndLine),
		fmt.Sprintf("%d", f.TotalTimeUs),
		fmt.Sprintf("%d", f.SwapBufferTimeUs),
		fmt.Sprintf("%d", f.APITotalTimeUs),
		fmt.Sprintf("%d", len(f.APICalls)),
	}); err != nil {
		return err
	}
	return nil
}

type ShaderCompileCSVExporter struct {
	Infos []core.ShaderCompileInfo
}

func (e ShaderCompileCSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	if err := csvWriter.Write([]string{"Type", "CompileCount", "TotalCompileTimeUs"}); err != nil {
		return err
	}
	for _, s := range e.Infos {
		if err := csvWriter.Write([]string{
			s.Type,
			fmt.Sprintf("%d", s.CompileCount),
			fmt.Sprintf("%d", s.TotalCompileTimeUs),
		}); err != nil {
			return err
		}
	}
	return nil
}

type ShaderInfoCSVExporter struct {
	Shaders []*core.ShaderInfo
}

func (e ShaderInfoCSVExporter) Export(w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()
	if err := csvWriter.Write([]string{"ID", "Source"}); err != nil {
		return err
	}
	for _, s := range e.Shaders {
		if err := csvWriter.Write([]string{
			fmt.Sprintf("%d", s.ID),
			s.Source,
		}); err != nil {
			return err
		}
	}
	return nil
}

type JSONExporter[T any] struct {
	Data T
}

func (e JSONExporter[T]) Export(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(e.Data)
}

type CSVExporter[T any] struct {
	Data T
}

func (e CSVExporter[T]) Export(w io.Writer) error {
	return json.NewEncoder(w).Encode(e.Data)
}

func ExportSearchResults(results []core.SearchResult, format string, w io.Writer) error {
	switch format {
	case "txt":
		return TXTExporter{Results: results}.Export(w)
	case "csv":
		return SearchResultCSVExporter{Results: results}.Export(w)
	case "json":
		return JSONExporter[[]core.SearchResult]{Data: results}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func ExportFrameDetail(frame *core.FrameInfo, format string, w io.Writer) error {
	if frame == nil {
		return fmt.Errorf("frame is nil")
	}
	switch format {
	case "txt":
		return exportFrameDetailTxt([]core.FrameInfo{*frame}, w)
	case "csv":
		return FramesCSVExporter{Frames: []core.FrameInfo{*frame}}.Export(w)
	case "json":
		return JSONExporter[*core.FrameInfo]{Data: frame}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

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

func ExportFuncStats(stats []core.FuncStats, format string, w io.Writer) error {
	switch format {
	case "txt":
		return exportFuncStatsTxt(stats, w)
	case "csv":
		return FuncStatsCSVExporter{Stats: stats}.Export(w)
	case "json":
		return JSONExporter[[]core.FuncStats]{Data: stats}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

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

func ExportShaderStats(infos []core.ShaderCompileInfo, format string, w io.Writer) error {
	switch format {
	case "txt":
		return exportShaderStatsTxt(infos, w)
	case "csv":
		return ShaderCompileCSVExporter{Infos: infos}.Export(w)
	case "json":
		return JSONExporter[[]core.ShaderCompileInfo]{Data: infos}.Export(w)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func exportShaderStatsTxt(infos []core.ShaderCompileInfo, w io.Writer) error {
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

func ExportFramesTxt(frames []core.FrameInfo, w io.Writer) error {
	return exportFrameDetailTxt(frames, w)
}

func ExportFuncStatsTxt(stats []core.FuncStats, w io.Writer) error {
	return exportFuncStatsTxt(stats, w)
}

func ExportShaderInfosTxt(shaders []*core.ShaderInfo, w io.Writer) error {
	return exportShaderInfoTxt(shaders, w)
}

func ExportSearchResultsTxt(results []core.SearchResult, w io.Writer) error {
	return TXTExporter{Results: results}.Export(w)
}

func exportShaderInfoTxt(shaders []*core.ShaderInfo, w io.Writer) error {
	for i, s := range shaders {
		if i > 0 {
			if _, err := fmt.Fprintf(w, "\n%s\n\n", strings.Repeat("=", 50)); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "Shader ID: %d\n", s.ID); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s\n", strings.Repeat("-", 50)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s\n", s.Source); err != nil {
			return err
		}
	}
	return nil
}

func ExportAnalysisResult(data interface{}, format string, w io.Writer) error {
	if format != "txt" {
		return fmt.Errorf("ExportAnalysisResult only supports txt format")
	}
	switch v := data.(type) {
	case []core.FrameInfo:
		return exportFrameDetailTxt(v, w)
	case core.FrameInfo:
		return exportFrameDetailTxt([]core.FrameInfo{v}, w)
	case []core.FuncStats:
		return exportFuncStatsTxt(v, w)
	case []*core.ShaderInfo:
		return exportShaderInfoTxt(v, w)
	case []core.SearchResult:
		return TXTExporter{Results: v}.Export(w)
	default:
		return fmt.Errorf("unsupported data type for analysis result export")
	}
}
