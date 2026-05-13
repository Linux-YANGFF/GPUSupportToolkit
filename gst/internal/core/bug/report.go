package bug

import (
	"strings"
	"time"

	"gst/internal/core"
)

func GenerateReport(sourceFile string, findings []core.Finding) *core.DiagnosisReport {
	summary := core.DiagnosisSummary{
		TotalFindings: len(findings),
	}

	for _, f := range findings {
		switch f.Severity {
		case core.SeverityCritical:
			summary.CriticalCount++
		case core.SeverityHigh:
			summary.HighCount++
		case core.SeverityMedium:
			summary.MediumCount++
		case core.SeverityLow:
			summary.LowCount++
		case core.SeverityInfo:
			summary.InfoCount++
		}
	}

	return &core.DiagnosisReport{
		SourceFile:  sourceFile,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:     summary,
		Findings:    findings,
	}
}

func GenerateMarkdownReport(findings []core.Finding, sourceFile string) string {
	bySeverity := groupBySeverity(findings)

	var b strings.Builder

	b.WriteString("# GPU 诊断报告\n\n")
	b.WriteString("**源文件**: " + sourceFile + "\n")
	b.WriteString("**生成时间**: " + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "\n")
	b.WriteString("**发现问题总数**: " + itoaSimple(len(findings)) + "\n\n")

	b.WriteString("---\n\n")
	b.WriteString("## 摘要\n\n")

	severityOrder := []struct {
		label string
		count int
	}{
		{"Critical (严重)", countSeverity(bySeverity, core.SeverityCritical)},
		{"High (高)", countSeverity(bySeverity, core.SeverityHigh)},
		{"Medium (中)", countSeverity(bySeverity, core.SeverityMedium)},
		{"Low (低)", countSeverity(bySeverity, core.SeverityLow)},
		{"Info (信息)", countSeverity(bySeverity, core.SeverityInfo)},
	}

	for _, s := range severityOrder {
		b.WriteString("- **")
		b.WriteString(s.label)
		b.WriteString("**: ")
		b.WriteString(itoaSimple(s.count))
		b.WriteString("\n")
	}

	b.WriteString("\n---\n\n")

	severityLabels := []struct {
		label string
		sev   core.Severity
	}{
		{"严重问题 (Critical)", core.SeverityCritical},
		{"高风险问题 (High)", core.SeverityHigh},
		{"中风险问题 (Medium)", core.SeverityMedium},
		{"低风险问题 (Low)", core.SeverityLow},
		{"信息 (Info)", core.SeverityInfo},
	}

	for _, sl := range severityLabels {
		list := bySeverity[sl.sev]
		if len(list) == 0 {
			continue
		}
		b.WriteString("## ")
		b.WriteString(sl.label)
		b.WriteString("\n\n")

		for i, f := range list {
			b.WriteString("### ")
			b.WriteString(itoaSimple(i + 1))
			b.WriteString(". ")
			b.WriteString(f.Description)
			b.WriteString("\n\n")

			if f.Category != "" {
				b.WriteString("- **类别**: ")
				b.WriteString(f.Category)
				b.WriteString("\n")
			}

			if f.Kind != "" {
				b.WriteString("- **类型**: ")
				b.WriteString(string(f.Kind))
				b.WriteString("\n")
			}

			if f.Confidence != "" {
				b.WriteString("- **置信度**: ")
				b.WriteString(string(f.Confidence))
				b.WriteString("\n")
			}

			if f.Count > 0 {
				b.WriteString("- **数量**: ")
				b.WriteString(itoaSimple(f.Count))
				b.WriteString("\n")
			}

			b.WriteString("- **严重程度**: ")
			b.WriteString(string(f.Severity))
			b.WriteString("\n")

			if f.Evidence != "" {
				b.WriteString("- **证据**: ")
				b.WriteString(f.Evidence)
				b.WriteString("\n")
			}

			if len(f.Examples) > 0 {
				b.WriteString("- **示例**:\n")
				for _, example := range f.Examples {
					b.WriteString("  - ")
					b.WriteString(example)
					b.WriteString("\n")
				}
			}

			if len(f.RootCauseChain) > 0 {
				b.WriteString("- **根因链**:\n")
				for _, rc := range f.RootCauseChain {
					b.WriteString("  1. ")
					b.WriteString(rc)
					b.WriteString("\n")
				}
			}

			if f.FixSuggestion != "" {
				b.WriteString("- **修复建议**: ")
				b.WriteString(f.FixSuggestion)
				b.WriteString("\n")
			}

			b.WriteString("\n")
		}

		b.WriteString("---\n\n")
	}

	return b.String()
}

func groupBySeverity(findings []core.Finding) map[core.Severity][]core.Finding {
	result := make(map[core.Severity][]core.Finding)
	for _, f := range findings {
		result[f.Severity] = append(result[f.Severity], f)
	}
	return result
}

func countSeverity(groups map[core.Severity][]core.Finding, sev core.Severity) int {
	return len(groups[sev])
}

func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewNullPointerDetector())
	r.Register(NewResourceLeakDetector())
	r.Register(&ShaderErrorDetector{})
	r.Register(NewAntiPatternDetector())
	r.Register(NewPerfAnomalyDetector())
	r.Register(NewThreadSafetyDetector())
	r.Register(&DriverErrorDetector{})
	r.Register(NewUnbatchedDrawCallDetector())
	r.Register(NewExcessiveGetErrorDetector())
	return r
}

func GenerateFullReport(registry *Registry, log *core.ParsedLog, sourceFile string) *core.DiagnosisReport {
	findings := registry.RunAll(log)
	return GenerateReport(sourceFile, findings)
}
