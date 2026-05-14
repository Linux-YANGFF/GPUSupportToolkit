package glstats

import (
	"sort"

	"gst/internal/core"
)

type APICounter struct {
	APIName     string `json:"api_name"`
	Category    string `json:"category"`
	Label       string `json:"label"`
	Family      string `json:"family"`
	Key         string `json:"key,omitempty"`
	Count       int    `json:"count"`
	TimeUs      int64  `json:"time_us"`
	AvgTimeUs   int64  `json:"avg_time_us"`
	Source      string `json:"source"`
	HasTiming   bool   `json:"has_timing"`
	RawSequence int    `json:"raw_sequence,omitempty"`
}

type CategoryCounter struct {
	Category  string       `json:"category"`
	Label     string       `json:"label"`
	Count     int          `json:"count"`
	TimeUs    int64        `json:"time_us"`
	AvgTimeUs int64        `json:"avg_time_us"`
	TopAPIs   []APICounter `json:"top_apis"`
}

type FrameStats struct {
	FrameNum         int               `json:"frame_num"`
	StartLine        int               `json:"start_line"`
	EndLine          int               `json:"end_line"`
	TotalTimeUs      int64             `json:"total_time_us"`
	SwapBufferTimeUs int64             `json:"swap_buffer_time_us"`
	APITotalTimeUs   int64             `json:"api_total_time_us"`
	APICallCount     int               `json:"api_call_count"`
	RawAPICallCount  int               `json:"raw_api_call_count"`
	DrawCallCount    int               `json:"draw_call_count"`
	HasTiming        bool              `json:"has_timing"`
	TimingSource     string            `json:"timing_source"`
	StatsSource      string            `json:"stats_source"`
	CategoryStats    []CategoryCounter `json:"category_stats"`
	KeyAPIs          []APICounter      `json:"key_apis"`
	TopAPIs          []APICounter      `json:"top_apis"`
}

type CaseStats struct {
	FrameCount      int               `json:"frame_count"`
	TotalTimeUs     int64             `json:"total_time_us"`
	FPS             float64           `json:"fps"`
	HasTiming       bool              `json:"has_timing"`
	CategoryStats   []CategoryCounter `json:"category_stats"`
	KeyAPIs         []APICounter      `json:"key_apis"`
	TopAPIs         []APICounter      `json:"top_apis"`
	TopFramesByDraw []FrameStats      `json:"top_frames_by_draw"`
	TopFramesByTime []FrameStats      `json:"top_frames_by_time"`
	AIContract      AIContract        `json:"ai_contract"`
}

type AIContract struct {
	SourceOfTruth string   `json:"source_of_truth"`
	Units         []string `json:"units"`
	Notes         []string `json:"notes"`
}

func AnalyzeFrame(frame core.FrameInfo) FrameStats {
	apiCounters := frameAPICounters(frame)
	categoryStats := categoryCounters(apiCounters, 3)
	keyAPIs := keyCounters(apiCounters)
	topAPIs := topCounters(apiCounters, 12)
	rawCount := rawAPICount(frame)
	drawCount := drawCallCountFromCounters(apiCounters)
	if frame.DrawCallCount > 0 {
		drawCount = frame.DrawCallCount
	}

	return FrameStats{
		FrameNum:         frame.FrameNum,
		StartLine:        frame.StartLine,
		EndLine:          frame.EndLine,
		TotalTimeUs:      frame.TotalTimeUs,
		SwapBufferTimeUs: frame.SwapBufferTimeUs,
		APITotalTimeUs:   frame.APITotalTimeUs,
		APICallCount:     summaryAPICount(apiCounters, frame),
		RawAPICallCount:  rawCount,
		DrawCallCount:    drawCount,
		HasTiming:        frame.HasTiming || frame.TotalTimeUs > 0,
		TimingSource:     timingSource(frame),
		StatsSource:      statsSource(frame),
		CategoryStats:    categoryStats,
		KeyAPIs:          keyAPIs,
		TopAPIs:          topAPIs,
	}
}

