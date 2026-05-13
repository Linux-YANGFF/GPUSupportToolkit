package bug

import (
	"testing"

	"gst/internal/core"
)

func TestNewContextManager(t *testing.T) {
	cm := NewContextManager()
	if cm == nil {
		t.Fatal("NewContextManager returned nil")
	}
	if len(cm.GetContexts()) != 0 {
		t.Fatal("new ContextManager should have no contexts")
	}
}

func TestContextManager_ProcessCall_SingleContext(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 123", GCAddr: "0xffff60638d80"},
		{APIName: "glGenBuffers", RawParams: "3", GCAddr: "0xffff60638d80"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xffff60638d80"},
		{APIName: "glBindBuffer", RawParams: "0x8893 498", GCAddr: "0xffff60638d80"},
		{APIName: "glBindTexture", RawParams: "0x0de1 42", GCAddr: "0xffff60638d80"},
		{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xffff60638d80"},
		{APIName: "glBindFramebuffer", RawParams: "0x8d40 5", GCAddr: "0xffff60638d80"},
		{APIName: "glCreateShader", RawParams: "0x8b31 1", GCAddr: "0xffff60638d80"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if len(cm.GetContexts()) != 1 {
		t.Fatalf("expected 1 context, got %d", len(cm.GetContexts()))
	}

	ctx := cm.GetContextInfo("0xffff60638d80")
	if ctx == nil {
		t.Fatal("context not found")
	}

	if !ctx.VBOIDs[199] {
		t.Error("VBO 199 should be tracked")
	}
	if !ctx.VBOIDs[498] {
		t.Error("VBO 498 should be tracked")
	}
	if !ctx.TextureIDs[42] {
		t.Error("Texture 42 should be tracked")
	}
	if !ctx.ProgramIDs[18] {
		t.Error("Program 18 should be tracked")
	}
	if !ctx.FBOIDs[5] {
		t.Error("FBO 5 should be tracked")
	}
	if !ctx.ShaderIDs[1] {
		t.Error("Shader 1 should be tracked")
	}
}

func TestContextManager_MultipleContexts(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 123", GCAddr: "0xgcA"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
		{APIName: "glBindTexture", RawParams: "0x0de1 42", GCAddr: "0xgcA"},
		{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 456", GCAddr: "0xgcB"},
		{APIName: "glBindBuffer", RawParams: "0x8892 501", GCAddr: "0xgcB"},
		{APIName: "glBindTexture", RawParams: "0x0de1 99", GCAddr: "0xgcB"},
		{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 789", GCAddr: "0xgcC"},
		{APIName: "glBindBuffer", RawParams: "0x8892 777", GCAddr: "0xgcC"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if len(cm.GetContexts()) != 3 {
		t.Fatalf("expected 3 contexts, got %d", len(cm.GetContexts()))
	}

	ctxA := cm.GetContextInfo("0xgcA")
	if ctxA == nil {
		t.Fatal("context A not found")
	}
	if !ctxA.VBOIDs[199] || !ctxA.TextureIDs[42] {
		t.Error("context A resources not tracked correctly")
	}

	ctxB := cm.GetContextInfo("0xgcB")
	if ctxB == nil {
		t.Fatal("context B not found")
	}
	if !ctxB.VBOIDs[501] || !ctxB.TextureIDs[99] {
		t.Error("context B resources not tracked correctly")
	}

	ctxC := cm.GetContextInfo("0xgcC")
	if ctxC == nil {
		t.Fatal("context C not found")
	}
	if !ctxC.VBOIDs[777] {
		t.Error("context C resources not tracked correctly")
	}

	if ctxA.VBOIDs[501] {
		t.Error("context A should not have VBO 501 (belongs to context B)")
	}
}

func TestContextManager_ShareListNil(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 0", GCAddr: "0xgcNoShare"},
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 1", GCAddr: "0xgcWithShare"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if !cm.HasShareListNil("0xgcNoShare") {
		t.Error("0xgcNoShare should have shareList=nil")
	}
	if cm.HasShareListNil("0xgcWithShare") {
		t.Error("0xgcWithShare should NOT have shareList=nil")
	}
	if cm.HasShareListNil("0xnonexistent") {
		t.Error("nonexistent context should not have shareList=nil")
	}
}

func TestContextManager_IsResourceShared(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 0", GCAddr: "0xgcNoShare"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcNoShare"},
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 1", GCAddr: "0xgcShare"},
		{APIName: "glBindBuffer", RawParams: "0x8892 501", GCAddr: "0xgcShare"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if cm.IsResourceShared(199, "0xgcNoShare", "0xgcB") {
		t.Error("resource should NOT be shared when shareList=(nil)")
	}

	if !cm.IsResourceShared(501, "0xgcShare", "0xgcB") {
		t.Error("resource should be shared when shareList=1")
	}

	if cm.IsResourceShared(999, "0xgcShare", "0xgcB") {
		t.Error("nonexistent resource should not be shared")
	}

	if cm.IsResourceShared(199, "0xnonexistent", "0xgcB") {
		t.Error("nonexistent context should not share resources")
	}
}

func TestContextManager_IsResourceOwnedByContext(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
		{APIName: "glBindTexture", RawParams: "0x0de1 42", GCAddr: "0xgcA"},
		{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if !cm.IsResourceOwnedByContext(199, "0xgcA") {
		t.Error("VBO 199 should be owned by 0xgcA")
	}
	if !cm.IsResourceOwnedByContext(42, "0xgcA") {
		t.Error("Texture 42 should be owned by 0xgcA")
	}
	if !cm.IsResourceOwnedByContext(18, "0xgcA") {
		t.Error("Program 18 should be owned by 0xgcA")
	}
	if cm.IsResourceOwnedByContext(199, "0xgcB") {
		t.Error("VBO 199 should NOT be owned by 0xgcB")
	}
	if cm.IsResourceOwnedByContext(199, "0xnonexistent") {
		t.Error("nonexistent context should not own any resource")
	}
}

func TestContextManager_DetectCrossContextUsage(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 0", GCAddr: "0xgcA"},
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 0", GCAddr: "0xgcB"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
		{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	finding := cm.DetectCrossContextUsage(199, "0xgcA")
	if finding != nil {
		t.Error("same context usage should not trigger cross-context detection")
	}

	finding = cm.DetectCrossContextUsage(199, "0xgcB")
	if finding == nil {
		t.Fatal("cross-context usage should be detected")
	}
	if finding.Severity != core.SeverityCritical {
		t.Error("cross-context usage should be critical severity")
	}
	if finding.Category != "cross_context_resource" {
		t.Error("cross-context category mismatch")
	}

	finding = cm.DetectCrossContextUsage(999, "0xgcB")
	if finding != nil {
		t.Error("unowned resource should not trigger detection")
	}
}

func TestContextManager_DetectCrossContextUsage_ResourceOwned(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	finding := cm.DetectCrossContextUsage(199, "0xgcA")
	if finding != nil {
		t.Error("same context resource should not be flagged")
	}
}

func TestContextManager_ShaderResourceTracking(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glCreateShader", RawParams: "0x8b31 5", GCAddr: "0xgcA"},
		{APIName: "glShaderSource", RawParams: "5", GCAddr: "0xgcA"},
		{APIName: "glCompileShader", RawParams: "5", GCAddr: "0xgcA"},
		{APIName: "glCreateShader", RawParams: "0x8b30 7", GCAddr: "0xgcA"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	ctx := cm.GetContextInfo("0xgcA")
	if !ctx.ShaderIDs[5] {
		t.Error("Shader 5 should be tracked")
	}
	if !ctx.ShaderIDs[7] {
		t.Error("Shader 7 should be tracked")
	}
}

func TestContextManager_ProgramResourceTracking(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glCreateProgram", RawParams: "", GCAddr: "0xgcA"},
		{APIName: "glUseProgram", RawParams: "22", GCAddr: "0xgcA"},
		{APIName: "glUseProgram", RawParams: "18", GCAddr: "0xgcA"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	ctx := cm.GetContextInfo("0xgcA")
	if !ctx.ProgramIDs[22] {
		t.Error("Program 22 should be tracked")
	}
	if !ctx.ProgramIDs[18] {
		t.Error("Program 18 should be tracked")
	}
}

func TestContextManager_CurrentGCTracking(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 123", GCAddr: "0xgcA"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xgcA"},
		{APIName: "glXMakeCurrent", RawParams: "dpy = 0x1c00, drawable = 456", GCAddr: "0xgcB"},
		{APIName: "glBindBuffer", RawParams: "0x8892 501", GCAddr: "0xgcB"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if cm.CurrentGC != "0xgcB" {
		t.Errorf("expected current GC to be 0xgcB, got %s", cm.CurrentGC)
	}
}

func TestContextManager_EmptyGCAddrIgnored(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: ""},
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, share_list = 0", GCAddr: ""},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if len(cm.GetContexts()) != 0 {
		t.Error("calls with empty GCAddr should not create contexts")
	}
}

func TestContextManager_MultiContextSIGSEGVScenario(t *testing.T) {
	cm := NewContextManager()

	calls := []core.APILogEntry{
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 0", GCAddr: "0xfffe6985a840"},
		{APIName: "glGenBuffers", RawParams: "3", GCAddr: "0xfffe6985a840"},
		{APIName: "glBindBuffer", RawParams: "0x8892 199", GCAddr: "0xfffe6985a840"},
		{APIName: "glVertexAttribPointer", RawParams: "0 3 0x1406 0x0 ptr=(nil)", GCAddr: "0xfffe6985a840"},
		{APIName: "glXCreateContextAttribsARB", RawParams: "dpy = 0x1c00, config = 0x8b, share_list = 0", GCAddr: "0xffff7079a900"},
	}

	for _, call := range calls {
		cm.ProcessCall(&call, nil)
	}

	if !cm.HasShareListNil("0xfffe6985a840") {
		t.Error("first context should have shareList=(nil)")
	}
	if !cm.HasShareListNil("0xffff7079a900") {
		t.Error("second context should have shareList=(nil)")
	}

	if cm.IsResourceShared(199, "0xfffe6985a840", "0xffff7079a900") {
		t.Error("VBO 199 should NOT be shared between nil-shareList contexts")
	}

	finding := cm.DetectCrossContextUsage(199, "0xffff7079a900")
	if finding == nil {
		t.Error("cross-context usage of VBO 199 should be detected")
	}
}

func TestExtractDecimalIDs(t *testing.T) {
	tests := []struct {
		params string
		want   []int
	}{
		{"0x8892 199", []int{199}},
		{"0x0de1 42", []int{42}},
		{"18", []int{18}},
		{"0x8d40 5", []int{5}},
		{"0 3 0x1406 0x0", []int{3}}, // skip GL constants as hex-like
		{"0x8b31 7", []int{7}},
		{"1 2 3", []int{1, 2, 3}},
		{"dpy = 0x1c00, config = 0x8b, share_list = 0", []int(nil)}, // 0 is skipped
		{"", nil},
	}

	for _, tt := range tests {
		got := extractDecimalIDs(tt.params)
		if len(got) != len(tt.want) {
			t.Errorf("extractDecimalIDs(%q) = %v, want %v", tt.params, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractDecimalIDs(%q)[%d] = %d, want %d", tt.params, i, got[i], tt.want[i])
			}
		}
	}
}
