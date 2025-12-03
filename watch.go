package main

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WatchServices continuously watches service events
func WatchServices(clientset *kubernetes.Clientset, namespace string) {
	fmt.Println("\n🔍 Watching services for changes...\n")

	watcher, err := clientset.CoreV1().Services(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}

	events := watcher.ResultChan()

	for event := range events {
		svc := event.Object.(*v1.Service)

		fmt.Printf("\n📌 EVENT: %s | Service: %s\n", event.Type, svc.Name)

		if len(svc.ManagedFields) == 0 {
			fmt.Println("   No managed fields available for this service.")
		}

		for _, mf := range svc.ManagedFields {
			fmt.Println("---- Change Event ----")
			fmt.Println("Manager       :", mf.Manager)       // kubectl, controller, kube-apiserver
			fmt.Println("Operation     :", mf.Operation)     // Update, Apply, Patch
			fmt.Println("APIVersion    :", mf.APIVersion)
			if mf.Time != nil {
				fmt.Println("Time          :", mf.Time.Time)
			} else {
				fmt.Println("Time          : <nil>")
			}
			fmt.Println("-----------------------")
		}

		fmt.Println("-----------------------------------------------------")
	}

	// optional: stop watcher automatically after 60 secs
	go func() {
		time.Sleep(60 * time.Second)
		fmt.Println("⛔ Watcher Stopped")
		watcher.Stop()
	}()
}
