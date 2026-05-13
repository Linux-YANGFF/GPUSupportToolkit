package core

import (
	"encoding/json"
	"testing"
)

func TestAPILogEntry_Defaults(t *testing.T) {
	e := APILogEntry{}
	if e.APIName != "" {
		t.Error("APIName should default to empty")
	}
	if e.IsError {
		t.Error("IsError should default to false")
	}
	if e.HasNilPtr {
		t.Error("HasNilPtr should default to false")
	}
}

func TestAPILogEntry_Fields(t *testing.T) {
	e := APILogEntry{
		APIName:   "glBindBuffer",
		Count:     10,
		TimeUs:    1000,
		LineNum:   5,
		RawParams: "0x8892 199",
		GCAddr:    "0xabcdef",
		TID:       "0x1234",
		IsError:   true,
		ErrorCode: "GL_INVALID_OPERATION",
		HasNilPtr: true,
	}
	if e.APIName != "glBindBuffer" {
		t.Errorf("APIName = %s", e.APIName)
	}
	if e.Count != 10 {
		t.Errorf("Count = %d", e.Count)
	}
	if e.TimeUs != 1000 {
		t.Errorf("TimeUs = %d", e.TimeUs)
	}
	if e.LineNum != 5 {
		t.Errorf("LineNum = %d", e.LineNum)
	}
	if !e.IsError {
		t.Error("IsError should be true")
	}
	if !e.HasNilPtr {
		t.Error("HasNilPtr should be true")
	}
}

func TestFrameInfo_Defaults(t *testing.T) {
	f := FrameInfo{}
	if f.APISummary != nil {
		t.Error("APISummary should be nil by default")
	}
	if f.Shaders != nil {
		t.Error("Shaders should be nil by default")
	}
}

func TestParsedLog_Defaults(t *testing.T) {
	l := ParsedLog{}
	if l.FPS != 0 {
		t.Error("FPS should default to 0")
	}
	if l.TotalTimeUs != 0 {
		t.Error("TotalTimeUs should default to 0")
	}
}

func TestSearchResult_Defaults(t *testing.T) {
	r := SearchResult{}
	if r.LineNum != 0 {
		t.Error("LineNum should default to 0")
	}
	if r.PageNum != 0 {
		t.Error("PageNum should default to 0")
	}
}

func TestFuncStats_Defaults(t *testing.T) {
	s := FuncStats{}
	if s.FuncName != "" {
		t.Error("FuncName should default to empty")
	}
}

func TestShaderInfo_Defaults(t *testing.T) {
	s := ShaderInfo{}
	if s.ID != 0 {
		t.Error("ID should default to 0")
	}
}

func TestShaderCompileInfo_Defaults(t *testing.T) {
	s := ShaderCompileInfo{}
	if s.CompileCount != 0 {
		t.Error("CompileCount should default to 0")
	}
}

func TestSeverityConstants(t *testing.T) {
	if SeverityCritical != "critical" {
		t.Errorf("SeverityCritical = %s", SeverityCritical)
	}
	if SeverityHigh != "high" {
		t.Errorf("SeverityHigh = %s", SeverityHigh)
	}
	if SeverityMedium != "medium" {
		t.Errorf("SeverityMedium = %s", SeverityMedium)
	}
	if SeverityLow != "low" {
		t.Errorf("SeverityLow = %s", SeverityLow)
	}
	if SeverityInfo != "info" {
		t.Errorf("SeverityInfo = %s", SeverityInfo)
	}
}

func TestFinding_Defaults(t *testing.T) {
	f := Finding{}
	if f.Severity != "" {
		t.Error("Severity should default to empty")
	}
}

func TestFinding_Fields(t *testing.T) {
	f := Finding{
		Severity:       SeverityCritical,
		Category:       "driver_error",
		Description:    "test",
		Evidence:       "line 1",
		RootCauseChain: []string{"a", "b"},
		FixSuggestion:  "fix",
	}
	if f.Severity != SeverityCritical {
		t.Error("Severity mismatch")
	}
	if f.Category != "driver_error" {
		t.Error("Category mismatch")
	}
	if len(f.RootCauseChain) != 2 {
		t.Error("RootCauseChain length mismatch")
	}
}

