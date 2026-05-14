package analyzer

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"gst/internal/core"
)

// FrameAnalyzer 帧分析器
type FrameAnalyzer struct {
	log *core.ParsedLog
}

// NewFrameAnalyzer 创建帧分析器
func NewFrameAnalyzer(log *core.ParsedLog) *FrameAnalyzer {
	return &FrameAnalyzer{log: log}
}

// FindTopSlowFrames 找出最慢的N帧
func (fa *FrameAnalyzer) FindTopSlowFrames(n int) []core.FrameInfo {
	if fa.log == nil || n <= 0 {
		return nil
	}

	frames := make([]core.FrameInfo, len(fa.log.Frames))
	copy(frames, fa.log.Frames)

	sort.Slice(frames, func(i, j int) bool {
		if frames[i].TotalTimeUs == frames[j].TotalTimeUs {
			di := countFrameDrawCalls(frames[i])
			dj := countFrameDrawCalls(frames[j])
			if di == dj {
				return len(frames[i].APICalls) > len(frames[j].APICalls)
			}
			return di > dj
		}
		return frames[i].TotalTimeUs > frames[j].TotalTimeUs
	})

	if n > len(frames) {
		n = len(frames)
	}
	return frames[:n]
}

// GetFrameSummary 获取帧统计摘要
func (fa *FrameAnalyzer) GetFrameSummary() *core.FrameSummary {
	if fa.log == nil || len(fa.log.Frames) == 0 {
		return nil
	}

	var totalUs int64
	maxUs := fa.log.Frames[0].TotalTimeUs
	minUs := fa.log.Frames[0].TotalTimeUs

	for _, f := range fa.log.Frames {
		totalUs += f.TotalTimeUs
		if f.TotalTimeUs > maxUs {
			maxUs = f.TotalTimeUs
		}
		if f.TotalTimeUs < minUs {
			minUs = f.TotalTimeUs
		}
	}

	avgUs := totalUs / int64(len(fa.log.Frames))

	return &core.FrameSummary{
		TotalFrames: len(fa.log.Frames),
		AvgTimeUs:   avgUs,
		MaxTimeUs:   maxUs,
		MinTimeUs:   minUs,
		TotalTimeUs: totalUs,
	}
}

// AnalyzeBottleneck 分析性能瓶颈
func (fa *FrameAnalyzer) AnalyzeBottleneck() *core.BottleneckAnalysis {
	if fa.log == nil || len(fa.log.Frames) == 0 {
		return nil
	}
	if !hasFrameTiming(fa.log.Frames) {
		funcAna := NewFuncAnalyzer(fa.log)
		funcStats := funcAna.Analyze()
		topBottleneck := ""
		if len(funcStats) > 0 {
			topBottleneck = formatFuncHotspot(funcStats[0])
		}
		return &core.BottleneckAnalysis{
			Type:          core.BottleneckUnknown,
			HasTiming:     false,
			Confidence:    0,
			TopBottleneck: topBottleneck,
			Details:       "当前 rawtrace 未提供 frame cost/profile 耗时，只能基于调用密度判断热点，不能给出 CPU/GPU 时间占比。",
		}
	}

	// 计算 SwapBuffer vs API 时间占比
	var totalSwapUs, totalAPIUs int64
	for _, f := range fa.log.Frames {
		totalSwapUs += f.SwapBufferTimeUs
		totalAPIUs += f.APITotalTimeUs
	}
	totalUs := totalSwapUs + totalAPIUs
	if totalUs == 0 {
		totalUs = 1
	}

	swapRatio := float64(totalSwapUs) / float64(totalUs)
	apiRatio := float64(totalAPIUs) / float64(totalUs)

	// 帧时间变异系数 (CV = stddev/mean)
	var avgTime float64
	for _, f := range fa.log.Frames {
		avgTime += float64(f.TotalTimeUs)
	}
	avgTime /= float64(len(fa.log.Frames))

	var cv float64
	if avgTime > 0 {
		var variance float64
		for _, f := range fa.log.Frames {
			diff := float64(f.TotalTimeUs) - avgTime
			variance += diff * diff
		}
		variance /= float64(len(fa.log.Frames))
		stddev := math.Sqrt(variance)
		cv = stddev / avgTime
	}

	// 最耗时函数
	funcAna := NewFuncAnalyzer(fa.log)
	funcStats := funcAna.Analyze()
	topBottleneck := ""
	hasFunctionTiming := false
	if len(funcStats) > 0 {
		hasFunctionTiming = funcStats[0].TotalTimeUs > 0
		topBottleneck = formatFuncHotspot(funcStats[0])
	}

	// 分类
	var bottleneckType core.BottleneckType
	var confidence float64
	var details []string

	if cv > 0.5 {
		bottleneckType = core.BottleneckUnstable
		confidence = clamp01(cv)
		details = append(details, fmt.Sprintf("帧时间变异系数%.2f, 帧率不稳定", cv))
	} else if swapRatio > 0.6 {
		bottleneckType = core.BottleneckGPU
		confidence = clamp01(swapRatio)
		details = append(details, fmt.Sprintf("SwapBuffer等待占比%.1f%%, GPU可能跟不上", swapRatio*100))
	} else if apiRatio > 0.7 {
		bottleneckType = core.BottleneckCPU
		confidence = clamp01(apiRatio)
		details = append(details, fmt.Sprintf("API调用耗时占比%.1f%%, CPU可能是瓶颈", apiRatio*100))
	} else {
		bottleneckType = core.BottleneckBalanced
		confidence = clamp01(1.0 - math.Abs(swapRatio-apiRatio))
		details = append(details, "CPU和GPU负载较为均衡")
	}

	if topBottleneck != "" {
		if hasFunctionTiming {
			details = append(details, fmt.Sprintf("最大瓶颈函数: %s", topBottleneck))
		} else {
			details = append(details, fmt.Sprintf("调用次数最高函数: %s", topBottleneck))
			details = append(details, "rawtrace 未提供函数级耗时，函数热点按调用次数排序")
		}
	}

	return &core.BottleneckAnalysis{
		Type:          bottleneckType,
		HasTiming:     true,
		Confidence:    confidence,
		SwapRatio:     swapRatio,
		APIRatio:      apiRatio,
		Stability:     cv,
		TopBottleneck: topBottleneck,
		Details:       strings.Join(details, "; "),
	}
}

func hasFrameTiming(frames []core.FrameInfo) bool {
	for _, frame := range frames {
		if frame.HasTiming || frame.TotalTimeUs > 0 {
			return true
		}
	}
	return false
}

func countFrameDrawCalls(frame core.FrameInfo) int {
	if frame.DrawCallCount > 0 && len(frame.APICalls) == 0 {
		return frame.DrawCallCount
	}
	count := 0
	for _, call := range frame.APICalls {
		if isDrawCall(call.APIName) {
			count += call.Count
			if call.Count == 0 {
				count++
			}
		}
	}
	return count
}

func formatFuncHotspot(stat core.FuncStats) string {
	if stat.TotalTimeUs > 0 {
		return fmt.Sprintf("%s (%.2f ms, %d次调用)", stat.FuncName, float64(stat.TotalTimeUs)/1000.0, stat.CallCount)
	}
	return fmt.Sprintf("%s (%d次调用)", stat.FuncName, stat.CallCount)
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
