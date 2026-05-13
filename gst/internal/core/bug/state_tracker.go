package bug

import (
	"strconv"
	"strings"
	"sync"

	"gst/internal/core"
)

const (
	GL_ARRAY_BUFFER         = 0x8892
	GL_ELEMENT_ARRAY_BUFFER = 0x8893
	GL_FRAMEBUFFER          = 0x8D40
)

type VertexAttribState struct {
	Size               int
	TypeHex            string
	Stride             int
	PointerVal         string
	ArrayBufferBinding int
	LineNum            int
}

type PerContextState struct {
	ArrayBuffer     int
	ElementBuffer   int
	VAO             int
	Program         int
	ActiveTexUnit   int
	TextureBindings map[int]int
	FBO             int
	VertexAttribs   map[int]VertexAttribState
	AttribEnabled   map[int]bool
}

func newPerContextState() *PerContextState {
	return &PerContextState{
		ArrayBuffer:     0,
		ElementBuffer:   0,
		VAO:             0,
		Program:         0,
		ActiveTexUnit:   0,
		TextureBindings: make(map[int]int),
		FBO:             0,
		VertexAttribs:   make(map[int]VertexAttribState),
		AttribEnabled:   make(map[int]bool),
	}
}

type GLStateTracker struct {
	mu       sync.RWMutex
	contexts map[string]*PerContextState
}

func NewGLStateTracker() *GLStateTracker {
	return &GLStateTracker{
		contexts: make(map[string]*PerContextState),
	}
}

func (s *GLStateTracker) getOrCreate(gcAddr string) *PerContextState {
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx
	}
	ctx := newPerContextState()
	s.contexts[gcAddr] = ctx
	return ctx
}

func (s *GLStateTracker) ProcessCall(call *core.APILogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := s.getOrCreate(call.GCAddr)
	params := call.RawParams

	switch call.APIName {
	case "glBindBuffer":
		s.handleBindBuffer(ctx, params)
	case "glBindVertexArray":
		s.handleBindVertexArray(ctx, params)
	case "glVertexAttribPointer":
		s.handleVertexAttribPointer(ctx, params)
	case "glVertexAttribIPointer", "glVertexAttribLPointer":
		s.handleVertexAttribPointer(ctx, params)
	case "glEnableVertexAttribArray":
		s.handleEnableVertexAttribArray(ctx, params)
	case "glDisableVertexAttribArray":
		s.handleDisableVertexAttribArray(ctx, params)
	case "glUseProgram":
		s.handleUseProgram(ctx, params)
	case "glActiveTexture":
		s.handleActiveTexture(ctx, params)
	case "glBindTexture":
		s.handleBindTexture(ctx, params)
	case "glBindFramebuffer":
		s.handleBindFramebuffer(ctx, params)
	case "glDeleteBuffers":
		s.handleDeleteBuffers(ctx, params)
	case "glDeleteVertexArrays":
		s.handleDeleteVAOs(ctx, params)
	case "glDeleteProgram":
		s.handleDeleteProgram(ctx, params)
	case "glDeleteTextures":
		s.handleDeleteTextures(ctx, params)
	case "glDeleteFramebuffers":
		s.handleDeleteFramebuffers(ctx, params)
	}
}

func (s *GLStateTracker) GetBoundVBO(gcAddr string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.ArrayBuffer
	}
	return 0
}

func (s *GLStateTracker) GetBoundEBO(gcAddr string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.ElementBuffer
	}
	return 0
}

func (s *GLStateTracker) IsVBOBound(gcAddr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.ArrayBuffer != 0
	}
	return false
}

func (s *GLStateTracker) IsEBOBound(gcAddr string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.ElementBuffer != 0
	}
	return false
}

func (s *GLStateTracker) GetCurrentProgram(gcAddr string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.Program
	}
	return 0
}

func (s *GLStateTracker) GetBoundVAO(gcAddr string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.VAO
	}
	return 0
}

func (s *GLStateTracker) GetBoundFBO(gcAddr string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.FBO
	}
	return 0
}

func (s *GLStateTracker) GetTextureBinding(gcAddr string, unit int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		if tex, ok2 := ctx.TextureBindings[unit]; ok2 {
			return tex
		}
	}
	return 0
}

func (s *GLStateTracker) GetVertexAttrib(gcAddr string, index int) (VertexAttribState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		attrib, ok2 := ctx.VertexAttribs[index]
		return attrib, ok2
	}
	return VertexAttribState{}, false
}

