package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

func main() {

    // get kubeconfig
    home, _ := os.UserHomeDir()
    kubeConfigPath := filepath.Join(home, ".kube/config")

    // use the current context in kubeconfig
    config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
    if err != nil {
        panic(err.Error())
    }

    // create a new client
    client := kubernetes.NewForConfigOrDie(config)

    // define the namespace
  //  namespace := "default"

    // define the pods client (easy for later use)
    // podsClient := client.CoreV1().Pods(namespace)

    // read all pods
    pods, err := client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        panic(err.Error())
    }
    fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

    for i, pod := range pods.Items {
        fmt.Printf("Name of %dth pod: %s\n", i, pod.Name)
    }
}
