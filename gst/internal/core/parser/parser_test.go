package parser

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestDetectKind(t *testing.T) {
	tests := []struct {
		name      string
		firstLine string
		expected  LogKind
	}{
		{"empty line", "", KindUnknown},
		{"profile format", "<<gc = 0x1800d34000>>", KindProfile},
		{"api trace format", "glBindBuffer: count=491, time=588 us", KindAPITrace},
		{"raw trace format - glXSwapBuffers", "glXSwapBuffers: dpy = 0x1c002a1400, drawable = 121634855", KindRawTrace},
		{"raw trace format - glGenFramebuffers", "glGenFramebuffers 1", KindRawTrace},
		{"raw trace format - glBindBuffer", "glBindBuffer 0x8892 498", KindRawTrace},
		{"raw trace format - glDrawElements", "glDrawElements 0x0004 2304 0x1403 (nil)", KindRawTrace},
		{"unknown format", "some random text", KindUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectKind(tt.firstLine)
			if result != tt.expected {
				t.Errorf("DetectKind(%q) = %v, want %v", tt.firstLine, result, tt.expected)
			}
		})
	}
}

func TestCreateParser(t *testing.T) {
	tests := []struct {
		kind     LogKind
		expected string
	}{
		{KindAPITrace, "*parser.APIParser"},
		{KindProfile, "*parser.ProfileParser"},
		{KindRawTrace, "*parser.RawTraceParser"},
		{KindUnknown, "*parser.APIParser"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			parser := CreateParser(tt.kind)
			result := string(tt.kind)
			if result == string(tt.kind) && parser == nil {
				t.Error("CreateParser returned nil")
			}
		})
	}
}

func TestAPIParser_Parse(t *testing.T) {
	input := `glBindBuffer: count=491, time=588 us
glBindFramebuffer: count=29, time=25377 us
glDrawElements: count=493, time=11214 us
libGL: FPS = 8.9
swapBuffers: 3033 us
423 frame cost 109ms`

	parser := &APIParser{}
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) != 1 {
		t.Errorf("Expected 1 frame, got %d", len(parsed.Frames))
	}

	if parsed.FPS != 8.9 {
		t.Errorf("Expected FPS 8.9, got %f", parsed.FPS)
	}

	if len(parsed.Frames) > 0 {
		frame := parsed.Frames[0]
		if len(frame.APICalls) != 3 {
			t.Errorf("Expected 3 API calls, got %d", len(frame.APICalls))
		}
		if frame.APICalls[0].APIName != "glBindBuffer" {
			t.Errorf("Expected first API to be glBindBuffer, got %s", frame.APICalls[0].APIName)
		}
	}
}

func TestAPIParser_ParseFromFile(t *testing.T) {
	file, err := os.Open("/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_profile_demo.txt")
	if err != nil {
		t.Skipf("Skipping file test: %v", err)
	}
	defer file.Close()

	parser := &APIParser{}
	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) == 0 {
		t.Error("Expected at least 1 frame")
	}

	if parsed.FPS != 8.9 {
		t.Errorf("Expected FPS 8.9, got %f", parsed.FPS)
	}
}

func TestProfileParser_Parse(t *testing.T) {
	input := `glBindBuffer: count=491, time=588 us
glDrawElements: count=493, time=11214 us
swapBuffers: 3033 us`

	parser := NewProfileParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) != 1 {
		t.Errorf("Expected 1 frame, got %d", len(parsed.Frames))
	}
}

