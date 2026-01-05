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

// Envoy Gateway CRD GroupVersionResources
var (
	envoyProxyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "envoyproxies",
	}

	backendTrafficPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "backendtrafficpolicies",
	}

	securityPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "securitypolicies",
	}

	clientTrafficPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "clienttrafficpolicies",
	}
)

// WatchBackendTrafficPolicies watches BackendTrafficPolicy resources
func WatchBackendTrafficPolicies(dynamicClient dynamic.Interface, namespace string, pipeline *EventPipeline) {
	// First, list existing policies
	fmt.Println("📋 Listing existing BackendTrafficPolicies...")
	existingPolicies, err := dynamicClient.Resource(backendTrafficPolicyGVR).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err == nil {
		for _, policy := range existingPolicies.Items {
			fmt.Printf("   Found existing BackendTrafficPolicy: %s/%s\n", policy.GetNamespace(), policy.GetName())
			policyCopy := policy
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceType:  ResourceTypeBackendTrafficPolicy,
				Namespace:     policyCopy.GetNamespace(),
				Name:          policyCopy.GetName(),
				Object:        &policyCopy,
				Timestamp:     time.Now(),
				ManagedFields: policyCopy.GetManagedFields(),
			})
		}
	}

	// Now start watching
	watcher, err := dynamicClient.Resource(backendTrafficPolicyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceType:  ResourceTypeBackendTrafficPolicy,
			Namespace:     obj.GetNamespace(),
			Name:          obj.GetName(),
			Object:        obj,
			Timestamp:     time.Now(),
			ManagedFields: obj.GetManagedFields(),
		})
	}
}

// WatchSecurityPolicies watches SecurityPolicy resources
func WatchSecurityPolicies(dynamicClient dynamic.Interface, namespace string, pipeline *EventPipeline) {
	// First, list existing policies
	fmt.Println("📋 Listing existing SecurityPolicies...")
	existingPolicies, err := dynamicClient.Resource(securityPolicyGVR).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err == nil {
		for _, policy := range existingPolicies.Items {
			fmt.Printf("   Found existing SecurityPolicy: %s/%s\n", policy.GetNamespace(), policy.GetName())
			policyCopy := policy
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceType:  ResourceTypeSecurityPolicy,
				Namespace:     policyCopy.GetNamespace(),
				Name:          policyCopy.GetName(),
				Object:        &policyCopy,
				Timestamp:     time.Now(),
				ManagedFields: policyCopy.GetManagedFields(),
			})
		}
	}

	// Now start watching
	watcher, err := dynamicClient.Resource(securityPolicyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceType:  ResourceTypeSecurityPolicy,
			Namespace:     obj.GetNamespace(),
			Name:          obj.GetName(),
			Object:        obj,
			Timestamp:     time.Now(),
			ManagedFields: obj.GetManagedFields(),
		})
	}
}

// WatchClientTrafficPolicies watches ClientTrafficPolicy resources
func WatchClientTrafficPolicies(dynamicClient dynamic.Interface, namespace string, pipeline *EventPipeline) {
	// First, list existing policies
	fmt.Println("📋 Listing existing ClientTrafficPolicies...")
	existingPolicies, err := dynamicClient.Resource(clientTrafficPolicyGVR).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err == nil {
		for _, policy := range existingPolicies.Items {
			fmt.Printf("   Found existing ClientTrafficPolicy: %s/%s\n", policy.GetNamespace(), policy.GetName())
			policyCopy := policy
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceType:  ResourceTypeClientTrafficPolicy,
				Namespace:     policyCopy.GetNamespace(),
				Name:          policyCopy.GetName(),
				Object:        &policyCopy,
				Timestamp:     time.Now(),
				ManagedFields: policyCopy.GetManagedFields(),
			})
		}
	}

	// Now start watching
	watcher, err := dynamicClient.Resource(clientTrafficPolicyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceType:  ResourceTypeClientTrafficPolicy,
			Namespace:     obj.GetNamespace(),
			Name:          obj.GetName(),
			Object:        obj,
			Timestamp:     time.Now(),
			ManagedFields: obj.GetManagedFields(),
		})
	}
}

// WatchEnvoyProxies watches EnvoyProxy resources
func WatchEnvoyProxies(dynamicClient dynamic.Interface, namespace string, pipeline *EventPipeline) {
	fmt.Println("📋 Listing existing EnvoyProxies...")
	existingProxies, err := dynamicClient.Resource(envoyProxyGVR).Namespace(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err == nil {
		for _, proxy := range existingProxies.Items {
			fmt.Printf("   Found existing EnvoyProxy: %s/%s\n", proxy.GetNamespace(), proxy.GetName())
			proxyCopy := proxy
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceType:  ResourceTypeEnvoyProxy,
				Namespace:     proxyCopy.GetNamespace(),
				Name:          proxyCopy.GetName(),
				Object:        &proxyCopy,
				Timestamp:     time.Now(),
				ManagedFields: proxyCopy.GetManagedFields(),
			})
		}
	}

	watcher, err := dynamicClient.Resource(envoyProxyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceType:  ResourceTypeEnvoyProxy,
			Namespace:     obj.GetNamespace(),
			Name:          obj.GetName(),
			Object:        obj,
			Timestamp:     time.Now(),
			ManagedFields: obj.GetManagedFields(),
		})
	}
}