package parser

import (
	"os"
	"strings"
	"testing"

	"gst/internal/core"
)

func TestRawTraceParser_EnhancedParse(t *testing.T) {
	data, err := os.ReadFile("../bug/testdata/sample_trace.log")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) < 2 {
		t.Fatalf("Expected at least 2 frames, got %d", len(parsed.Frames))
	}

	allCalls := make([]core.APILogEntry, 0)
	for _, frame := range parsed.Frames {
		allCalls = append(allCalls, frame.APICalls...)
	}

	var gcAddrFound, tidFound int
	var glSetErrorFound, segfaultFound int
	var nilPtrFound int
	var makeCurrentFound, createContextFound int

	for _, call := range allCalls {
		if call.GCAddr != "" {
			gcAddrFound++
		}
		if call.TID != "" {
			tidFound++
		}
		if call.APIName == "__glSetError" && call.IsError {
			glSetErrorFound++
			if call.ErrorCode == "" {
				t.Error("__glSetError entry missing ErrorCode")
			}
		}
		if call.APIName == "__segfault__" && call.IsError && call.ErrorCode == "SIGSEGV" {
			segfaultFound++
		}
		if call.HasNilPtr {
			nilPtrFound++
		}
		if call.APIName == "glXMakeCurrent" {
			makeCurrentFound++
		}
		if call.APIName == "glXCreateContextAttribsARB" {
			createContextFound++
		}
	}

	if gcAddrFound == 0 {
		t.Error("No entries with GCAddr populated")
	}
	if tidFound == 0 {
		t.Error("No entries with TID populated")
	}
	if glSetErrorFound < 2 {
		t.Errorf("Expected at least 2 __glSetError entries, got %d", glSetErrorFound)
	}
	if segfaultFound < 1 {
		t.Error("No __segfault__ entry found")
	}
	if nilPtrFound < 3 {
		t.Errorf("Expected at least 3 entries with HasNilPtr=true, got %d", nilPtrFound)
	}
	if makeCurrentFound < 2 {
		t.Errorf("Expected at least 2 glXMakeCurrent calls, got %d", makeCurrentFound)
	}
	if createContextFound < 1 {
		t.Errorf("Expected at least 1 glXCreateContextAttribsARB call, got %d", createContextFound)
	}

	t.Logf("gcAddr=%d tid=%d glSetError=%d segfault=%d nilPtr=%d makeCurrent=%d createContext=%d",
		gcAddrFound, tidFound, glSetErrorFound, segfaultFound, nilPtrFound, makeCurrentFound, createContextFound)
}

func TestRawTraceParser_InlineGCTID(t *testing.T) {
	input := `(gc=0xdeadbeef01, tid=0xabcd1234): glBindBuffer 0x8892 498
(gc=0xdeadbeef02, tid=0xabcd5678): glVertexAttribPointer 0 3 0x1406 0x0 ptr=(nil)
glDrawElements 0x0004 2304 0x1403 ptr=(nil)
glXSwapBuffers: dpy = 0x1c00, drawable = 123`

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(parsed.Frames))
	}

	frame := parsed.Frames[0]
	if len(frame.APICalls) != 3 {
		t.Fatalf("Expected 3 API calls, got %d", len(frame.APICalls))
	}

	call1 := frame.APICalls[0]
	if call1.APIName != "glBindBuffer" {
		t.Errorf("Expected glBindBuffer, got %s", call1.APIName)
	}
	if call1.GCAddr != "0xdeadbeef01" {
		t.Errorf("Expected GCAddr 0xdeadbeef01, got %s", call1.GCAddr)
	}
	if call1.TID != "0xabcd1234" {
		t.Errorf("Expected TID 0xabcd1234, got %s", call1.TID)
	}
	if call1.HasNilPtr {
		t.Error("glBindBuffer line should not have HasNilPtr")
	}

	call2 := frame.APICalls[1]
	if !call2.HasNilPtr {
		t.Error("glVertexAttribPointer with ptr=(nil) should have HasNilPtr=true")
	}
	if call2.GCAddr != "0xdeadbeef02" {
		t.Errorf("Expected GCAddr 0xdeadbeef02, got %s", call2.GCAddr)
	}

	call3 := frame.APICalls[2]
	if call3.GCAddr != "" {
		t.Error("Line without gc/tid prefix should have empty GCAddr")
	}
	if !call3.HasNilPtr {
		t.Error("glDrawElements with ptr=(nil) should have HasNilPtr=true")
	}
}

