package bug

import (
	"testing"

	"gst/internal/core"
)

func TestThreadSafetyDetector_EmptyLog(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{Frames: []core.FrameInfo{}}
	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings for empty log, got %d", len(findings))
	}
}

func TestThreadSafetyDetector_SingleThreadSingleContext(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", GCAddr: "0xgcA", TID: "0xT1", LineNum: 2},
					{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT1", LineNum: 3},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings for single thread, got %d", len(findings))
		for _, f := range findings {
			t.Logf("unexpected finding: %s", f.Description)
		}
	}
}

func TestThreadSafetyDetector_NoTIDOrGC(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 1},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", LineNum: 2},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings without TID/GC, got %d", len(findings))
	}
}

func TestThreadSafetyDetector_ThreadSwitchWithoutUnbind(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT1", LineNum: 2},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", GCAddr: "0xgcA", TID: "0xT2", LineNum: 3},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for thread switch without unbind, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != core.SeverityHigh {
		t.Errorf("expected high severity, got %s", f.Severity)
	}
	if f.Category != "thread_safety" {
		t.Errorf("expected category thread_safety, got %s", f.Category)
	}
	if !stringsContain(f.Description, "0xgcA") || !stringsContain(f.Description, "0xT1") || !stringsContain(f.Description, "0xT2") {
		t.Errorf("description should mention context and TIDs: %s", f.Description)
	}
	if len(f.RootCauseChain) < 2 {
		t.Errorf("expected at least 2 root causes, got %d", len(f.RootCauseChain))
	}
	if f.FixSuggestion == "" {
		t.Error("fix suggestion should not be empty")
	}
}

func TestThreadSafetyDetector_ThreadSwitchWithProperUnbind(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 0", GCAddr: "0xgcA", TID: "0xT1", LineNum: 2},
					{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 123", GCAddr: "0xgcA", TID: "0xT2", LineNum: 3},
					{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT2", LineNum: 4},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	for _, f := range findings {
		t.Logf("finding: %s (severity=%s)", f.Description, f.Severity)
	}
	hasHigh := false
	for _, f := range findings {
		if f.Severity == core.SeverityHigh {
			hasHigh = true
		}
	}
	if hasHigh {
		t.Error("expected no high-severity findings when context is properly unbound between threads")
	}
}

func TestThreadSafetyDetector_MultipleThreadsWithMediumFinding(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = None", GCAddr: "0xgcA", TID: "0xT1", LineNum: 2},
					{APIName: "glBindBuffer", RawParams: "0x8893 42", GCAddr: "0xgcA", TID: "0xT2", LineNum: 3},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	hasMedium := false
	for _, f := range findings {
		if f.Severity == core.SeverityMedium && f.Category == "thread_safety" {
			hasMedium = true
		}
	}
	if !hasMedium {
		t.Error("expected a medium-severity finding for multi-thread context access")
	}
}

func TestThreadSafetyDetector_DifferentContextsDifferentThreads(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glBindBuffer", RawParams: "0x8893 42", GCAddr: "0xgcB", TID: "0xT2", LineNum: 2},
					{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT1", LineNum: 3},
					{APIName: "glUseProgram", RawParams: "22", GCAddr: "0xgcB", TID: "0xT2", LineNum: 4},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings when different threads use different contexts, got %d", len(findings))
		for _, f := range findings {
			t.Logf("unexpected finding: %s", f.Description)
		}
	}
}

func TestThreadSafetyDetector_RapidAlternation(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT2", LineNum: 2},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", GCAddr: "0xgcA", TID: "0xT1", LineNum: 3},
					{APIName: "glBindTexture", RawParams: "0x0de1 42", GCAddr: "0xgcA", TID: "0xT2", LineNum: 4},
					{APIName: "glFlush", RawParams: "", GCAddr: "0xgcA", TID: "0xT1", LineNum: 5},
				},
			},
		},
	}
	findings := detector.Diagnose(log)

	switchCount := 0
	rapidFinding := false
	for _, f := range findings {
		t.Logf("finding: [%s] %s", f.Severity, f.Description)
		if stringsContain(f.Description, "Thread switch without unbind") {
			switchCount++
		}
		if stringsContain(f.Description, "Rapid thread switching") {
			rapidFinding = true
		}
	}
	if switchCount < 2 {
		t.Errorf("expected at least 2 switch findings for rapid alternation, got %d", switchCount)
	}
	if !rapidFinding {
		t.Error("expected a rapid thread switching summary finding")
	}
}

func TestThreadSafetyDetector_MultiFrame(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", GCAddr: "0xgcA", TID: "0xT1", LineNum: 2},
				},
			},
			{
				FrameNum: 2,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 200", GCAddr: "0xgcA", TID: "0xT2", LineNum: 10},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 6", GCAddr: "0xgcA", TID: "0xT2", LineNum: 11},
				},
			},
		},
	}
	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for cross-frame thread switch, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != core.SeverityHigh {
		t.Errorf("expected high severity, got %s", f.Severity)
	}
}

func TestThreadSafetyDetector_UnbindVariants(t *testing.T) {
	tests := []struct {
		name       string
		params     string
		expectHigh bool
	}{
		{"drawable None", "dpy = 0x1c00, drawable = None", false},
		{"drawable 0", "dpy = 0x1c00, drawable = 0", false},
		{"gc nil", "dpy = 0x1c00, drawable = 123, gc = (nil)", false},
		{"gc=nil", "dpy = 0x1c00, drawable = 123, gc=nil", false},
		{"ctx nil", "dpy = 0x1c00, drawable = 123, ctx = (nil)", false},
		{"draw=0", "dpy = 0x1c00, draw=0, gc=0x123", false},
		{"non-null bind", "dpy = 0x1c00, drawable = 123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewThreadSafetyDetector()
			log := &core.ParsedLog{
				Frames: []core.FrameInfo{{
					FrameNum: 1,
					APICalls: []core.APILogEntry{
						{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
						{APIName: "glXMakeCurrent", RawParams: tt.params, GCAddr: "0xgcA", TID: "0xT1", LineNum: 2},
						{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT2", LineNum: 3},
					},
				}},
			}

			findings := detector.Diagnose(log)
			hasHigh := false
			for _, f := range findings {
				if f.Severity == core.SeverityHigh {
					hasHigh = true
				}
			}
			if tt.expectHigh && !hasHigh {
				t.Errorf("expected high-severity finding, got none")
			}
			if !tt.expectHigh && hasHigh {
				t.Errorf("expected no high-severity finding, got one")
				for _, f := range findings {
					t.Logf("finding: %s", f.Description)
				}
			}
		})
	}
}

func TestThreadSafetyDetector_DuplicateSwitchSuppressed(t *testing.T) {
	detector := NewThreadSafetyDetector()
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{{
			FrameNum: 1,
			APICalls: []core.APILogEntry{
				{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", TID: "0xT1", LineNum: 1},
				{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", TID: "0xT2", LineNum: 2},
				{APIName: "glDrawArrays", RawParams: "0x0005 0 4", GCAddr: "0xgcA", TID: "0xT2", LineNum: 3},
			},
		}},
	}
	findings := detector.Diagnose(log)
	switchFindings := 0
	for _, f := range findings {
		if stringsContain(f.Description, "Thread switch without unbind") {
			switchFindings++
		}
	}
	if switchFindings != 1 {
		t.Errorf("expected exactly 1 switch finding for T1->T2 transition, got %d", switchFindings)
	}
}

func stringsContain(s, substr string) bool {
	return len(s) >= len(substr) && strSearch(s, substr)
}

func strSearch(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
