package analyzer

import (
	"testing"

	"gst/internal/core"
)

func TestTraceInspectorAnalyzer_SourceProgramFrameInsight(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum:  0,
				StartLine: 1,
				EndLine:   14,
				Shaders: []*core.ShaderInfo{
					{ID: 16, Source: "void main(){ gl_Position = vec4(0.0); }"},
					{ID: 17, Source: "out vec4 color; void main(){ color = vec4(1.0); }"},
				},
				APICalls: []core.APILogEntry{
					{APIName: "glCreateShader", RawParams: "0x8B31", ReturnValue: "16", LineNum: 1, GCAddr: "0xgc"},
					{APIName: "glShaderSource", RawParams: "16 1 0xffff (nil)", LineNum: 2, GCAddr: "0xgc"},
					{APIName: "glCompileShader", RawParams: "16", LineNum: 3, GCAddr: "0xgc"},
					{APIName: "glCreateShader", RawParams: "0x8B30", ReturnValue: "17", LineNum: 4, GCAddr: "0xgc"},
					{APIName: "glShaderSource", RawParams: "17 1 0xffff (nil)", LineNum: 5, GCAddr: "0xgc"},
					{APIName: "glCompileShader", RawParams: "17", LineNum: 6, GCAddr: "0xgc"},
					{APIName: "glCreateProgram", ReturnValue: "18", LineNum: 7, GCAddr: "0xgc"},
					{APIName: "glAttachShader", RawParams: "18 16", LineNum: 8, GCAddr: "0xgc"},
					{APIName: "glAttachShader", RawParams: "18 17", LineNum: 9, GCAddr: "0xgc"},
					{APIName: "glLinkProgram", RawParams: "18", LineNum: 10, GCAddr: "0xgc"},
					{APIName: "glUseProgram", RawParams: "18", LineNum: 11, GCAddr: "0xgc"},
					{APIName: "glBindBuffer", RawParams: "0x8892 7", LineNum: 12, GCAddr: "0xgc"},
					{APIName: "glBindBuffer", RawParams: "0x8893 8", LineNum: 13, GCAddr: "0xgc"},
					{APIName: "glDrawElements", RawParams: "0x0004 6 0x1403 0x0", LineNum: 14, GCAddr: "0xgc"},
				},
			},
		},
	}

	result := NewTraceInspectorAnalyzer(log).Analyze()
	if result == nil {
		t.Fatal("Analyze returned nil")
	}

	program := result.ProgramMap[18]
	if program == nil {
		t.Fatal("program 18 missing")
	}
	if program.SourceType != "source" {
		t.Fatalf("SourceType = %q, want source", program.SourceType)
	}
	if program.Confidence != "high" {
		t.Fatalf("Confidence = %q, want high", program.Confidence)
	}
	if program.DrawCallCount != 1 {
		t.Fatalf("DrawCallCount = %d, want 1", program.DrawCallCount)
	}
	if len(program.Shaders) != 2 || !program.Shaders[0].SourceAvailable || !program.Shaders[1].SourceAvailable {
		t.Fatalf("expected two shaders with source, got %+v", program.Shaders)
	}

	frame := result.FrameInsights[0]
	if frame == nil {
		t.Fatal("frame insight missing")
	}
	if frame.TotalDrawCalls != 1 {
		t.Fatalf("TotalDrawCalls = %d, want 1", frame.TotalDrawCalls)
	}
	if len(frame.Programs) != 1 || frame.Programs[0].ProgramID != 18 {
		t.Fatalf("unexpected program usages: %+v", frame.Programs)
	}
	drawCalls, ok := NewTraceInspectorAnalyzer(log).AnalyzeFrameDrawCalls(0, 0, result)
	if !ok {
		t.Fatal("frame draw calls not found")
	}
	if len(drawCalls) != 1 {
		t.Fatalf("draw call insight count = %d, want 1", len(drawCalls))
	}
	draw := drawCalls[0]
	if draw.ProgramID != 18 || draw.ArrayBuffer != 7 || draw.ElementBuffer != 8 {
		t.Fatalf("unexpected draw insight: %+v", draw)
	}
}

func TestTraceInspectorAnalyzer_BinaryAndUnknownConfidence(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 0,
				APICalls: []core.APILogEntry{
					{APIName: "glCreateProgram", ReturnValue: "4", LineNum: 1},
					{APIName: "glProgramBinary", RawParams: "4 0x8FC5 0xffff 5380", LineNum: 2},
					{APIName: "glUseProgram", RawParams: "4", LineNum: 3},
					{APIName: "glDrawArrays", RawParams: "0x0004 0 3", LineNum: 4},
					{APIName: "glUseProgram", RawParams: "99", LineNum: 5},
					{APIName: "glDrawArrays", RawParams: "0x0004 0 3", LineNum: 6},
				},
			},
		},
	}

	result := NewTraceInspectorAnalyzer(log).Analyze()
	if got := result.ProgramMap[4].SourceType; got != "program_binary" {
		t.Fatalf("program 4 SourceType = %q, want program_binary", got)
	}
	if got := result.ProgramMap[4].Confidence; got != "medium" {
		t.Fatalf("program 4 Confidence = %q, want medium", got)
	}
	if got := result.ProgramMap[99].Confidence; got != "low" {
		t.Fatalf("program 99 Confidence = %q, want low", got)
	}
	drawCalls, ok := NewTraceInspectorAnalyzer(log).AnalyzeFrameDrawCalls(0, 0, result)
	if !ok {
		t.Fatal("frame draw calls not found")
	}
	if len(drawCalls) != 2 {
		t.Fatalf("draw call count = %d, want 2", len(drawCalls))
	}
}
