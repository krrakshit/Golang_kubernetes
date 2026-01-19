package main

import (
	"fmt"
	"os"
)

// QueryChanges retrieves and displays annotation changes from the Redis queue
func QueryChanges(redisManager *RedisManager, numChanges int) error {
	if redisManager == nil {
		return fmt.Errorf("Redis manager not initialized")
	}

	// Get queue size
	size, err := redisManager.GetQueueSize()
	if err != nil {
		fmt.Printf("❌ Failed to get queue size: %v\n", err)
		return err
	}

	fmt.Printf("📊 Total annotation changes in queue: %d\n", size)

	// Print last n changes
	if err := redisManager.PrintLastNChanges(numChanges); err != nil {
		fmt.Printf("❌ Failed to retrieve changes: %v\n", err)
		return err
	}
	return nil
}

// CLI function to query from command line
func QueryChangesFromCLI(redisAddr string, numChanges int) {
	redisManager, err := NewRedisManager(redisAddr, "annotation_changes", 1000)
	if err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		os.Exit(1)
	}
	defer redisManager.Close()

	if err := QueryChanges(redisManager, numChanges); err != nil {
		os.Exit(1)
	}
}
