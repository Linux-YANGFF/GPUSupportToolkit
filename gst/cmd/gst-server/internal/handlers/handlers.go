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
	"gst/internal/core/glstats"
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
	var logFormat string

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
		logFormat = string(p.Kind())
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

		kind := parser.DetectKindFromReader(file, 5000)
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			http.Error(w, fmt.Sprintf("Failed to reset parser: %v", err), http.StatusInternalServerError)
			return
		}

		if kind == parser.KindRawTrace || kind == parser.KindUnknown {
			parsed, err = parser.ParseIndexedRawTraceFile(req.Path)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse indexed raw trace: %v", err), http.StatusInternalServerError)
				return
			}
			if len(parsed.Frames) == 0 && kind != parser.KindRawTrace {
				if _, err := file.Seek(0, io.SeekStart); err != nil {
					http.Error(w, fmt.Sprintf("Failed to reset parser: %v", err), http.StatusInternalServerError)
					return
				}
				p := parser.CreateParser(kind)
				parsed, err = p.Parse(file)
				if err != nil {
					http.Error(w, fmt.Sprintf("Failed to parse: %v", err), http.StatusInternalServerError)
					return
				}
				logFormat = string(p.Kind())
			} else {
				logFormat = string(parser.KindRawTrace)
			}
		} else {
			p := parser.CreateParser(kind)
			parsed, err = p.Parse(file)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to parse: %v", err), http.StatusInternalServerError)
				return
			}
			logFormat = string(p.Kind())
		}
		logFile = req.Path
	}

	var lines []string
	if !parsed.Indexed {
		lines = buildLinesFromLog(parsed)
	}
	parseResult := toParseResult(parsed)
	if logFormat != "" {
		parseResult.Format = logFormat
	}

	h.mu.Lock()
	h.logFile = logFile
	h.rawLogPath = logFile
	h.current = parsed
	h.lines = lines
	if !parsed.Indexed {
		h.index.Build(parsed)
	}
	h.format = parseResult.Format
	h.traceCache = parsed.Trace
	h.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(parseResult)
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
				avg := int64(0)
				if summary.Count > 0 {
					avg = summary.TimeUs / int64(summary.Count)
				}
				stats = append(stats, core.FuncStats{
					FuncName:    summary.APIName,
					CallCount:   summary.Count,
					TotalTimeUs: summary.TimeUs,
					AvgTimeUs:   avg,
				})
			}
			// Sort by total time descending
			sort.Slice(stats, func(i, j int) bool {
				if stats[i].TotalTimeUs == stats[j].TotalTimeUs {
					return stats[i].CallCount > stats[j].CallCount
				}
				return stats[i].TotalTimeUs > stats[j].TotalTimeUs
			})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(toFuncStatsList(stats))
			return
		}
	}

	http.Error(w, "Frame not found", http.StatusNotFound)
}

