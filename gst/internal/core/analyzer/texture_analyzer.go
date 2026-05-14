package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"gst/internal/core"
)

// TextureAnalyzer 纹理生命周期分析器
type TextureAnalyzer struct {
	log             *core.ParsedLog
	textures        map[int]*core.TextureInfo
	boundByTarget   map[string]int
	summaryOverride *core.TextureSummary
}

// NewTextureAnalyzer 创建纹理分析器
func NewTextureAnalyzer(log *core.ParsedLog) *TextureAnalyzer {
	ta := &TextureAnalyzer{
		log:           log,
		textures:      make(map[int]*core.TextureInfo),
		boundByTarget: make(map[string]int),
	}
	ta.analyze()
	return ta
}

func (ta *TextureAnalyzer) analyze() {
	if ta.log == nil {
		return
	}

	processedCalls := 0
	for _, frame := range ta.log.Frames {
		for i := range frame.APICalls {
			call := &frame.APICalls[i]
			ta.processAPICall(call, frame.FrameNum)
			processedCalls++
		}
	}
	if processedCalls == 0 && ta.log.Indexed {
		ta.analyzeIndexedSummaries()
	}
}

func (ta *TextureAnalyzer) analyzeIndexedSummaries() {
	generated := 0
	deleted := 0
	binds := 0
	uploads := 0
	byTarget := make(map[string]int)

	for _, frame := range ta.log.Frames {
		for apiName, summary := range frame.APISummary {
			name := apiName
			if summary != nil && summary.APIName != "" {
				name = summary.APIName
			}
			count := 1
			if summary != nil && summary.Count > 0 {
				count = summary.Count
			}

			switch {
			case strings.HasPrefix(name, "glGenTextures") || strings.HasPrefix(name, "glCreateTextures"):
				generated += count
			case strings.HasPrefix(name, "glDeleteTextures"):
				deleted += count
			case strings.HasPrefix(name, "glBindTexture"):
				binds += count
			}
			if target := textureTargetFromAPIName(name); target != "" {
				uploads += count
				byTarget[target] += count
			}
		}
	}

	totalActivity := generated + deleted + binds + uploads
	if totalActivity == 0 {
		return
	}

	total := generated
	inferred := 0
	if total == 0 {
		total = uploads
		if binds > total {
			total = binds
		}
		inferred = total
	}
	active := total - deleted
	if active < 0 {
		active = 0
	}
	if len(byTarget) == 0 && binds > 0 {
		byTarget["UNKNOWN_TEXTURE_TARGET"] = binds
	}

	ta.summaryOverride = &core.TextureSummary{
		TotalTextures:  total,
		ActiveTextures: active,
		InferredCount:  inferred,
		LeakedTextures: []core.TextureInfo{},
		ByTarget:       byTarget,
	}
}

func (ta *TextureAnalyzer) processAPICall(call *core.APILogEntry, frameNum int) {
	switch {
	case strings.HasPrefix(call.APIName, "glGenTextures") || strings.HasPrefix(call.APIName, "glCreateTextures"):
		ta.processGenTextures(call, frameNum)
	case strings.HasPrefix(call.APIName, "glBindTexture"):
		ta.processBindTexture(call, frameNum)
	case strings.HasPrefix(call.APIName, "glDeleteTextures"):
		ta.processDeleteTextures(call)
	case strings.HasPrefix(call.APIName, "glTexImage2D") || strings.HasPrefix(call.APIName, "glTexImage3D"):
		ta.processTexImage(call)
	case strings.HasPrefix(call.APIName, "glTexStorage2D") || strings.HasPrefix(call.APIName, "glTexStorage3D"):
		ta.processTexStorage(call)
	}
}

