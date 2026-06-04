package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"gst/internal/core"
	"gst/internal/core/analyzer"
	"gst/internal/core/bug"
	"gst/internal/core/exporter"
	"gst/internal/core/glstats"
	"gst/internal/core/parser"
	"gst/internal/core/search"
	"gst/internal/platform"
)

var (
	parseCmd       = flag.String("parse", "", "Parse log file")
	searchCmd      = flag.String("search", "", "Search keyword in log file")
	timeRange      = flag.String("time", "", "Time range search: startUs,endUs (e.g., 1000,50000)")
	topFrames      = flag.Int("top", 0, "Show top N slowest frames (0=disabled)")
	funcStats      = flag.Bool("funcs", false, "Show function statistics")
	shaderStats    = flag.Bool("shader", false, "Show shader statistics")
	exportFmt      = flag.String("export", "", "Export format: txt|csv|json")
	output         = flag.String("output", "", "Output file (default: stdout)")
	diagnose       = flag.Bool("diagnose", false, "Run bug diagnosis on log file (Markdown)")
	diagnoseJSON   = flag.Bool("diagnose-json", false, "Run bug diagnosis and output JSON")
	aiSummary      = flag.Bool("ai-summary", false, "Output AI-friendly case-level OpenGL statistics as JSON")
	frameStatsID   = flag.Int("frame-stats", -1, "Output one frame's OpenGL statistics as JSON")
	overviewFlag   = flag.Bool("overview", false, "Output comprehensive overview as JSON")
	drawcallsFlag  = flag.Bool("drawcalls", false, "Output draw call analysis as JSON")
	texturesFlag   = flag.Bool("textures", false, "Output texture lifecycle analysis as JSON")
	traceProgsFlag = flag.Bool("trace-programs", false, "Output trace inspector program/shader registry as JSON")
	frameRawID     = flag.Int("frame-raw", -1, "Output raw log lines for a specific frame as JSON")
	frameDistFlag  = flag.Bool("frame-distribution", false, "Output frame time distribution with clustering as JSON")
	help           = flag.Bool("help", false, "Show help")
	verbose        = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	if *verbose {
		platform.InitLogger("debug")
	} else {
		platform.InitLogger("warn")
	}

	if *help || flag.NFlag() == 0 {
		printHelp()
		os.Exit(0)
	}

	// Bug 诊断 — 独立模式，运行完直接退出
	if *diagnose && *parseCmd != "" {
		runDiagnose(*parseCmd)
		return
	}

	// JSON 诊断
	if *diagnoseJSON && *parseCmd != "" {
		printDiagnoseJSON(*parseCmd)
		return
	}

	if *aiSummary && *parseCmd != "" {
		printAISummary(*parseCmd)
		return
	}

	if *frameStatsID >= 0 && *parseCmd != "" {
		printFrameStats(*parseCmd, *frameStatsID)
		return
	}

	if *overviewFlag && *parseCmd != "" {
		printOverview(*parseCmd)
		return
	}

	if *drawcallsFlag && *parseCmd != "" {
		printDrawCalls(*parseCmd)
		return
	}

	if *texturesFlag && *parseCmd != "" {
		printTextures(*parseCmd)
		return
	}

	if *traceProgsFlag && *parseCmd != "" {
		printTracePrograms(*parseCmd)
		return
	}

	if *frameRawID >= 0 && *parseCmd != "" {
		printFrameRaw(*parseCmd, *frameRawID)
		return
	}

	if *frameDistFlag && *parseCmd != "" {
		printFrameDistribution(*parseCmd)
		return
	}

	// 解析日志文件
	if *parseCmd != "" {
		parseLog(*parseCmd)
	}

	// 关键字检索
	if *searchCmd != "" && *parseCmd != "" {
		searchKeyword(*parseCmd, *searchCmd)
	}

	// 时间段检索
	if *timeRange != "" && *parseCmd != "" {
		searchTimeRange(*parseCmd, *timeRange)
	}

	// 帧分析 — 仅在没有 JSON 分析命令时运行，避免重复解析
	hasJSONCmd := *aiSummary || *overviewFlag || *drawcallsFlag || *texturesFlag ||
		*traceProgsFlag || *diagnose || *diagnoseJSON || *frameStatsID >= 0 || *frameRawID >= 0
	if *topFrames > 0 && *parseCmd != "" && !hasJSONCmd {
		analyzeFrames(*parseCmd, *topFrames)
	}

	// 函数统计
	if *funcStats && *parseCmd != "" {
		showFuncStats(*parseCmd)
	}

	// Shader 统计
	if *shaderStats && *parseCmd != "" {
		showShaderStats(*parseCmd)
	}

	// 导出
	if *exportFmt != "" && *parseCmd != "" {
		exportResults(*parseCmd, *exportFmt, *output)
	}

}

