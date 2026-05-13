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

	if summary.TotalFrames != 3 {
		t.Errorf("total_frames = %v, want 3", summary.TotalFrames)
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

	if summary.ShaderTypes == 0 {
		t.Error("shader_types should be non-zero")
	}
}

func TestShaderAnalyzer_GetShaderSummary_NilLog(t *testing.T) {
	analyzer := NewShaderAnalyzer(nil)
	summary := analyzer.GetShaderSummary()
	if summary != nil {
		t.Error("GetShaderSummary with nil log should return nil")
	}
}

func TestShaderAnalyzer_IncShaderStat_MultipleCalls(t *testing.T) {
	analyzer := NewShaderAnalyzer(nil)
	analyzer.shaders = make(map[string]*core.ShaderCompileInfo)
	analyzer.incShaderStat("Compile", 1000)
	analyzer.incShaderStat("Compile", 2000)
	analyzer.incShaderStat("Create", 500)

	if info, ok := analyzer.shaders["Compile"]; !ok {
		t.Error("Compile stat should exist")
	} else {
		if info.CompileCount != 2 {
			t.Errorf("expected CompileCount=2, got %d", info.CompileCount)
		}
		if info.TotalCompileTimeUs != 3000 {
			t.Errorf("expected TotalCompileTimeUs=3000, got %d", info.TotalCompileTimeUs)
		}
	}
	if info, ok := analyzer.shaders["Create"]; !ok {
		t.Error("Create stat should exist")
	} else {
		if info.CompileCount != 1 {
			t.Errorf("expected CompileCount=1, got %d", info.CompileCount)
		}
	}
}

// FuncAnalyzer GetFuncSummary tests
func TestFuncAnalyzer_GetFuncSummary(t *testing.T) {
	log := createTestParsedLog()
	analyzer := NewFuncAnalyzer(log)

	summary := analyzer.GetFuncSummary()
	if summary == nil {
		t.Fatal("GetFuncSummary returned nil")
	}
	if summary.TotalFunctions == 0 {
		t.Error("TotalFunctions should be non-zero")
	}
	if summary.TotalCalls == 0 {
		t.Error("TotalCalls should be non-zero")
	}
	if summary.TotalTimeUs == 0 {
		t.Error("TotalTimeUs should be non-zero")
	}
	if len(summary.TopFunctions) > 10 {
		t.Error("TopFunctions should be at most 10")
	}
}

func TestFuncAnalyzer_GetFuncSummary_NilLog(t *testing.T) {
	analyzer := NewFuncAnalyzer(nil)
	summary := analyzer.GetFuncSummary()
	if summary != nil {
		t.Error("GetFuncSummary with nil log should return nil")
	}
}

func TestFuncAnalyzer_GetFuncSummary_EmptyLog(t *testing.T) {
	log := &core.ParsedLog{Frames: []core.FrameInfo{}}
	analyzer := NewFuncAnalyzer(log)
	summary := analyzer.GetFuncSummary()
	if summary == nil {
		t.Fatal("GetFuncSummary should return non-nil for empty log")
	}
	if summary.TotalFunctions != 0 {
		t.Errorf("expected 0 TotalFunctions, got %d", summary.TotalFunctions)
	}
}

// FrameAnalyzer edge case tests
func TestFrameAnalyzer_GetFrameSummary_NilLog(t *testing.T) {
	analyzer := NewFrameAnalyzer(nil)
	summary := analyzer.GetFrameSummary()
	if summary != nil {
		t.Error("GetFrameSummary with nil log should return nil")
	}
}

func TestFrameAnalyzer_GetFrameSummary_EmptyFrames(t *testing.T) {
	log := &core.ParsedLog{Frames: []core.FrameInfo{}}
	analyzer := NewFrameAnalyzer(log)
	summary := analyzer.GetFrameSummary()
	if summary != nil {
		t.Error("GetFrameSummary with empty frames should return nil")
	}
}

