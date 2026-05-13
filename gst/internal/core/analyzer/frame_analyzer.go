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
	if len(funcStats) > 0 {
		topBottleneck = fmt.Sprintf("%s (%.2f ms, %d次调用)",
			funcStats[0].FuncName,
			float64(funcStats[0].TotalTimeUs)/1000.0,
			funcStats[0].CallCount)
	}

	// 分类
	var bottleneckType core.BottleneckType
	var confidence float64
	var details []string

	if cv > 0.5 {
		bottleneckType = core.BottleneckUnstable
		confidence = cv
		details = append(details, fmt.Sprintf("帧时间变异系数%.2f, 帧率不稳定", cv))
	} else if swapRatio > 0.6 {
		bottleneckType = core.BottleneckGPU
		confidence = swapRatio
		details = append(details, fmt.Sprintf("SwapBuffer等待占比%.1f%%, GPU可能跟不上", swapRatio*100))
	} else if apiRatio > 0.7 {
		bottleneckType = core.BottleneckCPU
		confidence = apiRatio
		details = append(details, fmt.Sprintf("API调用耗时占比%.1f%%, CPU可能是瓶颈", apiRatio*100))
	} else {
		bottleneckType = core.BottleneckBalanced
		confidence = 1.0 - math.Abs(swapRatio-apiRatio)
		details = append(details, "CPU和GPU负载较为均衡")
	}

	if topBottleneck != "" {
		details = append(details, fmt.Sprintf("最大瓶颈函数: %s", topBottleneck))
	}

	return &core.BottleneckAnalysis{
		Type:          bottleneckType,
		Confidence:    confidence,
		SwapRatio:     swapRatio,
		APIRatio:      apiRatio,
		Stability:     cv,
		TopBottleneck: topBottleneck,
		Details:       strings.Join(details, "; "),
	}
}
