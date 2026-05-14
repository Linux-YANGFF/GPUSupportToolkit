package parser

import (
	"bufio"
	"gst/internal/core"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// RawTraceParser handles raw apiTrace format where each line is an individual API call.
// Example:
//   glXSwapBuffers: dpy = 0x1c002a1400, drawable = 121634855
//   glGenFramebuffers 1
//   glBindBuffer 0x8892 498
//   glBufferSubData 0x8892 0 8512 0x7fa1ba6970
type RawTraceParser struct{}

var (
	gcTidRegex         = regexp.MustCompile(`^\[\s*\d+\]\s*\(gc=(0x[0-9a-fA-F]+),\s*tid=(0x[0-9a-fA-F]+)\):\s*(.*)`)
	gcTidNoPrefixRegex = regexp.MustCompile(`^\(gc=(0x[0-9a-fA-F]+),\s*tid=(0x[0-9a-fA-F]+)\):\s*(.*)`)
	glSetErrorRegex    = regexp.MustCompile(`ERROR!!!\s*__glSetError\s*\(gl_error=([^,\)]+)`)
	segfaultRegex      = regexp.MustCompile(`(?:段错误|SIGSEGV|core\s+dumped)`)
	nilPtrRegex        = regexp.MustCompile(`(nil)`)
	ctxMgmtRegex       = regexp.MustCompile(`^(glXMakeCurrent|glXCreateContextAttribsARB)`)
)

// NewRawTraceParser creates a new RawTraceParser
func NewRawTraceParser() *RawTraceParser {
	return &RawTraceParser{}
}

func (p *RawTraceParser) Kind() LogKind {
	return KindRawTrace
}

func (p *RawTraceParser) Parse(reader io.Reader) (*core.ParsedLog, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, DefaultBufferSize)
	scanner.Buffer(buf, DefaultBufferSize)

	var parsedLog core.ParsedLog
	var currentFrame *core.FrameInfo
	frameNum := 0
	lineNum := 0
	var inShaderBlock bool
	var currentShaderSource []string
	var currentShaderID int
	var currentShaderCommand string
	var pendingShader bool
	var pendingShaderID int
	var pendingShaderCommand string
	var pendingReturnAPI string
	var pendingReturnValues []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		normalized := normalizeRawLine(line)

		if inShaderBlock {
			if isRawShaderBoundary(normalized) {
				inShaderBlock = false
				if currentFrame != nil && currentShaderID > 0 && len(currentShaderSource) > 0 {
					currentFrame.Shaders = append(currentFrame.Shaders, &core.ShaderInfo{
						ID:          currentShaderID,
						CommandLine: currentShaderCommand,
						Source:      strings.Join(currentShaderSource, "\n"),
					})
				}
				currentShaderSource = nil
				currentShaderID = 0
				currentShaderCommand = ""
			} else {
				currentShaderSource = append(currentShaderSource, normalized)
			}
			continue
		}

		if pendingShader && isRawShaderBoundary(normalized) {
			inShaderBlock = true
			currentShaderSource = []string{}
			currentShaderID = pendingShaderID
			currentShaderCommand = pendingShaderCommand
			pendingShader = false
			continue
		}
		if pendingShader {
			pendingShader = false
		}

		if pendingReturnAPI != "" {
			switch {
			case normalized == "{":
				continue
			case normalized == "}":
				assignReturnToLastCall(currentFrame, pendingReturnAPI, strings.Join(pendingReturnValues, " "))
				pendingReturnAPI = ""
				pendingReturnValues = nil
				continue
			case normalized != "" && !looksLikeRawAPICall(normalized) && !strings.Contains(normalized, "=>"):
				pendingReturnValues = append(pendingReturnValues, normalized)
				continue
			default:
				assignReturnToLastCall(currentFrame, pendingReturnAPI, strings.Join(pendingReturnValues, " "))
				pendingReturnAPI = ""
				pendingReturnValues = nil
			}
		}

		if matches := fpsRegex.FindStringSubmatch(normalized); len(matches) > 1 {
			if fps, err := strconv.ParseFloat(matches[1], 64); err == nil {
				parsedLog.FPS = fps
			}
			continue
		}

		if matches := swapBuffersRegex.FindStringSubmatch(normalized); len(matches) > 2 {
			swapTimeUs, _ := strconv.ParseInt(matches[2], 10, 64)
			applySwapTiming(&parsedLog, currentFrame, swapTimeUs)
			continue
		}

		if matches := frameCostRegex.FindStringSubmatch(normalized); len(matches) > 3 {
			frameID, _ := strconv.Atoi(matches[2])
			frameCostMs, _ := strconv.ParseInt(matches[3], 10, 64)
			applyFrameCostTiming(&parsedLog, currentFrame, frameID, frameCostMs*1000)
			continue
		}

		if apiName, value, ok := parseReturnLine(normalized); ok {
			if value == "" {
				pendingReturnAPI = apiName
				pendingReturnValues = []string{}
			} else {
				assignReturnToLastCall(currentFrame, apiName, value)
			}
			continue
		}

		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		if chromeLogRegex.MatchString(line) {
			continue
		}

		if warningRegex.MatchString(line) {
			continue
		}

		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t' || line[0] == '=') {
			continue
		}

		// Skip return value indicators and structural markers (but NOT __glSetError)
		if strings.HasPrefix(normalized, "=>") ||
			strings.HasPrefix(normalized, "src:") || strings.HasPrefix(normalized, "dst:") ||
			strings.HasPrefix(normalized, "{") || strings.HasPrefix(normalized, "}") ||
			strings.HasPrefix(normalized, "[__dri3") {
			continue
		}

		// Check for __glSetError before the generic __ prefix skip
		if strings.HasPrefix(line, "__glSetError") || strings.Contains(line, "__glSetError") {
			entry := core.APILogEntry{
				APIName:   "__glSetError",
				Count:     1,
				LineNum:   lineNum,
				IsError:   true,
				RawParams: line,
			}
			if matches := glSetErrorRegex.FindStringSubmatch(line); len(matches) > 1 {
				entry.ErrorCode = "gl_error=" + matches[1]
			}
			ensureFrame(&currentFrame, frameNum, lineNum)
			appendRawCall(currentFrame, entry)
			continue
		}

		// Skip other __ prefixed lines
		if strings.HasPrefix(line, "__") {
			continue
		}

		// Detect segment fault markers
		if segfaultRegex.MatchString(line) {
			entry := core.APILogEntry{
				APIName:   "__segfault__",
				Count:     1,
				LineNum:   lineNum,
				IsError:   true,
				ErrorCode: "SIGSEGV",
				RawParams: line,
			}
			ensureFrame(&currentFrame, frameNum, lineNum)
			appendRawCall(currentFrame, entry)
			continue
		}

		// Extract gc= and tid= from context prefix: (gc=0x..., tid=0x...):
		gcAddr, tid, workLine := extractGCTID(line)

		apiName, params := parseAPICall(workLine)
		if apiName == "" {
			continue
		}

		// Frame boundary detection
		if isRawFrameBoundary(apiName) {
			if currentFrame != nil {
				currentFrame.EndLine = lineNum
				finalizeRawFrame(currentFrame)
				parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
				currentFrame = nil
				frameNum++
			}
			continue
		}

		ensureFrame(&currentFrame, frameNum, lineNum)

		hasNilPtr := nilPtrRegex.MatchString(params)

		entry := core.APILogEntry{
			APIName:   apiName,
			Count:     1,
			TimeUs:    0,
			LineNum:   lineNum,
			RawParams: params,
			GCAddr:    gcAddr,
			TID:       tid,
			HasNilPtr: hasNilPtr,
		}

		appendRawCall(currentFrame, entry)

		if apiName == "glShaderSource" {
			if shaderID := firstIntParam(params); shaderID > 0 {
				pendingShader = true
				pendingShaderID = shaderID
				pendingShaderCommand = normalized
			}
		}

		if apiName == "glUseProgram" {
			if progID := extractProgramID(params); progID > 0 {
				currentFrame.Programs = append(currentFrame.Programs, progID)
			}
		}

		if apiName == "glGenBuffers" || apiName == "glCreateBuffers" {
			if ids := extractBufferIDs(params); len(ids) > 0 {
				for _, id := range ids {
					bufInfo := core.BufferInfo{
						ID:     id,
						Target: "GL_ARRAY_BUFFER",
						Size:   0,
						Usage:  "",
					}
					currentFrame.BufferCreations = append(currentFrame.BufferCreations, bufInfo)
				}
			}
		}
	}

	if pendingReturnAPI != "" {
		assignReturnToLastCall(currentFrame, pendingReturnAPI, strings.Join(pendingReturnValues, " "))
	}
	if inShaderBlock && currentFrame != nil && currentShaderID > 0 && len(currentShaderSource) > 0 {
		currentFrame.Shaders = append(currentFrame.Shaders, &core.ShaderInfo{
			ID:          currentShaderID,
			CommandLine: currentShaderCommand,
			Source:      strings.Join(currentShaderSource, "\n"),
		})
	}
	if currentFrame != nil {
		currentFrame.EndLine = lineNum
		finalizeRawFrame(currentFrame)
		parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
	}

	for _, frame := range parsedLog.Frames {
		parsedLog.TotalTimeUs += frame.TotalTimeUs
	}
	if parsedLog.FPS == 0 && len(parsedLog.Frames) > 0 && parsedLog.TotalTimeUs > 0 {
		parsedLog.FPS = float64(len(parsedLog.Frames)) * 1e6 / float64(parsedLog.TotalTimeUs)
	}

	return &parsedLog, scanner.Err()
}

