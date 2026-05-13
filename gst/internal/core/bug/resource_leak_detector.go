package bug

import (
	"fmt"
	"strings"

	"gst/internal/core"
)

const (
	resCategoryResourceLeak = "resource_leak"
)

var (
	pluralGenCreates = map[string]string{
		"glGenBuffers":          "buffer",
		"glCreateBuffers":       "buffer",
		"glGenTextures":         "texture",
		"glCreateTextures":      "texture",
		"glGenFramebuffers":     "framebuffer",
		"glCreateFramebuffers":  "framebuffer",
		"glGenRenderbuffers":    "renderbuffer",
		"glCreateRenderbuffers": "renderbuffer",
		"glGenVertexArrays":     "vertex_array",
	}

	singularCreates = map[string]string{
		"glCreateShader":  "shader",
		"glCreateProgram": "program",
	}

	bindUseAPIs = map[string]string{
		"glBindBuffer":       "buffer",
		"glBindTexture":      "texture",
		"glBindFramebuffer":  "framebuffer",
		"glBindRenderbuffer": "renderbuffer",
		"glBindVertexArray":  "vertex_array",
		"glShaderSource":     "shader",
		"glCompileShader":    "shader",
		"glUseProgram":       "program",
		"glAttachShader":     "program",
		"glLinkProgram":      "program",
	}

	singularDeletes = map[string]string{
		"glDeleteShader":  "shader",
		"glDeleteProgram": "program",
	}

	pluralDeletes = map[string]string{
		"glDeleteBuffers":       "buffer",
		"glDeleteTextures":      "texture",
		"glDeleteFramebuffers":  "framebuffer",
		"glDeleteRenderbuffers": "renderbuffer",
		"glDeleteVertexArrays":  "vertex_array",
	}
)

type contextLeakTracker struct {
	buffers       map[int]bool
	textures      map[int]bool
	shaders       map[int]bool
	programs      map[int]bool
	framebuffers  map[int]bool
	renderbuffers map[int]bool
	vertexArrays  map[int]bool

	bufferDeletes       int
	textureDeletes      int
	shaderDeletes       int
	programDeletes      int
	framebufferDeletes  int
	renderbufferDeletes int
	vertexArrayDeletes  int
}

func newContextLeakTracker() *contextLeakTracker {
	return &contextLeakTracker{
		buffers:       make(map[int]bool),
		textures:      make(map[int]bool),
		shaders:       make(map[int]bool),
		programs:      make(map[int]bool),
		framebuffers:  make(map[int]bool),
		renderbuffers: make(map[int]bool),
		vertexArrays:  make(map[int]bool),
	}
}

func (t *contextLeakTracker) resourceMap(resType string) map[int]bool {
	switch resType {
	case "buffer":
		return t.buffers
	case "texture":
		return t.textures
	case "shader":
		return t.shaders
	case "program":
		return t.programs
	case "framebuffer":
		return t.framebuffers
	case "renderbuffer":
		return t.renderbuffers
	case "vertex_array":
		return t.vertexArrays
	}
	return nil
}

type ResourceLeakDetector struct {
	ctxManager *ContextManager
	trackers   map[string]*contextLeakTracker
}

func NewResourceLeakDetector() *ResourceLeakDetector {
	return &ResourceLeakDetector{
		ctxManager: NewContextManager(),
		trackers:   make(map[string]*contextLeakTracker),
	}
}

func (d *ResourceLeakDetector) getOrCreateTracker(gcAddr string) *contextLeakTracker {
	if t, ok := d.trackers[gcAddr]; ok {
		return t
	}
	t := newContextLeakTracker()
	d.trackers[gcAddr] = t
	return t
}

func (d *ResourceLeakDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	d.trackers = make(map[string]*contextLeakTracker)
	d.ctxManager = NewContextManager()

	for _, frame := range log.Frames {
		for _, call := range frame.APICalls {
			d.processCall(&call)
		}
	}

	return d.buildFindings()
}

