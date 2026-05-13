package bug

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"gst/internal/core"
)

type PerfAnomalyDetector struct{}

func NewPerfAnomalyDetector() *PerfAnomalyDetector {
	return &PerfAnomalyDetector{}
}

func (d *PerfAnomalyDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	var findings []core.Finding

	if len(log.Frames) == 0 {
		return findings
	}

	findings = append(findings, d.detectFrameTimeSpikes(log.Frames)...)
	findings = append(findings, d.detectDrawCallSpikes(log.Frames)...)
	findings = append(findings, d.detectSlowAPICalls(log.Frames)...)

	return findings
}

func meanStddev(values []float64) (float64, float64) {
	n := float64(len(values))
	var sum float64
	for _, v := range values {
		sum += v
	}
	avg := sum / n
	var varSum float64
	for _, v := range values {
		diff := v - avg
		varSum += diff * diff
	}
	return avg, math.Sqrt(varSum / n)
}

func robustStats(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	mean, stddev := meanStddev(values)
	if stddev == 0 {
		return mean, stddev
	}
	threshold := mean + 2*stddev
	var trimmed []float64
	for _, v := range values {
		if v <= threshold {
			trimmed = append(trimmed, v)
		}
	}
	if len(trimmed) < 2 || float64(len(trimmed)) < float64(len(values))*0.5 {
		return mean, stddev
	}
	trimMean, trimStddev := meanStddev(trimmed)
	if trimStddev == 0 {
		proxyStddev := trimMean * 0.1
		if proxyStddev < 100 {
			proxyStddev = 100
		}
		return trimMean, proxyStddev
	}
	return trimMean, trimStddev
}

func frameRobustStats(frames []core.FrameInfo) (float64, float64) {
	values := make([]float64, len(frames))
	for i, f := range frames {
		values[i] = float64(f.TotalTimeUs)
	}
	return robustStats(values)
}

func medianFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2.0
	}
	return sorted[mid]
}

func (d *PerfAnomalyDetector) detectFrameTimeSpikes(frames []core.FrameInfo) []core.Finding {
	var findings []core.Finding

	if len(frames) < 3 {
		return findings
	}

	mean, stddev := frameRobustStats(frames)
	if stddev == 0 {
		return findings
	}

	threshold := mean + 2*stddev

	for _, f := range frames {
		if float64(f.TotalTimeUs) > threshold {
			evidence := fmt.Sprintf(
				"Frame %d: %.1fms (baseline_mean=%.1fms, baseline_stddev=%.1fms, threshold=%.1fms)",
				f.FrameNum,
				float64(f.TotalTimeUs)/1000.0,
				mean/1000.0,
				stddev/1000.0,
				threshold/1000.0,
			)
			findings = append(findings, core.Finding{
				Severity:       core.SeverityMedium,
				Category:       "perf_anomaly",
				Description:    fmt.Sprintf("Frame time spike detected at frame %d", f.FrameNum),
				Evidence:       evidence,
				RootCauseChain: []string{"GPU pipeline stall or heavy draw workload"},
				FixSuggestion: fmt.Sprintf(
					"Investigate frame %d for excessive draw calls, shader complexity, or resource contention",
					f.FrameNum,
				),
			})
		}
	}

	return findings
}

var drawCallAPIs = map[string]bool{
	"glDrawArrays":             true,
	"glDrawElements":           true,
	"glDrawRangeElements":      true,
	"glDrawArraysIndirect":     true,
	"glDrawElementsIndirect":   true,
	"glMultiDrawArrays":        true,
	"glMultiDrawElements":      true,
	"glDrawElementsInstanced":  true,
	"glDrawArraysInstanced":    true,
}

func countDrawCalls(frame *core.FrameInfo) int {
	count := 0
	for _, call := range frame.APICalls {
		if drawCallAPIs[call.APIName] {
			count += call.Count
			if call.Count == 0 {
				count++
			}
		}
	}
	if count == 0 {
		for _, call := range frame.APICalls {
			if strings.Contains(strings.ToLower(call.APIName), "draw") {
				count += call.Count
				if call.Count == 0 {
					count++
				}
			}
		}
	}
	return count
}

