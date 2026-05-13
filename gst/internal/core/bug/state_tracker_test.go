package bug

import (
	"os"
	"strings"
	"testing"

	"gst/internal/core"
)

func TestGLStateTracker_VBOBinding(t *testing.T) {
	tracker := NewGLStateTracker()

	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8892 199",
		GCAddr:    ctx,
	})

	if !tracker.IsVBOBound(ctx) {
		t.Error("VBO should be bound after glBindBuffer(ARRAY_BUFFER, 199)")
	}
	if vbo := tracker.GetBoundVBO(ctx); vbo != 199 {
		t.Errorf("expected VBO 199, got %d", vbo)
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8893 498",
		GCAddr:    ctx,
	})

	if !tracker.IsEBOBound(ctx) {
		t.Error("EBO should be bound after glBindBuffer(ELEMENT_ARRAY_BUFFER, 498)")
	}
	if ebo := tracker.GetBoundEBO(ctx); ebo != 498 {
		t.Errorf("expected EBO 498, got %d", ebo)
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8892 0",
		GCAddr:    ctx,
	})

	if tracker.IsVBOBound(ctx) {
		t.Error("VBO should NOT be bound after glBindBuffer(ARRAY_BUFFER, 0)")
	}
}

func TestGLStateTracker_VAOState(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindVertexArray",
		RawParams: "7",
		GCAddr:    ctx,
	})

	vao := tracker.GetBoundVAO(ctx)
	if vao != 7 {
		t.Errorf("expected VAO 7, got %d", vao)
	}
}

func TestGLStateTracker_ShaderProgram(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glUseProgram",
		RawParams: "18",
		GCAddr:    ctx,
	})

	prog := tracker.GetCurrentProgram(ctx)
	if prog != 18 {
		t.Errorf("expected program 18, got %d", prog)
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glUseProgram",
		RawParams: "0",
		GCAddr:    ctx,
	})

	prog = tracker.GetCurrentProgram(ctx)
	if prog != 0 {
		t.Errorf("expected program 0 after disabling, got %d", prog)
	}
}

func TestGLStateTracker_TextureState(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glActiveTexture",
		RawParams: "0x84c0",
		GCAddr:    ctx,
	})

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindTexture",
		RawParams: "0x0de1 42",
		GCAddr:    ctx,
	})

	tex := tracker.GetTextureBinding(ctx, 0x84c0)
	if tex != 42 {
		t.Errorf("expected texture 42 on unit 0x84c0, got %d", tex)
	}
}

func TestGLStateTracker_FramebufferState(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindFramebuffer",
		RawParams: "0x8d40 3",
		GCAddr:    ctx,
	})

	fbo := tracker.GetBoundFBO(ctx)
	if fbo != 3 {
		t.Errorf("expected FBO 3, got %d", fbo)
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindFramebuffer",
		RawParams: "0x8d40 0",
		GCAddr:    ctx,
	})

	fbo = tracker.GetBoundFBO(ctx)
	if fbo != 0 {
		t.Errorf("expected FBO 0 (unbind), got %d", fbo)
	}
}

func TestGLStateTracker_VertexAttribPointer(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8892 199",
		GCAddr:    ctx,
	})

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glVertexAttribPointer",
		RawParams: "0 3 0x1406 0x0 ptr=(nil)",
		GCAddr:    ctx,
	})

	attrib, ok := tracker.GetVertexAttrib(ctx, 0)
	if !ok {
		t.Fatal("vertex attrib 0 should be set")
	}
	if attrib.Size != 3 {
		t.Errorf("expected size 3, got %d", attrib.Size)
	}
	if attrib.TypeHex != "0x1406" {
		t.Errorf("expected type 0x1406, got %s", attrib.TypeHex)
	}
	if attrib.Stride != 0 {
		t.Errorf("expected stride 0, got %d", attrib.Stride)
	}
	if attrib.PointerVal != "ptr=(nil)" {
		t.Errorf("expected pointer ptr=(nil), got %s", attrib.PointerVal)
	}
}

