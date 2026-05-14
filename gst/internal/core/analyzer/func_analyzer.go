package analyzer

import (
	"gst/internal/core"
	"sort"
	"strings"
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

	// Prefer per-frame summaries for indexed large logs. They contain aggregate
	// count/time without retaining every raw API call in memory.
	for _, frame := range fa.log.Frames {
		if len(frame.APICalls) == 0 && len(frame.APISummary) > 0 {
			for _, summary := range frame.APISummary {
				if stats, ok := fa.stats[summary.APIName]; ok {
					stats.CallCount += summary.Count
					stats.TotalTimeUs += summary.TimeUs
				} else {
					fa.stats[summary.APIName] = &core.FuncStats{
						FuncName:    summary.APIName,
						CallCount:   summary.Count,
						TotalTimeUs: summary.TimeUs,
					}
				}
			}
			continue
		}
		for _, call := range frame.APICalls {
			count := call.Count
			if count <= 0 {
				count = 1
			}
			if stats, ok := fa.stats[call.APIName]; ok {
				stats.CallCount += count
				stats.TotalTimeUs += call.TimeUs
			} else {
				fa.stats[call.APIName] = &core.FuncStats{
					FuncName:    call.APIName,
					CallCount:   count,
					TotalTimeUs: call.TimeUs,
				}
			}
		}
	}

	// 计算平均值并转为 slice
	results := make([]core.FuncStats, 0, len(fa.stats))
	for _, stats := range fa.stats {
		if stats.CallCount > 0 {
			stats.AvgTimeUs = stats.TotalTimeUs / int64(stats.CallCount)
		}
		results = append(results, *stats)
	}

	// 按总耗时降序
	sort.Slice(results, func(i, j int) bool {
		if results[i].TotalTimeUs == results[j].TotalTimeUs {
			if results[i].CallCount == results[j].CallCount {
				return results[i].FuncName < results[j].FuncName
			}
			return results[i].CallCount > results[j].CallCount
		}
		return results[i].TotalTimeUs > results[j].TotalTimeUs
	})

	return results
}

// GetFuncSummary 获取函数统计摘要
func (fa *FuncAnalyzer) GetFuncSummary() *core.FuncSummary {
	stats := fa.Analyze()
	if stats == nil {
		return nil
	}

	var totalCalls int
	var totalTime int64
	topN := 10
	if len(stats) < topN {
		topN = len(stats)
	}

	for _, s := range stats {
		totalCalls += s.CallCount
		totalTime += s.TotalTimeUs
	}

	return &core.FuncSummary{
		TotalFunctions: len(stats),
		TotalCalls:     totalCalls,
		TotalTimeUs:    totalTime,
		TopFunctions:   stats[:topN],
	}
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
