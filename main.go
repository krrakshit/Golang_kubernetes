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
	redisAddr := flag.String("redis", "localhost:6379", "Redis server address")
	maxChanges := flag.Int("max-changes", 100, "Maximum number of changes to keep in queue")
	httpPort := flag.String("port", "8080", "HTTP server port")
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
	// STEP 0: Initialize Redis Manager
	// ========================================================================
	fmt.Printf("🔗 Connecting to Redis at %s...\n", *redisAddr)
	redisManager, err := NewRedisManager(*redisAddr, "annotation_changes", *maxChanges)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		panic(err)
	}
	fmt.Println("✅ Redis connected successfully")
	defer redisManager.Close()

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
	pipeline := NewEventPipeline(1000, redisManager)
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
	fmt.Println("   Enabled Resources:")

	enabledResources := watcherConfig.GetEnabledResources()

	if len(enabledResources) == 0 {
		fmt.Println("   ⚠️  No resources enabled in configuration!")
		os.Exit(1)
	}

	for _, resource := range enabledResources {
		namespaceStr := "all namespaces"
		if len(resource.Namespaces) > 0 {
			namespaceStr = fmt.Sprintf("%v", resource.Namespaces)
		}

		fmt.Printf("      ✓ %s (%s/%s) - Watching %s\n",
			resource.Kind,
			resource.Group,
			resource.Resource,
			namespaceStr)

		// Start watcher for this resource with its namespaces
		go WatchResource(
			dynamicClient,
			resource.ToGVR(),
			resource.Namespaces, // Pass namespace array
			resource.Kind,
			pipeline,
		)
	}

	fmt.Println("\n✅ All watchers active")
	fmt.Println("⚡ Pipeline running. Press Ctrl+C to stop")
	fmt.Println("=======================================\n")

	// ========================================================================
	// STEP 6: Start HTTP server (non-blocking)
	// ========================================================================
	go StartHTTPServer(redisManager, *httpPort)

	// Block forever
	select {}
}
