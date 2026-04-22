package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sort"
	"strings"
	"sync"

	"gst/internal/core"
	"gst/internal/core/analyzer"
	"gst/internal/core/exporter"
	"gst/internal/core/parser"
	"gst/internal/core/search"
)

// Handler HTTP handler with shared state
type Handler struct {
	mu         sync.RWMutex
	logFile    string          // current log file path
	rawLogPath string          // raw log file path for search
	current    *core.ParsedLog // current parsed log
	lines      []string        // raw lines for search
	index      *search.KeywordIndex
}

// NewHandler creates a new Handler
func NewHandler() *Handler {
	return &Handler{
		index: search.NewKeywordIndex(),
	}
}

// R1: ParseLog handles POST /api/log/parse
func (h *Handler) ParseLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var parsed *core.ParsedLog
	var logFile string

	// Check content type for multipart/form-data
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse multipart form: %v", err), http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get uploaded file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := r.FormValue("filename")
		if filename == "" {
			filename = "uploaded_log"
		}

		p, err := parser.CreateParserAuto(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to detect parser: %v", err), http.StatusInternalServerError)
			return
		}
		parsed, err = p.Parse(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse: %v", err), http.StatusInternalServerError)
			return
		}
		logFile = filename
	} else {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
			return
		}

		file, err := os.Open(req.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		p, err := parser.CreateParserAuto(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to detect parser: %v", err), http.StatusInternalServerError)
			return
		}
		parsed, err = p.Parse(file)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse: %v", err), http.StatusInternalServerError)
			return
		}
		logFile = req.Path
	}

	lines := buildLinesFromLog(parsed)
	_ = search.KeywordSearchSimple([]string{""}, lines)

	h.mu.Lock()
	h.logFile = logFile
	h.rawLogPath = logFile
	h.current = parsed
	h.lines = lines
	h.index.Build(parsed)
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toParseResult(parsed))
}

// R2: GetFrames handles GET /api/log/frames
// Returns paginated frame list
func (h *Handler) GetFrames(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	// Parse pagination params
	page := 1
	pageSize := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	total := len(current.Frames)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		start = 0
		end = 0
		page = 1
	}
	if end > total {
		end = total
	}

	// Build lightweight frame summaries to avoid returning large data
	var summaries []FrameSummary
	for _, frame := range current.Frames[start:end] {
		summaries = append(summaries, FrameSummary{
			FrameNum:         frame.FrameNum,
			StartLine:        frame.StartLine,
			EndLine:          frame.EndLine,
			TotalTimeUs:      frame.TotalTimeUs,
			SwapBufferTimeUs: frame.SwapBufferTimeUs,
			APITotalTimeUs:   frame.APITotalTimeUs,
		})
	}
	if summaries == nil {
		summaries = []FrameSummary{}
	}

	response := FramesResponse{
		Frames:    summaries,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// R3: GetFrameDetail handles GET /api/log/frames/:id
// Returns frame summary only (no APICalls/APISummary/Shaders to avoid large payloads)
func (h *Handler) GetFrameDetail(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	for _, frame := range current.Frames {
		if frame.FrameNum == id {
			summary := FrameSummary{
				FrameNum:         frame.FrameNum,
				StartLine:        frame.StartLine,
				EndLine:          frame.EndLine,
				TotalTimeUs:      frame.TotalTimeUs,
				SwapBufferTimeUs: frame.SwapBufferTimeUs,
				APITotalTimeUs:   frame.APITotalTimeUs,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(summary)
			return
		}
	}

	http.Error(w, "Frame not found", http.StatusNotFound)
}

// GetFrameFuncs handles GET /api/log/frames/:id/funcs
// Returns function statistics for a single frame
func (h *Handler) GetFrameFuncs(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-2]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	for _, frame := range current.Frames {
		if frame.FrameNum == id {
			// Convert APISummary map to sorted slice
			var stats []core.FuncStats
			for _, summary := range frame.APISummary {
				stats = append(stats, core.FuncStats{
					FuncName:    summary.APIName,
					CallCount:   summary.Count,
					TotalTimeUs: summary.TimeUs,
					AvgTimeUs:   summary.TimeUs / int64(summary.Count),
				})
			}
			// Sort by total time descending
			sort.Slice(stats, func(i, j int) bool {
				return stats[i].TotalTimeUs > stats[j].TotalTimeUs
			})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stats)
			return
		}
	}

	http.Error(w, "Frame not found", http.StatusNotFound)
}

