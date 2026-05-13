package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gst/internal/core"
	"gst/internal/core/analyzer"
	"gst/internal/core/bug"
	"gst/internal/core/exporter"
	"gst/internal/core/parser"
	"gst/internal/core/search"
)

const (
	DefaultBufferSize            = 10 * 1024 * 1024
	MaxMultipartSize             = 100 << 20
	ShaderSourceTruncateLen      = 2000
	TraceShaderSourceTruncateLen = 50000
)

// Handler HTTP handler with shared state
type Handler struct {
	mu         sync.RWMutex
	logFile    string          // current log file path
	rawLogPath string          // raw log file path for search
	current    *core.ParsedLog // current parsed log
	lines      []string        // raw lines for search
	index      *search.KeywordIndex
	format     string // detected log format
	traceCache *core.TraceAnalysis
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

	slog.Debug("ParseLog request", "method", r.Method, "remote", r.RemoteAddr)
	var parsed *core.ParsedLog
	var logFile string

	h.mu.Lock()
	h.current = nil
	h.lines = nil
	h.traceCache = nil
	h.index = search.NewKeywordIndex()
	h.mu.Unlock()
	runtime.GC()

	// Check content type for multipart/form-data
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(MaxMultipartSize); err != nil {
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

		if err := validateLogPath(req.Path); err != nil {
			http.Error(w, fmt.Sprintf("Access denied: %v", err), http.StatusBadRequest)
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

	h.mu.Lock()
	h.logFile = logFile
	h.rawLogPath = logFile
	h.current = parsed
	h.lines = lines
	h.index.Build(parsed)
	h.format = toParseResult(parsed).Format
	h.traceCache = nil
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
		summaries = append(summaries, toFrameSummary(frame))
	}
	if summaries == nil {
		summaries = []FrameSummary{}
	}

	response := FramesResponse{
		Frames:   summaries,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// R3: GetFrameDetail handles GET /api/log/frames/:id
// Returns full frame info including APICalls and APISummary
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
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(toFrameDetail(frame))
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
			json.NewEncoder(w).Encode(toFuncStatsList(stats))
			return
		}
	}

	http.Error(w, "Frame not found", http.StatusNotFound)
}

// GetFramePrograms handles GET /api/log/frames/:id/programs
// Returns qapitrace-like static program/shader usage insight for a frame.
func (h *Handler) GetFramePrograms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := frameIDFromPath(r.URL.Path, "programs")
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	trace, err := h.getTraceAnalysis()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	insight, ok := trace.FrameInsights[id]
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insight)
}

