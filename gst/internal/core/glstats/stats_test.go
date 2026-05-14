package glstats

import (
	"testing"

	"gst/internal/core"
)

func TestClassifyImportantOpenGLFunctions(t *testing.T) {
	tests := []struct {
		api      string
		category string
		key      string
	}{
		{"glDrawElements", CategoryDraw, "glDrawElements"},
		{"glDrawElementsInstancedBaseVertex", CategoryDraw, "glDrawElements"},
		{"glDrawArrays", CategoryDraw, "glDrawArrays"},
		{"glBufferSubData", CategoryBuffer, "bufferTransfer"},
		{"glTexSubImage2D", CategoryTexture, "textureTransfer"},
		{"glReadPixels", CategorySyncQuery, "readback"},
		{"glCompileShader", CategoryShader, "shaderCompile"},
		{"glLinkProgram", CategoryShader, "programLink"},
		{"glUseProgram", CategoryShader, "programUse"},
		{"glVertexAttribPointer", CategoryVertexInput, ""},
		{"glBindFramebuffer", CategoryFramebuffer, ""},
		{"glFenceSync", CategorySyncQuery, "sync"},
	}

	for _, tt := range tests {
		got := Classify(tt.api)
		if got.Category != tt.category {
			t.Fatalf("Classify(%s).Category = %s, want %s", tt.api, got.Category, tt.category)
		}
		if got.Key != tt.key {
			t.Fatalf("Classify(%s).Key = %s, want %s", tt.api, got.Key, tt.key)
		}
	}
}

func TestAnalyzeFramePrefersProfileSummary(t *testing.T) {
	frame := core.FrameInfo{
		FrameNum:         7,
		TotalTimeUs:      16000,
		SwapBufferTimeUs: 3000,
		APITotalTimeUs:   13000,
		APICallCount:     20,
		DrawCallCount:    8,
		HasTiming:        true,
		TimingSource:     "frame_cost",
		APICalls: []core.APILogEntry{
			{APIName: "glDrawArrays", Count: 1},
			{APIName: "glDrawElements", Count: 1},
		},
		APISummary: map[string]*core.APISummary{
			"glDrawElements":  {APIName: "glDrawElements", Count: 5, TimeUs: 900},
			"glDrawArrays":    {APIName: "glDrawArrays", Count: 3, TimeUs: 300},
			"glBufferSubData": {APIName: "glBufferSubData", Count: 4, TimeUs: 1000},
		},
	}

	stats := AnalyzeFrame(frame)
	if stats.StatsSource != "profile_summary" {
		t.Fatalf("StatsSource = %s, want profile_summary", stats.StatsSource)
	}
	if stats.APICallCount != 12 {
		t.Fatalf("APICallCount = %d, want 12", stats.APICallCount)
	}
	if stats.RawAPICallCount != 20 {
		t.Fatalf("RawAPICallCount = %d, want 20", stats.RawAPICallCount)
	}
	if stats.DrawCallCount != 8 {
		t.Fatalf("DrawCallCount = %d, want 8", stats.DrawCallCount)
	}
	if len(stats.KeyAPIs) == 0 {
		t.Fatal("expected key API stats")
	}
	foundDrawElements := false
	for _, api := range stats.KeyAPIs {
		if api.Key == "glDrawElements" && api.Count == 5 && api.TimeUs == 900 {
			foundDrawElements = true
		}
	}
	if !foundDrawElements {
		t.Fatalf("missing glDrawElements key API: %#v", stats.KeyAPIs)
	}
}
