package bug

import (
	"fmt"
	"strings"

	"gst/internal/core"
)

var glErrorNames = map[string]string{
	"0x0500": "GL_INVALID_ENUM",
	"0x0501": "GL_INVALID_VALUE",
	"0x0502": "GL_INVALID_OPERATION",
	"0x0503": "GL_STACK_OVERFLOW",
	"0x0504": "GL_STACK_UNDERFLOW",
	"0x0505": "GL_OUT_OF_MEMORY",
	"0x0506": "GL_INVALID_FRAMEBUFFER_OPERATION",
	"0x500":  "GL_INVALID_ENUM",
	"0x501":  "GL_INVALID_VALUE",
	"0x502":  "GL_INVALID_OPERATION",
	"0x503":  "GL_STACK_OVERFLOW",
	"0x504":  "GL_STACK_UNDERFLOW",
	"0x505":  "GL_OUT_OF_MEMORY",
	"0x506":  "GL_INVALID_FRAMEBUFFER_OPERATION",
}

type DriverErrorDetector struct{}

func (d *DriverErrorDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	var findings []core.Finding

	for frameIdx := range log.Frames {
		frame := &log.Frames[frameIdx]
		findings = append(findings, d.diagnoseFrame(frame)...)
	}

	return findings
}

func (d *DriverErrorDetector) diagnoseFrame(frame *core.FrameInfo) []core.Finding {
	var findings []core.Finding

	glGetErrorPerContext := map[string]bool{}

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		if call.APIName == "glGetError" {
			glGetErrorPerContext[call.GCAddr] = true
		}
	}

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		if !call.IsError || call.APIName != "__glSetError" {
			continue
		}

		errorName := d.resolveErrorName(call.ErrorCode)

		triggerCall := d.findTriggerCall(frame, i, call.GCAddr)

		appDidCheck := glGetErrorPerContext[call.GCAddr]

		finding := d.buildFinding(call, errorName, triggerCall, appDidCheck, frame.FrameNum)
		findings = append(findings, finding)
	}

	return findings
}

func (d *DriverErrorDetector) resolveErrorName(code string) string {
	normalized := strings.ToLower(code)
	if name, ok := glErrorNames[normalized]; ok {
		return name
	}
	return "GL_UNKNOWN_ERROR"
}

func (d *DriverErrorDetector) findTriggerCall(frame *core.FrameInfo, errorIdx int, gcAddr string) *core.APILogEntry {
	for j := errorIdx - 1; j >= 0; j-- {
		call := &frame.APICalls[j]
		if call.APIName == "__glSetError" {
			continue
		}
		if gcAddr != "" && call.GCAddr != gcAddr {
			continue
		}
		return call
	}
	return nil
}

func (d *DriverErrorDetector) buildFinding(errorCall *core.APILogEntry, errorName string, triggerCall *core.APILogEntry, appDidCheck bool, frameNum int) core.Finding {
	var evidence strings.Builder
	evidence.WriteString(fmt.Sprintf("Driver error %s (code=%s) at line %d",
		errorName, errorCall.ErrorCode, errorCall.LineNum))

	if triggerCall != nil {
		evidence.WriteString(fmt.Sprintf("; triggering call: %s at line %d",
			triggerCall.APIName, triggerCall.LineNum))
	} else {
		evidence.WriteString("; no triggering call identified")
	}

	if gc := errorCall.GCAddr; gc != "" {
		evidence.WriteString(fmt.Sprintf("; gc=%s", gc))
	}

	desc := fmt.Sprintf("Driver reported error: %s", errorName)
	if triggerCall != nil {
		desc = fmt.Sprintf("Driver reported %s after %s", errorName, triggerCall.APIName)
	}

	rootCause := []string{}
	if triggerCall != nil {
		rootCause = append(rootCause, fmt.Sprintf("%s at line %d triggered %s (code=%s)",
			triggerCall.APIName, triggerCall.LineNum, errorName, errorCall.ErrorCode))
	} else {
		rootCause = append(rootCause, fmt.Sprintf("Driver reported %s (code=%s) at line %d with no clear trigger",
			errorName, errorCall.ErrorCode, errorCall.LineNum))
	}

	if !appDidCheck {
		rootCause = append(rootCause, "Application does not appear to call glGetError — errors may be silently ignored")
	}

	fixSuggestion := "Check GL state validity before the triggering call"
	if triggerCall != nil {
		fixSuggestion = fmt.Sprintf("Validate GL state before calling %s; ensure correct enum values, buffer bindings, and shader/program state", triggerCall.APIName)
	}
	if !appDidCheck {
		fixSuggestion += ". Add glGetError() polling after GPU calls to detect and handle errors early"
	}

	return core.Finding{
		Severity:       core.SeverityCritical,
		Category:       "driver_error",
		Description:    desc,
		Evidence:       evidence.String(),
		RootCauseChain: rootCause,
		FixSuggestion:  fixSuggestion,
	}
}
