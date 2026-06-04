package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gst/internal/core"
	"gst/internal/core/analyzer"
	"gst/internal/core/exporter"
	"gst/internal/core/search"
)

// Export handles POST /api/log/export.
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
