package bug

import (
	"testing"

	"gst/internal/core"
)

func TestResourceLeakDetector_NoLeaks(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenBuffers", RawParams: "3", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 200", GCAddr: "0xgcA"},
				{APIName: "glDeleteBuffers", RawParams: "2", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no leaks, got %d findings", len(findings))
	}
}

func TestResourceLeakDetector_BufferLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenBuffers", RawParams: "2", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 498", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected buffer leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_TextureLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenTextures", RawParams: "1", GCAddr: "0xgcA"},
				{APIName: "glBindTexture", RawParams: "0x0de1 42", GCAddr: "0xgcA"},
				{APIName: "glBindTexture", RawParams: "0x0de1 99", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected texture leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_ShaderLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glCreateShader", RawParams: "0x8b31 5", GCAddr: "0xgcA"},
				{APIName: "glShaderSource", RawParams: "5", GCAddr: "0xgcA"},
				{APIName: "glCompileShader", RawParams: "5", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected shader leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_ShaderDeleted(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glCreateShader", RawParams: "0x8b31 5", GCAddr: "0xgcA"},
				{APIName: "glShaderSource", RawParams: "5", GCAddr: "0xgcA"},
				{APIName: "glDeleteShader", RawParams: "5", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no shader leak after delete, got %d findings", len(findings))
	}
}

func TestResourceLeakDetector_ProgramLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glCreateProgram", RawParams: "", GCAddr: "0xgcA"},
				{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected program leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_ProgramDeleted(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glCreateProgram", RawParams: "", GCAddr: "0xgcA"},
				{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA"},
				{APIName: "glDeleteProgram", RawParams: "18", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no program leak after delete, got %d findings", len(findings))
	}
}

func TestResourceLeakDetector_FramebufferLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenFramebuffers", RawParams: "1", GCAddr: "0xgcA"},
				{APIName: "glBindFramebuffer", RawParams: "0x8d40 5", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected framebuffer leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_RenderbufferLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenRenderbuffers", RawParams: "1", GCAddr: "0xgcA"},
				{APIName: "glBindRenderbuffer", RawParams: "0x8d41 7", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected renderbuffer leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_VertexArrayLeak(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenVertexArrays", RawParams: "1", GCAddr: "0xgcA"},
				{APIName: "glBindVertexArray", RawParams: "7", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected vertex array leak findings")
	}

	for _, f := range findings {
		if f.Severity != core.SeverityLow {
			t.Errorf("expected severity low, got %s", f.Severity)
		}
		if f.Category != "resource_leak" {
			t.Errorf("expected category resource_leak, got %s", f.Category)
		}
	}
}

func TestResourceLeakDetector_MultipleContexts(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenBuffers", RawParams: "1", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
				{APIName: "glGenTextures", RawParams: "1", GCAddr: "0xgcB"},
				{APIName: "glBindTexture", RawParams: "0x0de1 42", GCAddr: "0xgcB"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) < 2 {
		t.Fatalf("expected at least 2 findings for two contexts, got %d", len(findings))
	}

	gcAHasBufferLeak := false
	gcBHasTextureLeak := false
	for _, f := range findings {
		if f.Category == "resource_leak" {
			if contains(f.Evidence, "0xgcA") && contains(f.Description, "buffer") {
				gcAHasBufferLeak = true
			}
			if contains(f.Evidence, "0xgcB") && contains(f.Description, "texture") {
				gcBHasTextureLeak = true
			}
		}
	}
	if !gcAHasBufferLeak {
		t.Error("expected buffer leak in context 0xgcA")
	}
	if !gcBHasTextureLeak {
		t.Error("expected texture leak in context 0xgcB")
	}
}

func TestResourceLeakDetector_EmptyLog(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{}

	findings := d.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected no findings for empty log, got %d", len(findings))
	}
}

func TestResourceLeakDetector_MixedResourceTypes(t *testing.T) {
	d := NewResourceLeakDetector()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{APICalls: []core.APILogEntry{
				{APIName: "glGenBuffers", RawParams: "2", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 1", GCAddr: "0xgcA"},
				{APIName: "glBindBuffer", RawParams: "0x8892 2", GCAddr: "0xgcA"},
				{APIName: "glGenTextures", RawParams: "1", GCAddr: "0xgcA"},
				{APIName: "glBindTexture", RawParams: "0x0de1 10", GCAddr: "0xgcA"},
				{APIName: "glCreateShader", RawParams: "0x8b31 5", GCAddr: "0xgcA"},
				{APIName: "glDeleteBuffers", RawParams: "2", GCAddr: "0xgcA"},
				{APIName: "glDeleteTextures", RawParams: "1", GCAddr: "0xgcA"},
			}},
		}}

	findings := d.Diagnose(log)
	if len(findings) == 0 {
		t.Fatal("expected shader leak finding")
	}

	hasShaderLeak := false
	hasBufferLeak := false
	for _, f := range findings {
		if contains(f.Description, "shader") {
			hasShaderLeak = true
		}
		if contains(f.Description, "buffer") {
			hasBufferLeak = true
		}
	}
	if !hasShaderLeak {
		t.Error("expected shader leak finding (shader created but not deleted)")
	}
	if hasBufferLeak {
		t.Error("unexpected buffer leak (all buffers deleted)")
	}
}
