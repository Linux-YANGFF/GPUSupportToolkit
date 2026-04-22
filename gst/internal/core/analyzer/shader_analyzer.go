package analyzer

import (
	"regexp"
	"sort"
	"gst/internal/core"
)

var (
	shaderSourceRegex  = regexp.MustCompile(`glShaderSource`)
	compileShaderRegex = regexp.MustCompile(`glCompileShader`)
	createShaderRegex  = regexp.MustCompile(`glCreateShader`)
)

// ShaderAnalyzer Shader分析器
type ShaderAnalyzer struct {
	log     *core.ParsedLog
	shaders map[string]*core.ShaderCompileInfo
}

// NewShaderAnalyzer 创建Shader分析器
func NewShaderAnalyzer(log *core.ParsedLog) *ShaderAnalyzer {
	return &ShaderAnalyzer{log: log, shaders: make(map[string]*core.ShaderCompileInfo)}
}

// Analyze 分析Shader使用情况
func (sa *ShaderAnalyzer) Analyze() []core.ShaderCompileInfo {
	if sa.log == nil {
		return nil
	}

	sa.shaders = make(map[string]*core.ShaderCompileInfo)

	// 遍历所有帧
	for _, frame := range sa.log.Frames {
		for _, call := range frame.APICalls {
			switch {
			case createShaderRegex.MatchString(call.APIName):
				sa.incShaderStat("Create", call.TimeUs)
			case compileShaderRegex.MatchString(call.APIName):
				sa.incShaderStat("Compile", call.TimeUs)
			case shaderSourceRegex.MatchString(call.APIName):
				sa.incShaderStat("Source", call.TimeUs)
			}
		}
	}

	results := make([]core.ShaderCompileInfo, 0, len(sa.shaders))
	for _, info := range sa.shaders {
		results = append(results, *info)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalCompileTimeUs > results[j].TotalCompileTimeUs
	})

	return results
}

func (sa *ShaderAnalyzer) incShaderStat(shaderType string, timeUs int64) {
	key := shaderType
	if info, ok := sa.shaders[key]; ok {
		info.CompileCount++
		info.TotalCompileTimeUs += timeUs
	} else {
		sa.shaders[key] = &core.ShaderCompileInfo{
			Type:               shaderType,
			CompileCount:       1,
			TotalCompileTimeUs: timeUs,
		}
	}
}

// GetShaderSummary 获取Shader统计摘要
func (sa *ShaderAnalyzer) GetShaderSummary() map[string]interface{} {
	infos := sa.Analyze()
	if infos == nil {
		return nil
	}

	var totalCount int
	var totalTime int64

	for _, info := range infos {
		totalCount += info.CompileCount
		totalTime += info.TotalCompileTimeUs
	}

	return map[string]interface{}{
		"shader_types":  len(infos),
		"total_compile": totalCount,
		"total_time_us": totalTime,
	}
}
