package bug

import (
	"fmt"
	"strings"

	"gst/internal/core"
)

// UnbatchedDrawCallDetector 检测未批量的 draw call（连续小批量绘制）
type UnbatchedDrawCallDetector struct{}

func NewUnbatchedDrawCallDetector() *UnbatchedDrawCallDetector {
	return &UnbatchedDrawCallDetector{}
}

func (d *UnbatchedDrawCallDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	var findings []core.Finding

	for _, frame := range log.Frames {
		finding := d.analyzeFrame(&frame)
		if finding != nil {
			findings = append(findings, *finding)
		}
	}
	return findings
}

func (d *UnbatchedDrawCallDetector) analyzeFrame(frame *core.FrameInfo) *core.Finding {
	// 检测连续的 glDrawArrays(mode, 0, N) 其中 N 很小（如 2）
	const (
		smallVertexThreshold = 6  // 小于此值认为是小批量
		consecutiveThreshold = 50  // 连续超过此数量才报告
	)

	var (
		consecutiveCount int
		totalVertices    int
		drawMode         string
		startLine        int
	)

	for i := range frame.APICalls {
		call := &frame.APICalls[i]

		if !strings.HasPrefix(call.APIName, "glDrawArrays") {
			if consecutiveCount >= consecutiveThreshold {
				return &core.Finding{
					Severity:       core.SeverityHigh,
					Category:       "performance",
					Description:    fmt.Sprintf("Unbatched draw calls: %d consecutive glDrawArrays(%s) with %d avg vertices in frame %d", consecutiveCount, drawMode, totalVertices/consecutiveCount, frame.FrameNum),
					Evidence:       fmt.Sprintf("Line %d-%d: %d draw calls with small vertex counts (avg %d), should be batched into fewer calls", startLine, call.LineNum, consecutiveCount, totalVertices/consecutiveCount),
					RootCauseChain: []string{"Each draw call has driver overhead (state validation, command buffer submission)", "Small draw calls prevent GPU from achieving peak throughput"},
					FixSuggestion:  "Batch geometry into a single VBO and draw with one glDrawArrays call, or use instanced rendering",
				}
			}
			consecutiveCount = 0
			totalVertices = 0
			drawMode = ""
			startLine = 0
			continue
		}

		// 解析 glDrawArrays mode first count
		vertexCount := extractVertexCount(call.RawParams)
		mode := extractDrawMode(call.RawParams)

		if vertexCount > 0 && vertexCount <= smallVertexThreshold {
			if consecutiveCount == 0 {
				startLine = call.LineNum
				drawMode = mode
			}
			consecutiveCount++
			totalVertices += vertexCount
		} else {
			if consecutiveCount >= consecutiveThreshold {
				return &core.Finding{
					Severity:       core.SeverityHigh,
					Category:       "performance",
					Description:    fmt.Sprintf("Unbatched draw calls: %d consecutive glDrawArrays(%s) with %d avg vertices in frame %d", consecutiveCount, drawMode, totalVertices/consecutiveCount, frame.FrameNum),
					Evidence:       fmt.Sprintf("Line %d-%d: %d draw calls with small vertex counts, should be batched", startLine, call.LineNum, consecutiveCount),
					RootCauseChain: []string{"Each draw call has driver overhead", "Small draw calls prevent GPU peak throughput"},
					FixSuggestion:  "Batch geometry into a single VBO and draw with one glDrawArrays call",
				}
			}
			consecutiveCount = 0
			totalVertices = 0
			drawMode = ""
		}
	}

	// Check tail
	if consecutiveCount >= consecutiveThreshold {
		return &core.Finding{
			Severity:       core.SeverityHigh,
			Category:       "performance",
			Description:    fmt.Sprintf("Unbatched draw calls: %d consecutive glDrawArrays(%s) with %d avg vertices in frame %d", consecutiveCount, drawMode, totalVertices/consecutiveCount, frame.FrameNum),
			Evidence:       fmt.Sprintf("Line %d+: %d draw calls with small vertex counts, should be batched", startLine, consecutiveCount),
			RootCauseChain: []string{"Each draw call has driver overhead", "Small draw calls prevent GPU peak throughput"},
			FixSuggestion:  "Batch geometry into a single VBO and draw with one glDrawArrays call",
		}
	}

	return nil
}

func extractVertexCount(params string) int {
	parts := strings.Fields(params)
	if len(parts) >= 3 {
		return parseHexOrDec(parts[2])
	}
	return 0
}

func extractDrawMode(params string) string {
	parts := strings.Fields(params)
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}