func (d *ResourceLeakDetector) processCall(call *core.APILogEntry) {
	gcAddr := call.GCAddr
	apiName := call.APIName
	params := call.RawParams

	d.ctxManager.ProcessCall(call, nil)

	if gcAddr == "" {
		return
	}

	tracker := d.getOrCreateTracker(gcAddr)

	if _, ok := pluralGenCreates[apiName]; ok {
	}

	if resType, ok := singularCreates[apiName]; ok {
		ids := extractDecimalIDs(params)
		m := tracker.resourceMap(resType)
		if m != nil {
			for _, id := range ids {
				m[id] = true
			}
		}
	}

	if resType, ok := bindUseAPIs[apiName]; ok {
		ids := extractDecimalIDs(params)
		m := tracker.resourceMap(resType)
		if m != nil {
			for _, id := range ids {
				m[id] = true
			}
		}
	}

	if resType, ok := singularDeletes[apiName]; ok {
		ids := extractDecimalIDs(params)
		m := tracker.resourceMap(resType)
		if m != nil {
			for _, id := range ids {
				delete(m, id)
			}
		}
		switch resType {
		case "shader":
			tracker.shaderDeletes++
		case "program":
			tracker.programDeletes++
		}
	}

	if resType, ok := pluralDeletes[apiName]; ok {
		count := parseDeleteCount(params)
		switch resType {
		case "buffer":
			tracker.bufferDeletes += count
		case "texture":
			tracker.textureDeletes += count
		case "framebuffer":
			tracker.framebufferDeletes += count
		case "renderbuffer":
			tracker.renderbufferDeletes += count
		case "vertex_array":
			tracker.vertexArrayDeletes += count
		}
	}
}

func parseDeleteCount(params string) int {
	parts := strings.Fields(params)
	if len(parts) == 0 {
		return 0
	}
	first := strings.TrimSpace(parts[0])
	if len(first) > 2 && (first[:2] == "0x" || first[:2] == "0X") {
		return 1
	}
	count := 0
	for _, r := range first {
		if r >= '0' && r <= '9' {
			count = count*10 + int(r-'0')
		} else {
			break
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func (d *ResourceLeakDetector) buildFindings() []core.Finding {
	var findings []core.Finding

	for gcAddr, tracker := range d.trackers {
		findings = append(findings, d.checkResourceType("buffer", gcAddr, tracker.buffers, tracker.bufferDeletes)...)
		findings = append(findings, d.checkResourceType("texture", gcAddr, tracker.textures, tracker.textureDeletes)...)
		findings = append(findings, d.checkResourceType("shader", gcAddr, tracker.shaders, tracker.shaderDeletes)...)
		findings = append(findings, d.checkResourceType("program", gcAddr, tracker.programs, tracker.programDeletes)...)
		findings = append(findings, d.checkResourceType("framebuffer", gcAddr, tracker.framebuffers, tracker.framebufferDeletes)...)
		findings = append(findings, d.checkResourceType("renderbuffer", gcAddr, tracker.renderbuffers, tracker.renderbufferDeletes)...)
		findings = append(findings, d.checkResourceType("vertex_array", gcAddr, tracker.vertexArrays, tracker.vertexArrayDeletes)...)
	}

	return findings
}

func (d *ResourceLeakDetector) checkResourceType(resType, gcAddr string, resources map[int]bool, deleteCount int) []core.Finding {
	known := len(resources)

	if known == 0 {
		return nil
	}

	remaining := known - deleteCount
	if remaining <= 0 {
		return nil
	}

	var leakedIDs []int
	for id := range resources {
		if len(leakedIDs) >= 10 {
			break
		}
		leakedIDs = append(leakedIDs, id)
	}

	evidence := fmt.Sprintf("Context %s has %d %s(s) tracked but only %d deletions confirmed, %d likely leaked",
		gcAddr, known, resType, deleteCount, remaining)

	leakedStr := idList(leakedIDs)
	if len(resources) > 10 {
		leakedStr += fmt.Sprintf(" ... and %d more", len(resources)-10)
	}
	evidence += fmt.Sprintf(" (leaked IDs: %s)", leakedStr)

	return []core.Finding{{
		Severity:    core.SeverityLow,
		Category:    resCategoryResourceLeak,
		Kind:        core.FindingKindCompatibility,
		Confidence:  core.ConfidenceLow,
		Count:       remaining,
		Description: fmt.Sprintf("Possible %s lifetime issue at log end: %d resource(s) still tracked", resType, remaining),
		Evidence:    evidence + ". This is advisory because the trace may not include application shutdown or context destruction.",
		RootCauseChain: []string{
			fmt.Sprintf("%s(s) were observed but matching delete calls were not observed before log end", resType),
			"The trace may be truncated before normal cleanup, so this is not a high-confidence leak by itself",
		},
		FixSuggestion: fmt.Sprintf("If the log covers full application shutdown, ensure every glGen*/glCreate* call for %s has a matching glDelete* call. Otherwise treat this as lifecycle context rather than a primary bug.", resType),
	}}
}

func idList(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, ", ")
}