// GetFrameAPIs handles GET /api/log/frames/:id/apis
// Returns paginated raw API calls for indexed logs and full-frame pages for legacy logs.
func (h *Handler) GetFrameAPIs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := frameIDFromPath(r.URL.Path, "apis")
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	current := h.current
	rawLogPath := h.rawLogPath
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	frame, ok := findFrame(current, id)
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}

	page, pageSize := paginationParams(r, 1, 100)
	var calls []core.APILogEntry
	var total int
	if current.Indexed {
		calls, total, err = parser.ParseIndexedFrameAPICalls(rawLogPath, frame, page, pageSize)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read frame APIs: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		total = len(frame.APICalls)
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
		if start < total {
			calls = frame.APICalls[start:end]
		} else {
			calls = []core.APILogEntry{}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FrameAPICallsResponse{
		FrameNum: id,
		APICalls: toAPICalls(calls),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetFrameRawLines handles GET /api/log/frames/:id/raw-lines
// Returns paginated original frame log text for UI preview. Numeric apitrace
// sequence prefixes are stripped by default, while gc/tid/API text is preserved.
func (h *Handler) GetFrameRawLines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := frameIDFromPath(r.URL.Path, "raw-lines")
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	current := h.current
	rawLogPath := h.rawLogPath
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	frame, ok := findFrame(current, id)
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}

	page, pageSize := rawLinePaginationParams(r)
	stripLineNumber := r.URL.Query().Get("strip_line_number") != "false"
	var lines []string
	var total int
	if current.Indexed && rawLogPath != "" {
		lines, total, err = parser.ParseIndexedFrameRawLines(rawLogPath, frame, page, pageSize, stripLineNumber)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read frame raw log: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		all := frameRawCallLines(frame)
		total = len(all)
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
		lines = []string{}
		if start < total {
			lines = all[start:end]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(FrameRawLinesResponse{
		FrameNum:           id,
		Lines:              lines,
		Total:              total,
		Page:               page,
		PageSize:           pageSize,
		StartLine:          frameFullStartLine(frame),
		EndLine:            frameFullEndLine(frame),
		StrippedLineNumber: stripLineNumber,
	})
}

// DownloadFrameRawLog handles GET /api/log/frames/:id/download.
// It streams the original frame range from disk for indexed logs.
func (h *Handler) DownloadFrameRawLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id, err := frameIDFromPath(r.URL.Path, "download")
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	current := h.current
	rawLogPath := h.rawLogPath
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	frame, ok := findFrame(current, id)
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=frame_%d.log", id))
	if current.Indexed && rawLogPath != "" {
		if err := parser.CopyIndexedFrameRawLog(w, rawLogPath, frame); err != nil {
			slog.Warn("Failed to stream frame log", "frame", id, "error", err)
		}
		return
	}

	writer := bufio.NewWriter(w)
	defer writer.Flush()
	for _, line := range frameRawCallLines(frame) {
		fmt.Fprintln(writer, line)
	}
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
	json.NewEncoder(w).Encode(normalizeFrameProgramInsight(insight))
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
	rawLogPath := h.rawLogPath
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

	frame, ok := findFrame(current, id)
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}

	page, pageSize := paginationParams(r, 1, 100)
	var pageItems []core.DrawCallInsight
	var total int
	if current.Indexed {
		pageItems, total, err = parser.ParseIndexedFrameDrawCalls(rawLogPath, frame, programFilter, page, pageSize, trace)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read frame draw calls: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		drawCalls, ok := analyzer.NewTraceInspectorAnalyzer(current).AnalyzeFrameDrawCalls(id, programFilter, trace)
		if !ok {
			http.Error(w, "Frame not found", http.StatusNotFound)
			return
		}
		total = len(drawCalls)
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
		pageItems = []core.DrawCallInsight{}
		if start < total {
			pageItems = drawCalls[start:end]
		}
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

type V2CaseOverviewResponse struct {
	CaseID             string            `json:"case_id"`
	Format             string            `json:"format"`
	SourceFile         string            `json:"source_file"`
	Stats              glstats.CaseStats `json:"stats"`
	DeprecatedFeatures []string          `json:"deprecated_features"`
}

type V2AIFrameSummary struct {
	FrameNum      int                       `json:"frame_num"`
	LineRange     string                    `json:"line_range"`
	TotalTimeUs   int64                     `json:"total_time_us"`
	SwapTimeUs    int64                     `json:"swap_time_us"`
	APITimeUs     int64                     `json:"api_time_us"`
	APICallCount  int                       `json:"api_call_count"`
	DrawCallCount int                       `json:"draw_call_count"`
	KeyAPIs       []glstats.APICounter      `json:"key_apis"`
	Categories    []glstats.CategoryCounter `json:"categories"`
	Evidence      []string                  `json:"evidence"`
}

type V2AISummaryResponse struct {
	CaseID        string                    `json:"case_id"`
	Format        string                    `json:"format"`
	SourceFile    string                    `json:"source_file"`
	FrameCount    int                       `json:"frame_count"`
	HasTiming     bool                      `json:"has_timing"`
	TopFrames     []V2AIFrameSummary        `json:"top_frames"`
	TopAPIs       []glstats.APICounter      `json:"top_apis"`
	CategoryStats []glstats.CategoryCounter `json:"category_stats"`
	Contract      glstats.AIContract        `json:"contract"`
}

// GetV2CaseOverview handles GET /api/v2/cases/current/overview.
func (h *Handler) GetV2CaseOverview(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	format := h.format
	logFile := h.logFile
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}
	response := V2CaseOverviewResponse{
		CaseID:     "current",
		Format:     format,
		SourceFile: logFile,
		Stats:      glstats.AnalyzeCase(current, 10),
		DeprecatedFeatures: []string{
			"bug_diagnosis",
			"workflow_analysis",
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleV2CurrentFrame routes /api/v2/cases/current/frames/:id/{stats,apis,drawcalls,programs}.
func (h *Handler) HandleV2CurrentFrame(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case strings.HasSuffix(path, "/stats"):
		h.GetV2FrameStats(w, r)
	case strings.HasSuffix(path, "/apis"):
		h.GetFrameAPIs(w, r)
	case strings.HasSuffix(path, "/drawcalls"):
		h.GetFrameDrawCalls(w, r)
	case strings.HasSuffix(path, "/programs"):
		h.GetFramePrograms(w, r)
	case strings.HasSuffix(path, "/raw-lines"):
		h.GetFrameRawLines(w, r)
	case strings.HasSuffix(path, "/download"):
		h.DownloadFrameRawLog(w, r)
	default:
		h.GetFrameDetail(w, r)
	}
}

// GetV2FrameStats handles GET /api/v2/cases/current/frames/:id/stats.
func (h *Handler) GetV2FrameStats(w http.ResponseWriter, r *http.Request) {
	id, err := frameIDFromPath(r.URL.Path, "stats")
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid frame ID: %v", err), http.StatusBadRequest)
		return
	}
	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}
	frame, ok := findFrame(current, id)
	if !ok {
		http.Error(w, "Frame not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(glstats.AnalyzeFrame(frame))
}

// GetV2AISummary returns a small stable payload intended for AI tools.
func (h *Handler) GetV2AISummary(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	current := h.current
	format := h.format
	logFile := h.logFile
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	n := 10
	if raw := r.URL.Query().Get("n"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			n = parsed
		}
	}
	caseStats := glstats.AnalyzeCase(current, n)
	topFrames := make([]V2AIFrameSummary, 0, len(caseStats.TopFramesByTime))
	for _, frameStats := range caseStats.TopFramesByTime {
		topFrames = append(topFrames, toV2AIFrameSummary(frameStats))
	}
	response := V2AISummaryResponse{
		CaseID:        "current",
		Format:        format,
		SourceFile:    logFile,
		FrameCount:    caseStats.FrameCount,
		HasTiming:     caseStats.HasTiming,
		TopFrames:     topFrames,
		TopAPIs:       caseStats.TopAPIs,
		CategoryStats: caseStats.CategoryStats,
		Contract:      caseStats.AIContract,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

type TimeRangeSearchResponse struct {
	Frames   []FrameSummary     `json:"frames"`
	APICalls []core.APILogEntry `json:"api_calls"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
	StartUs  int64              `json:"start_us"`
	EndUs    int64              `json:"end_us"`
	Indexed  bool               `json:"indexed"`
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
	if pageSize > 500 {
		pageSize = 500
	}

	keywords := strings.Fields(query)

	// Open raw log file and stream scan
	file, err := os.Open(rawLogPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open log file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	pageResults := []SearchResultItem{}
	scanner := bufio.NewScanner(file)
	buf := make([]byte, DefaultBufferSize)
	scanner.Buffer(buf, DefaultBufferSize)
	lineNum := 0
	total := 0
	start := (page - 1) * pageSize
	end := start + pageSize

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if matchLine(line, keywords) {
			total++
			if total > start && total <= end {
				pageResults = append(pageResults, SearchResultItem{
					LineNumber: lineNum,
					Content:    line,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan log file: %v", err), http.StatusInternalServerError)
		return
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

// SearchTimeRange handles GET /api/log/search/time?start_us=1000&end_us=50000.
func (h *Handler) SearchTimeRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startUs, err := strconv.ParseInt(r.URL.Query().Get("start_us"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid or missing start_us", http.StatusBadRequest)
		return
	}
	endUs, err := strconv.ParseInt(r.URL.Query().Get("end_us"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid or missing end_us", http.StatusBadRequest)
		return
	}
	if startUs > endUs {
		http.Error(w, "start_us must be <= end_us", http.StatusBadRequest)
		return
	}

	page, pageSize := paginationParams(r, 1, 50)
	if pageSize > 500 {
		pageSize = 500
	}

	h.mu.RLock()
	current := h.current
	h.mu.RUnlock()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	matches := make([]core.FrameInfo, 0)
	for _, frame := range current.Frames {
		if frame.TotalTimeUs >= startUs && frame.TotalTimeUs <= endUs {
			matches = append(matches, frame)
		}
	}

	total := len(matches)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	frameSummaries := make([]FrameSummary, 0, end-start)
	apiCalls := []core.APILogEntry{}
	for _, frame := range matches[start:end] {
		frameSummaries = append(frameSummaries, toFrameSummary(frame))
		if !current.Indexed {
			apiCalls = append(apiCalls, frame.APICalls...)
		}
	}

	response := TimeRangeSearchResponse{
		Frames:   frameSummaries,
		APICalls: apiCalls,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		StartUs:  startUs,
		EndUs:    endUs,
		Indexed:  current.Indexed,
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
	if current.Trace != nil {
		return current.Trace, nil
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

func rawLinePaginationParams(r *http.Request) (int, int) {
	page, pageSize := paginationParams(r, 1, 200)
	if pageSize > 1000 {
		pageSize = 1000
	}
	return page, pageSize
}

func frameFullStartLine(frame core.FrameInfo) int {
	if frame.FullStartLine > 0 {
		return frame.FullStartLine
	}
	return frame.StartLine
}

func frameFullEndLine(frame core.FrameInfo) int {
	if frame.FullEndLine > 0 {
		return frame.FullEndLine
	}
	return frame.EndLine
}

func frameRawCallLines(frame core.FrameInfo) []string {
	lines := make([]string, 0, len(frame.APICalls))
	for _, call := range frame.APICalls {
		if line := formatRawCallLine(call); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func formatRawCallLine(call core.APILogEntry) string {
	if call.APIName == "" {
		return ""
	}
	if call.RawParams != "" && strings.HasPrefix(call.APIName, "__") {
		return call.RawParams
	}

	var b strings.Builder
	if call.GCAddr != "" && call.TID != "" {
		b.WriteString("(gc=")
		b.WriteString(call.GCAddr)
		b.WriteString(", tid=")
		b.WriteString(call.TID)
		b.WriteString("): ")
	}
	b.WriteString(call.APIName)
	if call.RawParams != "" {
		b.WriteByte(' ')
		b.WriteString(call.RawParams)
	} else if call.Count > 0 || call.TimeUs > 0 {
		b.WriteString(": count=")
		b.WriteString(strconv.Itoa(normalizedCallCount(call.Count)))
		b.WriteString(", time=")
		b.WriteString(strconv.FormatInt(call.TimeUs, 10))
		b.WriteString(" us")
	}
	if call.ReturnValue != "" {
		b.WriteString(" => ")
		b.WriteString(call.ReturnValue)
	}
	return b.String()
}

func normalizedCallCount(count int) int {
	if count > 0 {
		return count
	}
	return 1
}

func stripProgramSource(program core.ProgramInfo, includeSource bool) core.ProgramInfo {
	if !includeSource {
		program.Shaders = nil
		if program.ShaderIDs == nil {
			program.ShaderIDs = []int{}
		}
		if program.FramesUsed == nil {
			program.FramesUsed = []int{}
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

func normalizeFrameProgramInsight(insight *core.FrameProgramInsight) core.FrameProgramInsight {
	normalized := *insight
	normalized.Programs = append([]core.ProgramUsage(nil), insight.Programs...)
	if normalized.Programs == nil {
		normalized.Programs = []core.ProgramUsage{}
	}
	for i := range normalized.Programs {
		if normalized.Programs[i].ShaderIDs == nil {
			normalized.Programs[i].ShaderIDs = []int{}
		}
		if normalized.Programs[i].Shaders == nil {
			normalized.Programs[i].Shaders = []core.ShaderObjectInfo{}
		}
	}
	if normalized.Segments == nil {
		normalized.Segments = []core.ProgramSegment{}
	}
	return normalized
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

	shaders := buildShaderResponses(current)

	response := ShadersResponse{
		Shaders: shaders,
		Total:   len(shaders),
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

type FrameAPICallsResponse struct {
	FrameNum int               `json:"frame_num"`
	APICalls []APICallResponse `json:"api_calls"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type FrameRawLinesResponse struct {
	FrameNum           int      `json:"frame_num"`
	Lines              []string `json:"lines"`
	Total              int      `json:"total"`
	Page               int      `json:"page"`
	PageSize           int      `json:"page_size"`
	StartLine          int      `json:"start_line"`
	EndLine            int      `json:"end_line"`
	StrippedLineNumber bool     `json:"stripped_line_number"`
}

// FrameSummary is a lightweight frame representation for list endpoints
// Avoids returning large fields like APICalls, Shaders, Programs, etc.
type FrameSummary struct {
	FrameNum         int                       `json:"frame_num"`
	StartLine        int                       `json:"start_line"`
	EndLine          int                       `json:"end_line"`
	TotalTimeUs      int64                     `json:"total_time_us"`
	SwapBufferTimeUs int64                     `json:"swap_buffer_time_us"`
	APITotalTimeUs   int64                     `json:"api_total_time_us"`
	APICount         int                       `json:"api_count"`
	DrawCallCount    int                       `json:"draw_call_count"`
	HasTiming        bool                      `json:"has_timing"`
	TimingSource     string                    `json:"timing_source"`
	StatsSource      string                    `json:"stats_source"`
	CategoryStats    []glstats.CategoryCounter `json:"category_stats,omitempty"`
	KeyAPIs          []glstats.APICounter      `json:"key_apis,omitempty"`
}

type APICallResponse struct {
	Name          string `json:"name"`
	Count         int    `json:"count"`
	TimeUs        int64  `json:"time_us"`
	LineNum       int    `json:"line_num"`
	RawParams     string `json:"raw_params,omitempty"`
	ReturnValue   string `json:"return_value,omitempty"`
	GCAddr        string `json:"gc_addr,omitempty"`
	TID           string `json:"tid,omitempty"`
	IsError       bool   `json:"is_error,omitempty"`
	ErrorCode     string `json:"error_code,omitempty"`
	HasNilPtr     bool   `json:"has_nil_ptr,omitempty"`
	Category      string `json:"category,omitempty"`
	CategoryLabel string `json:"category_label,omitempty"`
	Family        string `json:"family,omitempty"`
	Key           string `json:"key,omitempty"`
	Source        string `json:"source,omitempty"`
}

type FuncStatResponse struct {
	Name        string `json:"name"`
	CallCount   int    `json:"call_count"`
	TotalTimeUs int64  `json:"total_time_us"`
	AvgTimeUs   int64  `json:"avg_time_us"`
}

type ShaderResponse struct {
	ID          int    `json:"id"`
	Kind        string `json:"kind"`
	APIName     string `json:"api_name,omitempty"`
	Count       int    `json:"count,omitempty"`
	TimeUs      int64  `json:"time_us,omitempty"`
	AvgTimeUs   int64  `json:"avg_time_us,omitempty"`
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
	DrawCallCount    int                `json:"draw_call_count"`
	HasTiming        bool               `json:"has_timing"`
	TimingSource     string             `json:"timing_source"`
	APICalls         []APICallResponse  `json:"api_calls"`
	FuncStats        []FuncStatResponse `json:"func_stats"`
	Shaders          []ShaderResponse   `json:"shaders"`
	Programs         []int              `json:"programs"`
	BufferCreations  []core.BufferInfo  `json:"buffer_creations"`
	Stats            glstats.FrameStats `json:"stats"`
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
	HasTiming    bool    `json:"has_timing"`
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
	stats := glstats.AnalyzeFrame(frame)
	return FrameSummary{
		FrameNum:         frame.FrameNum,
		StartLine:        frame.StartLine,
		EndLine:          frame.EndLine,
		TotalTimeUs:      frame.TotalTimeUs,
		SwapBufferTimeUs: frame.SwapBufferTimeUs,
		APITotalTimeUs:   frame.APITotalTimeUs,
		APICount:         stats.APICallCount,
		DrawCallCount:    stats.DrawCallCount,
		HasTiming:        frame.HasTiming || frame.TotalTimeUs > 0,
		TimingSource:     frameTimingSource(frame),
		StatsSource:      stats.StatsSource,
		CategoryStats:    stats.CategoryStats,
		KeyAPIs:          stats.KeyAPIs,
	}
}

func toV2AIFrameSummary(stats glstats.FrameStats) V2AIFrameSummary {
	evidence := []string{
		fmt.Sprintf("frame=%d lines=%d-%d", stats.FrameNum, stats.StartLine, stats.EndLine),
		fmt.Sprintf("timing_source=%s stats_source=%s", stats.TimingSource, stats.StatsSource),
	}
	for _, api := range stats.KeyAPIs {
		if len(evidence) >= 6 {
			break
		}
		evidence = append(evidence, fmt.Sprintf("%s count=%d time_us=%d", api.APIName, api.Count, api.TimeUs))
	}
	return V2AIFrameSummary{
		FrameNum:      stats.FrameNum,
		LineRange:     fmt.Sprintf("%d-%d", stats.StartLine, stats.EndLine),
		TotalTimeUs:   stats.TotalTimeUs,
		SwapTimeUs:    stats.SwapBufferTimeUs,
		APITimeUs:     stats.APITotalTimeUs,
		APICallCount:  stats.APICallCount,
		DrawCallCount: stats.DrawCallCount,
		KeyAPIs:       stats.KeyAPIs,
		Categories:    stats.CategoryStats,
		Evidence:      evidence,
	}
}

func toAPICalls(calls []core.APILogEntry) []APICallResponse {
	result := make([]APICallResponse, 0, len(calls))
	for _, call := range calls {
		class := glstats.Classify(call.APIName)
		result = append(result, APICallResponse{
			Name:          call.APIName,
			Count:         call.Count,
			TimeUs:        call.TimeUs,
			LineNum:       call.LineNum,
			RawParams:     call.RawParams,
			ReturnValue:   call.ReturnValue,
			GCAddr:        call.GCAddr,
			TID:           call.TID,
			IsError:       call.IsError,
			ErrorCode:     call.ErrorCode,
			HasNilPtr:     call.HasNilPtr,
			Category:      class.Category,
			CategoryLabel: class.Label,
			Family:        class.Family,
			Key:           class.Key,
			Source:        "raw_sequence",
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
	source := shader.Source
	if len(source) > ShaderSourceTruncateLen {
		source = source[:ShaderSourceTruncateLen] + "\n[Source truncated]"
	}
	return ShaderResponse{
		ID:          shader.ID,
		Kind:        "source",
		CommandLine: shader.CommandLine,
		Source:      source,
	}
}

func toShaderList(shaders []*core.ShaderInfo) []ShaderResponse {
	result := make([]ShaderResponse, 0, len(shaders))
	for _, shader := range shaders {
		result = append(result, toShader(shader))
	}
	return result
}

func buildShaderResponses(log *core.ParsedLog) []ShaderResponse {
	if log == nil {
		return []ShaderResponse{}
	}
	responses := []ShaderResponse{}
	seenSource := make(map[int]bool)
	for _, frame := range log.Frames {
		for _, shader := range frame.Shaders {
			if shader == nil {
				continue
			}
			if shader.ID != 0 && seenSource[shader.ID] {
				continue
			}
			seenSource[shader.ID] = true
			responses = append(responses, toShader(shader))
		}
	}

	stats := collectShaderAPIStats(log)
	responses = append(responses, stats...)
	return responses
}

func collectShaderAPIStats(log *core.ParsedLog) []ShaderResponse {
	byAPI := make(map[string]*core.APISummary)
	for _, frame := range log.Frames {
		if len(frame.APISummary) > 0 {
			for _, summary := range frame.APISummary {
				if summary == nil || !isShaderProgramAPI(summary.APIName) {
					continue
				}
				addShaderAPIStat(byAPI, summary.APIName, summary.Count, summary.TimeUs)
			}
			continue
		}
		for _, call := range frame.APICalls {
			if !isShaderProgramAPI(call.APIName) {
				continue
			}
			count := call.Count
			if count <= 0 {
				count = 1
			}
			addShaderAPIStat(byAPI, call.APIName, count, call.TimeUs)
		}
	}

	stats := make([]*core.APISummary, 0, len(byAPI))
	for _, stat := range byAPI {
		stats = append(stats, stat)
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].TimeUs == stats[j].TimeUs {
			if stats[i].Count == stats[j].Count {
				return stats[i].APIName < stats[j].APIName
			}
			return stats[i].Count > stats[j].Count
		}
		return stats[i].TimeUs > stats[j].TimeUs
	})

	responses := make([]ShaderResponse, 0, len(stats))
	for _, stat := range stats {
		avg := int64(0)
		if stat.Count > 0 {
			avg = stat.TimeUs / int64(stat.Count)
		}
		responses = append(responses, ShaderResponse{
			Kind:      "api_stat",
			APIName:   stat.APIName,
			Count:     stat.Count,
			TimeUs:    stat.TimeUs,
			AvgTimeUs: avg,
			Source:    fmt.Sprintf("%s: count=%d, time=%d us", stat.APIName, stat.Count, stat.TimeUs),
		})
	}
	return responses
}

func addShaderAPIStat(stats map[string]*core.APISummary, apiName string, count int, timeUs int64) {
	if apiName == "" || count <= 0 {
		return
	}
	item := stats[apiName]
	if item == nil {
		stats[apiName] = &core.APISummary{APIName: apiName, Count: count, TimeUs: timeUs}
		return
	}
	item.Count += count
	item.TimeUs += timeUs
}

func isShaderProgramAPI(apiName string) bool {
	if apiName == "" {
		return false
	}
	class := glstats.Classify(apiName)
	return class.Category == glstats.CategoryShader
}

func toFrameDetail(frame core.FrameInfo) FrameDetailResponse {
	stats := glstats.AnalyzeFrame(frame)
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
		APICount:         stats.APICallCount,
		DrawCallCount:    stats.DrawCallCount,
		HasTiming:        frame.HasTiming || frame.TotalTimeUs > 0,
		TimingSource:     frameTimingSource(frame),
		APICalls:         toAPICalls(frame.APICalls),
		FuncStats:        toFuncStatsList(funcStats),
		Shaders:          toShaderList(frame.Shaders),
		Programs:         frame.Programs,
		BufferCreations:  frame.BufferCreations,
		Stats:            stats,
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
		HasTiming:   hasHandlerTiming(p.Frames),
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

func hasHandlerTiming(frames []core.FrameInfo) bool {
	for _, frame := range frames {
		if frame.HasTiming || frame.TotalTimeUs > 0 {
			return true
		}
	}
	return false
}

func frameTimingSource(frame core.FrameInfo) string {
	if frame.TimingSource != "" {
		return frame.TimingSource
	}
	if frame.TotalTimeUs > 0 {
		return "profile"
	}
	return "none"
}

func findFrame(log *core.ParsedLog, id int) (core.FrameInfo, bool) {
	if log == nil {
		return core.FrameInfo{}, false
	}
	for _, frame := range log.Frames {
		if frame.FrameNum == id {
			return frame, true
		}
	}
	return core.FrameInfo{}, false
}

func frameAPICallCount(frame core.FrameInfo) int {
	if frame.APICallCount > 0 {
		return frame.APICallCount
	}
	return len(frame.APICalls)
}

func frameDrawCallCount(frame core.FrameInfo) int {
	if frame.DrawCallCount > 0 {
		return frame.DrawCallCount
	}
	return countHandlerDrawCalls(frame)
}

func countHandlerDrawCalls(frame core.FrameInfo) int {
	total := 0
	for _, call := range frame.APICalls {
		if call.APIName == "" {
			continue
		}
		if strings.Contains(call.APIName, "DrawArrays") ||
			strings.Contains(call.APIName, "DrawElements") ||
			strings.Contains(call.APIName, "DrawRangeElements") ||
			strings.Contains(call.APIName, "DispatchCompute") {
			if call.Count > 0 {
				total += call.Count
			} else {
				total++
			}
		}
	}
	return total
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
