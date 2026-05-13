package bug

import (
	"fmt"

	"gst/internal/core"
)

type NullPointerDetector struct {
	tracker *GLStateTracker
}

type clientArrayUse struct {
	count     int
	firstLine int
	examples  []string
}

func NewNullPointerDetector() *NullPointerDetector {
	return &NullPointerDetector{
		tracker: NewGLStateTracker(),
	}
}

// isLegitimateNil 判断 (nil) 是否是合法的参数（如 shareList=(nil), gc=(nil)）
func isLegitimateNil(call *core.APILogEntry) bool {
	switch call.APIName {
	case "glXCreateContextAttribsARB", "glXMakeCurrent", "wglMakeCurrent",
		"eglMakeCurrent", "glXCreateContext", "__glCreateContext":
		return true
	}
	return false
}

func (d *NullPointerDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	d.tracker = NewGLStateTracker()
	var findings []core.Finding
	var lastBindBufferFailed bool

	// Client-side arrays are legal in compatibility profiles. Report them only
	// when enabled attributes are actually consumed by draw calls.
	clientArrayDraws := make(map[string]*clientArrayUse)

	for _, frame := range log.Frames {
		for _, call := range frame.APICalls {
			if call.APIName == "glBindBuffer" && call.IsError {
				lastBindBufferFailed = true
				continue
			}

			d.tracker.ProcessCall(&call)

			if call.APIName == "glBindBuffer" {
				lastBindBufferFailed = false
			}

			// 检测 (nil) 空指针
			if call.HasNilPtr && !isLegitimateNil(&call) {
				finding := d.analyzeNilPtr(&call, lastBindBufferFailed)
				if finding != nil {
					findings = append(findings, *finding)
				}
			}

			if isDrawCall(call.APIName) && call.GCAddr != "" {
				d.collectClientArrayUse(call, clientArrayDraws)
			}
		}
	}

	// 聚合报告被 draw 实际使用的 client-side vertex arrays。
	for gcAddr, usage := range clientArrayDraws {
		if usage.count < 5 {
			continue
		}
		findings = append(findings, core.Finding{
			Severity:       core.SeverityMedium,
			Category:       "performance",
			Kind:           core.FindingKindPerformance,
			Confidence:     core.ConfidenceMedium,
			Count:          usage.count,
			Examples:       usage.examples,
			Description:    fmt.Sprintf("Client-side vertex arrays used by draw calls: %d enabled attributes on context %s read from client memory", usage.count, gcAddr),
			Evidence:       fmt.Sprintf("Context %s uses enabled vertex attributes without a VBO during draw calls (first observed at line %d). This is legal in compatibility profiles but can cause CPU/GPU synchronization and poor throughput", gcAddr, usage.firstLine),
			RootCauseChain: []string{"Enabled vertex attributes were configured while GL_ARRAY_BUFFER was 0", "Draw calls consume client memory instead of VBO-backed data", "This is a compatibility/performance concern rather than a null-pointer bug when pointer values are non-zero"},
			FixSuggestion:  "For performance, upload vertex data to VBOs and bind GL_ARRAY_BUFFER before glVertexAttribPointer. If this is intentional compatibility-profile rendering, this finding can be treated as advisory.",
		})
	}

	return findings
}

func (d *NullPointerDetector) collectClientArrayUse(call core.APILogEntry, clientArrayDraws map[string]*clientArrayUse) {
	state := d.tracker.GetState(call.GCAddr)
	if state == nil {
		return
	}
	for index, attrib := range state.VertexAttribs {
		if !state.AttribEnabled[index] {
			continue
		}
		if attrib.ArrayBufferBinding != 0 || isNilPointerValue(attrib.PointerVal) {
			continue
		}
		usage := clientArrayDraws[call.GCAddr]
		if usage == nil {
			usage = &clientArrayUse{firstLine: call.LineNum}
			clientArrayDraws[call.GCAddr] = usage
		}
		usage.count++
		if len(usage.examples) < 5 {
			usage.examples = append(usage.examples, fmt.Sprintf("draw line %d uses attrib %d client pointer %s", call.LineNum, index, attrib.PointerVal))
		}
	}
}

func (d *NullPointerDetector) analyzeNilPtr(call *core.APILogEntry, lastBindBufferFailed bool) *core.Finding {
	switch call.APIName {
	case "glDrawElements":
		return d.checkDrawElementsNilPtr(call, lastBindBufferFailed)
	default:
		if isVertexAttributeCall(call.APIName) {
			return d.checkVertexAttribNilPtr(call, lastBindBufferFailed)
		}
		return d.checkGeneralNilPtr(call, lastBindBufferFailed)
	}
}