func TestRawTraceParser_ErrorDetection(t *testing.T) {
	input := `(gc=0x1, tid=0x2): glClear 0x4100
ERROR!!! __glSetError (gl_error=0x0502, errno=0, msg=(nil))
(gc=0x1, tid=0x2): glFlush
段错误 (SIGSEGV) (core dumped)`

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(parsed.Frames))
	}

	frame := parsed.Frames[0]
	if len(frame.APICalls) != 4 {
		t.Fatalf("Expected 4 API calls, got %d: %v", len(frame.APICalls), frame.APICalls)
	}

	setErrorEntry := frame.APICalls[1]
	if setErrorEntry.APIName != "__glSetError" {
		t.Errorf("Expected __glSetError, got %s", setErrorEntry.APIName)
	}
	if !setErrorEntry.IsError {
		t.Error("__glSetError entry should have IsError=true")
	}
	if setErrorEntry.ErrorCode != "gl_error=0x0502" {
		t.Errorf("Expected ErrorCode gl_error=0x0502, got %s", setErrorEntry.ErrorCode)
	}

	segfaultEntry := frame.APICalls[3]
	if segfaultEntry.APIName != "__segfault__" {
		t.Errorf("Expected __segfault__, got %s", segfaultEntry.APIName)
	}
	if !segfaultEntry.IsError {
		t.Error("__segfault__ entry should have IsError=true")
	}
	if segfaultEntry.ErrorCode != "SIGSEGV" {
		t.Errorf("Expected ErrorCode SIGSEGV, got %s", segfaultEntry.ErrorCode)
	}
}

func TestRawTraceParser_ReturnValuesAndShaderSource(t *testing.T) {
	input := `[ 100] (gc=0x1, tid=0x2): glCreateShader 0x8B31
[ 101]         glCreateShader => 16
[ 102] (gc=0x1, tid=0x2): glShaderSource 16 1 0xffff (nil)
[ 103] ####
[ 104] #version 400
[ 105] void main(){}
[ 106] ####
[ 107] (gc=0x1, tid=0x2): glCreateProgram
[ 108]         glCreateProgram => 18
[ 109] (gc=0x1, tid=0x2): glAttachShader 18 16
[ 110] (gc=0x1, tid=0x2): glUseProgram 18
[ 111] (gc=0x1, tid=0x2): glDrawArrays 0x0004 0 3
[ 112] glXSwapBuffers: dpy = 0x1, drawable = 2`

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(parsed.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(parsed.Frames))
	}

	frame := parsed.Frames[0]
	if len(frame.Shaders) != 1 {
		t.Fatalf("Expected 1 shader source, got %d", len(frame.Shaders))
	}
	if frame.Shaders[0].ID != 16 {
		t.Fatalf("Shader ID = %d, want 16", frame.Shaders[0].ID)
	}
	if !strings.Contains(frame.Shaders[0].Source, "void main") {
		t.Fatalf("Shader source not captured: %q", frame.Shaders[0].Source)
	}

	var shaderReturn, programReturn string
	for _, call := range frame.APICalls {
		if call.APIName == "glCreateShader" {
			shaderReturn = call.ReturnValue
		}
		if call.APIName == "glCreateProgram" {
			programReturn = call.ReturnValue
		}
	}
	if shaderReturn != "16" {
		t.Fatalf("glCreateShader ReturnValue = %q, want 16", shaderReturn)
	}
	if programReturn != "18" {
		t.Fatalf("glCreateProgram ReturnValue = %q, want 18", programReturn)
	}
}