func printHelp() {
	fmt.Print(`GST CLI - GPU Support Toolkit 命令行工具

用法:
  gst [选项] -parse <文件>        解析日志文件
  gst -search <关键字> -parse <文件>   关键字检索
  gst -time <start,end> -parse <文件>  时间段检索
  gst -top <N> -parse <文件>      显示最慢的N帧
  gst -funcs -parse <文件>        显示函数统计
  gst -shader -parse <文件>       显示Shader统计
  gst -export <格式> -parse <文件>  导出结果

兼容入口:
  gst-cli 与 gst 等价

AI 分析（JSON 输出）:
  gst -ai-summary -parse <文件>           输出 case 级 OpenGL 统计 JSON
  gst -frame-stats <N> -parse <文件>      输出单帧 OpenGL 统计 JSON
  gst -overview -parse <文件>             输出综合概览 JSON
  gst -diagnose-json -parse <文件>        输出资源泄漏/Shader错误/GL错误检测 JSON
  gst -drawcalls -parse <文件>            输出 draw call 分析 JSON
  gst -textures -parse <文件>             输出纹理分析 JSON
  gst -trace-programs -parse <文件>       输出 program/shader 注册表 JSON
  gst -frame-raw <N> -parse <文件>       输出帧原始日志 JSON

选项:
  -parse <文件>            解析指定的日志文件
  -search <关键字>         搜索关键字（支持多个，空格分隔）
  -time <start,end>        时间范围（微秒）
  -top <N>                显示最慢的N帧 (默认: 10)
  -funcs                  显示函数调用统计
  -shader                 显示Shader统计
  -export <格式>          导出格式: txt, csv, json
  -output <文件>          输出文件 (默认: stdout)
  -diagnose               运行 Bug 诊断（Markdown 输出）
  -diagnose-json          输出资源泄漏/Shader错误/GL错误检测 JSON
  -ai-summary             输出 case 级 OpenGL 统计 JSON
  -frame-stats <N>        输出单帧 OpenGL 分类和重点 API 统计 JSON
  -overview               输出综合概览 JSON
  -drawcalls              输出 draw call 分析 JSON
  -textures               输出纹理生命周期分析 JSON
  -trace-programs         输出 program/shader 注册表 JSON
  -frame-raw <N>          输出帧 N 的原始日志 JSON（小帧完整，大帧仅关键API）
  -verbose                启用详细日志
  -help                   显示帮助

示例:
  gst -parse trace.api.txt
  gst -parse trace.api.txt -search "glDrawElements"
  gst -parse trace.api.txt -time "1000,50000"
  gst -parse trace.api.txt -top 20
  gst -parse trace.api.txt -funcs
  gst -parse trace.api.txt -export json -output result.json
  gst -diagnose -parse trace.api.txt
  gst -ai-summary -parse trace.api.txt
  gst -frame-stats 423 -parse trace.api.txt
  gst -overview -parse trace.api.txt
  gst -diagnose-json -parse trace.api.txt
  gst -drawcalls -parse trace.api.txt
  gst -textures -parse trace.api.txt
  gst -trace-programs -parse trace.api.txt
  gst -frame-raw 137 -parse trace.api.txt
`)
}

