package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// HTTPResponse is a generic response wrapper
type HTTPResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ChangesResponse represents the response for /changes endpoint
type ChangesResponse struct {
	Resource string           `json:"resource"`
	Count    int              `json:"count"`
	Changes  []ResourceChange `json:"changes"`
}

// StartHTTPServer starts the HTTP server with change endpoints
func StartHTTPServer(redisManager *RedisManager, port string) error {
	http.HandleFunc("/changes", func(w http.ResponseWriter, r *http.Request) {
		handleChanges(w, r, redisManager)
	})

	http.HandleFunc("/changes/", func(w http.ResponseWriter, r *http.Request) {
		handleChangesByVersion(w, r, redisManager)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HTTPResponse{
			Success: true,
			Message: "Server is healthy",
		})
	})

	fmt.Printf("🌐 HTTP Server starting on :%s\n", port)
	fmt.Printf("   📍 GET /changes?resource=<KIND> - Get all changes for a resource\n")
	fmt.Printf("   📍 GET /changes/<VERSION>?resource=<KIND> - Get specific version\n")
	fmt.Printf("   📍 GET /health - Health check\n\n")

	return http.ListenAndServe(":"+port, nil)
}

// handleChanges handles GET /changes?resource=<resource_kind>
// Returns all changes from the queue for a specific resource kind
func handleChanges(w http.ResponseWriter, r *http.Request, redisManager *RedisManager) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resourceKind := r.URL.Query().Get("resource")
	if resourceKind == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing 'resource' query parameter. Example: /changes?resource=Gateway")
		return
	}

	// Get all changes from queue
	allChanges, err := redisManager.GetResourceChanges("")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve changes: %v", err))
		return
	}

	// Filter by resource kind
	filteredChanges := []ResourceChange{}
	for _, change := range allChanges {
		if change.ResourceKind == resourceKind {
			filteredChanges = append(filteredChanges, change)
		}
	}

	response := HTTPResponse{
		Success: true,
		Data: ChangesResponse{
			Resource: resourceKind,
			Count:    len(filteredChanges),
			Changes:  filteredChanges,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChangesByVersion handles GET /changes/<version>?resource=<resource_kind>
// Returns a specific change by its version number
func handleChangesByVersion(w http.ResponseWriter, r *http.Request, redisManager *RedisManager) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resourceKind := r.URL.Query().Get("resource")
	if resourceKind == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing 'resource' query parameter. Example: /changes/1?resource=Gateway")
		return
	}

	// Extract version from path /changes/<version>
	// r.URL.Path will be like "/changes/1"
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[2] == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing version number in path. Example: /changes/1?resource=Gateway")
		return
	}

	version, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid version number. Must be a positive integer.")
		return
	}

	// Get all changes from queue
	allChanges, err := redisManager.GetResourceChanges("")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve changes: %v", err))
		return
	}

	// Find change with matching version and resource kind
	var foundChange *ResourceChange
	for i := range allChanges {
		if allChanges[i].Version == version && allChanges[i].ResourceKind == resourceKind {
			foundChange = &allChanges[i]
			break
		}
	}

	if foundChange == nil {
		writeErrorResponse(w, http.StatusNotFound,
			fmt.Sprintf("Change not found for resource '%s' at version %d", resourceKind, version))
		return
	}

	response := HTTPResponse{
		Success: true,
		Data:    foundChange,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes a formatted error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(HTTPResponse{
		Success: false,
		Error:   message,
	})
}