func normalizeRawLine(line string) string {
	return strings.TrimSpace(removeRawTraceLinePrefix(strings.TrimSpace(line)))
}

func isRawShaderBoundary(line string) bool {
	return strings.TrimSpace(line) == "####"
}

func looksLikeRawAPICall(line string) bool {
	apiName, _ := parseAPICall(line)
	return apiName != ""
}

func parseReturnLine(line string) (apiName string, value string, ok bool) {
	idx := strings.Index(line, "=>")
	if idx < 0 {
		return "", "", false
	}
	left := strings.TrimSpace(line[:idx])
	right := strings.TrimSpace(line[idx+2:])
	if left == "" {
		return "", "", false
	}
	fields := strings.Fields(left)
	if len(fields) == 0 {
		return "", "", false
	}
	name := fields[0]
	if !(strings.HasPrefix(name, "gl") || strings.HasPrefix(name, "egl") || strings.HasPrefix(name, "glut")) {
		return "", "", false
	}
	return name, right, true
}

func assignReturnToLastCall(frame *core.FrameInfo, apiName string, value string) {
	if frame == nil || apiName == "" {
		return
	}
	for i := len(frame.APICalls) - 1; i >= 0; i-- {
		if frame.APICalls[i].APIName == apiName && frame.APICalls[i].ReturnValue == "" {
			frame.APICalls[i].ReturnValue = value
			return
		}
	}
}

