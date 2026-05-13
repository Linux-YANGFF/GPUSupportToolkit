package bug

import (
	"regexp"
	"strconv"
	"strings"

	"gst/internal/core"
)

var (
	shareListRe     = regexp.MustCompile(`share_list\s*=\s*(\S+)`)
	decimalNumRe    = regexp.MustCompile(`\b([1-9]\d*)\b`)
	hexOrConstRe    = regexp.MustCompile(`^0x[0-9a-fA-F]+$`)
)

type ContextInfo struct {
	GCAddr     string
	ShareList  string
	VBOIDs     map[int]bool
	TextureIDs map[int]bool
	ShaderIDs  map[int]bool
	ProgramIDs map[int]bool
	FBOIDs     map[int]bool
}

type ContextManager struct {
	contexts  map[string]*ContextInfo
	CurrentGC string
}

func NewContextManager() *ContextManager {
	return &ContextManager{
		contexts: make(map[string]*ContextInfo),
	}
}

func (cm *ContextManager) getOrCreateContext(gcAddr string) *ContextInfo {
	if ctx, ok := cm.contexts[gcAddr]; ok {
		return ctx
	}
	ctx := &ContextInfo{
		GCAddr:     gcAddr,
		VBOIDs:     make(map[int]bool),
		TextureIDs: make(map[int]bool),
		ShaderIDs:  make(map[int]bool),
		ProgramIDs: make(map[int]bool),
		FBOIDs:     make(map[int]bool),
	}
	cm.contexts[gcAddr] = ctx
	return ctx
}

func (cm *ContextManager) ProcessCall(call *core.APILogEntry, stateTracker interface{}) {
	gcAddr := call.GCAddr
	if gcAddr == "" {
		return
	}

	ctx := cm.getOrCreateContext(gcAddr)
	cm.CurrentGC = gcAddr

	apiName := call.APIName
	params := strings.TrimSpace(call.RawParams)

	switch apiName {
	case "glXMakeCurrent":
		_ = ctx
		_ = params
	case "glXCreateContextAttribsARB":
		cm.parseShareList(ctx, params)
	case "glBindBuffer":
		cm.recordID(ctx.VBOIDs, params)
	case "glGenBuffers", "glCreateBuffers":
		cm.recordID(ctx.VBOIDs, params)
	case "glBindTexture":
		cm.recordID(ctx.TextureIDs, params)
	case "glGenTextures", "glCreateTextures":
		cm.recordID(ctx.TextureIDs, params)
	case "glCreateShader", "glShaderSource", "glCompileShader":
		cm.recordID(ctx.ShaderIDs, params)
	case "glCreateProgram", "glUseProgram":
		cm.recordID(ctx.ProgramIDs, params)
	case "glBindFramebuffer":
		cm.recordID(ctx.FBOIDs, params)
	case "glGenFramebuffers", "glCreateFramebuffers":
		cm.recordID(ctx.FBOIDs, params)
	}
}

func (cm *ContextManager) parseShareList(ctx *ContextInfo, params string) {
	if matches := shareListRe.FindStringSubmatch(params); len(matches) > 1 {
		ctx.ShareList = matches[1]
	}
}

func (cm *ContextManager) recordID(idMap map[int]bool, params string) {
	for _, id := range extractDecimalIDs(params) {
		idMap[id] = true
	}
}

func extractDecimalIDs(params string) []int {
	var ids []int
	matches := decimalNumRe.FindAllStringSubmatch(params, -1)
	seen := make(map[int]bool)
	for _, m := range matches {
		val := m[1]
		if hexOrConstRe.MatchString(val) {
			continue
		}
		if id, err := strconv.Atoi(val); err == nil && id > 0 {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (cm *ContextManager) IsResourceShared(resourceID int, fromGC, toGC string) bool {
	fromCtx := cm.contexts[fromGC]
	if fromCtx == nil {
		return false
	}

	nonShared := fromCtx.ShareList == "(nil)" || fromCtx.ShareList == "0" || fromCtx.ShareList == "0x0"
	if nonShared {
		return false
	}

	_ = toGC
	return fromCtx.VBOIDs[resourceID] ||
		fromCtx.TextureIDs[resourceID] ||
		fromCtx.ShaderIDs[resourceID] ||
		fromCtx.ProgramIDs[resourceID] ||
		fromCtx.FBOIDs[resourceID]
}

func (cm *ContextManager) IsResourceOwnedByContext(resourceID int, gcAddr string) bool {
	ctx := cm.contexts[gcAddr]
	if ctx == nil {
		return false
	}
	return ctx.VBOIDs[resourceID] ||
		ctx.TextureIDs[resourceID] ||
		ctx.ShaderIDs[resourceID] ||
		ctx.ProgramIDs[resourceID] ||
		ctx.FBOIDs[resourceID]
}

func (cm *ContextManager) HasShareListNil(gcAddr string) bool {
	ctx := cm.contexts[gcAddr]
	if ctx == nil {
		return false
	}
	return ctx.ShareList == "(nil)" || ctx.ShareList == "0" || ctx.ShareList == "0x0"
}

func (cm *ContextManager) GetContexts() []string {
	var gcAddrs []string
	for gc := range cm.contexts {
		gcAddrs = append(gcAddrs, gc)
	}
	return gcAddrs
}

func (cm *ContextManager) GetContextInfo(gcAddr string) *ContextInfo {
	return cm.contexts[gcAddr]
}

func (cm *ContextManager) DetectCrossContextUsage(resourceID int, usingGC string) *core.Finding {
	usingCtx := cm.contexts[usingGC]
	if usingCtx == nil {
		return nil
	}

	if usingCtx.VBOIDs[resourceID] || usingCtx.TextureIDs[resourceID] ||
		usingCtx.ShaderIDs[resourceID] || usingCtx.ProgramIDs[resourceID] ||
		usingCtx.FBOIDs[resourceID] {
		return nil
	}

	for gcAddr, ctx := range cm.contexts {
		if gcAddr == usingGC {
			continue
		}
		if ctx.VBOIDs[resourceID] || ctx.TextureIDs[resourceID] ||
			ctx.ShaderIDs[resourceID] || ctx.ProgramIDs[resourceID] ||
			ctx.FBOIDs[resourceID] {
			evidence := "资源在 Context " + gcAddr + " 中创建，但在 Context " + usingGC + " 中使用"
			return &core.Finding{
				Severity:       core.SeverityCritical,
				Category:       "cross_context_resource",
				Description:    "跨 Context 资源使用：" + evidence,
				Evidence:       evidence,
				RootCauseChain: []string{"Context 间 shareList=(nil)，资源不共享"},
				FixSuggestion:  "确保资源在正确的 Context 中使用，或使用 shareList 共享资源 Context",
			}
		}
	}
	return nil
}
