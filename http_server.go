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

// StartHTTPServer starts the HTTP server with the three main APIs
func StartHTTPServer(redisManager *RedisManager, port string) error {
	// API 1: Get resource history (generations & timestamps)
	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		handleGetResourceHistory(w, r, redisManager)
	})

	// API 2: Get specific generation YAML
	http.HandleFunc("/api/generation", func(w http.ResponseWriter, r *http.Request) {
		handleGetGenerationYAML(w, r, redisManager)
	})

	// API 3: List all resource tuples
	http.HandleFunc("/api/resources", func(w http.ResponseWriter, r *http.Request) {
		handleListAllResources(w, r, redisManager)
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
	fmt.Printf("   📍 GET /api/history?kind=<KIND>&name=<NAME>&namespace=<NS> - Get resource history\n")
	fmt.Printf("   📍 GET /api/generation?kind=<KIND>&name=<NAME>&namespace=<NS>&generation=<GEN> - Get specific generation\n")
	fmt.Printf("   📍 GET /api/resources - List all resources\n")
	fmt.Printf("   📍 GET /health - Health check\n\n")

	return http.ListenAndServe(":"+port, nil)
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

// getObjectGeneration extracts the generation number from a Kubernetes object
func getObjectGeneration(obj interface{}) int64 {
	if obj == nil {
		return 0
	}

	// First, unwrap if it's a StoredObject
	actualObj := obj
	if objMap, ok := obj.(map[string]interface{}); ok {
		if innerObj, hasObject := objMap["object"]; hasObject {
			actualObj = innerObj
		}
	}

	// Try to convert to map (for unstructured objects)
	if objMap, ok := actualObj.(map[string]interface{}); ok {
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

// getObjectTimestamp extracts the timestamp from a Kubernetes object
// Priority: 1) stored_timestamp (if wrapped), 2) managedFields[].time (most recent), 3) creationTimestamp
func getObjectTimestamp(obj interface{}) string {
	if obj == nil {
		return ""
	}

	// First, try to get stored_timestamp from StoredObject wrapper (new format)
	if objMap, ok := obj.(map[string]interface{}); ok {
		if ts, hasTS := objMap["stored_timestamp"]; hasTS {
			if tsStr, ok := ts.(string); ok {
				return tsStr
			}
		}
		
		// If not wrapped, try to unwrap and get the actual object
		actualObj := obj
		if innerObj, hasObject := objMap["object"]; hasObject {
			actualObj = innerObj
		}
		
		// Try to get timestamp from managedFields (shows when each generation was updated)
		if actualObjMap, ok := actualObj.(map[string]interface{}); ok {
			if metadata, hasMetadata := actualObjMap["metadata"]; hasMetadata {
				if metadataMap, ok := metadata.(map[string]interface{}); ok {
					// Get the most recent time from managedFields
					if managedFields, hasMF := metadataMap["managedFields"]; hasMF {
						if mfArray, ok := managedFields.([]interface{}); ok && len(mfArray) > 0 {
							// Get the last managedField entry (most recent)
							if lastMF, ok := mfArray[len(mfArray)-1].(map[string]interface{}); ok {
								if time, hasTime := lastMF["time"]; hasTime {
									if timeStr, ok := time.(string); ok {
										return timeStr
									}
								}
							}
						}
					}
					
					// Fallback to creationTimestamp
					if ts, hasTS := metadataMap["creationTimestamp"]; hasTS {
						if tsStr, ok := ts.(string); ok {
							return tsStr
						}
					}
				}
			}
		}
	}

	return ""
}

// ============================================================================
// NEW API HANDLERS
// ============================================================================

// ResourceHistoryItem represents a single history entry with generation and timestamp
type ResourceHistoryItem struct {
	Generation int64  `json:"generation"`
	Timestamp  string `json:"timestamp"`
}

// ResourceTuple represents a kind/name/namespace tuple
type ResourceTuple struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// handleGetResourceHistory handles GET /api/history?kind=<KIND>&name=<NAME>&namespace=<NAMESPACE>
// API 1: Returns list of changes (only generation & timestamp)
func handleGetResourceHistory(w http.ResponseWriter, r *http.Request, redisManager *RedisManager) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get query parameters
	kind := r.URL.Query().Get("kind")
	name := r.URL.Query().Get("name")
	namespace := r.URL.Query().Get("namespace")

	if kind == "" || name == "" || namespace == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameters: kind, name, namespace")
		return
	}

	resourceKey := fmt.Sprintf("%s/%s/%s", kind, name, namespace)

	// Get all versions of this resource
	objects, err := redisManager.GetResourceObjects(resourceKey)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve resource: %v", err))
		return
	}

	if len(objects) == 0 {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Resource not found: %s", resourceKey))
		return
	}

	// Extract generation and timestamp from each object
	history := make([]ResourceHistoryItem, 0, len(objects))
	for _, obj := range objects {
		generation := getObjectGeneration(obj)
		timestamp := getObjectTimestamp(obj)
		
		history = append(history, ResourceHistoryItem{
			Generation: generation,
			Timestamp:  timestamp,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleGetGenerationYAML handles GET /api/generation?kind=<KIND>&name=<NAME>&namespace=<NAMESPACE>&generation=<GEN>
// API 2: Returns the YAML for only the specified generation
func handleGetGenerationYAML(w http.ResponseWriter, r *http.Request, redisManager *RedisManager) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get query parameters
	kind := r.URL.Query().Get("kind")
	name := r.URL.Query().Get("name")
	namespace := r.URL.Query().Get("namespace")
	generationStr := r.URL.Query().Get("generation")

	if kind == "" || name == "" || namespace == "" || generationStr == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameters: kind, name, namespace, generation")
		return
	}

	targetGeneration, err := strconv.ParseInt(generationStr, 10, 64)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid generation number. Must be a positive integer.")
		return
	}

	resourceKey := fmt.Sprintf("%s/%s/%s", kind, name, namespace)

	// Get all versions of this resource
	objects, err := redisManager.GetResourceObjects(resourceKey)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve resource: %v", err))
		return
	}

	if len(objects) == 0 {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("Resource not found: %s", resourceKey))
		return
	}

	// Find the object with matching generation
	var foundObject interface{}
	for _, obj := range objects {
		if getObjectGeneration(obj) == targetGeneration {
			foundObject = obj
			break
		}
	}

	if foundObject == nil {
		writeErrorResponse(w, http.StatusNotFound, 
			fmt.Sprintf("Generation %d not found for resource %s", targetGeneration, resourceKey))
		return
	}

	// Unwrap the StoredObject to get the actual Kubernetes object
	actualObject := foundObject
	if objMap, ok := foundObject.(map[string]interface{}); ok {
		if innerObj, hasObject := objMap["object"]; hasObject {
			actualObject = innerObj
		}
	}

	// Convert to YAML
	yamlString, err := ConvertToYAMLWithStoredMetadata(actualObject)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to convert to YAML: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Write([]byte(yamlString))
}

