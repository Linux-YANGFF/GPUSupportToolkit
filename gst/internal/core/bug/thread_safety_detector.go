package bug

import (
	"fmt"
	"strings"

	"gst/internal/core"
)

type ThreadSafetyDetector struct{}

func NewThreadSafetyDetector() *ThreadSafetyDetector {
	return &ThreadSafetyDetector{}
}

func (d *ThreadSafetyDetector) Diagnose(log *core.ParsedLog) []core.Finding {
	var findings []core.Finding

	var allCalls []*core.APILogEntry
	for i := range log.Frames {
		for j := range log.Frames[i].APICalls {
			allCalls = append(allCalls, &log.Frames[i].APICalls[j])
		}
	}

	if len(allCalls) == 0 {
		return findings
	}

	gcLastTID := make(map[string]string)
	gcAllTIDs := make(map[string]map[string]bool)
	seen := make(map[string]bool)
	switchCount := make(map[string]int)
	reported := make(map[string]bool)

	for _, call := range allCalls {
		gcAddr := call.GCAddr
		tid := call.TID

		if gcAddr == "" || tid == "" {
			continue
		}

		if gcAllTIDs[gcAddr] == nil {
			gcAllTIDs[gcAddr] = make(map[string]bool)
		}
		gcAllTIDs[gcAddr][tid] = true

		if isMakeCurrentCall(call.APIName) && isUnbindMakeCurrent(call.RawParams) {
			delete(gcLastTID, gcAddr)
			continue
		}

		prevTID, hasPrev := gcLastTID[gcAddr]
		if hasPrev && prevTID != tid {
			switchCount[gcAddr]++

			key := gcAddr + ":" + prevTID + ":" + tid
			if !reported[key] {
				reported[key] = true
				findings = append(findings, d.buildSwitchFinding(call, gcAddr, prevTID, tid))
			}
		}

		gcLastTID[gcAddr] = tid
		seen[gcAddr] = true
	}

	for gcAddr, counts := range switchCount {
		if counts >= 3 {
			findings = append(findings, core.Finding{
				Severity: core.SeverityHigh,
				Category: "thread_safety",
				Description: fmt.Sprintf(
					"Rapid thread switching on context %s: %d thread switches detected",
					gcAddr, counts,
				),
				Evidence: fmt.Sprintf(
					"Context %s experienced %d thread switches without proper MakeCurrent unbind, indicating a pattern of thread-unsafe GL usage",
					gcAddr, counts,
				),
				RootCauseChain: []string{
					"Multiple threads frequently alternate on the same GL context",
					"Each switch without unbind risks data races and undefined behavior",
					"GPU drivers are not designed for concurrent multi-threaded context access without explicit synchronization",
				},
				FixSuggestion: "Restructure rendering to use a single dedicated rendering thread, or use separate GL contexts per thread with proper resource sharing via share lists",
			})
		}
	}

	for gcAddr, tids := range gcAllTIDs {
		if len(tids) > 1 && switchCount[gcAddr] == 0 {
			tidList := make([]string, 0, len(tids))
			for tid := range tids {
				tidList = append(tidList, tid)
			}
			findings = append(findings, core.Finding{
				Severity: core.SeverityMedium,
				Category: "thread_safety",
				Description: fmt.Sprintf(
					"Context %s accessed by multiple threads: %s",
					gcAddr, strings.Join(tidList, ", "),
				),
				Evidence: fmt.Sprintf(
					"Context %s has %d distinct TIDs: %s. Context was properly unbound between threads but multi-thread usage still requires careful synchronization",
					gcAddr, len(tids), strings.Join(tidList, ", "),
				),
				RootCauseChain: []string{
					"GL context accessed from multiple threads",
					"Even with proper MakeCurrent unbind, coordinating context sharing across threads adds complexity and risk",
				},
				FixSuggestion: "Consider using a dedicated rendering thread to simplify thread safety, or ensure all thread synchronization is correct",
			})
		}
	}

	return findings
}

func (d *ThreadSafetyDetector) buildSwitchFinding(call *core.APILogEntry, gcAddr, prevTID, tid string) core.Finding {
	desc := fmt.Sprintf(
		"Thread switch without unbind: context %s used by TID %s at line %d (%s), previously used by TID %s",
		gcAddr, tid, call.LineNum, call.APIName, prevTID,
	)
	evidence := fmt.Sprintf(
		"%s at line %d: gc=%s, tid=%s, previous_tid=%s, params=%q",
		call.APIName, call.LineNum, gcAddr, tid, prevTID, call.RawParams,
	)
	return core.Finding{
		Severity: core.SeverityHigh,
		Category: "thread_safety",
		Description:    desc,
		Evidence:       evidence,
		RootCauseChain: []string{
			"Multiple threads accessing the same GL context without proper unbind",
			"OpenGL contexts are not thread-safe: only one thread may use a context at a time",
			"Context was not unbound via glXMakeCurrent(dpy, None, NULL) before another thread accessed it",
		},
		FixSuggestion: "Call glXMakeCurrent(dpy, None, NULL) to unbind the context from the current thread before allowing another thread to make it current. Alternatively, use separate GL contexts per thread with shared resource lists.",
	}
}

func isMakeCurrentCall(apiName string) bool {
	return apiName == "glXMakeCurrent" || apiName == "wglMakeCurrent" || apiName == "eglMakeCurrent"
}

func isUnbindMakeCurrent(params string) bool {
	p := strings.ToLower(params)
	isNull := (strings.Contains(p, "drawable = none") ||
		strings.Contains(p, "drawable = 0") ||
		strings.Contains(p, "drawable=none") ||
		strings.Contains(p, "drawable=0") ||
		strings.Contains(p, "draw=0") ||
		strings.Contains(p, "draw = 0") ||
		strings.Contains(p, "draw=none") ||
		strings.Contains(p, "draw = none") ||
		strings.Contains(p, "gc = (nil)") ||
		strings.Contains(p, "gc=(nil)") ||
		strings.Contains(p, "gc = nil") ||
		strings.Contains(p, "gc=nil") ||
		strings.Contains(p, "ctx = (nil)") ||
		strings.Contains(p, "ctx=(nil)") ||
		strings.Contains(p, "ctx = nil") ||
		strings.Contains(p, "ctx=nil") ||
		strings.Contains(p, "ctx=0") ||
		strings.Contains(p, "ctx = 0"))
	return isNull
}
