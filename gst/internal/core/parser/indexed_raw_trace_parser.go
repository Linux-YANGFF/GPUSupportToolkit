package parser

import (
	"bufio"
	"errors"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"gst/internal/core"
)

type indexedFrameDraft struct {
	frame       core.FrameInfo
	rawSummary  map[string]*core.APISummary
	aggSummary  map[string]*core.APISummary
	usages      map[int]*core.ProgramUsage
	segments    []core.ProgramSegment
	activeSeg   *core.ProgramSegment
	contextAtIn map[string]int
}

// ParseIndexedRawTraceFile parses large raw/hybrid apiTrace logs without keeping
// every API call in memory. It uses frame cost lines as the source of truth when
// present, so incomplete startup/tail segments are not published as frames.
func ParseIndexedRawTraceFile(path string) (*core.ParsedLog, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	log := &core.ParsedLog{
		Indexed:    true,
		SourcePath: path,
		Trace: &core.TraceAnalysis{
			Programs:      []core.ProgramInfo{},
			ProgramMap:    make(map[int]*core.ProgramInfo),
			FrameInsights: make(map[int]*core.FrameProgramInsight),
			DrawCalls:     make(map[int][]core.DrawCallInsight),
		},
	}

	reader := bufio.NewReaderSize(file, 1024*1024)
	contextPrograms := make(map[string]int)
	var candidate *indexedFrameDraft
	var lastSwapped *indexedFrameDraft
	lastCommittedIdx := -1
	aggregatedByFrameIdx := make(map[int]bool)
	frameCostSeen := false
	offset := int64(0)
	lineNum := 0

	for {
		rawLine, readErr := reader.ReadString('\n')
		if len(rawLine) > 0 {
			lineNum++
			lineOffset := offset
			offset += int64(len(rawLine))
			line := strings.TrimRight(rawLine, "\r\n")
			normalized := normalizeRawLine(line)

			if matches := fpsRegex.FindStringSubmatch(normalized); len(matches) > 1 {
				log.FPS = parseFloat64(matches[1])
				goto nextLine
			}

			if matches := swapBuffersRegex.FindStringSubmatch(normalized); len(matches) > 2 {
				if lastSwapped != nil {
					lastSwapped.frame.SwapBufferTimeUs = parseInt64(matches[2])
					touchIndexedFullRange(&lastSwapped.frame, lineNum, offset)
				}
				goto nextLine
			}

			if matches := frameCostRegex.FindStringSubmatch(normalized); len(matches) > 3 {
				frameCostSeen = true
				frameID := parseInt(matches[2])
				totalTimeUs := parseInt64(matches[3]) * 1000
				if lastSwapped != nil {
					touchIndexedFullRange(&lastSwapped.frame, lineNum, offset)
					commitIndexedFrame(log, lastSwapped, frameID, totalTimeUs)
					lastCommittedIdx = len(log.Frames) - 1
					lastSwapped = nil
				}
				goto nextLine
			}

			if matches := apiLineRegex.FindStringSubmatch(normalized); len(matches) > 4 {
				if lastCommittedIdx >= 0 {
					frame := &log.Frames[lastCommittedIdx]
					if !aggregatedByFrameIdx[lastCommittedIdx] {
						frame.APISummary = make(map[string]*core.APISummary)
						frame.APITotalTimeUs = 0
						aggregatedByFrameIdx[lastCommittedIdx] = true
					}
					touchIndexedFullRange(frame, lineNum, offset)
					addAPISummary(frame.APISummary, matches[2], parseInt(matches[3]), parseInt64(matches[4]))
					frame.APITotalTimeUs += parseInt64(matches[4])
				}
				goto nextLine
			}

			if len(strings.TrimSpace(line)) == 0 ||
				chromeLogRegex.MatchString(line) ||
				warningRegex.MatchString(line) ||
				strings.HasPrefix(normalized, "=>") ||
				strings.HasPrefix(normalized, "src:") ||
				strings.HasPrefix(normalized, "dst:") ||
				strings.HasPrefix(normalized, "{") ||
				strings.HasPrefix(normalized, "}") ||
				strings.HasPrefix(normalized, "[__dri3") ||
				strings.HasPrefix(normalized, "<<gc") ||
				strings.HasPrefix(line, "__") {
				goto nextLine
			}

			if strings.Contains(line, "__glSetError") {
				ensureIndexedCandidate(&candidate, lineNum, lineOffset, contextPrograms)
				candidate.frame.APICallCount++
				addAPISummary(candidate.rawSummary, "__glSetError", 1, 0)
				goto nextLine
			}

			if segfaultRegex.MatchString(line) {
				ensureIndexedCandidate(&candidate, lineNum, lineOffset, contextPrograms)
				candidate.frame.APICallCount++
				addAPISummary(candidate.rawSummary, "__segfault__", 1, 0)
				goto nextLine
			}

			gcAddr, _, workLine := extractGCTID(line)
			apiName, params := parseAPICall(workLine)
			if apiName == "" {
				goto nextLine
			}

			if isRawFrameBoundary(apiName) {
				if candidate != nil {
					candidate.frame.EndLine = lineNum
					candidate.frame.EndOffset = offset
					touchIndexedFullRange(&candidate.frame, lineNum, offset)
					lastSwapped = candidate
					if !frameCostSeen {
						// Kept only as a fallback for logs without frame cost lines.
						commitIndexedFrame(log, candidate, len(log.Frames), candidate.frame.TotalTimeUs)
						lastCommittedIdx = len(log.Frames) - 1
					}
					candidate = nil
				}
				goto nextLine
			}

			ensureIndexedCandidate(&candidate, lineNum, lineOffset, contextPrograms)
			candidate.frame.EndLine = lineNum
			candidate.frame.EndOffset = offset
			touchIndexedFullRange(&candidate.frame, lineNum, offset)
			candidate.frame.APICallCount++
			addAPISummary(candidate.rawSummary, apiName, 1, 0)
			updateIndexedTrace(log.Trace, candidate, contextPrograms, gcAddr, apiName, params, lineNum)
		}

	nextLine:
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}

	if frameCostSeen {
		filterIndexedCostFrames(log)
	}
	finalizeIndexedLog(log)
	return log, nil
}

