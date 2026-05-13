package bug

import (
	"fmt"
	"strings"

	"gst/internal/core"
)

type AntiPatternDetector struct {
	stateTracker   *GLStateTracker
	contextManager *ContextManager
}

func NewAntiPatternDetector() *AntiPatternDetector {
	return &AntiPatternDetector{
		stateTracker:   NewGLStateTracker(),
		contextManager: NewContextManager(),
	}
}

func (d *AntiPatternDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	// Collect raw findings per type, then aggregate
	typeCounts := make(map[string]int)
	var aggregated []core.Finding

	for fi := range log.Frames {
		frame := &log.Frames[fi]

		// Redundant state changes - count per pattern, report once
		redundant := d.countRedundantStateChanges(frame)
		for pattern, count := range redundant {
			key := "redundant_state:" + pattern
			if _, exists := typeCounts[key]; !exists {
				typeCounts[key] = count
			} else {
				typeCounts[key] += count
			}
		}

		// Invalid bindings
		findings := d.detectInvalidBindings(frame)
		for _, f := range findings {
			key := "invalid_binding:" + f.Description
			if _, exists := typeCounts[key]; !exists {
				typeCounts[key] = 1
				aggregated = append(aggregated, f)
			} else {
				typeCounts[key]++
			}
		}

		// Context switches
		findings = d.detectUnnecessaryContextSwitches(frame)
		for _, f := range findings {
			aggregated = append(aggregated, f)
		}

		// Uniforms on null program
		findings = d.detectUniformsOnNullProgram(frame)
		for _, f := range findings {
			aggregated = append(aggregated, f)
		}

		// Draw without program - count per frame, report once if frequent
		drawNoProg := d.countDrawWithoutProgram(frame)
		if drawNoProg > 0 {
			key := "draw_without_program"
			if _, exists := typeCounts[key]; !exists {
				typeCounts[key] = drawNoProg
			} else {
				typeCounts[key] += drawNoProg
			}
		}
	}

	// Generate aggregated findings for redundant state changes
	for pattern, count := range typeCounts {
		if strings.HasPrefix(pattern, "redundant_state:") && count > 0 {
			desc := strings.TrimPrefix(pattern, "redundant_state:")
			aggregated = append(aggregated, core.Finding{
				Severity:       core.SeverityMedium,
				Category:       "antipattern",
				Description:    fmt.Sprintf("Redundant state changes: %s (found %d times across all frames)", desc, count),
				Evidence:       fmt.Sprintf("Pattern detected %d times, consider removing redundant calls", count),
				RootCauseChain: []string{"Repeated identical state changes waste CPU cycles"},
				FixSuggestion:  "Remove redundant calls, or batch state changes more efficiently",
			})
		}
	}

	// Generate aggregated finding for draw without program
	if count, ok := typeCounts["draw_without_program"]; ok && count > 0 {
		aggregated = append(aggregated, core.Finding{
			Severity:       core.SeverityMedium,
			Category:       "antipattern",
			Description:    fmt.Sprintf("Draw calls without active shader program: %d occurrences across all frames", count),
			Evidence:       fmt.Sprintf("glDrawArrays/glDrawElements called after glUseProgram(0) %d times", count),
			RootCauseChain: []string{"No active shader program when draw calls are issued", "May indicate missing glUseProgram or premature glUseProgram(0)"},
			FixSuggestion:  "Ensure glUseProgram is called with a valid program before each draw call sequence",
		})
	}

	return aggregated
}

func (d *AntiPatternDetector) countRedundantStateChanges(frame *core.FrameInfo) map[string]int {
	counts := make(map[string]int)
	lastEnableCall := ""
	lastEnableParams := ""
	lastDisableCall := ""
	lastDisableParams := ""

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		switch call.APIName {
		case "glEnable", "glEnableVertexAttribArray":
			params := strings.TrimSpace(call.RawParams)
			if call.APIName == lastEnableCall && params == lastEnableParams {
				counts[call.APIName+"("+params+")"]++
			}
			lastEnableCall = call.APIName
			lastEnableParams = params
		case "glDisable", "glDisableVertexAttribArray":
			params := strings.TrimSpace(call.RawParams)
			if call.APIName == lastDisableCall && params == lastDisableParams {
				counts[call.APIName+"("+params+")"]++
			}
			lastDisableCall = call.APIName
			lastDisableParams = params
		}
	}
	return counts
}

