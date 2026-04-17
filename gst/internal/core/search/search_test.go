package search

import (
	"testing"

	"gst/internal/core"
)

func TestKeywordSearch(t *testing.T) {
	lines := []string{
		"glBindBuffer: count=491, time=588 us",
		"glBindFramebuffer: count=29, time=25377 us",
		"glDrawElements: count=493, time=11214 us",
		"libGL: FPS = 8.9",
		"swapBuffers: 3033 us",
	}

	ks := NewKeywordSearch()

	// Test single keyword
	results := ks.Search([]string{"glBind"}, lines)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'glBind', got %d", len(results))
	}

	// Test AND logic (multiple keywords)
	results = ks.Search([]string{"count", "493"}, lines)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'count' AND '493', got %d", len(results))
	}

	// Test case insensitive
	results = ks.Search([]string{"GLBIND"}, lines)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for uppercase 'GLBIND', got %d", len(results))
	}

	// Test no match
	results = ks.Search([]string{"notexist"}, lines)
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'notexist', got %d", len(results))
	}
}

func TestKeywordSearchWithPagination(t *testing.T) {
	// Create 2500 lines
	lines := make([]string, 2500)
	for i := range lines {
		lines[i] = "glBindBuffer: count=491, time=588 us"
	}

	ks := NewKeywordSearch()

	// Get first page
	page0, total := ks.SearchWithPagination([]string{"glBind"}, lines, 0, 1000)
	if total != 2500 {
		t.Errorf("Expected total 2500, got %d", total)
	}
	if len(page0) != 1000 {
		t.Errorf("Expected page size 1000, got %d", len(page0))
	}

	// Get second page
	page1, total := ks.SearchWithPagination([]string{"glBind"}, lines, 1, 1000)
	if len(page1) != 1000 {
		t.Errorf("Expected page size 1000, got %d", len(page1))
	}

	// Get third page (partial)
	page2, total := ks.SearchWithPagination([]string{"glBind"}, lines, 2, 1000)
	if len(page2) != 500 {
		t.Errorf("Expected page size 500, got %d", len(page2))
	}

	// Page beyond range
	page3, total := ks.SearchWithPagination([]string{"glBind"}, lines, 10, 1000)
	if page3 != nil {
		t.Errorf("Expected nil for out of range page, got %v", page3)
	}
}

func TestTimeRangeSearch(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum:    1,
				TotalTimeUs: 3000,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", TimeUs: 1000, LineNum: 1},
					{APIName: "glDrawElements", TimeUs: 2000, LineNum: 2},
				},
			},
			{
				FrameNum:    2,
				TotalTimeUs: 800,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", TimeUs: 500, LineNum: 3},
					{APIName: "glDrawElements", TimeUs: 300, LineNum: 4},
				},
			},
		},
	}

	trs := NewTimeRangeSearch()

	// Search in range [1500, 5000] - frame 1 with TotalTimeUs=3000 matches, returns 2 calls
	results := trs.Search(log, 1500, 5000)
	if len(results) != 2 {
		t.Errorf("Expected 2 results in range [1500, 5000], got %d", len(results))
	}

	// Search in range [0, 1000] - frame 2 with TotalTimeUs=800 matches, returns 2 calls
	results = trs.Search(log, 0, 1000)
	if len(results) != 2 {
		t.Errorf("Expected 2 results in range [0, 1000], got %d", len(results))
	}

	// Search in range [0, 500] - no frames match
	results = trs.Search(log, 0, 500)
	if len(results) != 0 {
		t.Errorf("Expected 0 results in range [0, 500], got %d", len(results))
	}

	// Search by frame range
	frames := trs.SearchByFrameRange(log, 1, 1)
	if len(frames) != 1 {
		t.Errorf("Expected 1 frame in range [1, 1], got %d", len(frames))
	}
}

func TestKeywordIndex(t *testing.T) {
	log := &core.ParsedLog{
		Frames: []core.FrameInfo{
			{
				FrameNum: 1,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", LineNum: 1},
					{APIName: "glDrawElements", LineNum: 2},
				},
			},
			{
				FrameNum: 2,
				APICalls: []core.APILogEntry{
					{APIName: "glBindBuffer", LineNum: 3},
				},
			},
		},
	}

	ki := NewKeywordIndex()

	// Should not be indexed yet
	if ki.HasIndexed() {
		t.Error("Expected HasIndexed to be false before Build")
	}

	ki.Build(log)

	// Should be indexed now
	if !ki.HasIndexed() {
		t.Error("Expected HasIndexed to be true after Build")
	}

	// Search for function
	lines := ki.Search("glBindBuffer")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines for 'glBindBuffer', got %d", len(lines))
	}
}

func TestKeywordSearchSimple(t *testing.T) {
	lines := []string{
		"glBindBuffer: count=491, time=588 us",
		"glBindFramebuffer: count=29, time=25377 us",
		"glDrawElements: count=493, time=11214 us",
	}

	// Test simple search
	results := KeywordSearchSimple([]string{"glBind"}, lines)
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'glBind', got %d", len(results))
	}

	// Test AND logic
	results = KeywordSearchSimple([]string{"count", "493"}, lines)
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'count' AND '493', got %d", len(results))
	}
}