func parseLog(filePath string) {
	fmt.Printf("=== 解析日志文件: %s ===\n\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("无法打开文件", "path", filePath, "error", err)
		fmt.Fprintf(os.Stderr, "错误: 无法打开文件: %v\n", err)
		os.Exit(1)
	}

	// 使用改进的检测函数扫描前100行找到第一个有效行
	kind := parser.DetectKindFromReader(file, 100)
	file.Close()

	fmt.Printf("检测类型: %s\n", kind)

	// 重新打开文件解析
	file, err = os.Open(filePath)
	if err != nil {
		slog.Error("无法打开文件", "path", filePath, "error", err)
		return
	}
	defer file.Close()
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	fmt.Printf("解析结果:\n")
	fmt.Printf("  总帧数: %d\n", len(parsed.Frames))
	fmt.Printf("  FPS: %.1f\n", parsed.FPS)
	if len(parsed.Frames) > 0 {
		fmt.Printf("  首帧耗时: %d us\n", parsed.Frames[0].TotalTimeUs)
		fmt.Printf("  末帧耗时: %d us\n", parsed.Frames[len(parsed.Frames)-1].TotalTimeUs)
	}
	fmt.Println()
}

func searchKeyword(filePath string, keyword string) {
	if keyword == "" {
		fmt.Println("错误: -search 需要指定关键字")
		return
	}

	keywords := strings.Split(keyword, " ")
	fmt.Printf("=== 关键字检索: %s ===\n\n", keyword)

	// Parse log to get frame boundaries for frame number mapping
	parsed, _ := parseLogFile(filePath)
	type frameRange struct {
		num, start, end int
	}
	var frames []frameRange
	if parsed != nil {
		for _, f := range parsed.Frames {
			s := f.FullStartLine
			e := f.FullEndLine
			if s == 0 {
				s = f.StartLine
			}
			if e == 0 {
				e = f.EndLine
			}
			frames = append(frames, frameRange{f.FrameNum, s, e})
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("错误: 无法打开文件: %v\n", err)
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	ks := search.NewKeywordSearch()
	results := ks.Search(keywords, lines)

	findFrame := func(lineNum int) int {
		for _, fr := range frames {
			if lineNum >= fr.start && lineNum <= fr.end {
				return fr.num
			}
		}
		return -1
	}

	fmt.Printf("找到 %d 条匹配结果:\n\n", len(results))
	for i, r := range results {
		if i >= 100 {
			fmt.Printf("  ... 还有 %d 条结果\n", len(results)-100)
			break
		}
		fn := findFrame(r.LineNum)
		if fn >= 0 {
			fmt.Printf("[Frame %d, line %d] %s\n", fn, r.LineNum, r.Content)
		} else {
			fmt.Printf("[line %d] %s\n", r.LineNum, r.Content)
		}
	}
	fmt.Println()
}

func searchTimeRange(filePath string, timeRangeStr string) {
	parts := strings.Split(timeRangeStr, ",")
	if len(parts) != 2 {
		fmt.Println("错误: -time 格式应为 startUs,endUs (如: 1000,50000)")
		return
	}

	startUs, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		fmt.Printf("错误: 无效的 startUs: %v\n", err)
		return
	}
	endUs, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		fmt.Printf("错误: 无效的 endUs: %v\n", err)
		return
	}

	fmt.Printf("=== 时间段检索: [%d, %d] us ===\n\n", startUs, endUs)

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("无法打开文件", "path", filePath, "error", err)
		return
	}
	defer file.Close()
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	trs := search.NewTimeRangeSearch()
	results := trs.Search(parsed, startUs, endUs)

	fmt.Printf("找到 %d 条匹配结果:\n\n", len(results))
	for _, r := range results {
		fmt.Printf("[%d us] %s (count=%d)\n", r.TimeUs, r.APIName, r.Count)
	}
	fmt.Println()
}

