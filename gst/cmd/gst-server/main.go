package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gst/cmd/gst-server/internal/handlers"
)

var (
	port        = flag.String("port", "8080", "Server port")
	openBrowser = flag.Bool("browser", true, "Open browser on startup")
	webDir      = flag.String("web-dir", "web", "Directory containing web files")
	pidFile     = flag.String("pidfile", "", "PID file path")
)

var mimeTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript; charset=utf-8",
	".json": "application/json",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".svg":  "image/svg+xml",
	".ico":  "image/x-icon",
	".woff": "font/woff",
	".woff2": "font/woff2",
	".ttf":  "font/ttf",
}

var srv *http.Server

func main() {
	flag.Parse()

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("GST Server starting on %s", addr)
	log.Printf("Serving web files from: %s", *webDir)

	// Create handler
	h := handlers.NewHandler()

	// Create mux
	mux := http.NewServeMux()

	// API routes (R1-R5)
	mux.HandleFunc("/api/log/parse", h.ParseLog)
	mux.HandleFunc("/api/log/frames", h.GetFrames)
	mux.HandleFunc("/api/log/frames/", func(w http.ResponseWriter, r *http.Request) {
		// Route /api/log/frames/:id/funcs or /api/log/frames/:id
		path := strings.TrimSuffix(r.URL.Path, "/")
		if strings.HasSuffix(path, "/funcs") {
			h.GetFrameFuncs(w, r)
		} else {
			h.GetFrameDetail(w, r)
		}
	})
	mux.HandleFunc("/api/log/search", h.Search)
	mux.HandleFunc("/api/log/analyze/top", h.AnalyzeTop)
	mux.HandleFunc("/api/log/analyze/shaders", h.AnalyzeShaders)
	mux.HandleFunc("/api/log/analyze/funcs", h.AnalyzeFuncs)
	mux.HandleFunc("/api/log/export", h.Export)
	mux.HandleFunc("/api/shutdown", handleShutdown)
	mux.HandleFunc("/health", h.Health)

	// Static files
	mux.HandleFunc("/", serveStatic)

	// Open browser if requested
	if *openBrowser {
		go func() {
			url := fmt.Sprintf("http://localhost:%s", *port)
			log.Printf("Opening browser at %s", url)
			if err := exec.Command("xdg-open", url).Start(); err != nil {
				log.Printf("Failed to open browser: %v", err)
			}
		}()
	}

	// Write PID file if requested
	if *pidFile != "" {
		if err := os.WriteFile(*pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
			log.Fatalf("Failed to write PID file: %v", err)
		}
	}

	// Start server
	srv = &http.Server{Addr: addr, Handler: mux}
	log.Printf("Server ready - visit http://localhost:%s", *port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// handleShutdown gracefully shuts down the server
func handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Println("Shutdown requested via API")
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