func ensureFrame(frame **core.FrameInfo, frameNum int, lineNum int) {
	if *frame == nil {
		*frame = &core.FrameInfo{
			FrameNum:     frameNum,
			StartLine:    lineNum,
			TotalTimeUs:  0,
			HasTiming:    false,
			TimingSource: "none",
			APICalls:     []core.APILogEntry{},
			APISummary:   make(map[string]*core.APISummary),
			Shaders:      []*core.ShaderInfo{},
		}
	}
}

func appendRawCall(frame *core.FrameInfo, entry core.APILogEntry) {
	if frame == nil {
		return
	}
	if entry.Count == 0 {
		entry.Count = 1
	}
	frame.APICalls = append(frame.APICalls, entry)
	if frame.APISummary == nil {
		frame.APISummary = make(map[string]*core.APISummary)
	}
	summary, ok := frame.APISummary[entry.APIName]
	if !ok {
		frame.APISummary[entry.APIName] = &core.APISummary{
			APIName: entry.APIName,
			Count:   entry.Count,
			TimeUs:  entry.TimeUs,
		}
		return
	}
	summary.Count += entry.Count
	summary.TimeUs += entry.TimeUs
}

func finalizeRawFrame(frame *core.FrameInfo) {
	if frame == nil {
		return
	}
	if frame.TimingSource == "" {
		frame.TimingSource = "none"
	}
	if frame.HasTiming && frame.TotalTimeUs > 0 {
		frame.APITotalTimeUs = frame.TotalTimeUs - frame.SwapBufferTimeUs
		if frame.APITotalTimeUs < 0 {
			frame.APITotalTimeUs = 0
		}
	}
}

func applySwapTiming(log *core.ParsedLog, currentFrame *core.FrameInfo, swapTimeUs int64) {
	frame := lastRawFrame(log, currentFrame, -1)
	if frame == nil {
		return
	}
	frame.SwapBufferTimeUs = swapTimeUs
	finalizeRawFrame(frame)
}

func applyFrameCostTiming(log *core.ParsedLog, currentFrame *core.FrameInfo, frameNum int, totalTimeUs int64) {
	frame := lastRawFrame(log, currentFrame, frameNum)
	if frame == nil {
		return
	}
	frame.TotalTimeUs = totalTimeUs
	frame.HasTiming = totalTimeUs > 0
	if frame.HasTiming {
		frame.TimingSource = "frame_cost"
	}
	finalizeRawFrame(frame)
}

func lastRawFrame(log *core.ParsedLog, currentFrame *core.FrameInfo, frameNum int) *core.FrameInfo {
	if frameNum >= 0 {
		for i := len(log.Frames) - 1; i >= 0; i-- {
			if log.Frames[i].FrameNum == frameNum {
				return &log.Frames[i]
			}
		}
		if currentFrame != nil && currentFrame.FrameNum == frameNum {
			return currentFrame
		}
	}
	if len(log.Frames) > 0 {
		return &log.Frames[len(log.Frames)-1]
	}
	return currentFrame
}

