package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Command-line flags
	configFile := flag.String("config", "resources.json", "Path to resources configuration file")
	flag.Parse()

	home, _ := os.UserHomeDir()
	kubeConfigPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err)
	}

	// Create dynamic client - ONE client for everything
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	fmt.Println("🚀 Starting Generic Kubernetes Watcher")
	fmt.Println("=======================================")

	// ========================================================================
	// STEP 1: Load configuration from JSON file
	// ========================================================================
	fmt.Printf("📄 Loading configuration from: %s\n", *configFile)
	
	watcherConfig, err := LoadConfigFromFile(*configFile)
	if err != nil {
		fmt.Printf("⚠️  Failed to load config file: %v\n", err)
		fmt.Println("📋 Using default configuration...")
		watcherConfig = GetDefaultWatcherConfig()
	} else {
		fmt.Println("✅ Configuration loaded successfully")
	}

	// ========================================================================
	// STEP 2: Create the Event Pipeline
	// ========================================================================
	pipeline := NewEventPipeline(1000)

	// ========================================================================
	// STEP 3: Register custom handlers (using string kind now)
	// ========================================================================
	
	// Handler 1: Alert on Gateway changes
	pipeline.RegisterHandler(func(event ResourceEvent, changes *ChangeDetails) {
		if event.ResourceKind == "Gateway" && event.Type == EventTypeModified {
			fmt.Printf("🚨 ALERT: Gateway %s/%s was modified!\n", event.Namespace, event.Name)
		}
	})

	// Handler 2: Alert on SecurityPolicy changes
	pipeline.RegisterHandler(func(event ResourceEvent, changes *ChangeDetails) {
		if event.ResourceKind == "SecurityPolicy" {
			if len(changes.SpecChanges) > 0 {
				fmt.Printf("🔒 SECURITY: SecurityPolicy %s/%s spec changed!\n", 
					event.Namespace, event.Name)
			}
		}
	})

	// Handler 3: Log all changes
	pipeline.RegisterHandler(func(event ResourceEvent, changes *ChangeDetails) {
		if event.Type == EventTypeModified {
			fmt.Printf("📊 CHANGE DETECTED: %s %s/%s\n", 
				event.ResourceKind, event.Namespace, event.Name)
		}
	})

	// ========================================================================
	// STEP 4: Start the pipeline
	// ========================================================================
	go pipeline.Start()

	// ========================================================================
	// STEP 5: Start watchers for enabled resources
	// ========================================================================
	fmt.Println("\n📡 Starting Watchers...")
	fmt.Printf("   Namespace: %s\n", watcherConfig.Namespace)
	fmt.Println("   Enabled Resources:")
	
	enabledResources := watcherConfig.GetEnabledResources()
	
	if len(enabledResources) == 0 {
		fmt.Println("   ⚠️  No resources enabled in configuration!")
		os.Exit(1)
	}
	
	for _, resource := range enabledResources {
		fmt.Printf("      ✓ %s (%s/%s)\n", 
			resource.Kind, 
			resource.Group, 
			resource.Resource)
		
		// Start watcher for this resource
		go WatchResource(
			dynamicClient,
			resource.ToGVR(),
			watcherConfig.Namespace,
			resource.Kind, // Just a string now
			pipeline,
		)
	}

	fmt.Println("\n✅ All watchers active")
	fmt.Println("⚡ Pipeline running. Press Ctrl+C to stop")
	fmt.Println("=======================================\n")
	
	// Block forever
	select {}
}
