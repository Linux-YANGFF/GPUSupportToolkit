package analyzer

import (
	"sort"
	"strings"
	"gst/internal/core"
)

// FuncAnalyzer 函数分析器
type FuncAnalyzer struct {
	log   *core.ParsedLog
	stats map[string]*core.FuncStats
}

// NewFuncAnalyzer 创建函数分析器
func NewFuncAnalyzer(log *core.ParsedLog) *FuncAnalyzer {
	return &FuncAnalyzer{log: log, stats: make(map[string]*core.FuncStats)}
}

// Analyze 分析所有函数调用
func (fa *FuncAnalyzer) Analyze() []core.FuncStats {
	if fa.log == nil {
		return nil
	}

	fa.stats = make(map[string]*core.FuncStats)

	// 遍历所有帧的所有API调用
	for _, frame := range fa.log.Frames {
		for _, call := range frame.APICalls {
			if stats, ok := fa.stats[call.APIName]; ok {
				stats.CallCount++
				stats.TotalTimeUs += call.TimeUs
			} else {
				fa.stats[call.APIName] = &core.FuncStats{
					FuncName:    call.APIName,
					CallCount:   1,
					TotalTimeUs: call.TimeUs,
				}
			}
		}
	}

	// 计算平均值并转为 slice
	results := make([]core.FuncStats, 0, len(fa.stats))
	for _, stats := range fa.stats {
		stats.AvgTimeUs = stats.TotalTimeUs / int64(stats.CallCount)
		results = append(results, *stats)
	}

	// 按总耗时降序
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalTimeUs > results[j].TotalTimeUs
	})

	return results
}

// FilterByPrefix 按前缀过滤函数
func (fa *FuncAnalyzer) FilterByPrefix(prefix string) []core.FuncStats {
	all := fa.Analyze()
	if prefix == "" {
		return all
	}

	var filtered []core.FuncStats
	for _, f := range all {
		if strings.HasPrefix(f.FuncName, prefix) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
