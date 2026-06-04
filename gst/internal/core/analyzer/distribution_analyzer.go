package analyzer

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"gst/internal/core"
)

const StartupFrameSkip = 3

type FrameDistributionEntry struct {
	FrameNum int   `json:"frame"`
	TimeUs   int64 `json:"time_us"`
	APIUs    int64 `json:"api_us"`
	SwapUs   int64 `json:"swap_us"`
	GapUs    int64 `json:"gap_us"`
	GapPct   int   `json:"gap_pct"`
	Draws    int   `json:"draws"`
}

type FrameDistributionSummary struct {
	AvgTimeUs int64 `json:"avg_time_us"`
	P50TimeUs int64 `json:"p50_time_us"`
	P95TimeUs int64 `json:"p95_time_us"`
	P99TimeUs int64 `json:"p99_time_us"`
	AvgDraws  int   `json:"avg_draws"`
	AvgGapPct int   `json:"avg_gap_pct"`
}

type FrameCluster struct {
	Label       string `json:"label"`
	FrameCount  int    `json:"frame_count"`
	AvgTimeUs   int64  `json:"avg_time_us"`
	AvgDraws    int    `json:"avg_draws"`
	AvgGapPct   int    `json:"avg_gap_pct"`
	SampleFrames []int `json:"sample_frames"`
}

type BottleneckInfo struct {
	Type       string `json:"type"`
	Confidence string `json:"confidence"`
	Evidence   string `json:"evidence"`
	SceneHint  string `json:"scene_hint,omitempty"`
}

type DistributionResult struct {
	StartupFramesSkipped int                       `json:"startup_frames_skipped"`
	TotalFrames          int                       `json:"total_frames"`
	Frames               []FrameDistributionEntry  `json:"frames"`
	Summary              FrameDistributionSummary   `json:"summary"`
	FrameClusters        []FrameCluster             `json:"frame_clusters"`
	Bottleneck           BottleneckInfo             `json:"bottleneck"`
}

type DistributionAnalyzer struct {
	log *core.ParsedLog
}

func NewDistributionAnalyzer(log *core.ParsedLog) *DistributionAnalyzer {
	return &DistributionAnalyzer{log: log}
}

func (da *DistributionAnalyzer) Analyze() *DistributionResult {
	if da.log == nil || len(da.log.Frames) == 0 {
		return nil
	}

	start := StartupFrameSkip
	if start > len(da.log.Frames) {
		start = 0
	}
	frames := da.log.Frames[start:]

	entries := make([]FrameDistributionEntry, 0, len(frames))
	for _, f := range frames {
		gapUs := f.TotalTimeUs - f.APITotalTimeUs - f.SwapBufferTimeUs
		if gapUs < 0 {
			gapUs = 0
		}
		gapPct := 0
		if f.TotalTimeUs > 0 {
			gapPct = int(float64(gapUs) / float64(f.TotalTimeUs) * 100)
		}
		draws := f.DrawCallCount
		if draws == 0 {
			draws = countFrameDrawCalls(f)
		}
		entries = append(entries, FrameDistributionEntry{
			FrameNum: f.FrameNum,
			TimeUs:   f.TotalTimeUs,
			APIUs:    f.APITotalTimeUs,
			SwapUs:   f.SwapBufferTimeUs,
			GapUs:    gapUs,
			GapPct:   gapPct,
			Draws:    draws,
		})
	}

	if len(entries) == 0 {
		return nil
	}

	summary := computeDistributionSummary(entries)
	clusters := computeFrameClusters(entries, summary.AvgTimeUs)
	bottleneck := computeBottleneck(entries, frames)

	return &DistributionResult{
		StartupFramesSkipped: start,
		TotalFrames:          len(entries),
		Frames:               entries,
		Summary:              summary,
		FrameClusters:        clusters,
		Bottleneck:           bottleneck,
	}
}

