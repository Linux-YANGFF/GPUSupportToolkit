package bug

import (
	"testing"

	"gst/internal/core"
)

func TestNewRegistry_Empty(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.diagnosers) != 0 {
		t.Errorf("expected 0 diagnosers, got %d", len(r.diagnosers))
	}
	if r.byName == nil {
		t.Error("expected non-nil byName map")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	d := &DriverErrorDetector{}

	r.Register(d)

	if len(r.diagnosers) != 1 {
		t.Errorf("expected 1 diagnoser, got %d", len(r.diagnosers))
	}
}

func TestRegistry_RegisterNamed(t *testing.T) {
	r := NewRegistry()
	d := &DriverErrorDetector{}

	r.RegisterNamed("driver_errors", d)

	if len(r.diagnosers) != 1 {
		t.Errorf("expected 1 diagnoser, got %d", len(r.diagnosers))
	}
	if r.byName["driver_errors"] != d {
		t.Error("expected diagnoser to be retrievable by name")
	}
}

func TestRegistry_GetDiagnoser_Exists(t *testing.T) {
	r := NewRegistry()
	d := &ShaderErrorDetector{}
	r.RegisterNamed("shaders", d)

	got := r.GetDiagnoser("shaders")

	if got != d {
		t.Error("expected to get the same diagnoser back")
	}
}

func TestRegistry_GetDiagnoser_NotFound(t *testing.T) {
	r := NewRegistry()

	got := r.GetDiagnoser("nonexistent")

	if got != nil {
		t.Errorf("expected nil for unknown name, got %v", got)
	}
}

func TestRegistry_RunAll_EmptyRegistry(t *testing.T) {
	r := NewRegistry()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{FrameNum: 1, APICalls: []core.APILogEntry{
				{APIName: "glClear", LineNum: 1},
			}},
		},
	}

	findings := r.RunAll(log)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings from empty registry, got %d", len(findings))
	}
}

func TestRegistry_RunAll_SingleDiagnoser(t *testing.T) {
	r := NewRegistry()
	r.Register(&DriverErrorDetector{})

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindTexture", RawParams: "0x0de1 42", LineNum: 1, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 2, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := r.RunAll(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Category != "driver_error" {
		t.Errorf("expected driver_error category, got %s", findings[0].Category)
	}
}

func TestRegistry_RunAll_MultipleDiagnosers(t *testing.T) {
	r := NewRegistry()
	r.Register(&DriverErrorDetector{})
	r.Register(&ShaderErrorDetector{})

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindTexture", RawParams: "0x0de1 42", LineNum: 1, GCAddr: "0xaaa"},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 2, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := r.RunAll(log)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding from driver error detector, got %d", len(findings))
	}
	if findings[0].Category != "driver_error" {
		t.Errorf("expected driver_error, got %s", findings[0].Category)
	}
}

func TestRegistry_RunAll_ReturnsAllFindingsFromAllDiagnosers(t *testing.T) {
	r := NewRegistry()
	r.Register(&DriverErrorDetector{})
	r.Register(NewNullPointerDetector())

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindTexture", RawParams: "0x0de1 42", LineNum: 1, GCAddr: "0xaaa"},
					{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xaaa", LineNum: 2, HasNilPtr: true},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 3, GCAddr: "0xaaa"},
				},
			},
		},
	}

	findings := r.RunAll(log)

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (1 driver + 1 null pointer), got %d", len(findings))
	}
	categories := make(map[string]int)
	for _, f := range findings {
		categories[f.Category]++
	}
	if categories["driver_error"] != 1 {
		t.Errorf("expected 1 driver_error, got %d", categories["driver_error"])
	}
	if categories["null_pointer"] != 1 {
		t.Errorf("expected 1 null_pointer, got %d", categories["null_pointer"])
	}
}

func TestRegistry_RunAll_EmptyLog(t *testing.T) {
	r := NewRegistry()
	r.Register(&DriverErrorDetector{})
	r.Register(NewNullPointerDetector())

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{},
	}

	findings := r.RunAll(log)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty log, got %d", len(findings))
	}
}

func TestRegistry_RunAll_NoErrors(t *testing.T) {
	r := NewRegistry()
	r.Register(&DriverErrorDetector{})
	r.Register(NewNullPointerDetector())

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

	findings := r.RunAll(log)

	if len(findings) != 0 {
		t.Errorf("expected 0 findings for clean log, got %d", len(findings))
	}
}

func TestRegistry_RegisterMultiple(t *testing.T) {
	r := NewRegistry()
	d1 := &DriverErrorDetector{}
	d2 := &ShaderErrorDetector{}
	d3 := NewNullPointerDetector()

	r.Register(d1)
	r.Register(d2)
	r.Register(d3)

	if len(r.diagnosers) != 3 {
		t.Errorf("expected 3 diagnosers, got %d", len(r.diagnosers))
	}
	if r.diagnosers[0] != d1 {
		t.Error("expected d1 first")
	}
	if r.diagnosers[1] != d2 {
		t.Error("expected d2 second")
	}
	if r.diagnosers[2] != d3 {
		t.Error("expected d3 third")
	}
}

func TestRegistry_RegisterNamed_Overwrites(t *testing.T) {
	r := NewRegistry()
	d1 := &DriverErrorDetector{}
	d2 := &ShaderErrorDetector{}

	r.RegisterNamed("errors", d1)
	r.RegisterNamed("errors", d2)

	got := r.GetDiagnoser("errors")
	if got != d2 {
		t.Error("expected second registration to overwrite name lookup")
	}
	if len(r.diagnosers) != 2 {
		t.Errorf("expected 2 diagnosers in list, got %d", len(r.diagnosers))
	}
}
