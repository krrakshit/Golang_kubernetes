package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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
}

// ChangeHandler is a function that handles change events
type ChangeHandler func(event ResourceEvent, changes *ChangeDetails)

// NewEventPipeline creates a new event pipeline
func NewEventPipeline(bufferSize int) *EventPipeline {	
	return &EventPipeline{
		eventChannel:   make(chan ResourceEvent, bufferSize),
		previousStates: make(map[string]interface{}),
		changeHandlers: make([]ChangeHandler, 0),
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

	// Log the event
	ep.logEvent(event, changes)

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

// logEvent logs the event to console
// logEvent logs the event to console with pretty JSON formatting
func (ep *EventPipeline) logEvent(event ResourceEvent, changes *ChangeDetails) {
	fmt.Printf("\n📌 EVENT: %s | %s: %s/%s (at %s)\n",
		event.Type,
		event.ResourceKind,
		event.Namespace,
		event.Name,
		event.Timestamp.Format("15:04:05"),
	)

	if event.Type == EventTypeModified {
		hasChanges := false

		// Log metadata changes with pretty JSON
		if len(changes.MetadataChanges) > 0 {
			hasChanges = true
			fmt.Println("\n   🔍 METADATA CHANGES:")
			
			for key, value := range changes.MetadataChanges {
				if changeMap, ok := value.(map[string]interface{}); ok {
					fmt.Printf("      📝 %s\n", key)
					formatted := FormatMapChange(changeMap)
					fmt.Print(formatted)
				} else {
					fmt.Printf("      📝 %s: %v\n", key, value)
				}
			}
		}

		// Log spec changes with pretty JSON
		if len(changes.SpecChanges) > 0 {
			hasChanges = true
			fmt.Println("\n   🔍 SPEC CHANGES:")
			
			for key, value := range changes.SpecChanges {
				if changeMap, ok := value.(map[string]interface{}); ok {
					fmt.Printf("      📝 %s\n", key)
					formatted := FormatMapChange(changeMap)
					fmt.Print(formatted)
				} else {
					fmt.Printf("      📝 %s: %v\n", key, value)
				}
			}
		}

		if !hasChanges {
			fmt.Println("\n   ℹ️  No significant changes detected")
		}
	} else if event.Type == EventTypeAdded {
		fmt.Println("\n   → New resource created")
		
		// Optionally show the full spec for new resources
		if obj, ok := event.Object.(*unstructured.Unstructured); ok {
			if spec, found, _ := unstructured.NestedMap(obj.Object, "spec"); found {
				fmt.Println("\n   📋 Resource Spec:")
				specJSON := MarshalToPrettyJSON(spec)
				// Truncate if too long
				specJSON = TruncateJSON(specJSON, 20)
				fmt.Print(IndentJSON(specJSON, 6))
				fmt.Println()
			}
		}
	} else if event.Type == EventTypeDeleted {
		fmt.Println("\n   → Resource deleted")
	}

	fmt.Println("\n" + strings.Repeat("-", 80))
}

// deepCopyObject creates a deep copy of an object
func (ep *EventPipeline) deepCopyObject(obj interface{}) interface{} {
	if unstr, ok := obj.(*unstructured.Unstructured); ok {
		return unstr.DeepCopy()
	}
	return obj
}