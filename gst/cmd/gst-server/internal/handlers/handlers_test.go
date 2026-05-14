package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gst/internal/core"
	"gst/internal/core/parser"
)

func TestValidateLogPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gst-handlers-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile, err := os.CreateTemp(tmpDir, "test-log-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	allowedPath, _ := filepath.Abs(tmpDir)
	os.Setenv("GST_LOG_DIR", allowedPath)
	defer os.Unsetenv("GST_LOG_DIR")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid file in allowed dir", tmpFile.Name(), false},
		{"empty path", "", true},
		{"dot dot outside allowed", "../etc/passwd", true},
		{"absolute system path", "/etc/passwd", true},
		{"relative outside allowed", filepath.Join(tmpDir, "../outside/file.log"), true},
		{"allowed dir itself", allowedPath, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLogPath(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for path %q, got nil", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}

func TestFrameSummaryJSONUsesSnakeCase(t *testing.T) {
	frame := core.FrameInfo{
		FrameNum:         7,
		StartLine:        10,
		EndLine:          20,
		TotalTimeUs:      16000,
		SwapBufferTimeUs: 9000,
		APITotalTimeUs:   6000,
		APICalls: []core.APILogEntry{
			{APIName: "glDrawArrays"},
		},
	}

	payload, err := json.Marshal(toFrameSummary(frame))
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, key := range []string{"frame_num", "start_line", "end_line", "total_time_us", "swap_buffer_time_us", "api_total_time_us", "api_count"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing snake_case key %q in %s", key, payload)
		}
	}
	if _, ok := got["FrameNum"]; ok {
		t.Fatalf("unexpected PascalCase key in %s", payload)
	}
}

func TestValidateLogPathDefaultAllowed(t *testing.T) {
	os.Unsetenv("GST_LOG_DIR")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"cwd file", filepath.Join(cwd, "test.log"), false},
		{"empty path", "", true},
		{"dot dot within proj", "../etc/passwd", false},
		{"/etc/passwd", "/etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLogPath(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for path %q, got nil", tt.path)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for path %q: %v", tt.path, err)
			}
		})
	}
}

func TestMimeType(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"json", "application/json"},
		{"csv", "text/csv"},
		{"txt", "text/plain"},
		{"xml", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, tt := range tests {
		got := mimeType(tt.format)
		if got != tt.want {
			t.Errorf("mimeType(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestMatchLine(t *testing.T) {
	tests := []struct {
		line     string
		keywords []string
		want     bool
	}{
		{"glBindBuffer: count=491", []string{"glBindBuffer"}, true},
		{"glBindBuffer: count=491", []string{"glBindBuffer", "count"}, true},
		{"glBindBuffer: count=491", []string{"glDrawElements"}, false},
		{"glBindBuffer: count=491", []string{"glBindBuffer", "nonexistent"}, false},
		{"MIXED CASE TEXT", []string{"mixed"}, true},
		{"", []string{"test"}, false},
		{"line", []string{}, true},
	}
	for _, tt := range tests {
		got := matchLine(tt.line, tt.keywords)
		if got != tt.want {
			t.Errorf("matchLine(%q, %v) = %v, want %v", tt.line, tt.keywords, got, tt.want)
		}
	}
}

func TestSearchTimeRange(t *testing.T) {
	h := NewHandler()
	h.current = &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum:    1,
				TotalTimeUs: 3000,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", LineNum: 1},
					{APIName: "glDrawArrays", LineNum: 2},
				},
			},
			{
				FrameNum:    2,
				TotalTimeUs: 800,
				APICalls: []core.APILogEntry{
					{APIName: "glClear", LineNum: 3},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/log/search/time?start_us=1000&end_us=5000", nil)
	rr := httptest.NewRecorder()
	h.SearchTimeRange(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var got TimeRangeSearchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if got.Total != 1 || len(got.Frames) != 1 || got.Frames[0].FrameNum != 1 {
		t.Fatalf("unexpected frames response: %+v", got)
	}
	if len(got.APICalls) != 2 {
		t.Fatalf("expected 2 api calls for non-indexed log, got %d", len(got.APICalls))
	}
}

func TestBuildLinesFromLog(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", Count: 10, TimeUs: 1000},
					{APIName: "glDrawElements", Count: 5, TimeUs: 40000},
				},
			},
		},
	}
	lines := buildLinesFromLog(log)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "glBindBuffer: count=10, time=1000" {
		t.Errorf("line 0: %q", lines[0])
	}
	if lines[1] != "glDrawElements: count=5, time=40000" {
		t.Errorf("line 1: %q", lines[1])
	}
}