func analyzeFrames(filePath string, topN int) {
	fmt.Printf("=== 帧分析 (Top %d) ===\n\n", topN)

	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}
	if parsed == nil {
		fmt.Println("错误: 解析结果为空")
		return
	}

	fa := analyzer.NewFrameAnalyzer(parsed)
	topFrames := fa.FindTopSlowFrames(topN)
	summary := fa.GetFrameSummary()
	if summary == nil {
		fmt.Println("错误: 无法生成统计摘要")
		return
	}

	fmt.Printf("统计摘要:\n")
	fmt.Printf("  总帧数: %d\n", summary.TotalFrames)
	fmt.Printf("  平均耗时: %d us\n", summary.AvgTimeUs)
	fmt.Printf("  最大耗时: %d us\n", summary.MaxTimeUs)
	fmt.Printf("  最小耗时: %d us\n", summary.MinTimeUs)
	fmt.Println()

	fmt.Printf("Top %d 最慢帧:\n\n", len(topFrames))
	for i, f := range topFrames {
		gapUs := f.TotalTimeUs - f.APITotalTimeUs - f.SwapBufferTimeUs
		if gapUs < 0 {
			gapUs = 0
		}
		gapPct := float64(0)
		if f.TotalTimeUs > 0 {
			gapPct = float64(gapUs) / float64(f.TotalTimeUs) * 100
		}
		apiCallCount := f.APICallCount
		if apiCallCount == 0 && len(f.APICalls) > 0 {
			apiCallCount = len(f.APICalls)
		}
		fmt.Printf("[%d] Frame %d: %d us (%d ms)\n",
			i+1, f.FrameNum, f.TotalTimeUs, f.TotalTimeUs/1000)
		fmt.Printf("    API调用数: %d, DrawCalls: %d, Gap: %d us (%.1f%%)\n",
			apiCallCount, f.DrawCallCount, gapUs, gapPct)
		if f.APITotalTimeUs > 0 {
			fmt.Printf("    API时间: %d us, Swap: %d us\n",
				f.APITotalTimeUs, f.SwapBufferTimeUs)
		}
		fmt.Println()
	}
}

func showFuncStats(filePath string) {
	fmt.Println("=== 函数统计 ===")
	fmt.Println()

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("无法打开文件", "path", filePath, "error", err)
		return
	}
	defer file.Close()
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	funcAnalyzer := analyzer.NewFuncAnalyzer(parsed)
	stats := funcAnalyzer.Analyze()

	fmt.Printf("%-30s %10s %15s %15s\n", "函数名", "调用次数", "总耗时(us)", "平均耗时(us)")
	fmt.Println(strings.Repeat("-", 75))

	for _, s := range stats {
		fmt.Printf("%-30s %10d %15d %15d\n",
			s.FuncName, s.CallCount, s.TotalTimeUs, s.AvgTimeUs)
	}
	fmt.Println()
}

func showShaderStats(filePath string) {
	fmt.Println("=== Shader 统计 ===")
	fmt.Println()

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("无法打开文件", "path", filePath, "error", err)
		return
	}
	defer file.Close()
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	shaderAnalyzer := analyzer.NewShaderAnalyzer(parsed)
	stats := shaderAnalyzer.Analyze()

	if len(stats) == 0 {
		fmt.Println("未检测到 Shader 相关调用")
		return
	}

	fmt.Printf("%-20s %15s %15s\n", "类型", "编译次数", "总耗时(us)")
	fmt.Println(strings.Repeat("-", 55))

	for _, s := range stats {
		fmt.Printf("%-20s %15d %15d\n",
			s.Type, s.CompileCount, s.TotalCompileTimeUs)
	}
	fmt.Println()
}

