package bug

import (
	"testing"

	"gst/internal/core"
)

func TestNullPointerDetector_VBOBoundLegitimate(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", LineNum: 1, HasNilPtr: false},
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcA", LineNum: 2, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings when VBO is bound, got %d findings", len(findings))
		for _, f := range findings {
			t.Logf("unexpected finding: %s", f.Description)
		}
	}
}

func TestNullPointerDetector_NoVBOBoundDangerous(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glVertexAttribPointer", RawParams: "3 4 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcA", LineNum: 7, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when no VBO bound, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != core.SeverityCritical {
		t.Errorf("expected critical severity, got %s", f.Severity)
	}
	if f.Category != "null_pointer" {
		t.Errorf("expected category null_pointer, got %s", f.Category)
	}
	if !contains(f.Description, "ptr=(nil)") || !contains(f.Description, "no VBO bound") {
		t.Errorf("description should mention nil ptr and no VBO: %s", f.Description)
	}
	if !contains(f.Description, "client-array mode") {
		t.Errorf("description should mention client-array mode: %s", f.Description)
	}
}

func TestNullPointerDetector_glDrawElementsWithEBO(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8893 498", GCAddr: "0xgcA", LineNum: 1, HasNilPtr: false},
					{APIName: "glDrawElements", RawParams: "0x0004 2304 0x1403 ptr=(nil)", GCAddr: "0xgcA", LineNum: 2, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings when EBO is bound for glDrawElements, got %d", len(findings))
	}
}

func TestNullPointerDetector_glDrawElementsNoEBO(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glDrawElements", RawParams: "0x0004 2304 0x1403 ptr=(nil)", GCAddr: "0xgcA", LineNum: 5, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when no EBO bound for glDrawElements, got %d", len(findings))
	}
	f := findings[0]
	if f.Category != "null_pointer" {
		t.Errorf("expected category null_pointer, got %s", f.Category)
	}
	if !contains(f.Description, "glDrawElements") || !contains(f.Description, "no EBO bound") {
		t.Errorf("description should mention glDrawElements and no EBO: %s", f.Description)
	}
}

func TestNullPointerDetector_FailedBindBuffer(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", LineNum: 1, IsError: true},
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcA", LineNum: 2, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding when glBindBuffer failed before nil ptr, got %d", len(findings))
	}
	f := findings[0]
	if !contains(f.Description, "Previous glBindBuffer") {
		t.Errorf("description should mention Previous glBindBuffer: %s", f.Description)
	}
	if len(f.RootCauseChain) < 2 {
		t.Errorf("expected at least 2 root causes (one for nil ptr, one for failed bind), got %d: %v", len(f.RootCauseChain), f.RootCauseChain)
	}
}

func TestNullPointerDetector_VBOUnboundAfterDelete(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", LineNum: 1},
					{APIName: "glDeleteBuffers", RawParams: "1 199", GCAddr: "0xgcA", LineNum: 2},
					{APIName: "glVertexAttribPointer", RawParams: "3 4 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcA", LineNum: 3, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding after VBO deletion, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != core.SeverityCritical {
		t.Errorf("expected critical severity, got %s", f.Severity)
	}
	if !contains(f.Description, "no VBO bound") {
		t.Errorf("description should mention no VBO bound: %s", f.Description)
	}
}

func TestNullPointerDetector_TextureNilDataIsBenign(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glTexImage2D", RawParams: "0x0de1 0 0x1908 1920 1080 0 0x80e1 0x1401 ptr=(nil)", GCAddr: "0xgcA", LineNum: 10, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Fatalf("expected no finding for glTexImage2D nil data allocation, got %d", len(findings))
	}
}

func TestNullPointerDetector_GeneralNilPtrStillReported(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glMapBufferRange", RawParams: "0x8892 0 4096 0x1 ptr=(nil)", GCAddr: "0xgcA", LineNum: 10, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for general nil ptr, got %d", len(findings))
	}
	f := findings[0]
	if f.Category != "null_pointer" {
		t.Errorf("expected category null_pointer, got %s", f.Category)
	}
	if f.Kind != core.FindingKindBug || f.Confidence != core.ConfidenceHigh {
		t.Errorf("expected high-confidence bug metadata, got kind=%s confidence=%s", f.Kind, f.Confidence)
	}
}

func TestNullPointerDetector_ClientPointerNoDrawIsNotBug(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0 0x7ffeabcd", GCAddr: "0xgcA", LineNum: 10},
					{APIName: "glEnableVertexAttribArray", RawParams: "0", GCAddr: "0xgcA", LineNum: 11},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Fatalf("expected no bug/performance finding until client pointer is consumed by draw, got %d", len(findings))
	}
}