func ParseIndexedFrameAPICalls(path string, frame core.FrameInfo, page int, pageSize int) ([]core.APILogEntry, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	if _, err := file.Seek(frame.StartOffset, io.SeekStart); err != nil {
		return nil, 0, err
	}

	reader := bufio.NewReaderSize(file, 1024*1024)
	offset := frame.StartOffset
	lineNum := frame.StartLine - 1
	total := 0
	items := []core.APILogEntry{}
	start := (page - 1) * pageSize
	end := start + pageSize

	for offset < frame.EndOffset {
		rawLine, readErr := reader.ReadString('\n')
		if len(rawLine) > 0 {
			lineNum++
			lineOffset := offset
			offset += int64(len(rawLine))
			line := strings.TrimRight(rawLine, "\r\n")
			if offset > frame.EndOffset && lineOffset >= frame.EndOffset {
				break
			}
			if entry, ok := indexedAPICallFromLine(line, lineNum); ok {
				total++
				if total > start && total <= end {
					items = append(items, entry)
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, total, readErr
		}
	}
	return items, total, nil
}

func ParseIndexedFrameRawLines(path string, frame core.FrameInfo, page int, pageSize int, stripLineNumber bool) ([]string, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	startOffset, endOffset := indexedFrameLogRange(frame)
	if endOffset <= startOffset {
		return []string{}, 0, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	if _, err := file.Seek(startOffset, io.SeekStart); err != nil {
		return nil, 0, err
	}

	reader := bufio.NewReaderSize(file, 1024*1024)
	offset := startOffset
	total := 0
	items := []string{}
	start := (page - 1) * pageSize
	end := start + pageSize

	for offset < endOffset {
		rawLine, readErr := reader.ReadString('\n')
		if len(rawLine) > 0 {
			lineOffset := offset
			offset += int64(len(rawLine))
			if lineOffset >= endOffset {
				break
			}
			if offset > endOffset {
				rawLine = rawLine[:endOffset-lineOffset]
			}
			line := strings.TrimRight(rawLine, "\r\n")
			if stripLineNumber {
				line = StripRawTraceLineNumber(line)
			}
			total++
			if total > start && total <= end {
				items = append(items, line)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, total, readErr
		}
	}
	return items, total, nil
}

func CopyIndexedFrameRawLog(w io.Writer, path string, frame core.FrameInfo) error {
	startOffset, endOffset := indexedFrameLogRange(frame)
	if endOffset <= startOffset {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(startOffset, io.SeekStart); err != nil {
		return err
	}
	_, err = io.CopyN(w, file, endOffset-startOffset)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func ParseIndexedFrameDrawCalls(path string, frame core.FrameInfo, programFilter int, page int, pageSize int, trace *core.TraceAnalysis) ([]core.DrawCallInsight, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()
	if _, err := file.Seek(frame.StartOffset, io.SeekStart); err != nil {
		return nil, 0, err
	}

	contextPrograms := copyStringIntMap(frame.ContextPrograms)
	reader := bufio.NewReaderSize(file, 1024*1024)
	offset := frame.StartOffset
	lineNum := frame.StartLine - 1
	total := 0
	frameDrawIndex := 0
	items := []core.DrawCallInsight{}
	start := (page - 1) * pageSize
	end := start + pageSize

	for offset < frame.EndOffset {
		rawLine, readErr := reader.ReadString('\n')
		if len(rawLine) > 0 {
			lineNum++
			lineOffset := offset
			offset += int64(len(rawLine))
			line := strings.TrimRight(rawLine, "\r\n")
			if offset > frame.EndOffset && lineOffset >= frame.EndOffset {
				break
			}
			gcAddr, tid, workLine := extractGCTID(line)
			apiName, params := parseAPICall(workLine)
			if apiName == "" || isRawFrameBoundary(apiName) {
				goto drawNext
			}
			ctxKey := indexedContextKey(gcAddr)
			if apiName == "glUseProgram" {
				contextPrograms[ctxKey] = firstIntParam(params)
				goto drawNext
			}
			if !indexedIsDrawCall(apiName) {
				goto drawNext
			}
			frameDrawIndex++
			programID := contextPrograms[ctxKey]
			if programFilter > 0 && programID != programFilter {
				goto drawNext
			}
			total++
			if total > start && total <= end {
				confidence := "low"
				if trace != nil {
					if program := trace.ProgramMap[programID]; program != nil {
						confidence = program.Confidence
					}
				}
				items = append(items, core.DrawCallInsight{
					Index:      frameDrawIndex,
					LineNum:    lineNum,
					APIName:    apiName,
					DrawType:   indexedDrawType(apiName),
					RawParams:  params,
					ProgramID:  programID,
					GCAddr:     gcAddr,
					TID:        tid,
					Confidence: confidence,
				})
			}
		}
	drawNext:
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, total, readErr
		}
	}
	return items, total, nil
}

func ensureIndexedCandidate(candidate **indexedFrameDraft, lineNum int, offset int64, contextPrograms map[string]int) {
	if *candidate != nil {
		return
	}
	ctx := copyStringIntMap(contextPrograms)
	*candidate = &indexedFrameDraft{
		frame: core.FrameInfo{
			StartLine:       lineNum,
			EndLine:         lineNum,
			StartOffset:     offset,
			EndOffset:       offset,
			FullStartLine:   lineNum,
			FullEndLine:     lineNum,
			FullStartOffset: offset,
			FullEndOffset:   offset,
			HasTiming:       false,
			TimingSource:    "none",
			APISummary:      make(map[string]*core.APISummary),
			Programs:        []int{},
			ContextPrograms: ctx,
		},
		rawSummary:  make(map[string]*core.APISummary),
		aggSummary:  make(map[string]*core.APISummary),
		usages:      make(map[int]*core.ProgramUsage),
		contextAtIn: ctx,
	}
}

func commitIndexedFrame(log *core.ParsedLog, draft *indexedFrameDraft, frameID int, totalTimeUs int64) {
	if draft == nil || draft.frame.APICallCount == 0 {
		return
	}
	frame := draft.frame
	frame.FrameNum = frameID
	frame.TotalTimeUs = totalTimeUs
	frame.HasTiming = totalTimeUs > 0
	if frame.HasTiming {
		frame.TimingSource = "frame_cost"
	}
	if len(draft.aggSummary) > 0 {
		frame.APISummary = draft.aggSummary
		for _, summary := range frame.APISummary {
			frame.APITotalTimeUs += summary.TimeUs
		}
	} else {
		frame.APISummary = draft.rawSummary
		if frame.TotalTimeUs > frame.SwapBufferTimeUs {
			frame.APITotalTimeUs = frame.TotalTimeUs - frame.SwapBufferTimeUs
		}
	}
	if frame.ContextPrograms == nil {
		frame.ContextPrograms = draft.contextAtIn
	}
	if frame.FullStartOffset == 0 && frame.StartOffset > 0 {
		frame.FullStartLine = frame.StartLine
		frame.FullStartOffset = frame.StartOffset
	}
	if frame.FullEndOffset == 0 && frame.EndOffset > 0 {
		frame.FullEndLine = frame.EndLine
		frame.FullEndOffset = frame.EndOffset
	}
	log.Frames = append(log.Frames, frame)
	log.Trace.FrameInsights[frame.FrameNum] = buildIndexedFrameInsight(frame, draft)
}

func buildIndexedFrameInsight(frame core.FrameInfo, draft *indexedFrameDraft) *core.FrameProgramInsight {
	usages := make([]core.ProgramUsage, 0, len(draft.usages))
	for _, usage := range draft.usages {
		usages = append(usages, *usage)
	}
	sort.Slice(usages, func(i, j int) bool {
		if usages[i].DrawCallCount == usages[j].DrawCallCount {
			return usages[i].ProgramID < usages[j].ProgramID
		}
		return usages[i].DrawCallCount > usages[j].DrawCallCount
	})
	segments := append([]core.ProgramSegment(nil), draft.segments...)
	if draft.activeSeg != nil {
		seg := *draft.activeSeg
		seg.EndLine = frame.EndLine
		segments = append(segments, seg)
	}
	return &core.FrameProgramInsight{
		FrameNum:       frame.FrameNum,
		StartLine:      frame.StartLine,
		EndLine:        frame.EndLine,
		HasTiming:      frame.HasTiming,
		TotalTimeUs:    frame.TotalTimeUs,
		APICallCount:   frame.APICallCount,
		TotalDrawCalls: frame.DrawCallCount,
		Programs:       usages,
		Segments:       segments,
	}
}

func updateIndexedTrace(trace *core.TraceAnalysis, draft *indexedFrameDraft, contextPrograms map[string]int, gcAddr string, apiName string, params string, lineNum int) {
	ctxKey := indexedContextKey(gcAddr)
	switch apiName {
	case "glUseProgram":
		programID := firstIntParam(params)
		contextPrograms[ctxKey] = programID
		addUniqueProgramID(&draft.frame.Programs, programID)
		program := indexedEnsureProgram(trace, programID)
		program.UseCount++
		indexedAddProgramLine(program, lineNum)
		indexedAddFrameUsed(program, draft.frame.FrameNum)
		usage := indexedEnsureUsage(draft, programID)
		usage.UseCount++
		indexedAddUsageLine(usage, lineNum)
		if draft.activeSeg != nil {
			draft.activeSeg.EndLine = lineNum - 1
			draft.segments = append(draft.segments, *draft.activeSeg)
		}
		draft.activeSeg = &core.ProgramSegment{ProgramID: programID, StartLine: lineNum, EndLine: draft.frame.EndLine}
	case "glProgramBinary":
		fields := strings.Fields(params)
		if len(fields) >= 4 {
			programID := parseNumericParam(fields[0])
			if programID > 0 {
				program := indexedEnsureProgram(trace, programID)
				program.SourceType = "program_binary"
				program.Confidence = "medium"
				program.BinaryLine = lineNum
				program.BinaryFormat = fields[1]
				program.BinarySizeBytes = parseNumericParam(fields[len(fields)-1])
				indexedAddProgramLine(program, lineNum)
			}
		}
	case "glAttachShader":
		nums := indexedNumbers(params)
		if len(nums) >= 2 {
			program := indexedEnsureProgram(trace, nums[0])
			addUniqueProgramID(&program.ShaderIDs, nums[1])
			program.SourceType = "source"
			program.Confidence = "medium"
			indexedAddProgramLine(program, lineNum)
		}
	case "glLinkProgram":
		programID := firstIntParam(params)
		if programID > 0 {
			program := indexedEnsureProgram(trace, programID)
			program.LinkLines = append(program.LinkLines, lineNum)
			indexedAddProgramLine(program, lineNum)
		}
	}
	if !indexedIsDrawCall(apiName) {
		return
	}
	draft.frame.DrawCallCount++
	programID := contextPrograms[ctxKey]
	if programID <= 0 {
		return
	}
	program := indexedEnsureProgram(trace, programID)
	program.DrawCallCount++
	indexedAddProgramLine(program, lineNum)
	indexedAddFrameUsed(program, draft.frame.FrameNum)
	usage := indexedEnsureUsage(draft, programID)
	usage.DrawCallCount++
	indexedAddUsageLine(usage, lineNum)
	if draft.activeSeg != nil && draft.activeSeg.ProgramID == programID {
		draft.activeSeg.DrawCallCount++
	}
}

func indexedAPICallFromLine(line string, lineNum int) (core.APILogEntry, bool) {
	if strings.Contains(line, "__glSetError") {
		return core.APILogEntry{APIName: "__glSetError", Count: 1, LineNum: lineNum, IsError: true, RawParams: line}, true
	}
	gcAddr, tid, workLine := extractGCTID(line)
	apiName, params := parseAPICall(workLine)
	if apiName == "" || isRawFrameBoundary(apiName) {
		return core.APILogEntry{}, false
	}
	return core.APILogEntry{
		APIName:   apiName,
		Count:     1,
		LineNum:   lineNum,
		RawParams: params,
		GCAddr:    gcAddr,
		TID:       tid,
		HasNilPtr: nilPtrRegex.MatchString(params),
	}, true
}

func finalizeIndexedLog(log *core.ParsedLog) {
	log.TotalTimeUs = 0
	for i := range log.Frames {
		log.TotalTimeUs += log.Frames[i].TotalTimeUs
		if log.Frames[i].APICallCount == 0 {
			log.Frames[i].APICallCount = len(log.Frames[i].APICalls)
		}
	}
	if log.FPS == 0 && len(log.Frames) > 0 && log.TotalTimeUs > 0 {
		log.FPS = float64(len(log.Frames)) * 1e6 / float64(log.TotalTimeUs)
	}
	finalizeIndexedPrograms(log.Trace)
	syncIndexedFrameUsageMetadata(log.Trace)
}

func filterIndexedCostFrames(log *core.ParsedLog) {
	frames := log.Frames[:0]
	kept := make(map[int]bool)
	for _, frame := range log.Frames {
		if frame.HasTiming && frame.TimingSource == "frame_cost" {
			frames = append(frames, frame)
			kept[frame.FrameNum] = true
		}
	}
	log.Frames = frames
	for frameNum := range log.Trace.FrameInsights {
		if !kept[frameNum] {
			delete(log.Trace.FrameInsights, frameNum)
		}
	}
}

func finalizeIndexedPrograms(trace *core.TraceAnalysis) {
	if trace == nil {
		return
	}
	reconcileIndexedProgramStats(trace)
	programs := make([]core.ProgramInfo, 0, len(trace.ProgramMap))
	for _, program := range trace.ProgramMap {
		sort.Ints(program.ShaderIDs)
		sort.Ints(program.FramesUsed)
		if program.SourceType == "" {
			program.SourceType = "unknown"
		}
		if program.Confidence == "" {
			if program.SourceType == "unknown" {
				program.Confidence = "low"
			} else {
				program.Confidence = "medium"
			}
		}
		programs = append(programs, *program)
	}
	sort.Slice(programs, func(i, j int) bool {
		return programs[i].ID < programs[j].ID
	})
	trace.Programs = programs
}

func reconcileIndexedProgramStats(trace *core.TraceAnalysis) {
	for _, program := range trace.ProgramMap {
		program.DrawCallCount = 0
		program.UseCount = 0
		program.FramesUsed = program.FramesUsed[:0]
	}
	for _, insight := range trace.FrameInsights {
		for _, usage := range insight.Programs {
			program := trace.ProgramMap[usage.ProgramID]
			if program == nil {
				continue
			}
			program.DrawCallCount += usage.DrawCallCount
			program.UseCount += usage.UseCount
			if usage.DrawCallCount > 0 || usage.UseCount > 0 {
				indexedAddFrameUsed(program, insight.FrameNum)
			}
		}
	}
}

func syncIndexedFrameUsageMetadata(trace *core.TraceAnalysis) {
	if trace == nil {
		return
	}
	for _, insight := range trace.FrameInsights {
		for i := range insight.Programs {
			program := trace.ProgramMap[insight.Programs[i].ProgramID]
			if program == nil {
				continue
			}
			insight.Programs[i].SourceType = program.SourceType
			insight.Programs[i].Confidence = program.Confidence
			if program.ShaderIDs == nil {
				insight.Programs[i].ShaderIDs = []int{}
			} else {
				insight.Programs[i].ShaderIDs = append([]int(nil), program.ShaderIDs...)
			}
			insight.Programs[i].BinarySizeBytes = program.BinarySizeBytes
		}
	}
}

func indexedEnsureProgram(trace *core.TraceAnalysis, id int) *core.ProgramInfo {
	if id <= 0 {
		return &core.ProgramInfo{ID: 0, SourceType: "unknown", Confidence: "low"}
	}
	if program := trace.ProgramMap[id]; program != nil {
		return program
	}
	program := &core.ProgramInfo{
		ID:         id,
		SourceType: "unknown",
		Confidence: "low",
		ShaderIDs:  []int{},
		FramesUsed: []int{},
	}
	trace.ProgramMap[id] = program
	return program
}

func indexedEnsureUsage(draft *indexedFrameDraft, programID int) *core.ProgramUsage {
	if usage := draft.usages[programID]; usage != nil {
		return usage
	}
	usage := &core.ProgramUsage{
		ProgramID:  programID,
		SourceType: "unknown",
		Confidence: "low",
		ShaderIDs:  []int{},
	}
	draft.usages[programID] = usage
	return usage
}

func addAPISummary(summary map[string]*core.APISummary, name string, count int, timeUs int64) {
	if name == "" || count <= 0 {
		return
	}
	item := summary[name]
	if item == nil {
		summary[name] = &core.APISummary{APIName: name, Count: count, TimeUs: timeUs}
		return
	}
	item.Count += count
	item.TimeUs += timeUs
}

func StripRawTraceLineNumber(line string) string {
	return strings.TrimSpace(removeRawTraceLinePrefix(line))
}

func touchIndexedFullRange(frame *core.FrameInfo, lineNum int, endOffset int64) {
	if frame == nil {
		return
	}
	if frame.FullStartLine == 0 {
		frame.FullStartLine = frame.StartLine
	}
	if frame.FullStartOffset == 0 {
		frame.FullStartOffset = frame.StartOffset
	}
	frame.FullEndLine = lineNum
	frame.FullEndOffset = endOffset
}

func indexedFrameLogRange(frame core.FrameInfo) (int64, int64) {
	start := frame.FullStartOffset
	end := frame.FullEndOffset
	if start == 0 {
		start = frame.StartOffset
	}
	if end == 0 {
		end = frame.EndOffset
	}
	return start, end
}

func indexedAddProgramLine(program *core.ProgramInfo, line int) {
	if program == nil || program.ID <= 0 || line <= 0 {
		return
	}
	if program.FirstLine == 0 || line < program.FirstLine {
		program.FirstLine = line
	}
	if line > program.LastLine {
		program.LastLine = line
	}
}

func indexedAddUsageLine(usage *core.ProgramUsage, line int) {
	if usage == nil || line <= 0 {
		return
	}
	if usage.FirstLine == 0 || line < usage.FirstLine {
		usage.FirstLine = line
	}
	if line > usage.LastLine {
		usage.LastLine = line
	}
}

func indexedAddFrameUsed(program *core.ProgramInfo, frameNum int) {
	if program == nil || program.ID <= 0 || frameNum <= 0 {
		return
	}
	for _, existing := range program.FramesUsed {
		if existing == frameNum {
			return
		}
	}
	program.FramesUsed = append(program.FramesUsed, frameNum)
}

func addUniqueProgramID(ids *[]int, id int) {
	if id <= 0 {
		return
	}
	for _, existing := range *ids {
		if existing == id {
			return
		}
	}
	*ids = append(*ids, id)
}

func indexedContextKey(gcAddr string) string {
	if gcAddr == "" {
		return "__default__"
	}
	return gcAddr
}

func copyStringIntMap(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func indexedIsDrawCall(apiName string) bool {
	return indexedDrawType(apiName) != ""
}

func indexedDrawType(apiName string) string {
	switch {
	case strings.Contains(apiName, "DrawArraysInstanced") || strings.Contains(apiName, "DrawElementsInstanced"):
		return "instanced"
	case strings.Contains(apiName, "DrawArraysIndirect") || strings.Contains(apiName, "DrawElementsIndirect"):
		return "indirect"
	case strings.Contains(apiName, "DispatchCompute"):
		return "compute"
	case strings.Contains(apiName, "DrawArrays"):
		return "draw_arrays"
	case strings.Contains(apiName, "DrawElements") || strings.Contains(apiName, "DrawRangeElements"):
		return "draw_elements"
	default:
		return ""
	}
}

func indexedNumbers(params string) []int {
	fields := strings.Fields(strings.ReplaceAll(params, ",", " "))
	nums := make([]int, 0, len(fields))
	for _, field := range fields {
		if n := parseNumericParam(field); n != 0 || field == "0" {
			nums = append(nums, n)
		}
	}
	return nums
}

func parseInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func parseFloat64(s string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n
}