// BufferAnalyzer helper function tests
func TestSplitParams(t *testing.T) {
	tests := []struct {
		params string
		want   []string
	}{
		{"0x8892 199", []string{"0x8892", "199"}},
		{"0x8892 8512 0x7fa1ba6970 GL_STATIC_DRAW", []string{"0x8892", "8512", "0x7fa1ba6970", "GL_STATIC_DRAW"}},
		{"0x8892 0 8512 0x7fa1ba6970", []string{"0x8892", "0", "8512", "0x7fa1ba6970"}},
		{"1, 2, 3", []string{"1", "2", "3"}},
		{"dpy=0x1c00, config=0x8b, share_list=0", []string{"dpy=0x1c00", "config=0x8b", "share_list=0"}},
		{"(nil)", []string{"(nil)"}},
		{"", nil},
		{"   ", nil},
	}
	for _, tt := range tests {
		got := splitParams(tt.params)
		if len(got) != len(tt.want) {
			t.Errorf("splitParams(%q) len = %d, want %d", tt.params, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitParams(%q)[%d] = %q, want %q", tt.params, i, got[i], tt.want[i])
			}
		}
	}
}

func TestTrimHexPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0x8892", "8892"},
		{"0X1A2B", "1A2B"},
		{"199", "199"},
		{"GL_ARRAY_BUFFER", "GL_ARRAY_BUFFER"},
		{"", ""},
	}
	for _, tt := range tests {
		got := trimHexPrefix(tt.input)
		if got != tt.want {
			t.Errorf("trimHexPrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseHexOrDec(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0x8892", 0x8892},
		{"0x8D40", 0x8D40},
		{"199", 199},
		{"8512", 8512},
		{"0x88E4", 0x88E4},
		{"0", 0},
		{"abc", 0},
		{"0xGHI", 0},
	}
	for _, tt := range tests {
		got := parseHexOrDec(tt.input)
		if got != tt.want {
			t.Errorf("parseHexOrDec(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeHexToName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0x8892", TargetArrayBuffer},
		{"GL_ARRAY_BUFFER", TargetArrayBuffer},
		{"0x8D40", TargetElementArrayBuffer},
		{"0x88B8", TargetPixelPackBuffer},
		{"0x88B9", TargetPixelUnpackBuffer},
		{"0x8B11", TargetUniformBuffer},
		{"0x8C8A", TargetTransformFeedback},
		{"0x8B8F", TargetCopyReadBuffer},
		{"0x8B8E", TargetCopyWriteBuffer},
		{"0x8F3F", TargetDrawIndirectBuffer},
		{"0x90D2", TargetShaderStorageBuffer},
		{"unknown_hex", "unknown_hex"},
	}
	for _, tt := range tests {
		got := normalizeHexToName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeHexToName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTargetToHex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{TargetArrayBuffer, "0x8892"},
		{TargetElementArrayBuffer, "0x8D40"},
		{TargetPixelPackBuffer, "0x88B8"},
		{TargetPixelUnpackBuffer, "0x88B9"},
		{TargetUniformBuffer, "0x8B11"},
		{TargetTransformFeedback, "0x8C8A"},
		{TargetCopyReadBuffer, "0x8B8F"},
		{TargetCopyWriteBuffer, "0x8B8E"},
		{TargetDrawIndirectBuffer, "0x8F3F"},
		{TargetShaderStorageBuffer, "0x90D2"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := targetToHex(tt.input)
		if got != tt.want {
			t.Errorf("targetToHex(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeUsage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GL_STREAM_DRAW", UsageStreamDraw},
		{"0x88E0", UsageStreamDraw},
		{"GL_STATIC_DRAW", UsageStaticDraw},
		{"0x88E4", UsageStaticDraw},
		{"GL_DYNAMIC_DRAW", UsageDynamicDraw},
		{"0x88E8", UsageDynamicDraw},
		{"GL_STREAM_READ", UsageStreamRead},
		{"GL_STATIC_READ", UsageStaticRead},
		{"GL_DYNAMIC_READ", UsageDynamicRead},
		{"GL_STREAM_COPY", UsageStreamCopy},
		{"GL_STATIC_COPY", UsageStaticCopy},
		{"GL_DYNAMIC_COPY", UsageDynamicCopy},
		{"UNKNOWN_USAGE", "UNKNOWN_USAGE"},
	}
	for _, tt := range tests {
		got := normalizeUsage(tt.input)
		if got != tt.want {
			t.Errorf("normalizeUsage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// BufferAnalyzer tests
func TestNewBufferAnalyzer_NilLog(t *testing.T) {
	ba := NewBufferAnalyzer(nil)
	if ba == nil {
		t.Fatal("NewBufferAnalyzer returned nil")
	}
	if ba.GetBufferCount() != 0 {
		t.Errorf("expected 0 buffers, got %d", ba.GetBufferCount())
	}
}

func TestNewBufferAnalyzer_EmptyLog(t *testing.T) {
	log := &core.ParsedLog{Frames: []core.FrameInfo{}}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Errorf("expected 0 buffers, got %d", ba.GetBufferCount())
	}
}

func TestBufferAnalyzer_ProcessGenBuffers(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "3", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 1 {
		t.Errorf("expected 1 buffer, got %d", ba.GetBufferCount())
	}
	bufs := ba.GetAllBuffers()
	if bufs[0].ID != 3 {
		t.Errorf("expected buffer ID 3, got %d", bufs[0].ID)
	}
}

func TestBufferAnalyzer_ProcessGenBuffers_EmptyParams(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Errorf("expected 0 buffers for empty params, got %d", ba.GetBufferCount())
	}
}

func TestBufferAnalyzer_ProcessBindBuffer(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 1},
					{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 2},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	bufs := ba.GetAllBuffers()
	if len(bufs) != 1 {
		t.Fatal("expected 1 buffer")
	}
	if bufs[0].Target != TargetArrayBuffer {
		t.Errorf("expected Target=GL_ARRAY_BUFFER, got %s", bufs[0].Target)
	}
}

func TestBufferAnalyzer_ProcessBindBuffer_EmptyParams(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("empty params should not create buffer")
	}
}

func TestBufferAnalyzer_ProcessBindBuffer_InsufficientParts(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("insufficient parts should not create buffer")
	}
}

func TestBufferAnalyzer_ProcessBufferData(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 1},
					{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 2},
					{APIName: "glBufferData", RawParams: "0x8892 8512 0x7fa1ba6970 GL_STATIC_DRAW", LineNum: 3},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	bufs := ba.GetAllBuffers()
	if len(bufs) != 1 {
		t.Fatal("expected 1 buffer")
	}
	if bufs[0].Size != 8512 {
		t.Errorf("expected Size=8512, got %d", bufs[0].Size)
	}
	if bufs[0].Usage != UsageStaticDraw {
		t.Errorf("expected Usage=GL_STATIC_DRAW, got %s", bufs[0].Usage)
	}
}

func TestBufferAnalyzer_ProcessBufferData_EmptyParams(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glBufferData", RawParams: "", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("empty params should be noop")
	}
}

