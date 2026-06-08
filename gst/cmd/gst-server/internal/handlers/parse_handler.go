package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gst/internal/core"
	"gst/internal/core/parser"
	"gst/internal/core/search"
)

// ParseLog handles POST /api/log/parse.
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

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, MaxMultipartSize)
		if err := r.ParseMultipartForm(MaxMultipartSize); err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "request body too large") {
				status = http.StatusRequestEntityTooLarge
			}
			http.Error(w, fmt.Sprintf("Failed to parse multipart form: %v", err), status)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get uploaded file: %v", err), http.StatusBadRequest)
			return
		}
		defer file.Close()

		filename := strings.TrimSpace(filepath.Base(r.FormValue("filename")))
		if filename == "" || filename == "." || filename == string(filepath.Separator) {
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
