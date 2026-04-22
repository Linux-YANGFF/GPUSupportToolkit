package analyzer

import (
	"fmt"
	"gst/internal/core"
)

// BufferTarget constants
const (
	TargetArrayBuffer         = "GL_ARRAY_BUFFER"
	TargetElementArrayBuffer = "GL_ELEMENT_ARRAY_BUFFER"
	TargetPixelPackBuffer    = "GL_PIXEL_PACK_BUFFER"
	TargetPixelUnpackBuffer  = "GL_PIXEL_UNPACK_BUFFER"
	TargetUniformBuffer      = "GL_UNIFORM_BUFFER"
	TargetTransformFeedback  = "GL_TRANSFORM_FEEDBACK"
	TargetCopyReadBuffer     = "GL_COPY_READ_BUFFER"
	TargetCopyWriteBuffer    = "GL_COPY_WRITE_BUFFER"
	TargetDrawIndirectBuffer = "GL_DRAW_INDIRECT_BUFFER"
	TargetShaderStorageBuffer = "GL_SHADER_STORAGE_BUFFER"
)

// BufferUsage constants
const (
	UsageStreamDraw  = "GL_STREAM_DRAW"
	UsageStaticDraw  = "GL_STATIC_DRAW"
	UsageDynamicDraw = "GL_DYNAMIC_DRAW"
	UsageStreamRead  = "GL_STREAM_READ"
	UsageStaticRead  = "GL_STATIC_READ"
	UsageDynamicRead = "GL_DYNAMIC_READ"
	UsageStreamCopy  = "GL_STREAM_COPY"
	UsageStaticCopy  = "GL_STATIC_COPY"
	UsageDynamicCopy = "GL_DYNAMIC_COPY"
)

// BufferUsageHint maps common usage patterns to OpenGL constants
var BufferUsageHint = map[string]string{
	"0x88B0": TargetArrayBuffer,
	"0x88B1": TargetElementArrayBuffer,
	"0x88B8": TargetPixelPackBuffer,
	"0x88B9": TargetPixelUnpackBuffer,
	"0x8B11": TargetUniformBuffer,
	"0x8C8A": TargetTransformFeedback,
	"0x8B8F": TargetCopyReadBuffer,
	"0x8B8E": TargetCopyWriteBuffer,
	"0x8F3F": TargetDrawIndirectBuffer,
	"0x90D2": TargetShaderStorageBuffer,
}

// BufferUsagePattern maps OpenGL constants to usage strings
var BufferUsagePattern = map[string]string{
	"0x88B0": UsageStaticDraw, // GL_ARRAY_BUFFER
	"0x88B1": UsageStaticDraw, // GL_ELEMENT_ARRAY_BUFFER
	"0x88B8": UsageStreamDraw, // GL_PIXEL_PACK_BUFFER
	"0x88B9": UsageStreamDraw, // GL_PIXEL_UNPACK_BUFFER
	"0x8B11": UsageStaticDraw, // GL_UNIFORM_BUFFER
	"0x8C8A": UsageStreamDraw, // GL_TRANSFORM_FEEDBACK
	"0x8B8F": UsageStreamDraw, // GL_COPY_READ_BUFFER
	"0x8B8E": UsageStreamDraw, // GL_COPY_WRITE_BUFFER
	"0x8F3F": UsageStreamDraw, // GL_DRAW_INDIRECT_BUFFER
	"0x90D2": UsageStaticDraw, // GL_SHADER_STORAGE_BUFFER
}

// BufferAnalyzer 缓冲区分析器
type BufferAnalyzer struct {
	log         *core.ParsedLog
	buffers     map[int]*core.BufferInfo
	byTarget    map[string][]*core.BufferInfo
	totalSize   int64
}

// NewBufferAnalyzer 创建缓冲区分析器
func NewBufferAnalyzer(log *core.ParsedLog) *BufferAnalyzer {
	ba := &BufferAnalyzer{
		log:      log,
		buffers:  make(map[int]*core.BufferInfo),
		byTarget: make(map[string][]*core.BufferInfo),
	}
	ba.analyze()
	return ba
}

// analyze 执行分析
func (ba *BufferAnalyzer) analyze() {
	if ba.log == nil {
		return
	}

	for _, frame := range ba.log.Frames {
		for i := range frame.APICalls {
			call := &frame.APICalls[i]
			ba.processAPICall(call)
		}

		// 处理帧内创建的缓冲区
		for i := range frame.BufferCreations {
			buf := &frame.BufferCreations[i]
			ba.addBuffer(buf)
		}
	}
}