func TestNullPointerDetector_ClientPointerDrawUseIsPerformanceFinding(t *testing.T) {
	detector := NewNullPointerDetector()

	calls := []core.APILogEntry{
		{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0 0x7ffeabcd", GCAddr: "0xgcA", LineNum: 10},
		{APIName: "glEnableVertexAttribArray", RawParams: "0", GCAddr: "0xgcA", LineNum: 11},
	}
	for i := 0; i < 5; i++ {
		calls = append(calls, core.APILogEntry{APIName: "glDrawArrays", RawParams: "0x0004 0 3", GCAddr: "0xgcA", LineNum: 20 + i})
	}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{FrameNum: 1, APICalls: calls},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected one aggregated performance finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != core.SeverityMedium || f.Category != "performance" || f.Kind != core.FindingKindPerformance {
		t.Errorf("expected medium performance finding, got severity=%s category=%s kind=%s", f.Severity, f.Category, f.Kind)
	}
	if f.Count != 5 {
		t.Errorf("expected count 5, got %d", f.Count)
	}
	if len(f.Examples) == 0 {
		t.Error("expected examples for aggregated finding")
	}
}

func TestNullPointerDetector_SampleTrace(t *testing.T) {
	detector := NewNullPointerDetector()

	tracker := NewGLStateTracker()
	ctxA := "0xfffe6985a840"

	calls := []core.APILogEntry{
		{APIName: "glGenBuffers", RawParams: "3", GCAddr: ctxA, LineNum: 3},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: ctxA, LineNum: 4},
		{APIName: "glBufferData", RawParams: "0x8892 8512 0x7fa1ba6970 0x88e4", GCAddr: ctxA, LineNum: 5},
		{APIName: "glBindBuffer", RawParams: "0x8893 498", GCAddr: ctxA, LineNum: 6},
		{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: ctxA, LineNum: 7, HasNilPtr: true},
		{APIName: "glEnableVertexAttribArray", RawParams: "0", GCAddr: ctxA, LineNum: 8},
		{APIName: "glUseProgram", RawParams: "18", GCAddr: ctxA, LineNum: 9},
		{APIName: "glDrawElements", RawParams: "0x0004 2304 0x1403 ptr=(nil)", GCAddr: ctxA, LineNum: 14, HasNilPtr: true},
		{APIName: "glVertexAttribPointer", RawParams: "1 2 0x1406 0x0 ptr=(nil)", GCAddr: ctxA, LineNum: 15, HasNilPtr: true},
		{APIName: "glEnableVertexAttribArray", RawParams: "1", GCAddr: ctxA, LineNum: 16},
		{APIName: "glDeleteBuffers", RawParams: "3 0x7ffc8a1b0", GCAddr: ctxA, LineNum: 24},
		{APIName: "glVertexAttribPointer", RawParams: "3 4 0x1406 0x0 ptr=(nil)", GCAddr: ctxA, LineNum: 26, HasNilPtr: true},
		{APIName: "glEnableVertexAttribArray", RawParams: "3", GCAddr: ctxA, LineNum: 27},
	}

	for _, call := range calls {
		tracker.ProcessCall(&call)
	}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{FrameNum: 1, APICalls: calls},
		},
	}

	findings := detector.Diagnose(log)

	for _, f := range findings {
		t.Logf("finding: %s", f.Description)
		if f.Category != "null_pointer" {
			t.Errorf("all findings should have category null_pointer, got %s", f.Category)
		}
	}

	if len(findings) != 0 {
		t.Errorf("expected 0 findings from sample trace (VBO is bound for all nil-ptr vertex attrib calls, EBO bound for glDrawElements), got %d", len(findings))
	}
}

func TestNullPointerDetector_EmptyLog(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings for empty log, got %d", len(findings))
	}
}

func TestNullPointerDetector_NoNilPtrCalls(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", LineNum: 1},
					{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA", LineNum: 2},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", GCAddr: "0xgcA", LineNum: 3},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings for log without nil ptr, got %d", len(findings))
	}
}

func TestNullPointerDetector_MultiFrame(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", LineNum: 1},
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcA", LineNum: 2, HasNilPtr: true},
				},
			},
			{
				FrameNum: 2,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 0", GCAddr: "0xgcA", LineNum: 10},
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcA", LineNum: 11, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (frame 2 has unbound VBO), got %d", len(findings))
	}
	if findings[0].Evidence == "" {
		t.Error("finding should have evidence")
	}
}

func TestNullPointerDetector_PerContextIsolation(t *testing.T) {
	detector := NewNullPointerDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA", LineNum: 1},
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xgcB", LineNum: 2, HasNilPtr: true},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding: ctxB has no VBO bound, got %d", len(findings))
	}
	if !contains(findings[0].Description, "no VBO bound") {
		t.Errorf("finding should indicate no VBO bound: %s", findings[0].Description)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
