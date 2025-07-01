package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bradfitz/gomemcache/memcache"

	"github.com/mingrammer/keyflare"
	memcachedWrapper "github.com/mingrammer/keyflare/pkg/memcached"
)

// KeySplittingPolicyExample demonstrates KeyFlare with Memcached using Key Splitting Policy
func KeySplittingPolicyExample(runSimulation bool) {
	fmt.Println("=== Memcached + Key Splitting Policy Example ===")

	// Initialize KeyFlare with Key Splitting Policy
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          30,
			DecayFactor:   0.97,
			DecayInterval: 45,
			HotThreshold:  80,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.KeySplitting,
			Parameters: keyflare.KeySplittingParams{
				Shards: 4, // Split hot keys across 4 shards
			},
			WhitelistKeys: []string{
				"counter:global:views",
				"analytics:live:data",
				"leaderboard:realtime",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9125",
			HotKeyMetricLimit:   15,
			EnableAPI:           true,
		}),
	)
	if err != nil {
		log.Fatal("Failed to initialize KeyFlare:", err)
	}

	if err := keyflare.Start(); err != nil {
		log.Fatal("Failed to start KeyFlare:", err)
	}
	defer keyflare.Stop()

	// Create Memcached client
	mc := memcache.New("localhost:11211")

	// Wrap with KeyFlare
	client, err := memcachedWrapper.Wrap(mc)
	if err != nil {
		log.Fatal("Failed to wrap Memcached client:", err)
	}

	// Test connection
	if err := client.Ping(); err != nil {
		log.Fatal("Failed to connect to Memcached:", err)
	}
	fmt.Println("Memcached connected successfully")

	// Usage demonstration
	fmt.Println("\n--- Usage Example ---")

	// Normal Memcached operations work exactly the same
	item := &memcache.Item{
		Key:        "counter:global:views",
		Value:      []byte("98765"),
		Expiration: int32(time.Now().Add(time.Hour).Unix()),
	}
	err = client.Set(item)
	if err != nil {
		log.Printf("Failed to set item: %v", err)
		return
	}
	fmt.Println("✓ client.Set() works exactly the same")

	retrievedItem, err := client.Get("counter:global:views")
	if err != nil {
		log.Printf("Failed to get item: %v", err)
		return
	}
	fmt.Printf("✓ client.Get() retrieved: %s\n", string(retrievedItem.Value))

	// Monitoring
	fmt.Println("\n--- Monitoring ---")
	fmt.Println("Metrics: http://localhost:9125/metrics")
	fmt.Println("Hot Keys API: http://localhost:9125/hot-keys")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if runSimulation {
		fmt.Println("\nStarting traffic simulation...")
		fmt.Println("Press Ctrl+C to stop gracefully")

		// Run simulation in goroutine
		go func() {
			RunTrafficSimulation(client)

			// Check for shard keys after simulation
			fmt.Println("\n--- Checking for shard keys ---")
			for i := 0; i < 4; i++ {
				shardKey := fmt.Sprintf("counter:global:views:shard:%d", i)
				item, err := client.Get(shardKey)
				if err == nil && item != nil {
					fmt.Printf("Shard key found: %s = %s\n", shardKey, string(item.Value))
				}
			}
		}()

		// Wait for signal
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
	} else {
		fmt.Println("\nℹ️  Run with simulation to see key splitting in action")
		fmt.Println("📊 Monitor metrics at: http://localhost:9125/metrics")
		fmt.Println("🔥 Check hot keys at: http://localhost:9125/hot-keys")
		fmt.Println("\nPress Ctrl+C to exit")

		// Wait for signal
		<-sigChan
		fmt.Println("\nShutting down...")
	}

	fmt.Println("\nMemcached Key Splitting Policy Setup Complete!")
	fmt.Println("\nHow it works:")
	fmt.Println("• Hot keys in whitelist get split across multiple shard keys")
	fmt.Println("• Reduces contention on individual Memcached keys")
	fmt.Println("• Load balancing improves cache performance")
	fmt.Println("• Automatic shard management")
	fmt.Println("\nNext steps:")
	fmt.Println("• Adjust Shards parameter based on your load")
	fmt.Println("• Configure WhitelistKeys for contended keys")
	fmt.Println("• Monitor shard distribution via metrics")
}
