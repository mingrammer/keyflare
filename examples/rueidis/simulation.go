package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	rueidisWrapper "github.com/mingrammer/keyflare/pkg/rueidis"
	"github.com/redis/rueidis"
)

// RunTrafficSimulation generates varied traffic patterns to demonstrate KeyFlare functionality
func RunTrafficSimulation(ctx context.Context, client *rueidisWrapper.Wrapper) {
	fmt.Println("\n--- Running Traffic Simulation ---")

	// Hot keys that should trigger KeyFlare policies
	hotKeys := []string{
		"analytics:realtime:dashboard",
		"stream:live:events",
		"config:feature:flags",
	}

	// Normal keys with moderate access
	normalKeys := []string{
		"analytics:hourly:report",
		"stream:archive:data",
		"config:user:settings",
	}

	// Set initial data using Rueidis command builder
	fmt.Println("Setting up initial data...")
	for _, key := range append(hotKeys, normalKeys...) {
		setCmd := client.B().Set().Key(key).Value(fmt.Sprintf("value_for_%s", key)).Ex(time.Hour).Build()
		result := client.Do(ctx, setCmd)
		if result.Error() != nil {
			log.Printf("Failed to set key %s: %v", key, result.Error())
		}
	}

	var wg sync.WaitGroup

	// Generate heavy traffic for hot keys
	fmt.Println("Generating heavy traffic for hot keys...")
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				// Access hot keys frequently
				for _, key := range hotKeys {
					getCmd := client.B().Get().Key(key).Build()
					result := client.Do(ctx, getCmd)
					if result.Error() != nil && !rueidis.IsRedisNil(result.Error()) {
						log.Printf("Worker %d: Error getting %s: %v", workerID, key, result.Error())
					} else if !rueidis.IsRedisNil(result.Error()) && j%15 == 0 {
						val, _ := result.ToString()
						fmt.Printf("Worker %d: Got %s = %s\n", workerID, key, val)
					}
					time.Sleep(20 * time.Millisecond)
				}

				// Occasional multi-command operations
				if j%15 == 0 {
					cmd1 := client.B().Get().Key(hotKeys[0]).Build()
					cmd2 := client.B().Get().Key(hotKeys[1]).Build()
					client.DoMulti(ctx, cmd1, cmd2)
				}

				// Occasional writes
				if j%10 == 0 {
					key := hotKeys[j%len(hotKeys)]
					setCmd := client.B().Set().Key(key).Value(fmt.Sprintf("updated_by_worker_%d_at_%d", workerID, j)).Ex(time.Hour).Build()
					client.Do(ctx, setCmd)
				}

				// Access normal keys occasionally
				if j%20 == 0 {
					for _, key := range normalKeys {
						getCmd := client.B().Get().Key(key).Build()
						client.Do(ctx, getCmd)
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