func TestBufferAnalyzer_ProcessBufferData_InsufficientParts(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glBufferData", RawParams: "0x8892", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("insufficient parts should be noop")
	}
}

func TestBufferAnalyzer_ProcessBufferSubData(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 1},
					{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 2},
					{APIName: "glBufferSubData", RawParams: "0x8892 0 8512 0x7fa1ba6970", LineNum: 3},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	bufs := ba.GetAllBuffers()
	if len(bufs) != 1 {
		t.Fatal("expected 1 buffer")
	}
	if bufs[0].Size != 8512 {
		t.Errorf("expected Size=8512, got %d", bufs[0].Size)
	}
}

func TestBufferAnalyzer_ProcessDeleteBuffers(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 1},
					{APIName: "glDeleteBuffers", RawParams: "199", LineNum: 2},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Errorf("expected 0 buffers after delete, got %d", ba.GetBufferCount())
	}
}

func TestBufferAnalyzer_ProcessDeleteBuffers_EmptyParams(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glDeleteBuffers", RawParams: "", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("empty params should be noop")
	}
}

func TestBufferAnalyzer_GetBuffersByTarget(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	byTarget := ba.GetBuffersByTarget()
	if _, ok := byTarget[TargetArrayBuffer]; !ok {
		t.Error("expected ARRAY_BUFFER target group")
	}
	if len(byTarget[TargetArrayBuffer]) != 1 {
		t.Errorf("expected 1 buffer in ARRAY_BUFFER group, got %d", len(byTarget[TargetArrayBuffer]))
	}
}

