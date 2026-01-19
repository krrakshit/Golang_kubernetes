package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// ResourceChange represents a single resource change with versioning
type ResourceChange struct {
	Version      int64                  `json:"version"` // Version number (1, 2, 3...)
	ResourceKind string                 `json:"resource_kind"`
	Namespace    string                 `json:"namespace"`
	ResourceName string                 `json:"resource_name"`
	Timestamp    time.Time              `json:"timestamp"`
	Object       interface{}            `json:"object"`  // Full object snapshot
	Changes      map[string]interface{} `json:"changes"` // What changed from previous version
}

// RedisManager manages Redis queue operations for resource changes
type RedisManager struct {
	client    *redis.Client
	queueName string
	maxSize   int
}

// NewRedisManager creates a new Redis manager
func NewRedisManager(redisAddr string, queueName string, maxSize int) (*RedisManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisManager{
		client:    client,
		queueName: queueName,
		maxSize:   maxSize,
	}, nil
}

// PushResourceChange pushes a new resource change to the global change queue
// Queue has fixed size - oldest changes are automatically removed when queue is full
func (rm *RedisManager) PushResourceChange(resourceKey string, change ResourceChange) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current version for this resource
	version, err := rm.GetCurrentVersion(resourceKey)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}
	change.Version = version + 1

	// Marshal change to JSON
	data, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("failed to marshal change: %w", err)
	}

	// Push to queue (LPUSH adds to the beginning - most recent first)
	// Queue key: resource_changes (all changes from all resources)
	if err := rm.client.LPush(ctx, rm.queueName, string(data)).Err(); err != nil {
		return fmt.Errorf("failed to push to queue: %w", err)
	}

	// Trim queue to maxSize (keep only the most recent N changes)
	// When queue is full and new item added, oldest gets removed automatically
	if err := rm.client.LTrim(ctx, rm.queueName, 0, int64(rm.maxSize-1)).Err(); err != nil {
		return fmt.Errorf("failed to trim queue: %w", err)
	}

	rm.logResourceChange(change, change.Version)
	return nil
}

// GetResourceChanges retrieves all changes from the global queue
func (rm *RedisManager) GetResourceChanges(resourceKey string) ([]ResourceChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get all items from the queue
	results, err := rm.client.LRange(ctx, rm.queueName, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from queue: %w", err)
	}

	changes := make([]ResourceChange, 0, len(results))

	// Unmarshal each result and filter by resourceKey if needed
	for _, result := range results {
		var change ResourceChange
		if err := json.Unmarshal([]byte(result), &change); err != nil {
			continue
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// GetCurrentVersion returns the current version number for a resource (count from queue)
func (rm *RedisManager) GetCurrentVersion(resourceKey string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Count items for this resource in the queue
	results, err := rm.client.LRange(ctx, rm.queueName, 0, -1).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to count versions: %w", err)
	}

	version := int64(0)
	for _, result := range results {
		var change ResourceChange
		if err := json.Unmarshal([]byte(result), &change); err != nil {
			continue
		}
		// Count versions for this specific resource
		key := fmt.Sprintf("%s/%s/%s", change.ResourceKind, change.Namespace, change.ResourceName)
		if key == resourceKey && change.Version > version {
			version = change.Version
		}
	}

	return version, nil
}

// GetQueueSize returns the current number of items in the queue
func (rm *RedisManager) GetQueueSize() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	size, err := rm.client.LLen(ctx, rm.queueName).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get queue size: %w", err)
	}

	return size, nil
}

// ClearQueue removes all changes from the queue
func (rm *RedisManager) ClearQueue() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rm.client.Del(ctx, rm.queueName).Err(); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	fmt.Printf("✅ Queue '%s' cleared\n", rm.queueName)
	return nil
}

// logResourceChange logs the versioned resource change
func (rm *RedisManager) logResourceChange(change ResourceChange, version int64) {
	fmt.Println()
	fmt.Println("📝 RESOURCE CHANGE DETECTED AND STORED")
	fmt.Println("================================================================================")

	fmt.Printf("   Resource: %s\n", change.ResourceKind)
	fmt.Printf("   Namespace: %s\n", change.Namespace)
	fmt.Printf("   Name: %s\n", change.ResourceName)
	fmt.Printf("   Version: %d\n", version)
	fmt.Printf("   Timestamp: %s\n", change.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Println()
	fmt.Println("   FULL OBJECT:")
	objJSON, _ := json.MarshalIndent(change.Object, "      ", "  ")
	fmt.Println(string(objJSON))

	if len(change.Changes) > 0 {
		fmt.Println()
		fmt.Println("   CHANGES FROM PREVIOUS VERSION:")
		changesJSON, _ := json.MarshalIndent(change.Changes, "      ", "  ")
		fmt.Println(string(changesJSON))
	}

	fmt.Println("================================================================================")
}

// GetLastNChanges retrieves the last n changes from the queue
func (rm *RedisManager) GetLastNChanges(n int) ([]ResourceChange, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get last n items from the queue (0 to n-1)
	results, err := rm.client.LRange(ctx, rm.queueName, 0, int64(n-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from queue: %w", err)
	}

	changes := make([]ResourceChange, 0, len(results))

	// Unmarshal each result
	for _, result := range results {
		var change ResourceChange
		if err := json.Unmarshal([]byte(result), &change); err != nil {
			continue
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// Close closes the Redis connection
func (rm *RedisManager) Close() error {
	return rm.client.Close()
}

// PrintLastNChanges prints the last n changes from the queue in a formatted way
func (rm *RedisManager) PrintLastNChanges(n int) error {
	changes, err := rm.GetLastNChanges(n)
	if err != nil {
		return err
	}

	if len(changes) == 0 {
		fmt.Println("\n📭 No changes in the queue")
		return nil
	}

	fmt.Printf("\n📋 Last %d Changes in Queue:\n", len(changes))
	fmt.Println("================================================================================")

	for i, change := range changes {
		fmt.Printf("\n[%d] %s - %s/%s (Version %d at %s)\n",
			i+1,
			change.ResourceKind,
			change.Namespace,
			change.ResourceName,
			change.Version,
			change.Timestamp.Format("2006-01-02 15:04:05"),
		)

		fmt.Println("   FULL OBJECT:")
		objJSON, _ := json.MarshalIndent(change.Object, "      ", "  ")
		fmt.Println(string(objJSON))

		if len(change.Changes) > 0 {
			fmt.Println("   CHANGES:")
			changesJSON, _ := json.MarshalIndent(change.Changes, "      ", "  ")
			fmt.Println(string(changesJSON))
		}
	}

	fmt.Println("\n================================================================================")
	return nil
}
