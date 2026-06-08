package bug

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"gst/internal/core"
)

func GenerateReport(sourceFile string, findings []core.Finding) *core.DiagnosisReport {
	findings = normalizeFindings(findings)
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
		SchemaVersion: core.DiagnosisSchemaVersion,
		SourceFile:    sourceFile,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Summary:       summary,
		Findings:      findings,
	}
}

func GenerateMarkdownReport(findings []core.Finding, sourceFile string) string {
	findings = normalizeFindings(findings)
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

func normalizeFindings(findings []core.Finding) []core.Finding {
	if len(findings) == 0 {
		return []core.Finding{}
	}
	normalized := make([]core.Finding, 0, len(findings))
	for i, finding := range findings {
		normalized = append(normalized, normalizeFinding(finding, i))
	}
	return normalized
}

func normalizeFinding(f core.Finding, index int) core.Finding {
	f.Severity = normalizeSeverity(f.Severity)
	f.SeverityRank = severityRank(f.Severity)
	f.Category = normalizeCategory(f.Category)
	if f.CategoryLabel == "" {
		f.CategoryLabel = categoryLabel(f.Category)
	}
	if f.Kind == "" {
		f.Kind = inferFindingKind(f.Category, f.Severity)
	}
	if f.Confidence == "" {
		f.Confidence = core.ConfidenceMedium
	}
	if f.RootCauseChain == nil {
		f.RootCauseChain = []string{}
	}
	if f.ID == "" {
		f.ID = stableFindingID(f, index)
	}
	return f
}

func normalizeSeverity(severity core.Severity) core.Severity {
	switch severity {
	case core.SeverityCritical, core.SeverityHigh, core.SeverityMedium, core.SeverityLow, core.SeverityInfo:
		return severity
	default:
		return core.SeverityInfo
	}
}

func severityRank(severity core.Severity) int {
	switch severity {
	case core.SeverityCritical:
		return 1
	case core.SeverityHigh:
		return 2
	case core.SeverityMedium:
		return 3
	case core.SeverityLow:
		return 4
	default:
		return 5
	}
}

func normalizeCategory(category string) string {
	category = strings.TrimSpace(strings.ToLower(category))
	if category == "" {
		return "general"
	}
	return strings.ReplaceAll(category, "-", "_")
}

func categoryLabel(category string) string {
	switch category {
	case "driver_error":
		return "Driver Error"
	case "null_pointer":
		return "Null Pointer"
	case "resource_leak":
		return "Resource Leak"
	case "shader_error":
		return "Shader Error"
	case "antipattern":
		return "API Anti-Pattern"
	case "perf_anomaly", "performance":
		return "Performance"
	case "thread_safety":
		return "Thread Safety"
	case "cross_context_resource":
		return "Cross-Context Resource"
	case "unbatched_drawcall":
		return "Unbatched Draw Call"
	case "general":
		return "General"
	default:
		parts := strings.Split(category, "_")
		for i, part := range parts {
			if part == "" {
				continue
			}
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
		return strings.Join(parts, " ")
	}
}

func inferFindingKind(category string, severity core.Severity) core.FindingKind {
	switch category {
	case "performance", "perf_anomaly", "unbatched_drawcall", "antipattern":
		return core.FindingKindPerformance
	case "general":
		if severity == core.SeverityInfo {
			return core.FindingKindInfo
		}
		return core.FindingKindBug
	default:
		return core.FindingKindBug
	}
}

func stableFindingID(f core.Finding, index int) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(string(f.Severity)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(f.Category))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(f.Description))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(f.Evidence))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(fmt.Sprintf("%d", index)))
	return fmt.Sprintf("finding-%08x", h.Sum32())
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
