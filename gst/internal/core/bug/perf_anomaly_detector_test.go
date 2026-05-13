package bug

import (
	"math"
	"strings"
	"testing"

	"gst/internal/core"
)

func TestPerfAnomalyDetector_FrameTimeSpike(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := []core.FrameInfo{
		{FrameNum: 1, TotalTimeUs: 100000},
		{FrameNum: 2, TotalTimeUs: 105000},
		{FrameNum: 3, TotalTimeUs: 98000},
		{FrameNum: 4, TotalTimeUs: 102000},
		{FrameNum: 5, TotalTimeUs: 500000},
		{FrameNum: 6, TotalTimeUs: 101000},
		{FrameNum: 7, TotalTimeUs: 99000},
		{FrameNum: 8, TotalTimeUs: 103000},
		{FrameNum: 9, TotalTimeUs: 100000},
		{FrameNum: 10, TotalTimeUs: 97000},
	}

	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	var frameSpikeFindings []core.Finding
	for _, f := range findings {
		if f.Description != "" && f.Description[:len("Frame time spike")] == "Frame time spike" {
			frameSpikeFindings = append(frameSpikeFindings, f)
		}
	}

	if len(frameSpikeFindings) == 0 {
		t.Fatal("expected at least one frame time spike finding")
	}

	spike := frameSpikeFindings[0]
	if spike.Severity != core.SeverityMedium {
		t.Errorf("expected severity medium, got %s", spike.Severity)
	}
	if spike.Category != "perf_anomaly" {
		t.Errorf("expected category perf_anomaly, got %s", spike.Category)
	}

	t.Logf("Spike detected: %s", spike.Evidence)
}

func TestPerfAnomalyDetector_NoSpikeInStableFrames(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := make([]core.FrameInfo, 20)
	for i := range frames {
		frames[i] = core.FrameInfo{
			FrameNum:    i + 1,
			TotalTimeUs: int64(100000 + (i%3)*2000),
		}
	}

	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if f.Description[:len("Frame time spike")] == "Frame time spike" {
			t.Errorf("unexpected frame time spike in stable data: %s", f.Evidence)
		}
	}
}

func TestPerfAnomalyDetector_DrawCallSpike(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := []core.FrameInfo{
		{FrameNum: 1, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 10, TimeUs: 50000},
			{APIName: "glBindBuffer", Count: 5, TimeUs: 1000},
		}},
		{FrameNum: 2, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 11, TimeUs: 50000},
		}},
		{FrameNum: 3, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 12, TimeUs: 50000},
		}},
		{FrameNum: 4, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 10, TimeUs: 50000},
		}},
		{FrameNum: 5, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 11, TimeUs: 50000},
		}},
		{FrameNum: 6, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 500, TimeUs: 80000},
		}},
		{FrameNum: 7, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 490, TimeUs: 78000},
		}},
		{FrameNum: 8, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 10, TimeUs: 50000},
		}},
		{FrameNum: 9, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 11, TimeUs: 50000},
		}},
		{FrameNum: 10, TotalTimeUs: 100000, APICalls: []core.APILogEntry{
			{APIName: "glDrawElements", Count: 10, TimeUs: 50000},
		}},
	}

	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	var drawCallFindings []core.Finding
	for _, f := range findings {
		if f.Description != "" {
			for i := range f.Description {
				if len(f.Description)-i >= len("Sudden draw call") &&
					f.Description[i:i+len("Sudden draw call")] == "Sudden draw call" {
					drawCallFindings = append(drawCallFindings, f)
					break
				}
			}
		}
	}

	if len(drawCallFindings) == 0 {
		t.Fatal("expected at least one draw call spike finding")
	}

	for _, f := range drawCallFindings {
		if f.Severity != core.SeverityMedium {
			t.Errorf("expected severity medium, got %s", f.Severity)
		}
		if f.Category != "perf_anomaly" {
			t.Errorf("expected category perf_anomaly, got %s", f.Category)
		}
		t.Logf("Draw call spike: %s", f.Evidence)
	}
}

func TestPerfAnomalyDetector_SlowAPICall(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	normalTime := int64(1000)
	spikeTime := int64(50000)

	frames := make([]core.FrameInfo, 20)
	for i := range frames {
		tm := normalTime
		if i == 10 {
			tm = spikeTime
		}
		frames[i] = core.FrameInfo{
			FrameNum:    i + 1,
			TotalTimeUs: 100000,
			APICalls: []core.APILogEntry{
				{APIName: "glCompileShader", Count: 1, TimeUs: tm},
			},
		}
	}

	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	var slowCallFindings []core.Finding
	for _, f := range findings {
		if f.Description != "" {
			for i := range f.Description {
				if len(f.Description)-i >= len("Slow ") &&
					f.Description[i:i+len("Slow ")] == "Slow " {
					slowCallFindings = append(slowCallFindings, f)
					break
				}
			}
		}
	}

	if len(slowCallFindings) == 0 {
		t.Fatal("expected at least one slow API call finding")
	}

	sc := slowCallFindings[0]
	if sc.Severity != core.SeverityMedium {
		t.Errorf("expected severity medium, got %s", sc.Severity)
	}
	if sc.Category != "perf_anomaly" {
		t.Errorf("expected category perf_anomaly, got %s", sc.Category)
	}
	t.Logf("Slow API call: %s", sc.Evidence)
}

