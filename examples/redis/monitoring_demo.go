package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/mingrammer/keyflare"
	redisWrapper "github.com/mingrammer/keyflare/pkg/redis"
)

// MonitoringExample demonstrates KeyFlare monitoring capabilities with Redis
func MonitoringExample() {
	fmt.Println("=== Redis + KeyFlare Monitoring Example ===")

	// Initialize KeyFlare with basic configuration for monitoring
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          15,
			DecayFactor:   0.94,
			DecayInterval: 35,
			HotThreshold:  40,
		}),
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.LocalCache,
			Parameters: keyflare.LocalCacheParams{
				TTL:      120,
				Capacity: 500,
			},
			WhitelistKeys: []string{
				"monitor:hot_key_1",
				"monitor:hot_key_2",
				"monitor:hot_key_3",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9123", // Different port for monitoring demo
			HotKeyMetricLimit:   10,
			HotKeyHistorySize:   15,
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

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start monitoring info printer
	go printMonitoringInfo()

	fmt.Println("\nStarting monitoring demonstration...")
	fmt.Println("Press Ctrl+C to stop gracefully")

	// Generate varied traffic for monitoring demonstration in goroutine
	go func() {
		fmt.Println("\n--- Generating traffic for monitoring demonstration ---")

		// Set initial data
		monitorKeys := []string{
			"monitor:hot_key_1",
			"monitor:hot_key_2",
			"monitor:hot_key_3",
			"monitor:normal_key_1",
			"monitor:normal_key_2",
		}

		for _, key := range monitorKeys {
			err := client.Set(ctx, key, fmt.Sprintf("value_for_%s", key), time.Hour).Err()
			if err != nil {
				log.Printf("Failed to set key %s: %v", key, err)
			}
		}

		// Generate traffic with different patterns
		var wg sync.WaitGroup

		// Heavy traffic for hot key 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 80; i++ {
				client.Get(ctx, "monitor:hot_key_1")
				time.Sleep(100 * time.Millisecond)
			}
		}()

		// Medium traffic for hot key 2
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 60; i++ {
				client.Get(ctx, "monitor:hot_key_2")
				time.Sleep(200 * time.Millisecond)
			}
		}()

		// Light traffic for hot key 3
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				client.Get(ctx, "monitor:hot_key_3")
				time.Sleep(300 * time.Millisecond)
			}
		}()

		// Occasional access to normal keys
		wg.Add(1)
		go func() {
			defer wg.Done()
			normalKeys := []string{"monitor:normal_key_1", "monitor:normal_key_2"}
			for i := 0; i < 20; i++ {
				for _, key := range normalKeys {
					client.Get(ctx, key)
				}
				time.Sleep(1 * time.Second)
			}
		}()

		// Wait for traffic generation to complete
		wg.Wait()

		fmt.Println("\n--- Traffic generation completed ---")
		fmt.Println("Continue monitoring for 30 seconds...")
		time.Sleep(30 * time.Second)
	}()

	// Wait for signal
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down gracefully...")

	fmt.Println("\n✅ Monitoring Demo completed!")
	fmt.Println("\n📊 Monitoring endpoints:")
	fmt.Println("• Prometheus metrics: http://localhost:9123/metrics")
	fmt.Println("• Hot keys API: http://localhost:9123/hot-keys")
}

func printMonitoringInfo() {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n=== KeyFlare Monitoring Dashboard ===")
		fmt.Println("Metrics Server: http://localhost:9123/metrics")
		fmt.Println("Hot Keys API: http://localhost:9123/hot-keys")

		// Fetch and display hot keys
		resp, err := http.Get("http://localhost:9123/hot-keys?limit=5")
		if err != nil {
			fmt.Printf("Failed to fetch hot keys: %v\n", err)
		} else {
			defer resp.Body.Close()
			fmt.Printf("Hot Keys API Status: %s\n", resp.Status)

			if resp.StatusCode == 200 {
				body, err := io.ReadAll(resp.Body)
				if err == nil {
					fmt.Printf("Hot Keys Response: %s\n", string(body))
				}
			}
		}

		// Show some Prometheus metrics
		resp2, err := http.Get("http://localhost:9123/metrics")
		if err == nil {
			defer resp2.Body.Close()
			fmt.Printf("Prometheus Metrics Status: %s\n", resp2.Status)
		}

		fmt.Println("===================================")
	}
}
