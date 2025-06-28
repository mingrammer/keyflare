package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	redisWrapper "github.com/mingrammer/keyflare/pkg/redis"
	"github.com/redis/go-redis/v9"
)

// RunTrafficSimulation generates varied traffic patterns to demonstrate KeyFlare functionality
func RunTrafficSimulation(ctx context.Context, client *redisWrapper.Wrapper) {
	fmt.Println("\n--- Running Traffic Simulation ---")

	// Hot keys that should trigger KeyFlare policies
	hotKeys := []string{
		"user:profile:popular_user",
		"counter:global:requests",
		"product:details:trending_item",
	}

	// Normal keys with moderate access
	normalKeys := []string{
		"user:profile:regular_user_1",
		"user:profile:regular_user_2",
		"product:details:normal_item",
	}

	// Set initial data
	fmt.Println("Setting up initial data...")
	for _, key := range append(hotKeys, normalKeys...) {
		err := client.Set(ctx, key, fmt.Sprintf("value_for_%s", key), time.Hour).Err()
		if err != nil {
			log.Printf("Failed to set key %s: %v", key, err)
		}
	}

	var wg sync.WaitGroup

	// Generate heavy traffic for hot keys
	fmt.Println("Generating heavy traffic for hot keys...")
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				// Access hot keys frequently
				for _, key := range hotKeys {
					val, err := client.Get(ctx, key).Result()
					if err != nil && err != redis.Nil {
						log.Printf("Worker %d: Error getting %s: %v", workerID, key, err)
					} else if j%15 == 0 {
						fmt.Printf("Worker %d: Got %s = %s\n", workerID, key, val)
					}
					time.Sleep(10 * time.Millisecond)
				}

				// Occasional writess
				if j%10 == 0 {
					key := hotKeys[j%len(hotKeys)]
					newVal := fmt.Sprintf("updated_by_worker_%d_at_%d", workerID, j)
					client.Set(ctx, key, newVal, time.Hour)
				}

				// Access normal keys occasionally
				if j%20 == 0 {
					for _, key := range normalKeys {
						client.Get(ctx, key)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	fmt.Println("✓ Traffic simulation completed")
}