// processAPICall 处理API调用
func (ba *BufferAnalyzer) processAPICall(call *core.APILogEntry) {
	switch call.APIName {
	case "glGenBuffers", "glCreateBuffers":
		ba.processGenBuffers(call)
	case "glBindBuffer":
		ba.processBindBuffer(call)
	case "glBufferData":
		ba.processBufferData(call)
	case "glBufferSubData":
		ba.processBufferSubData(call)
	case "glDeleteBuffers":
		ba.processDeleteBuffers(call)
	}
}

// processGenBuffers 处理 glGenBuffers/glCreateBuffers 调用
// 格式: glGenBuffers 1 或 glCreateBuffers 1
func (ba *BufferAnalyzer) processGenBuffers(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	// 提取 buffer ID
	var ids []int
	for _, part := range splitParams(call.RawParams) {
		part = trimHexPrefix(part)
		if id := parseHexOrDec(part); id > 0 {
			ids = append(ids, id)
		}
	}

	for _, id := range ids {
		if _, exists := ba.buffers[id]; !exists {
			buf := &core.BufferInfo{
				ID:     id,
				Target: TargetArrayBuffer, // 默认值，后续 glBindBuffer 会更新
			}
			ba.addBuffer(buf)
		}
	}
}

// processBindBuffer 处理 glBindBuffer 调用
// 格式: glBindBuffer 0x8892 498 或 glBindBuffer GL_ARRAY_BUFFER 498
func (ba *BufferAnalyzer) processBindBuffer(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	parts := splitParams(call.RawParams)
	if len(parts) < 2 {
		return
	}

	target := normalizeHexToName(parts[0])
	bufID := parseHexOrDec(parts[1])

	if bufID > 0 {
		if buf, exists := ba.buffers[bufID]; exists {
			buf.Target = target
		}
	}
}

// processBufferData 处理 glBufferData 调用
// 格式: glBufferData 0x8892 8512 0x7fa1ba6970 GL_STATIC_DRAW
func (ba *BufferAnalyzer) processBufferData(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	parts := splitParams(call.RawParams)
	if len(parts) < 3 {
		return
	}

	target := normalizeHexToName(parts[0])
	size := parseHexOrDec(parts[1])
	usage := parts[len(parts)-1] // 最后一项是 usage

	// 更新所有该 target 类型的 buffer 大小
	for _, buf := range ba.buffers {
		if buf.Target == target || targetToHex(target) == parts[0] {
			if size > 0 {
				buf.Size = int64(size)
			}
			if usage != "" {
				buf.Usage = normalizeUsage(usage)
			}
		}
	}
}

// processBufferSubData 处理 glBufferSubData 调用
// 格式: glBufferSubData 0x8892 0 8512 0x7fa1ba6970
func (ba *BufferAnalyzer) processBufferSubData(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	parts := splitParams(call.RawParams)
	if len(parts) < 4 {
		return
	}

	target := normalizeHexToName(parts[0])
	size := parseHexOrDec(parts[2])

	// 更新该 target 类型的 buffer 大小
	for _, buf := range ba.buffers {
		if buf.Target == target || targetToHex(target) == parts[0] {
			if size > 0 && buf.Size == 0 {
				buf.Size = int64(size)
			}
		}
	}
}

// processDeleteBuffers 处理 glDeleteBuffers 调用
func (ba *BufferAnalyzer) processDeleteBuffers(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	for _, part := range splitParams(call.RawParams) {
		part = trimHexPrefix(part)
		if id := parseHexOrDec(part); id > 0 {
			delete(ba.buffers, id)
		}
	}
}

// addBuffer 添加缓冲区
func (ba *BufferAnalyzer) addBuffer(buf *core.BufferInfo) {
	if buf == nil {
		return
	}
	if _, exists := ba.buffers[buf.ID]; !exists {
		ba.buffers[buf.ID] = buf
		ba.byTarget[buf.Target] = append(ba.byTarget[buf.Target], buf)
		ba.totalSize += buf.Size
	}
}

// GetAllBuffers 返回所有缓冲区
func (ba *BufferAnalyzer) GetAllBuffers() []*core.BufferInfo {
	result := make([]*core.BufferInfo, 0, len(ba.buffers))
	for _, buf := range ba.buffers {
		result = append(result, buf)
	}
	return result
}

// GetBuffersByTarget 按目标类型分组返回缓冲区
func (ba *BufferAnalyzer) GetBuffersByTarget() map[string][]*core.BufferInfo {
	return ba.byTarget
}