func (ta *TextureAnalyzer) processGenTextures(call *core.APILogEntry, frameNum int) {
	ids := textureIDsFromReturn(call.ReturnValue)
	if len(ids) == 0 {
		return
	}

	defaultTarget := "GL_TEXTURE_2D"
	parts := splitParams(call.RawParams)
	if strings.HasPrefix(call.APIName, "glCreateTextures") && len(parts) > 0 {
		defaultTarget = normalizeTextureTarget(parts[0])
	}

	for _, id := range ids {
		if _, exists := ta.textures[id]; !exists {
			ta.textures[id] = &core.TextureInfo{
				ID:         id,
				Target:     defaultTarget,
				Created:    true,
				FrameNum:   frameNum,
				CreateLine: call.LineNum,
				Source:     "generated",
			}
		}
	}
}

func (ta *TextureAnalyzer) processBindTexture(call *core.APILogEntry, frameNum int) {
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
		tex, exists := ta.textures[textureID]
		if !exists {
			tex = &core.TextureInfo{
				ID:       textureID,
				Target:   target,
				Created:  false,
				FrameNum: frameNum,
				Source:   "inferred",
			}
			ta.textures[textureID] = tex
		}
		tex.Bound = true
		tex.BindCount++
		tex.LastBindLine = call.LineNum
		if target != "" {
			tex.Target = target
		}
		ta.boundByTarget[target] = textureID
		return
	}
	delete(ta.boundByTarget, target)
}