func AnalyzeCase(log *core.ParsedLog, topN int) CaseStats {
	if log == nil {
		return CaseStats{}
	}
	if topN <= 0 {
		topN = 10
	}

	allCounters := make(map[string]APICounter)
	frames := make([]FrameStats, 0, len(log.Frames))
	hasTiming := false
	for _, frame := range log.Frames {
		stats := AnalyzeFrame(frame)
		frames = append(frames, stats)
		if stats.HasTiming {
			hasTiming = true
		}
		for _, api := range frameAPICounters(frame) {
			accumulateCounter(allCounters, api)
		}
	}

	all := mapToCounters(allCounters)
	sortCounters(all)
	topFramesByDraw := append([]FrameStats(nil), frames...)
	sort.Slice(topFramesByDraw, func(i, j int) bool {
		if topFramesByDraw[i].DrawCallCount == topFramesByDraw[j].DrawCallCount {
			return topFramesByDraw[i].TotalTimeUs > topFramesByDraw[j].TotalTimeUs
		}
		return topFramesByDraw[i].DrawCallCount > topFramesByDraw[j].DrawCallCount
	})
	topFramesByTime := append([]FrameStats(nil), frames...)
	sort.Slice(topFramesByTime, func(i, j int) bool {
		if topFramesByTime[i].TotalTimeUs == topFramesByTime[j].TotalTimeUs {
			return topFramesByTime[i].DrawCallCount > topFramesByTime[j].DrawCallCount
		}
		return topFramesByTime[i].TotalTimeUs > topFramesByTime[j].TotalTimeUs
	})

	if len(topFramesByDraw) > topN {
		topFramesByDraw = topFramesByDraw[:topN]
	}
	if len(topFramesByTime) > topN {
		topFramesByTime = topFramesByTime[:topN]
	}

	return CaseStats{
		FrameCount:      len(log.Frames),
		TotalTimeUs:     log.TotalTimeUs,
		FPS:             log.FPS,
		HasTiming:       hasTiming,
		CategoryStats:   categoryCounters(all, 5),
		KeyAPIs:         keyCounters(all),
		TopAPIs:         limitCounters(all, topN),
		TopFramesByDraw: topFramesByDraw,
		TopFramesByTime: topFramesByTime,
		AIContract: AIContract{
			SourceOfTruth: "The original apitrace log is the source of truth; returned line and offset ranges point back to that log.",
			Units:         []string{"time_us is microseconds", "line numbers are original log line numbers", "counts are per-frame aggregated profile counts when available"},
			Notes: []string{
				"Raw API sequence entries do not imply per-call timing unless time_us is present.",
				"Profile aggregate blocks are attributed to the frame immediately preceding swapBuffers.",
				"AI consumers should request paginated frame APIs or drawcalls for evidence instead of loading the whole log.",
			},
		},
	}
}

func frameAPICounters(frame core.FrameInfo) []APICounter {
	if len(frame.APISummary) > 0 {
		counters := make([]APICounter, 0, len(frame.APISummary))
		for _, summary := range frame.APISummary {
			if summary == nil || summary.APIName == "" {
				continue
			}
			counters = append(counters, newCounter(summary.APIName, summary.Count, summary.TimeUs, "profile_summary", 0))
		}
		sortCounters(counters)
		return counters
	}

	counters := make(map[string]APICounter)
	for _, call := range frame.APICalls {
		if call.APIName == "" {
			continue
		}
		count := call.Count
		if count <= 0 {
			count = 1
		}
		accumulateCounter(counters, newCounter(call.APIName, count, call.TimeUs, "raw_sequence", 1))
	}
	result := mapToCounters(counters)
	sortCounters(result)
	return result
}

func newCounter(apiName string, count int, timeUs int64, source string, rawSequence int) APICounter {
	if count <= 0 {
		count = 1
	}
	class := Classify(apiName)
	counter := APICounter{
		APIName:     apiName,
		Category:    class.Category,
		Label:       class.Label,
		Family:      class.Family,
		Key:         class.Key,
		Count:       count,
		TimeUs:      timeUs,
		Source:      source,
		HasTiming:   timeUs > 0,
		RawSequence: rawSequence,
	}
	if count > 0 {
		counter.AvgTimeUs = timeUs / int64(count)
	}
	return counter
}

