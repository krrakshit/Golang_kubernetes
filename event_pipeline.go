package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ResourceType defines the type of Kubernetes resource
type ResourceType string

const (
	// Gateway API Resources
	ResourceTypeGateway   ResourceType = "Gateway"
	ResourceTypeHTTPRoute ResourceType = "HTTPRoute"

	// Envoy Gateway CRDs
	ResourceTypeEnvoyProxy           ResourceType = "EnvoyProxy"
	ResourceTypeBackendTrafficPolicy ResourceType = "BackendTrafficPolicy"
	ResourceTypeSecurityPolicy       ResourceType = "SecurityPolicy"
	ResourceTypeClientTrafficPolicy  ResourceType = "ClientTrafficPolicy"
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
	ResourceType  ResourceType
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
	key := fmt.Sprintf("%s/%s/%s", event.ResourceType, event.Namespace, event.Name)

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
		changes = ep.calculateChanges(oldState, event.Object, event.ResourceType)
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
func (ep *EventPipeline) calculateChanges(oldObj, newObj interface{}, resourceType ResourceType) *ChangeDetails {
	changes := &ChangeDetails{
		MetadataChanges: make(map[string]interface{}),
		SpecChanges:     make(map[string]interface{}),
		OldObject:       oldObj,
		NewObject:       newObj,
	}

	// Handle different resource types
	switch resourceType {
	case ResourceTypeGateway:
		ep.compareGateways(oldObj.(*gatewayv1.Gateway), newObj.(*gatewayv1.Gateway), changes)
	case ResourceTypeHTTPRoute:
		ep.compareHTTPRoutes(oldObj.(*gatewayv1.HTTPRoute), newObj.(*gatewayv1.HTTPRoute), changes)
	case ResourceTypeBackendTrafficPolicy, ResourceTypeSecurityPolicy, ResourceTypeClientTrafficPolicy, ResourceTypeEnvoyProxy:
		ep.compareUnstructured(oldObj.(*unstructured.Unstructured), newObj.(*unstructured.Unstructured), changes)
	}

	return changes
}

// compareGateways compares two Gateway objects
func (ep *EventPipeline) compareGateways(old, new *gatewayv1.Gateway, changes *ChangeDetails) {
	// Compare labels
	if !reflect.DeepEqual(old.Labels, new.Labels) {
		changes.MetadataChanges["labels"] = map[string]interface{}{
			"old": old.Labels,
			"new": new.Labels,
		}
	}

	// Compare annotations
	if !reflect.DeepEqual(old.Annotations, new.Annotations) {
		changes.MetadataChanges["annotations"] = map[string]interface{}{
			"old": old.Annotations,
			"new": new.Annotations,
		}
	}

	// Compare GatewayClassName
	if old.Spec.GatewayClassName != new.Spec.GatewayClassName {
		changes.SpecChanges["gatewayClassName"] = map[string]interface{}{
			"old": old.Spec.GatewayClassName,
			"new": new.Spec.GatewayClassName,
		}
	}

	// Compare Listeners
	if !reflect.DeepEqual(old.Spec.Listeners, new.Spec.Listeners) {
		changes.SpecChanges["listeners"] = map[string]interface{}{
			"old": old.Spec.Listeners,
			"new": new.Spec.Listeners,
		}
	}
}

// compareHTTPRoutes compares two HTTPRoute objects
func (ep *EventPipeline) compareHTTPRoutes(old, new *gatewayv1.HTTPRoute, changes *ChangeDetails) {
	// Compare labels
	if !reflect.DeepEqual(old.Labels, new.Labels) {
		changes.MetadataChanges["labels"] = map[string]interface{}{
			"old": old.Labels,
			"new": new.Labels,
		}
	}

	// Compare annotations
	if !reflect.DeepEqual(old.Annotations, new.Annotations) {
		changes.MetadataChanges["annotations"] = map[string]interface{}{
			"old": old.Annotations,
			"new": new.Annotations,
		}
	}

	// Compare Hostnames
	if !reflect.DeepEqual(old.Spec.Hostnames, new.Spec.Hostnames) {
		changes.SpecChanges["hostnames"] = map[string]interface{}{
			"old": old.Spec.Hostnames,
			"new": new.Spec.Hostnames,
		}
	}

	// Compare ParentRefs
	if !reflect.DeepEqual(old.Spec.ParentRefs, new.Spec.ParentRefs) {
		changes.SpecChanges["parentRefs"] = map[string]interface{}{
			"changed": true,
		}
	}

	// Compare Rules
	if !reflect.DeepEqual(old.Spec.Rules, new.Spec.Rules) {
		changes.SpecChanges["rules"] = map[string]interface{}{
			"changed": true,
		}
	}
}

// compareUnstructured compares two Unstructured objects (for Envoy Gateway CRDs)
func (ep *EventPipeline) compareUnstructured(old, new *unstructured.Unstructured, changes *ChangeDetails) {
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
}

// logEvent logs the event to console
func (ep *EventPipeline) logEvent(event ResourceEvent, changes *ChangeDetails) {
	fmt.Printf("\n📌 EVENT: %s | %s: %s/%s (at %s)\n",
		event.Type,
		event.ResourceType,
		event.Namespace,
		event.Name,
		event.Timestamp.Format("15:04:05"),
	)

	if event.Type == EventTypeModified {
		if len(changes.MetadataChanges) > 0 {
			fmt.Println("   🔍 METADATA CHANGES:")
			for key, value := range changes.MetadataChanges {
				fmt.Printf("      📝 %s changed\n", key)
				if m, ok := value.(map[string]interface{}); ok {
					if old, ok := m["old"]; ok {
						fmt.Printf("         Old: %v\n", old)
					}
					if new, ok := m["new"]; ok {
						fmt.Printf("         New: %v\n", new)
					}
				}
			}
		}

		if len(changes.SpecChanges) > 0 {
			fmt.Println("   🔍 SPEC CHANGES:")
			for key := range changes.SpecChanges {
				fmt.Printf("      📝 %s changed\n", key)
			}
		}

		if len(changes.MetadataChanges) == 0 && len(changes.SpecChanges) == 0 {
			fmt.Println("   ℹ️  No significant changes detected")
		}
	} else if event.Type == EventTypeAdded {
		fmt.Println("   → New resource created")
	} else if event.Type == EventTypeDeleted {
		fmt.Println("   → Resource deleted")
	}

	fmt.Println("-----------------------------------------------------")
}

// deepCopyObject creates a deep copy of an object
func (ep *EventPipeline) deepCopyObject(obj interface{}) interface{} {
	switch v := obj.(type) {
	case *gatewayv1.Gateway:
		return v.DeepCopy()
	case *gatewayv1.HTTPRoute:
		return v.DeepCopy()
	case *unstructured.Unstructured:
		return v.DeepCopy()
	default:
		return obj
	}
}