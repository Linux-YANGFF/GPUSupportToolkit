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
	exporter := CSVExporter{Data: results}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("CSVExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LineNum,Content,PageNum") {
		t.Error("CSVExporter missing header")
	}
	if !strings.Contains(output, "10,glBindBuffer,1") {
		t.Error("CSVExporter missing first row")
	}
}

func TestCSVExporter_Export_FuncStats(t *testing.T) {
	stats := []core.FuncStats{
		{FuncName: "glBindBuffer", CallCount: 491, TotalTimeUs: 588000, AvgTimeUs: 1197},
	}

	var buf bytes.Buffer
	exporter := CSVExporter{Data: stats}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("CSVExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "FuncName,CallCount,TotalTimeUs,AvgTimeUs") {
		t.Error("CSVExporter missing header")
	}
}

func TestCSVExporter_Export_ShaderInfo(t *testing.T) {
	infos := []core.ShaderInfo{
		{Type: "Vertex", CompileCount: 10, TotalCompileTimeUs: 50000},
	}

	var buf bytes.Buffer
	exporter := CSVExporter{Data: infos}
	err := exporter.Export(&buf)
	if err != nil {
		t.Fatalf("CSVExporter.Export failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Type,CompileCount,TotalCompileTimeUs") {
		t.Error("CSVExporter missing header for ShaderInfo")
	}
}

func TestCSVExporter_Export_UnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	exporter := CSVExporter{Data: "unsupported"}
	err := exporter.Export(&buf)
	if err == nil {
		t.Error("CSVExporter should fail for unsupported type")
	}
}

func TestJSONExporter_Export(t *testing.T) {
	results := []core.SearchResult{
		{LineNum: 10, Content: "glBindBuffer", PageNum: 1},
	}

	var buf bytes.Buffer
	exporter := JSONExporter{Data: results}
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
	infos := []core.ShaderInfo{
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