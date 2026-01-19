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

// ObjectResponse wraps an object in the data response
type ObjectResponse struct {
	Object interface{} `json:"object"`
}

// ObjectsResponse wraps objects in the data response
type ObjectsResponse struct {
	Objects []interface{} `json:"objects"`
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
// Returns all objects from the queue for a specific resource kind
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

	// Get all objects from queue
	allObjects, err := redisManager.GetAllObjects()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve objects: %v", err))
		return
	}

	// Filter by resource kind
	filteredObjects := []interface{}{}
	for _, obj := range allObjects {
		objKind := getObjectKind(obj)
		if objKind == resourceKind {
			filteredObjects = append(filteredObjects, obj)
		}
	}

	response := HTTPResponse{
		Success: true,
		Data: ObjectsResponse{
			Objects: filteredObjects,
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

	// Get all objects from queue
	allObjects, err := redisManager.GetAllObjects()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve objects: %v", err))
		return
	}

	// Find object with matching Kubernetes object generation number and resource kind
	var foundObject interface{}
	for _, obj := range allObjects {
		objKind := getObjectKind(obj)
		if objKind == resourceKind {
			// Extract generation from the object's metadata
			objGen := getObjectGeneration(obj)
			if objGen == targetGeneration {
				foundObject = obj
				break
			}
		}
	}

	if foundObject == nil {
		writeErrorResponse(w, http.StatusNotFound,
			fmt.Sprintf("No object found for resource '%s' at generation number %d", resourceKind, targetGeneration))
		return
	}

	response := HTTPResponse{
		Success: true,
		Data: ObjectResponse{
			Object: foundObject,
		},
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

// getObjectKind extracts the kind from a Kubernetes object
func getObjectKind(obj interface{}) string {
	if obj == nil {
		return ""
	}

	// Try to convert to map (for unstructured objects)
	if objMap, ok := obj.(map[string]interface{}); ok {
		if kind, hasKind := objMap["kind"]; hasKind {
			if kindStr, ok := kind.(string); ok {
				return kindStr
			}
		}
	}

	// If it's an unstructured.Unstructured object
	if unstrObj, ok := obj.(interface{ Object() map[string]interface{} }); ok {
		objMap := unstrObj.Object()
		if kind, hasKind := objMap["kind"]; hasKind {
			if kindStr, ok := kind.(string); ok {
				return kindStr
			}
		}
	}

	return ""
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
