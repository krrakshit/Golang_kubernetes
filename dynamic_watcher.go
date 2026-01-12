package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// WatchResource is a generic watcher for any Kubernetes resource using dynamic client
func WatchResource(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	kind string,
	pipeline *EventPipeline,
) {
	resourceName := gvr.Resource
	
	// First, list existing resources
	fmt.Printf("📋 Listing existing %s...\n", kind)
	existingResources, err := dynamicClient.Resource(gvr).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	
	if err == nil && len(existingResources.Items) > 0 {
		for _, resource := range existingResources.Items {
			fmt.Printf("   Found existing %s: %s/%s\n", 
				kind, resource.GetNamespace(), resource.GetName())
			
			resourceCopy := resource.DeepCopy()
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceKind:  kind,
				Namespace:     resourceCopy.GetNamespace(),
				Name:          resourceCopy.GetName(),
				Object:        resourceCopy,
				Timestamp:     time.Now(),
				ManagedFields: resourceCopy.GetManagedFields(),
			})
		}
	} else if err != nil {
		fmt.Printf("   ⚠️  Could not list %s: %v\n", resourceName, err)
	}

	// Now start watching for changes
	watcher, err := dynamicClient.Resource(gvr).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("⚠️  Failed to watch %s: %v\n", resourceName, err)
		return
	}
	defer watcher.Stop()

	fmt.Printf("✅ Watching %s for changes\n", kind)

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		// Send to pipeline
		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceKind:  kind,
			Namespace:     obj.GetNamespace(),
			Name:          obj.GetName(),
			Object:        obj,
			Timestamp:     time.Now(),
			ManagedFields: obj.GetManagedFields(),
		})
	}
}