func (ta *TextureAnalyzer) processDeleteTextures(call *core.APILogEntry) {
	if call.RawParams == "" && call.ReturnValue == "" {
		return
	}

	for _, id := range textureIDsFromCallOrReturn(call.RawParams, call.ReturnValue) {
		if tex, exists := ta.textures[id]; exists {
			tex.Deleted = true
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
		if tex := ta.boundTexture(target); tex != nil {
			tex.Width = width
			tex.Height = height
			tex.Format = normalizeTextureFormat(format)
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

	if tex := ta.boundTexture(target); tex != nil {
		tex.Width = width
		tex.Height = height
		tex.Format = normalizeTextureFormat(format)
	}
}

// GetSummary 获取纹理统计摘要
func (ta *TextureAnalyzer) GetSummary() *core.TextureSummary {
	if ta.summaryOverride != nil {
		return ta.summaryOverride
	}

	var active int
	var inferred int
	leaked := make([]core.TextureInfo, 0)
	byTarget := make(map[string]int)

	for _, tex := range ta.textures {
		if !tex.Deleted {
			active++
		}
		if tex.Source == "inferred" {
			inferred++
		}
		byTarget[tex.Target]++
		if tex.Created && !tex.Deleted {
			leaked = append(leaked, *tex)
		}
	}

	sort.Slice(leaked, func(i, j int) bool {
		if leaked[i].FrameNum == leaked[j].FrameNum {
			return leaked[i].ID < leaked[j].ID
		}
		return leaked[i].FrameNum < leaked[j].FrameNum
	})

	// 限制泄漏列表大小
	if len(leaked) > 20 {
		leaked = leaked[:20]
	}

	return &core.TextureSummary{
		TotalTextures:  len(ta.textures),
		ActiveTextures: active,
		InferredCount:  inferred,
		LeakedTextures: leaked,
		ByTarget:       byTarget,
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

func (ta *TextureAnalyzer) boundTexture(target string) *core.TextureInfo {
	if id, ok := ta.boundByTarget[target]; ok && id > 0 {
		return ta.textures[id]
	}
	if strings.HasPrefix(target, "GL_TEXTURE_CUBE_MAP_") {
		if id, ok := ta.boundByTarget["GL_TEXTURE_CUBE_MAP"]; ok && id > 0 {
			return ta.textures[id]
		}
	}
	return nil
}

func textureIDsFromReturn(value string) []int {
	return textureIDsFromCallOrReturn("", value)
}

func textureIDsFromCallOrReturn(params string, returnValue string) []int {
	source := strings.TrimSpace(returnValue)
	if source == "" {
		source = params
	}
	parts := splitParams(strings.NewReplacer("{", " ", "}", " ", "\n", " ", "\r", " ").Replace(source))
	ids := make([]int, 0, len(parts))
	for _, part := range parts {
		id := parseHexOrDec(strings.Trim(part, ","))
		if id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func normalizeTextureTarget(s string) string {
	key := strings.ToUpper(strings.TrimSpace(s))
	switch key {
	case "0X0DE1", "GL_TEXTURE_2D":
		return "GL_TEXTURE_2D"
	case "0X0DE0", "GL_TEXTURE_1D":
		return "GL_TEXTURE_1D"
	case "0X806F", "GL_TEXTURE_3D":
		return "GL_TEXTURE_3D"
	case "0X8513", "GL_TEXTURE_CUBE_MAP":
		return "GL_TEXTURE_CUBE_MAP"
	case "0X8515", "GL_TEXTURE_CUBE_MAP_POSITIVE_X",
		"0X8516", "GL_TEXTURE_CUBE_MAP_NEGATIVE_X",
		"0X8517", "GL_TEXTURE_CUBE_MAP_POSITIVE_Y",
		"0X8518", "GL_TEXTURE_CUBE_MAP_NEGATIVE_Y",
		"0X8519", "GL_TEXTURE_CUBE_MAP_POSITIVE_Z",
		"0X851A", "GL_TEXTURE_CUBE_MAP_NEGATIVE_Z":
		return "GL_TEXTURE_CUBE_MAP_FACE"
	case "0X8D63", "GL_TEXTURE_2D_ARRAY":
		return "GL_TEXTURE_2D_ARRAY"
	case "0X9100", "GL_TEXTURE_2D_MULTISAMPLE":
		return "GL_TEXTURE_2D_MULTISAMPLE"
	case "0X9102", "GL_TEXTURE_2D_MULTISAMPLE_ARRAY":
		return "GL_TEXTURE_2D_MULTISAMPLE_ARRAY"
	case "0X84F5", "GL_TEXTURE_RECTANGLE":
		return "GL_TEXTURE_RECTANGLE"
	case "0X8C2A", "GL_TEXTURE_BUFFER":
		return "GL_TEXTURE_BUFFER"
	default:
		if strings.HasPrefix(key, "GL_") {
			return key
		}
		return fmt.Sprintf("UNKNOWN(%s)", strings.TrimSpace(s))
	}
}

func normalizeTextureFormat(s string) string {
	key := strings.ToUpper(strings.TrimSpace(s))
	switch key {
	case "0X1903", "GL_RED":
		return "GL_RED"
	case "0X1907", "GL_RGB":
		return "GL_RGB"
	case "0X1908", "GL_RGBA":
		return "GL_RGBA"
	case "0X8814", "GL_RGBA32F":
		return "GL_RGBA32F"
	case "0X8815", "GL_RGB32F":
		return "GL_RGB32F"
	case "0X8058", "GL_RGBA8":
		return "GL_RGBA8"
	default:
		if strings.HasPrefix(key, "GL_") {
			return key
		}
		return s
	}
}

func textureTargetFromAPIName(apiName string) string {
	switch {
	case strings.Contains(apiName, "TexImage3D") ||
		strings.Contains(apiName, "TexSubImage3D") ||
		strings.Contains(apiName, "TexStorage3D") ||
		strings.Contains(apiName, "CompressedTexImage3D") ||
		strings.Contains(apiName, "CopyTexSubImage3D"):
		return "GL_TEXTURE_3D"
	case strings.Contains(apiName, "TexImage1D") ||
		strings.Contains(apiName, "TexSubImage1D") ||
		strings.Contains(apiName, "TexStorage1D") ||
		strings.Contains(apiName, "CompressedTexImage1D") ||
		strings.Contains(apiName, "CopyTexImage1D") ||
		strings.Contains(apiName, "CopyTexSubImage1D"):
		return "GL_TEXTURE_1D"
	case strings.Contains(apiName, "TexImage2D") ||
		strings.Contains(apiName, "TexSubImage2D") ||
		strings.Contains(apiName, "TexStorage2D") ||
		strings.Contains(apiName, "CompressedTexImage2D") ||
		strings.Contains(apiName, "CopyTexImage2D") ||
		strings.Contains(apiName, "CopyTexSubImage2D"):
		return "GL_TEXTURE_2D"
	default:
		return ""
	}
}