// GetFrameDrawCalls handles GET /api/log/frames/:id/drawcalls
// Returns paginated draw calls with inferred active program and lightweight state.
func (h *Handler) GetFrameDrawCalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := frameIDFromPath(r.URL.Path, "drawcalls")
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	trace, err := h.getTraceAnalysis()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	programFilter := 0
	if programParam := r.URL.Query().Get("program"); programParam != "" {
		programID, err := strconv.Atoi(programParam)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid program filter: %v", err), http.StatusBadRequest)
			return
		}
		programFilter = programID
	}

	drawCalls, ok := analyzer.NewTraceInspectorAnalyzer(current).AnalyzeFrameDrawCalls(id, programFilter, trace)
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}

	page, pageSize := paginationParams(r, 1, 100)
	total := len(drawCalls)
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

	pageItems := []core.DrawCallInsight{}
	if start < total {
		pageItems = drawCalls[start:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(core.FrameDrawCallPage{
		FrameNum:  id,
		DrawCalls: pageItems,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	})
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
	slog.Debug("Search request", "method", r.Method, "remote", r.RemoteAddr, "query", r.URL.Query().Get("q"))
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
	buf := make([]byte, DefaultBufferSize)
	scanner.Buffer(buf, DefaultBufferSize)
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

func (h *Handler) getTraceAnalysis() (*core.TraceAnalysis, error) {
	h.mu.RLock()
	current := h.current
	cached := h.traceCache
	h.mu.RUnlock()

	if current == nil {
		return nil, errors.New("No log parsed")
	}
	if cached != nil {
		return cached, nil
	}

	trace := analyzer.NewTraceInspectorAnalyzer(current).Analyze()
	if trace == nil {
		return nil, errors.New("Trace analysis failed")
	}

	h.mu.Lock()
	if h.current == current {
		h.traceCache = trace
		cached = trace
	} else {
		cached = h.traceCache
	}
	h.mu.Unlock()

	if cached == nil {
		return trace, nil
	}
	return cached, nil
}

func frameIDFromPath(path string, suffix string) (int, error) {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) < 2 || parts[len(parts)-1] != suffix {
		return 0, fmt.Errorf("expected /frames/:id/%s", suffix)
	}
	return strconv.Atoi(parts[len(parts)-2])
}

func paginationParams(r *http.Request, defaultPage int, defaultPageSize int) (int, int) {
	page := defaultPage
	pageSize := defaultPageSize
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
	return page, pageSize
}

func stripProgramSource(program core.ProgramInfo, includeSource bool) core.ProgramInfo {
	if !includeSource {
		program.Shaders = nil
		program.FramesUsed = []int{}
		if program.ShaderIDs == nil {
			program.ShaderIDs = []int{}
		}
		return program
	}
	if program.ShaderIDs == nil {
		program.ShaderIDs = []int{}
	}
	if program.FramesUsed == nil {
		program.FramesUsed = []int{}
	}
	for i := range program.Shaders {
		if len(program.Shaders[i].Source) > TraceShaderSourceTruncateLen {
			program.Shaders[i].Source = program.Shaders[i].Source[:TraceShaderSourceTruncateLen] + "\n[Source truncated]"
		}
	}
	return program
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
		frameSummaries = append(frameSummaries, toFrameSummary(frame))
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
		if len(shader.Source) > ShaderSourceTruncateLen {
			shader.Source = shader.Source[:ShaderSourceTruncateLen] + "\n[Source truncated]"
		}
	}

	response := ShadersResponse{
		Shaders: toShaderList(allShaders),
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
	json.NewEncoder(w).Encode(toFuncStatsList(stats))
}

// Export handles POST /api/log/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	slog.Debug("Export request", "method", r.Method, "remote", r.RemoteAddr)
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
		exportAnalysisTxt(data, w)
	case "csv":
		exportAnalysisCSV(data, w)
	case "json":
		exportAnalysisJSON(data, w)
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

func exportAnalysisTxt(data interface{}, w http.ResponseWriter) {
	switch v := data.(type) {
	case []core.FrameInfo:
		exporter.ExportFramesTxt(v, w)
	case core.FrameInfo:
		exporter.ExportFramesTxt([]core.FrameInfo{v}, w)
	case []core.FuncStats:
		exporter.ExportFuncStatsTxt(v, w)
	case []*core.ShaderInfo:
		exporter.ExportShaderInfosTxt(v, w)
	case []core.SearchResult:
		exporter.ExportSearchResultsTxt(v, w)
	default:
		http.Error(w, "Unsupported data type for TXT export", http.StatusBadRequest)
	}
}

func exportAnalysisCSV(data interface{}, w http.ResponseWriter) {
	switch v := data.(type) {
	case []core.FrameInfo:
		exporter.FramesCSVExporter{Frames: v}.Export(w)
	case core.FrameInfo:
		exporter.SingleFrameCSVExporter{Frame: v}.Export(w)
	case []core.FuncStats:
		exporter.FuncStatsCSVExporter{Stats: v}.Export(w)
	case []*core.ShaderInfo:
		exporter.ShaderInfoCSVExporter{Shaders: v}.Export(w)
	case []core.SearchResult:
		exporter.SearchResultCSVExporter{Results: v}.Export(w)
	case []core.ShaderCompileInfo:
		exporter.ShaderCompileCSVExporter{Infos: v}.Export(w)
	default:
		http.Error(w, "Unsupported data type for CSV export", http.StatusBadRequest)
	}
}

func exportAnalysisJSON(data interface{}, w http.ResponseWriter) {
	switch v := data.(type) {
	case []core.FrameInfo:
		exporter.JSONExporter[[]core.FrameInfo]{Data: v}.Export(w)
	case core.FrameInfo:
		exporter.JSONExporter[core.FrameInfo]{Data: v}.Export(w)
	case []core.FuncStats:
		exporter.JSONExporter[[]core.FuncStats]{Data: v}.Export(w)
	case []*core.ShaderInfo:
		exporter.JSONExporter[[]*core.ShaderInfo]{Data: v}.Export(w)
	case []core.SearchResult:
		exporter.JSONExporter[[]core.SearchResult]{Data: v}.Export(w)
	case []core.ShaderCompileInfo:
		exporter.JSONExporter[[]core.ShaderCompileInfo]{Data: v}.Export(w)
	default:
		http.Error(w, "Unsupported data type for JSON export", http.StatusBadRequest)
	}
}

// FramesResponse is the response type for frame list
type FramesResponse struct {
	Frames   []FrameSummary `json:"frames"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// FrameSummary is a lightweight frame representation for list endpoints
// Avoids returning large fields like APICalls, Shaders, Programs, etc.
type FrameSummary struct {
	FrameNum         int   `json:"frame_num"`
	StartLine        int   `json:"start_line"`
	EndLine          int   `json:"end_line"`
	TotalTimeUs      int64 `json:"total_time_us"`
	SwapBufferTimeUs int64 `json:"swap_buffer_time_us"`
	APITotalTimeUs   int64 `json:"api_total_time_us"`
	APICount         int   `json:"api_count"`
}

type APICallResponse struct {
	Name        string `json:"name"`
	Count       int    `json:"count"`
	TimeUs      int64  `json:"time_us"`
	LineNum     int    `json:"line_num"`
	RawParams   string `json:"raw_params,omitempty"`
	ReturnValue string `json:"return_value,omitempty"`
	GCAddr      string `json:"gc_addr,omitempty"`
	TID         string `json:"tid,omitempty"`
	IsError     bool   `json:"is_error,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
	HasNilPtr   bool   `json:"has_nil_ptr,omitempty"`
}

type FuncStatResponse struct {
	Name        string `json:"name"`
	CallCount   int    `json:"call_count"`
	TotalTimeUs int64  `json:"total_time_us"`
	AvgTimeUs   int64  `json:"avg_time_us"`
}

type ShaderResponse struct {
	ID          int    `json:"id"`
	CommandLine string `json:"command_line,omitempty"`
	Source      string `json:"source"`
}

type FrameDetailResponse struct {
	FrameNum         int                `json:"frame_num"`
	StartLine        int                `json:"start_line"`
	EndLine          int                `json:"end_line"`
	TotalTimeUs      int64              `json:"total_time_us"`
	SwapBufferTimeUs int64              `json:"swap_buffer_time_us"`
	APITotalTimeUs   int64              `json:"api_total_time_us"`
	APICount         int                `json:"api_count"`
	APICalls         []APICallResponse  `json:"api_calls"`
	FuncStats        []FuncStatResponse `json:"func_stats"`
	Shaders          []ShaderResponse   `json:"shaders"`
	Programs         []int              `json:"programs"`
	BufferCreations  []core.BufferInfo  `json:"buffer_creations"`
}

// TopAnalysisResponse is the response type for top N analysis
type TopAnalysisResponse struct {
	Frames     []FrameSummary     `json:"frames"`
	Total      int                `json:"total"`
	SlowFrames *core.FrameSummary `json:"slow_frames,omitempty"`
}

// ShadersResponse is the response type for shader list
type ShadersResponse struct {
	Shaders []ShaderResponse `json:"shaders"`
	Total   int              `json:"total"`
}

// ParseResult is the API response format for parse
type ParseResult struct {
	Format       string  `json:"format"`
	FrameCount   int     `json:"frame_count"`
	FPS          float64 `json:"fps"`
	MaxFrameTime float64 `json:"max_frame_time"`
	TotalTimeUs  int64   `json:"total_time_us"`
}

type OverviewPerformanceResponse struct {
	FrameTime      core.OverviewFrameTime `json:"frame_time"`
	SlowFramesTop5 []FrameSummary         `json:"slow_frames_top5"`
	BottleneckHint string                 `json:"bottleneck_hint"`
}

type OverviewResponse struct {
	Basic            core.OverviewBasic          `json:"basic"`
	Performance      OverviewPerformanceResponse `json:"performance"`
	DiagnosisSummary core.OverviewDiagnosis      `json:"diagnosis_summary"`
	Resources        core.OverviewResources      `json:"resources"`
	Summary          string                      `json:"summary"`
}

type WorkflowResponse struct {
	Workflow   string                 `json:"workflow"`
	Conclusion string                 `json:"conclusion"`
	Evidence   []string               `json:"evidence"`
	Details    map[string]interface{} `json:"details"`
}

func toFrameSummary(frame core.FrameInfo) FrameSummary {
	return FrameSummary{
		FrameNum:         frame.FrameNum,
		StartLine:        frame.StartLine,
		EndLine:          frame.EndLine,
		TotalTimeUs:      frame.TotalTimeUs,
		SwapBufferTimeUs: frame.SwapBufferTimeUs,
		APITotalTimeUs:   frame.APITotalTimeUs,
		APICount:         len(frame.APICalls),
	}
}

func toAPICalls(calls []core.APILogEntry) []APICallResponse {
	result := make([]APICallResponse, 0, len(calls))
	for _, call := range calls {
		result = append(result, APICallResponse{
			Name:        call.APIName,
			Count:       call.Count,
			TimeUs:      call.TimeUs,
			LineNum:     call.LineNum,
			RawParams:   call.RawParams,
			ReturnValue: call.ReturnValue,
			GCAddr:      call.GCAddr,
			TID:         call.TID,
			IsError:     call.IsError,
			ErrorCode:   call.ErrorCode,
			HasNilPtr:   call.HasNilPtr,
		})
	}
	return result
}

func toFuncStat(stat core.FuncStats) FuncStatResponse {
	return FuncStatResponse{
		Name:        stat.FuncName,
		CallCount:   stat.CallCount,
		TotalTimeUs: stat.TotalTimeUs,
		AvgTimeUs:   stat.AvgTimeUs,
	}
}

func toFuncStatsList(stats []core.FuncStats) []FuncStatResponse {
	result := make([]FuncStatResponse, 0, len(stats))
	for _, stat := range stats {
		result = append(result, toFuncStat(stat))
	}
	return result
}

func toShader(shader *core.ShaderInfo) ShaderResponse {
	if shader == nil {
		return ShaderResponse{}
	}
	return ShaderResponse{
		ID:          shader.ID,
		CommandLine: shader.CommandLine,
		Source:      shader.Source,
	}
}

func toShaderList(shaders []*core.ShaderInfo) []ShaderResponse {
	result := make([]ShaderResponse, 0, len(shaders))
	for _, shader := range shaders {
		result = append(result, toShader(shader))
	}
	return result
}

func toFrameDetail(frame core.FrameInfo) FrameDetailResponse {
	funcStats := make([]core.FuncStats, 0, len(frame.APISummary))
	for _, summary := range frame.APISummary {
		avg := int64(0)
		if summary.Count > 0 {
			avg = summary.TimeUs / int64(summary.Count)
		}
		funcStats = append(funcStats, core.FuncStats{
			FuncName:    summary.APIName,
			CallCount:   summary.Count,
			TotalTimeUs: summary.TimeUs,
			AvgTimeUs:   avg,
		})
	}
	sort.Slice(funcStats, func(i, j int) bool {
		return funcStats[i].TotalTimeUs > funcStats[j].TotalTimeUs
	})

	return FrameDetailResponse{
		FrameNum:         frame.FrameNum,
		StartLine:        frame.StartLine,
		EndLine:          frame.EndLine,
		TotalTimeUs:      frame.TotalTimeUs,
		SwapBufferTimeUs: frame.SwapBufferTimeUs,
		APITotalTimeUs:   frame.APITotalTimeUs,
		APICount:         len(frame.APICalls),
		APICalls:         toAPICalls(frame.APICalls),
		FuncStats:        toFuncStatsList(funcStats),
		Shaders:          toShaderList(frame.Shaders),
		Programs:         frame.Programs,
		BufferCreations:  frame.BufferCreations,
	}
}

func toOverviewResponse(result *core.OverviewResult) OverviewResponse {
	slowFrames := make([]FrameSummary, 0, len(result.Performance.SlowFramesTop5))
	for _, frame := range result.Performance.SlowFramesTop5 {
		slowFrames = append(slowFrames, toFrameSummary(frame))
	}
	return OverviewResponse{
		Basic: result.Basic,
		Performance: OverviewPerformanceResponse{
			FrameTime:      result.Performance.FrameTime,
			SlowFramesTop5: slowFrames,
			BottleneckHint: result.Performance.BottleneckHint,
		},
		DiagnosisSummary: result.DiagnosisSummary,
		Resources:        result.Resources,
		Summary:          result.Summary,
	}
}

func toWorkflowResponse(result *core.WorkflowResult) WorkflowResponse {
	return WorkflowResponse{
		Workflow:   result.Workflow,
		Conclusion: result.Conclusion,
		Evidence:   result.Evidence,
		Details:    normalizeWorkflowDetails(result.Details),
	}
}

func normalizeWorkflowDetails(details interface{}) map[string]interface{} {
	raw, ok := details.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	normalized := make(map[string]interface{}, len(raw))
	for key, value := range raw {
		switch v := value.(type) {
		case []core.FrameInfo:
			frames := make([]FrameSummary, 0, len(v))
			for _, frame := range v {
				frames = append(frames, toFrameSummary(frame))
			}
			normalized[key] = frames
		case []core.FuncStats:
			normalized[key] = toFuncStatsList(v)
		default:
			normalized[key] = v
		}
	}
	return normalized
}

// toParseResult converts core.ParsedLog to API ParseResult
// Note: Frames are NOT included - use /api/log/frames for paginated access
func toParseResult(p *core.ParsedLog) *ParseResult {
	result := &ParseResult{
		Format:      "unknown",
		FrameCount:  len(p.Frames),
		FPS:         p.FPS,
		TotalTimeUs: p.TotalTimeUs,
	}
	if len(p.Frames) > 0 {
		// Find frame with highest TotalTimeUs
		maxTime := p.Frames[0].TotalTimeUs
		for _, frame := range p.Frames {
			if frame.TotalTimeUs > maxTime {
				maxTime = frame.TotalTimeUs
			}
		}
		result.MaxFrameTime = float64(maxTime) / 1000.0
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

// validateLogPath checks for directory traversal and restricts to allowed directory
func validateLogPath(path string) error {
	if path == "" {
		return errors.New("path is empty")
	}
	allowed := os.Getenv("GST_LOG_DIR")
	if allowed == "" {
		allowed = ".."
	}
	absAllowed, err := filepath.Abs(allowed)
	if err != nil {
		return fmt.Errorf("failed to resolve allowed directory: %v", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %v", err)
	}
	if !strings.HasPrefix(absPath, absAllowed+string(filepath.Separator)) && absPath != absAllowed {
		return fmt.Errorf("access denied: path is outside allowed directory (%s)", absAllowed)
	}
	return nil
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

// HandleDiagnose handles POST /api/diagnose
func (h *Handler) HandleDiagnose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	current := h.current
	logFile := h.logFile
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed. Please parse a log file first.", http.StatusBadRequest)
		return
	}

	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(current)
	report := bug.GenerateReport(logFile, findings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// Overview handles GET /api/overview
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	format := h.format
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	oa := analyzer.NewOverviewAnalyzer(current, format)
	result := oa.Analyze()
	if result == nil {
		http.Error(w, "Failed to generate overview", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toOverviewResponse(result))
}

// Workflow handles POST /api/analyze/workflow
func (h *Handler) Workflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.mu.RLock()
	current := h.current
	format := h.format
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	var req core.WorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	validWorkflows := map[string]bool{
		"performance": true, "crash": true, "rendering": true, "memory": true,
	}
	if !validWorkflows[req.Workflow] {
		http.Error(w, fmt.Sprintf("Invalid workflow '%s'. Valid: performance, crash, rendering, memory", req.Workflow), http.StatusBadRequest)
		return
	}

	result := analyzer.AnalyzeWorkflow(current, format, req.Workflow)
	if result == nil {
		http.Error(w, "Workflow analysis failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toWorkflowResponse(result))
}

// GetTracePrograms handles GET /api/log/trace/programs
// Returns a lightweight global program registry without shader source payloads.
func (h *Handler) GetTracePrograms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	trace, err := h.getTraceAnalysis()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	programs := make([]core.ProgramInfo, 0, len(trace.Programs))
	for _, program := range trace.Programs {
		programs = append(programs, stripProgramSource(program, false))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"programs": programs,
		"total":    len(programs),
	})
}

// GetTraceProgramDetail handles GET /api/log/trace/programs/:id
// Returns one program with attached shader source or binary metadata.
func (h *Handler) GetTraceProgramDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid program ID: %v", err), http.StatusBadRequest)
		return
	}

	trace, err := h.getTraceAnalysis()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	program, ok := trace.ProgramMap[id]
	if !ok {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stripProgramSource(*program, true))
}

// AnalyzeDrawCalls handles GET /api/log/analyze/drawcalls
func (h *Handler) AnalyzeDrawCalls(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	dca := analyzer.NewDrawCallAnalyzer(current)

	nStr := r.URL.Query().Get("n")
	n := 0
	if nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}

	if n > 0 {
		hotFrames := dca.FindHotFrames(n)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hot_frames": hotFrames,
		})
		return
	}

	summary := dca.GetSummary()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// AnalyzeTextures handles GET /api/log/analyze/textures
func (h *Handler) AnalyzeTextures(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	ta := analyzer.NewTextureAnalyzer(current)
	summary := ta.GetSummary()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// AnalyzeBottleneck handles GET /api/log/analyze/bottleneck
func (h *Handler) AnalyzeBottleneck(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()

	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	fa := analyzer.NewFrameAnalyzer(current)
	result := fa.AnalyzeBottleneck()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
