package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    v1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/retry"
)

func main() {

    // Load kubeconfig
    home, _ := os.UserHomeDir()
    kubeConfigPath := filepath.Join(home, ".kube/config")

    config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
    if err != nil {
        panic(err)
    }

    client := kubernetes.NewForConfigOrDie(config)
    namespace := "default"
    podsClient := client.CoreV1().Pods(namespace)

    // List pods across all namespaces
    pods, err := client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
    if err != nil {
        panic(err)
    }

    fmt.Printf("\nThere are %d pods in the cluster\n", len(pods.Items))
    for i, pod := range pods.Items {
        fmt.Printf("%d -> %s (%s)\n", i+1, pod.Name, pod.Namespace)
    }

    // ---------------------------------------------------------------
    // Create Pod
    // ---------------------------------------------------------------
    podDefinition := &v1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            GenerateName: "demo-k8s-",
            Namespace:    namespace,
        },
        Spec: v1.PodSpec{
            Containers: []v1.Container{
                {
                    Name:  "nginx-container",
                    Image: "nginx:latest",
                },
            },
        },
    }

    newPod, err := podsClient.Create(context.TODO(), podDefinition, metav1.CreateOptions{})
    if err != nil {
        panic(err)
    }
    fmt.Printf("\nPod '%s' created successfully!\n", newPod.Name)

    // ---------------------------------------------------------------
    // Update Pod (change container image)
    // ---------------------------------------------------------------
    fmt.Println("\nUpdating pod image...")

    retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

        // get latest pod
        currentPod, getErr := podsClient.Get(context.TODO(), newPod.Name, metav1.GetOptions{})
        if getErr != nil {
            return getErr
        }

        // apply update
        currentPod.Spec.Containers[0].Image = "nginx:1.25.4"

        updatedPod, updateErr := podsClient.Update(context.TODO(), currentPod, metav1.UpdateOptions{})
        if updateErr == nil {
            fmt.Printf("Updated Pod Image -> %s\n", updatedPod.Spec.Containers[0].Image)
        }
        return updateErr
    })

    if retryErr != nil {
        panic(retryErr)
    }

    fmt.Println("Pod updated successfully!")


    deleteErr := podsClient.Delete(context.TODO(), "demo-k8s-7p7w9", metav1.DeleteOptions{})
    if deleteErr != nil {
        panic(deleteErr.Error())
    }
}
