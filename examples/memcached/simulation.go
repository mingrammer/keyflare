package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	memcachedWrapper "github.com/mingrammer/keyflare/pkg/memcached"
)

// RunTrafficSimulation generates varied traffic patterns to demonstrate KeyFlare functionality
func RunTrafficSimulation(client *memcachedWrapper.Wrapper) {
	fmt.Println("\n--- Running Traffic Simulation ---")

	// Hot keys that should trigger KeyFlare policies
	hotKeys := []string{
		"session:user:popular",
		"catalog:item:trending",
		"config:app:global",
	}

	// Normal keys with moderate access
	normalKeys := []string{
		"session:user:regular_1",
		"session:user:regular_2",
		"catalog:item:normal",
	}

	// Set initial data
	fmt.Println("Setting up initial data...")
	for _, key := range append(hotKeys, normalKeys...) {
		item := &memcache.Item{
			Key:        key,
			Value:      []byte(fmt.Sprintf("value_for_%s", key)),
			Expiration: int32(time.Now().Add(time.Hour).Unix()),
		}
		err := client.Set(item)
		if err != nil {
			log.Printf("Failed to set key %s: %v", key, err)
		}
	}

	var wg sync.WaitGroup

	// Generate heavy traffic for hot keys
	fmt.Println("Generating heavy traffic for hot keys...")
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				// Access hot keys frequently
				for _, key := range hotKeys {
					item, err := client.Get(key)
					if err != nil && err != memcache.ErrCacheMiss {
						log.Printf("Worker %d: Error getting %s: %v", workerID, key, err)
					} else if item != nil && j%15 == 0 {
						fmt.Printf("Worker %d: Got %s = %s\n", workerID, key, string(item.Value))
					}
					time.Sleep(15 * time.Millisecond)
				}

				// Occasional writes
				if j%10 == 0 {
					key := hotKeys[j%len(hotKeys)]
					newItem := &memcache.Item{
						Key:        key,
						Value:      []byte(fmt.Sprintf("updated_by_worker_%d_at_%d", workerID, j)),
						Expiration: int32(time.Now().Add(time.Hour).Unix()),
					}
					client.Set(newItem)
				}

				// Access normal keys occasionally
				if j%20 == 0 {
					for _, key := range normalKeys {
						client.Get(key)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	fmt.Println("✓ Traffic simulation completed")
	fmt.Println("Hot keys should now be detected and policies applied")
	fmt.Println("Check the monitoring endpoints to see results")
}
