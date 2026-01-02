package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

var (
	previousEnvoyProxies           = make(map[string]*unstructured.Unstructured)
	previousBackendTrafficPolicies = make(map[string]*unstructured.Unstructured)
	previousSecurityPolicies       = make(map[string]*unstructured.Unstructured)
	previousClientTrafficPolicies  = make(map[string]*unstructured.Unstructured)
	envoyMu                        sync.RWMutex
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

	envoyPatchPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "envoypatchpolicies",
	}
)

// hasEnvoyMetadataOrSpecChanges checks for metadata/spec changes
func hasEnvoyMetadataOrSpecChanges(mf metav1.ManagedFieldsEntry) bool {
	if mf.FieldsV1 == nil {
		return false
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(mf.FieldsV1.Raw, &fields); err != nil {
		return false
	}

	for key := range fields {
		if strings.HasPrefix(key, "f:metadata") || strings.HasPrefix(key, "f:spec") {
			return true
		}
	}

	return false
}

// WatchEnvoyProxies watches EnvoyProxy resources
func WatchEnvoyProxies(dynamicClient dynamic.Interface, namespace string) {
	fmt.Println("\n🔧 Watching EnvoyProxy resources for changes...\n")

	watcher, err := dynamicClient.Resource(envoyProxyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("⚠️  EnvoyProxy watching failed (CRD may not be installed): %v\n", err)
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		hasRelevantChanges := false
		managedFields := obj.GetManagedFields()
		for _, mf := range managedFields {
			if hasEnvoyMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != watch.Added {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | EnvoyProxy: %s (namespace: %s)\n",
			event.Type, obj.GetName(), obj.GetNamespace())

		envoyMu.RLock()
		oldObj := previousEnvoyProxies[obj.GetNamespace()+"/"+obj.GetName()]
		envoyMu.RUnlock()

		if event.Type == watch.Modified && oldObj != nil {
			compareEnvoyProxyChanges(oldObj, obj)
		} else if event.Type == watch.Added {
			fmt.Println("   → New EnvoyProxy created")
			displayEnvoyProxyInfo(obj)
		} else if event.Type == watch.Deleted {
			fmt.Println("   → EnvoyProxy deleted")
		}

		envoyMu.Lock()
		objCopy := obj.DeepCopy()
		previousEnvoyProxies[obj.GetNamespace()+"/"+obj.GetName()] = objCopy
		envoyMu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// WatchBackendTrafficPolicies watches BackendTrafficPolicy resources
func WatchBackendTrafficPolicies(dynamicClient dynamic.Interface, namespace string) {
	fmt.Println("\n📋 Watching BackendTrafficPolicy resources for changes...\n")

	watcher, err := dynamicClient.Resource(backendTrafficPolicyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("⚠️  BackendTrafficPolicy watching failed: %v\n", err)
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		hasRelevantChanges := false
		managedFields := obj.GetManagedFields()
		for _, mf := range managedFields {
			if hasEnvoyMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != watch.Added {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | BackendTrafficPolicy: %s (namespace: %s)\n",
			event.Type, obj.GetName(), obj.GetNamespace())

		envoyMu.RLock()
		oldObj := previousBackendTrafficPolicies[obj.GetNamespace()+"/"+obj.GetName()]
		envoyMu.RUnlock()

		if event.Type == watch.Modified && oldObj != nil {
			compareUnstructuredChanges(oldObj, obj, "BackendTrafficPolicy")
		} else if event.Type == watch.Added {
			fmt.Println("   → New BackendTrafficPolicy created")
			displayBackendTrafficPolicyInfo(obj)
		} else if event.Type == watch.Deleted {
			fmt.Println("   → BackendTrafficPolicy deleted")
		}

		envoyMu.Lock()
		objCopy := obj.DeepCopy()
		previousBackendTrafficPolicies[obj.GetNamespace()+"/"+obj.GetName()] = objCopy
		envoyMu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// WatchSecurityPolicies watches SecurityPolicy resources
func WatchSecurityPolicies(dynamicClient dynamic.Interface, namespace string) {
	fmt.Println("\n🔒 Watching SecurityPolicy resources for changes...\n")

	watcher, err := dynamicClient.Resource(securityPolicyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("⚠️  SecurityPolicy watching failed: %v\n", err)
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		hasRelevantChanges := false
		managedFields := obj.GetManagedFields()
		for _, mf := range managedFields {
			if hasEnvoyMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != watch.Added {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | SecurityPolicy: %s (namespace: %s)\n",
			event.Type, obj.GetName(), obj.GetNamespace())

		envoyMu.RLock()
		oldObj := previousSecurityPolicies[obj.GetNamespace()+"/"+obj.GetName()]
		envoyMu.RUnlock()

		if event.Type == watch.Modified && oldObj != nil {
			compareUnstructuredChanges(oldObj, obj, "SecurityPolicy")
		} else if event.Type == watch.Added {
			fmt.Println("   → New SecurityPolicy created")
			displaySecurityPolicyInfo(obj)
		} else if event.Type == watch.Deleted {
			fmt.Println("   → SecurityPolicy deleted")
		}

		envoyMu.Lock()
		objCopy := obj.DeepCopy()
		previousSecurityPolicies[obj.GetNamespace()+"/"+obj.GetName()] = objCopy
		envoyMu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// WatchClientTrafficPolicies watches ClientTrafficPolicy resources
func WatchClientTrafficPolicies(dynamicClient dynamic.Interface, namespace string) {
	fmt.Println("\n👥 Watching ClientTrafficPolicy resources for changes...\n")

	watcher, err := dynamicClient.Resource(clientTrafficPolicyGVR).Namespace(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		fmt.Printf("⚠️  ClientTrafficPolicy watching failed: %v\n", err)
		return
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		obj, ok := event.Object.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		hasRelevantChanges := false
		managedFields := obj.GetManagedFields()
		for _, mf := range managedFields {
			if hasEnvoyMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != watch.Added {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | ClientTrafficPolicy: %s (namespace: %s)\n",
			event.Type, obj.GetName(), obj.GetNamespace())

		envoyMu.RLock()
		oldObj := previousClientTrafficPolicies[obj.GetNamespace()+"/"+obj.GetName()]
		envoyMu.RUnlock()

		if event.Type == watch.Modified && oldObj != nil {
			compareUnstructuredChanges(oldObj, obj, "ClientTrafficPolicy")
		} else if event.Type == watch.Added {
			fmt.Println("   → New ClientTrafficPolicy created")
			displayClientTrafficPolicyInfo(obj)
		} else if event.Type == watch.Deleted {
			fmt.Println("   → ClientTrafficPolicy deleted")
		}

		envoyMu.Lock()
		objCopy := obj.DeepCopy()
		previousClientTrafficPolicies[obj.GetNamespace()+"/"+obj.GetName()] = objCopy
		envoyMu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// compareEnvoyProxyChanges compares EnvoyProxy changes
func compareEnvoyProxyChanges(oldObj, newObj *unstructured.Unstructured) {
	fmt.Println("\n   🔍 ENVOYPROXY CHANGES:")

	changesFound := false

	// Compare labels
	if !reflect.DeepEqual(oldObj.GetLabels(), newObj.GetLabels()) {
		changesFound = true
		fmt.Println("   📝 Labels changed")
	}

	// Compare spec
	oldSpec, _, _ := unstructured.NestedMap(oldObj.Object, "spec")
	newSpec, _, _ := unstructured.NestedMap(newObj.Object, "spec")

	if !reflect.DeepEqual(oldSpec, newSpec) {
		changesFound = true
		fmt.Println("   📝 Spec changed")

		// Check specific fields
		oldProvider, _, _ := unstructured.NestedString(oldObj.Object, "spec", "provider", "type")
		newProvider, _, _ := unstructured.NestedString(newObj.Object, "spec", "provider", "type")
		if oldProvider != newProvider {
			fmt.Printf("      Provider Type: %s → %s\n", oldProvider, newProvider)
		}
	}

	if !changesFound {
		fmt.Println("      (no significant changes detected)")
	}
}

// compareUnstructuredChanges compares generic unstructured objects
func compareUnstructuredChanges(oldObj, newObj *unstructured.Unstructured, resourceType string) {
	fmt.Printf("\n   🔍 %s CHANGES:\n", strings.ToUpper(resourceType))

	changesFound := false

	// Compare labels
	if !reflect.DeepEqual(oldObj.GetLabels(), newObj.GetLabels()) {
		changesFound = true
		fmt.Println("   📝 Labels changed")
	}

	// Compare annotations
	if !reflect.DeepEqual(oldObj.GetAnnotations(), newObj.GetAnnotations()) {
		changesFound = true
		fmt.Println("   📝 Annotations changed")
	}

	// Compare spec
	oldSpec, _, _ := unstructured.NestedMap(oldObj.Object, "spec")
	newSpec, _, _ := unstructured.NestedMap(newObj.Object, "spec")

	if !reflect.DeepEqual(oldSpec, newSpec) {
		changesFound = true
		fmt.Println("   📝 Spec changed")
	}

	if !changesFound {
		fmt.Println("      (no significant changes detected)")
	}
}

// Display functions
func displayEnvoyProxyInfo(obj *unstructured.Unstructured) {
	provider, _, _ := unstructured.NestedString(obj.Object, "spec", "provider", "type")
	if provider != "" {
		fmt.Printf("   Provider Type: %s\n", provider)
	}
}

func displayBackendTrafficPolicyInfo(obj *unstructured.Unstructured) {
	targetRef, _, _ := unstructured.NestedMap(obj.Object, "spec", "targetRef")
	if targetRef != nil {
		if kind, ok := targetRef["kind"].(string); ok {
			name, _ := targetRef["name"].(string)
			fmt.Printf("   Target: %s/%s\n", kind, name)
		}
	}
}

func displaySecurityPolicyInfo(obj *unstructured.Unstructured) {
	targetRef, _, _ := unstructured.NestedMap(obj.Object, "spec", "targetRef")
	if targetRef != nil {
		if kind, ok := targetRef["kind"].(string); ok {
			name, _ := targetRef["name"].(string)
			fmt.Printf("   Target: %s/%s\n", kind, name)
		}
	}

	// Check for CORS settings
	cors, found, _ := unstructured.NestedMap(obj.Object, "spec", "cors")
	if found && cors != nil {
		fmt.Println("   CORS: Enabled")
	}
}

func displayClientTrafficPolicyInfo(obj *unstructured.Unstructured) {
	targetRef, _, _ := unstructured.NestedMap(obj.Object, "spec", "targetRef")
	if targetRef != nil {
		if kind, ok := targetRef["kind"].(string); ok {
			name, _ := targetRef["name"].(string)
			fmt.Printf("   Target: %s/%s\n", kind, name)
		}
	}
}