func TestGLStateTracker_AttribEnableDisable(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glEnableVertexAttribArray",
		RawParams: "0",
		GCAddr:    ctx,
	})

	if !tracker.IsAttribEnabled(ctx, 0) {
		t.Error("attrib 0 should be enabled")
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glDisableVertexAttribArray",
		RawParams: "0",
		GCAddr:    ctx,
	})

	if tracker.IsAttribEnabled(ctx, 0) {
		t.Error("attrib 0 should be disabled")
	}
}

func TestGLStateTracker_PerContextIsolation(t *testing.T) {
	tracker := NewGLStateTracker()
	ctxA := "0xfffe6985a840"
	ctxB := "0xdeadbeef0000"

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8892 199",
		GCAddr:    ctxA,
	})

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8892 500",
		GCAddr:    ctxB,
	})

	if vboA := tracker.GetBoundVBO(ctxA); vboA != 199 {
		t.Errorf("ctxA VBO should be 199, got %d", vboA)
	}
	if vboB := tracker.GetBoundVBO(ctxB); vboB != 500 {
		t.Errorf("ctxB VBO should be 500, got %d", vboB)
	}
}

func TestGLStateTracker_DeleteOperations(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: ctx})
	tracker.ProcessCall(&core.APILogEntry{APIName: "glBindBuffer", RawParams: "0x8893 498", GCAddr: ctx})

	tracker.ProcessCall(&core.APILogEntry{APIName: "glDeleteBuffers", RawParams: "1 199", GCAddr: ctx})

	if tracker.IsVBOBound(ctx) {
		t.Error("VBO should be unbound after deletion")
	}
	if !tracker.IsEBOBound(ctx) {
		t.Error("EBO should still be bound (not deleted)")
	}

	tracker.ProcessCall(&core.APILogEntry{APIName: "glBindVertexArray", RawParams: "7", GCAddr: ctx})
	tracker.ProcessCall(&core.APILogEntry{APIName: "glDeleteVertexArrays", RawParams: "1 7", GCAddr: ctx})
	if vao := tracker.GetBoundVAO(ctx); vao != 0 {
		t.Errorf("VAO should be 0 after deletion, got %d", vao)
	}

	tracker.ProcessCall(&core.APILogEntry{APIName: "glUseProgram", RawParams: "18", GCAddr: ctx})
	tracker.ProcessCall(&core.APILogEntry{APIName: "glDeleteProgram", RawParams: "18", GCAddr: ctx})
	if prog := tracker.GetCurrentProgram(ctx); prog != 0 {
		t.Errorf("program should be 0 after deletion, got %d", prog)
	}

	tracker.ProcessCall(&core.APILogEntry{APIName: "glBindFramebuffer", RawParams: "0x8d40 3", GCAddr: ctx})
	tracker.ProcessCall(&core.APILogEntry{APIName: "glDeleteFramebuffers", RawParams: "1 3", GCAddr: ctx})
	if fbo := tracker.GetBoundFBO(ctx); fbo != 0 {
		t.Errorf("FBO should be 0 after deletion, got %d", fbo)
	}
}

func TestGLStateTracker_ClientArrayMode(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	if tracker.IsVBOBound(ctx) {
		t.Error("no VBO should be bound initially")
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glVertexAttribPointer",
		RawParams: "0 4 0x1406 0x0 ptr=(nil)",
		GCAddr:    ctx,
	})

	attrib, ok := tracker.GetVertexAttrib(ctx, 0)
	if !ok {
		t.Fatal("vertex attrib 0 should be set")
	}

	isNilPtr := attrib.PointerVal == "ptr=(nil)" || attrib.PointerVal == "(nil)"
	isClientArray := !tracker.IsVBOBound(ctx)
	if isNilPtr && isClientArray {
		t.Log("DANGER: ptr=(nil) with no VBO bound (client array mode, address 0x0)")
	}

	tracker.ProcessCall(&core.APILogEntry{
		APIName:   "glBindBuffer",
		RawParams: "0x8892 199",
		GCAddr:    ctx,
	})

	if !tracker.IsVBOBound(ctx) {
		t.Error("VBO should be bound")
	}

	if tracker.IsVBOBound(ctx) {
		t.Log("OK: ptr=(nil) with VBO bound, offset 0 in VBO")
	}
}

