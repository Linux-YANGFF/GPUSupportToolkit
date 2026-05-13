package exporter

import (
	"bytes"
	"strings"
	"testing"
	"gst/internal/core"
)

func TestTXTExporter_Export(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 20, Content: "glDrawElements", PageNum: 1},
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}

	var buf bytes.Buffer
	exporter := TXTExporter{Results: results}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("TXTExporter.Export failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Should be sorted by LineNum (10 before 20)
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "[10]") {
		t.Errorf("First line should contain [10], got: %s", lines[0])
	}
}

func TestCSVExporter_Export_SearchResult(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
		{LineNum: 20, Content: "glDrawElements", PageNum: 1},
	}

	var buf bytes.Buffer
	exporter := SearchResultCSVExporter{Results: results}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("SearchResultCSVExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LineNum,Content,PageNum") {
		t.Error("SearchResultCSVExporter missing header")
	}
	if !strings.Contains(output, "10,glBindBuffer,1") {
		t.Error("SearchResultCSVExporter missing first row")
	}
}

func TestCSVExporter_Export_FuncStats(t *testing.T) {
	stats := []core.FuncStats{
		{FuncName: "glBindBuffer", CallCount: 491, TotalTimeUs: 588000, AvgTimeUs: 1197},
	}

	var buf bytes.Buffer
	exporter := FuncStatsCSVExporter{Stats: stats}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("FuncStatsCSVExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "FuncName,CallCount,TotalTimeUs,AvgTimeUs") {
		t.Error("FuncStatsCSVExporter missing header")
	}
}

func TestCSVExporter_Export_ShaderInfo(t *testing.T) {
	infos := []core.ShaderCompileInfo{
		{Type: "Vertex", CompileCount: 10, TotalCompileTimeUs: 50000},
	}

	var buf bytes.Buffer
	exporter := ShaderCompileCSVExporter{Infos: infos}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("ShaderCompileCSVExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Type,CompileCount,TotalCompileTimeUs") {
		t.Error("ShaderCompileCSVExporter missing header")
	}
}

func TestJSONExporter_Export(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}

	var buf bytes.Buffer
	exporter := JSONExporter[[]core.SearchResult]{Data: results}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("JSONExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LineNum") || !strings.Contains(output, "10") {
		t.Error("JSONExporter missing expected content")
	}
}

