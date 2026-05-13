package bug

import (
	"strconv"
	"strings"

	"gst/internal/core"
)

const (
	GL_COMPILE_STATUS = 0x8B81
	GL_LINK_STATUS    = 0x8B82
)

type ShaderErrorDetector struct{}

func (d *ShaderErrorDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	var findings []core.Finding

	for frameIdx := range log.Frames {
		frame := &log.Frames[frameIdx]
		findings = append(findings, d.diagnoseFrame(frame)...)
	}

	return findings
}

func (d *ShaderErrorDetector) diagnoseFrame(frame *core.FrameInfo) []core.Finding {
	var findings []core.Finding

	compileCalls := map[int][]int{}
	getShaderivStatus := map[int]bool{}
	linkProgramCalls := map[int][]int{}
	getProgramivStatus := map[int]bool{}
	shaderCompileCount := map[int]int{}

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		switch call.APIName {
		case "glCompileShader":
			for _, id := range extractDecimalIDs(call.RawParams) {
				compileCalls[id] = append(compileCalls[id], call.LineNum)
				shaderCompileCount[id]++
			}
		case "glGetShaderiv":
			id, pname := parseShaderivParams(call.RawParams)
			if id > 0 && pname == GL_COMPILE_STATUS {
				getShaderivStatus[id] = true
			}
		case "glLinkProgram":
			for _, id := range extractDecimalIDs(call.RawParams) {
				linkProgramCalls[id] = append(linkProgramCalls[id], call.LineNum)
			}
		case "glGetProgramiv":
			id, pname := parseShaderivParams(call.RawParams)
			if id > 0 && pname == GL_LINK_STATUS {
				getProgramivStatus[id] = true
			}
		}
	}

	for shaderID, lineNums := range compileCalls {
		if !getShaderivStatus[shaderID] {
			findings = append(findings, core.Finding{
				Severity:       core.SeverityHigh,
				Category:       "shader_error",
				Description:    "glCompileShader 后未检查 COMPILE_STATUS",
				Evidence:       "Shader " + strconv.Itoa(shaderID) + " 在第 " + strconv.Itoa(lineNums[0]) + " 行 glCompileShader 后缺少 glGetShaderiv(COMPILE_STATUS) 检查",
				RootCauseChain: []string{"编译结果未验证，Shader 编译失败时程序继续执行"},
				FixSuggestion:  "在 glCompileShader 后添加 glGetShaderiv(shader, GL_COMPILE_STATUS, &status) 检查编译是否成功",
			})
		}
	}

	for shaderID, count := range shaderCompileCount {
		if count > 1 {
			findings = append(findings, core.Finding{
				Severity:       core.SeverityHigh,
				Category:       "shader_error",
				Description:    "同一帧内重复编译同一个 Shader",
				Evidence:       "Shader " + strconv.Itoa(shaderID) + " 在同一帧内被 glCompileShader 编译了 " + strconv.Itoa(count) + " 次",
				RootCauseChain: []string{"同一帧内重复编译同一 Shader 导致 GPU 性能浪费"},
				FixSuggestion:  "确保每个 Shader 仅在创建或修改时编译一次，避免重复编译",
			})
		}
	}

	for progID, lineNums := range linkProgramCalls {
		if !getProgramivStatus[progID] {
			findings = append(findings, core.Finding{
				Severity:       core.SeverityHigh,
				Category:       "shader_error",
				Description:    "glLinkProgram 后未检查 LINK_STATUS",
				Evidence:       "Program " + strconv.Itoa(progID) + " 在第 " + strconv.Itoa(lineNums[0]) + " 行 glLinkProgram 后缺少 glGetProgramiv(LINK_STATUS) 检查",
				RootCauseChain: []string{"链接结果未验证，Program 链接失败时程序继续执行"},
				FixSuggestion:  "在 glLinkProgram 后添加 glGetProgramiv(program, GL_LINK_STATUS, &status) 检查链接是否成功",
			})
		}
	}

	for i := range frame.APICalls {
		call := &frame.APICalls[i]
		if !call.IsError {
			continue
		}
		if call.APIName == "__glSetError" {
			nearShader := false
			for j := i + 1; j < len(frame.APICalls) && j <= i+5; j++ {
				name := frame.APICalls[j].APIName
				if name == "glCompileShader" || name == "glLinkProgram" || name == "glCreateShader" || name == "glCreateProgram" || name == "glShaderSource" {
					nearShader = true
					break
				}
			}
			for j := i - 1; j >= 0 && j >= i-5; j-- {
				name := frame.APICalls[j].APIName
				if name == "glCompileShader" || name == "glLinkProgram" || name == "glCreateShader" || name == "glCreateProgram" || name == "glShaderSource" {
					nearShader = true
					break
				}
			}
			if nearShader {
				findings = append(findings, core.Finding{
					Severity:       core.SeverityHigh,
					Category:       "shader_error",
					Description:    "Shader 相关调用附近出现 GL Error",
					Evidence:       "第 " + strconv.Itoa(call.LineNum) + " 行附近 Shader 调用检测到 " + call.APIName + " (错误码: " + call.ErrorCode + ")",
					RootCauseChain: []string{"Shader 操作（编译、链接等）引发了 GL Error", "可能原因：Shader 源码语法错误、资源未初始化、状态不正确"},
					FixSuggestion:  "检查 Shader 源码语法、验证 Shader 对象创建是否成功、确认 GL 上下文状态正确",
				})
			}
		}
	}

	return findings
}

func parseShaderivParams(params string) (id int, pname int) {
	parts := strings.Fields(params)
	if len(parts) < 2 {
		return 0, 0
	}
	id = parseHexOrDec(parts[0])
	pname = parseHexOrDec(parts[1])
	return
}
