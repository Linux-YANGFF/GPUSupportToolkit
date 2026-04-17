package main

import (
	"fmt"
	"os"
	"strings"
	
	"gst/internal/core"
	"gst/internal/core/analyzer"
	"gst/internal/core/exporter"
	"gst/internal/core/parser"
	"gst/internal/core/search"
)

func main() {
	fmt.Println("=== GST 功能测试 ===\n")

	// 1. 测试 Parser
	fmt.Println("1. Parser 测试")
	testLog := `glBindBuffer: count=491, time=588 us
glBindFramebuffer: count=29, time=25377 us
glDrawElements: count=493, time=11214 us
libGL: FPS = 8.9
swapBuffers: 3033 us
423 frame cost 109ms`

	p := parser.CreateParser(parser.KindAPITrace)
	parsed, err := p.Parse(strings.NewReader(testLog))
	if err != nil {
		fmt.Printf("   Parser 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   API 调用解析: %d 帧\n", len(parsed.Frames))
	if len(parsed.Frames) > 0 {
		fmt.Printf("   帧 0 包含 %d 个 API 调用\n", len(parsed.Frames[0].APICalls))
		fmt.Printf("   FPS: %.1f\n", parsed.FPS)
	}

	// 2. 测试 Analyzer
	fmt.Println("\n2. Analyzer 测试")
	frameAnalyzer := analyzer.NewFrameAnalyzer(parsed)
	topFrames := frameAnalyzer.FindTopSlowFrames(5)
	fmt.Printf("   找到 %d 个慢帧\n", len(topFrames))
	summary := frameAnalyzer.GetFrameSummary()
	if summary != nil {
		fmt.Printf("   总帧数: %v, 总时间: %v us\n", summary["total_frames"], summary["total_time_us"])
	}

	// 3. 测试 Search
	fmt.Println("\n3. Search 测试")
	lines := []string{
		"glBindBuffer: count=491, time=588 us",
		"glBindFramebuffer: count=29, time=25377 us",
		"glDrawElements: count=493, time=11214 us",
	}
	ks := search.NewKeywordSearch()
	results := ks.Search([]string{"glBind"}, lines)
	fmt.Printf("   'glBind' 搜索结果: %d 条\n", len(results))

	timeSearch := search.NewTimeRangeSearch()
	timeResults := timeSearch.Search(parsed, 1000, 50000)
	fmt.Printf("   时间范围 [1000, 50000] us 搜索结果: %d 条\n", len(timeResults))

	// 4. 测试 Exporter
	fmt.Println("\n4. Exporter 测试")
	
	// 测试 TXT Exporter
	tmpFile, _ := os.CreateTemp("", "gst_test_*.txt")
	defer os.Remove(tmpFile.Name())
	
	txtExp := exporter.TXTExporter{Results: []core.SearchResult{
		{LineNum: 1, Content: "test line 1", PageNum: 0},
		{LineNum: 2, Content: "test line 2", PageNum: 0},
	}}
	err = txtExp.Export(tmpFile)
	if err != nil {
		fmt.Printf("   TXT Export 失败: %v\n", err)
	} else {
		fmt.Println("   TXT Export: 成功")
	}
	tmpFile.Close()

	// 测试 JSON Exporter
	tmpJSON, _ := os.CreateTemp("", "gst_test_*.json")
	defer os.Remove(tmpJSON.Name())
	jsonExp := exporter.JSONExporter{Data: parsed}
	err = jsonExp.Export(tmpJSON)
	if err != nil {
		fmt.Printf("   JSON Export 失败: %v\n", err)
	} else {
		fmt.Println("   JSON Export: 成功")
	}
	tmpJSON.Close()

	// 5. 测试使用示例日志文件
	fmt.Println("\n5. 示例日志文件解析测试")
	profileFile, err := os.Open("/root/code/GPUSupportToolkit/GPUSupportToolkit/exmple_log/1frame_profile_demo.txt")
	if err != nil {
		fmt.Printf("   打开示例文件失败: %v\n", err)
	} else {
		defer profileFile.Close()
		profileParser := parser.CreateParser(parser.KindProfile)
		profileParsed, err := profileParser.Parse(profileFile)
		if err != nil {
			fmt.Printf("   解析示例文件失败: %v\n", err)
		} else {
			fmt.Printf("   解析 profile_demo.txt: %d 帧, FPS=%.1f\n", 
				len(profileParsed.Frames), profileParsed.FPS)
		}
	}

	fmt.Println("\n=== 功能测试完成 ===")
}