func TestBuildLinesFromLog_Empty(t *testing.T) {
	log := &core.ParsedLog{}
	lines := buildLinesFromLog(log)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestTraceEndpoints(t *testing.T) {
	h := NewHandler()
	h.current = &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum:  0,
				StartLine: 1,
				EndLine:   6,
				Shaders: []*core.ShaderInfo{
					{ID: 16, Source: "void main(){}"},
				},
				APICalls: []core.APILogEntry{
					{APIName: "glCreateShader", RawParams: "0x8B31", ReturnValue: "16", LineNum: 1},
					{APIName: "glShaderSource", RawParams: "16 1 0xffff (nil)", LineNum: 2},
					{APIName: "glCompileShader", RawParams: "16", LineNum: 3},
					{APIName: "glCreateProgram", ReturnValue: "18", LineNum: 4},
					{APIName: "glAttachShader", RawParams: "18 16", LineNum: 5},
					{APIName: "glUseProgram", RawParams: "18", LineNum: 6},
					{APIName: "glDrawArrays", RawParams: "0x0004 0 3", LineNum: 7},
				},
			},
		},
	}

	t.Run("program list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/log/trace/programs", nil)
		rec := httptest.NewRecorder()
		h.GetTracePrograms(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		var body struct {
			Programs []core.ProgramInfo `json:"programs"`
			Total    int                `json:"total"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if body.Total != 1 || body.Programs[0].ID != 18 {
			t.Fatalf("unexpected program list: %+v", body)
		}
	})

	t.Run("frame programs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/log/frames/0/programs", nil)
		rec := httptest.NewRecorder()
		h.GetFramePrograms(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		var body core.FrameProgramInsight
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if body.TotalDrawCalls != 1 || len(body.Programs) != 1 || body.Programs[0].ProgramID != 18 {
			t.Fatalf("unexpected frame insight: %+v", body)
		}
	})

	t.Run("draw calls", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/log/frames/0/drawcalls?page=1&page_size=10&program=18", nil)
		rec := httptest.NewRecorder()
		h.GetFrameDrawCalls(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		var body core.FrameDrawCallPage
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if body.Total != 1 || len(body.DrawCalls) != 1 || body.DrawCalls[0].ProgramID != 18 {
			t.Fatalf("unexpected draw calls: %+v", body)
		}
	})
}

func TestFrameRawLinesAndDownloadIndexed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hybrid.log")
	content := strings.Join([]string{
		"[    1] (gc=0x1, tid=0x1): glUseProgram 7",
		"[    2] (gc=0x1, tid=0x1): glDrawElements 0x0004 54 0x1403 0x2d0",
		"[    3] glXSwapBuffers: dpy = 0x1, drawable = 1",
		"[    4] swapBuffers: 100 us",
		"[    5] 184 frame cost 16ms",
		"[    6] glDrawElements: count=1, time=900 us",
		"[    7] (gc=0x1, tid=0x1): glDrawArrays 0x0004 0 3",
		"[    8] glXSwapBuffers: dpy = 0x1, drawable = 1",
		"[    9] 185 frame cost 20ms",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parsed, err := parser.ParseIndexedRawTraceFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler()
	h.current = parsed
	h.rawLogPath = path

	req := httptest.NewRequest(http.MethodGet, "/api/log/frames/184/raw-lines?page=1&page_size=20", nil)
	rec := httptest.NewRecorder()
	h.GetFrameRawLines(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("raw-lines status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body FrameRawLinesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if body.Total != 6 || len(body.Lines) != 6 {
		t.Fatalf("raw-lines total=%d len=%d body=%+v", body.Total, len(body.Lines), body)
	}
	if body.Lines[1] != "(gc=0x1, tid=0x1): glDrawElements 0x0004 54 0x1403 0x2d0" {
		t.Fatalf("line prefix not stripped or raw text changed: %#v", body.Lines)
	}
	if strings.Contains(body.Lines[0], "[    1]") {
		t.Fatalf("line number prefix leaked: %#v", body.Lines)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/log/frames/184/download", nil)
	rec = httptest.NewRecorder()
	h.DownloadFrameRawLog(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d body=%s", rec.Code, rec.Body.String())
	}
	download := rec.Body.String()
	if !strings.Contains(download, "[    6] glDrawElements: count=1, time=900 us") {
		t.Fatalf("download missing profile tail: %q", download)
	}
	if strings.Contains(download, "[    7] (gc=0x1, tid=0x1): glDrawArrays") {
		t.Fatalf("download leaked next frame: %q", download)
	}
}

func TestAnalyzeShadersFallsBackToAPIStats(t *testing.T) {
	h := NewHandler()
	h.current = &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 9,
				APISummary: map[string]*core.APISummary{
					"glUseProgram":        {APIName: "glUseProgram", Count: 3, TimeUs: 30},
					"glProgramUniform4fv": {APIName: "glProgramUniform4fv", Count: 7, TimeUs: 140},
					"glDrawElements":      {APIName: "glDrawElements", Count: 5, TimeUs: 500},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/log/analyze/shaders", nil)
	rec := httptest.NewRecorder()
	h.AnalyzeShaders(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body ShadersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if body.Total != 2 {
		t.Fatalf("shader stat total = %d, want 2: %+v", body.Total, body)
	}
	foundUseProgram := false
	for _, shader := range body.Shaders {
		if shader.APIName == "glUseProgram" {
			foundUseProgram = true
			if shader.Kind != "api_stat" || shader.Count != 3 || shader.TimeUs != 30 {
				t.Fatalf("unexpected glUseProgram shader stat: %+v", shader)
			}
		}
		if shader.APIName == "glDrawElements" {
			t.Fatalf("draw API should not be included in shader stats: %+v", shader)
		}
	}
	if !foundUseProgram {
		t.Fatalf("missing glUseProgram stat: %+v", body.Shaders)
	}
}

func TestFormatRawCallLineProfileSummary(t *testing.T) {
	line := formatRawCallLine(core.APILogEntry{
		APIName: "glTexImage2D",
		Count:   1,
		TimeUs:  899,
	})
	want := "glTexImage2D: count=1, time=899 us"
	if line != want {
		t.Fatalf("formatRawCallLine = %q, want %q", line, want)
	}
}

func TestToParseResult(t *testing.T) {
	p := &core.ParsedLog{
		FPS: 60.0,
		Frames: []core.FrameInfo{
			{TotalTimeUs: 10000},
			{TotalTimeUs: 30000},
			{TotalTimeUs: 20000},
		},
		TotalTimeUs: 60000,
	}
	result := toParseResult(p)
	if result.FrameCount != 3 {
		t.Errorf("FrameCount = %d, want 3", result.FrameCount)
	}
	if result.Format != "profile" {
		t.Errorf("Format = %s, want profile", result.Format)
	}
	if result.FPS != 60.0 {
		t.Errorf("FPS = %f, want 60.0", result.FPS)
	}
	if result.MaxFrameTime != 30.0 {
		t.Errorf("MaxFrameTime = %f, want 30.0", result.MaxFrameTime)
	}
}

func TestToParseResult_NoFPS(t *testing.T) {
	p := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{TotalTimeUs: 10000},
		},
	}
	result := toParseResult(p)
	if result.Format != "rawtrace" {
		t.Errorf("Format = %s, want rawtrace", result.Format)
	}
}

func TestToParseResult_Empty(t *testing.T) {
	p := &core.ParsedLog{}
	result := toParseResult(p)
	if result.FrameCount != 0 {
		t.Errorf("FrameCount = %d, want 0", result.FrameCount)
	}
	if result.MaxFrameTime != 0 {
		t.Errorf("MaxFrameTime = %f, want 0", result.MaxFrameTime)
	}
}

func TestHealth(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.Health(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Health status = %d, want 200", w.Code)
	}
}

func TestServeUI_Root(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeUI(w, req)
	if w.Code != http.StatusNotFound {
		t.Logf("ServeUI status: %d (expected 404 if web/index.html missing)", w.Code)
	}
}

func TestServeUI_NotFound(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	handler.ServeUI(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("ServeUI for /nonexistent = %d, want 404", w.Code)
	}
}
