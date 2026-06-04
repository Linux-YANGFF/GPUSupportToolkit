package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"gst/internal/core"
	"gst/internal/core/bug"
)

// OverviewAnalyzer 综合摘要分析器
type OverviewAnalyzer struct {
	log    *core.ParsedLog
	format string
}

// NewOverviewAnalyzer 创建综合摘要分析器
func NewOverviewAnalyzer(log *core.ParsedLog, format string) *OverviewAnalyzer {
	return &OverviewAnalyzer{log: log, format: format}
}

// Analyze 生成综合摘要
func (oa *OverviewAnalyzer) Analyze() *core.OverviewResult {
	if oa.log == nil || len(oa.log.Frames) == 0 {
		return nil
	}

	basic := oa.buildBasic()
	perf := oa.buildPerformance()
	diag := oa.buildDiagnosis()
	resources := oa.buildResources()
	summary := oa.buildSummary(basic, perf, diag)

	return &core.OverviewResult{
		Basic:            basic,
		Performance:      perf,
		DiagnosisSummary: diag,
		Resources:        resources,
		Summary:          summary,
	}
}

func (oa *OverviewAnalyzer) buildBasic() core.OverviewBasic {
	totalTimeMs := oa.log.TotalTimeUs / 1000
	return core.OverviewBasic{
		Format:      oa.format,
		FrameCount:  len(oa.log.Frames),
		TotalTimeMs: totalTimeMs,
		FPS:         oa.log.FPS,
	}
}

func (oa *OverviewAnalyzer) buildPerformance() core.OverviewPerformance {
	frames := oa.log.Frames
	n := len(frames)
	if n == 0 {
		return core.OverviewPerformance{}
	}

	// 收集有真实耗时来源的帧时间。没有 frame cost/profile 的 rawtrace 不参与百分位统计。
	times := make([]int64, 0, n)
	for _, f := range frames {
		if f.HasTiming || f.TotalTimeUs > 0 {
			times = append(times, f.TotalTimeUs)
		}
	}
	if len(times) == 0 {
		fa := NewFrameAnalyzer(oa.log)
		return core.OverviewPerformance{
			FrameTime:      core.OverviewFrameTime{},
			SlowFramesTop5: fa.FindTopSlowFrames(5),
			BottleneckHint: "无真实耗时数据（当前 rawtrace 未提供 frame cost/profile 耗时）",
		}
	}
	n = len(times)
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })

	// 统计
	minUs := times[0]
	maxUs := times[n-1]
	var totalUs int64
	for _, t := range times {
		totalUs += t
	}
	avgUs := totalUs / int64(n)

	// 百分位
	p95Idx := min(int(float64(n)*0.95), n-1)
	p99Idx := min(int(float64(n)*0.99), n-1)

	frameTime := core.OverviewFrameTime{
		MinUs: minUs,
		MaxUs: maxUs,
		AvgUs: avgUs,
		P95Us: times[p95Idx],
		P99Us: times[p99Idx],
	}

	// Top 5 慢帧
	fa := NewFrameAnalyzer(oa.log)
	topFrames := fa.FindTopSlowFrames(5)

	// 瓶颈判断
	hint := ""
	if totalUs > 0 {
		hint = oa.classifyBottleneck(avgUs, frameTime.P95Us)
	} else {
		hint = "无时间数据（rawtrace格式不含耗时信息）"
	}

	return core.OverviewPerformance{
		FrameTime:      frameTime,
		SlowFramesTop5: topFrames,
		BottleneckHint: hint,
	}
}

func (oa *OverviewAnalyzer) buildDiagnosis() core.OverviewDiagnosis {
	return core.OverviewDiagnosis{
		TopIssues: []string{},
	}
}

