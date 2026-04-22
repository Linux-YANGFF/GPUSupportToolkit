package parser

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"gst/internal/core"
)

var (
	// 匹配: [     1] glXxx: count=X, time=Y us 或 glXxx: count=X, time=Y us
	// 注意: 线程ID前缀是可选的
	apiLineRegex = regexp.MustCompile(`^(\[\s*\d+\]\s+)?(\w+):\s+count=(\d+),\s+time=(\d+)\s+us$`)
	// 匹配: swapBuffers: X us (可选线程ID前缀)
	swapBuffersRegex = regexp.MustCompile(`^(\[\s*\d+\]\s+)?swapBuffers:\s+(\d+)\s+us$`)
	// 匹配: frame cost Xms
	frameCostRegex = regexp.MustCompile(`^(\[\s*\d+\]\s+)?(\d+)\s+frame\s+cost\s+(\d+)ms$`)
	// 匹配: libGL: FPS = X
	fpsRegex = regexp.MustCompile(`libGL:\s+FPS\s*=\s*([\d.]+)`)
	// 匹配: vendor 行 (有或无前缀)
	vendorRegex = regexp.MustCompile(`^(\[\s*\d+\]\s+)?vendor:`)
	// 匹配: GC 行 <<gc = 0x...>>
	gcRegex = regexp.MustCompile(`^<<gc = 0x[a-fA-F0-9]+>>$`)
	// 匹配: ERROR 行 [tid:...:timestamp:ERROR:...]
	errorRegex = regexp.MustCompile(`^\[\d+:\d+:\d+.*ERROR.*`)
	// 匹配: Chrome日志行 [pid:tid:timestamp:module:message] 或行号开头的Chrome日志
	chromeLogRegex = regexp.MustCompile(`^(\d+\s+)?\[\d+:\d+:\d+.*\]`)
	// 匹配: warning 行
	warningRegex = regexp.MustCompile(`^warning:`)
	// 匹配: glShaderSource 行
	shaderSourceRegex = regexp.MustCompile(`(\[\s*\d+\]\s+)?glShaderSource`)
	// 匹配: #### shader source 边界 (可能带有行号前缀如 [ 54244], 以及可能的 \r)
	shaderBlockStartRegex = regexp.MustCompile(`^(\[\s*\d+\]\s+)?####\r?\n?$`)
	// 匹配原始 trace 格式的 glShaderSource 行:
	// [ 54243] (gc=..., tid=...): glShaderSource 16 1 0xffff... (nil)
	rawShaderSourceRegex = regexp.MustCompile(`^\[\s*\d+\]\s+\(gc=[^)]+,\s+tid=[^)]+\):\s+glShaderSource\s+(\d+)\s`)
)

// APIParser API日志解析器
type APIParser struct{}

func (p *APIParser) Kind() LogKind { return KindAPITrace }

