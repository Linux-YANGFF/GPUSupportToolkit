package bug

import (
	"strings"
	"testing"

	"gst/internal/core"
)

func buildFrame(calls []core.APILogEntry) core.FrameInfo {
	for i := range calls {
		if calls[i].LineNum == 0 {
			calls[i].LineNum = i + 1
		}
	}
	return core.FrameInfo{
		FrameNum: 1,
		APICalls: calls,
	}
}

func TestAntiPatternDetector_RedundantEnable(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glEnable", RawParams: "0x0be2", LineNum: 10},
		{APIName: "glEnable", RawParams: "0x0be2", LineNum: 11},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "antipattern" && strings.Contains(f.Description, "Redundant state changes") && strings.Contains(f.Description, "glEnable") {
			found = true
			if f.Severity != core.SeverityMedium {
				t.Errorf("expected severity medium, got %s", f.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected redundant glEnable detection")
	}
}

func TestAntiPatternDetector_RedundantDisable(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glDisable", RawParams: "0x0be2", LineNum: 20},
		{APIName: "glDisable", RawParams: "0x0be2", LineNum: 21},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "antipattern" && strings.Contains(f.Description, "Redundant state changes") && strings.Contains(f.Description, "glDisable") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected redundant glDisable detection")
	}
}

func TestAntiPatternDetector_NoRedundantWhenDifferent(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glEnable", RawParams: "0x0be2", LineNum: 10},
		{APIName: "glEnable", RawParams: "0x0b71", LineNum: 11},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if strings.Contains(f.Description, "Redundant") {
			t.Errorf("should not detect redundant with different params, got: %s", f.Description)
		}
	}
}

func TestAntiPatternDetector_MultipleBindBeforeDraw(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 10},
		{APIName: "glBindTexture", RawParams: "0x0de1 42", LineNum: 11},
		{APIName: "glDrawArrays", RawParams: "0x0004 0 6", LineNum: 12},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "antipattern" && strings.Contains(f.Description, "Multiple bind") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected invalid bindings detection")
	}
}

func TestAntiPatternDetector_BindThenDataThenDraw_Ok(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 10},
		{APIName: "glBufferData", RawParams: "0x8892 72 ptr 0x88e4", LineNum: 11},
		{APIName: "glDrawArrays", RawParams: "0x0004 0 6", LineNum: 12},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if strings.Contains(f.Description, "Multiple bind") {
			t.Errorf("should not detect invalid bindings when buffer data is uploaded: %s", f.Description)
		}
	}
}

func TestAntiPatternDetector_UnnecessaryContextSwitch(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glXMakeCurrent", RawParams: "dpy=0x555555 0x40001 ctx=0x555555", GCAddr: "0xfffe6985a840", LineNum: 10},
		{APIName: "glXMakeCurrent", RawParams: "dpy=0x555555 0x40002 ctx=0x555555", GCAddr: "0xfffe6985a849", LineNum: 11},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "antipattern" && strings.Contains(f.Description, "Unnecessary context switch") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected unnecessary context switch detection")
	}
}

func TestAntiPatternDetector_ContextSwitchWithDraw_Ok(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glXMakeCurrent", RawParams: "dpy=0x555555 0x40001 ctx=0x555555", GCAddr: "0xfffe6985a840", LineNum: 10},
		{APIName: "glDrawArrays", RawParams: "0x0004 0 6", LineNum: 11},
		{APIName: "glXMakeCurrent", RawParams: "dpy=0x555555 0x40002 ctx=0x555555", GCAddr: "0xfffe6985a849", LineNum: 12},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if strings.Contains(f.Description, "Unnecessary context switch") {
			t.Errorf("should not detect unnecessary context switch when draw exists between switches: %s", f.Description)
		}
	}
}

func TestAntiPatternDetector_UniformOnNullProgram(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glUseProgram", RawParams: "0", LineNum: 10},
		{APIName: "glUniform1i", RawParams: "0 1", LineNum: 11},
		{APIName: "glUniform4fv", RawParams: "1 1 ptr", LineNum: 12},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	count := 0
	for _, f := range findings {
		if f.Category == "antipattern" && strings.Contains(f.Description, "Uniform call on disabled program") {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 uniform-on-null-program findings, got %d", count)
	}
}

func TestAntiPatternDetector_UniformOnValidProgram_Ok(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glUseProgram", RawParams: "18", LineNum: 10},
		{APIName: "glUniform1i", RawParams: "0 1", LineNum: 11},
		{APIName: "glUniform4fv", RawParams: "1 1 ptr", LineNum: 12},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if strings.Contains(f.Description, "Uniform call on disabled program") {
			t.Errorf("should not detect uniform anti-pattern on valid program: %s", f.Description)
		}
	}
}

func TestAntiPatternDetector_DrawWithoutProgram(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glUseProgram", RawParams: "0", LineNum: 10},
		{APIName: "glDrawArrays", RawParams: "0x0004 0 6", LineNum: 11},
		{APIName: "glDrawElements", RawParams: "0x0004 6 0x1401 ptr", LineNum: 12},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	count := 0
	for _, f := range findings {
		if f.Category == "antipattern" && strings.Contains(f.Description, "Draw calls without active shader program") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 aggregated draw-without-program finding, got %d", count)
	}
}

func TestAntiPatternDetector_DrawWithProgram_Ok(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glUseProgram", RawParams: "22", LineNum: 10},
		{APIName: "glDrawArrays", RawParams: "0x0004 0 6", LineNum: 11},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if strings.Contains(f.Description, "Draw call without active shader") {
			t.Errorf("should not detect draw-without-program with valid program: %s", f.Description)
		}
	}
}

func TestAntiPatternDetector_EmptyLog(t *testing.T) {
	detector := NewAntiPatternDetector()
	log := &core.ParsedLog{Frames: []core.FrameInfo{}}
	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty log, got %d", len(findings))
	}
}

func TestAntiPatternDetector_AllFindingsHaveSeverityMedium(t *testing.T) {
	detector := NewAntiPatternDetector()
	frame := buildFrame([]core.APILogEntry{
		{APIName: "glEnable", RawParams: "0x0be2", LineNum: 10},
		{APIName: "glEnable", RawParams: "0x0be2", LineNum: 11},
		{APIName: "glUseProgram", RawParams: "0", LineNum: 12},
		{APIName: "glUniform1i", RawParams: "0 1", LineNum: 13},
		{APIName: "glDrawArrays", RawParams: "0x0004 0 6", LineNum: 14},
	})

	log := &core.ParsedLog{Frames: []core.FrameInfo{frame}}
	findings := detector.Diagnose(log)

	for i, f := range findings {
		if f.Category != "antipattern" {
			t.Errorf("finding %d: expected category 'antipattern', got '%s'", i, f.Category)
		}
		if f.Severity != core.SeverityMedium {
			t.Errorf("finding %d: expected severity medium, got '%s'", i, f.Severity)
		}
	}
}