func (oa *OverviewAnalyzer) buildResources() core.OverviewResources {
	var totalAPICalls int
	var drawCallCount int
	var shaderCount int
	var bufferCount int
	var bufferTotalSize int64

	for _, frame := range oa.log.Frames {
		if frame.APICallCount > 0 {
			totalAPICalls += frame.APICallCount
		} else {
			totalAPICalls += len(frame.APICalls)
		}
		shaderCount += len(frame.Shaders)
		bufferCount += len(frame.BufferCreations)
		for _, buf := range frame.BufferCreations {
			bufferTotalSize += buf.Size
		}
		if frame.DrawCallCount > 0 && len(frame.APICalls) == 0 {
			drawCallCount += frame.DrawCallCount
		} else {
			for _, call := range frame.APICalls {
				if isDrawCall(call.APIName) {
					if call.Count > 0 {
						drawCallCount += call.Count
					} else {
						drawCallCount++
					}
				}
			}
		}
	}

	frameCount := len(oa.log.Frames)
	drawCallsAvg := 0
	if frameCount > 0 {
		drawCallsAvg = drawCallCount / frameCount
	}

	return core.OverviewResources{
		ShaderCount:          shaderCount,
		BufferCount:          bufferCount,
		BufferTotalSizeBytes: bufferTotalSize,
		DrawCallsPerFrameAvg: drawCallsAvg,
		TotalAPICalls:        totalAPICalls,
	}
}

func (oa *OverviewAnalyzer) classifyBottleneck(avgUs, p95Us int64) string {
	if len(oa.log.Frames) == 0 {
		return ""
	}

	// 计算平均 SwapBuffer 和 API 时间占比
	var totalSwapUs, totalAPIUs int64
	for _, f := range oa.log.Frames {
		totalSwapUs += f.SwapBufferTimeUs
		totalAPIUs += f.APITotalTimeUs
	}
	totalUs := totalSwapUs + totalAPIUs
	if totalUs == 0 {
		totalUs = 1
	}

	swapRatio := float64(totalSwapUs) / float64(totalUs)
	apiRatio := float64(totalAPIUs) / float64(totalUs)

	// 帧时间稳定性
	if avgUs == 0 {
		return "无真实耗时数据（当前 rawtrace 未提供 frame cost/profile 耗时）"
	}
	variance := float64(p95Us-avgUs) / float64(avgUs) // p95/avg ratio

	var parts []string

	if swapRatio > 0.6 {
		parts = append(parts, "GPU-bound (SwapBuffer等待占比高)")
	} else if apiRatio > 0.7 {
		parts = append(parts, "CPU-bound (API调用耗时占比高)")
	}

	if variance > 2.0 {
		parts = append(parts, "帧率不稳定")
	}

	if len(parts) == 0 {
		parts = append(parts, "性能正常")
	}

	return strings.Join(parts, ", ")
}

func (oa *OverviewAnalyzer) buildSummary(basic core.OverviewBasic, perf core.OverviewPerformance, diag core.OverviewDiagnosis) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("%d帧%s日志", basic.FrameCount, basic.Format))

	if basic.FPS > 0 {
		parts = append(parts, fmt.Sprintf("%.1fFPS", basic.FPS))
	} else if basic.TotalTimeMs > 0 {
		parts = append(parts, fmt.Sprintf("总耗时%.1f秒", float64(basic.TotalTimeMs)/1000.0))
	} else {
		parts = append(parts, "无真实耗时数据")
	}

	if diag.TotalFindings > 0 {
		var issues []string
		if diag.Critical > 0 {
			issues = append(issues, fmt.Sprintf("%d个严重问题", diag.Critical))
		}
		if diag.High > 0 {
			issues = append(issues, fmt.Sprintf("%d个高优先级问题", diag.High))
		}
		if diag.Medium > 0 {
			issues = append(issues, fmt.Sprintf("%d个中等问题", diag.Medium))
		}
		if len(issues) > 0 {
			parts = append(parts, "发现"+strings.Join(issues, "和"))
		}
	} else {
		parts = append(parts, "基础分析完成")
	}

	return strings.Join(parts, ", ")
}

// AnalyzeWorkflow 执行预定义分析工作流
func AnalyzeWorkflow(log *core.ParsedLog, format string, workflow string) *core.WorkflowResult {
	if log == nil || len(log.Frames) == 0 {
		return nil
	}

	switch workflow {
	case "performance":
		return analyzePerformanceWorkflow(log, format)
	case "crash":
		return analyzeCrashWorkflow(log, format)
	case "rendering":
		return analyzeRenderingWorkflow(log, format)
	case "memory":
		return analyzeMemoryWorkflow(log, format)
	default:
		return nil
	}
}