func extractGCTID(line string) (gcAddr, tid, workLine string) {
	if matches := gcTidRegex.FindStringSubmatch(line); len(matches) > 3 {
		return matches[1], matches[2], matches[3]
	}
	if matches := gcTidNoPrefixRegex.FindStringSubmatch(line); len(matches) > 3 {
		return matches[1], matches[2], matches[3]
	}
	return "", "", line
}

// removeRawTraceLinePrefix removes line号前缀 [N] or [sequence]
func removeRawTraceLinePrefix(line string) string {
	if len(line) > 0 && line[0] == '[' {
		if idx := strings.Index(line, "]"); idx > 0 && idx < len(line)-1 {
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return line
}

// parseAPICall extracts the API name and parameters from a raw trace line
// Examples:
//   "[  4090] glXSwapBuffers: dpy = 0x1c002a1400, drawable = 121634855" -> "glXSwapBuffers", "dpy = 0x1c002a1400, drawable = 121634855"
//   "[     1] glGenFramebuffers 1" -> "glGenFramebuffers", "1"
//   "glBindBuffer 0x8892 498" -> "glBindBuffer", "0x8892 498"
func parseAPICall(line string) (string, string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}

	// Remove sequence prefix [N] if present
	line = removeRawTraceLinePrefix(line)

	// Handle "glXxx: params" format (with colon)
	if idx := strings.Index(line, ":"); idx > 0 && idx < 50 {
		apiName := strings.TrimSpace(line[:idx])
		// Make sure it's a valid API name (starts with gl)
		if strings.HasPrefix(apiName, "gl") || strings.HasPrefix(apiName, "egl") || strings.HasPrefix(apiName, "glut") {
			return apiName, strings.TrimSpace(line[idx+1:])
		}
	}

	// Handle "glXxx params" format (space-separated)
	parts := strings.Fields(line)
	if len(parts) >= 1 {
		apiName := parts[0]
		if strings.HasPrefix(apiName, "gl") || strings.HasPrefix(apiName, "egl") || strings.HasPrefix(apiName, "glut") {
			if idx := strings.Index(apiName, "("); idx > 0 {
				name := apiName[:idx]
				params := strings.TrimSuffix(apiName[idx+1:], ")")
				if len(parts) > 1 {
					if params != "" {
						params += " "
					}
					params += strings.Join(parts[1:], " ")
				}
				return name, strings.TrimSpace(params)
			}
			if len(parts) > 1 {
				return apiName, strings.Join(parts[1:], " ")
			}
			return apiName, ""
		}
	}

	return "", ""
}

// isRawFrameBoundary checks if the API call is a frame boundary
func isRawFrameBoundary(apiName string) bool {
	return apiName == "glXSwapBuffers" || apiName == "eglSwapBuffers" ||
		apiName == "glSwapBuffers" || apiName == "eglPresentationTime" ||
		apiName == "__dri3HandlePresentEvent"
}

// extractProgramID extracts program ID from glUseProgram params
// Format: "18" or "program = 18"
func extractProgramID(params string) int {
	params = strings.TrimSpace(params)
	// Try direct number first
	if id, err := strconv.Atoi(params); err == nil {
		return id
	}
	// Try "program = X" format
	re := regexp.MustCompile(`(?i)(?:program\s*=|)\s*(\d+)`)
	matches := re.FindStringSubmatch(params)
	if len(matches) > 1 {
		if id, err := strconv.Atoi(matches[1]); err == nil {
			return id
		}
	}
	return 0
}

func firstIntParam(params string) int {
	parts := strings.Fields(strings.TrimSpace(params))
	if len(parts) == 0 {
		return 0
	}
	return parseNumericParam(parts[0])
}

func parseNumericParam(s string) int {
	s = strings.TrimSpace(strings.Trim(s, ","))
	if s == "" || s == "(nil)" {
		return 0
	}
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		v, err := strconv.ParseInt(s[2:], 16, 64)
		if err != nil {
			return 0
		}
		return int(v)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

// extractBufferIDs extracts buffer IDs from glGenBuffers/glCreateBuffers params
// Format: "1" or "1, 2, 3" or just a single number
func extractBufferIDs(params string) []int {
	params = strings.TrimSpace(params)
	var ids []int
	// Split by comma or space
	parts := regexp.MustCompile(`[,\s]+`).Split(params, -1)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := strconv.Atoi(p); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
