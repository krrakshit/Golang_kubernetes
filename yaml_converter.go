package main

import (
	"encoding/json"
	"fmt"

	"sigs.k8s.io/yaml"
)

// CleanKubernetesObject removes only the verbose last-applied-configuration annotation
// Keeps ALL other fields: apiVersion, kind, full metadata (uid, resourceVersion, generation, etc.), spec, and status
func CleanKubernetesObject(obj interface{}) map[string]interface{} {
	// Convert to map for manipulation
	objJSON, _ := json.Marshal(obj)
	var objMap map[string]interface{}
	json.Unmarshal(objJSON, &objMap)

	// Create cleaned object - keep everything
	cleaned := make(map[string]interface{})

	// Keep apiVersion and kind
	if apiVersion, ok := objMap["apiVersion"]; ok {
		cleaned["apiVersion"] = apiVersion
	}
	if kind, ok := objMap["kind"]; ok {
		cleaned["kind"] = kind
	}

	// Keep ALL metadata fields, but remove the verbose last-applied-configuration annotation
	if metadata, ok := objMap["metadata"].(map[string]interface{}); ok {
		cleanedMetadata := make(map[string]interface{})
		
		// Copy all metadata fields
		for key, value := range metadata {
			cleanedMetadata[key] = value
		}
		
		// Remove only the verbose last-applied-configuration annotation
		if annotations, ok := cleanedMetadata["annotations"].(map[string]interface{}); ok {
			delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
			// If annotations is now empty, remove it
			if len(annotations) == 0 {
				delete(cleanedMetadata, "annotations")
			}
		}
		
		// Remove managedFields as it's very verbose (optional - comment out if you want to keep it)
		delete(cleanedMetadata, "managedFields")
		
		cleaned["metadata"] = cleanedMetadata
	}

	// Keep spec
	if spec, ok := objMap["spec"]; ok {
		cleaned["spec"] = spec
	}

	// Keep status (IMPORTANT - this was missing before!)
	if status, ok := objMap["status"]; ok {
		cleaned["status"] = status
	}

	return cleaned
}

// ConvertToYAML converts a Kubernetes object to YAML string (cleaned)
func ConvertToYAML(obj interface{}) (string, error) {
	// First clean the object
	cleanedObj := CleanKubernetesObject(obj)

	// Convert cleaned object to YAML
	yamlData, err := yaml.Marshal(cleanedObj)
	if err != nil {
		return "", fmt.Errorf("failed to convert to YAML: %w", err)
	}

	return string(yamlData), nil
}

// ConvertToYAMLWithStoredMetadata converts an object to YAML with appropriate timestamp and generation
// For generation 1: uses creationTimestamp
// For generation > 1: uses the latest modification time from managedFields
func ConvertToYAMLWithStoredMetadata(obj interface{}) (string, error) {
	// Extract generation from object
	generation := getObjectGenerationFromObject(obj)

	// Extract appropriate timestamp based on generation
	var timestamp string
	if generation == 1 {
		// For first generation, use creationTimestamp
		timestamp = getCreationTimestampFromObject(obj)
	} else {
		// For later generations, use the latest modification time
		timestamp = getModificationTimestampFromObject(obj)
	}

	// Get clean YAML
	yamlStr, err := ConvertToYAML(obj)
	if err != nil {
		return "", err
	}

	// Format with timestamp and generation
	result := fmt.Sprintf("timestamp: %s\ngeneration: %d\n---\n%s", timestamp, generation, yamlStr)
	return result, nil
}

// ConvertToYAMLMultipleWithStoredMetadata converts multiple objects to YAML with their metadata timestamps
func ConvertToYAMLMultipleWithStoredMetadata(objects []interface{}) (string, error) {
	if len(objects) == 0 {
		return "", nil
	}

	var result string
	for i, obj := range objects {
		yamlWithMeta, err := ConvertToYAMLWithStoredMetadata(obj)
		if err != nil {
			return "", err
		}

		result += yamlWithMeta
		// Add separator between objects (except last one)
		if i < len(objects)-1 {
			result += "\n"
		}
	}

	return result, nil
}

// getCreationTimestampFromObject extracts creationTimestamp from object metadata
func getCreationTimestampFromObject(obj interface{}) string {
	if obj == nil {
		return "unknown"
	}

	// Try to convert to map
	if objMap, ok := obj.(map[string]interface{}); ok {
		if metadata, ok := objMap["metadata"].(map[string]interface{}); ok {
			if ts, ok := metadata["creationTimestamp"].(string); ok && ts != "" {
				return ts
			}
		}
	}

	return "unknown"
}

// getModificationTimestampFromObject extracts the latest modification time from managedFields
func getModificationTimestampFromObject(obj interface{}) string {
	if obj == nil {
		return "unknown"
	}

	// Try to convert to map
	if objMap, ok := obj.(map[string]interface{}); ok {
		if metadata, ok := objMap["metadata"].(map[string]interface{}); ok {
			if managedFields, ok := metadata["managedFields"].([]interface{}); ok && len(managedFields) > 0 {
				// Get the last managed field entry (most recent)
				lastField := managedFields[len(managedFields)-1]
				if fieldMap, ok := lastField.(map[string]interface{}); ok {
					if time, ok := fieldMap["time"].(string); ok && time != "" {
						return time
					}
				}
			}
		}
	}

	// Fallback to creationTimestamp if managedFields not found
	return getCreationTimestampFromObject(obj)
}

// getObjectGenerationFromObject extracts generation from an object
func getObjectGenerationFromObject(obj interface{}) int64 {
	if obj == nil {
		return 0
	}

	// Try to convert to map
	if objMap, ok := obj.(map[string]interface{}); ok {
		if metadata, ok := objMap["metadata"].(map[string]interface{}); ok {
			if gen, ok := metadata["generation"]; ok {
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

	return 0
}
