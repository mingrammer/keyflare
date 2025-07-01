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

// LocalCachePolicyExample demonstrates KeyFlare with Memcached using Local Cache Policy
func LocalCachePolicyExample(runSimulation bool) {
	fmt.Println("=== Memcached + Local Cache Policy Example ===")

	// Initialize KeyFlare with Local Cache Policy
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          30,
			DecayFactor:   0.97,
			DecayInterval: 45,
			HotThreshold:  80,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.LocalCache,
			Parameters: keyflare.LocalCacheParams{
				TTL:          240, // 4 minutes local cache
				Jitter:       0.15,
				Capacity:     800,
				RefreshAhead: 0.75,
			},
			WhitelistKeys: []string{
				"session:user:popular",
				"catalog:item:trending",
				"config:app:global",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9124",
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
		Key:        "session:user:popular",
		Value:      []byte("popular_user_session_data"),
		Expiration: int32(time.Now().Add(time.Hour).Unix()),
	}
	err = client.Set(item)
	if err != nil {
		log.Printf("Failed to set item: %v", err)
		return
	}
	fmt.Println("✓ client.Set() works exactly the same")

	retrievedItem, err := client.Get("session:user:popular")
	if err != nil {
		log.Printf("Failed to get item: %v", err)
		return
	}
	fmt.Printf("✓ client.Get() retrieved: %s\n", string(retrievedItem.Value))

	// Monitoring
	fmt.Println("\n--- Monitoring ---")
	fmt.Println("Metrics: http://localhost:9124/metrics")
	fmt.Println("Hot Keys API: http://localhost:9124/hot-keys")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if runSimulation {
		fmt.Println("\nStarting traffic simulation...")
		fmt.Println("Press Ctrl+C to stop gracefully")

		// Run simulation in goroutine
		go RunTrafficSimulation(client)

		// Wait for signal
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down gracefully...")
	} else {
		fmt.Println("\nℹ️  Run with simulation to see hot key detection in action")
		fmt.Println("📊 Monitor metrics at: http://localhost:9124/metrics")
		fmt.Println("🔥 Check hot keys at: http://localhost:9124/hot-keys")
		fmt.Println("\nPress Ctrl+C to exit")

		// Wait for signal
		<-sigChan
		fmt.Println("\nShutting down...")
	}

	fmt.Println("\nMemcached Local Cache Policy Setup Complete!")
	fmt.Println("\nHow it works:")
	fmt.Println("• Keys in whitelist that exceed HotThreshold get cached locally")
	fmt.Println("• Cache hits serve from memory (faster than Memcached)")
	fmt.Println("• Reduces Memcached server load")
	fmt.Println("• Automatic cache population on misses")
	fmt.Println("\nNext steps:")
	fmt.Println("• Adjust HotThreshold based on your traffic")
	fmt.Println("• Configure WhitelistKeys for your hot keys")
	fmt.Println("• Monitor via /hot-keys API to tune parameters")
}