func (s *GLStateTracker) IsAttribEnabled(gcAddr string, index int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		return ctx.AttribEnabled[index]
	}
	return false
}

func (s *GLStateTracker) GetState(gcAddr string) *PerContextState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ctx, ok := s.contexts[gcAddr]; ok {
		copy := *ctx
		copy.TextureBindings = make(map[int]int)
		for k, v := range ctx.TextureBindings {
			copy.TextureBindings[k] = v
		}
		copy.VertexAttribs = make(map[int]VertexAttribState)
		for k, v := range ctx.VertexAttribs {
			copy.VertexAttribs[k] = v
		}
		copy.AttribEnabled = make(map[int]bool)
		for k, v := range ctx.AttribEnabled {
			copy.AttribEnabled[k] = v
		}
		return &copy
	}
	return nil
}

func (s *GLStateTracker) handleBindBuffer(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	target := parseHexOrDec(parts[0])
	buffer := parseHexOrDec(parts[1])
	switch target {
	case GL_ARRAY_BUFFER:
		ctx.ArrayBuffer = buffer
	case GL_ELEMENT_ARRAY_BUFFER:
		ctx.ElementBuffer = buffer
	}
}

func (s *GLStateTracker) handleBindVertexArray(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return
	}
	ctx.VAO = parseHexOrDec(parts[0])
}

func (s *GLStateTracker) handleVertexAttribPointer(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 5 {
		return
	}
	index := parseHexOrDec(parts[0])
	size := parseHexOrDec(parts[1])
	typeHex := parts[2]
	stride := parseHexOrDec(parts[3])
	ptrVal := parts[len(parts)-1]
	ctx.VertexAttribs[index] = VertexAttribState{
		Size:               size,
		TypeHex:            typeHex,
		Stride:             stride,
		PointerVal:         ptrVal,
		ArrayBufferBinding: ctx.ArrayBuffer,
	}
}

func (s *GLStateTracker) handleEnableVertexAttribArray(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return
	}
	index := parseHexOrDec(parts[0])
	ctx.AttribEnabled[index] = true
}

func (s *GLStateTracker) handleDisableVertexAttribArray(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return
	}
	index := parseHexOrDec(parts[0])
	ctx.AttribEnabled[index] = false
}

func (s *GLStateTracker) handleUseProgram(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return
	}
	ctx.Program = parseHexOrDec(parts[0])
}

func (s *GLStateTracker) handleActiveTexture(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return
	}
	texEnum := parseHexOrDec(parts[0])
	ctx.ActiveTexUnit = texEnum
}

func (s *GLStateTracker) handleBindTexture(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	texture := parseHexOrDec(parts[1])
	ctx.TextureBindings[ctx.ActiveTexUnit] = texture
}

func (s *GLStateTracker) handleBindFramebuffer(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	ctx.FBO = parseHexOrDec(parts[1])
}

func (s *GLStateTracker) handleDeleteBuffers(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	for _, p := range parts[1:] {
		id := parseHexOrDec(p)
		if id != 0 && id == ctx.ArrayBuffer {
			ctx.ArrayBuffer = 0
		}
		if id != 0 && id == ctx.ElementBuffer {
			ctx.ElementBuffer = 0
		}
	}
}

func (s *GLStateTracker) handleDeleteVAOs(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	for _, p := range parts[1:] {
		id := parseHexOrDec(p)
		if id != 0 && id == ctx.VAO {
			ctx.VAO = 0
		}
	}
}

func (s *GLStateTracker) handleDeleteProgram(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 1 {
		return
	}
	progID := parseHexOrDec(parts[0])
	if progID != 0 && progID == ctx.Program {
		ctx.Program = 0
	}
}

func (s *GLStateTracker) handleDeleteTextures(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	for _, p := range parts[1:] {
		id := parseHexOrDec(p)
		for unit, tex := range ctx.TextureBindings {
			if id != 0 && id == tex {
				delete(ctx.TextureBindings, unit)
			}
		}
	}
}

func (s *GLStateTracker) handleDeleteFramebuffers(ctx *PerContextState, params string) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return
	}
	for _, p := range parts[1:] {
		id := parseHexOrDec(p)
		if id != 0 && id == ctx.FBO {
			ctx.FBO = 0
		}
	}
}

func parseHexOrDec(s string) int {
	s = strings.TrimSpace(s)
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		v, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 0
		}
		return int(v)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}
