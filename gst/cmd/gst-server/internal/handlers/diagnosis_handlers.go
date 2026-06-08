package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gst/internal/core"
	"gst/internal/core/analyzer"
	"gst/internal/core/bug"
)

// HandleDiagnose handles POST /api/diagnose.
func (h *Handler) HandleDiagnose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	current, logFile, err := h.diagnosisSnapshot()
	if current == nil {
		http.Error(w, "No log parsed. Please parse a log file first.", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load diagnosis data: %v", err), http.StatusInternalServerError)
		return
	}

	registry := bug.NewDefaultRegistry()
	findings := registry.RunAll(current)
	report := bug.GenerateReport(logFile, findings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// Overview handles GET /api/overview.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	current, _ := h.overviewSnapshot()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	format := h.format
	h.mu.RUnlock()

	oa := analyzer.NewOverviewAnalyzer(current, format)
	result := oa.Analyze()
	if result == nil {
		http.Error(w, "Failed to generate overview", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toOverviewResponse(result))
}

// Workflow handles POST /api/analyze/workflow.
func (h *Handler) Workflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	current, _, err := h.diagnosisSnapshot()
	if current == nil {
		http.Error(w, "No log parsed", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load workflow data: %v", err), http.StatusInternalServerError)
		return
	}

	h.mu.RLock()
	format := h.format
	h.mu.RUnlock()

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