func TestDiagnosisSummary_Defaults(t *testing.T) {
	s := DiagnosisSummary{}
	if s.TotalFindings != 0 {
		t.Error("TotalFindings should default to 0")
	}
}

func TestDiagnosisReport_Defaults(t *testing.T) {
	r := DiagnosisReport{}
	if r.SourceFile != "" {
		t.Error("SourceFile should default to empty")
	}
	if r.GeneratedAt != "" {
		t.Error("GeneratedAt should default to empty")
	}
}

func TestFrameSummary_Defaults(t *testing.T) {
	s := FrameSummary{}
	if s.TotalFrames != 0 {
		t.Error("TotalFrames should default to 0")
	}
}

func TestBufferSummary_Defaults(t *testing.T) {
	s := BufferSummary{}
	if s.TargetStats != nil {
		t.Error("TargetStats should be nil by default")
	}
}

func TestFuncSummary_Defaults(t *testing.T) {
	s := FuncSummary{}
	if s.TopFunctions != nil {
		t.Error("TopFunctions should be nil by default")
	}
}

func TestShaderSummary_Defaults(t *testing.T) {
	s := ShaderSummary{}
	if s.ShaderTypes != 0 {
		t.Error("ShaderTypes should default to 0")
	}
}

func TestAPILogEntry_JSONTags(t *testing.T) {
	e := APILogEntry{
		APIName:   "glBindBuffer",
		GCAddr:    "0x123",
		TID:       "0x456",
		IsError:   true,
		ErrorCode: "GL_ERR",
		HasNilPtr: true,
	}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestFinding_JSONTags(t *testing.T) {
	f := Finding{
		Severity:      SeverityHigh,
		Category:      "perf",
		Description:   "test",
		Evidence:      "ev",
		FixSuggestion: "fix",
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestFrameSummary_JSON(t *testing.T) {
	s := FrameSummary{
		TotalFrames: 10,
		AvgTimeUs:   5000,
		MaxTimeUs:   10000,
		MinTimeUs:   1000,
		TotalTimeUs: 50000,
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestBufferSummary_JSON(t *testing.T) {
	s := BufferSummary{
		TotalCount: 5,
		TotalSize:  1024,
		TargetStats: map[string]BufferTargetStat{
			"GL_ARRAY_BUFFER": {Count: 3, TotalSize: 512},
		},
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestFuncSummary_JSON(t *testing.T) {
	s := FuncSummary{
		TotalFunctions: 3,
		TotalCalls:     100,
		TotalTimeUs:    50000,
		TopFunctions:   []FuncStats{{FuncName: "glBindBuffer", CallCount: 50, TotalTimeUs: 25000, AvgTimeUs: 500}},
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestShaderSummary_JSON(t *testing.T) {
	s := ShaderSummary{
		ShaderTypes:  2,
		TotalCompile: 10,
		TotalTimeUs:  5000,
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestDiagnosisReport_JSON(t *testing.T) {
	r := DiagnosisReport{
		SourceFile:  "test.log",
		GeneratedAt: "2024-01-01T00:00:00Z",
		Summary: DiagnosisSummary{
			TotalFindings: 3,
			CriticalCount: 1,
			HighCount:     1,
			MediumCount:   1,
		},
		Findings: []Finding{
			{Severity: SeverityCritical, Description: "test", FixSuggestion: "fix"},
		},
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("json output should not be empty")
	}
}

func TestBufferInfo_Defaults(t *testing.T) {
	b := BufferInfo{}
	if b.ID != 0 {
		t.Error("ID should default to 0")
	}
}

func TestBufferInfo_Fields(t *testing.T) {
	b := BufferInfo{
		ID:     1,
		Target: "GL_ARRAY_BUFFER",
		Size:   1024,
		Usage:  "GL_STATIC_DRAW",
	}
	if b.ID != 1 {
		t.Error("ID mismatch")
	}
	if b.Target != "GL_ARRAY_BUFFER" {
		t.Error("Target mismatch")
	}
	if b.Size != 1024 {
		t.Error("Size mismatch")
	}
}
