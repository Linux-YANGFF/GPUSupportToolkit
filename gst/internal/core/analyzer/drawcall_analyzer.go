package analyzer

import (
	"sort"
	"strings"

	"gst/internal/core"
)

// DrawCallAnalyzer Draw Call 分析器
type DrawCallAnalyzer struct {
	log *core.ParsedLog
}

// NewDrawCallAnalyzer 创建 Draw Call 分析器
func NewDrawCallAnalyzer(log *core.ParsedLog) *DrawCallAnalyzer {
	return &DrawCallAnalyzer{log: log}
}

// draw call 类型分类
func classifyDrawCall(apiName string) string {
	switch {
	case strings.Contains(apiName, "DrawArraysInstanced") || strings.Contains(apiName, "DrawElementsInstanced"):
		return "instanced"
	case strings.Contains(apiName, "DrawArraysIndirect") || strings.Contains(apiName, "DrawElementsIndirect"):
		return "indirect"
	case strings.Contains(apiName, "DispatchCompute"):
		return "compute"
	case strings.Contains(apiName, "DrawArrays"):
		return "draw_arrays"
	case strings.Contains(apiName, "DrawElements"):
		return "draw_elements"
	case strings.Contains(apiName, "DrawRangeElements"):
		return "draw_elements"
	default:
		return ""
	}
}

func isDrawCall(apiName string) bool {
	return strings.HasPrefix(apiName, "glDraw") || strings.HasPrefix(apiName, "glDispatchCompute")
}

// AnalyzePerFrame 分析每帧的 draw call 统计
func (dca *DrawCallAnalyzer) AnalyzePerFrame() []core.DrawCallStats {
	if dca.log == nil {
		return nil
	}

	var results []core.DrawCallStats
	for _, frame := range dca.log.Frames {
		stats := core.DrawCallStats{
			FrameNum: frame.FrameNum,
			TimeUs:   frame.TotalTimeUs,
		}

		for _, call := range frame.APICalls {
			if !isDrawCall(call.APIName) {
				continue
			}
			stats.TotalDrawCalls++
			switch classifyDrawCall(call.APIName) {
			case "draw_arrays":
				stats.DrawArraysCount++
			case "draw_elements":
				stats.DrawElementsCount++
			case "instanced":
				stats.InstancedCount++
			case "indirect":
				stats.IndirectCount++
			case "compute":
				stats.ComputeCount++
			}
		}

		results = append(results, stats)
	}
	return results
}

// GetSummary 获取 draw call 综合统计
func (dca *DrawCallAnalyzer) GetSummary() *core.DrawCallSummary {
	frames := dca.AnalyzePerFrame()
	if len(frames) == 0 {
		return nil
	}

	var totalDrawCalls int
	byType := make(map[string]int)

	for _, f := range frames {
		totalDrawCalls += f.TotalDrawCalls
		byType["draw_arrays"] += f.DrawArraysCount
		byType["draw_elements"] += f.DrawElementsCount
		byType["instanced"] += f.InstancedCount
		byType["indirect"] += f.IndirectCount
		byType["compute"] += f.ComputeCount
	}

	return &core.DrawCallSummary{
		TotalDrawCalls:      totalDrawCalls,
		DrawCallsPerFrameAvg: float64(totalDrawCalls) / float64(len(frames)),
		ByType:              byType,
		Frames:              frames,
	}
}

// FindHotFrames 找出 draw call 密度最高的帧
func (dca *DrawCallAnalyzer) FindHotFrames(n int) []core.DrawCallStats {
	frames := dca.AnalyzePerFrame()
	if len(frames) == 0 {
		return nil
	}

	sort.Slice(frames, func(i, j int) bool {
		return frames[i].TotalDrawCalls > frames[j].TotalDrawCalls
	})

	if n > len(frames) {
		n = len(frames)
	}
	return frames[:n]
}