func TestExportSearchResults(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}

	tests := []struct {
		format string
		wantErr bool
	}{
		{"txt", false},
		{"csv", false},
		{"json", false},
		{"xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExportSearchResults(results, tt.format, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExportSearchResults(%s) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestExportFrameDetail(t *testing.T) {
	frame := &core.FrameInfo{
		FrameNum:    0,
		StartLine:   10,
		EndLine:     20,
		TotalTimeUs: 30000,
		APICalls: []core.APILogEntry{
			{APIName: "glBindBuffer", Count: 491, TimeUs: 588, LineNum: 10},
		},
	}

	tests := []struct {
		format string
		wantErr bool
	}{
		{"txt", false},
		{"csv", false},
		{"json", false},
		{"xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExportFrameDetail(frame, tt.format, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExportFrameDetail(%s) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestExportFrameDetail_Nil(t *testing.T) {
	var buf bytes.Buffer
	err := ExportFrameDetail(nil, "txt", &buf)
	if err == nil {
		t.Error("ExportFrameDetail should fail with nil frame")
	}
}

func TestExportFuncStats(t *testing.T) {
	stats := []core.FuncStats{
		{FuncName: "glBindBuffer", CallCount: 491, TotalTimeUs: 588000, AvgTimeUs: 1197},
	}

	tests := []struct {
		format string
		wantErr bool
	}{
		{"txt", false},
		{"csv", false},
		{"json", false},
		{"xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExportFuncStats(stats, tt.format, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExportFuncStats(%s) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestExportShaderStats(t *testing.T) {
	infos := []core.ShaderCompileInfo{
		{Type: "Vertex", CompileCount: 10, TotalCompileTimeUs: 50000},
	}

	tests := []struct {
		format string
		wantErr bool
	}{
		{"txt", false},
		{"csv", false},
		{"json", false},
		{"xml", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			var buf bytes.Buffer
			err := ExportShaderStats(infos, tt.format, &buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExportShaderStats(%s) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestSingleFrameCSVExporter_Export(t *testing.T) {
	frame := core.FrameInfo{
		FrameNum:    0,
		StartLine:   1,
		EndLine:     10,
		TotalTimeUs: 50000,
		SwapBufferTimeUs: 5000,
		APITotalTimeUs: 45000,
		APICalls: []core.APILogEntry{
			{APIName: "glBindBuffer", Count: 10, LineNum: 2},
		},
	}
	var buf bytes.Buffer
	exporter := SingleFrameCSVExporter{Frame: frame}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("SingleFrameCSVExporter.Export failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "FrameNum,StartLine,EndLine,TotalTimeUs,SwapBufferTimeUs,APITotalTimeUs,APICallCount") {
		t.Error("SingleFrameCSVExporter missing header")
	}
	if !strings.Contains(output, "0,1,10,50000,5000,45000,1") {
		t.Error("SingleFrameCSVExporter missing frame data")
	}
}

func TestShaderInfoCSVExporter_Export(t *testing.T) {
	shaders := []*core.ShaderInfo{
		{ID: 1, Source: "void main() {}"},
		{ID: 2, Source: "#version 330"},
	}
	var buf bytes.Buffer
	exporter := ShaderInfoCSVExporter{Shaders: shaders}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("ShaderInfoCSVExporter.Export failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "ID,Source") {
		t.Error("ShaderInfoCSVExporter missing header")
	}
	if !strings.Contains(output, "1,void main() {}") {
		t.Error("ShaderInfoCSVExporter missing shader data")
	}
}

func TestCSVExporter_Generic(t *testing.T) {
	data := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}
	var buf bytes.Buffer
	exporter := CSVExporter[[]core.SearchResult]{Data: data}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("CSVExporter.Export failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("CSVExporter should produce output")
	}
}

func TestExportFrameDetailTxt_MultipleFrames(t *testing.T) {
	frames := []core.FrameInfo{
		{
			FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000,
			APICalls: []core.APILogEntry{
				{APIName: "glBindBuffer", Count: 10, TimeUs: 1000},
			},
		},
		{
			FrameNum: 1, StartLine: 11, EndLine: 20, TotalTimeUs: 80000,
			APICalls: []core.APILogEntry{
				{APIName: "glDrawElements", Count: 5, TimeUs: 40000},
			},
		},
	}
	var buf bytes.Buffer
	err := exportFrameDetailTxt(frames, &buf)
	if err != nil {
		t.Fatalf("exportFrameDetailTxt failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Frame #0") {
		t.Error("missing Frame #0")
	}
	if !strings.Contains(output, "Frame #1") {
		t.Error("missing Frame #1")
	}
}

func TestExportFuncStatsTxt_MultipleStats(t *testing.T) {
	stats := []core.FuncStats{
		{FuncName: "glBindBuffer", CallCount: 491, TotalTimeUs: 588000, AvgTimeUs: 1197},
		{FuncName: "glDrawElements", CallCount: 493, TotalTimeUs: 11214, AvgTimeUs: 22},
	}
	var buf bytes.Buffer
	err := exportFuncStatsTxt(stats, &buf)
	if err != nil {
		t.Fatalf("exportFuncStatsTxt failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "glBindBuffer") {
		t.Error("missing glBindBuffer")
	}
	if !strings.Contains(output, "glDrawElements") {
		t.Error("missing glDrawElements")
	}
}

func TestExportShaderStatsTxt_MultipleInfos(t *testing.T) {
	infos := []core.ShaderCompileInfo{
		{Type: "Vertex", CompileCount: 10, TotalCompileTimeUs: 50000},
		{Type: "Fragment", CompileCount: 5, TotalCompileTimeUs: 25000},
	}
	var buf bytes.Buffer
	err := exportShaderStatsTxt(infos, &buf)
	if err != nil {
		t.Fatalf("exportShaderStatsTxt failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Vertex") && !strings.Contains(output, "Fragment") {
		t.Error("missing shader stats")
	}
}

func TestExportFramesTxt(t *testing.T) {
	frames := []core.FrameInfo{
		{FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000},
	}
	var buf bytes.Buffer
	err := ExportFramesTxt(frames, &buf)
	if err != nil {
		t.Fatalf("ExportFramesTxt failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Frame #0") {
		t.Error("missing Frame #0")
	}
}

func TestExportFuncStatsTxt(t *testing.T) {
	stats := []core.FuncStats{
		{FuncName: "glBindBuffer", CallCount: 1, TotalTimeUs: 1000, AvgTimeUs: 1000},
	}
	var buf bytes.Buffer
	err := ExportFuncStatsTxt(stats, &buf)
	if err != nil {
		t.Fatalf("ExportFuncStatsTxt failed: %v", err)
	}
	if !strings.Contains(buf.String(), "glBindBuffer") {
		t.Error("missing stats")
	}
}

func TestExportShaderInfosTxt(t *testing.T) {
	shaders := []*core.ShaderInfo{
		{ID: 1, Source: "void main() { gl_Position = vec4(1.0); }"},
	}
	var buf bytes.Buffer
	err := ExportShaderInfosTxt(shaders, &buf)
	if err != nil {
		t.Fatalf("ExportShaderInfosTxt failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Shader ID: 1") {
		t.Error("missing Shader ID")
	}
}

func TestExportShaderInfosTxt_Multiple(t *testing.T) {
	shaders := []*core.ShaderInfo{
		{ID: 1, Source: "vertex shader"},
		{ID: 2, Source: "fragment shader"},
	}
	var buf bytes.Buffer
	err := ExportShaderInfosTxt(shaders, &buf)
	if err != nil {
		t.Fatalf("ExportShaderInfosTxt failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Shader ID: 1") || !strings.Contains(output, "Shader ID: 2") {
		t.Error("missing shader IDs")
	}
}

func TestExportSearchResultsTxt(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}
	var buf bytes.Buffer
	err := ExportSearchResultsTxt(results, &buf)
	if err != nil {
		t.Fatalf("ExportSearchResultsTxt failed: %v", err)
	}
	if !strings.Contains(buf.String(), "[10]") {
		t.Error("missing line number")
	}
}

func TestExportAnalysisResult_Frames(t *testing.T) {
	frames := []core.FrameInfo{
		{FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000},
	}
	var buf bytes.Buffer
	err := ExportAnalysisResult(frames, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportAnalysisResult(frames) failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Frame #0") {
		t.Error("missing Frame #0")
	}
}

func TestExportAnalysisResult_SingleFrame(t *testing.T) {
	frame := core.FrameInfo{FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000}
	var buf bytes.Buffer
	err := ExportAnalysisResult(frame, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportAnalysisResult(single frame) failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Frame #0") {
		t.Error("missing Frame #0")
	}
}

func TestExportAnalysisResult_FuncStats(t *testing.T) {
	stats := []core.FuncStats{
		{FuncName: "glBindBuffer", CallCount: 1, TotalTimeUs: 1000, AvgTimeUs: 1000},
	}
	var buf bytes.Buffer
	err := ExportAnalysisResult(stats, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportAnalysisResult(func stats) failed: %v", err)
	}
	if !strings.Contains(buf.String(), "glBindBuffer") {
		t.Error("missing func stats")
	}
}

func TestExportAnalysisResult_ShaderInfos(t *testing.T) {
	shaders := []*core.ShaderInfo{
		{ID: 1, Source: "void main() {}"},
	}
	var buf bytes.Buffer
	err := ExportAnalysisResult(shaders, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportAnalysisResult(shader infos) failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Shader ID: 1") {
		t.Error("missing Shader ID")
	}
}

func TestExportAnalysisResult_SearchResults(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}
	var buf bytes.Buffer
	err := ExportAnalysisResult(results, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportAnalysisResult(search results) failed: %v", err)
	}
	if !strings.Contains(buf.String(), "[10]") {
		t.Error("missing search result")
	}
}

func TestExportAnalysisResult_UnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := ExportAnalysisResult([]core.FrameInfo{}, "json", &buf)
	if err == nil {
		t.Error("should fail for non-txt format")
	}
}

func TestExportAnalysisResult_UnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	err := ExportAnalysisResult(42, "txt", &buf)
	if err == nil {
		t.Error("should fail for unsupported type")
	}
}

func TestFramesCSVExporter_Export(t *testing.T) {
	frames := []core.FrameInfo{
		{FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000, APICalls: []core.APILogEntry{{}}},
		{FrameNum: 1, StartLine: 11, EndLine: 20, TotalTimeUs: 80000, APICalls: []core.APILogEntry{{}, {}}},
	}
	var buf bytes.Buffer
	exporter := FramesCSVExporter{Frames: frames}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("FramesCSVExporter.Export failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "FrameNum,StartLine,EndLine,TotalTimeUs,APICallCount") {
		t.Error("FramesCSVExporter missing header")
	}
	if !strings.Contains(output, "0,1,10,50000,1") {
		t.Error("FramesCSVExporter missing first frame")
	}
}

func TestExportFuncStats_EmptyList(t *testing.T) {
	stats := []core.FuncStats{}
	var buf bytes.Buffer
	err := ExportFuncStats(stats, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportFuncStats empty failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Function") {
		t.Error("should have header even for empty list")
	}
}

func TestExportFrameDetail_CSV_Multiple(t *testing.T) {
	frame := &core.FrameInfo{
		FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000,
		APICalls: []core.APILogEntry{
			{APIName: "glBindBuffer", Count: 10, TimeUs: 1000},
		},
	}
	var buf bytes.Buffer
	err := ExportFrameDetail(frame, "csv", &buf)
	if err != nil {
		t.Fatalf("ExportFrameDetail csv failed: %v", err)
	}
	if !strings.Contains(buf.String(), "FrameNum") {
		t.Error("CSV export missing header")
	}
}

func TestExportFrameDetail_JSON(t *testing.T) {
	frame := &core.FrameInfo{
		FrameNum: 0, StartLine: 1, EndLine: 10, TotalTimeUs: 50000,
	}
	var buf bytes.Buffer
	err := ExportFrameDetail(frame, "json", &buf)
	if err != nil {
		t.Fatalf("ExportFrameDetail json failed: %v", err)
	}
	if !strings.Contains(buf.String(), "FrameNum") {
		t.Error("JSON export missing FrameNum")
	}
}

func TestExportSearchResults_TXT_Empty(t *testing.T) {
	results := []core.SearchResult{}
	var buf bytes.Buffer
	err := ExportSearchResults(results, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportSearchResults empty failed: %v", err)
	}
}

func TestExportShaderStats_Empty(t *testing.T) {
	infos := []core.ShaderCompileInfo{}
	var buf bytes.Buffer
	err := ExportShaderStats(infos, "txt", &buf)
	if err != nil {
		t.Fatalf("ExportShaderStats empty failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Type") {
		t.Error("should have header for empty list")
	}
}

func TestExportShaderInfosTxt_Empty(t *testing.T) {
	shaders := []*core.ShaderInfo{}
	var buf bytes.Buffer
	err := ExportShaderInfosTxt(shaders, &buf)
	if err != nil {
		t.Fatalf("ExportShaderInfosTxt empty failed: %v", err)
	}
}