func analyzePerformanceWorkflow(log *core.ParsedLog, format string) *core.WorkflowResult {
	fa := NewFrameAnalyzer(log)
	funcAna := NewFuncAnalyzer(log)

	topFrames := fa.FindTopSlowFrames(5)
	funcStats := funcAna.Analyze()
	summary := fa.GetFrameSummary()

	// 计算有真实耗时来源的帧时间，rawtrace 中未标记耗时的帧不参与性能分位数。
	sortedTimes, timedTotalUs := timedFrameTimes(log.Frames)
	sort.Slice(sortedTimes, func(i, j int) bool { return sortedTimes[i] < sortedTimes[j] })
	p95Us := int64(0)
	if len(sortedTimes) > 0 {
		p95Idx := min(int(float64(len(sortedTimes))*0.95), len(sortedTimes)-1)
		p95Us = sortedTimes[p95Idx]
	}

	var conclusion string
	var evidence []string

	if summary != nil {
		if timedTotalUs <= 0 || len(sortedTimes) == 0 {
			conclusion = "当前日志无真实耗时数据，性能工作流仅能基于调用密度定位热点"
			evidence = append(evidence, "未检测到 frame cost/profile 耗时，无法计算真实 FPS/P95")
		} else {
			fps := log.FPS
			if fps <= 0 {
				fps = float64(len(sortedTimes)) / (float64(timedTotalUs) / 1e6)
			}
			if fps < 30 {
				conclusion = fmt.Sprintf("帧率过低(%.1f FPS), 需要优化", fps)
			} else if fps < 60 {
				conclusion = fmt.Sprintf("帧率偏低(%.1f FPS), 有优化空间", fps)
			} else {
				conclusion = fmt.Sprintf("帧率正常(%.1f FPS)", fps)
			}
			evidence = append(evidence, fmt.Sprintf("有耗时来源帧: %d/%d", len(sortedTimes), len(log.Frames)))
			if log.FPS > 0 {
				evidence = append(evidence, fmt.Sprintf("日志FPS: %.1f", log.FPS))
			}
			evidence = append(evidence, fmt.Sprintf("平均帧时间: %.2f ms", float64(timedTotalUs)/float64(len(sortedTimes))/1000.0))
			evidence = append(evidence, fmt.Sprintf("最大帧时间: %.2f ms", float64(sortedTimes[len(sortedTimes)-1])/1000.0))
			evidence = append(evidence, fmt.Sprintf("P95帧时间: %.2f ms", float64(p95Us)/1000.0))
		}
	}

	// rawtrace 通常没有函数级耗时，此时按调用次数暴露热点，避免显示伪造的 0ms。
	if len(funcStats) > 0 {
		top := funcStats[0]
		if top.TotalTimeUs > 0 {
			evidence = append(evidence, fmt.Sprintf("最耗时函数: %s", formatFuncHotspot(top)))
		} else {
			evidence = append(evidence, fmt.Sprintf("调用次数最高函数: %s", formatFuncHotspot(top)))
		}
	}

	if len(topFrames) > 0 {
		if topFrames[0].TotalTimeUs > 0 {
			evidence = append(evidence, fmt.Sprintf("最慢帧: Frame %d (%.2f ms)", topFrames[0].FrameNum, float64(topFrames[0].TotalTimeUs)/1000.0))
		} else {
			count := topFrames[0].APICallCount
			if count == 0 {
				count = len(topFrames[0].APICalls)
			}
			evidence = append(evidence, fmt.Sprintf("调用密度最高帧: Frame %d (%d API调用)", topFrames[0].FrameNum, count))
		}
	}

	return &core.WorkflowResult{
		Workflow:   "performance",
		Conclusion: conclusion,
		Evidence:   evidence,
		Details: map[string]interface{}{
			"top_slow_frames": topFrames,
			"func_stats":      funcStats,
			"frame_summary":   summary,
		},
	}
}

func timedFrameTimes(frames []core.FrameInfo) ([]int64, int64) {
	times := make([]int64, 0, len(frames))
	var total int64
	for _, frame := range frames {
		if !frame.HasTiming && frame.TotalTimeUs <= 0 {
			continue
		}
		times = append(times, frame.TotalTimeUs)
		total += frame.TotalTimeUs
	}
	return times, total
}