// SearchResultItem is a single search result entry
type SearchResultItem struct {
	LineNumber int    `json:"line_number"`
	Content    string `json:"content"`
}

// SearchResponse is the response type for search
type SearchResponse struct {
	Results  []SearchResultItem `json:"results"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// Search handles GET /api/log/search
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	rawLogPath := h.rawLogPath
	h.mu.RUnlock()

	if rawLogPath == "" {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	// Parse pagination params
	page := 1
	pageSize := 50
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	keywords := strings.Fields(query)

	// Open raw log file and stream scan
	file, err := os.Open(rawLogPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open log file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var allMatches []SearchResultItem
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 10*1024*1024)
	scanner.Buffer(buf, 10*1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if matchLine(line, keywords) {
			allMatches = append(allMatches, SearchResultItem{
				LineNumber: lineNum,
				Content:    line,
			})
		}
	}

	total := len(allMatches)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		start = 0
		end = 0
		page = 1
	}
	if end > total {
		end = total
	}

	var pageResults []SearchResultItem
	if start < total {
		pageResults = allMatches[start:end]
	} else {
		pageResults = []SearchResultItem{}
	}

	response := SearchResponse{
		Results:  pageResults,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// matchLine checks if a line matches all keywords (case-insensitive)
func matchLine(line string, keywords []string) bool {
	lower := strings.ToLower(line)
	for _, kw := range keywords {
		if !strings.Contains(lower, strings.ToLower(kw)) {
			return false
		}
	}
	return true
}

// R5: AnalyzeTop handles GET /api/log/analyze/top
// Returns top N frames by TotalTimeUs
func (h *Handler) AnalyzeTop(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	topStr := r.URL.Query().Get("n")
	n := 20 // Default top 20
	if topStr != "" {
		if parsed, err := strconv.Atoi(topStr); err == nil && parsed > 0 {
			n = parsed
		}
	}

	fa := analyzer.NewFrameAnalyzer(current)
	frames := fa.FindTopSlowFrames(n)

	// Also include summary info
	summary := fa.GetFrameSummary()

	// Build lightweight frame summaries to avoid returning full FrameInfo with APICalls/Shaders
	frameSummaries := make([]FrameSummary, 0, len(frames))
	for _, frame := range frames {
		frameSummaries = append(frameSummaries, FrameSummary{
			FrameNum:         frame.FrameNum,
			StartLine:        frame.StartLine,
			EndLine:          frame.EndLine,
			TotalTimeUs:      frame.TotalTimeUs,
			SwapBufferTimeUs: frame.SwapBufferTimeUs,
			APITotalTimeUs:   frame.APITotalTimeUs,
		})
	}

	response := TopAnalysisResponse{
		Frames:     frameSummaries,
		Total:      len(current.Frames),
		SlowFrames: summary,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// R4: AnalyzeShaders handles GET /api/log/analyze/shaders
// Returns shader list from all frames
func (h *Handler) AnalyzeShaders(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	var allShaders []*core.ShaderInfo
	for _, frame := range current.Frames {
		allShaders = append(allShaders, frame.Shaders...)
	}

	// Truncate shader sources to avoid huge response sizes
	for _, shader := range allShaders {
		if len(shader.Source) > 2000 {
			shader.Source = shader.Source[:2000] + "\n[Source truncated]"
		}
	}

	response := ShadersResponse{
		Shaders: allShaders,
		Total:   len(allShaders),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// AnalyzeFuncs handles GET /api/log/analyze/funcs
// Returns function statistics aggregated across all frames
func (h *Handler) AnalyzeFuncs(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	fa := analyzer.NewFuncAnalyzer(current)
	stats := fa.Analyze()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Export handles POST /api/log/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	var req struct {
		Format string `json:"format"`
		Type   string `json:"type"`
		Query  string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	format := req.Format
	if format == "" {
		format = "json"
	}

	var data interface{}
	switch req.Type {
	case "frames":
		data = current.Frames
	case "funcs":
		fa := analyzer.NewFuncAnalyzer(current)
		data = fa.Analyze()
	case "shader":
		var allShaders []*core.ShaderInfo
		for _, frame := range current.Frames {
			allShaders = append(allShaders, frame.Shaders...)
		}
		data = allShaders
	case "search":
		h.mu.RLock()
		lines := h.lines
		h.mu.RUnlock()
		keywords := strings.Fields(req.Query)
		results := search.KeywordSearchSimple(keywords, lines)
		data = results
	case "top":
		n := 20
		if req.Query != "" {
			if parsed, err := strconv.Atoi(req.Query); err == nil && parsed > 0 {
				n = parsed
			}
		}
		fa := analyzer.NewFrameAnalyzer(current)
		data = fa.FindTopSlowFrames(n)
	case "longest":
		fa := analyzer.NewFrameAnalyzer(current)
		frames := fa.FindTopSlowFrames(1)
		if len(frames) > 0 {
			data = frames[0]
		} else {
			data = core.FrameInfo{}
		}
	default:
		data = current.Frames
	}

	w.Header().Set("Content-Type", mimeType(format))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=export.%s", format))

	switch format {
	case "txt":
		exporter.ExportAnalysisResult(data, "txt", w)
	case "csv":
		exporter.CSVExporter{Data: data}.Export(w)
	case "json":
		exporter.JSONExporter{Data: data}.Export(w)
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

// FramesResponse is the response type for frame list
type FramesResponse struct {
	Frames    interface{} `json:"frames"`
	Total     int         `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
}

