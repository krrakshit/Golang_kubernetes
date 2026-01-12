package main

import (
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceConfig defines what resources to watch
type ResourceConfig struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
	Kind     string `json:"kind"`
	Enabled  bool   `json:"enabled"`
}

// WatcherConfig holds all resources to watch
type WatcherConfig struct {
	Namespace string           `json:"namespace"`
	Resources []ResourceConfig `json:"resources"`
}

// ToGVR converts ResourceConfig to GroupVersionResource
func (rc *ResourceConfig) ToGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    rc.Group,
		Version:  rc.Version,
		Resource: rc.Resource,
	}
}

// LoadConfigFromFile loads configuration from JSON file
func LoadConfigFromFile(filepath string) (*WatcherConfig, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config WatcherConfig
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfigToFile saves configuration to JSON file
func (wc *WatcherConfig) SaveConfigToFile(filepath string) error {
	data, err := json.MarshalIndent(wc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetEnabledResources returns only enabled resources
func (wc *WatcherConfig) GetEnabledResources() []ResourceConfig {
	enabled := []ResourceConfig{}
	for _, res := range wc.Resources {
		if res.Enabled {
			enabled = append(enabled, res)
		}
	}
	return enabled
}

// EnableResource enables watching for a specific resource by kind
func (wc *WatcherConfig) EnableResource(kind string) {
	for i := range wc.Resources {
		if wc.Resources[i].Kind == kind {
			wc.Resources[i].Enabled = true
			return
		}
	}
}

// DisableResource disables watching for a specific resource by kind
func (wc *WatcherConfig) DisableResource(kind string) {
	for i := range wc.Resources {
		if wc.Resources[i].Kind == kind {
			wc.Resources[i].Enabled = false
			return
		}
	}
}

// AddResource adds a new resource to the configuration
func (wc *WatcherConfig) AddResource(resource ResourceConfig) {
	wc.Resources = append(wc.Resources, resource)
}

// GetDefaultWatcherConfig returns a default configuration (fallback)
func GetDefaultWatcherConfig() *WatcherConfig {
	return &WatcherConfig{
		Namespace: "default",
		Resources: []ResourceConfig{
			{
				Group:    "gateway.networking.k8s.io",
				Version:  "v1",
				Resource: "gateways",
				Kind:     "Gateway",
				Enabled:  true,
			},
			{
				Group:    "gateway.networking.k8s.io",
				Version:  "v1",
				Resource: "httproutes",
				Kind:     "HTTPRoute",
				Enabled:  true,
			},
			{
				Group:    "gateway.envoyproxy.io",
				Version:  "v1alpha1",
				Resource: "envoyproxies",
				Kind:     "EnvoyProxy",
				Enabled:  true,
			},
			{
				Group:    "gateway.envoyproxy.io",
				Version:  "v1alpha1",
				Resource: "backendtrafficpolicies",
				Kind:     "BackendTrafficPolicy",
				Enabled:  true,
			},
			{
				Group:    "gateway.envoyproxy.io",
				Version:  "v1alpha1",
				Resource: "securitypolicies",
				Kind:     "SecurityPolicy",
				Enabled:  true,
			},
			{
				Group:    "gateway.envoyproxy.io",
				Version:  "v1alpha1",
				Resource: "clienttrafficpolicies",
				Kind:     "ClientTrafficPolicy",
				Enabled:  true,
			},
		},
	}
}