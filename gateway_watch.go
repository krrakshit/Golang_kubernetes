package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// WatchGateways watches Gateway resources and sends to pipeline
func WatchGateways(gatewayClient *gatewayclientset.Clientset, namespace string, pipeline *EventPipeline) {
	// First, list existing gateways
	fmt.Println("📋 Listing existing Gateways...")
	existingGateways, err := gatewayClient.GatewayV1().Gateways(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err == nil {
		for _, gw := range existingGateways.Items {
			fmt.Printf("   Found existing Gateway: %s/%s\n", gw.Namespace, gw.Name)
			// Send as ADDED event
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceType:  ResourceTypeGateway,
				Namespace:     gw.Namespace,
				Name:          gw.Name,
				Object:        &gw,
				Timestamp:     time.Now(),
				ManagedFields: gw.ManagedFields,
			})
		}
	}

	// Now start watching for changes
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
			continue
		}

		// Send to pipeline
		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceType:  ResourceTypeGateway,
			Namespace:     gw.Namespace,
			Name:          gw.Name,
			Object:        gw,
			Timestamp:     time.Now(),
			ManagedFields: gw.ManagedFields,
		})
	}
}

// WatchHTTPRoutes watches HTTPRoute resources and sends to pipeline
func WatchHTTPRoutes(gatewayClient *gatewayclientset.Clientset, namespace string, pipeline *EventPipeline) {
	// First, list existing HTTPRoutes
	fmt.Println("📋 Listing existing HTTPRoutes...")
	existingRoutes, err := gatewayClient.GatewayV1().HTTPRoutes(namespace).List(
		context.TODO(),
		metav1.ListOptions{},
	)
	if err == nil {
		for _, route := range existingRoutes.Items {
			fmt.Printf("   Found existing HTTPRoute: %s/%s\n", route.Namespace, route.Name)
			// Send as ADDED event
			pipeline.SendEvent(ResourceEvent{
				Type:          EventTypeAdded,
				ResourceType:  ResourceTypeHTTPRoute,
				Namespace:     route.Namespace,
				Name:          route.Name,
				Object:        &route,
				Timestamp:     time.Now(),
				ManagedFields: route.ManagedFields,
			})
		}
	}

	// Now start watching for changes
	watcher, err := gatewayClient.GatewayV1().HTTPRoutes(namespace).Watch(
		context.TODO(),
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
			continue
		}

		// Send to pipeline
		pipeline.SendEvent(ResourceEvent{
			Type:          EventType(event.Type),
			ResourceType:  ResourceTypeHTTPRoute,
			Namespace:     route.Namespace,
			Name:          route.Name,
			Object:        route,
			Timestamp:     time.Now(),
			ManagedFields: route.ManagedFields,
		})
	}
}

// Helper function to get all gateways (if needed)
func GetAllGateways(gatewayClient *gatewayclientset.Clientset, namespace string) ([]gatewayv1.Gateway, error) {
	gws, err := gatewayClient.GatewayV1().Gateways(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return []gatewayv1.Gateway{}, err
	}
	return gws.Items, nil
}