func (d *NullPointerDetector) checkDrawElementsNilPtr(call *core.APILogEntry, lastBindBufferFailed bool) *core.Finding {
	ebo := d.tracker.GetBoundEBO(call.GCAddr)
	if ebo != 0 {
		return nil
	}

	desc := fmt.Sprintf("%s at line %d has ptr=(nil) with no EBO bound", call.APIName, call.LineNum)
	evidence := fmt.Sprintf("%s at line %d: params=%q, gc=%s, reason=EBO not bound", call.APIName, call.LineNum, call.RawParams, call.GCAddr)
	rootCause := []string{"No EBO bound when glDrawElements uses ptr=(nil)"}
	if lastBindBufferFailed {
		rootCause = append(rootCause, "Previous glBindBuffer for ELEMENT_ARRAY_BUFFER failed")
	}

	return &core.Finding{
		Severity:       core.SeverityCritical,
		Category:       "null_pointer",
		Kind:           core.FindingKindBug,
		Confidence:     core.ConfidenceHigh,
		Description:    desc,
		Evidence:       evidence,
		RootCauseChain: rootCause,
		FixSuggestion:  "Bind a valid EBO before glDrawElements",
	}
}

func (d *NullPointerDetector) checkVertexAttribNilPtr(call *core.APILogEntry, lastBindBufferFailed bool) *core.Finding {
	vbo := d.tracker.GetBoundVBO(call.GCAddr)
	if vbo != 0 {
		return nil
	}

	desc := fmt.Sprintf("%s at line %d has ptr=(nil) with no VBO bound (client-array mode, address 0x0)", call.APIName, call.LineNum)
	evidence := fmt.Sprintf("%s at line %d: params=%q, gc=%s, reason=VBO not bound", call.APIName, call.LineNum, call.RawParams, call.GCAddr)
	rootCause := []string{"No VBO bound when vertex attrib pointer uses ptr=(nil)"}
	if lastBindBufferFailed {
		rootCause = append(rootCause, "Previous glBindBuffer for ARRAY_BUFFER failed")
		desc += " (Previous glBindBuffer failed)"
	}

	return &core.Finding{
		Severity:       core.SeverityCritical,
		Category:       "null_pointer",
		Kind:           core.FindingKindBug,
		Confidence:     core.ConfidenceHigh,
		Description:    desc,
		Evidence:       evidence,
		RootCauseChain: rootCause,
		FixSuggestion:  "Bind a valid VBO before setting vertex attrib pointer",
	}
}

func (d *NullPointerDetector) checkGeneralNilPtr(call *core.APILogEntry, lastBindBufferFailed bool) *core.Finding {
	if isBenignNilDataPointer(call) {
		return nil
	}

	desc := fmt.Sprintf("%s at line %d has ptr=(nil)", call.APIName, call.LineNum)
	evidence := fmt.Sprintf("%s at line %d: params=%q, gc=%s, reason=null pointer argument", call.APIName, call.LineNum, call.RawParams, call.GCAddr)
	rootCause := []string{"Null pointer passed to GL API"}

	return &core.Finding{
		Severity:       core.SeverityCritical,
		Category:       "null_pointer",
		Kind:           core.FindingKindBug,
		Confidence:     core.ConfidenceHigh,
		Description:    desc,
		Evidence:       evidence,
		RootCauseChain: rootCause,
		FixSuggestion:  "Ensure a valid pointer is passed",
	}
}

func isBenignNilDataPointer(call *core.APILogEntry) bool {
	switch call.APIName {
	case "glTexImage1D", "glTexImage2D", "glTexImage3D",
		"glTexSubImage1D", "glTexSubImage2D", "glTexSubImage3D",
		"glCompressedTexImage1D", "glCompressedTexImage2D", "glCompressedTexImage3D",
		"glCompressedTexSubImage1D", "glCompressedTexSubImage2D", "glCompressedTexSubImage3D",
		"glBufferData":
		return true
	case "glShaderSource":
		// The optional length array may be NULL when strings are
		// null-terminated. apitrace often renders that argument as (nil).
		return true
	default:
		return false
	}
}

func isNilPointerValue(value string) bool {
	return value == "(nil)" || value == "ptr=(nil)" || value == "0" || value == "0x0"
}

func isVertexAttributeCall(apiName string) bool {
	switch apiName {
	case "glVertexAttribPointer", "glVertexAttribIPointer", "glVertexAttribLPointer",
		"glVertexPointer", "glNormalPointer", "glColorPointer",
		"glTexCoordPointer", "glSecondaryColorPointer", "glEdgeFlagPointer",
		"glFogCoordPointer":
		return true
	}
	return false
}
