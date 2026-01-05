package main

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

func main() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		panic(err)
	}

	// Initialize clients
	gatewayClient := gatewayclientset.NewForConfigOrDie(config)
	envoyClient, err := NewEnvoyGatewayClient(config)
	if err != nil {
		panic(err)
	}

	fmt.Println("🚀 Starting Gateway & Envoy Gateway Watcher with Event Pipeline")
	fmt.Println("================================================================")

	// ========================================================================
	// STEP 1: Create the Event Pipeline (buffer size = 1000 events)
	// ========================================================================
	pipeline := NewEventPipeline(1000)

	// ========================================================================
	// STEP 2: Register custom handlers (optional - add your own logic here)
	// ========================================================================
	
	// Handler 1: Alert on Gateway changes
	pipeline.RegisterHandler(func(event ResourceEvent, changes *ChangeDetails) {
		if event.ResourceType == ResourceTypeGateway && event.Type == EventTypeModified {
			fmt.Printf("🚨 CUSTOM ALERT: Gateway %s/%s was modified!\n", event.Namespace, event.Name)
		}
	})

	// Handler 2: Alert on SecurityPolicy changes
	pipeline.RegisterHandler(func(event ResourceEvent, changes *ChangeDetails) {
		if event.ResourceType == ResourceTypeSecurityPolicy {
			if len(changes.SpecChanges) > 0 {
				fmt.Printf("🔒 SECURITY ALERT: SecurityPolicy %s/%s spec changed!\n", 
					event.Namespace, event.Name)
			}
		}
	})

	// Handler 3: Count listener changes in Gateway
	pipeline.RegisterHandler(func(event ResourceEvent, changes *ChangeDetails) {
		if event.ResourceType == ResourceTypeGateway && event.Type == EventTypeModified {
			if listenerChange, ok := changes.SpecChanges["listeners"]; ok {
				fmt.Printf("📡 LISTENER ALERT: Gateway %s listeners changed: %v\n", 
					event.Name, listenerChange)
			}
		}
	})

	// ========================================================================
	// STEP 3: Start the pipeline processor (in background)
	// ========================================================================
	go pipeline.Start()

	// ========================================================================
	// STEP 4: Start watchers - ONLY Gateway API & Envoy Gateway CRDs
	// ========================================================================
	fmt.Println("\n📡 Starting Watchers...")
	fmt.Println("   🌐 Gateway API: Gateways, HTTPRoutes")
	fmt.Println("   🔧 Envoy Gateway: EnvoyProxy, BackendTrafficPolicy, SecurityPolicy, ClientTrafficPolicy")

	// Gateway API resources
	go WatchGateways(gatewayClient, "default", pipeline)
	go WatchHTTPRoutes(gatewayClient, "default", pipeline)

	// Envoy Gateway CRDs
	go WatchEnvoyProxies(envoyClient.GetDynamicClient(), "default", pipeline)
	go WatchBackendTrafficPolicies(envoyClient.GetDynamicClient(), "default", pipeline)
	go WatchSecurityPolicies(envoyClient.GetDynamicClient(), "default", pipeline)
	go WatchClientTrafficPolicies(envoyClient.GetDynamicClient(), "default", pipeline)

	fmt.Println("\n✅ All watchers active")
	fmt.Println("⚡ Pipeline running. Press Ctrl+C to stop")
	fmt.Println("================================================================\n")
	
	// Block forever
	select {}
}
