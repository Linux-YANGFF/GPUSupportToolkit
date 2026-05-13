package bug

import (
	"fmt"

	"gst/internal/core"
)

// ExcessiveGetErrorDetector 检测过量的 glGetError 调用
type ExcessiveGetErrorDetector struct{}

func NewExcessiveGetErrorDetector() *ExcessiveGetErrorDetector {
	return &ExcessiveGetErrorDetector{}
}

func (d *ExcessiveGetErrorDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	var findings []core.Finding

	totalGetError := 0
	frameCounts := make(map[int]int) // frameNum -> count

	for _, frame := range log.Frames {
		count := 0
		for _, call := range frame.APICalls {
			if call.APIName == "glGetError" {
				count++
				totalGetError++
			}
		}
		if count > 100 {
			frameCounts[frame.FrameNum] = count
		}
	}

	if totalGetError > 500 {
		desc := fmt.Sprintf("Excessive glGetError calls: %d total across %d frames", totalGetError, len(log.Frames))
		evidence := fmt.Sprintf("glGetError is a synchronous pipeline stall that forces CPU-GPU sync. Found %d calls", totalGetError)

		if len(frameCounts) > 0 {
			worstFrame := 0
			worstCount := 0
			for fn, c := range frameCounts {
				if c > worstCount {
					worstCount = c
					worstFrame = fn
				}
			}
			evidence += fmt.Sprintf(". Worst frame: %d with %d glGetError calls", worstFrame, worstCount)
		}

		findings = append(findings, core.Finding{
			Severity:       core.SeverityHigh,
			Category:       "performance",
			Description:    desc,
			Evidence:       evidence,
			RootCauseChain: []string{"glGetError forces a full pipeline flush, stalling CPU until GPU finishes all pending work", "In release builds, glGetError should be disabled or wrapped in debug-only macros"},
			FixSuggestion:  "Remove glGetError from release builds, or use GL_KHR_debug extension / debug output callback instead of polling",
		})
	}

	return findings
}
