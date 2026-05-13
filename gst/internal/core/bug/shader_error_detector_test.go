package bug

import (
	"testing"

	"gst/internal/core"
)

func TestShaderErrorDetector_CompileShaderWithoutStatusCheck(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
					{APIName: "glDrawElements", RawParams: "0x0004 2304 0x1403", LineNum: 11},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "glCompileShader 后未检查 COMPILE_STATUS" {
			found = true
			if f.Severity != core.SeverityHigh {
				t.Errorf("expected severity high, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected finding for glCompileShader without COMPILE_STATUS check")
	}
}

func TestShaderErrorDetector_CompileShaderWithStatusCheck(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
					{APIName: "glGetShaderiv", RawParams: "5 0x8B81 0x7ffc8a1b0", LineNum: 11},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "glCompileShader 后未检查 COMPILE_STATUS" {
			t.Error("should NOT flag when COMPILE_STATUS is checked")
		}
	}
}

func TestShaderErrorDetector_LinkProgramWithoutStatusCheck(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glLinkProgram", RawParams: "18", LineNum: 20},
					{APIName: "glUseProgram", RawParams: "18", LineNum: 21},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "glLinkProgram 后未检查 LINK_STATUS" {
			found = true
			if f.Severity != core.SeverityHigh {
				t.Errorf("expected severity high, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected finding for glLinkProgram without LINK_STATUS check")
	}
}

func TestShaderErrorDetector_LinkProgramWithStatusCheck(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glLinkProgram", RawParams: "18", LineNum: 20},
					{APIName: "glGetProgramiv", RawParams: "18 0x8B82 0x7ffc8a1b0", LineNum: 21},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "glLinkProgram 后未检查 LINK_STATUS" {
			t.Error("should NOT flag when LINK_STATUS is checked")
		}
	}
}

func TestShaderErrorDetector_RepeatedShaderCompilation(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
					{APIName: "glCompileShader", RawParams: "5", LineNum: 15},
					{APIName: "glGetShaderiv", RawParams: "5 0x8B81 0x7ffc8a1b0", LineNum: 16},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "同一帧内重复编译同一个 Shader" {
			found = true
			if f.Severity != core.SeverityHigh {
				t.Errorf("expected severity high, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected finding for repeated shader compilation in same frame")
	}
}

func TestShaderErrorDetector_SingleCompileNoRepeatFlag(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "同一帧内重复编译同一个 Shader" {
			t.Error("should NOT flag single compilation as repeated")
		}
	}
}

func TestShaderErrorDetector_GlSetErrorNearShaderCall(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCreateShader", RawParams: "0x8b31 5", LineNum: 5},
					{APIName: "glShaderSource", RawParams: "5", LineNum: 6},
					{APIName: "glCompileShader", RawParams: "5", LineNum: 7},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0500", LineNum: 8},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "Shader 相关调用附近出现 GL Error" {
			found = true
			if f.Severity != core.SeverityHigh {
				t.Errorf("expected severity high, got %s", f.Severity)
			}
		}
	}
	if !found {
		t.Error("expected finding for __glSetError near shader calls")
	}
}

func TestShaderErrorDetector_GlSetErrorNotNearShader(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", RawParams: "0x8892 199", LineNum: 1},
					{APIName: "glDrawElements", RawParams: "0x0004 2304 0x1403", LineNum: 2},
					{APIName: "__glSetError", IsError: true, ErrorCode: "0x0501", LineNum: 3},
					{APIName: "glBindTexture", RawParams: "0x0de1 42", LineNum: 4},
					{APIName: "glUseProgram", RawParams: "18", LineNum: 5},
					{APIName: "glDrawArrays", RawParams: "0x0005 0 4", LineNum: 6},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "Shader 相关调用附近出现 GL Error" {
			t.Error("should NOT flag __glSetError when not near shader calls")
		}
	}
}

func TestShaderErrorDetector_EmptyLog(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty log, got %d", len(findings))
	}
}

func TestShaderErrorDetector_MultipleDetections(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
					{APIName: "glCompileShader", RawParams: "7", LineNum: 11},
					{APIName: "glLinkProgram", RawParams: "18", LineNum: 12},
					{APIName: "glCompileShader", RawParams: "5", LineNum: 13},
					{APIName: "glCompileShader", RawParams: "9", LineNum: 14},
					{APIName: "glLinkProgram", RawParams: "22", LineNum: 15},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	compileStatusMiss := 0
	linkStatusMiss := 0
	repeatedCompile := 0

	for _, f := range findings {
		switch f.Description {
		case "glCompileShader 后未检查 COMPILE_STATUS":
			compileStatusMiss++
		case "glLinkProgram 后未检查 LINK_STATUS":
			linkStatusMiss++
		case "同一帧内重复编译同一个 Shader":
			repeatedCompile++
		}
	}

	if compileStatusMiss != 3 {
		t.Errorf("expected 3 missing COMPILE_STATUS, got %d", compileStatusMiss)
	}
	if linkStatusMiss != 2 {
		t.Errorf("expected 2 missing LINK_STATUS, got %d", linkStatusMiss)
	}
	if repeatedCompile != 1 {
		t.Errorf("expected 1 repeated compile, got %d", repeatedCompile)
	}
}

