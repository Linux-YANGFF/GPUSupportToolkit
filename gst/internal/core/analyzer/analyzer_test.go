package analyzer

import (
	"testing"
	"gst/internal/core"
)

func createTestParsedLog() *core.ParsedLog {
	return &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum:    0,
				StartLine:   1,
				EndLine:     10,
				TotalTimeUs: 50000,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", Count: 10, TimeUs: 1000, LineNum: 2},
					{APIName: "glDrawElements", Count: 5, TimeUs: 40000, LineNum: 5},
					{APIName: "glShaderSource", Count: 1, TimeUs: 5000, LineNum: 8},
				},
			},
			{
				FrameNum:    1,
				StartLine:   11,
				EndLine:     20,
				TotalTimeUs: 80000,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", Count: 15, TimeUs: 2000, LineNum: 12},
					{APIName: "glDrawElements", Count: 8, TimeUs: 70000, LineNum: 15},
					{APIName: "glCompileShader", Count: 1, TimeUs: 8000, LineNum: 18},
				},
			},
			{
				FrameNum:    2,
				StartLine:   21,
				EndLine:     30,
				TotalTimeUs: 30000,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", Count: 5, TimeUs: 500, LineNum: 22},
					{APIName: "glDrawElements", Count: 3, TimeUs: 25000, LineNum: 25},
				},
			},
		},
		TotalTimeUs: 160000,
		FPS:         10.0,
	}
}

// FrameAnalyzer tests
func TestFrameAnalyzer_FindTopSlowFrames(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewFrameAnalyzer(log)

	tests := []struct {
		name   string
		n      int
		expect int
	}{
		{"top 1", 1, 1},
		{"top 2", 2, 2},
		{"top all", 10, 3},
		{"n=0", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.FindTopSlowFrames(tt.n)
			if len(result) != tt.expect {
				t.Errorf("FindTopSlowFrames(n=%d) returned %d frames, want %d", tt.n, len(result), tt.expect)
			}
		})
	}

	// Verify ordering
	top2 := analyzer.FindTopSlowFrames(2)
	if top2[0].TotalTimeUs < top2[1].TotalTimeUs {
		t.Error("Top frames should be sorted by TotalTimeUs descending")
	}
}

func TestFrameAnalyzer_GetFrameSummary(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewFrameAnalyzer(log)

	summary := analyzer.GetFrameSummary()
	if summary == nil {
		t.Fatal("GetFrameSummary returned nil")
	}

	if summary["total_frames"] != 3 {
		t.Errorf("total_frames = %v, want 3", summary["total_frames"])
	}
}

func TestFrameAnalyzer_NilLog(t *testing.T) {
	analyzer := NewFrameAnalyzer(nil)
	result := analyzer.FindTopSlowFrames(5)
	if result != nil {
		t.Error("FindTopSlowFrames with nil log should return nil")
	}
}

// FuncAnalyzer tests
func TestFuncAnalyzer_Analyze(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewFuncAnalyzer(log)
	stats := analyzer.Analyze()

	if len(stats) == 0 {
		t.Fatal("Analyze returned empty")
	}

	// Verify glDrawElements is first (highest total time: 40000+70000+25000=135000)
	if stats[0].FuncName != "glDrawElements" {
		t.Errorf("Expected first func to be glDrawElements, got %s", stats[0].FuncName)
	}
}

func TestFuncAnalyzer_Analyze_NilLog(t *testing.T) {
	analyzer := NewFuncAnalyzer(nil)
	result := analyzer.Analyze()
	if result != nil {
		t.Error("Analyze with nil log should return nil")
	}
}

func TestFuncAnalyzer_FilterByPrefix(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewFuncAnalyzer(log)

	glStats := analyzer.FilterByPrefix("gl")
	nonGlStats := analyzer.FilterByPrefix("glDraw")

	// All stats start with "gl"
	if len(glStats) != len(analyzer.Analyze()) {
		t.Errorf("FilterByPrefix(gl) should return all stats")
	}

	// Only glDrawElements starts with glDraw
	if len(nonGlStats) != 1 || nonGlStats[0].FuncName != "glDrawElements" {
		t.Error("FilterByPrefix(glDraw) should return only glDrawElements")
	}
}

func TestFuncAnalyzer_FilterByPrefix_EmptyPrefix(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewFuncAnalyzer(log)

	allStats := analyzer.FilterByPrefix("")
	if len(allStats) != len(analyzer.Analyze()) {
		t.Error("FilterByPrefix(\"\") should return all stats")
	}
}

// ShaderAnalyzer tests
func TestShaderAnalyzer_Analyze(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewShaderAnalyzer(log)
	shaders := analyzer.Analyze()

	if len(shaders) == 0 {
		t.Fatal("Analyze returned empty")
	}

	// Verify shader-related APIs are detected
	shaderTypes := make(map[string]bool)
	for _, s := range shaders {
		shaderTypes[s.Type] = true
	}

	if !shaderTypes["Source"] {
		t.Error("glShaderSource should be detected")
	}
	if !shaderTypes["Compile"] {
		t.Error("glCompileShader should be detected")
	}
}

func TestShaderAnalyzer_Analyze_NilLog(t *testing.T) {
	analyzer := NewShaderAnalyzer(nil)
	result := analyzer.Analyze()
	if result != nil {
		t.Error("Analyze with nil log should return nil")
	}
}

func TestShaderAnalyzer_GetShaderSummary(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewShaderAnalyzer(log)

	summary := analyzer.GetShaderSummary()
	if summary == nil {
		t.Fatal("GetShaderSummary returned nil")
	}

	if summary["shader_types"] == nil {
		t.Error("shader_types should be set")
	}
}
