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

// KeySplittingPolicyExample demonstrates KeyFlare with Rueidis using Key Splitting Policy
func KeySplittingPolicyExample(runSimulation bool) {
	fmt.Println("=== Rueidis + Key Splitting Policy Example ===")

	// Initialize KeyFlare with Key Splitting Policy
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          40,
			DecayFactor:   0.96,
			DecayInterval: 50,
			HotThreshold:  80,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.KeySplitting,
			Parameters: keyflare.KeySplittingParams{
				Shards: 6, // Split hot keys across 6 shards
			},
			WhitelistKeys: []string{
				"stream:live:events",
				"counter:api:requests",
				"cache:realtime:feed",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9127",
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
	defer func() {
		keyflare.Stop()
		keyflare.Shutdown()
	}()

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
	hotKey := "stream:live:events"
	setCmd := wrappedClient.B().Set().Key(hotKey).Value("live_event_data_stream").Ex(time.Hour).Build()
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
	fmt.Println("Metrics: http://localhost:9127/metrics")
	fmt.Println("Hot Keys API: http://localhost:9127/hot-keys")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if runSimulation {
		fmt.Println("\nStarting traffic simulation...")
		fmt.Println("Press Ctrl+C to stop gracefully")

		// Run simulation in goroutine
		go func() {
			RunTrafficSimulation(ctx, wrappedClient)

			// Check for shard keys after simulation
			fmt.Println("\n--- Checking for shard keys ---")
			for i := 0; i < 6; i++ {
				shardKey := fmt.Sprintf("%s:shard:%d", hotKey, i)
				getCmd := wrappedClient.B().Get().Key(shardKey).Build()
				result := wrappedClient.Do(ctx, getCmd)
				if result.Error() == nil {
					val, _ := result.ToString()
					fmt.Printf("Shard key found: %s = %s\n", shardKey, val)
				}
			}
		}()

		// Wait for signal
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
	} else {
		fmt.Println("\nℹ️  Run with simulation to see key splitting in action")
		fmt.Println("📊 Monitor metrics at: http://localhost:9127/metrics")
		fmt.Println("🔥 Check hot keys at: http://localhost:9127/hot-keys")
		fmt.Println("\nPress Ctrl+C to exit")

		// Wait for signal
		<-sigChan
		fmt.Println("\nShutting down...")
	}

	fmt.Println("\nRueidis Key Splitting Policy Setup Complete!")
	fmt.Println("\nHow it works:")
	fmt.Println("• Hot keys in whitelist get split across multiple shard keys")
	fmt.Println("• Reduces contention on individual Redis keys")
	fmt.Println("• Load balancing improves cache performance")
	fmt.Println("• Automatic shard management")
	fmt.Println("\nNext steps:")
	fmt.Println("• Adjust Shards parameter based on your load")
	fmt.Println("• Configure WhitelistKeys for contended keys")
	fmt.Println("• Monitor shard distribution via metrics")
}
