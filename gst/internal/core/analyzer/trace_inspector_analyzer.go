package analyzer

import (
	"sort"
	"strconv"
	"strings"

	"gst/internal/core"
)

const (
	traceSourceTypeSource  = "source"
	traceSourceTypeBinary  = "program_binary"
	traceSourceTypeUnknown = "unknown"

	traceConfidenceHigh   = "high"
	traceConfidenceMedium = "medium"
	traceConfidenceLow    = "low"
)

type TraceInspectorAnalyzer struct {
	log *core.ParsedLog
}

type traceContextState struct {
	Program       int
	VAO           int
	ArrayBuffer   int
	ElementBuffer int
	FBO           int
	ActiveTexture int
	Textures      map[int]int
}

func NewTraceInspectorAnalyzer(log *core.ParsedLog) *TraceInspectorAnalyzer {
	return &TraceInspectorAnalyzer{log: log}
}

func (tia *TraceInspectorAnalyzer) Analyze() *core.TraceAnalysis {
	if tia.log == nil {
		return nil
	}

	result := &core.TraceAnalysis{
		Programs:      []core.ProgramInfo{},
		ProgramMap:    make(map[int]*core.ProgramInfo),
		FrameInsights: make(map[int]*core.FrameProgramInsight),
		DrawCalls:     make(map[int][]core.DrawCallInsight),
	}

	shaders := tia.buildShaderAndProgramRegistry(result)
	tia.finalizePrograms(result, shaders)
	tia.buildFrameInsights(result)
	tia.finalizePrograms(result, shaders)

	return result
}

func (tia *TraceInspectorAnalyzer) buildShaderAndProgramRegistry(result *core.TraceAnalysis) map[int]*core.ShaderObjectInfo {
	shaders := make(map[int]*core.ShaderObjectInfo)
	sourceByID := make(map[int]*core.ShaderInfo)
	for _, frame := range tia.log.Frames {
		for _, shader := range frame.Shaders {
			if shader != nil && shader.ID > 0 {
				sourceByID[shader.ID] = shader
			}
		}
	}

	for _, frame := range tia.log.Frames {
		for _, call := range frame.APICalls {
			switch call.APIName {
			case "glCreateShader":
				shaderID := intFromReturn(call.ReturnValue)
				shaderType := shaderTypeFromParams(call.RawParams)
				if shaderID == 0 {
					nums := numbersFromParams(call.RawParams)
					if len(nums) >= 2 {
						shaderID = nums[1]
					}
				}
				if shaderID > 0 {
					shader := ensureShader(shaders, shaderID)
					shader.Type = shaderType
					shader.CreateLine = call.LineNum
					shader.GCAddr = call.GCAddr
				}
			case "glShaderSource":
				shaderID := firstNumber(call.RawParams)
				if shaderID > 0 {
					shader := ensureShader(shaders, shaderID)
					shader.SourceLine = call.LineNum
					if src := sourceByID[shaderID]; src != nil {
						shader.Source = src.Source
						shader.SourceAvailable = src.Source != ""
					}
				}
			case "glCompileShader":
				shaderID := firstNumber(call.RawParams)
				if shaderID > 0 {
					ensureShader(shaders, shaderID).CompileLine = call.LineNum
				}
			case "glCreateProgram":
				programID := intFromReturn(call.ReturnValue)
				if programID == 0 {
					programID = firstNumber(call.RawParams)
				}
				if programID > 0 {
					program := ensureProgram(result, programID)
					program.CreateLine = call.LineNum
					addProgramLine(program, call.LineNum)
				}
			case "glAttachShader":
				nums := numbersFromParams(call.RawParams)
				if len(nums) >= 2 {
					program := ensureProgram(result, nums[0])
					addUniqueInt(&program.ShaderIDs, nums[1])
					addProgramLine(program, call.LineNum)
				}
			case "glLinkProgram":
				programID := firstNumber(call.RawParams)
				if programID > 0 {
					program := ensureProgram(result, programID)
					program.LinkLines = append(program.LinkLines, call.LineNum)
					addProgramLine(program, call.LineNum)
				}
			case "glProgramBinary":
				fields := strings.Fields(call.RawParams)
				if len(fields) >= 4 {
					programID := parseGLInt(fields[0])
					if programID > 0 {
						program := ensureProgram(result, programID)
						program.SourceType = traceSourceTypeBinary
						program.BinaryLine = call.LineNum
						program.BinaryFormat = fields[1]
						program.BinarySizeBytes = parseGLInt(fields[len(fields)-1])
						addProgramLine(program, call.LineNum)
					}
				}
			}
		}
	}

	return shaders
}