func (d *PerfAnomalyDetector) detectDrawCallSpikes(frames []core.FrameInfo) []core.Finding {
	var findings []core.Finding

	if len(frames) < 3 {
		return findings
	}

	drawCounts := make([]int, len(frames))
	for i := range frames {
		drawCounts[i] = countDrawCalls(&frames[i])
	}

	deltas := make([]float64, 0, len(frames)-1)
	for i := 1; i < len(drawCounts); i++ {
		deltas = append(deltas, math.Abs(float64(drawCounts[i]-drawCounts[i-1])))
	}
	if len(deltas) == 0 {
		return findings
	}

	medDelta := medianFloat64(deltas)
	if medDelta == 0 {
		medDelta = 1
	}
	spikeThreshold := medDelta * 5.0
	if spikeThreshold < 10 {
		spikeThreshold = 10
	}

	for i := 1; i < len(drawCounts); i++ {
		delta := math.Abs(float64(drawCounts[i] - drawCounts[i-1]))
		if delta > spikeThreshold {
			deltaPct := 0.0
			if drawCounts[i-1] > 0 {
				deltaPct = delta / float64(drawCounts[i-1]) * 100.0
			}
			evidence := fmt.Sprintf(
				"Draw calls: frame %d=%d -> frame %d=%d (delta=%.0f, %.1f%%, threshold=%.0f)",
				frames[i-1].FrameNum, drawCounts[i-1],
				frames[i].FrameNum, drawCounts[i],
				delta, deltaPct, spikeThreshold,
			)
			findings = append(findings, core.Finding{
				Severity:       core.SeverityMedium,
				Category:       "perf_anomaly",
				Description:    fmt.Sprintf("Sudden draw call count change between frames %d and %d", frames[i-1].FrameNum, frames[i].FrameNum),
				Evidence:       evidence,
				RootCauseChain: []string{"Scene complexity change or conditional rendering path"},
				FixSuggestion:  "Check for unexpected geometry variations or LOD transitions between consecutive frames",
			})
		}
	}

	return findings
}

func (d *PerfAnomalyDetector) detectSlowAPICalls(frames []core.FrameInfo) []core.Finding {
	var findings []core.Finding

	callDurations := make(map[string][]float64)
	for _, f := range frames {
		for _, call := range f.APICalls {
			if call.TimeUs > 0 {
				callDurations[call.APIName] = append(callDurations[call.APIName], float64(call.TimeUs))
			}
		}
	}

	type apiThreshold struct {
		mean      float64
		threshold float64
	}
	thresholds := make(map[string]apiThreshold)
	for apiName, durations := range callDurations {
		mean, stddev := robustStats(durations)
		if stddev == 0 {
			continue
		}
		thresholds[apiName] = apiThreshold{
			mean:      mean,
			threshold: mean + 2*stddev,
		}
	}

	for _, f := range frames {
		for _, call := range f.APICalls {
			if call.TimeUs <= 0 {
				continue
			}
			th, ok := thresholds[call.APIName]
			if !ok {
				continue
			}
			if float64(call.TimeUs) > th.threshold {
				evidence := fmt.Sprintf(
					"Frame %d: %s duration=%dus (baseline_mean=%.1fus, threshold=%.1fus)",
					f.FrameNum,
					call.APIName,
					call.TimeUs,
					th.mean,
					th.threshold,
				)
				findings = append(findings, core.Finding{
					Severity:       core.SeverityMedium,
					Category:       "perf_anomaly",
					Description:    fmt.Sprintf("Slow %s call in frame %d", call.APIName, f.FrameNum),
					Evidence:       evidence,
					RootCauseChain: []string{"Unusual GPU work in this specific API call"},
					FixSuggestion:  fmt.Sprintf("Profile %s parameters for this frame to identify excessive work", call.APIName),
				})
			}
		}
	}

	return findings
}
