package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Store previous state of resources
var (
	previousServices    = make(map[string]*v1.Service)
	previousDeployments = make(map[string]*appsv1.Deployment)
	previousReplicaSets = make(map[string]*appsv1.ReplicaSet)
	mu                  sync.RWMutex
)

// hasMetadataOrSpecChanges checks if the managed field contains metadata or spec changes
func hasMetadataOrSpecChanges(mf metav1.ManagedFieldsEntry) bool {
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

// compareAndLogChanges compares two objects and logs the differences
func compareAndLogChanges(oldObj, newObj interface{}, objType string) {
	if oldObj == nil {
		fmt.Println("   → New resource created")
		return
	}

	oldVal := reflect.ValueOf(oldObj).Elem()
	newVal := reflect.ValueOf(newObj).Elem()

	// Compare Metadata
	fmt.Println("\n   🔍 METADATA CHANGES:")
	oldMeta := oldVal.FieldByName("ObjectMeta")
	newMeta := newVal.FieldByName("ObjectMeta")
	compareMetadata(oldMeta, newMeta)

	// Compare Spec
	fmt.Println("\n   🔍 SPEC CHANGES:")
	oldSpec := oldVal.FieldByName("Spec")
	newSpec := newVal.FieldByName("Spec")
	compareSpec(oldSpec, newSpec)
}

// compareMetadata compares metadata fields
func compareMetadata(oldMeta, newMeta reflect.Value) {
	changesFound := false

	// Check Labels
	oldLabels := oldMeta.FieldByName("Labels").Interface().(map[string]string)
	newLabels := newMeta.FieldByName("Labels").Interface().(map[string]string)
	if !reflect.DeepEqual(oldLabels, newLabels) {
		changesFound = true
		fmt.Println("   📝 Labels:")
		for key, newVal := range newLabels {
			oldVal, existed := oldLabels[key]
			if !existed {
				fmt.Printf("      + %s: %s (added)\n", key, newVal)
			} else if oldVal != newVal {
				fmt.Printf("      ± %s: %s → %s\n", key, oldVal, newVal)
			}
		}
		for key, oldVal := range oldLabels {
			if _, exists := newLabels[key]; !exists {
				fmt.Printf("      - %s: %s (removed)\n", key, oldVal)
			}
		}
	}

	// Check Annotations
	oldAnnotations := oldMeta.FieldByName("Annotations").Interface().(map[string]string)
	newAnnotations := newMeta.FieldByName("Annotations").Interface().(map[string]string)
	if !reflect.DeepEqual(oldAnnotations, newAnnotations) {
		changesFound = true
		fmt.Println("   📝 Annotations:")
		for key, newVal := range newAnnotations {
			oldVal, existed := oldAnnotations[key]
			if !existed {
				fmt.Printf("      + %s: %s (added)\n", key, newVal)
			} else if oldVal != newVal {
				fmt.Printf("      ± %s: %s → %s\n", key, oldVal, newVal)
			}
		}
		for key, oldVal := range oldAnnotations {
			if _, exists := newAnnotations[key]; !exists {
				fmt.Printf("      - %s: %s (removed)\n", key, oldVal)
			}
		}
	}

	// Check ResourceVersion
	oldRV := oldMeta.FieldByName("ResourceVersion").String()
	newRV := newMeta.FieldByName("ResourceVersion").String()
	if oldRV != newRV {
		changesFound = true
		fmt.Printf("   📝 ResourceVersion: %s → %s\n", oldRV, newRV)
	}

	if !changesFound {
		fmt.Println("      (no changes)")
	}
}

// compareSpec compares spec fields
func compareSpec(oldSpec, newSpec reflect.Value) {
	if !oldSpec.IsValid() || !newSpec.IsValid() {
		fmt.Println("      (no spec to compare)")
		return
	}

	changesFound := false

	// Generic comparison for all spec fields
	oldSpecType := oldSpec.Type()
	for i := 0; i < oldSpec.NumField(); i++ {
		fieldName := oldSpecType.Field(i).Name
		oldField := oldSpec.Field(i)
		newField := newSpec.Field(i)

		if !reflect.DeepEqual(oldField.Interface(), newField.Interface()) {
			changesFound = true
			
			// Special handling for common fields
			switch fieldName {
			case "Replicas":
				if oldField.Kind() == reflect.Ptr && !oldField.IsNil() {
					oldVal := oldField.Elem().Int()
					newVal := newField.Elem().Int()
					fmt.Printf("   📝 Replicas: %d → %d\n", oldVal, newVal)
				}
			case "Selector":
				fmt.Printf("   📝 Selector changed\n")
				compareLabels(oldField, newField, "      Selector")
			case "Template":
				fmt.Printf("   📝 Template changed\n")
				// You can add more detailed template comparison here
			case "Ports":
				fmt.Printf("   📝 Ports changed\n")
				comparePorts(oldField, newField)
			case "Type":
				fmt.Printf("   📝 Type: %v → %v\n", oldField.Interface(), newField.Interface())
			case "ClusterIP":
				fmt.Printf("   📝 ClusterIP: %v → %v\n", oldField.Interface(), newField.Interface())
			default:
				fmt.Printf("   📝 %s: %v → %v\n", fieldName, oldField.Interface(), newField.Interface())
			}
		}
	}

	if !changesFound {
		fmt.Println("      (no changes)")
	}
}

// compareLabels compares label selectors
func compareLabels(oldField, newField reflect.Value, prefix string) {
	if oldField.Kind() == reflect.Ptr && !oldField.IsNil() {
		oldLabels := oldField.Elem().FieldByName("MatchLabels").Interface().(map[string]string)
		newLabels := newField.Elem().FieldByName("MatchLabels").Interface().(map[string]string)
		
		for key, newVal := range newLabels {
			oldVal, existed := oldLabels[key]
			if !existed {
				fmt.Printf("%s + %s: %s\n", prefix, key, newVal)
			} else if oldVal != newVal {
				fmt.Printf("%s ± %s: %s → %s\n", prefix, key, oldVal, newVal)
			}
		}
	}
}

// comparePorts compares service ports
func comparePorts(oldField, newField reflect.Value) {
	if oldField.Len() != newField.Len() {
		fmt.Printf("      Port count changed: %d → %d\n", oldField.Len(), newField.Len())
	}
	
	for i := 0; i < newField.Len() && i < oldField.Len(); i++ {
		oldPort := oldField.Index(i)
		newPort := newField.Index(i)
		
		oldPortNum := oldPort.FieldByName("Port").Int()
		newPortNum := newPort.FieldByName("Port").Int()
		
		if oldPortNum != newPortNum {
			fmt.Printf("      Port[%d]: %d → %d\n", i, oldPortNum, newPortNum)
		}
	}
}

// WatchServices continuously watches service events
func WatchServices(clientset *kubernetes.Clientset, namespace string) {
	fmt.Println("\n🔍 Watching services for changes (metadata/spec only)...\n")

	watcher, err := clientset.CoreV1().Services(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		svc := event.Object.(*v1.Service)

		// Filter: only show if there are metadata or spec changes
		hasRelevantChanges := false
		for _, mf := range svc.ManagedFields {
			if hasMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}
		

		if !hasRelevantChanges && event.Type != "ADDED" {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | Service: %s\n", event.Type, svc.Name)

		// Get previous state
		mu.RLock()
		oldSvc := previousServices[svc.Name]
		mu.RUnlock()

		// Compare and log changes
		if event.Type == "MODIFIED" && oldSvc != nil {
			compareAndLogChanges(oldSvc, svc, "Service")
		} else if event.Type == "ADDED" {
			fmt.Println("   → New service created")
		}

		// Store current state
		mu.Lock()
		svcCopy := svc.DeepCopy()
		previousServices[svc.Name] = svcCopy
		mu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// WatchDeployments continuously watches deployment events
func WatchDeployments(clientset *kubernetes.Clientset, namespace string) {
	fmt.Println("\n🔍 Watching deployments for changes (metadata/spec only)...\n")

	watcher, err := clientset.AppsV1().Deployments(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		deploy := event.Object.(*appsv1.Deployment)

		// Filter: only show if there are metadata or spec changes
		hasRelevantChanges := false
		for _, mf := range deploy.ManagedFields {
			if hasMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != "ADDED" {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | Deployment: %s\n", event.Type, deploy.Name)

		// Get previous state
		mu.RLock()
		oldDeploy := previousDeployments[deploy.Name]
		mu.RUnlock()

		// Compare and log changes
		if event.Type == "MODIFIED" && oldDeploy != nil {
			compareAndLogChanges(oldDeploy, deploy, "Deployment")
		} else if event.Type == "ADDED" {
			fmt.Println("   → New deployment created")
		}

		// Store current state
		mu.Lock()
		deployCopy := deploy.DeepCopy()
		previousDeployments[deploy.Name] = deployCopy
		mu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// WatchReplicaSets continuously watches replicaset events
func WatchReplicaSets(clientset *kubernetes.Clientset, namespace string) {
	fmt.Println("\n🔍 Watching replicasets for changes (metadata/spec only)...\n")

	watcher, err := clientset.AppsV1().ReplicaSets(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		rs := event.Object.(*appsv1.ReplicaSet)

		// Filter: only show if there are metadata or spec changes
		hasRelevantChanges := false
		for _, mf := range rs.ManagedFields {
			if hasMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != "ADDED" {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | ReplicaSet: %s\n", event.Type, rs.Name)

		// Get previous state
		mu.RLock()
		oldRS := previousReplicaSets[rs.Name]
		mu.RUnlock()

		// Compare and log changes
		if event.Type == "MODIFIED" && oldRS != nil {
			compareAndLogChanges(oldRS, rs, "ReplicaSet")
		} else if event.Type == "ADDED" {
			fmt.Println("   → New replicaset created")
		}

		// Store current state
		mu.Lock()
		rsCopy := rs.DeepCopy()
		previousReplicaSets[rs.Name] = rsCopy
		mu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}