func (tia *TraceInspectorAnalyzer) finalizePrograms(result *core.TraceAnalysis, shaders map[int]*core.ShaderObjectInfo) {
	programs := make([]core.ProgramInfo, 0, len(result.ProgramMap))
	for _, program := range result.ProgramMap {
		sort.Ints(program.ShaderIDs)
		sort.Ints(program.FramesUsed)
		program.Shaders = make([]core.ShaderObjectInfo, 0, len(program.ShaderIDs))
		sourceAvailable := false
		for _, shaderID := range program.ShaderIDs {
			if shader, ok := shaders[shaderID]; ok {
				if shader.SourceAvailable {
					sourceAvailable = true
				}
				program.Shaders = append(program.Shaders, *shader)
			}
		}

		switch {
		case len(program.ShaderIDs) > 0:
			program.SourceType = traceSourceTypeSource
		case program.BinaryLine > 0:
			program.SourceType = traceSourceTypeBinary
		case program.SourceType == "":
			program.SourceType = traceSourceTypeUnknown
		}

		switch {
		case program.SourceType == traceSourceTypeSource && sourceAvailable && len(program.LinkLines) > 0:
			program.Confidence = traceConfidenceHigh
		case program.SourceType == traceSourceTypeSource || program.SourceType == traceSourceTypeBinary:
			program.Confidence = traceConfidenceMedium
		default:
			program.Confidence = traceConfidenceLow
		}
		programs = append(programs, *program)
	}

	sort.Slice(programs, func(i, j int) bool {
		return programs[i].ID < programs[j].ID
	})
	result.Programs = programs
}

func (tia *TraceInspectorAnalyzer) buildFrameInsights(result *core.TraceAnalysis) {
	contexts := make(map[string]*traceContextState)
	frameSeen := make(map[int]map[int]bool)

	for _, frame := range tia.log.Frames {
		usages := make(map[int]*core.ProgramUsage)
		var segments []core.ProgramSegment
		var activeSegment *core.ProgramSegment
		totalDrawCalls := 0

		for _, call := range frame.APICalls {
			ctx := getTraceContext(contexts, call.GCAddr)

			if call.APIName == "glUseProgram" {
				programID := firstNumber(call.RawParams)
				ctx.Program = programID
				program := ensureProgram(result, programID)
				program.UseCount++
				addProgramLine(program, call.LineNum)
				addFrameUsed(program, frame.FrameNum, frameSeen)

				usage := ensureProgramUsage(usages, program)
				usage.UseCount++
				addUsageLine(usage, call.LineNum)

				if activeSegment != nil {
					activeSegment.EndLine = call.LineNum - 1
					segments = append(segments, *activeSegment)
				}
				activeSegment = &core.ProgramSegment{
					ProgramID: programID,
					StartLine: call.LineNum,
					EndLine:   frame.EndLine,
				}
				continue
			}

			updateTraceState(ctx, call)

			if !isDrawCall(call.APIName) {
				continue
			}

			totalDrawCalls++
			programID := ctx.Program
			if programID > 0 {
				program := ensureProgram(result, programID)
				program.DrawCallCount++
				addProgramLine(program, call.LineNum)
				addFrameUsed(program, frame.FrameNum, frameSeen)

				usage := ensureProgramUsage(usages, program)
				usage.DrawCallCount++
				addUsageLine(usage, call.LineNum)
			}

			if activeSegment != nil && activeSegment.ProgramID == programID {
				activeSegment.DrawCallCount++
			}
		}

		if activeSegment != nil {
			activeSegment.EndLine = frame.EndLine
			segments = append(segments, *activeSegment)
		}

		programUsages := make([]core.ProgramUsage, 0, len(usages))
		for _, usage := range usages {
			programUsages = append(programUsages, *usage)
		}
		sort.Slice(programUsages, func(i, j int) bool {
			if programUsages[i].DrawCallCount == programUsages[j].DrawCallCount {
				return programUsages[i].ProgramID < programUsages[j].ProgramID
			}
			return programUsages[i].DrawCallCount > programUsages[j].DrawCallCount
		})

		result.FrameInsights[frame.FrameNum] = &core.FrameProgramInsight{
			FrameNum:       frame.FrameNum,
			StartLine:      frame.StartLine,
			EndLine:        frame.EndLine,
			TotalDrawCalls: totalDrawCalls,
			Programs:       programUsages,
			Segments:       segments,
		}
	}
}