func TestBufferAnalyzer_GetTotalSize(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				BufferCreations: []core.BufferInfo{
					{ID: 1, Target: TargetArrayBuffer, Size: 1024, Usage: UsageStaticDraw},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetTotalSize() != 1024 {
		t.Errorf("expected total size 1024, got %d", ba.GetTotalSize())
	}
}

func TestBufferAnalyzer_GetBufferSummary(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				BufferCreations: []core.BufferInfo{
					{ID: 1, Target: TargetArrayBuffer, Size: 1024, Usage: UsageStaticDraw},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	summary := ba.GetBufferSummary()
	if summary == nil {
		t.Fatal("GetBufferSummary returned nil")
	}
	if summary.TotalCount != 1 {
		t.Errorf("expected TotalCount=1, got %d", summary.TotalCount)
	}
	if summary.TotalSize != 1024 {
		t.Errorf("expected TotalSize=1024, got %d", summary.TotalSize)
	}
}

func TestBufferAnalyzer_WithBufferCreations(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				BufferCreations: []core.BufferInfo{
					{ID: 5, Target: TargetArrayBuffer, Size: 2048, Usage: UsageDynamicDraw},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 1 {
		t.Errorf("expected 1 buffer from BufferCreations, got %d", ba.GetBufferCount())
	}
	bufs := ba.GetAllBuffers()
	if bufs[0].Size != 2048 {
		t.Errorf("expected Size=2048, got %d", bufs[0].Size)
	}
}

func TestBufferAnalyzer_AddBuffer_Duplicate(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 1},
					{APIName: "glGenBuffers", RawParams: "199", LineNum: 2},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 1 {
		t.Errorf("expected 1 buffer (dedup), got %d", ba.GetBufferCount())
	}
}

func TestBufferAnalyzer_ProcessBufferSubData_EmptyParams(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glBufferSubData", RawParams: "", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("empty params should be noop")
	}
}

func TestBufferAnalyzer_ProcessBufferSubData_InsufficientParts(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glBufferSubData", RawParams: "0x8892 0", LineNum: 1},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	if ba.GetBufferCount() != 0 {
		t.Error("insufficient parts should be noop")
	}
}

func TestBufferAnalyzer_GetBufferSummary_Empty(t *testing.T) {
	log := &core.ParsedLog{Frames: []core.FrameInfo{}}
	ba := NewBufferAnalyzer(log)
	summary := ba.GetBufferSummary()
	if summary == nil {
		t.Fatal("GetBufferSummary should not return nil for empty log")
	}
	if summary.TotalCount != 0 {
		t.Errorf("expected TotalCount=0, got %d", summary.TotalCount)
	}
}

func TestBufferAnalyzer_MultipleBuffersDifferentTargets(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				BufferCreations: []core.BufferInfo{
					{ID: 1, Target: TargetArrayBuffer, Size: 100, Usage: UsageStaticDraw},
					{ID: 2, Target: TargetElementArrayBuffer, Size: 200, Usage: UsageDynamicDraw},
				},
			},
		},
	}
	ba := NewBufferAnalyzer(log)
	byTarget := ba.GetBuffersByTarget()
	if len(byTarget) != 2 {
		t.Errorf("expected 2 target groups, got %d", len(byTarget))
	}
}
