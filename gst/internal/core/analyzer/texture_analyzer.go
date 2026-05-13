package analyzer

import (
	"fmt"
	"strings"

	"gst/internal/core"
)

// TextureAnalyzer 纹理生命周期分析器
type TextureAnalyzer struct {
	log      *core.ParsedLog
	textures map[int]*core.TextureInfo
	byTarget map[string]int
}

// NewTextureAnalyzer 创建纹理分析器
func NewTextureAnalyzer(log *core.ParsedLog) *TextureAnalyzer {
	ta := &TextureAnalyzer{
		log:      log,
		textures: make(map[int]*core.TextureInfo),
		byTarget: make(map[string]int),
	}
	ta.analyze()
	return ta
}

func (ta *TextureAnalyzer) analyze() {
	if ta.log == nil {
		return
	}

	for _, frame := range ta.log.Frames {
		for i := range frame.APICalls {
			call := &frame.APICalls[i]
			ta.processAPICall(call, frame.FrameNum)
		}
	}
}

func (ta *TextureAnalyzer) processAPICall(call *core.APILogEntry, frameNum int) {
	switch {
	case strings.HasPrefix(call.APIName, "glGenTextures") || strings.HasPrefix(call.APIName, "glCreateTextures"):
		ta.processGenTextures(call, frameNum)
	case strings.HasPrefix(call.APIName, "glBindTexture"):
		ta.processBindTexture(call)
	case strings.HasPrefix(call.APIName, "glDeleteTextures"):
		ta.processDeleteTextures(call)
	case strings.HasPrefix(call.APIName, "glTexImage2D") || strings.HasPrefix(call.APIName, "glTexImage3D"):
		ta.processTexImage(call)
	case strings.HasPrefix(call.APIName, "glTexStorage2D") || strings.HasPrefix(call.APIName, "glTexStorage3D"):
		ta.processTexStorage(call)
	}
}

func (ta *TextureAnalyzer) processGenTextures(call *core.APILogEntry, frameNum int) {
	if call.RawParams == "" {
		return
	}

	for _, part := range splitParams(call.RawParams) {
		part = trimHexPrefix(part)
		if id := parseHexOrDec(part); id > 0 {
			if _, exists := ta.textures[id]; !exists {
				ta.textures[id] = &core.TextureInfo{
					ID:       id,
					Target:   "GL_TEXTURE_2D",
					Created:  true,
					FrameNum: frameNum,
				}
			}
		}
	}
}

func (ta *TextureAnalyzer) processBindTexture(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	parts := splitParams(call.RawParams)
	if len(parts) < 2 {
		return
	}

	target := normalizeTextureTarget(parts[0])
	textureID := parseHexOrDec(trimHexPrefix(parts[1]))

	if textureID > 0 {
		if tex, exists := ta.textures[textureID]; exists {
			tex.Bound = true
			if target != "" {
				tex.Target = target
			}
		}
		ta.byTarget[target]++
	}
}

func (ta *TextureAnalyzer) processDeleteTextures(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	for _, part := range splitParams(call.RawParams) {
		part = trimHexPrefix(part)
		if id := parseHexOrDec(part); id > 0 {
			if tex, exists := ta.textures[id]; exists {
				tex.Deleted = true
			}
		}
	}
}

func (ta *TextureAnalyzer) processTexImage(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	parts := splitParams(call.RawParams)
	if len(parts) < 3 {
		return
	}

	// glTexImage2D target, level, internalformat, width, height, border, format, type, data
	target := normalizeTextureTarget(parts[0])
	if len(parts) >= 5 {
		width := parseHexOrDec(parts[3])
		height := parseHexOrDec(parts[4])
		format := parts[2]

		// 更新最近绑定的该 target 纹理
		for _, tex := range ta.textures {
			if tex.Target == target && tex.Bound && !tex.Deleted {
				tex.Width = width
				tex.Height = height
				tex.Format = format
				break
			}
		}
	}
}

func (ta *TextureAnalyzer) processTexStorage(call *core.APILogEntry) {
	if call.RawParams == "" {
		return
	}

	parts := splitParams(call.RawParams)
	if len(parts) < 5 {
		return
	}

	// glTexStorage2D target, levels, internalformat, width, height
	target := normalizeTextureTarget(parts[0])
	width := parseHexOrDec(parts[3])
	height := parseHexOrDec(parts[4])
	format := parts[2]

	for _, tex := range ta.textures {
		if tex.Target == target && tex.Bound && !tex.Deleted {
			tex.Width = width
			tex.Height = height
			tex.Format = format
			break
		}
	}
}

// GetSummary 获取纹理统计摘要
func (ta *TextureAnalyzer) GetSummary() *core.TextureSummary {
	var active int
	leaked := make([]core.TextureInfo, 0)

	for _, tex := range ta.textures {
		if !tex.Deleted {
			active++
		}
		if tex.Created && !tex.Deleted {
			leaked = append(leaked, *tex)
		}
	}

	// 限制泄漏列表大小
	if len(leaked) > 20 {
		leaked = leaked[:20]
	}

	return &core.TextureSummary{
		TotalTextures:  len(ta.textures),
		ActiveTextures: active,
		LeakedTextures: leaked,
		ByTarget:       ta.byTarget,
	}
}

// GetLeakedCount 获取泄漏的纹理数量
func (ta *TextureAnalyzer) GetLeakedCount() int {
	count := 0
	for _, tex := range ta.textures {
		if tex.Created && !tex.Deleted {
			count++
		}
	}
	return count
}

// GetByTarget 按目标类型获取纹理数量
func (ta *TextureAnalyzer) GetByTarget() map[string]int {
	result := make(map[string]int)
	for _, tex := range ta.textures {
		result[tex.Target]++
	}
	return result
}

func normalizeTextureTarget(s string) string {
	switch s {
	case "0x0DE1", "GL_TEXTURE_2D":
		return "GL_TEXTURE_2D"
	case "0x0DE0", "GL_TEXTURE_1D":
		return "GL_TEXTURE_1D"
	case "0x806F", "GL_TEXTURE_3D":
		return "GL_TEXTURE_3D"
	case "0x8513", "GL_TEXTURE_CUBE_MAP":
		return "GL_TEXTURE_CUBE_MAP"
	case "0x8D63", "GL_TEXTURE_2D_ARRAY":
		return "GL_TEXTURE_2D_ARRAY"
	case "0x9100", "GL_TEXTURE_2D_MULTISAMPLE":
		return "GL_TEXTURE_2D_MULTISAMPLE"
	case "0x9102", "GL_TEXTURE_2D_MULTISAMPLE_ARRAY":
		return "GL_TEXTURE_2D_MULTISAMPLE_ARRAY"
	case "0x84C0", "GL_TEXTURE_RECTANGLE":
		return "GL_TEXTURE_RECTANGLE"
	case "0x84F5", "GL_TEXTURE_BUFFER":
		return "GL_TEXTURE_BUFFER"
	default:
		if strings.HasPrefix(s, "GL_") {
			return s
		}
		return fmt.Sprintf("UNKNOWN(%s)", s)
	}
}