func exportResults(filePath string, format string, outputPath string) {
	fmt.Printf("=== 导出结果 (格式: %s) ===\n\n", format)

	file, err := os.Open(filePath)
	if err != nil {
		slog.Error("无法打开文件", "path", filePath, "error", err)
		return
	}
	defer file.Close()
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	var buf bytes.Buffer

	switch format {
	case "txt":
		exp := exporter.TXTExporter{}
		if err := exp.Export(&buf); err != nil {
			slog.Error("导出 TXT 失败", "error", err)
			return
		}
	case "csv":
		exp := exporter.FramesCSVExporter{Frames: parsed.Frames}
		if err := exp.Export(&buf); err != nil {
			slog.Error("导出 CSV 失败", "error", err)
			return
		}
	case "json":
		exp := exporter.JSONExporter[*core.ParsedLog]{Data: parsed}
		if err := exp.Export(&buf); err != nil {
			slog.Error("导出 JSON 失败", "error", err)
			return
		}
	default:
		fmt.Printf("错误: 不支持的格式: %s (支持: txt, csv, json)\n", format)
		return
	}

	if outputPath != "" {
		os.WriteFile(outputPath, buf.Bytes(), 0644)
		fmt.Printf("已导出到: %s\n", outputPath)
	} else {
		fmt.Print(buf.String())
	}
}

// compactCaseStats strips per-frame category/key/top details from CaseStats
// for AI consumption — keeps top frames lightweight (just frame_num, time, counts).
type compactFrameBrief struct {
	FrameNum      int    `json:"frame_num"`
	TotalTimeUs   int64  `json:"total_time_us"`
	APICallCount  int    `json:"api_call_count"`
	DrawCallCount int    `json:"draw_call_count"`
	HasTiming     bool   `json:"has_timing"`
	TimingSource  string `json:"timing_source,omitempty"`
}

type compactCaseStats struct {
	FrameCount      int                       `json:"frame_count"`
	TotalTimeUs     int64                     `json:"total_time_us"`
	FPS             float64                   `json:"fps"`
	HasTiming       bool                      `json:"has_timing"`
	CategoryStats   []glstats.CategoryCounter `json:"category_stats"`
	KeyAPIs         []glstats.APICounter      `json:"key_apis"`
	TopAPIs         []glstats.APICounter      `json:"top_apis"`
	TopFramesByTime []compactFrameBrief       `json:"top_frames_by_time"`
	TopFramesByDraw []compactFrameBrief       `json:"top_frames_by_draw"`
	AIContract      glstats.AIContract        `json:"ai_contract"`
	Bottleneck      *glstats.CaseBottleneck   `json:"bottleneck,omitempty"`
}

func toCompactCaseStats(cs glstats.CaseStats) compactCaseStats {
	topByTime := make([]compactFrameBrief, len(cs.TopFramesByTime))
	for i, f := range cs.TopFramesByTime {
		topByTime[i] = compactFrameBrief{
			FrameNum: f.FrameNum, TotalTimeUs: f.TotalTimeUs,
			APICallCount: f.APICallCount, DrawCallCount: f.DrawCallCount,
			HasTiming: f.HasTiming, TimingSource: f.TimingSource,
		}
	}
	topByDraw := make([]compactFrameBrief, len(cs.TopFramesByDraw))
	for i, f := range cs.TopFramesByDraw {
		topByDraw[i] = compactFrameBrief{
			FrameNum: f.FrameNum, TotalTimeUs: f.TotalTimeUs,
			APICallCount: f.APICallCount, DrawCallCount: f.DrawCallCount,
			HasTiming: f.HasTiming, TimingSource: f.TimingSource,
		}
	}
	return compactCaseStats{
		FrameCount: cs.FrameCount, TotalTimeUs: cs.TotalTimeUs,
		FPS: cs.FPS, HasTiming: cs.HasTiming,
		CategoryStats: cs.CategoryStats, KeyAPIs: cs.KeyAPIs, TopAPIs: cs.TopAPIs,
		TopFramesByTime: topByTime, TopFramesByDraw: topByDraw,
		AIContract: cs.AIContract,
		Bottleneck:      cs.Bottleneck,
	}
}

