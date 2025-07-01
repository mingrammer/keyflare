package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/rueidis"

	"github.com/mingrammer/keyflare"
	rueidisWrapper "github.com/mingrammer/keyflare/pkg/rueidis"
)

// LocalCachePolicyExample demonstrates KeyFlare with Rueidis using Local Cache Policy
func LocalCachePolicyExample(runSimulation bool) {
	fmt.Println("=== Rueidis + Local Cache Policy Example ===")

	// Initialize KeyFlare with Local Cache Policy
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          40,
			DecayFactor:   0.96,
			DecayInterval: 50,
			HotThreshold:  80,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.LocalCache,
			Parameters: keyflare.LocalCacheParams{
				TTL:          200, // 3.3 minutes local cache
				Jitter:       0.2,
				Capacity:     600,
				RefreshAhead: 0.75,
			},
			WhitelistKeys: []string{
				"analytics:realtime:dashboard",
				"config:feature:flags",
				"leaderboard:top:users",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9126",
			HotKeyMetricLimit:   12,
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

	// Create Rueidis client
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: []string{"localhost:7001", "localhost:7002", "localhost:7003"},
	})
	if err != nil {
		log.Fatal("Failed to create Rueidis client:", err)
	}
	defer client.Close()

	// Wrap with KeyFlare
	wrappedClient, err := rueidisWrapper.Wrap(client)
	if err != nil {
		log.Fatal("Failed to wrap Rueidis client:", err)
	}

	ctx := context.Background()

	// Test connection
	pong := wrappedClient.Do(ctx, wrappedClient.B().Ping().Build())
	if pong.Error() != nil {
		log.Fatal("Failed to connect to Redis:", pong.Error())
	}
	fmt.Println("Rueidis connected successfully:", pong.String())

	// Usage demonstration
	fmt.Println("\n--- Usage Example ---")

	// Normal Rueidis operations work exactly the same
	hotKey := "analytics:realtime:dashboard"
	setCmd := wrappedClient.B().Set().Key(hotKey).Value("real_time_analytics_data").Ex(time.Hour).Build()
	result := wrappedClient.Do(ctx, setCmd)
	if result.Error() != nil {
		log.Printf("Failed to set data: %v", result.Error())
		return
	}
	fmt.Println("✓ wrappedClient.Do(SET) works exactly the same")

	getCmd := wrappedClient.B().Get().Key(hotKey).Build()
	result = wrappedClient.Do(ctx, getCmd)
	if result.Error() != nil && !rueidis.IsRedisNil(result.Error()) {
		log.Printf("Failed to get data: %v", result.Error())
		return
	}
	if !rueidis.IsRedisNil(result.Error()) {
		val, _ := result.ToString()
		fmt.Printf("✓ wrappedClient.Do(GET) retrieved: %s\n", val)
	}

	// Monitoring
	fmt.Println("\n--- Monitoring ---")
	fmt.Println("Metrics: http://localhost:9126/metrics")
	fmt.Println("Hot Keys API: http://localhost:9126/hot-keys")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if runSimulation {
		fmt.Println("\nStarting traffic simulation...")
		fmt.Println("Press Ctrl+C to stop gracefully")

		// Run simulation in goroutine
		go RunTrafficSimulation(ctx, wrappedClient)

		// Wait for signal
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
	} else {
		fmt.Println("\nℹ️  Run with simulation to see hot key detection in action")
		fmt.Println("📊 Monitor metrics at: http://localhost:9126/metrics")
		fmt.Println("🔥 Check hot keys at: http://localhost:9126/hot-keys")
		fmt.Println("\nPress Ctrl+C to exit")

		// Wait for signal
		<-sigChan
		fmt.Println("\nShutting down...")
	}

	fmt.Println("\nRueidis Local Cache Policy Setup Complete!")
	fmt.Println("\nHow it works:")
	fmt.Println("• Keys in whitelist that exceed HotThreshold get cached locally")
	fmt.Println("• Cache hits serve from memory (faster than Redis)")
	fmt.Println("• Reduces Redis server load")
	fmt.Println("• Automatic cache population on misses")
	fmt.Println("\nNext steps:")
	fmt.Println("• Adjust HotThreshold based on your traffic")
	fmt.Println("• Configure WhitelistKeys for your hot keys")
	fmt.Println("• Monitor via /hot-keys API to tune parameters")
}