func (p *APIParser) Parse(reader io.Reader) (*core.ParsedLog, error) {
	scanner := bufio.NewScanner(reader)
	// Increase buffer size to 10MB to support large log files (default is 64KB)
	buf := make([]byte, 10*1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var parsedLog core.ParsedLog
	var currentFrame *core.FrameInfo
	var fps float64
	var pendingFrameCostUs int64       // Frame cost time waiting to be applied (for current frame)
	var savedFrameCostUs int64         // Frame cost saved at ParsedLog level (for previous frame without API calls)
	var inShaderBlock bool
	var currentShaderSource []string
	var currentShaderCommand string // 当前 shader 的原始 glShaderSource 行
	var shaderID int
	var pendingRawShaderID int    // 原始格式 glShaderSource 的 ID
	var pendingRawShaderCommand string // 原始格式 glShaderSource 行
	var pendingRawShader bool    // 标记下一行是否是 #### 开头
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 跳过空行
		if len(line) == 0 {
			continue
		}

		// 跳过 Chrome 日志行
		if chromeLogRegex.MatchString(line) {
			continue
		}

		// 跳过 WARNING 行
		if warningRegex.MatchString(line) {
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

		// 检测原始 trace 格式的 glShaderSource 行
		if matches := rawShaderSourceRegex.FindStringSubmatch(line); len(matches) > 1 {
			if id, err := strconv.Atoi(matches[1]); err == nil {
				pendingRawShaderID = id
				pendingRawShaderCommand = line // 保存原始行
				pendingRawShader = true
			}
			continue
		}

		// 如果上一行是原始格式 glShaderSource，且当前行是 ####，则开始收集 shader 源码
		if pendingRawShader && shaderBlockStartRegex.MatchString(line) {
			inShaderBlock = true
			currentShaderSource = []string{}
			shaderID = pendingRawShaderID
			currentShaderCommand = pendingRawShaderCommand // 保存原始 glShaderSource 行
			pendingRawShader = false
			continue
		}
		// 如果上一行是原始格式 glShaderSource，但当前行不是 ####，则取消 pending
		if pendingRawShader {
			pendingRawShader = false
		}

		// 检测 shader source 块边界 (####)
		if shaderBlockStartRegex.MatchString(line) {
			if inShaderBlock {
				// 块结束 - 不收集 #### 行，直接保存
				inShaderBlock = false
				if currentFrame != nil && len(currentShaderSource) > 0 {
					currentFrame.Shaders = append(currentFrame.Shaders, &core.ShaderInfo{
						ID:          shaderID,
						CommandLine: currentShaderCommand,
						Source:      strings.Join(currentShaderSource, "\n"),
					})
				}
				currentShaderSource = nil
				currentShaderCommand = ""
			} else {
				// 块开始
				inShaderBlock = true
				currentShaderSource = []string{}
				shaderID = pendingRawShaderID          // 使用 glShaderSource 记录的 ID
				currentShaderCommand = pendingRawShaderCommand // 保存原始 glShaderSource 行
			}
			continue
		}
		if inShaderBlock {
			currentShaderSource = append(currentShaderSource, line) // 只收集非 #### 行
			continue
		}

		// 检测 frame cost - 可能是 swapBuffers 之后才出现
		// 格式: [thread_id] N frame cost Xms 或 N frame cost Xms
		if matches := frameCostRegex.FindStringSubmatch(line); len(matches) > 3 {
			frameCostMs, _ := strconv.ParseInt(matches[3], 10, 64)
			pendingFrameCostUs = frameCostMs * 1000

			// Bug 1 fix: 如果当前没有帧，保存到 ParsedLog 级别
			if currentFrame == nil && pendingFrameCostUs > 0 {
				savedFrameCostUs = pendingFrameCostUs
			}
			// 如果有pending的帧，用frame cost的时间覆盖
			if currentFrame != nil && pendingFrameCostUs > 0 {
				currentFrame.TotalTimeUs = pendingFrameCostUs
				pendingFrameCostUs = 0
			}
			continue
		}

		// 检测 glShaderSource 行 (聚合格式)
		if shaderSourceRegex.MatchString(line) {
			shaderID++
		}

		// 尝试解析 API 行 (聚合格式)
		if matches := apiLineRegex.FindStringSubmatch(line); len(matches) > 4 {
			apiName := matches[2]
			count, err := strconv.Atoi(matches[3])
			if err != nil {
				return nil, err
			}
			timeUs, err := strconv.ParseInt(matches[4], 10, 64)
			if err != nil {
				return nil, err
			}
			entry := core.APILogEntry{
				APIName: apiName,
				Count:   count,
				TimeUs:  timeUs,
				LineNum: lineNum,
			}
			if currentFrame == nil {
				currentFrame = &core.FrameInfo{
					FrameNum:    len(parsedLog.Frames),
					StartLine:   lineNum,
					TotalTimeUs: 0,
					APICalls:    []core.APILogEntry{},
					APISummary:  make(map[string]*core.APISummary),
					Shaders:     []*core.ShaderInfo{},
				}
				// Bug 1 fix: 创建新帧时检查是否有待处理的 frame cost
				if savedFrameCostUs > 0 {
					currentFrame.TotalTimeUs = savedFrameCostUs
					savedFrameCostUs = 0
				}
			}
			currentFrame.APICalls = append(currentFrame.APICalls, entry)
			// 只有在没有pending frame cost时才累加
			if pendingFrameCostUs == 0 {
				currentFrame.TotalTimeUs += timeUs
			}
			// Update summary
			if existing, ok := currentFrame.APISummary[apiName]; ok {
				existing.Count += count
				existing.TimeUs += timeUs
			} else {
				currentFrame.APISummary[apiName] = &core.APISummary{
					APIName: apiName,
					Count:   count,
					TimeUs:  timeUs,
				}
			}
			continue
		}

		// 尝试匹配 swapBuffers - 这是真正的帧边界
		if matches := swapBuffersRegex.FindStringSubmatch(line); len(matches) > 2 {
			swapTimeUs, _ := strconv.ParseInt(matches[2], 10, 64)
			if currentFrame != nil {
				currentFrame.EndLine = lineNum
				currentFrame.SwapBufferTimeUs = swapTimeUs
				// Bug fix: 先应用 savedFrameCostUs/pendingFrameCostUs，再计算 APITotalTimeUs
				if savedFrameCostUs > 0 {
					currentFrame.TotalTimeUs = savedFrameCostUs
					savedFrameCostUs = 0
				} else if pendingFrameCostUs > 0 {
					currentFrame.TotalTimeUs = pendingFrameCostUs
					pendingFrameCostUs = 0
				}
				// APITotalTimeUs = TotalTimeUs - SwapBufferTimeUs (在 TotalTimeUs 确定后计算)
				currentFrame.APITotalTimeUs = currentFrame.TotalTimeUs - swapTimeUs
				if currentFrame.APITotalTimeUs < 0 {
					currentFrame.APITotalTimeUs = 0
				}
				parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
				currentFrame = nil
			}
			continue
		}
	}

	// 处理最后一帧
	if currentFrame != nil {
		currentFrame.EndLine = lineNum
		if pendingFrameCostUs > 0 {
			currentFrame.TotalTimeUs = pendingFrameCostUs
		}
		parsedLog.Frames = append(parsedLog.Frames, *currentFrame)
	}

	// Handle any remaining shader block
	if inShaderBlock && len(currentShaderSource) > 0 && len(parsedLog.Frames) > 0 {
		lastFrame := &parsedLog.Frames[len(parsedLog.Frames)-1]
		lastFrame.Shaders = append(lastFrame.Shaders, &core.ShaderInfo{
			ID:          shaderID,
			CommandLine: currentShaderCommand,
			Source:      strings.Join(currentShaderSource, "\n"),
		})
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
