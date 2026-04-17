package analyzer

import (
	"sort"
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
func (fa *FrameAnalyzer) GetFrameSummary() map[string]interface{} {
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

	return map[string]interface{}{
		"total_frames":  len(fa.log.Frames),
		"avg_time_us":   avgUs,
		"max_time_us":   maxUs,
		"min_time_us":   minUs,
		"total_time_us": totalUs,
	}
}