func TestParseShaderivParams(t *testing.T) {
	id, pname := parseShaderivParams("5 0x8B81 0x7ffc8a1b0")
	if id != 5 {
		t.Errorf("expected id 5, got %d", id)
	}
	if pname != 0x8B81 {
		t.Errorf("expected pname 0x8B81, got 0x%X", pname)
	}

	id, pname = parseShaderivParams("18 0x8B82 0x7ffc8a1b0")
	if id != 18 {
		t.Errorf("expected id 18, got %d", id)
	}
	if pname != 0x8B82 {
		t.Errorf("expected pname 0x8B82, got 0x%X", pname)
	}

	id, pname = parseShaderivParams("")
	if id != 0 || pname != 0 {
		t.Errorf("expected (0, 0) for empty params, got (%d, %d)", id, pname)
	}

	id, pname = parseShaderivParams("42")
	if id != 0 || pname != 0 {
		t.Errorf("expected (0, 0) for single param, got (%d, %d)", id, pname)
	}
}

func TestShaderErrorDetector_CrossFrameIsolation(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
				},
			},
			{
				FrameNum: 2,
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "7", LineNum: 20},
					{APIName: "glGetShaderiv", RawParams: "7 0x8B81 0x7ffc8a1b0", LineNum: 21},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	compileStatusMiss := 0
	for _, f := range findings {
		if f.Description == "glCompileShader 后未检查 COMPILE_STATUS" {
			compileStatusMiss++
		}
	}
	if compileStatusMiss != 1 {
		t.Errorf("expected 1 missing COMPILE_STATUS across frames, got %d", compileStatusMiss)
	}
}

func TestShaderErrorDetector_GetShaderivNonCompileStatus(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCompileShader", RawParams: "5", LineNum: 10},
					{APIName: "glGetShaderiv", RawParams: "5 0x8B82 0x7ffc8a1b0", LineNum: 11},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "glCompileShader 后未检查 COMPILE_STATUS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("should flag when GetShaderiv queries non-COMPILE_STATUS pname")
	}
}

func TestShaderErrorDetector_GetProgramivNonLinkStatus(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glLinkProgram", RawParams: "18", LineNum: 20},
					{APIName: "glGetProgramiv", RawParams: "18 0x8B81 0x7ffc8a1b0", LineNum: 21},
				},
			},
		},
	}

	findings := detector.Diagnose(log)

	found := false
	for _, f := range findings {
		if f.Category == "shader_error" && f.Description == "glLinkProgram 后未检查 LINK_STATUS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("should flag when GetProgramiv queries non-LINK_STATUS pname")
	}
}

func TestShaderErrorDetector_MultipleShadersOneFrame(t *testing.T) {
	detector := &ShaderErrorDetector{}

	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				APICalls: []core.APILogEntry{
					{APIName: "glCreateShader", RawParams: "0x8b31 1", LineNum: 1},
					{APIName: "glShaderSource", RawParams: "1", LineNum: 2},
					{APIName: "glCompileShader", RawParams: "1", LineNum: 3},
					{APIName: "glGetShaderiv", RawParams: "1 0x8B81 0x7ffc8a1b0", LineNum: 4},
					{APIName: "glCreateShader", RawParams: "0x8b30 2", LineNum: 5},
					{APIName: "glShaderSource", RawParams: "2", LineNum: 6},
					{APIName: "glCompileShader", RawParams: "2", LineNum: 7},
					{APIName: "glGetShaderiv", RawParams: "2 0x8B81 0x7ffc8a1b0", LineNum: 8},
					{APIName: "glCreateProgram", RawParams: "", LineNum: 9},
					{APIName: "glLinkProgram", RawParams: "18", LineNum: 10},
					{APIName: "glGetProgramiv", RawParams: "18 0x8B82 0x7ffc8a1b0", LineNum: 11},
				},
			},
		},
	}

	findings := detector.Diagnose(log)
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for well-checked shaders, got %d", len(findings))
	}
}
