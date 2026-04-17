package search

import (
	"gst/internal/core"
)

// TimeRangeSearch 时间段检索
type TimeRangeSearch struct{}

func NewTimeRangeSearch() *TimeRangeSearch {
	return &TimeRangeSearch{}
}

// Search 在指定时间范围内搜索帧
// startUs: 开始时间（微秒）
// endUs: 结束时间（微秒）
// 返回: 该时间段内的所有帧
func (trs *TimeRangeSearch) Search(log *core.ParsedLog, startUs, endUs int64) []core.APILogEntry {
	if log == nil {
		return nil
	}

	var results []core.APILogEntry

	for _, frame := range log.Frames {
		// Check if frame's total time is in range
		if frame.TotalTimeUs >= startUs && frame.TotalTimeUs <= endUs {
			results = append(results, frame.APICalls...)
		}
	}

	return results
}

// SearchByFrameRange 在指定帧范围内搜索
func (trs *TimeRangeSearch) SearchByFrameRange(log *core.ParsedLog, startFrame, endFrame int) []core.FrameInfo {
	if log == nil {
		return nil
	}

	var results []core.FrameInfo

	for _, frame := range log.Frames {
		if frame.FrameNum >= startFrame && frame.FrameNum <= endFrame {
			results = append(results, frame)
		}
	}

	return results
}