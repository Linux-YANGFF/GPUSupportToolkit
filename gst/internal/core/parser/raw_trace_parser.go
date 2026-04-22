package parser

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"gst/internal/core"
)

// RawTraceParser handles raw apiTrace format where each line is an individual API call.
// Example:
//   glXSwapBuffers: dpy = 0x1c002a1400, drawable = 121634855
//   glGenFramebuffers 1
//   glBindBuffer 0x8892 498
//   glBufferSubData 0x8892 0 8512 0x7fa1ba6970
type RawTraceParser struct{}

// NewRawTraceParser creates a new RawTraceParser
func NewRawTraceParser() *RawTraceParser {
	return &RawTraceParser{}
}

func (p *RawTraceParser) Kind() LogKind {
	return KindRawTrace
}

func (p *RawTraceParser) Parse(reader io.Reader) (*core.ParsedLog, error) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size to 10MB to support large log files (default is 64KB)
	buf := make([]byte, 10*1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var parsedLog core.ParsedLog
	var currentFrame *core.FrameInfo
	frameNum := 0
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Skip Chrome log lines: [pid:tid:timestamp:module:message]
		if chromeLogRegex.MatchString(line) {
			continue
		}

		// Skip WARNING lines
		if warningRegex.MatchString(line) {
			continue
		}

		// Skip indented continuation lines (return values, etc.)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t' || line[0] == '=') {
			continue
		}

		// Skip special markers
		if strings.HasPrefix(line, "=>") || strings.HasPrefix(line, "__") ||
			strings.HasPrefix(line, "src:") || strings.HasPrefix(line, "dst:") ||
			strings.HasPrefix(line, "{") || strings.HasPrefix(line, "}") ||
			strings.HasPrefix(line, "[__dri3") {
			continue
		}

		// Skip lines with context prefix: (gc=0x..., tid=0x...):
		if strings.HasPrefix(line, "(") && strings.Contains(line, "gc=") {
			continue
		}

		// Parse the API call
		apiName, params := parseAPICall(line)
		if apiName == "" {
			continue
		}

		// Check if this is a frame boundary (swapBuffers)
		if isRawFrameBoundary(apiName) {
			// Save current frame if exists
			if currentFrame != nil {
				currentFrame.EndLine = lineNum - 1
				parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
				currentFrame = nil
			}
			frameNum++
			continue
		}

		// Start a new frame if needed
		if currentFrame == nil {
			currentFrame = &core.FrameInfo{
				FrameNum:    frameNum,
				StartLine:   lineNum,
				TotalTimeUs: 0, // Raw format has no timing info
				APICalls:    []core.APILogEntry{},
				APISummary:  make(map[string]*core.APISummary),
				Shaders:     []*core.ShaderInfo{},
			}
		}

		// Create API entry
		entry := core.APILogEntry{
			APIName:   apiName,
			Count:     1, // Raw format: each line is one call
			TimeUs:    0, // Raw format: no timing info
			LineNum:   lineNum,
			RawParams: params,
		}

		currentFrame.APICalls = append(currentFrame.APICalls, entry)

		// Track shader programs
		if apiName == "glUseProgram" {
			if progID := extractProgramID(params); progID > 0 {
				currentFrame.Programs = append(currentFrame.Programs, progID)
			}
		}

		// Track buffer operations
		if apiName == "glGenBuffers" || apiName == "glCreateBuffers" {
			if ids := extractBufferIDs(params); len(ids) > 0 {
				for _, id := range ids {
					bufInfo := core.BufferInfo{
						ID:     id,
						Target: "GL_ARRAY_BUFFER", // Default, will be updated by glBindBuffer
						Size:   0,
						Usage:  "",
					}
					currentFrame.BufferCreations = append(currentFrame.BufferCreations, bufInfo)
				}
			}
		}
	}

	// Handle last frame
	if currentFrame != nil {
		currentFrame.EndLine = lineNum
		parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
	}

	// Calculate stats
	for _, frame := range parsedLog.Frames {
		parsedLog.TotalTimeUs += frame.TotalTimeUs
	}
	if len(parsedLog.Frames) > 0 && parsedLog.TotalTimeUs > 0 {
		parsedLog.FPS = float64(len(parsedLog.Frames)) * 1e6 / float64(parsedLog.TotalTimeUs)
	}

	return &parsedLog, scanner.Err()
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