func printAISummary(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	cs := glstats.AnalyzeCase(parsed, 10)
	compact := toCompactCaseStats(cs)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(compact); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

func printFrameStats(filePath string, frameID int) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	for _, frame := range parsed.Frames {
		if frame.FrameNum == frameID {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(glstats.AnalyzeFrame(frame)); err != nil {
				fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "错误: 未找到帧 %d\n", frameID)
}

func parseLogFile(filePath string) (*core.ParsedLog, error) {
	parsed, _, err := parseLogFileWithFormat(filePath)
	return parsed, err
}

func parseLogFileWithFormat(filePath string) (*core.ParsedLog, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	kind := parser.DetectKindFromReader(file, 5000)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, "", err
	}
	if kind == parser.KindRawTrace || kind == parser.KindUnknown {
		parsed, err := parser.ParseIndexedRawTraceFile(filePath)
		if err == nil && len(parsed.Frames) > 0 {
			return parsed, string(kind), nil
		}
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
			return nil, "", seekErr
		}
	}
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	return parsed, string(kind), err
}

func hydrateLogForDiagnosis(log *core.ParsedLog) error {
	return parser.HydrateIndexedAPICalls(log)
}

func runDiagnose(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		slog.Error("解析失败", "error", err)
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	if err := hydrateLogForDiagnosis(parsed); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 诊断数据加载失败: %v\n", err)
		return
	}

	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(parsed)

	markdown := bug.GenerateMarkdownReport(findings, filePath)
	fmt.Print(markdown)
}

func printDiagnoseJSON(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	if err := hydrateLogForDiagnosis(parsed); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 诊断数据加载失败: %v\n", err)
		return
	}
	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(parsed)
	report := bug.GenerateReport(filePath, findings)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