func categoryCounters(apiCounters []APICounter, topN int) []CategoryCounter {
	if topN <= 0 {
		topN = 3
	}
	byCategory := make(map[string][]APICounter)
	for _, counter := range apiCounters {
		byCategory[counter.Category] = append(byCategory[counter.Category], counter)
	}

	results := make([]CategoryCounter, 0, len(byCategory))
	for category, counters := range byCategory {
		sortCounters(counters)
		total := CategoryCounter{
			Category: category,
			Label:    CategoryLabel(category),
			TopAPIs:  limitCounters(counters, topN),
		}
		for _, counter := range counters {
			total.Count += counter.Count
			total.TimeUs += counter.TimeUs
		}
		if total.Count > 0 {
			total.AvgTimeUs = total.TimeUs / int64(total.Count)
		}
		results = append(results, total)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].TimeUs == results[j].TimeUs {
			if results[i].Count == results[j].Count {
				return results[i].Category < results[j].Category
			}
			return results[i].Count > results[j].Count
		}
		return results[i].TimeUs > results[j].TimeUs
	})
	return results
}

func keyCounters(counters []APICounter) []APICounter {
	byKey := make(map[string]APICounter)
	for _, counter := range counters {
		if counter.Key == "" {
			continue
		}
		accumulateCounter(byKey, counter)
	}
	results := mapToCounters(byKey)
	sortCounters(results)
	return results
}

func topCounters(counters []APICounter, n int) []APICounter {
	copied := append([]APICounter(nil), counters...)
	sortCounters(copied)
	return limitCounters(copied, n)
}

func limitCounters(counters []APICounter, n int) []APICounter {
	if n <= 0 || len(counters) <= n {
		return counters
	}
	return counters[:n]
}

func sortCounters(counters []APICounter) {
	sort.Slice(counters, func(i, j int) bool {
		if counters[i].TimeUs == counters[j].TimeUs {
			if counters[i].Count == counters[j].Count {
				return counters[i].APIName < counters[j].APIName
			}
			return counters[i].Count > counters[j].Count
		}
		return counters[i].TimeUs > counters[j].TimeUs
	})
}

func accumulateCounter(target map[string]APICounter, incoming APICounter) {
	key := incoming.APIName
	if incoming.Key != "" {
		key = incoming.Key
	}
	existing, ok := target[key]
	if !ok {
		if incoming.Key != "" {
			incoming.APIName = incoming.Key
		}
		target[key] = incoming
		return
	}
	existing.Count += incoming.Count
	existing.TimeUs += incoming.TimeUs
	existing.RawSequence += incoming.RawSequence
	existing.HasTiming = existing.HasTiming || incoming.HasTiming
	if existing.Count > 0 {
		existing.AvgTimeUs = existing.TimeUs / int64(existing.Count)
	}
	target[key] = existing
}

func mapToCounters(items map[string]APICounter) []APICounter {
	results := make([]APICounter, 0, len(items))
	for _, counter := range items {
		results = append(results, counter)
	}
	return results
}

func rawAPICount(frame core.FrameInfo) int {
	if frame.APICallCount > 0 {
		return frame.APICallCount
	}
	return len(frame.APICalls)
}

func summaryAPICount(counters []APICounter, frame core.FrameInfo) int {
	total := 0
	for _, counter := range counters {
		total += counter.Count
	}
	if total > 0 {
		return total
	}
	return rawAPICount(frame)
}

func drawCallCountFromCounters(counters []APICounter) int {
	total := 0
	for _, counter := range counters {
		if counter.Category == CategoryDraw {
			total += counter.Count
		}
	}
	return total
}

func statsSource(frame core.FrameInfo) string {
	if len(frame.APISummary) > 0 {
		return "profile_summary"
	}
	if len(frame.APICalls) > 0 || frame.APICallCount > 0 {
		return "raw_sequence"
	}
	return "none"
}

func timingSource(frame core.FrameInfo) string {
	if frame.TimingSource != "" {
		return frame.TimingSource
	}
	if frame.TotalTimeUs > 0 {
		return "frame_cost"
	}
	return "none"
}