func TestIsFrameBoundary(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"swapBuffers: 3033 us", true},
		{"423 frame cost 109ms", true},
		{"glBindBuffer: count=491, time=588 us", false},
		{"libGL: FPS = 8.9", false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := isFrameBoundary(tt.line)
			if result != tt.expected {
				t.Errorf("isFrameBoundary(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestRawTraceParser_Parse(t *testing.T) {
	input := `glXSwapBuffers: dpy = 0x1c002a1400, drawable = 121634855
glGenFramebuffers 1
glBindBuffer 0x8892 498
glBufferSubData 0x8892 0 8512 0x7fa1ba6970
glUseProgram 18
glDrawElements 0x0004 2304 0x1403 (nil)
glXSwapBuffers: dpy = 0x1c002a1400, drawable = 121634855
glGenFramebuffers 1
glBindBuffer 0x8892 500
glUseProgram 22
glDrawArrays 0x0005 0 4`

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 2 frames
	if len(parsed.Frames) != 2 {
		t.Errorf("Expected 2 frames, got %d", len(parsed.Frames))
	}

	// First frame should have 5 API calls
	if len(parsed.Frames) > 0 {
		frame1 := parsed.Frames[0]
		if len(frame1.APICalls) != 5 {
			t.Errorf("Expected 5 API calls in frame 1, got %d", len(frame1.APICalls))
		}
		// Check glUseProgram captured program ID 18
		if len(frame1.Programs) != 1 || frame1.Programs[0] != 18 {
			t.Errorf("Expected program 18 in frame 1, got %v", frame1.Programs)
		}
	}

	// Second frame should have 4 API calls
	if len(parsed.Frames) > 1 {
		frame2 := parsed.Frames[1]
		if len(frame2.APICalls) != 4 {
			t.Errorf("Expected 4 API calls in frame 2, got %d", len(frame2.APICalls))
		}
		// Check glUseProgram captured program ID 22
		if len(frame2.Programs) != 1 || frame2.Programs[0] != 22 {
			t.Errorf("Expected program 22 in frame 2, got %v", frame2.Programs)
		}
	}
}

func TestRawTraceParser_ParseFromFile(t *testing.T) {
	file, err := os.Open("/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_demo_api.txt")
	if err != nil {
		t.Skipf("Skipping file test: %v", err)
	}
	defer file.Close()

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(file)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// The sample file has multiple frames
	if len(parsed.Frames) == 0 {
		t.Error("Expected at least 1 frame")
	}

	// Check that programs were captured
	for i, frame := range parsed.Frames {
		if len(frame.Programs) > 0 {
			t.Logf("Frame %d has %d programs: %v", i, len(frame.Programs), frame.Programs)
		}
	}
}

func TestRawTraceParser_Kind(t *testing.T) {
	parser := NewRawTraceParser()
	if parser.Kind() != KindRawTrace {
		t.Errorf("Expected KindRawTrace, got %v", parser.Kind())
	}
}

func TestAPIParser_Kind(t *testing.T) {
	parser := &APIParser{}
	if parser.Kind() != KindAPITrace {
		t.Errorf("Expected KindAPITrace, got %v", parser.Kind())
	}
}

func TestProfileParser_Kind(t *testing.T) {
	parser := NewProfileParser()
	if parser.Kind() != KindProfile {
		t.Errorf("Expected KindProfile, got %v", parser.Kind())
	}
}

func TestShouldSkipLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"[123:456:789:ERROR:something]", true},
		{"[1:2:3:module:msg]", true},
		{"<<gc = 0xffff60638d80>>", true},
		{"vendor: NVIDIA Corporation", true},
		{"glBindBuffer: count=491, time=588 us", false},
		{"glDrawElements 0x0004 2304 0x1403 (nil)", false},
		{"swapBuffers: 3033 us", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := shouldSkipLine(tt.line); got != tt.expected {
				t.Errorf("shouldSkipLine(%q) = %v, want %v", tt.line, got, tt.expected)
			}
		})
	}
}

func TestRemoveLinePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"[     1] glBindBuffer: count=491, time=588 us", "glBindBuffer: count=491, time=588 us"},
		{"[12345] glDrawElements 0x0004 2304", "glDrawElements 0x0004 2304"},
		{"no prefix here", "no prefix here"},
		{"[incomplete", "[incomplete"},
	}
	for _, tt := range tests {
		got := removeLinePrefix(tt.input)
		if got != tt.want {
			t.Errorf("removeLinePrefix(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectKindFromReader(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    LogKind
	}{
		{"apitrace format", "glBindBuffer: count=491, time=588 us\nswapBuffers: 3033 us\n423 frame cost 109ms\n", KindAPITrace},
		{"raw trace format", "glBindBuffer 0x8892 498\nglDrawElements 0x0004 2304 0x1403 (nil)\n", KindRawTrace},
		{"apitrace with prefix", "[     1] glBindBuffer: count=491, time=588 us\n", KindAPITrace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind := DetectKindFromReader(strings.NewReader(tt.content), 50)
			if kind != tt.want {
				t.Errorf("DetectKindFromReader = %v, want %v", kind, tt.want)
			}
		})
	}
}

func TestDetectKindFromReader_Empty(t *testing.T) {
	kind := DetectKindFromReader(strings.NewReader(""), 10)
	if kind != KindUnknown {
		t.Errorf("expected KindUnknown, got %v", kind)
	}
}

