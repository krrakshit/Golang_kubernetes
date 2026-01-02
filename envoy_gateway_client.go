package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// EnvoyGatewayClient wraps the dynamic client for Envoy Gateway CRDs
type EnvoyGatewayClient struct {
	dynamicClient dynamic.Interface
}

// NewEnvoyGatewayClient creates a new Envoy Gateway client
func NewEnvoyGatewayClient(config *rest.Config) (*EnvoyGatewayClient, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &EnvoyGatewayClient{
		dynamicClient: dynamicClient,
	}, nil
}

// Define all Envoy Gateway CRD GroupVersionResources
var (
	EnvoyProxyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "envoyproxies",
	}

	BackendTrafficPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "backendtrafficpolicies",
	}

	SecurityPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "securitypolicies",
	}

	ClientTrafficPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "clienttrafficpolicies",
	}

	EnvoyPatchPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "envoypatchpolicies",
	}

	EnvoyExtensionPolicyGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "envoyextensionpolicies",
	}

	BackendGVR = schema.GroupVersionResource{
		Group:    "gateway.envoyproxy.io",
		Version:  "v1alpha1",
		Resource: "backends",
	}
)

// ============================================================================
// ENVOYPROXY METHODS
// ============================================================================

// ListEnvoyProxies lists all EnvoyProxy resources in a namespace
func (c *EnvoyGatewayClient) ListEnvoyProxies(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(EnvoyProxyGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetEnvoyProxy gets a specific EnvoyProxy resource
func (c *EnvoyGatewayClient) GetEnvoyProxy(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(EnvoyProxyGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// CreateEnvoyProxy creates a new EnvoyProxy resource
func (c *EnvoyGatewayClient) CreateEnvoyProxy(namespace string, envoyProxy *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(EnvoyProxyGVR).Namespace(namespace).Create(
		context.Background(),
		envoyProxy,
		metav1.CreateOptions{},
	)
}

// UpdateEnvoyProxy updates an existing EnvoyProxy resource
func (c *EnvoyGatewayClient) UpdateEnvoyProxy(namespace string, envoyProxy *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(EnvoyProxyGVR).Namespace(namespace).Update(
		context.Background(),
		envoyProxy,
		metav1.UpdateOptions{},
	)
}

// DeleteEnvoyProxy deletes an EnvoyProxy resource
func (c *EnvoyGatewayClient) DeleteEnvoyProxy(namespace, name string) error {
	return c.dynamicClient.Resource(EnvoyProxyGVR).Namespace(namespace).Delete(
		context.Background(),
		name,
		metav1.DeleteOptions{},
	)
}

// ============================================================================
// BACKENDTRAFFICPOLICY METHODS
// ============================================================================

// ListBackendTrafficPolicies lists all BackendTrafficPolicy resources
func (c *EnvoyGatewayClient) ListBackendTrafficPolicies(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(BackendTrafficPolicyGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetBackendTrafficPolicy gets a specific BackendTrafficPolicy
func (c *EnvoyGatewayClient) GetBackendTrafficPolicy(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(BackendTrafficPolicyGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// CreateBackendTrafficPolicy creates a BackendTrafficPolicy
func (c *EnvoyGatewayClient) CreateBackendTrafficPolicy(namespace string, policy *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(BackendTrafficPolicyGVR).Namespace(namespace).Create(
		context.Background(),
		policy,
		metav1.CreateOptions{},
	)
}

// UpdateBackendTrafficPolicy updates a BackendTrafficPolicy
func (c *EnvoyGatewayClient) UpdateBackendTrafficPolicy(namespace string, policy *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(BackendTrafficPolicyGVR).Namespace(namespace).Update(
		context.Background(),
		policy,
		metav1.UpdateOptions{},
	)
}

// DeleteBackendTrafficPolicy deletes a BackendTrafficPolicy
func (c *EnvoyGatewayClient) DeleteBackendTrafficPolicy(namespace, name string) error {
	return c.dynamicClient.Resource(BackendTrafficPolicyGVR).Namespace(namespace).Delete(
		context.Background(),
		name,
		metav1.DeleteOptions{},
	)
}

// ============================================================================
// SECURITYPOLICY METHODS
// ============================================================================

// ListSecurityPolicies lists all SecurityPolicy resources
func (c *EnvoyGatewayClient) ListSecurityPolicies(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(SecurityPolicyGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetSecurityPolicy gets a specific SecurityPolicy
func (c *EnvoyGatewayClient) GetSecurityPolicy(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(SecurityPolicyGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// CreateSecurityPolicy creates a SecurityPolicy
func (c *EnvoyGatewayClient) CreateSecurityPolicy(namespace string, policy *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(SecurityPolicyGVR).Namespace(namespace).Create(
		context.Background(),
		policy,
		metav1.CreateOptions{},
	)
}

// UpdateSecurityPolicy updates a SecurityPolicy
func (c *EnvoyGatewayClient) UpdateSecurityPolicy(namespace string, policy *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(SecurityPolicyGVR).Namespace(namespace).Update(
		context.Background(),
		policy,
		metav1.UpdateOptions{},
	)
}

// DeleteSecurityPolicy deletes a SecurityPolicy
func (c *EnvoyGatewayClient) DeleteSecurityPolicy(namespace, name string) error {
	return c.dynamicClient.Resource(SecurityPolicyGVR).Namespace(namespace).Delete(
		context.Background(),
		name,
		metav1.DeleteOptions{},
	)
}

// ============================================================================
// CLIENTTRAFFICPOLICY METHODS
// ============================================================================

// ListClientTrafficPolicies lists all ClientTrafficPolicy resources
func (c *EnvoyGatewayClient) ListClientTrafficPolicies(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(ClientTrafficPolicyGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetClientTrafficPolicy gets a specific ClientTrafficPolicy
func (c *EnvoyGatewayClient) GetClientTrafficPolicy(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(ClientTrafficPolicyGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// ============================================================================
// ENVOYPATCHPOLICY METHODS
// ============================================================================

// ListEnvoyPatchPolicies lists all EnvoyPatchPolicy resources
func (c *EnvoyGatewayClient) ListEnvoyPatchPolicies(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(EnvoyPatchPolicyGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetEnvoyPatchPolicy gets a specific EnvoyPatchPolicy
func (c *EnvoyGatewayClient) GetEnvoyPatchPolicy(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(EnvoyPatchPolicyGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// ============================================================================
// ENVOYEXTENSIONPOLICY METHODS
// ============================================================================

// ListEnvoyExtensionPolicies lists all EnvoyExtensionPolicy resources
func (c *EnvoyGatewayClient) ListEnvoyExtensionPolicies(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(EnvoyExtensionPolicyGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetEnvoyExtensionPolicy gets a specific EnvoyExtensionPolicy
func (c *EnvoyGatewayClient) GetEnvoyExtensionPolicy(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(EnvoyExtensionPolicyGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// ============================================================================
// BACKEND METHODS
// ============================================================================

// ListBackends lists all Backend resources
func (c *EnvoyGatewayClient) ListBackends(namespace string) (*unstructured.UnstructuredList, error) {
	return c.dynamicClient.Resource(BackendGVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
}

// GetBackend gets a specific Backend
func (c *EnvoyGatewayClient) GetBackend(namespace, name string) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(BackendGVR).Namespace(namespace).Get(
		context.Background(),
		name,
		metav1.GetOptions{},
	)
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// GetDynamicClient returns the underlying dynamic client
func (c *EnvoyGatewayClient) GetDynamicClient() dynamic.Interface {
	return c.dynamicClient
}

// PrintResource prints a resource in a readable format
func PrintResource(resource *unstructured.Unstructured) {
	fmt.Printf("Name: %s\n", resource.GetName())
	fmt.Printf("Namespace: %s\n", resource.GetNamespace())
	fmt.Printf("Kind: %s\n", resource.GetKind())
	fmt.Printf("APIVersion: %s\n", resource.GetAPIVersion())
	fmt.Printf("Labels: %v\n", resource.GetLabels())
	fmt.Printf("Annotations: %v\n", resource.GetAnnotations())
	fmt.Println("---")
}