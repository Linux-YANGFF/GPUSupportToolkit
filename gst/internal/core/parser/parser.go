package parser

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"gst/internal/core"
)

// LogKind 日志类型
type LogKind string

const (
	KindAPITrace LogKind = "apitrace"  // 聚合统计格式: glXxx: count=X, time=Y us
	KindProfile  LogKind = "profile"   // Profile格式 (同APITrace)
	KindRawTrace LogKind = "rawtrace"  // 原始apiTrace格式: 每行一个API调用
	KindUnknown  LogKind = "unknown"
)

// Parser 解析器接口
type Parser interface {
	Parse(reader io.Reader) (*core.ParsedLog, error)
	Kind() LogKind
}

// 跳过行的正则表达式
var (
	// ERROR 行: [timestamp:ERROR:...] 或 [pid:tid:timestamp:ERROR:...]
	errorLineRegex = regexp.MustCompile(`^\[\d+:\d+:\d+.*ERROR.*`)
	// Chrome 日志行: [pid:tid:timestamp:...]
	chromeLogLineRegex = regexp.MustCompile(`^\[\d+:\d+:\d+.*\]`)
	// 空行或只包含空白
	emptyLineRegex = regexp.MustCompile(`^\s*$`)
	// vendor 行（有或无序列号前缀）
	vendorLineRegex = regexp.MustCompile(`^(\[\s*\d+\]\s+)?vendor:`)
	// GC 行
	gcLineRegex = regexp.MustCompile(`^<<gc = 0x[a-fA-F0-9]+>>$`)
)

// DetectKind 自动检测日志类型（检查多行找到第一个有效行）
func DetectKind(firstLine string) LogKind {
	if firstLine == "" {
		return KindUnknown
	}
	// 如果第一行是有效的，直接检测
	if kind := detectFromLine(firstLine); kind != KindUnknown {
		return kind
	}
	return KindUnknown
}

// DetectKindFromReader 从读取器检测日志类型（扫描前N行找到第一个有效行）
func DetectKindFromReader(reader io.Reader, maxLines int) LogKind {
	scanner := bufio.NewScanner(reader)
	// Increase default buffer size for large lines
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	lineNum := 0
	hasAggregatedFormat := false

	for scanner.Scan() && lineNum < maxLines {
		line := scanner.Text()
		lineNum++

		// 在判断是否跳过之前，先检查是否是聚合适式的关键特征
		// 这样即使该行会被跳过，我们也能识别出文件包含聚合适式
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "count=") && strings.Contains(trimmed, "time=") {
			hasAggregatedFormat = true
		}
		// swapBuffers 带时间格式是帧边界的明确标志
		if strings.Contains(trimmed, "swapBuffers:") && strings.Contains(trimmed, "us") {
			hasAggregatedFormat = true
		}
		// frame cost 是帧边界标记
		if strings.Contains(trimmed, "frame cost") {
			hasAggregatedFormat = true
		}

		// 跳过错误行、日志行（但我们已经检查过聚合适式特征）
		if shouldSkipLine(line) {
			continue
		}

		// 检测有效行
		if kind := detectFromLine(line); kind != KindUnknown {
			// 如果发现聚合适式，优先使用APITrace解析器
			if hasAggregatedFormat {
				return KindAPITrace
			}
			// 如果检测到原始格式且没有聚合适式特征，直接返回
			if kind == KindRawTrace {
				return KindRawTrace
			}
			// 其他已知格式直接返回
			return kind
		}

		// 即使当前行不是已知格式，只要发现聚合适式特征就使用APITrace
		if hasAggregatedFormat {
			return KindAPITrace
		}
	}

	// 如果扫描完所有行发现有聚合适式但没检测到，用APITrace
	if hasAggregatedFormat {
		return KindAPITrace
	}

	return KindUnknown
}

// shouldSkipLine 判断是否应该跳过该行
func shouldSkipLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	// 跳过 ERROR 行
	if errorLineRegex.MatchString(trimmed) {
		return true
	}
	// 跳过 Chrome 日志行（通常是 [pid:tid:timestamp:module:message] 格式）
	if chromeLogLineRegex.MatchString(trimmed) && !strings.Contains(trimmed, "gl") && !strings.Contains(trimmed, "swapBuffers") {
		return true
	}
	// 跳过空行
	if emptyLineRegex.MatchString(trimmed) {
		return true
	}
	// 跳过 GC 行
	if gcLineRegex.MatchString(trimmed) {
		return true
	}
	// 跳过 vendor 行
	if vendorLineRegex.MatchString(trimmed) {
		return true
	}
	return false
}

// detectFromLine 从单行检测日志类型
func detectFromLine(line string) LogKind {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return KindUnknown
	}

	// 移除可能的行号前缀 [N] 或 [sequence]
	// 格式: [     1] glXxx 或 [12345] glXxx
	trimmed = removeLinePrefix(trimmed)

	// Profile格式通常以 "<<" 开头
	if strings.HasPrefix(trimmed, "<<") {
		return KindProfile
	}

	// 聚合API trace格式包含 "count=" 和 "time="
	if strings.Contains(trimmed, "count=") && strings.Contains(trimmed, "time=") {
		return KindAPITrace
	}

	// 原始apiTrace格式
	if isRawTraceFormat(trimmed) {
		return KindRawTrace
	}

	return KindUnknown
}

// removeLinePrefix 移除行号前缀
func removeLinePrefix(line string) string {
	// 处理 [N] 或 [序列] 前缀
	if len(line) > 0 && line[0] == '[' {
		// 找到匹配的 ]
		if idx := strings.Index(line, "]"); idx > 0 && idx < len(line)-1 {
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return line
}

// isRawTraceFormat 判断是否是原始apiTrace格式
func isRawTraceFormat(line string) bool {
	trimmed := strings.TrimSpace(line)
	// 以 glXSwapBuffers: 或 eglSwapBuffers: 开头（帧边界调用）
	if strings.HasPrefix(trimmed, "glXSwapBuffers") || strings.HasPrefix(trimmed, "eglSwapBuffers") {
		return true
	}
	// 以 glGen/glCreate/glBind/glBuffer 等开头（OpenGL API调用）
	patterns := []string{
		"glX", "glGen", "glCreate", "glBind", "glBuffer", "glDraw", "glClear",
		"glUseProgram", "glShader", "glCompile", "glLink", "glVertex",
		"glPixel", "glTex", "glEnable", "glDisable", "glFlush", "glFinish",
		"glMap", "glUnmap", "glDelete", "glGet", "glIs", "glRead",
	}
	for _, p := range patterns {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

// CreateParser 创建对应类型的解析器
func CreateParser(kind LogKind) Parser {
	switch kind {
	case KindAPITrace:
		return &APIParser{}
	case KindProfile:
		return NewProfileParser()
	case KindRawTrace:
		return NewRawTraceParser()
	default:
		return &APIParser{}
	}
}

// CreateParserAuto 自动检测日志类型并创建解析器
func CreateParserAuto(reader io.Reader) (Parser, error) {
	// 需要支持 Seek 的 reader 来重复读取
	seeker, ok := reader.(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("reader must support seeking for auto-detection")
	}

	kind := DetectKindFromReader(reader, 50)

	// 重置读取位置
	_, err := seeker.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to reset reader: %w", err)
	}

	return CreateParser(kind), nil
}
