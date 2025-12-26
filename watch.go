package main

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
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
			fmt.Println("Manager       :", mf.Manager)
			fmt.Println("Operation     :", mf.Operation)
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
}

// WatchDeployments continuously watches deployment events
func WatchDeployments(clientset *kubernetes.Clientset, namespace string) {
	fmt.Println("\n🔍 Watching deployments for changes...\n")

	watcher, err := clientset.AppsV1().Deployments(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}

	events := watcher.ResultChan()

	for event := range events {
		deploy := event.Object.(*appsv1.Deployment)

		fmt.Printf("\n📌 EVENT: %s | Deployment: %s\n", event.Type, deploy.Name)

		if len(deploy.ManagedFields) == 0 {
			fmt.Println("   No managed fields available for this deployment.")
		}

		for _, mf := range deploy.ManagedFields {
			fmt.Println("---- Change Event ----")
			fmt.Println("Manager       :", mf.Manager)
			fmt.Println("Operation     :", mf.Operation)
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
}

// WatchReplicaSets continuously watches replicaset events
func WatchReplicaSets(clientset *kubernetes.Clientset, namespace string) {
	fmt.Println("\n🔍 Watching replicasets for changes...\n")

	watcher, err := clientset.AppsV1().ReplicaSets(namespace).Watch(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err != nil {
		panic(err)
	}

	events := watcher.ResultChan()

	for event := range events {
		rs := event.Object.(*appsv1.ReplicaSet)

		fmt.Printf("\n📌 EVENT: %s | ReplicaSet: %s\n", event.Type, rs.Name)

		if len(rs.ManagedFields) == 0 {
			fmt.Println("   No managed fields available for this replicaset.")
		}

		for _, mf := range rs.ManagedFields {
			fmt.Println("---- Change Event ----")
			fmt.Println("Manager       :", mf.Manager)
			fmt.Println("Operation     :", mf.Operation)
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
}