package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gst/internal/core/analyzer"
	"gst/internal/core/exporter"
	"gst/internal/core/parser"
	"gst/internal/core/search"
)

var (
	// 命令 flags
	parseCmd    = flag.String("parse", "", "Parse log file")
	searchCmd   = flag.String("search", "", "Search keyword in log file")
	timeRange   = flag.String("time", "", "Time range search: startUs,endUs (e.g., 1000,50000)")
	topFrames   = flag.Int("top", 10, "Show top N slowest frames")
	funcStats   = flag.Bool("funcs", false, "Show function statistics")
	shaderStats = flag.Bool("shader", false, "Show shader statistics")
	exportFmt   = flag.String("export", "", "Export format: txt|csv|json")
	output      = flag.String("output", "", "Output file (default: stdout)")
	help        = flag.Bool("help", false, "Show help")
)

func main() {
	flag.Parse()

	if *help || flag.NFlag() == 0 {
		printHelp()
		os.Exit(0)
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

	// 帧分析
	if *topFrames > 0 && *parseCmd != "" {
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
	fmt.Println(`GST CLI - GPU Support Toolkit 命令行工具

用法:
  gst-cli [选项] -parse <文件>        解析日志文件
  gst-cli -search <关键字> -parse <文件>   关键字检索
  gst-cli -time <start,end> -parse <文件>  时间段检索
  gst-cli -top <N> -parse <文件>      显示最慢的N帧
  gst-cli -funcs -parse <文件>        显示函数统计
  gst-cli -shader -parse <文件>       显示Shader统计
  gst-cli -export <格式> -parse <文件>  导出结果

选项:
  -parse <文件>      解析指定的日志文件
  -search <关键字>   搜索关键字（支持多个，空格分隔）
  -time <start,end>  时间范围（微秒）
  -top <N>          显示最慢的N帧 (默认: 10)
  -funcs            显示函数调用统计
  -shader           显示Shader统计
  -export <格式>    导出格式: txt, csv, json
  -output <文件>    输出文件 (默认: stdout)
  -help             显示帮助

示例:
  gst-cli -parse trace.api.txt
  gst-cli -parse trace.api.txt -search "glDrawElements"
  gst-cli -parse trace.api.txt -time "1000,50000"
  gst-cli -parse trace.api.txt -top 20
  gst-cli -parse trace.api.txt -funcs
  gst-cli -parse trace.api.txt -export json -output result.json
`)
}

func parseLog(filePath string) {
	fmt.Printf("=== 解析日志文件: %s ===\n\n", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("错误: 无法打开文件: %v\n", err)
		return
	}

	// 使用改进的检测函数扫描前100行找到第一个有效行
	kind := parser.DetectKindFromReader(file, 100)
	file.Close()

	fmt.Printf("检测类型: %s\n", kind)

	// 重新打开文件解析
	file, _ = os.Open(filePath)
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	file.Close()

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

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("错误: 无法打开文件: %v\n", err)
		return
	}
	defer file.Close()

	// 读取所有行
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	ks := search.NewKeywordSearch()
	results := ks.Search(keywords, lines)

	fmt.Printf("找到 %d 条匹配结果:\n\n", len(results))
	for i, r := range results {
		if i >= 100 {
			fmt.Printf("  ... 还有 %d 条结果\n", len(results)-100)
			break
		}
		fmt.Printf("[%d] %s\n", r.LineNum, r.Content)
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

	file, _ := os.Open(filePath)
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	file.Close()

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

	file, _ := os.Open(filePath)
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	file.Close()

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	fa := analyzer.NewFrameAnalyzer(parsed)
	topFrames := fa.FindTopSlowFrames(topN)
	summary := fa.GetFrameSummary()

	fmt.Printf("统计摘要:\n")
	fmt.Printf("  总帧数: %d\n", summary["total_frames"])
	fmt.Printf("  平均耗时: %d us\n", summary["avg_time_us"])
	fmt.Printf("  最大耗时: %d us\n", summary["max_time_us"])
	fmt.Printf("  最小耗时: %d us\n", summary["min_time_us"])
	fmt.Println()

	fmt.Printf("Top %d 最慢帧:\n\n", len(topFrames))
	for i, f := range topFrames {
		fmt.Printf("[%d] Frame %d: %d us (%d ms)\n",
			i+1, f.FrameNum, f.TotalTimeUs, f.TotalTimeUs/1000)
		fmt.Printf("    API调用数: %d\n", len(f.APICalls))
		if len(f.APICalls) > 0 {
			topCall := f.APICalls[0]
			for _, c := range f.APICalls {
				if c.TimeUs > topCall.TimeUs {
					topCall = c
				}
			}
			fmt.Printf("    最耗时调用: %s (%d us)\n", topCall.APIName, topCall.TimeUs)
		}
		fmt.Println()
	}
}

func showFuncStats(filePath string) {
	fmt.Println("=== 函数统计 ===\n")

	file, _ := os.Open(filePath)
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	file.Close()

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
	fmt.Println("=== Shader 统计 ===\n")

	file, _ := os.Open(filePath)
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	file.Close()

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

	file, _ := os.Open(filePath)
	kind := parser.DetectKind("")
	p := parser.CreateParser(kind)
	parsed, err := p.Parse(file)
	file.Close()

	if err != nil {
		fmt.Printf("错误: 解析失败: %v\n", err)
		return
	}

	var buf bytes.Buffer

	switch format {
	case "txt":
		exp := exporter.TXTExporter{}
		_ = exp.Export(&buf)
	case "csv":
		exp := exporter.CSVExporter{Data: parsed}
		_ = exp.Export(&buf)
	case "json":
		exp := exporter.JSONExporter{Data: parsed}
		_ = exp.Export(&buf)
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

// 辅助函数: 计时装饰器
func timeit(name string, f func()) {
	start := time.Now()
	f()
	fmt.Printf("[耗时: %v]\n", time.Since(start))
}