func printOverview(filePath string) {
	parsed, format, err := parseLogFileWithFormat(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	if err := hydrateLogForDiagnosis(parsed); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 诊断数据加载失败: %v\n", err)
		return
	}
	oa := analyzer.NewOverviewAnalyzer(parsed, format)
	result := oa.Analyze()
	stripOverviewAPICalls(result)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

func stripOverviewAPICalls(result *core.OverviewResult) {
	if result == nil {
		return
	}
	for i := range result.Performance.SlowFramesTop5 {
		result.Performance.SlowFramesTop5[i].APICalls = nil
	}
}

type compactDrawCallBrief struct {
	FrameNum       int   `json:"frame_num"`
	TotalDrawCalls int   `json:"total_draw_calls"`
	TimeUs         int64 `json:"time_us"`
	HasTiming      bool  `json:"has_timing"`
}

type compactDrawCallSummary struct {
	TotalDrawCalls    int64                  `json:"total_draw_calls"`
	DrawCallsPerFrame float64                `json:"draw_calls_per_frame_avg"`
	HasTiming         bool                   `json:"has_timing"`
	ByType            map[string]int         `json:"by_type"`
	TypeRatio         map[string]float64     `json:"type_ratio,omitempty"`
	TopFramesByDC     []compactDrawCallBrief `json:"top_frames_by_draw_count"`
	TopFramesByTime   []compactDrawCallBrief `json:"top_frames_by_time"`
	StartupFrameSkip  int                    `json:"startup_frame_skip"`
}

func printDrawCalls(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	dca := analyzer.NewDrawCallAnalyzer(parsed)
	summary := dca.GetSummary()
	if summary == nil {
		fmt.Fprintf(os.Stderr, "错误: 未找到 draw call 数据\n")
		return
	}

	total := int64(summary.TotalDrawCalls)
	frameCount := len(summary.Frames)
	if frameCount == 0 {
		frameCount = 1
	}

	typeRatio := make(map[string]float64)
	for k, v := range summary.ByType {
		if total > 0 {
			typeRatio[k] = float64(v) / float64(total)
		}
	}

	startupSkip := 3
	if frameCount <= startupSkip {
		startupSkip = 0
	}

	topByDC := make([]compactDrawCallBrief, 0, 10)
	sorted := make([]core.DrawCallStats, len(summary.Frames))
	copy(sorted, summary.Frames)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TotalDrawCalls > sorted[j].TotalDrawCalls
	})
	for i, f := range sorted {
		if i >= 10 {
			break
		}
		topByDC = append(topByDC, compactDrawCallBrief{
			FrameNum: f.FrameNum, TotalDrawCalls: f.TotalDrawCalls,
			TimeUs: f.TimeUs, HasTiming: f.HasTiming,
		})
	}

	topByTime := make([]compactDrawCallBrief, 0, 10)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TimeUs > sorted[j].TimeUs
	})
	for i, f := range sorted {
		if i >= 10 {
			break
		}
		topByTime = append(topByTime, compactDrawCallBrief{
			FrameNum: f.FrameNum, TotalDrawCalls: f.TotalDrawCalls,
			TimeUs: f.TimeUs, HasTiming: f.HasTiming,
		})
	}

	compact := compactDrawCallSummary{
		TotalDrawCalls:    total,
		DrawCallsPerFrame: summary.DrawCallsPerFrameAvg,
		HasTiming:         summary.HasTiming,
		ByType:            summary.ByType,
		TypeRatio:         typeRatio,
		TopFramesByDC:     topByDC,
		TopFramesByTime:   topByTime,
		StartupFrameSkip:  startupSkip,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(compact); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

func printTextures(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	ta := analyzer.NewTextureAnalyzer(parsed)
	summary := ta.GetSummary()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

func printTracePrograms(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	tia := analyzer.NewTraceInspectorAnalyzer(parsed)
	result := tia.Analyze()
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

type frameRawSummary struct {
	FrameNum      int            `json:"frame_num"`
	StartLine     int            `json:"start_line"`
	EndLine       int            `json:"end_line"`
	LineCount     int            `json:"line_count"`
	GLCallCount   int            `json:"gl_call_count"`
	AppLogCount   int            `json:"app_log_count"`
	KeyAPISummary *keyAPISummary `json:"key_api_summary"`
	OutputFile    string         `json:"output_file"`
}

type keyAPICategorySummary struct {
	Count int            `json:"count"`
	Types map[string]int `json:"types,omitempty"`
}

type keyAPISummary struct {
	Draw            *keyAPICategorySummary `json:"draw,omitempty"`
	Buffer          *keyAPICategorySummary `json:"buffer,omitempty"`
	Texture         *keyAPICategorySummary `json:"texture,omitempty"`
	Readback        *keyAPICategorySummary `json:"readback,omitempty"`
	Sync            *keyAPICategorySummary `json:"sync,omitempty"`
	ContextSwitches int                    `json:"context_switches"`
}

func printFrameRaw(filePath string, frameID int) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}

	var targetFrame *core.FrameInfo
	for _, f := range parsed.Frames {
		if f.FrameNum == frameID {
			targetFrame = &f
			break
		}
	}
	if targetFrame == nil {
		fmt.Fprintf(os.Stderr, "错误: 未找到帧 %d\n", frameID)
		return
	}

	startLine := targetFrame.FullStartLine
	endLine := targetFrame.FullEndLine
	if startLine == 0 {
		startLine = targetFrame.StartLine
	}
	if endLine == 0 {
		endLine = targetFrame.EndLine
	}
	lineCount := endLine - startLine + 1

	outputPath := *output
	if outputPath == "" {
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("gst-frame-%d-*.log", frameID))
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: 创建临时文件失败: %v\n", err)
			return
		}
		outputPath = tmpFile.Name()
		tmpFile.Close()
	}

	hasOffsets := targetFrame.FullStartOffset > 0 || targetFrame.StartOffset > 0
	if hasOffsets {
		if err := copyIndexedFrameToFile(outputPath, filePath, targetFrame); err != nil {
			_ = os.Remove(outputPath)
			fmt.Fprintf(os.Stderr, "错误: 写入帧数据失败: %v\n", err)
			return
		}
	} else {
		if err := writeFrameFromLineScan(outputPath, filePath, startLine, endLine); err != nil {
			_ = os.Remove(outputPath)
			fmt.Fprintf(os.Stderr, "错误: 写入帧数据失败: %v\n", err)
			return
		}
	}

	summary := computeKeyAPISummaryFromFrame(targetFrame)
	glCallCount := 0
	for _, s := range targetFrame.APISummary {
		glCallCount += s.Count
	}
	if glCallCount == 0 {
		for _, c := range targetFrame.APICalls {
			cnt := c.Count
			if cnt <= 0 {
				cnt = 1
			}
			glCallCount += cnt
		}
	}
	if glCallCount == 0 {
		glCallCount = targetFrame.APICallCount
	}

	result := frameRawSummary{
		FrameNum:      frameID,
		StartLine:     startLine,
		EndLine:       endLine,
		LineCount:     lineCount,
		GLCallCount:   glCallCount,
		AppLogCount:   lineCount - glCallCount,
		KeyAPISummary: summary,
		OutputFile:    outputPath,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}

