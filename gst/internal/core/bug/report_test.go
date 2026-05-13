package bug

import (
	"strings"
	"testing"

	"gst/internal/core"
)

func TestGenerateReport_EmptyFindings(t *testing.T) {
	findings := []core.Finding{}

	report := GenerateReport("test.log", findings)

	if report.SourceFile != "test.log" {
		t.Errorf("expected source file test.log, got %s", report.SourceFile)
	}
	if report.GeneratedAt == "" {
		t.Error("expected non-empty GeneratedAt")
	}
	if report.Summary.TotalFindings != 0 {
		t.Errorf("expected 0 total findings, got %d", report.Summary.TotalFindings)
	}
	if report.Summary.CriticalCount != 0 {
		t.Errorf("expected 0 critical, got %d", report.Summary.CriticalCount)
	}
	if len(report.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(report.Findings))
	}
}

func TestGenerateReport_NilFindings(t *testing.T) {
	report := GenerateReport("test.log", nil)

	if report.Summary.TotalFindings != 0 {
		t.Errorf("expected 0 total findings for nil, got %d", report.Summary.TotalFindings)
	}
}

func TestGenerateReport_SingleCriticalFinding(t *testing.T) {
	findings := []core.Finding{
		{
			Severity:       core.SeverityCritical,
			Category:       "driver_error",
			Description:    "GL_INVALID_OPERATION",
			Evidence:       "line 10",
			RootCauseChain: []string{"bad state"},
			FixSuggestion:  "check state",
		},
	}

	report := GenerateReport("test.log", findings)

	if report.Summary.TotalFindings != 1 {
		t.Errorf("expected 1 total, got %d", report.Summary.TotalFindings)
	}
	if report.Summary.CriticalCount != 1 {
		t.Errorf("expected 1 critical, got %d", report.Summary.CriticalCount)
	}
	if report.Summary.HighCount != 0 {
		t.Errorf("expected 0 high, got %d", report.Summary.HighCount)
	}
	if report.Summary.MediumCount != 0 {
		t.Errorf("expected 0 medium, got %d", report.Summary.MediumCount)
	}
	if report.Summary.LowCount != 0 {
		t.Errorf("expected 0 low, got %d", report.Summary.LowCount)
	}
	if report.Summary.InfoCount != 0 {
		t.Errorf("expected 0 info, got %d", report.Summary.InfoCount)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != core.SeverityCritical {
		t.Errorf("expected critical severity, got %s", report.Findings[0].Severity)
	}
}

func TestGenerateReport_MixedSeverities(t *testing.T) {
	findings := []core.Finding{
		{Severity: core.SeverityCritical, Description: "c1"},
		{Severity: core.SeverityHigh, Description: "h1"},
		{Severity: core.SeverityHigh, Description: "h2"},
		{Severity: core.SeverityMedium, Description: "m1"},
		{Severity: core.SeverityMedium, Description: "m2"},
		{Severity: core.SeverityMedium, Description: "m3"},
		{Severity: core.SeverityLow, Description: "l1"},
		{Severity: core.SeverityLow, Description: "l2"},
		{Severity: core.SeverityLow, Description: "l3"},
		{Severity: core.SeverityLow, Description: "l4"},
		{Severity: core.SeverityInfo, Description: "i1"},
	}

	report := GenerateReport("mixed.log", findings)

	if report.Summary.TotalFindings != 11 {
		t.Errorf("expected 11 total, got %d", report.Summary.TotalFindings)
	}
	if report.Summary.CriticalCount != 1 {
		t.Errorf("expected 1 critical, got %d", report.Summary.CriticalCount)
	}
	if report.Summary.HighCount != 2 {
		t.Errorf("expected 2 high, got %d", report.Summary.HighCount)
	}
	if report.Summary.MediumCount != 3 {
		t.Errorf("expected 3 medium, got %d", report.Summary.MediumCount)
	}
	if report.Summary.LowCount != 4 {
		t.Errorf("expected 4 low, got %d", report.Summary.LowCount)
	}
	if report.Summary.InfoCount != 1 {
		t.Errorf("expected 1 info, got %d", report.Summary.InfoCount)
	}
}

func TestGenerateReport_SourceFilePreserved(t *testing.T) {
	report := GenerateReport("/path/to/file.trace", nil)

	if report.SourceFile != "/path/to/file.trace" {
		t.Errorf("expected /path/to/file.trace, got %s", report.SourceFile)
	}
}

func TestGenerateMarkdownReport_EmptyFindings(t *testing.T) {
	result := GenerateMarkdownReport(nil, "test.log")

	if !strings.Contains(result, "# GPU 诊断报告") {
		t.Error("expected markdown title")
	}
	if !strings.Contains(result, "test.log") {
		t.Error("expected source file in markdown")
	}
	if !strings.Contains(result, "发现问题总数") {
		t.Error("expected findings total section")
	}
}

func TestGenerateMarkdownReport_SingleFinding(t *testing.T) {
	findings := []core.Finding{
		{
			Severity:       core.SeverityHigh,
			Category:       "resource_leak",
			Description:    "Texture leak detected",
			Evidence:       "3 textures not deleted",
			RootCauseChain: []string{"missing glDeleteTextures"},
			FixSuggestion:  "add glDeleteTextures call",
		},
	}

	result := GenerateMarkdownReport(findings, "leak.log")

	if !strings.Contains(result, "# GPU 诊断报告") {
		t.Error("expected markdown title")
	}
	if !strings.Contains(result, "leak.log") {
		t.Error("expected source file")
	}
	if !strings.Contains(result, "Texture leak detected") {
		t.Error("expected description")
	}
	if !strings.Contains(result, "resource_leak") {
		t.Error("expected category")
	}
	if !strings.Contains(result, "3 textures not deleted") {
		t.Error("expected evidence")
	}
	if !strings.Contains(result, "missing glDeleteTextures") {
		t.Error("expected root cause chain")
	}
	if !strings.Contains(result, "add glDeleteTextures call") {
		t.Error("expected fix suggestion")
	}
	if !strings.Contains(result, "高风险问题 (High)") {
		t.Error("expected severity section header")
	}
}

func TestGenerateMarkdownReport_MultipleFindings(t *testing.T) {
	findings := []core.Finding{
		{
			Severity: core.SeverityCritical, Category: "driver_error",
			Description: "GL error", Evidence: "err1",
			FixSuggestion: "fix1",
		},
		{
			Severity: core.SeverityMedium, Category: "perf_anomaly",
			Description: "Slow frame", Evidence: "err2",
			FixSuggestion: "fix2",
		},
		{
			Severity: core.SeverityLow, Category: "antipattern",
			Description: "Redundant call", Evidence: "err3",
			FixSuggestion: "fix3",
		},
	}

	result := GenerateMarkdownReport(findings, "multi.log")

	if !strings.Contains(result, "发现问题总数") {
		t.Error("expected findings count")
	}
	if !strings.Contains(result, "GL error") {
		t.Error("expected first finding")
	}
	if !strings.Contains(result, "Slow frame") {
		t.Error("expected second finding")
	}
	if !strings.Contains(result, "Redundant call") {
		t.Error("expected third finding")
	}
}

func TestGenerateMarkdownReport_IncludesGeneratedAt(t *testing.T) {
	result := GenerateMarkdownReport(nil, "test.log")

	if !strings.Contains(result, "生成时间") {
		t.Error("expected Generated At timestamp")
	}
}

func TestGenerateMarkdownReport_NoRootCauseChain(t *testing.T) {
	findings := []core.Finding{
		{
			Severity:      core.SeverityInfo,
			Description:   "Simple info",
			FixSuggestion: "no action needed",
		},
	}

	result := GenerateMarkdownReport(findings, "test.log")

	if !strings.Contains(result, "Simple info") {
		t.Error("expected description in output")
	}
}

func TestGenerateMarkdownReport_WithRootCauseChain(t *testing.T) {
	findings := []core.Finding{
		{
			Severity:       core.SeverityHigh,
			Description:    "Test",
			RootCauseChain: []string{"cause 1", "cause 2"},
			FixSuggestion:  "fix",
		},
	}

	result := GenerateMarkdownReport(findings, "test.log")

	if !strings.Contains(result, "cause 1") {
		t.Error("expected cause 1")
	}
	if !strings.Contains(result, "cause 2") {
		t.Error("expected cause 2")
	}
	if !strings.Contains(result, "根因链") {
		t.Error("expected root cause chain section")
	}
}

func TestGenerateMarkdownReport_SeverityGrouping(t *testing.T) {
	findings := []core.Finding{
		{Severity: core.SeverityCritical, Description: "C1", FixSuggestion: "f"},
		{Severity: core.SeverityCritical, Description: "C2", FixSuggestion: "f"},
		{Severity: core.SeverityHigh, Description: "H1", FixSuggestion: "f"},
	}

	result := GenerateMarkdownReport(findings, "test.log")

	if !strings.Contains(result, "严重问题 (Critical)") {
		t.Error("expected Critical severity section")
	}
	if !strings.Contains(result, "C1") || !strings.Contains(result, "C2") {
		t.Error("expected both critical findings")
	}
	if !strings.Contains(result, "H1") {
		t.Error("expected high finding")
	}
}

func TestNewDefaultRegistry_RegistersAllSeven(t *testing.T) {
	r := NewDefaultRegistry()

	if len(r.diagnosers) != 9 {
		t.Errorf("expected 9 diagnosers, got %d", len(r.diagnosers))
	}
}

func TestGenerateFullReport_Integration(t *testing.T) {
	r := NewDefaultRegistry()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glClear", LineNum: 1},
				},
			},
		},
	}

	report := GenerateFullReport(r, log, "test.trace")

	if report.SourceFile != "test.trace" {
		t.Errorf("expected source test.trace, got %s", report.SourceFile)
	}
	if report.GeneratedAt == "" {
		t.Error("expected GeneratedAt")
	}
}

func TestGenerateFullReport_EmptyLog(t *testing.T) {
	r := NewDefaultRegistry()

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{},
	}

	report := GenerateFullReport(r, log, "empty.log")

	if report.Summary.TotalFindings != 0 {
		t.Errorf("expected 0 findings for empty log, got %d", report.Summary.TotalFindings)
	}
}
