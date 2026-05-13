package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gst/cmd/gst-server/internal/handlers"
	"gst/internal/platform"
)

var (
	port        = flag.String("port", "8080", "Server port")
	openBrowser = flag.Bool("browser", true, "Open browser on startup")
	webDir      = flag.String("web-dir", "web", "Directory containing web files")
	pidFile     = flag.String("pidfile", "", "PID file path")
	logLevel    = flag.String("log-level", "info", "Log level: debug, info, warn, error")
)

var mimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript; charset=utf-8",
	".json":  "application/json",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
}

var srv *http.Server

func main() {
	flag.Parse()

	platform.InitLogger(*logLevel)

	addr := fmt.Sprintf(":%s", *port)
	slog.Info("GST Server starting", "addr", addr)
	slog.Info("Serving web files", "dir", *webDir)

	// Create handler
	h := handlers.NewHandler()

	// Create mux
	mux := http.NewServeMux()

	// API routes (R1-R5)
	mux.HandleFunc("/api/log/parse", h.ParseLog)
	mux.HandleFunc("/api/log/frames", h.GetFrames)
	mux.HandleFunc("/api/log/frames/", func(w http.ResponseWriter, r *http.Request) {
		// Route /api/log/frames/:id/{funcs,programs,drawcalls} or /api/log/frames/:id
		path := strings.TrimSuffix(r.URL.Path, "/")
		switch {
		case strings.HasSuffix(path, "/funcs"):
			h.GetFrameFuncs(w, r)
		case strings.HasSuffix(path, "/programs"):
			h.GetFramePrograms(w, r)
		case strings.HasSuffix(path, "/drawcalls"):
			h.GetFrameDrawCalls(w, r)
		default:
			h.GetFrameDetail(w, r)
		}
	})
	mux.HandleFunc("/api/log/search", h.Search)
	mux.HandleFunc("/api/log/trace/programs", h.GetTracePrograms)
	mux.HandleFunc("/api/log/trace/programs/", h.GetTraceProgramDetail)
	mux.HandleFunc("/api/log/analyze/top", h.AnalyzeTop)
	mux.HandleFunc("/api/log/analyze/shaders", h.AnalyzeShaders)
	mux.HandleFunc("/api/log/analyze/funcs", h.AnalyzeFuncs)
	mux.HandleFunc("/api/log/analyze/drawcalls", h.AnalyzeDrawCalls)
	mux.HandleFunc("/api/log/analyze/textures", h.AnalyzeTextures)
	mux.HandleFunc("/api/log/analyze/bottleneck", h.AnalyzeBottleneck)
	mux.HandleFunc("/api/log/analyze/workflow", h.Workflow)
	mux.HandleFunc("/api/log/export", h.Export)
	mux.HandleFunc("/api/overview", h.Overview)
	mux.HandleFunc("/api/diagnose", h.HandleDiagnose)
	mux.HandleFunc("/api/shutdown", handleShutdown)
	mux.HandleFunc("/health", h.Health)

	// Static files
	mux.HandleFunc("/", serveStatic)

	// Open browser if requested
	if *openBrowser {
		go func() {
			url := fmt.Sprintf("http://localhost:%s", *port)
			slog.Info("Opening browser", "url", url)
			if err := exec.Command("xdg-open", url).Start(); err != nil {
				slog.Warn("Failed to open browser", "error", err)
			}
		}()
	}

	// Write PID file if requested
	if *pidFile != "" {
		absPath, err := filepath.Abs(*pidFile)
		if err != nil {
			slog.Error("Invalid PID file path", "error", err)
			os.Exit(1)
		}
		if strings.HasPrefix(absPath, "/etc/") || strings.HasPrefix(absPath, "/sys/") ||
			strings.HasPrefix(absPath, "/proc/") || strings.HasPrefix(absPath, "/dev/") {
			slog.Error("PID file path not allowed in system directory", "path", absPath)
			os.Exit(1)
		}
		if err := os.WriteFile(*pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
			slog.Error("Failed to write PID file", "path", *pidFile, "error", err)
			os.Exit(1)
		}
	}

	// Start server
	srv = &http.Server{Addr: addr, Handler: mux}
	slog.Info("Server ready", "url", fmt.Sprintf("http://localhost:%s", *port))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

// handleShutdown gracefully shuts down the server
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Info("Shutdown requested via API")
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"stopping"}`)

	// Give the response time to be sent before shutting down
	go func() {
		time.Sleep(500 * time.Millisecond)
		if srv != nil {
			srv.Shutdown(context.Background())
		}
		// Clean up PID file
		if *pidFile != "" {
			os.Remove(*pidFile)
		}
	}()
}

// serveStatic serves static files with correct MIME types
func serveStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Root -> index.html
	if path == "/" {
		path = "/index.html"
	}

	// Add web directory prefix
	filePath := filepath.Join(*webDir, path)

	// Security: prevent directory traversal
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		http.Error(w, "Invalid path", 400)
		return
	}
	absWebDir, _ := filepath.Abs(*webDir)
	if !strings.HasPrefix(absPath, absWebDir) {
		http.Error(w, "Access denied", 403)
		return
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Try index.html for SPA routing
		indexPath := filepath.Join(*webDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, r, indexPath)
			return
		}
		http.NotFound(w, r)
		return
	}

	// Determine content type
	ext := strings.ToLower(filepath.Ext(filePath))
	if ct, ok := mimeTypes[ext]; ok {
		w.Header().Set("Content-Type", ct)
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}