func copyIndexedFrameToFile(outputPath, sourcePath string, frame *core.FrameInfo) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "=== Frame %d (lines %d-%d, %d lines) ===\n",
		frame.FrameNum,
		frame.FullStartLine, frame.FullEndLine,
		frame.FullEndLine-frame.FullStartLine+1)

	return parser.CopyIndexedFrameRawLog(f, sourcePath, *frame)
}

func writeFrameFromLineScan(outputPath, filePath string, startLine, endLine int) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(outFile, "=== Frame (lines %d-%d, %d lines) ===\n",
		startLine, endLine, endLine-startLine+1)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum < startLine {
			continue
		}
		if lineNum > endLine {
			break
		}
		fmt.Fprintln(outFile, strings.TrimRight(scanner.Text(), "\r"))
	}
	return scanner.Err()
}

func computeKeyAPISummaryFromFrame(frame *core.FrameInfo) *keyAPISummary {
	var s keyAPISummary
	classifyAPI := func(name string, count int) {
		switch {
		case strings.HasPrefix(name, "glDraw"):
			if s.Draw == nil {
				s.Draw = &keyAPICategorySummary{Types: make(map[string]int)}
			}
			s.Draw.Count += count
			s.Draw.Types[name] += count
		case strings.HasPrefix(name, "glBuffer"):
			if s.Buffer == nil {
				s.Buffer = &keyAPICategorySummary{Types: make(map[string]int)}
			}
			s.Buffer.Count += count
			s.Buffer.Types[name] += count
		case strings.HasPrefix(name, "glTexImage") || strings.HasPrefix(name, "glTexSubImage"):
			if s.Texture == nil {
				s.Texture = &keyAPICategorySummary{Types: make(map[string]int)}
			}
			s.Texture.Count += count
			s.Texture.Types[name] += count
		case strings.HasPrefix(name, "glReadPixels"):
			if s.Readback == nil {
				s.Readback = &keyAPICategorySummary{Types: make(map[string]int)}
			}
			s.Readback.Count += count
			s.Readback.Types[name] += count
		case strings.HasPrefix(name, "glFence") || strings.HasPrefix(name, "glClientWait") ||
			strings.HasPrefix(name, "glFinish") || strings.HasPrefix(name, "glFlush"):
			if s.Sync == nil {
				s.Sync = &keyAPICategorySummary{Types: make(map[string]int)}
			}
			s.Sync.Count += count
			s.Sync.Types[name] += count
		case strings.HasPrefix(name, "glXMakeContextCurrent"):
			s.ContextSwitches += count
		}
	}

	for _, entry := range frame.APISummary {
		if entry == nil || entry.APIName == "" {
			continue
		}
		classifyAPI(entry.APIName, entry.Count)
	}
	for _, call := range frame.APICalls {
		if call.APIName == "" {
			continue
		}
		count := call.Count
		if count <= 0 {
			count = 1
		}
		classifyAPI(call.APIName, count)
	}

	return &s
}

func printFrameDistribution(filePath string) {
	parsed, err := parseLogFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 解析失败: %v\n", err)
		return
	}
	da := analyzer.NewDistributionAnalyzer(parsed)
	result := da.Analyze()
	if result == nil {
		fmt.Fprintf(os.Stderr, "错误: 无帧数据\n")
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "错误: JSON 输出失败: %v\n", err)
	}
}
