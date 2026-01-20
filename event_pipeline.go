package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// EventType defines the type of event
type EventType string

const (
	EventTypeAdded    EventType = "ADDED"
	EventTypeModified EventType = "MODIFIED"
	EventTypeDeleted  EventType = "DELETED"
)

// ResourceEvent represents a standardized event from any watcher
type ResourceEvent struct {
	Type          EventType
	ResourceKind  string // Changed from ResourceType to string
	Namespace     string
	Name          string
	Object        interface{}
	Timestamp     time.Time
	ManagedFields []metav1.ManagedFieldsEntry
}

// ChangeDetails represents the details of what changed
type ChangeDetails struct {
	MetadataChanges map[string]interface{} // labels, annotations, etc.
	SpecChanges     map[string]interface{} // spec field changes
	OldObject       interface{}
	NewObject       interface{}
}

// EventPipeline manages the event processing pipeline
type EventPipeline struct {
	eventChannel   chan ResourceEvent
	previousStates map[string]interface{} // unified state storage
	stateMutex     sync.RWMutex
	changeHandlers []ChangeHandler
	redisManager   *RedisManager
}

// ChangeHandler is a function that handles change events
type ChangeHandler func(event ResourceEvent, changes *ChangeDetails)

// NewEventPipeline creates a new event pipeline
func NewEventPipeline(bufferSize int, redisManager *RedisManager) *EventPipeline {
	return &EventPipeline{
		eventChannel:   make(chan ResourceEvent, bufferSize),
		previousStates: make(map[string]interface{}),
		changeHandlers: make([]ChangeHandler, 0),
		redisManager:   redisManager,
	}
}

// RegisterHandler registers a change handler
func (ep *EventPipeline) RegisterHandler(handler ChangeHandler) {
	ep.changeHandlers = append(ep.changeHandlers, handler)
}

// SendEvent sends an event to the pipeline
func (ep *EventPipeline) SendEvent(event ResourceEvent) {
	ep.eventChannel <- event
}

// Start starts the event processing pipeline
func (ep *EventPipeline) Start() {
	fmt.Println("🚀 Event Pipeline Started - Processing events...\n")

	for event := range ep.eventChannel {
		ep.processEvent(event)
	}
}

// processEvent processes a single event
func (ep *EventPipeline) processEvent(event ResourceEvent) {
	// Generate unique key for this resource
	key := fmt.Sprintf("%s/%s/%s", event.ResourceKind, event.Namespace, event.Name)

	// Check if this is a metadata/spec change
	if !ep.hasRelevantChanges(event) && event.Type != EventTypeAdded {
		return // Skip status-only changes
	}

	// Get previous state
	ep.stateMutex.RLock()
	oldState := ep.previousStates[key]
	ep.stateMutex.RUnlock()

	// Calculate changes
	var changes *ChangeDetails
	if event.Type == EventTypeModified && oldState != nil {
		changes = ep.calculateChanges(oldState, event.Object)
	} else {
		changes = &ChangeDetails{
			MetadataChanges: make(map[string]interface{}),
			SpecChanges:     make(map[string]interface{}),
			NewObject:       event.Object,
		}
	}

	// Store full object changes to Redis with versioning
	ep.storeVersionedResourceChange(event, oldState, changes)

	// Call all registered handlers
	for _, handler := range ep.changeHandlers {
		handler(event, changes)
	}

	// Update state
	ep.stateMutex.Lock()
	ep.previousStates[key] = ep.deepCopyObject(event.Object)
	ep.stateMutex.Unlock()
}

// hasRelevantChanges checks if event has metadata or spec changes
func (ep *EventPipeline) hasRelevantChanges(event ResourceEvent) bool {
	for _, mf := range event.ManagedFields {
		if mf.FieldsV1 == nil {
			continue
		}

		var fields map[string]interface{}
		if err := json.Unmarshal(mf.FieldsV1.Raw, &fields); err != nil {
			continue
		}

		for key := range fields {
			if key == "f:metadata" || key == "f:spec" {
				return true
			}
		}
	}
	return false
}

