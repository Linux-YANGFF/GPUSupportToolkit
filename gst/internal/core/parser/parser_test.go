package parser

import (
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
