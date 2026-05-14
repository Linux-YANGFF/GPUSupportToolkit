package parser

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIndexedRawTraceFile_FrameCostIsSourceOfTruth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hybrid.log")
	log := strings.Join([]string{
		"[    1] (gc=0x1, tid=0x1): glDrawArrays 0x0004 0 3",
		"[    2] glXSwapBuffers: dpy = 0x1, drawable = 1",
		"[    3] swapBuffers: 100 us",
		"[    4] glDrawArrays: count=1, time=99 us",
		"[    5] (gc=0x1, tid=0x1): glUseProgram 7",
		"[    6] (gc=0x1, tid=0x1): glDrawArrays 0x0004 0 3",
		"[    7] glXSwapBuffers: dpy = 0x1, drawable = 1",
		"[    8] swapBuffers: 200 us",
		"[    9] 1 frame cost 16ms",
		"[   10] glDrawArrays: count=1, time=700 us",
		"[   11] glUseProgram: count=1, time=50 us",
		"[   12] (gc=0x1, tid=0x1): glDrawElements 0x0004 6 0x1403 0x0",
		"[   13] glXSwapBuffers: dpy = 0x1, drawable = 1",
		"[   14] swapBuffers: 300 us",
		"[   15] 2 frame cost 20ms",
		"[   16] glDrawElements: count=1, time=900 us",
		"[   17] (gc=0x1, tid=0x1): glDrawArrays 0x0004 0 3",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(log), 0644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseIndexedRawTraceFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Indexed {
		t.Fatal("expected indexed parsed log")
	}
	if got := len(parsed.Frames); got != 2 {
		t.Fatalf("frame count = %d, want 2", got)
	}
	if parsed.Frames[0].FrameNum != 1 || parsed.Frames[1].FrameNum != 2 {
		t.Fatalf("frame numbers = %d,%d; want 1,2", parsed.Frames[0].FrameNum, parsed.Frames[1].FrameNum)
	}
	if got := parsed.Frames[0].APICallCount; got != 2 {
		t.Fatalf("frame 1 api count = %d, want 2", got)
	}
	if got := parsed.Frames[0].APITotalTimeUs; got != 750 {
		t.Fatalf("frame 1 api total = %d, want 750", got)
	}
	if got := parsed.Frames[0].DrawCallCount; got != 1 {
		t.Fatalf("frame 1 draw count = %d, want 1", got)
	}
	if got := parsed.Frames[1].APICallCount; got != 1 {
		t.Fatalf("frame 2 api count = %d, want 1", got)
	}
	program := parsed.Trace.ProgramMap[7]
	if program == nil {
		t.Fatal("program 7 missing from trace analysis")
	}
	if got := program.DrawCallCount; got != 2 {
		t.Fatalf("program 7 draw count = %d, want 2", got)
	}
	if got := program.UseCount; got != 1 {
		t.Fatalf("program 7 use count = %d, want 1", got)
	}
	if got := program.FramesUsed; len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("program 7 frames used = %#v, want [1 2]", got)
	}

	calls, total, err := ParseIndexedFrameAPICalls(path, parsed.Frames[0], 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(calls) != 2 {
		t.Fatalf("api page total=%d len=%d, want 2/2", total, len(calls))
	}
	if calls[0].APIName != "glUseProgram" || calls[1].APIName != "glDrawArrays" {
		t.Fatalf("unexpected api page: %#v", calls)
	}

	lines, total, err := ParseIndexedFrameRawLines(path, parsed.Frames[0], 1, 20, true)
	if err != nil {
		t.Fatal(err)
	}
	if total != 7 || len(lines) != 7 {
		t.Fatalf("raw lines total=%d len=%d, want 7/7: %#v", total, len(lines), lines)
	}
	if lines[0] != "(gc=0x1, tid=0x1): glUseProgram 7" {
		t.Fatalf("first stripped line = %q", lines[0])
	}
	if lines[4] != "1 frame cost 16ms" || lines[6] != "glUseProgram: count=1, time=50 us" {
		t.Fatalf("raw lines did not include frame cost/profile tail: %#v", lines)
	}

	var downloaded bytes.Buffer
	if err := CopyIndexedFrameRawLog(&downloaded, path, parsed.Frames[0]); err != nil {
		t.Fatal(err)
	}
	downloadText := downloaded.String()
	if !strings.Contains(downloadText, "[    5] (gc=0x1, tid=0x1): glUseProgram 7") {
		t.Fatalf("download missing raw frame start: %q", downloadText)
	}
	if !strings.Contains(downloadText, "[   11] glUseProgram: count=1, time=50 us") {
		t.Fatalf("download missing profile tail: %q", downloadText)
	}
	if strings.Contains(downloadText, "[   12] (gc=0x1, tid=0x1): glDrawElements") {
		t.Fatalf("download leaked next frame: %q", downloadText)
	}
}