// FrameSummary is a lightweight frame representation for list endpoints
// Avoids returning large fields like APICalls, Shaders, Programs, etc.
type FrameSummary struct {
	FrameNum         int   `json:"FrameNum"`
	StartLine        int   `json:"StartLine"`
	EndLine          int   `json:"EndLine"`
	TotalTimeUs      int64 `json:"TotalTimeUs"`
	SwapBufferTimeUs int64 `json:"SwapBufferTimeUs"`
	APITotalTimeUs   int64 `json:"APITotalTimeUs"`
}

// TopAnalysisResponse is the response type for top N analysis
type TopAnalysisResponse struct {
	Frames     interface{}              `json:"frames"`
	Total      int                     `json:"total"`
	SlowFrames map[string]interface{}   `json:"slow_frames,omitempty"`
}

// ShadersResponse is the response type for shader list
type ShadersResponse struct {
	Shaders interface{} `json:"shaders"`
	Total   int        `json:"total"`
}

// ParseResult is the API response format for parse
type ParseResult struct {
	Format         string       `json:"format"`
	FrameCount     int          `json:"frame_count"`
	FPS            float64      `json:"fps"`
	FirstFrameTime float64      `json:"first_frame_time"`
	LastFrameTime  float64      `json:"last_frame_time"`
	TotalTimeUs    int64        `json:"total_time_us"`
	Frames         interface{}  `json:"frames"`
}

// toParseResult converts core.ParsedLog to API ParseResult
// Note: Frames are NOT included - use /api/log/frames for paginated access
func toParseResult(p *core.ParsedLog) *ParseResult {
	result := &ParseResult{
		Format:         "unknown",
		FrameCount:     len(p.Frames),
		FPS:            p.FPS,
		TotalTimeUs:    p.TotalTimeUs,
		Frames:         nil, // Frames not included in parse result - use /api/log/frames
	}
	if len(p.Frames) > 0 {
		result.FirstFrameTime = float64(p.Frames[0].TotalTimeUs) / 1000.0
		result.LastFrameTime = float64(p.Frames[len(p.Frames)-1].TotalTimeUs) / 1000.0
	}
	if p.FPS > 0 {
		result.Format = "profile"
	} else {
		result.Format = "rawtrace"
	}
	return result
}

func mimeType(format string) string {
	switch format {
	case "json":
		return "application/json"
	case "csv":
		return "text/csv"
	case "txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// buildLinesFromLog builds raw lines from parsed log for searching
func buildLinesFromLog(log *core.ParsedLog) []string {
	var lines []string
	for _, frame := range log.Frames {
		for _, call := range frame.APICalls {
			line := fmt.Sprintf("%s: count=%d, time=%d", call.APIName, call.Count, call.TimeUs)
			lines = append(lines, line)
		}
	}
	return lines
}

// ServeUI handles GET /
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "web/index.html")
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}