// calculateChanges calculates what changed between old and new objects
func (ep *EventPipeline) calculateChanges(oldObj, newObj interface{}) *ChangeDetails {
	changes := &ChangeDetails{
		MetadataChanges: make(map[string]interface{}),
		SpecChanges:     make(map[string]interface{}),
		OldObject:       oldObj,
		NewObject:       newObj,
	}

	// Everything is unstructured
	old := oldObj.(*unstructured.Unstructured)
	new := newObj.(*unstructured.Unstructured)

	// Compare labels
	if !reflect.DeepEqual(old.GetLabels(), new.GetLabels()) {
		changes.MetadataChanges["labels"] = map[string]interface{}{
			"old": old.GetLabels(),
			"new": new.GetLabels(),
		}
	}

	// Compare annotations
	if !reflect.DeepEqual(old.GetAnnotations(), new.GetAnnotations()) {
		changes.MetadataChanges["annotations"] = map[string]interface{}{
			"old": old.GetAnnotations(),
			"new": new.GetAnnotations(),
		}
	}

	// Compare spec
	oldSpec, _, _ := unstructured.NestedMap(old.Object, "spec")
	newSpec, _, _ := unstructured.NestedMap(new.Object, "spec")

	if !reflect.DeepEqual(oldSpec, newSpec) {
		changes.SpecChanges["spec"] = map[string]interface{}{
			"old": oldSpec,
			"new": newSpec,
		}
	}

	return changes
}

// getObjectNameNamespace extracts name and namespace from a Kubernetes object
func getObjectNameNamespace(obj interface{}) (string, string) {
	if obj == nil {
		return "", ""
	}
	if objMap, ok := obj.(map[string]interface{}); ok {
		if metadata, ok := objMap["metadata"].(map[string]interface{}); ok {
			name, _ := metadata["name"].(string)
			namespace, _ := metadata["namespace"].(string)
			return name, namespace
		}
	}
	return "", ""
}

// storeVersionedResourceChange stores the full object directly in Redis queue
// Only stores if the object's generation has changed
func (ep *EventPipeline) storeVersionedResourceChange(event ResourceEvent, oldObj interface{}, changes *ChangeDetails) {
	if ep.redisManager == nil {
		return
	}

	// Check if generation has changed
	newGen := getObjectGenerationFromEvent(event.Object)
	oldGen := getObjectGenerationFromEvent(oldObj)

	resourceKey := fmt.Sprintf("%s/%s/%s", event.ResourceKind, event.Namespace, event.Name)

	// Debug logging
	fmt.Printf("📊 Generation Check - Resource: %s | Old Gen: %d | New Gen: %d\n", resourceKey, oldGen, newGen)

	// Only store if generation changed or if this is a new object
	if oldObj != nil && newGen == oldGen {
		fmt.Printf("⏭️  Skipping - Generation unchanged (still %d)\n\n", newGen)
		return // Skip storing if generation hasn't changed
	}

	// Deduplication: check Redis for same resource/generation
	allObjects, _ := ep.redisManager.GetAllObjects()
	for _, obj := range allObjects {
		objKind := getObjectKind(obj)
		objGen := getObjectGenerationFromEvent(obj)
		name, ns := getObjectNameNamespace(obj)
		if objKind == event.ResourceKind && objGen == newGen && name == event.Name && ns == event.Namespace {
			fmt.Printf("⏭️  Skipping - Duplicate in Redis for %s gen %d\n\n", resourceKey, newGen)
			return
		}
	}

	// Push object directly to queue
	if newGen > 0 {
		fmt.Printf("✅ Storing object with generation %d\n\n", newGen)
		if err := ep.redisManager.PushObject(resourceKey, event.Object); err != nil {
			fmt.Printf("⚠️  Failed to store object in queue: %v\n", err)
		}
	} else {
		fmt.Printf("ℹ️  No generation found, storing anyway\n\n")
		if err := ep.redisManager.PushObject(resourceKey, event.Object); err != nil {
			fmt.Printf("⚠️  Failed to store object in queue: %v\n", err)
		}
	}
}

// getObjectGenerationFromEvent extracts generation number from an object
func getObjectGenerationFromEvent(obj interface{}) int64 {
	if obj == nil {
		return 0
	}

	// First try as unstructured.Unstructured (most common case)
	if unstr, ok := obj.(*unstructured.Unstructured); ok {
		objMap := unstr.UnstructuredContent()
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
		return 0
	}

	// Try to convert to map (for other object types)
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

	return 0
}

// deepCopyObject creates a deep copy of an object
func (ep *EventPipeline) deepCopyObject(obj interface{}) interface{} {
	if unstr, ok := obj.(*unstructured.Unstructured); ok {
		return unstr.DeepCopy()
	}
	return obj
}