func (d *AntiPatternDetector) detectInvalidBindings(frame *core.FrameInfo) []core.Finding {
	var findings []core.Finding
	consecutiveBinds := 0
	consecutiveBindAPI := ""
	consecutiveBindLine := 0

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		switch call.APIName {
		case "glBindBuffer", "glBindVertexArray", "glBindTexture", "glBindFramebuffer":
			consecutiveBinds++
			if consecutiveBinds == 1 {
				consecutiveBindAPI = call.APIName
				consecutiveBindLine = call.LineNum
			}
		case "glDrawArrays", "glDrawElements", "glDrawRangeElements":
			if consecutiveBinds >= 2 {
				findings = append(findings, core.Finding{
					Severity:       core.SeverityMedium,
					Category:       "antipattern",
					Description:    "Multiple bind calls before draw without data setup: " + consecutiveBindAPI + " followed by draw",
					Evidence:       fmt.Sprintf("Line %d: %s followed by draw on line %d without vertex data setup", consecutiveBindLine, consecutiveBindAPI, call.LineNum),
					RootCauseChain: []string{"No glBufferData/glVertexAttribPointer between bind and draw"},
					FixSuggestion:  "Ensure buffer data is uploaded before draw calls",
				})
			}
			consecutiveBinds = 0
		case "glBufferData", "glBufferSubData", "glVertexAttribPointer":
			consecutiveBinds = 0
		}
	}
	return findings
}

func (d *AntiPatternDetector) detectUnnecessaryContextSwitches(frame *core.FrameInfo) []core.Finding {
	var findings []core.Finding
	lastMakeCurrentLine := 0
	lastMakeCurrentGC := ""
	hasDrawSinceSwitch := false
	firstMakeCurrentSeen := false

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		if call.APIName == "glXMakeCurrent" || call.APIName == "wglMakeCurrent" || call.APIName == "eglMakeCurrent" {
			if firstMakeCurrentSeen && !hasDrawSinceSwitch {
				findings = append(findings, core.Finding{
					Severity:       core.SeverityMedium,
					Category:       "antipattern",
					Description:    "Unnecessary context switch: " + call.APIName + " without draw calls since last switch",
					Evidence:       fmt.Sprintf("Line %d (GC: %s) followed by switch on line %d with no draws", lastMakeCurrentLine, lastMakeCurrentGC, call.LineNum),
					RootCauseChain: []string{"Context switches have overhead", "No rendering work between switches"},
					FixSuggestion:  "Ensure draw calls exist between context switches",
				})
			}
			lastMakeCurrentLine = call.LineNum
			lastMakeCurrentGC = call.GCAddr
			hasDrawSinceSwitch = false
			firstMakeCurrentSeen = true
		}
		if isDrawCall(call.APIName) {
			hasDrawSinceSwitch = true
		}
	}
	return findings
}

func (d *AntiPatternDetector) detectUniformsOnNullProgram(frame *core.FrameInfo) []core.Finding {
	var findings []core.Finding
	currentProgram := 0
	programZeroLine := 0

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		if call.APIName == "glUseProgram" {
			parts := strings.Fields(call.RawParams)
			if len(parts) >= 1 {
				currentProgram = parseHexOrDec(parts[0])
				if currentProgram == 0 {
					programZeroLine = call.LineNum
				}
			}
			continue
		}
		if currentProgram == 0 && isUniformCall(call.APIName) {
			findings = append(findings, core.Finding{
				Severity:       core.SeverityMedium,
				Category:       "antipattern",
				Description:    "Uniform call on disabled program: " + call.APIName + " after glUseProgram(0)",
				Evidence:       fmt.Sprintf("Line %d: %s called while no program active (glUseProgram(0) on line %d)", call.LineNum, call.APIName, programZeroLine),
				RootCauseChain: []string{"glUniform with no active program has no effect"},
				FixSuggestion:  "Call glUseProgram with valid program before glUniform calls",
			})
		}
	}
	return findings
}

func (d *AntiPatternDetector) countDrawWithoutProgram(frame *core.FrameInfo) int {
	currentProgram := -1 // -1 = unknown, 0 = null, >0 = valid
	count := 0

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		if call.APIName == "glUseProgram" {
			parts := strings.Fields(call.RawParams)
			if len(parts) >= 1 {
				currentProgram = parseHexOrDec(parts[0])
			}
			continue
		}
		// Only count if we've seen at least one glUseProgram (to avoid false positives at start)
		if currentProgram == 0 && isDrawCall(call.APIName) {
			count++
		}
	}
	return count
}

func isUniformCall(apiName string) bool {
	return strings.HasPrefix(apiName, "glUniform") || strings.HasPrefix(apiName, "glProgramUniform")
}

func isDrawCall(apiName string) bool {
	switch apiName {
	case "glDrawArrays", "glDrawElements", "glDrawRangeElements",
		"glDrawArraysInstanced", "glDrawElementsInstanced",
		"glMultiDrawArrays", "glMultiDrawElements",
		"glDrawArraysIndirect", "glDrawElementsIndirect":
		return true
	}
	return false
}

func itoaSimple(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func itoa(n int) string {
	return strings.TrimSpace(itoaSimple(n))
}