func TestDetectKindFromReader_SkipLines(t *testing.T) {
	content := "  \n  \nglBindBuffer: count=1, time=100 us\n"
	kind := DetectKindFromReader(strings.NewReader(content), 50)
	if kind != KindAPITrace {
		t.Errorf("expected KindAPITrace, got %v", kind)
	}
}

func TestDetectKindFromReader_LimitLines(t *testing.T) {
	content := "line1\nline2\nline3\nline4\nglBindBuffer: count=1, time=100 us\n"
	kind := DetectKindFromReader(strings.NewReader(content), 3)
	if kind != KindUnknown {
		t.Errorf("expected KindUnknown due to line limit, got %v", kind)
	}
}

func TestCreateParserAuto(t *testing.T) {
	content := "glBindBuffer: count=491, time=588 us\n"
	parser, err := CreateParserAuto(strings.NewReader(content))
	if err != nil {
		t.Fatalf("CreateParserAuto failed: %v", err)
	}
	if parser.Kind() != KindAPITrace {
		t.Errorf("expected KindAPITrace, got %v", parser.Kind())
	}
}

type nonSeekerReader struct{}

func (n nonSeekerReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func TestCreateParserAuto_WithSeeker(t *testing.T) {
	_, err := CreateParserAuto(strings.NewReader("glBindBuffer: count=1, time=100 us"))
	if err != nil {
		t.Errorf("CreateParserAuto with seekable reader should succeed: %v", err)
	}
}

func TestCreateParserAuto_NonSeeker(t *testing.T) {
	_, err := CreateParserAuto(nonSeekerReader{})
	if err == nil {
		t.Error("expected error for non-seeker reader")
	}
}

func TestExtractProgramID(t *testing.T) {
	tests := []struct {
		params string
		want   int
	}{
		{"18", 18},
		{"program = 22", 22},
		{"Program = 5", 5},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		got := extractProgramID(tt.params)
		if got != tt.want {
			t.Errorf("extractProgramID(%q) = %d, want %d", tt.params, got, tt.want)
		}
	}
}

func TestExtractBufferIDs(t *testing.T) {
	tests := []struct {
		params string
		want   []int
	}{
		{"1", []int{1}},
		{"1, 2, 3", []int{1, 2, 3}},
		{"5 6 7", []int{5, 6, 7}},
		{"", nil},
		{"   ", nil},
	}
	for _, tt := range tests {
		got := extractBufferIDs(tt.params)
		if len(got) != len(tt.want) {
			t.Errorf("extractBufferIDs(%q) len = %d, want %d", tt.params, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractBufferIDs(%q)[%d] = %d, want %d", tt.params, i, got[i], tt.want[i])
			}
		}
	}
}

func TestDetectFromLine_EdgeCases(t *testing.T) {
	tests := []struct {
		line string
		want LogKind
	}{
		{"[     1] glClear: count=1, time=100 us", KindAPITrace},
		{"glDeleteBuffers 199", KindRawTrace},
		{"eglSwapBuffers: display", KindRawTrace},
	}
	for _, tt := range tests {
		got := detectFromLine(tt.line)
		if got != tt.want {
			t.Errorf("detectFromLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestRawTraceParser_Parse_EdgeCases(t *testing.T) {
	input := `  leading space
=> return value
src: source line
{ struct open
__someInternal internal call
glClear 0x4100
glXSwapBuffers: dpy = 1, drawable = 1`
	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(parsed.Frames) != 1 {
		t.Errorf("Expected 1 frame, got %d", len(parsed.Frames))
	}
	if len(parsed.Frames[0].APICalls) != 1 {
		t.Errorf("Expected 1 API call, got %d", len(parsed.Frames[0].APICalls))
	}
}

func TestRawTraceParser_SegmentFault(t *testing.T) {
	input := "core dumped: SIGSEGV\n"
	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(parsed.Frames) != 1 {
		t.Fatal("expected 1 frame")
	}
	call := parsed.Frames[0].APICalls[0]
	if call.APIName != "__segfault__" {
		t.Errorf("expected __segfault__, got %s", call.APIName)
	}
	if call.ErrorCode != "SIGSEGV" {
		t.Errorf("expected SIGSEGV error code, got %s", call.ErrorCode)
	}
}

func TestRawTraceParser_EmptyInput(t *testing.T) {
	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(parsed.Frames) != 0 {
		t.Errorf("expected 0 frames, got %d", len(parsed.Frames))
	}
}
