package bug

import (
	"strings"
	"testing"

	"gst/internal/core"
)

func TestDriverErrorDetector_DetectsGlSetErrorWithTrigger(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindTexture", RawParams: "0x0de1 42", LineNum: 1, GCAddr: "0xffff60638d80"},
					{APIName: "glDrawElements", RawParams: "0x0004 2304 0x1403", LineNum: 2, GCAddr: "0xffff60638d80"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0502", LineNum: 3, GCAddr: "0xffff60638d80"},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.Category != "driver_error" {
		t.Errorf("expected category driver_error, got %s", f.Category)
	}
	if f.Severity != core.SeverityCritical {
		t.Errorf("expected severity critical, got %s", f.Severity)
	}
	if !strings.Contains(f.Description, "GL_INVALID_OPERATION") {
		t.Errorf("expected GL_INVALID_OPERATION in description, got %s", f.Description)
	}
	if !strings.Contains(f.Evidence, "triggering call: glDrawElements") {
		t.Errorf("expected glDrawElements as triggering call, got %s", f.Evidence)
	}
	if !strings.Contains(f.Evidence, "triggering call: glDrawElements at line 2") {
		t.Errorf("expected trigger at line 2, got %s", f.Evidence)
	}
}

func TestDriverErrorDetector_ErrorCodeMapping(t *testing.T) {
	detector := &DriverErrorDetector{}

	tests := []struct {
		code     string
		expected string
	}{
		{"0x0500", "GL_INVALID_ENUM"},
		{"0x0501", "GL_INVALID_VALUE"},
		{"0x0502", "GL_INVALID_OPERATION"},
		{"0x0503", "GL_STACK_OVERFLOW"},
		{"0x0504", "GL_STACK_UNDERFLOW"},
		{"0x0505", "GL_OUT_OF_MEMORY"},
		{"0x0506", "GL_INVALID_FRAMEBUFFER_OPERATION"},
		{"0x500", "GL_INVALID_ENUM"},
		{"0x501", "GL_INVALID_VALUE"},
		{"0x9999", "GL_UNKNOWN_ERROR"},
	}

	for _, tt := range tests {
		call := core.APILogEntry{
			APIName:   "__glSetError",
			IsError:   true,
			ErrorCode: tt.code,
			LineNum:   10,
		}
		log := &core.ParsedLog{
			Frames: []core.FrameInfo{
				{
					APICalls: []core.APILogEntry{
						{APIName: "glClear", LineNum: 9},
						call,
					},
				},
			},
		}
		findings := detector.Diagnose(log)
		if len(findings) != 1 {
			t.Fatalf("code=%s: expected 1 finding, got %d", tt.code, len(findings))
		}
		if !strings.Contains(findings[0].Description, tt.expected) {
			t.Errorf("code=%s: expected %s in description, got %s", tt.code, tt.expected, findings[0].Description)
		}
	}
}

func TestDriverErrorDetector_MissingGlGetError(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glClear", LineNum: 1, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 2, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if !strings.Contains(f.RootCauseChain[1], "does not appear to call glGetError") {
		t.Errorf("expected missing glGetError in root cause, got %v", f.RootCauseChain)
	}
	if !strings.Contains(f.FixSuggestion, "glGetError") {
		t.Errorf("expected glGetError in fix suggestion, got %s", f.FixSuggestion)
	}
}

func TestDriverErrorDetector_WithGlGetErrorPresent(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glGetError", LineNum: 1, GCAddr: "0xaaa"},
					{APIName: "glClear", LineNum: 2, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0501", LineNum: 3, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	for _, rc := range f.RootCauseChain {
		if strings.Contains(rc, "does not appear to call glGetError") {
			t.Error("should not report missing glGetError when present")
		}
	}
}

func TestDriverErrorDetector_BackwardTraceSkipsOtherErrors(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glUseProgram", RawParams: "18", LineNum: 10, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 11, GCAddr: "0xaaa"},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", LineNum: 12, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0501", LineNum: 13, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	if !strings.Contains(findings[0].Evidence, "triggering call: glUseProgram at line 10") {
		t.Errorf("expected first error triggered by glUseProgram, got %s", findings[0].Evidence)
	}
	if !strings.Contains(findings[1].Evidence, "triggering call: glDrawArrays at line 12") {
		t.Errorf("expected second error triggered by glDrawArrays, got %s", findings[1].Evidence)
	}
}

func TestDriverErrorDetector_ContextIsolation(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindTexture", LineNum: 1, GCAddr: "0xAAA"},
					{APIName: "glClear", LineNum: 2, GCAddr: "0xBBB"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 3, GCAddr: "0xAAA"},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Evidence, "triggering call: glBindTexture at line 1") {
		t.Errorf("expected context-aware triggering call, got %s", findings[0].Evidence)
	}
}

func TestDriverErrorDetector_NoTriggerCallFound(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0503", LineNum: 1},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if !strings.Contains(findings[0].Evidence, "no triggering call identified") {
		t.Errorf("expected no trigger identified, got %s", findings[0].Evidence)
	}
}

func TestDriverErrorDetector_MultipleErrorsSameContext(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glUseProgram", LineNum: 10, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 11, GCAddr: "0xaaa"},
					{APIName: "glUseProgram", LineNum: 12, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0501", LineNum: 13, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.Category != "driver_error" {
			t.Errorf("expected category driver_error, got %s", f.Category)
		}
		if f.Severity != core.SeverityCritical {
			t.Errorf("expected severity critical, got %s", f.Severity)
		}
	}
}

func TestDriverErrorDetector_EmptyLog(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty log, got %d", len(findings))
	}
}

func TestDriverErrorDetector_NoErrors(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glClear", LineNum: 1},
					{APIName: "glDrawArrays", LineNum: 2},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestDriverErrorDetector_NonGlSetErrorIsErrorIgnored(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glDrawElements", IsError: true, ErrorCode: "0x0500", LineNum: 1},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for non-__glSetError with IsError, got %d", len(findings))
	}
}

func TestDriverErrorDetector_MultipleFrames(t *testing.T) {
	detector := &DriverErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glClear", LineNum: 1},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 2},
				},
			},
			{
				FrameNum: 2,
				APICalls: []core.APILogEntry{
					{APIName: "glDrawArrays", LineNum: 3},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0501", LineNum: 4},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings across frames, got %d", len(findings))
	}
}