func TestPerfAnomalyDetector_ExtremeFrameTimeSpike(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := []core.FrameInfo{
		{FrameNum: 1, TotalTimeUs: 16666},
		{FrameNum: 2, TotalTimeUs: 16666},
		{FrameNum: 3, TotalTimeUs: 16666},
		{FrameNum: 4, TotalTimeUs: 16666},
		{FrameNum: 5, TotalTimeUs: 16666},
		{FrameNum: 6, TotalTimeUs: 16666},
		{FrameNum: 7, TotalTimeUs: 500000},
		{FrameNum: 8, TotalTimeUs: 16666},
	}

	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	var spikeFound bool
	for _, f := range findings {
		if f.Description != "" && f.Description[:len("Frame time spike")] == "Frame time spike" {
			spikeFound = true
			if f.FixSuggestion == "" {
				t.Error("fix suggestion should not be empty")
			}
			if len(f.RootCauseChain) == 0 {
				t.Error("root cause chain should not be empty")
			}
			t.Logf("Extreme spike: %s", f.Evidence)
		}
	}
	if !spikeFound {
		t.Fatal("expected frame time spike not found for extreme outlier")
	}
}

func TestPerfAnomalyDetector_EmptyLog(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := []core.FrameInfo{}
	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	if len(findings) != 0 {
		t.Errorf("expected no findings for empty log, got %d", len(findings))
	}
}

func TestPerfAnomalyDetector_SingleFrame(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := []core.FrameInfo{
		{FrameNum: 1, TotalTimeUs: 100000},
	}
	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	for _, f := range findings {
		if len(findings) > 0 {
			t.Logf("finding on single frame: %s", f.Description)
		}
	}
}

func TestPerfAnomalyDetector_RealWorldPattern(t *testing.T) {
	detector := NewPerfAnomalyDetector()

	frames := []core.FrameInfo{
		{
			FrameNum:    1,
			TotalTimeUs: 16667,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 150, TimeUs: 8000},
				{APIName: "glBindBuffer", Count: 20, TimeUs: 200},
			},
		},
		{
			FrameNum:    2,
			TotalTimeUs: 16333,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 145, TimeUs: 7800},
				{APIName: "glBindBuffer", Count: 18, TimeUs: 200},
			},
		},
		{
			FrameNum:    3,
			TotalTimeUs: 200000,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 800, TimeUs: 120000},
				{APIName: "glBindBuffer", Count: 300, TimeUs: 5000},
				{APIName: "glCompileShader", Count: 5, TimeUs: 70000},
			},
		},
		{
			FrameNum:    4,
			TotalTimeUs: 16500,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 148, TimeUs: 7900},
				{APIName: "glBindBuffer", Count: 19, TimeUs: 200},
			},
		},
		{
			FrameNum:    5,
			TotalTimeUs: 16700,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 152, TimeUs: 8100},
				{APIName: "glBindBuffer", Count: 21, TimeUs: 200},
			},
		},
		{
			FrameNum:    6,
			TotalTimeUs: 16600,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 147, TimeUs: 7950},
				{APIName: "glBindBuffer", Count: 17, TimeUs: 200},
			},
		},
		{
			FrameNum:    7,
			TotalTimeUs: 16400,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 149, TimeUs: 8000},
				{APIName: "glBindBuffer", Count: 22, TimeUs: 200},
			},
		},
		{
			FrameNum:    8,
			TotalTimeUs: 16550,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 151, TimeUs: 8050},
				{APIName: "glBindBuffer", Count: 20, TimeUs: 200},
			},
		},
		{
			FrameNum:    9,
			TotalTimeUs: 16680,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 146, TimeUs: 7900},
				{APIName: "glBindBuffer", Count: 18, TimeUs: 200},
			},
		},
		{
			FrameNum:    10,
			TotalTimeUs: 16500,
			APICalls: []core.APILogEntry{
				{APIName: "glClear", Count: 1, TimeUs: 500},
				{APIName: "glDrawElements", Count: 150, TimeUs: 8000},
				{APIName: "glBindBuffer", Count: 21, TimeUs: 200},
			},
		},
	}

	log := &core.ParsedLog{Frames: frames}
	findings := detector.Diagnose(log)

	if len(findings) == 0 {
		t.Fatal("expected findings for real-world test pattern with clear anomalies")
	}

	frameSpikeCount := 0
	drawCallSpikeCount := 0
	slowCallCount := 0
	for _, f := range findings {
		if f.Severity != core.SeverityMedium {
			t.Errorf("expected severity medium, got %s", f.Severity)
		}
		if f.Category != "perf_anomaly" {
			t.Errorf("expected category perf_anomaly, got %s", f.Category)
		}

		desc := f.Description
		if len(desc) >= len("Frame time spike") && desc[:len("Frame time spike")] == "Frame time spike" {
			frameSpikeCount++
		}
		if len(desc) >= len("Sudden draw call") && strings.Contains(desc, "Sudden draw call") {
			drawCallSpikeCount++
		}
		if len(desc) >= len("Slow ") && desc[:len("Slow ")] == "Slow " {
			slowCallCount++
		}
	}

	if frameSpikeCount == 0 {
		t.Error("expected at least one frame time spike")
	}
	if drawCallSpikeCount == 0 {
		t.Error("expected at least one draw call spike")
	}
	if slowCallCount == 0 {
		t.Error("expected at least one slow API call")
	}

	t.Logf("Findings: %d frame spikes, %d draw call spikes, %d slow calls",
		frameSpikeCount, drawCallSpikeCount, slowCallCount)

	mean := float64(0)
	for _, f := range frames {
		mean += float64(f.TotalTimeUs)
	}
	mean /= float64(len(frames))
	var variance float64
	for _, f := range frames {
		diff := float64(f.TotalTimeUs) - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(frames)))
	t.Logf("Frame stats: mean=%.1fms, stddev=%.1fms, threshold=%.1fms",
		mean/1000.0, stddev/1000.0, (mean+2*stddev)/1000.0)

	if float64(frames[2].TotalTimeUs) <= (mean + 2*stddev) {
		t.Error("frame 3 (200ms) should be above 2-stddev threshold")
	}
}