func (tia *TraceInspectorAnalyzer) AnalyzeFrameDrawCalls(frameNum int, programFilter int, trace *core.TraceAnalysis) ([]core.DrawCallInsight, bool) {
	if tia.log == nil {
		return nil, false
	}

	contexts := make(map[string]*traceContextState)
	var drawCalls []core.DrawCallInsight
	found := false

	for _, frame := range tia.log.Frames {
		for _, call := range frame.APICalls {
			ctx := getTraceContext(contexts, call.GCAddr)
			if call.APIName == "glUseProgram" {
				ctx.Program = firstNumber(call.RawParams)
				continue
			}
			updateTraceState(ctx, call)
			if !isDrawCall(call.APIName) {
				continue
			}
			if frame.FrameNum != frameNum {
				continue
			}
			found = true
			programID := ctx.Program
			if programFilter > 0 && programID != programFilter {
				continue
			}
			confidence := traceConfidenceLow
			if trace != nil {
				if program := trace.ProgramMap[programID]; program != nil {
					confidence = program.Confidence
				}
			}
			drawCalls = append(drawCalls, core.DrawCallInsight{
				LineNum:       call.LineNum,
				APIName:       call.APIName,
				DrawType:      classifyDrawCall(call.APIName),
				RawParams:     call.RawParams,
				ProgramID:     programID,
				GCAddr:        call.GCAddr,
				TID:           call.TID,
				VAO:           ctx.VAO,
				ArrayBuffer:   ctx.ArrayBuffer,
				ElementBuffer: ctx.ElementBuffer,
				FBO:           ctx.FBO,
				Textures:      copyTextureBindings(ctx.Textures),
				Confidence:    confidence,
			})
		}
		if frame.FrameNum == frameNum {
			return drawCalls, found
		}
	}

	return drawCalls, found
}

func ensureProgram(result *core.TraceAnalysis, id int) *core.ProgramInfo {
	if id <= 0 {
		return &core.ProgramInfo{ID: 0, SourceType: traceSourceTypeUnknown, Confidence: traceConfidenceLow}
	}
	if program, ok := result.ProgramMap[id]; ok {
		return program
	}
	program := &core.ProgramInfo{
		ID:         id,
		SourceType: traceSourceTypeUnknown,
		Confidence: traceConfidenceLow,
		ShaderIDs:  []int{},
		FramesUsed: []int{},
	}
	result.ProgramMap[id] = program
	return program
}

func ensureShader(shaders map[int]*core.ShaderObjectInfo, id int) *core.ShaderObjectInfo {
	if shader, ok := shaders[id]; ok {
		return shader
	}
	shader := &core.ShaderObjectInfo{ID: id, Type: "unknown"}
	shaders[id] = shader
	return shader
}

func ensureProgramUsage(usages map[int]*core.ProgramUsage, program *core.ProgramInfo) *core.ProgramUsage {
	if usage, ok := usages[program.ID]; ok {
		return usage
	}
	usage := &core.ProgramUsage{
		ProgramID:       program.ID,
		SourceType:      program.SourceType,
		Confidence:      program.Confidence,
		ShaderIDs:       append([]int(nil), program.ShaderIDs...),
		SourceAvailable: programHasSource(program),
		BinarySizeBytes: program.BinarySizeBytes,
	}
	if usage.ShaderIDs == nil {
		usage.ShaderIDs = []int{}
	}
	usages[program.ID] = usage
	return usage
}

func getTraceContext(contexts map[string]*traceContextState, gcAddr string) *traceContextState {
	key := gcAddr
	if key == "" {
		key = "__default__"
	}
	if ctx, ok := contexts[key]; ok {
		return ctx
	}
	ctx := &traceContextState{Textures: make(map[int]int)}
	contexts[key] = ctx
	return ctx
}

