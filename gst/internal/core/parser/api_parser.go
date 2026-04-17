package parser

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"gst/internal/core"
)

var (
	// 匹配: [     1] glXxx: count=X, time=Y us (带线程ID前缀)
	apiLineRegex = regexp.MustCompile(`^\[\s*\d+\]\s+(\w+):\s+count=(\d+),\s+time=(\d+)\s+us$`)
	// 匹配: swapBuffers: X us
	swapBuffersRegex = regexp.MustCompile(`^\[\s*\d+\]\s+swapBuffers:\s+(\d+)\s+us$`)
	// 匹配: frame cost Xms
	frameCostRegex = regexp.MustCompile(`^\[\s*\d+\]\s+(\d+)\s+frame\s+cost\s+(\d+)ms$`)
	// 匹配: libGL: FPS = X
	fpsRegex = regexp.MustCompile(`libGL:\s+FPS\s*=\s*([\d.]+)`)
	// 匹配: vendor 行
	vendorRegex = regexp.MustCompile(`^vendor:`)
	// 匹配: GC 行 <<gc = 0x...>>
	gcRegex = regexp.MustCompile(`^<<gc = 0x[a-fA-F0-9]+>>$`)
	// 匹配: ERROR 行 [tid:...:timestamp:ERROR:...]
	errorRegex = regexp.MustCompile(`^\[\d+:\d+:\d+.*ERROR.*`)
)

// APIParser API日志解析器
type APIParser struct{}

func (p *APIParser) Kind() LogKind { return KindAPITrace }

func (p *APIParser) Parse(reader io.Reader) (*core.ParsedLog, error) {
	scanner := bufio.NewScanner(reader)
	var parsedLog core.ParsedLog
	var currentFrame *core.FrameInfo
	var fps float64
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 跳过空行
		if len(line) == 0 {
			continue
		}

		// 跳过 ERROR 行
		if errorRegex.MatchString(line) {
			continue
		}

		// 跳过 GC 行
		if gcRegex.MatchString(line) {
			continue
		}

		// 跳过 vendor 行
		if vendorRegex.MatchString(line) {
			continue
		}

		// 解析 FPS
		if matches := fpsRegex.FindStringSubmatch(line); len(matches) > 1 {
			if f, err := strconv.ParseFloat(matches[1], 64); err == nil {
				fps = f
			}
			continue
		}

		// 尝试解析 API 行
		if matches := apiLineRegex.FindStringSubmatch(line); len(matches) > 3 {
			count, err := strconv.Atoi(matches[2])
			if err != nil {
				return nil, err
			}
			timeUs, err := strconv.ParseInt(matches[3], 10, 64)
			if err != nil {
				return nil, err
			}
			entry := core.APILogEntry{
				APIName: matches[1],
				Count:   count,
				TimeUs:  timeUs,
				LineNum: lineNum,
			}
			if currentFrame == nil {
				currentFrame = &core.FrameInfo{
					FrameNum:    len(parsedLog.Frames),
					StartLine:    lineNum,
					TotalTimeUs:  0,
					APICalls:     []core.APILogEntry{},
				}
			}
			currentFrame.APICalls = append(currentFrame.APICalls, entry)
			currentFrame.TotalTimeUs += timeUs
			continue
		}

		// 检测帧边界: swapBuffers 或 frame cost
		if isFrameBoundary(line) {
			if currentFrame != nil {
				currentFrame.EndLine = lineNum
				parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
				currentFrame = nil
			}
			continue
		}

		// 尝试匹配 swapBuffers
		if matches := swapBuffersRegex.FindStringSubmatch(line); len(matches) > 1 {
			if currentFrame != nil {
				currentFrame.EndLine = lineNum
				parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
				currentFrame = nil
			}
			continue
		}

		// 尝试匹配 frame cost
		if matches := frameCostRegex.FindStringSubmatch(line); len(matches) > 2 {
			if currentFrame != nil {
				currentFrame.EndLine = lineNum
				parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
				currentFrame = nil
			}
			continue
		}
	}

	// 处理最后一帧
	if currentFrame != nil {
		currentFrame.EndLine = lineNum
		parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
	}

	// 计算总时间和 FPS
	for _, frame := range parsedLog.Frames {
		parsedLog.TotalTimeUs += frame.TotalTimeUs
	}
	if fps > 0 {
		parsedLog.FPS = fps
	} else if parsedLog.TotalTimeUs > 0 && len(parsedLog.Frames) > 0 {
		parsedLog.FPS = float64(len(parsedLog.Frames)) * 1e6 / float64(parsedLog.TotalTimeUs)
	}

	return &parsedLog, scanner.Err()
}

// isFrameBoundary 判断是否是帧边界
func isFrameBoundary(line string) bool {
	return swapBuffersRegex.MatchString(line) || frameCostRegex.MatchString(line)
}