// handleListAllResources handles GET /api/resources
// API 3: Returns all Kind/Name/Namespace tuples by querying keys in Redis
func handleListAllResources(w http.ResponseWriter, r *http.Request, redisManager *RedisManager) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get all resource keys
	keys, err := redisManager.GetAllResourceKeys()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve resource keys: %v", err))
		return
	}

	// Parse keys into tuples
	resources := make([]ResourceTuple, 0, len(keys))
	for _, key := range keys {
		parts := strings.Split(key, "/")
		if len(parts) == 3 {
			resources = append(resources, ResourceTuple{
				Kind:      parts[0],
				Name:      parts[1],
				Namespace: parts[2],
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// getObjectKind extracts the kind from a Kubernetes object
func getObjectKind(obj interface{}) string {
	if obj == nil {
		return ""
	}

	// First, unwrap if it's a StoredObject
	actualObj := obj
	if objMap, ok := obj.(map[string]interface{}); ok {
		if innerObj, hasObject := objMap["object"]; hasObject {
			actualObj = innerObj
		}
	}

	// Try to convert to map (for unstructured objects)
	if objMap, ok := actualObj.(map[string]interface{}); ok {
		if kind, hasKind := objMap["kind"]; hasKind {
			if kindStr, ok := kind.(string); ok {
				return kindStr
			}
		}
	}

	return ""
}
