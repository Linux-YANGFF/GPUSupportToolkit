package handlers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gst/internal/core"
	"gst/internal/core/parser"
	"gst/internal/core/search"
)

const (
	DefaultBufferSize            = 10 * 1024 * 1024
	MaxMultipartSize             = 100 << 20
	ShaderSourceTruncateLen      = 2000
	TraceShaderSourceTruncateLen = 50000
)

// Handler is the HTTP handler with shared current-case state.
type Handler struct {
	mu         sync.RWMutex
	logFile    string
	rawLogPath string
	current    *core.ParsedLog
	lines      []string
	index      *search.KeywordIndex
	format     string
	traceCache *core.TraceAnalysis
}

// NewHandler creates a new Handler.
func NewHandler() *Handler {
	return &Handler{
		index: search.NewKeywordIndex(),
	}
}

func (h *Handler) diagnosisSnapshot() (*core.ParsedLog, string, error) {
	h.mu.RLock()
	current := h.current
	logFile := h.logFile
	rawLogPath := h.rawLogPath
	h.mu.RUnlock()

	if current == nil {
		return nil, logFile, nil
	}

	snapshot := *current
	snapshot.Frames = append([]core.FrameInfo(nil), current.Frames...)
	if snapshot.SourcePath == "" {
		snapshot.SourcePath = rawLogPath
	}
	if err := parser.HydrateIndexedAPICalls(&snapshot); err != nil {
		return nil, logFile, err
	}
	return &snapshot, logFile, nil
}

func (h *Handler) overviewSnapshot() (*core.ParsedLog, string) {
	h.mu.RLock()
	current := h.current
	logFile := h.logFile
	rawLogPath := h.rawLogPath
	h.mu.RUnlock()

	if current == nil {
		return nil, logFile
	}

	snapshot := *current
	snapshot.Frames = append([]core.FrameInfo(nil), current.Frames...)
	if snapshot.SourcePath == "" {
		snapshot.SourcePath = rawLogPath
	}
	return &snapshot, logFile
}

// validateLogPath restricts local-path parsing to configured log roots.
func validateLogPath(path string) error {
	if path == "" {
		return errors.New("path is empty")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %v", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %v", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path is not a regular file")
	}

	if os.Getenv("GST_ALLOW_ANY_LOG_PATH") == "1" {
		return nil
	}

	allowedDirs, err := allowedLogDirs()
	if err != nil {
		return err
	}
	for _, allowed := range allowedDirs {
		if pathWithinDir(absPath, allowed) {
			return nil
		}
	}
	return fmt.Errorf("path is outside allowed log directories (%s)", strings.Join(allowedDirs, string(os.PathListSeparator)))
}

func allowedLogDirs() ([]string, error) {
	raw := os.Getenv("GST_LOG_DIR")
	if raw == "" {
		raw = "."
	}

	parts := strings.Split(raw, string(os.PathListSeparator))
	allowed := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		abs, err := filepath.Abs(part)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve allowed directory %q: %v", part, err)
		}
		allowed = append(allowed, abs)
	}
	if len(allowed) == 0 {
		return nil, errors.New("no allowed log directories configured")
	}
	return allowed, nil
}

func pathWithinDir(absPath, absDir string) bool {
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

// buildLinesFromLog builds raw lines from parsed logs for non-indexed search/export.
func buildLinesFromLog(log *core.ParsedLog) []string {
	if log == nil {
		return []string{}
	}
	var lines []string
	for _, frame := range log.Frames {
		for _, call := range frame.APICalls {
			line := fmt.Sprintf("%s: count=%d, time=%d", call.APIName, call.Count, call.TimeUs)
			lines = append(lines, line)
		}
	}
	return lines
}
