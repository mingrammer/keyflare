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

// LocalCachePolicyExample demonstrates KeyFlare with Redis Cluster using Local Cache Policy
func LocalCachePolicyExample(runSimulation bool) {
	fmt.Println("=== Redis + Local Cache Policy Example ===")

	// Initialize KeyFlare with Local Cache Policy
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          50,
			DecayFactor:   0.98,
			DecayInterval: 60,
			HotThreshold:  80,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.LocalCache,
			Parameters: keyflare.LocalCacheParams{
				TTL:          300,  // 5 minutes local cache
				Jitter:       0.1,  // 10% TTL randomization
				Capacity:     1000, // Cache up to 1000 items
				RefreshAhead: 0.8,  // Refresh at 80% of TTL
			},
			WhitelistKeys: []string{
				"user:profile:popular_user",
				"product:details:trending_item",
				"session:heavy_user_session",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9122",
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
	err = client.Set(ctx, "user:profile:popular_user", "user_data_value", time.Hour).Err()
	if err != nil {
		log.Printf("Failed to set key: %v", err)
		return
	}
	fmt.Println("✓ client.Set() works exactly the same")

	val, err := client.Get(ctx, "user:profile:popular_user").Result()
	if err != nil {
		log.Printf("Failed to get key: %v", err)
		return
	}
	fmt.Printf("✓ client.Get() retrieved: %s\n", val)

	// Monitoring
	fmt.Println("\n--- Monitoring ---")
	fmt.Println("Metrics: http://localhost:9122/metrics")
	fmt.Println("Hot Keys API: http://localhost:9122/hot-keys")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if runSimulation {
		fmt.Println("\nStarting traffic simulation...")
		fmt.Println("Press Ctrl+C to stop gracefully")

		// Run simulation in goroutine
		go RunTrafficSimulation(ctx, client)

		// Wait for signal
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
	} else {
		fmt.Println("\nℹ️  Run with simulation to see hot key detection in action")
		fmt.Println("📊 Monitor metrics at: http://localhost:9122/metrics")
		fmt.Println("🔥 Check hot keys at: http://localhost:9122/hot-keys")
		fmt.Println("\nPress Ctrl+C to exit")

		// Wait for signal
		<-sigChan
		fmt.Println("\nShutting down...")
	}

	fmt.Println("\nLocal Cache Policy Setup Complete!")
	fmt.Println("\nHow it works:")
	fmt.Println("• Keys in whitelist that exceed HotThreshold get cached locally")
	fmt.Println("• Cache hits serve from memory (faster response)")
	fmt.Println("• Cache misses populate cache asynchronously")
	fmt.Println("• TTL jitter prevents cache stampede")
	fmt.Println("\nNext steps:")
	fmt.Println("• Adjust HotThreshold based on your traffic")
	fmt.Println("• Configure WhitelistKeys for your hot keys")
	fmt.Println("• Monitor via /hot-keys API to tune parameters")
}