func computeDistributionSummary(entries []FrameDistributionEntry) FrameDistributionSummary {
	times := make([]int64, len(entries))
	var totalTimeUs int64
	var totalDraws int
	var totalGapPct int

	for i, e := range entries {
		times[i] = e.TimeUs
		totalTimeUs += e.TimeUs
		totalDraws += e.Draws
		totalGapPct += e.GapPct
	}

	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })

	n := len(times)
	avgTimeUs := totalTimeUs / int64(n)
	p50 := percentile(times, 50)
	p95 := percentile(times, 95)
	p99 := percentile(times, 99)

	avgDraws := 0
	avgGapPct := 0
	if n > 0 {
		avgDraws = totalDraws / n
		avgGapPct = totalGapPct / n
	}

	return FrameDistributionSummary{
		AvgTimeUs: avgTimeUs,
		P50TimeUs: p50,
		P95TimeUs: p95,
		P99TimeUs: p99,
		AvgDraws:  avgDraws,
		AvgGapPct: avgGapPct,
	}
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(p)/100.0*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func computeFrameClusters(entries []FrameDistributionEntry, avgTimeUs int64) []FrameCluster {
	type bucket struct {
		entries []FrameDistributionEntry
	}

	buckets := map[string]*bucket{
		"heavy":   {},
		"medium":  {},
		"light":   {},
		"anomaly": {},
	}

	for _, e := range entries {
		label := classifyFrame(e, avgTimeUs)
		buckets[label].entries = append(buckets[label].entries, e)
	}

	var clusters []FrameCluster
	for _, label := range []string{"heavy", "medium", "light", "anomaly"} {
		b := buckets[label]
		if len(b.entries) == 0 {
			continue
		}

		var totalTimeUs int64
		var totalDraws, totalGapPct int
		for _, e := range b.entries {
			totalTimeUs += e.TimeUs
			totalDraws += e.Draws
			totalGapPct += e.GapPct
		}
		n := len(b.entries)

		samples := make([]int, 0, 3)
		for i := 0; i < n && i < 3; i++ {
			samples = append(samples, b.entries[i].FrameNum)
		}

		clusters = append(clusters, FrameCluster{
			Label:        label,
			FrameCount:   n,
			AvgTimeUs:    totalTimeUs / int64(n),
			AvgDraws:     totalDraws / n,
			AvgGapPct:    totalGapPct / n,
			SampleFrames: samples,
		})
	}

	return clusters
}

func classifyFrame(e FrameDistributionEntry, avgTimeUs int64) string {
	if avgTimeUs > 0 && e.TimeUs > avgTimeUs*3 {
		return "anomaly"
	}
	if e.Draws > 10000 {
		return "heavy"
	}
	if e.Draws > 1000 {
		return "medium"
	}
	return "light"
}

