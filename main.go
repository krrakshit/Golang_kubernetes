package main

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		panic(err)
	}


	// Envoy Gateway client (our custom wrapper)
	envoyClient, err := NewEnvoyGatewayClient(config)
	if err != nil {
		panic(err)
	}

	fmt.Println("🚀 Starting Kubernetes Watcher with Envoy Gateway Support")

	// Test the Envoy Gateway client
	testEnvoyGatewayClient(envoyClient)

	// Envoy Gateway watchers using our client
	go WatchEnvoyProxies(envoyClient.GetDynamicClient(), "default")
	go WatchBackendTrafficPolicies(envoyClient.GetDynamicClient(), "default")
	go WatchSecurityPolicies(envoyClient.GetDynamicClient(), "default")
	go WatchClientTrafficPolicies(envoyClient.GetDynamicClient(), "default")	

	fmt.Println("\n⚡ All watchers started. Press Ctrl+C to stop\n")
	select {}
}

// testEnvoyGatewayClient tests the Envoy Gateway client
func testEnvoyGatewayClient(client *EnvoyGatewayClient) {
	fmt.Println("\n🧪 Testing Envoy Gateway Client...")

	// Test listing BackendTrafficPolicies
	policies, err := client.ListBackendTrafficPolicies("default")
	if err != nil {
		fmt.Printf("⚠️  Error listing BackendTrafficPolicies: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d BackendTrafficPolicies\n", len(policies.Items))
		for _, policy := range policies.Items {
			fmt.Printf("   - %s/%s\n", policy.GetNamespace(), policy.GetName())
		}
	}

	// Test listing SecurityPolicies
	secPolicies, err := client.ListSecurityPolicies("default")
	if err != nil {
		fmt.Printf("⚠️  Error listing SecurityPolicies: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d SecurityPolicies\n", len(secPolicies.Items))
		for _, policy := range secPolicies.Items {
			fmt.Printf("   - %s/%s\n", policy.GetNamespace(), policy.GetName())
		}
	}

	fmt.Println()
}