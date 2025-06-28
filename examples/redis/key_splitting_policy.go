package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mingrammer/keyflare"
	redisWrapper "github.com/mingrammer/keyflare/pkg/redis"
)

// KeySplittingPolicyExample demonstrates KeyFlare with Redis Cluster using Key Splitting Policy
func KeySplittingPolicyExample(runSimulation bool) {
	fmt.Println("=== Redis + Key Splitting Policy Example ===")

	// Initialize KeyFlare with Key Splitting Policy
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          50,
			DecayFactor:   0.98,
			DecayInterval: 60,
			HotThreshold:  80,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.KeySplitting,
			Parameters: keyflare.KeySplittingParams{
				Shards: 5, // Split hot keys across 5 shards
			},
			WhitelistKeys: []string{
				"counter:global:requests",
				"leaderboard:top_players",
				"analytics:real_time:stats",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9123",
			HotKeyMetricLimit:   20,
			EnableAPI:           true,
		}),
	)
	if err != nil {
		log.Fatal("Failed to initialize KeyFlare:", err)
	}

	if err := keyflare.Start(); err != nil {
		log.Fatal("Failed to start KeyFlare:", err)
	}
	defer func() {
		keyflare.Stop()
		keyflare.Shutdown()
	}()

	// Create Redis Cluster client
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: []string{
			"localhost:7001",
			"localhost:7002",
			"localhost:7003",
		},
	})

	// Wrap with KeyFlare
	client, err := redisWrapper.Wrap(rdb)
	if err != nil {
		log.Fatal("Failed to wrap Redis client:", err)
	}

	ctx := context.Background()

	// Test connection
	pong, err := client.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis cluster:", err)
	}
	fmt.Printf("Redis Cluster connected: %s\n", pong)

	// Usage demonstration
	fmt.Println("\n--- Usage Example ---")

	// Normal Redis operations work exactly the same
	err = client.Set(ctx, "counter:global:requests", "12345", time.Hour).Err()
	if err != nil {
		log.Printf("Failed to set key: %v", err)
		return
	}
	fmt.Println("✓ client.Set() works exactly the same")

	val, err := client.Get(ctx, "counter:global:requests").Result()
	if err != nil {
		log.Printf("Failed to get key: %v", err)
		return
	}
	fmt.Printf("✓ client.Get() retrieved: %s\n", val)

	// Monitoring
	fmt.Println("\n--- Monitoring ---")
	fmt.Println("Metrics: http://localhost:9123/metrics")
	fmt.Println("Hot Keys API: http://localhost:9123/hot-keys")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if runSimulation {
		fmt.Println("\nStarting traffic simulation...")
		fmt.Println("Press Ctrl+C to stop gracefully")

		// Run simulation in goroutine
		go func() {
			RunTrafficSimulation(ctx, client)

			// Show shard keys after simulation
			fmt.Println("\n--- Checking for shard keys ---")
			for i := 0; i < 5; i++ {
				shardKey := fmt.Sprintf("counter:global:requests:shard:%d", i)
				val, err := client.Get(ctx, shardKey).Result()
				if err == nil {
					fmt.Printf("Shard key found: %s = %s\n", shardKey, val)
				}
			}
		}()

		// Wait for signal
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
	} else {
		fmt.Println("\nℹ️  Run with simulation to see key splitting in action")
		fmt.Println("📊 Monitor metrics at: http://localhost:9123/metrics")
		fmt.Println("🔥 Check hot keys at: http://localhost:9123/hot-keys")
		fmt.Println("\nPress Ctrl+C to exit")

		// Wait for signal
		<-sigChan
		fmt.Println("\nShutting down...")
	}

	fmt.Println("\nKey Splitting Policy Setup Complete!")
	fmt.Println("\nHow it works:")
	fmt.Println("• Hot keys in whitelist get split across multiple shard keys")
	fmt.Println("• Reads use look-aside pattern (shard first, fallback to original)")
	fmt.Println("• Writes replicate to all shards asynchronously")
	fmt.Println("• Reduces contention on individual keys")
	fmt.Println("\nNext steps:")
	fmt.Println("• Adjust Shards parameter based on your load")
	fmt.Println("• Configure WhitelistKeys for contended keys")
	fmt.Println("• Monitor shard distribution via metrics")
}
