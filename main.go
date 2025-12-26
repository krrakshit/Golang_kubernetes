package main

import (
	"fmt"
	"os"
	"path/filepath"
    

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".kube/config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil { panic(err) }

	client := kubernetes.NewForConfigOrDie(config)

	// -------------------------
	// call watcher file function
	// -------------------------
	fmt.Println("Starting Kubernetes Watcher...")
	go WatchServices(client, "default")  
	go WatchDeployments(client, "default") // ← from watch.go
	go WatchReplicaSets(client, "default")

	// block main so program doesn't exit immediately
	select {}
}