func TestGLStateTracker_GetStateCopy(t *testing.T) {
	tracker := NewGLStateTracker()
	ctx := "0xfffe6985a840"

	tracker.ProcessCall(&core.APILogEntry{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: ctx})
	tracker.ProcessCall(&core.APILogEntry{APIName: "glUseProgram", RawParams: "18", GCAddr: ctx})

	state := tracker.GetState(ctx)
	if state == nil {
		t.Fatal("GetState should return non-nil for existing context")
	}
	if state.ArrayBuffer != 199 {
		t.Errorf("state copy VBO should be 199, got %d", state.ArrayBuffer)
	}
	if state.Program != 18 {
		t.Errorf("state copy program should be 18, got %d", state.Program)
	}

	state.ArrayBuffer = 999
	if vbo := tracker.GetBoundVBO(ctx); vbo != 199 {
		t.Errorf("original VBO should still be 199, got %d", vbo)
	}

	if state := tracker.GetState("nonexistent"); state != nil {
		t.Error("GetState should return nil for unknown context")
	}
}

func TestGLStateTracker_SampleTraceWalkthrough(t *testing.T) {
	data, err := os.ReadFile("testdata/sample_trace.log")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	tracker := NewGLStateTracker()

	ctxA := "0xfffe6985a840"
	lines := strings.Split(string(data), "\n")
	callCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(line, "DANGER:") || strings.Contains(line, "段错误") || strings.Contains(line, "frame cost") {
			continue
		}
		if !strings.Contains(line, "(gc=") && !strings.Contains(line, "ERROR!!!") {
			continue
		}

		parts := strings.SplitN(line, "):", 2)
		if len(parts) < 2 {
			if strings.Contains(line, "ERROR!!!") {
				parts = strings.SplitN(line, "ERROR!!!", 2)
			}
			if len(parts) < 2 {
				continue
			}
		}

		callText := strings.TrimSpace(parts[len(parts)-1])

		var apiName, params string
		if idx := strings.Index(callText, " "); idx > 0 {
			apiName = callText[:idx]
			params = callText[idx+1:]
		} else {
			apiName = callText
		}

		if apiName == "" || !strings.HasPrefix(apiName, "gl") {
			continue
		}

		tracker.ProcessCall(&core.APILogEntry{
			APIName:   apiName,
			RawParams: params,
			GCAddr:    ctxA,
		})
		callCount++
	}

	t.Logf("Processed %d GL calls from sample trace", callCount)

	if prog := tracker.GetCurrentProgram(ctxA); prog != 22 {
		t.Errorf("after trace, expected program 22 (last glUseProgram), got %d", prog)
	}

	if fbo := tracker.GetBoundFBO(ctxA); fbo != 0 {
		t.Errorf("after trace, expected FBO 0, got %d", fbo)
	}

	if !tracker.IsAttribEnabled(ctxA, 3) {
		t.Error("attrib 3 should be enabled")
	}
	if _, ok := tracker.GetVertexAttrib(ctxA, 3); !ok {
		t.Error("vertex attrib 3 should be set")
	}

	if !tracker.IsAttribEnabled(ctxA, 0) {
		t.Error("attrib 0 should be enabled")
	}

	if !tracker.IsEBOBound(ctxA) {
		t.Error("EBO should remain bound (pointer-based glDeleteBuffers cannot resolve IDs)")
	}
}