func updateTraceState(ctx *traceContextState, call core.APILogEntry) {
	fields := strings.Fields(call.RawParams)
	switch call.APIName {
	case "glBindBuffer":
		if len(fields) >= 2 {
			target := parseGLInt(fields[0])
			buffer := parseGLInt(fields[1])
			switch target {
			case 0x8892:
				ctx.ArrayBuffer = buffer
			case 0x8893:
				ctx.ElementBuffer = buffer
			}
		}
	case "glBindVertexArray":
		if len(fields) >= 1 {
			ctx.VAO = parseGLInt(fields[0])
		}
	case "glBindFramebuffer":
		if len(fields) >= 2 {
			ctx.FBO = parseGLInt(fields[1])
		}
	case "glActiveTexture":
		if len(fields) >= 1 {
			ctx.ActiveTexture = parseGLInt(fields[0])
		}
	case "glBindTexture":
		if len(fields) >= 2 {
			ctx.Textures[ctx.ActiveTexture] = parseGLInt(fields[1])
		}
	}
}

func shaderTypeFromParams(params string) string {
	fields := strings.Fields(params)
	if len(fields) == 0 {
		return "unknown"
	}
	switch parseGLInt(fields[0]) {
	case 0x8B31:
		return "vertex"
	case 0x8B30:
		return "fragment"
	case 0x8DD9:
		return "geometry"
	case 0x8E88:
		return "tess_control"
	case 0x8E87:
		return "tess_eval"
	case 0x91B9:
		return "compute"
	default:
		return "unknown"
	}
}

func intFromReturn(value string) int {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0
	}
	return parseGLInt(fields[0])
}

func firstNumber(params string) int {
	nums := numbersFromParams(params)
	if len(nums) == 0 {
		return 0
	}
	return nums[0]
}

func numbersFromParams(params string) []int {
	fields := strings.Fields(strings.ReplaceAll(params, ",", " "))
	nums := make([]int, 0, len(fields))
	for _, field := range fields {
		if n := parseGLInt(field); n != 0 || field == "0" {
			nums = append(nums, n)
		}
	}
	return nums
}

func parseGLInt(value string) int {
	value = strings.TrimSpace(strings.Trim(value, ","))
	if value == "" || value == "(nil)" || strings.HasPrefix(value, "0xaaa") || strings.HasPrefix(value, "0xffff") || strings.HasPrefix(value, "0xfffe") {
		return 0
	}
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		n, err := strconv.ParseInt(value[2:], 16, 64)
		if err != nil {
			return 0
		}
		return int(n)
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

func addUniqueInt(target *[]int, value int) {
	if value <= 0 {
		return
	}
	for _, existing := range *target {
		if existing == value {
			return
		}
	}
	*target = append(*target, value)
}

func addFrameUsed(program *core.ProgramInfo, frameNum int, seen map[int]map[int]bool) {
	if program.ID <= 0 {
		return
	}
	if _, ok := seen[program.ID]; !ok {
		seen[program.ID] = make(map[int]bool)
	}
	if seen[program.ID][frameNum] {
		return
	}
	seen[program.ID][frameNum] = true
	program.FramesUsed = append(program.FramesUsed, frameNum)
}

func addProgramLine(program *core.ProgramInfo, line int) {
	if program.ID <= 0 || line <= 0 {
		return
	}
	if program.FirstLine == 0 || line < program.FirstLine {
		program.FirstLine = line
	}
	if line > program.LastLine {
		program.LastLine = line
	}
}

func addUsageLine(usage *core.ProgramUsage, line int) {
	if line <= 0 {
		return
	}
	if usage.FirstLine == 0 || line < usage.FirstLine {
		usage.FirstLine = line
	}
	if line > usage.LastLine {
		usage.LastLine = line
	}
}

func programHasSource(program *core.ProgramInfo) bool {
	for _, shader := range program.Shaders {
		if shader.SourceAvailable {
			return true
		}
	}
	return false
}

func copyTextureBindings(src map[int]int) map[int]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[int]int, len(src))
	for unit, tex := range src {
		dst[unit] = tex
	}
	return dst
}