func analyzeCrashWorkflow(log *core.ParsedLog, format string) *core.WorkflowResult {
	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(log)

	var criticalFindings []core.Finding
	var evidence []string

	for _, f := range findings {
		if f.Severity == core.SeverityCritical || f.Severity == core.SeverityHigh {
			criticalFindings = append(criticalFindings, f)
			evidence = append(evidence, fmt.Sprintf("[%s] %s: %s", f.Severity, f.Category, f.Description))
		}
	}

	// 检查 segfault
	var segfaultFrames []int
	for _, frame := range log.Frames {
		for _, call := range frame.APICalls {
			if call.APIName == "__segfault__" {
				segfaultFrames = append(segfaultFrames, frame.FrameNum)
				evidence = append(evidence, fmt.Sprintf("段错误发生在 Frame %d, Line %d", frame.FrameNum, call.LineNum))
			}
		}
	}

	conclusion := "未发现崩溃相关问题"
	if len(segfaultFrames) > 0 {
		conclusion = fmt.Sprintf("发现段错误, 涉及 %d 个帧", len(segfaultFrames))
	} else if len(criticalFindings) > 0 {
		conclusion = fmt.Sprintf("发现 %d 个可能导致崩溃的问题", len(criticalFindings))
	}

	return &core.WorkflowResult{
		Workflow:   "crash",
		Conclusion: conclusion,
		Evidence:   evidence,
		Details: map[string]interface{}{
			"critical_findings": criticalFindings,
			"segfault_frames":   segfaultFrames,
			"all_findings":      findings,
		},
	}
}

func analyzeRenderingWorkflow(log *core.ParsedLog, format string) *core.WorkflowResult {
	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(log)

	// 收集渲染相关问题
	var renderIssues []core.Finding
	var evidence []string

	renderCategories := map[string]bool{
		"shader_error": true, "null_pointer": true, "driver_error": true,
		"shader": true, "texture": true, "fbo": true,
	}

	for _, f := range findings {
		if renderCategories[f.Category] {
			renderIssues = append(renderIssues, f)
			evidence = append(evidence, fmt.Sprintf("[%s] %s", f.Category, f.Description))
		}
	}

	sa := NewShaderAnalyzer(log)
	shaderSummary := sa.GetShaderSummary()
	if shaderSummary != nil {
		evidence = append(evidence, fmt.Sprintf("Shader编译: %d次, 总耗时%.2f ms", shaderSummary.TotalCompile, float64(shaderSummary.TotalTimeUs)/1000.0))
	}

	conclusion := "未发现渲染相关问题"
	if len(renderIssues) > 0 {
		conclusion = fmt.Sprintf("发现 %d 个渲染相关问题", len(renderIssues))
	}

	return &core.WorkflowResult{
		Workflow:   "rendering",
		Conclusion: conclusion,
		Evidence:   evidence,
		Details: map[string]interface{}{
			"render_issues":  renderIssues,
			"shader_summary": shaderSummary,
			"total_shaders":  len(log.Frames) > 0 && len(log.Frames[0].Shaders) > 0,
		},
	}
}

func analyzeMemoryWorkflow(log *core.ParsedLog, format string) *core.WorkflowResult {
	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(log)

	var memoryIssues []core.Finding
	var evidence []string

	memoryCategories := map[string]bool{
		"resource_leak": true, "memory": true, "buffer": true,
	}

	for _, f := range findings {
		if memoryCategories[f.Category] {
			memoryIssues = append(memoryIssues, f)
			evidence = append(evidence, fmt.Sprintf("[%s] %s", f.Category, f.Description))
		}
	}

	// Buffer 统计
	ba := NewBufferAnalyzer(log)
	bufferSummary := ba.GetBufferSummary()
	if bufferSummary != nil {
		evidence = append(evidence, fmt.Sprintf("Buffer: %d个, 总大小%.2f KB", bufferSummary.TotalCount, float64(bufferSummary.TotalSize)/1024.0))
	}

	conclusion := "未发现内存相关问题"
	if len(memoryIssues) > 0 {
		conclusion = fmt.Sprintf("发现 %d 个内存/资源问题", len(memoryIssues))
	}

	return &core.WorkflowResult{
		Workflow:   "memory",
		Conclusion: conclusion,
		Evidence:   evidence,
		Details: map[string]interface{}{
			"memory_issues":  memoryIssues,
			"buffer_summary": bufferSummary,
		},
	}
}
