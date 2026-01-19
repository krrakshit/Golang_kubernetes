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

// handleChangesByVersion handles GET /changes/<generation_no>?resource=<resource_kind>
// Returns a specific change by its Kubernetes object generation number
func handleChangesByVersion(w http.ResponseWriter, r *http.Request, redisManager *RedisManager) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resourceKind := r.URL.Query().Get("resource")
	if resourceKind == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing 'resource' query parameter. Example: /changes/2?resource=Gateway")
		return
	}

	// Extract generation number from path /changes/<generation_no>
	// r.URL.Path will be like "/changes/2"
	pathParts := strings.Split(strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/"), "/changes/"), "/")
	generationStr := pathParts[0]

	if generationStr == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing generation number in path. Example: /changes/2?resource=Gateway")
		return
	}

	targetGeneration, err := strconv.ParseInt(generationStr, 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid generation number. Must be a positive integer.")
		return
	}

	// Get all changes from queue
	allChanges, err := redisManager.GetResourceChanges("")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve changes: %v", err))
		return
	}

	// Find change with matching Kubernetes object generation number and resource kind
	var foundChange *ResourceChange
	for i := range allChanges {
		if allChanges[i].ResourceKind == resourceKind {
			// Extract generation from the object's metadata
			objGen := getObjectGeneration(allChanges[i].Object)
			if objGen == targetGeneration {
				foundChange = &allChanges[i]
				break
			}
		}
	}

	if foundChange == nil {
		writeErrorResponse(w, http.StatusNotFound,
			fmt.Sprintf("No change found for resource '%s' at generation number %d", resourceKind, targetGeneration))
		return
	}

	response := HTTPResponse{
		Success: true,
		Data:    foundChange,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getObjectGeneration extracts the generation number from a Kubernetes object
func getObjectGeneration(obj interface{}) int64 {
	if obj == nil {
		return 0
	}

	// Try to convert to map (for unstructured objects)
	if objMap, ok := obj.(map[string]interface{}); ok {
		if metadata, hasMetadata := objMap["metadata"]; hasMetadata {
			if metadataMap, ok := metadata.(map[string]interface{}); ok {
				if gen, hasGen := metadataMap["generation"]; hasGen {
					if genFloat, ok := gen.(float64); ok {
						return int64(genFloat)
					}
					if genInt, ok := gen.(int64); ok {
						return genInt
					}
					if genInt, ok := gen.(int); ok {
						return int64(genInt)
					}
				}
			}
		}
	}

	// If it's an unstructured.Unstructured object
	if unstrObj, ok := obj.(interface{ Object() map[string]interface{} }); ok {
		objMap := unstrObj.Object()
		if metadata, hasMetadata := objMap["metadata"]; hasMetadata {
			if metadataMap, ok := metadata.(map[string]interface{}); ok {
				if gen, hasGen := metadataMap["generation"]; hasGen {
					if genFloat, ok := gen.(float64); ok {
						return int64(genFloat)
					}
					if genInt, ok := gen.(int64); ok {
						return genInt
					}
					if genInt, ok := gen.(int); ok {
						return int64(genInt)
					}
				}
			}
		}
	}

	return 0
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