// GetTotalSize 返回缓冲区总大小
func (ba *BufferAnalyzer) GetTotalSize() int64 {
	return ba.totalSize
}

// GetBufferCount 返回缓冲区数量
func (ba *BufferAnalyzer) GetBufferCount() int {
	return len(ba.buffers)
}

// GetBufferSummary 返回缓冲区统计摘要
func (ba *BufferAnalyzer) GetBufferSummary() map[string]interface{} {
	targetStats := make(map[string]map[string]interface{})
	for target, bufs := range ba.byTarget {
		var totalSize int64
		for _, buf := range bufs {
			totalSize += buf.Size
		}
		targetStats[target] = map[string]interface{}{
			"count":     len(bufs),
			"totalSize": totalSize,
		}
	}

	return map[string]interface{}{
		"totalCount":   len(ba.buffers),
		"totalSize":    ba.totalSize,
		"targetStats":  targetStats,
	}
}

// Helper functions

func splitParams(params string) []string {
	var result []string
	var current string
	inParen := false

	for _, ch := range params {
		switch ch {
		case ' ', '\t', ',':
			if !inParen && current != "" {
				result = append(result, current)
				current = ""
			}
		case '(', '[':
			inParen = true
			current += string(ch)
		case ')', ']':
			inParen = false
			current += string(ch)
		default:
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

func trimHexPrefix(s string) string {
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		return s[2:]
	}
	return s
}

func parseHexOrDec(s string) int {
	var val int
	if _, err := fmt.Sscanf(s, "0x%x", &val); err != nil {
		fmt.Sscanf(s, "%d", &val)
	}
	return val
}

func normalizeHexToName(hex string) string {
	switch hex {
	case "0x8892", "GL_ARRAY_BUFFER":
		return TargetArrayBuffer
	case "0x8D40", "GL_ELEMENT_ARRAY_BUFFER":
		return TargetElementArrayBuffer
	case "0x88B8", "GL_PIXEL_PACK_BUFFER":
		return TargetPixelPackBuffer
	case "0x88B9", "GL_PIXEL_UNPACK_BUFFER":
		return TargetPixelUnpackBuffer
	case "0x8B11", "GL_UNIFORM_BUFFER":
		return TargetUniformBuffer
	case "0x8C8A", "GL_TRANSFORM_FEEDBACK":
		return TargetTransformFeedback
	case "0x8B8F", "GL_COPY_READ_BUFFER":
		return TargetCopyReadBuffer
	case "0x8B8E", "GL_COPY_WRITE_BUFFER":
		return TargetCopyWriteBuffer
	case "0x8F3F", "GL_DRAW_INDIRECT_BUFFER":
		return TargetDrawIndirectBuffer
	case "0x90D2", "GL_SHADER_STORAGE_BUFFER":
		return TargetShaderStorageBuffer
	default:
		return hex
	}
}

func targetToHex(name string) string {
	switch name {
	case TargetArrayBuffer:
		return "0x8892"
	case TargetElementArrayBuffer:
		return "0x8D40"
	case TargetPixelPackBuffer:
		return "0x88B8"
	case TargetPixelUnpackBuffer:
		return "0x88B9"
	case TargetUniformBuffer:
		return "0x8B11"
	case TargetTransformFeedback:
		return "0x8C8A"
	case TargetCopyReadBuffer:
		return "0x8B8F"
	case TargetCopyWriteBuffer:
		return "0x8B8E"
	case TargetDrawIndirectBuffer:
		return "0x8F3F"
	case TargetShaderStorageBuffer:
		return "0x90D2"
	default:
		return name
	}
}

func normalizeUsage(usage string) string {
	switch usage {
	case "GL_STREAM_DRAW", "0x88E0":
		return UsageStreamDraw
	case "GL_STATIC_DRAW", "0x88E4":
		return UsageStaticDraw
	case "GL_DYNAMIC_DRAW", "0x88E8":
		return UsageDynamicDraw
	case "GL_STREAM_READ", "0x88E1":
		return UsageStreamRead
	case "GL_STATIC_READ", "0x88E5":
		return UsageStaticRead
	case "GL_DYNAMIC_READ", "0x88E9":
		return UsageDynamicRead
	case "GL_STREAM_COPY", "0x88E2":
		return UsageStreamCopy
	case "GL_STATIC_COPY", "0x88E6":
		return UsageStaticCopy
	case "GL_DYNAMIC_COPY", "0x88EA":
		return UsageDynamicCopy
	default:
		return usage
	}
}