func TestRawTraceParser_FrameCostAndSwapTiming(t *testing.T) {
	input := `[ 1] (gc=0x1, tid=0x2): glClear 0x4100
[ 2] (gc=0x1, tid=0x2): glDrawArrays 0x0004 0 3
[ 3] glXSwapBuffers: dpy = 0x1, drawable = 2
[ 4] swapBuffers: 1234 us
[ 5] 0 frame cost 16ms
libGL: FPS = 60.0`

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(parsed.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(parsed.Frames))
	}
	frame := parsed.Frames[0]
	if !frame.HasTiming {
		t.Fatal("frame should have timing from frame cost")
	}
	if frame.TimingSource != "frame_cost" {
		t.Fatalf("TimingSource = %q, want frame_cost", frame.TimingSource)
	}
	if frame.TotalTimeUs != 16000 {
		t.Fatalf("TotalTimeUs = %d, want 16000", frame.TotalTimeUs)
	}
	if frame.SwapBufferTimeUs != 1234 {
		t.Fatalf("SwapBufferTimeUs = %d, want 1234", frame.SwapBufferTimeUs)
	}
	if frame.APITotalTimeUs != 14766 {
		t.Fatalf("APITotalTimeUs = %d, want 14766", frame.APITotalTimeUs)
	}
	if parsed.TotalTimeUs != 16000 {
		t.Fatalf("Parsed TotalTimeUs = %d, want 16000", parsed.TotalTimeUs)
	}
	if parsed.FPS != 60.0 {
		t.Fatalf("FPS = %f, want 60.0", parsed.FPS)
	}
	if frame.APISummary["glDrawArrays"] == nil || frame.APISummary["glDrawArrays"].Count != 1 {
		t.Fatalf("APISummary missing glDrawArrays: %+v", frame.APISummary)
	}
}

func TestRawTraceParser_ContextManagement(t *testing.T) {
	input := `(gc=0xa, tid=0xb): glXMakeCurrent: dpy = 0x1c00, drawable = 121
(gc=0xa, tid=0xb): glXCreateContextAttribsARB: dpy = 0x1c00, config = 0x8b, share_list = 0
(gc=0xa, tid=0xb): glClear 0x4100
glXSwapBuffers: dpy = 0x1c00, drawable = 121`

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(parsed.Frames))
	}

	frame := parsed.Frames[0]
	if len(frame.APICalls) != 3 {
		t.Fatalf("Expected 3 API calls, got %d", len(frame.APICalls))
	}

	names := make([]string, len(frame.APICalls))
	for i, c := range frame.APICalls {
		names[i] = c.APIName
	}

	if names[0] != "glXMakeCurrent" {
		t.Errorf("Expected glXMakeCurrent, got %s", names[0])
	}
	if names[1] != "glXCreateContextAttribsARB" {
		t.Errorf("Expected glXCreateContextAttribsARB, got %s", names[1])
	}
	if names[2] != "glClear" {
		t.Errorf("Expected glClear, got %s", names[2])
	}
}

func TestRawTraceParser_APILogEntry_Initialized(t *testing.T) {
	data, err := os.ReadFile("../bug/testdata/sample_trace.log")
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	parser := NewRawTraceParser()
	parsed, err := parser.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for fi, frame := range parsed.Frames {
		for ci, call := range frame.APICalls {
			if call.APIName == "" {
				t.Errorf("Frame %d call %d has empty APIName", fi, ci)
			}
			if call.Count != 1 {
				t.Errorf("Frame %d call %d has Count=%d, want 1", fi, ci, call.Count)
			}
			if call.LineNum == 0 {
				t.Errorf("Frame %d call %d has LineNum=0", fi, ci)
			}
		}
	}
}
