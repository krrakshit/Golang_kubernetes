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
// If namespaces is empty, watches across all namespaces
func WatchResource(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespaces []string,
	kind string,
	pipeline *EventPipeline,
) {
	// If no namespaces specified, watch all namespaces
	if len(namespaces) == 0 {
		watchAllNamespaces(dynamicClient, gvr, kind, pipeline)
		return
	}

	// Watch each specified namespace
	for _, namespace := range namespaces {
		go watchNamespace(dynamicClient, gvr, namespace, kind, pipeline)
	}
}

// watchNamespace watches resources in a specific namespace
func watchNamespace(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	namespace string,
	kind string,
	pipeline *EventPipeline,
) {
	resourceName := gvr.Resource

	// First, list existing resources
	fmt.Printf("📋 Listing existing %s in namespace %s...\n", kind, namespace)
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
		fmt.Printf("⚠️  Failed to watch %s in namespace %s: %v\n", resourceName, namespace, err)
		return
	}
	defer watcher.Stop()

	fmt.Printf("✅ Watching %s in namespace %s for changes\n", kind, namespace)

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		fmt.Printf("debugging  %v\n", obj)

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

// watchAllNamespaces watches resources across all namespaces
func watchAllNamespaces(
	dynamicClient dynamic.Interface,
	gvr schema.GroupVersionResource,
	kind string,
	pipeline *EventPipeline,
) {
	resourceName := gvr.Resource

	// First, list existing resources across all namespaces
	fmt.Printf("📋 Listing existing %s across all namespaces...\n", kind)
	existingResources, err := dynamicClient.Resource(gvr).List(
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

	// Now start watching for changes across all namespaces
	watcher, err := dynamicClient.Resource(gvr).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("⚠️  Failed to watch %s across all namespaces: %v\n", resourceName, err)
		return
	}
	defer watcher.Stop()

	fmt.Printf("✅ Watching %s across all namespaces for changes\n", kind)

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		fmt.Printf("Event: %v\n", event)

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