func computeBottleneck(entries []FrameDistributionEntry, frames []core.FrameInfo) BottleneckInfo {
	if len(entries) == 0 {
		return BottleneckInfo{Type: "unknown", Confidence: "low", Evidence: "no frame data"}
	}

	hasTiming := false
	for _, f := range frames {
		if f.HasTiming || f.TotalTimeUs > 0 {
			hasTiming = true
			break
		}
	}
	if !hasTiming {
		return BottleneckInfo{
			Type:       "unknown",
			Confidence: "low",
			Evidence:   "no timing data available, analysis limited to call density only",
		}
	}

	var totalAPIUs, totalSwapUs, totalTimeUs int64
	var totalReadbackUs, totalSyncUs int64
	for _, f := range frames {
		totalAPIUs += f.APITotalTimeUs
		totalSwapUs += f.SwapBufferTimeUs
		totalTimeUs += f.TotalTimeUs

		for name, summary := range f.APISummary {
			if name == "glReadPixels" || name == "glGetTexImage" || name == "glGetTextureImage" {
				totalReadbackUs += summary.TimeUs
			}
			if name == "glFinish" || name == "glFlush" || name == "glClientWaitSync" || name == "glWaitSync" {
				totalSyncUs += summary.TimeUs
			}
		}
	}

	gapUs := totalTimeUs - totalAPIUs - totalSwapUs
	if gapUs < 0 {
		gapUs = 0
	}

	apiPct := float64(0)
	gapPct := float64(0)
	readbackPct := float64(0)
	if totalTimeUs > 0 {
		apiPct = float64(totalAPIUs) / float64(totalTimeUs) * 100
		gapPct = float64(gapUs) / float64(totalTimeUs) * 100
		readbackPct = float64(totalReadbackUs+totalSyncUs) / float64(totalTimeUs) * 100
	}

	sceneHint := detectScene(frames)

	switch {
	case gapPct > 70:
		return BottleneckInfo{
			Type:       "cpu_blocked_io",
			Confidence: "high",
			Evidence:   fmt.Sprintf("Average gap is %.0f%% of frame time (API only %.0f%%). Non-GL operations dominate.", gapPct, apiPct),
			SceneHint:  sceneHint,
		}
	case readbackPct > 20:
		return BottleneckInfo{
			Type:       "sync_stall",
			Confidence: "high",
			Evidence:   fmt.Sprintf("Sync/readback operations consume %.0f%% of frame time (%.1fms).", readbackPct, float64(totalReadbackUs+totalSyncUs)/1000),
			SceneHint:  sceneHint,
		}
	case apiPct > 60:
		return BottleneckInfo{
			Type:       "cpu_bound_draw",
			Confidence: "high",
			Evidence:   fmt.Sprintf("API calls consume %.0f%% of frame time. Draw call overhead is the bottleneck.", apiPct),
			SceneHint:  sceneHint,
		}
	case gapPct < 30 && apiPct < 30 && totalSwapUs > 0:
		return BottleneckInfo{
			Type:       "gpu_bound",
			Confidence: "medium",
			Evidence:   fmt.Sprintf("Neither API (%.0f%%) nor gap (%.0f%%) dominates. GPU may be the bottleneck (swap=%.0f%%).", apiPct, gapPct, float64(totalSwapUs)/float64(totalTimeUs)*100),
			SceneHint:  sceneHint,
		}
	default:
		return BottleneckInfo{
			Type:       "mixed",
			Confidence: "medium",
			Evidence:   fmt.Sprintf("Multiple factors: API=%.0f%%, gap=%.0f%%, readback/sync=%.0f%%.", apiPct, gapPct, readbackPct),
			SceneHint:  sceneHint,
		}
	}
}

func detectScene(frames []core.FrameInfo) string {
	var drawArrays, drawElements, instanced, compute int
	var hasLineWidth bool

	for _, f := range frames {
		for name, summary := range f.APISummary {
			switch name {
			case "glDrawArrays":
				drawArrays += summary.Count
			case "glDrawElements":
				drawElements += summary.Count
			case "glLineWidth":
				hasLineWidth = true
			}
		}
	}

	for _, f := range frames {
		for _, call := range f.APICalls {
			n := call.APIName
			switch {
			case n == "glDrawElementsInstanced" || n == "glDrawArraysInstanced" ||
				strings.HasPrefix(n, "glDrawElementsInstanced") || strings.HasPrefix(n, "glDrawArraysInstanced"):
				instanced += call.Count
			case n == "glDispatchCompute":
				compute += call.Count
			case n == "glLineWidth":
				hasLineWidth = true
			}
		}
	}

	total := drawArrays + drawElements
	if total == 0 {
		return "unknown"
	}

	drawArraysRatio := float64(drawArrays) / float64(total)

	if instanced > 0 || compute > 0 {
		return "3d"
	}
	if drawElements > 0 && float64(drawElements)/float64(total) > 0.3 {
		return "3d"
	}
	if drawArraysRatio > 0.95 && hasLineWidth {
		return "2d"
	}
	if drawArraysRatio > 0.95 {
		return "2d"
	}
	return "mixed"
}
