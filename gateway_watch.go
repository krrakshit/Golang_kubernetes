package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var (
	previousGateways   = make(map[string]*gatewayv1.Gateway)
	previousHTTPRoutes = make(map[string]*gatewayv1.HTTPRoute)
	gwMu               sync.RWMutex
)

// hasMetadataOrSpecChanges checks if the managed field contains metadata or spec changes
func hasGatewayMetadataOrSpecChanges(mf metav1.ManagedFieldsEntry) bool {
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

// WatchGateways watches Gateway API Gateway resources
func WatchGateways(gatewayClient *gatewayclientset.Clientset, namespace string) {
	fmt.Println("\n🌐 Watching Gateways for changes (metadata/spec only)...\n")

	watcher, err := gatewayClient.GatewayV1().Gateways(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		gw, ok := event.Object.(*gatewayv1.Gateway)
		if !ok {
			fmt.Println("⚠️  Failed to cast to Gateway")
			continue
		}

		// Filter: only show if there are metadata or spec changes
		hasRelevantChanges := false
		for _, mf := range gw.ManagedFields {
			if hasGatewayMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != watch.Added {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | Gateway: %s (namespace: %s)\n", event.Type, gw.Name, gw.Namespace)

		// Get previous state
		gwMu.RLock()
		oldGW := previousGateways[gw.Namespace+"/"+gw.Name]
		gwMu.RUnlock()

		// Compare and log changes
		if event.Type == watch.Modified && oldGW != nil {
			compareGatewayChanges(oldGW, gw)
		} else if event.Type == watch.Added {
			fmt.Println("   → New Gateway created")
			displayGatewayInfo(gw)
		} else if event.Type == watch.Deleted {
			fmt.Println("   → Gateway deleted")
		}

		// Store current state
		gwMu.Lock()
		gwCopy := gw.DeepCopy()
		previousGateways[gw.Namespace+"/"+gw.Name] = gwCopy
		gwMu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

func getAllgateways(gatewayClient *gatewayclientset.Clientset, namespace string) ([]gatewayv1.Gateway, error) {
	gws, err := gatewayClient.GatewayV1().Gateways(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return []gatewayv1.Gateway{}, err
	}
	return gws.Items, nil
}

// WatchHTTPRoutes watches Gateway API HTTPRoute resources
func WatchHTTPRoutes(gatewayClient *gatewayclientset.Clientset, namespace string) {
	fmt.Println("\n🛣️  Watching HTTPRoutes for changes (metadata/spec only)...\n")

	watcher, err := gatewayClient.GatewayV1().HTTPRoutes(namespace).Watch(context.TODO(),
	metav1.ListOptions{},
)

	if err != nil {
		panic(err)
	}
	defer watcher.Stop()

	events := watcher.ResultChan()

	for event := range events {
		route, ok := event.Object.(*gatewayv1.HTTPRoute)
		if !ok {
			fmt.Println("⚠️  Failed to cast to HTTPRoute")
			continue
		}

		// Filter: only show if there are metadata or spec changes
		hasRelevantChanges := false
		for _, mf := range route.ManagedFields {
			if hasGatewayMetadataOrSpecChanges(mf) {
				hasRelevantChanges = true
				break
			}
		}

		if !hasRelevantChanges && event.Type != watch.Added {
			continue
		}

		fmt.Printf("\n📌 EVENT: %s | HTTPRoute: %s (namespace: %s)\n", event.Type, route.Name, route.Namespace)

		// Get previous state
		gwMu.RLock()
		oldRoute := previousHTTPRoutes[route.Namespace+"/"+route.Name]
		gwMu.RUnlock()

		// Compare and log changes
		if event.Type == watch.Modified && oldRoute != nil {
			compareHTTPRouteChanges(oldRoute, route)
		} else if event.Type == watch.Added {
			fmt.Println("   → New HTTPRoute created")
			displayHTTPRouteInfo(route)
		} else if event.Type == watch.Deleted {
			fmt.Println("   → HTTPRoute deleted")
		}

		// Store current state
		gwMu.Lock()
		routeCopy := route.DeepCopy()
		previousHTTPRoutes[route.Namespace+"/"+route.Name] = routeCopy
		gwMu.Unlock()

		fmt.Println("-----------------------------------------------------")
	}
}

// compareGatewayChanges compares two Gateway objects
func compareGatewayChanges(oldGW, newGW *gatewayv1.Gateway) {
	fmt.Println("\n   🔍 GATEWAY CHANGES:")

	changesFound := false

	// Compare Labels
	if !reflect.DeepEqual(oldGW.Labels, newGW.Labels) {
		changesFound = true
		fmt.Println("   📝 Labels:")
		for key, newVal := range newGW.Labels {
			oldVal, existed := oldGW.Labels[key]
			if !existed {
				fmt.Printf("      + %s: %s (added)\n", key, newVal)
			} else if oldVal != newVal {
				fmt.Printf("      ± %s: %s → %s\n", key, oldVal, newVal)
			}
		}
		for key, oldVal := range oldGW.Labels {
			if _, exists := newGW.Labels[key]; !exists {
				fmt.Printf("      - %s: %s (removed)\n", key, oldVal)
			}
		}
	}

	// Compare Annotations
	if !reflect.DeepEqual(oldGW.Annotations, newGW.Annotations) {
		changesFound = true
		fmt.Println("   📝 Annotations changed")
	}

	// Compare GatewayClassName
	if oldGW.Spec.GatewayClassName != newGW.Spec.GatewayClassName {
		changesFound = true
		fmt.Printf("   📝 GatewayClassName: %s → %s\n", oldGW.Spec.GatewayClassName, newGW.Spec.GatewayClassName)
	}

	// Compare Listeners
	if len(oldGW.Spec.Listeners) != len(newGW.Spec.Listeners) {
		changesFound = true
		fmt.Printf("   📝 Listener count changed: %d → %d\n", len(oldGW.Spec.Listeners), len(newGW.Spec.Listeners))
	} else {
		for i, newListener := range newGW.Spec.Listeners {
			if i < len(oldGW.Spec.Listeners) {
				oldListener := oldGW.Spec.Listeners[i]
				if !reflect.DeepEqual(oldListener, newListener) {
					changesFound = true
					fmt.Printf("   📝 Listener[%d] (%s) changed:\n", i, newListener.Name)
					
					if oldListener.Protocol != newListener.Protocol {
						fmt.Printf("      Protocol: %s → %s\n", oldListener.Protocol, newListener.Protocol)
					}
					if oldListener.Port != newListener.Port {
						fmt.Printf("      Port: %d → %d\n", oldListener.Port, newListener.Port)
					}
					if !reflect.DeepEqual(oldListener.TLS, newListener.TLS) {
						fmt.Println("      TLS configuration changed")
					}
				}
			}
		}
	}

	if !changesFound {
		fmt.Println("      (no significant changes detected)")
	}
}

// compareHTTPRouteChanges compares two HTTPRoute objects
func compareHTTPRouteChanges(oldRoute, newRoute *gatewayv1.HTTPRoute) {
	fmt.Println("\n   🔍 HTTPROUTE CHANGES:")

	changesFound := false

	// Compare Hostnames
	if !reflect.DeepEqual(oldRoute.Spec.Hostnames, newRoute.Spec.Hostnames) {
		changesFound = true
		fmt.Println("   📝 Hostnames changed:")
		fmt.Printf("      Old: %v\n", oldRoute.Spec.Hostnames)
		fmt.Printf("      New: %v\n", newRoute.Spec.Hostnames)
	}

	// Compare ParentRefs
	if !reflect.DeepEqual(oldRoute.Spec.ParentRefs, newRoute.Spec.ParentRefs) {
		changesFound = true
		fmt.Println("   📝 ParentRefs changed")
	}

	// Compare Rules count
	if len(oldRoute.Spec.Rules) != len(newRoute.Spec.Rules) {
		changesFound = true
		fmt.Printf("   📝 Rules count changed: %d → %d\n", len(oldRoute.Spec.Rules), len(newRoute.Spec.Rules))
	}

	if !changesFound {
		fmt.Println("      (no significant changes detected)")
	}
}

// displayGatewayInfo shows Gateway details
func displayGatewayInfo(gw *gatewayv1.Gateway) {
	fmt.Printf("   GatewayClass: %s\n", gw.Spec.GatewayClassName)
	fmt.Printf("   Listeners: %d\n", len(gw.Spec.Listeners))
	for i, listener := range gw.Spec.Listeners {
		fmt.Printf("     [%d] Name: %s, Protocol: %s, Port: %d\n", 
			i, listener.Name, listener.Protocol, listener.Port)
		if listener.TLS != nil {
			fmt.Printf("         TLS: Mode=%s, CertRefs=%d\n", 
				*listener.TLS.Mode, len(listener.TLS.CertificateRefs))
		}
	}
}

// displayHTTPRouteInfo shows HTTPRoute details
func displayHTTPRouteInfo(route *gatewayv1.HTTPRoute) {
	fmt.Printf("   Hostnames: %v\n", route.Spec.Hostnames)
	fmt.Printf("   Parent Gateways: %d\n", len(route.Spec.ParentRefs))
	for i, parent := range route.Spec.ParentRefs {
		fmt.Printf("     [%d] Gateway: %s\n", i, parent.Name)
	}
	fmt.Printf("   Rules: %d\n", len(route.Spec.Rules))
}