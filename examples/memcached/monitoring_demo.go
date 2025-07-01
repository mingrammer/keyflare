package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/mingrammer/keyflare"
	memcachedWrapper "github.com/mingrammer/keyflare/pkg/memcached"
)

// MonitoringExample demonstrates KeyFlare monitoring capabilities with Memcached
func MonitoringExample() {
	fmt.Println("=== Memcached + KeyFlare Monitoring Example ===")

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
				TTL:      100,
				Capacity: 300,
			},
			WhitelistKeys: []string{
				"monitor:mc:hot_item_1",
				"monitor:mc:hot_item_2",
				"monitor:mc:hot_item_3",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9129", // Port for Memcached monitoring
			HotKeyMetricLimit:   8,
			HotKeyHistorySize:   12,
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
			"monitor:mc:hot_item_1",
			"monitor:mc:hot_item_2",
			"monitor:mc:hot_item_3",
			"monitor:mc:normal_item_1",
			"monitor:mc:normal_item_2",
		}

		for _, key := range monitorKeys {
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

		// Generate traffic with different patterns
		var wg sync.WaitGroup

		// Heavy traffic for hot item 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 60; i++ {
				client.Get("monitor:mc:hot_item_1")
				time.Sleep(150 * time.Millisecond)
			}
		}()

		// Medium traffic for hot item 2
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 45; i++ {
				client.Get("monitor:mc:hot_item_2")
				time.Sleep(250 * time.Millisecond)
			}
		}()

		// Light traffic for hot item 3
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 30; i++ {
				client.Get("monitor:mc:hot_item_3")
				time.Sleep(400 * time.Millisecond)
			}
		}()

		// Occasional access to normal items
		wg.Add(1)
		go func() {
			defer wg.Done()
			normalKeys := []string{"monitor:mc:normal_item_1", "monitor:mc:normal_item_2"}
			for i := 0; i < 15; i++ {
				for _, key := range normalKeys {
					client.Get(key)
				}
				time.Sleep(1200 * time.Millisecond)
			}
		}()

		// Some write operations
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				item := &memcache.Item{
					Key:        "monitor:mc:hot_item_1",
					Value:      []byte(fmt.Sprintf("updated_value_%d", i)),
					Expiration: int32(time.Now().Add(time.Hour).Unix()),
				}
				client.Set(item)
				time.Sleep(2 * time.Second)
			}
		}()

		// Wait for traffic generation to complete
		wg.Wait()

		fmt.Println("\n--- Traffic generation completed ---")
		fmt.Println("Continue monitoring for 25 seconds...")
		time.Sleep(25 * time.Second)
	}()

	// Wait for signal
	<-sigChan
	fmt.Println("\nReceived interrupt signal, shutting down gracefully...")

	fmt.Println("\n✅ Memcached Monitoring Demo completed!")
	fmt.Println("\n📊 Monitoring endpoints:")
	fmt.Println("• Prometheus metrics: http://localhost:9129/metrics")
	fmt.Println("• Hot keys API: http://localhost:9129/hot-keys")
}

func printMonitoringInfo() {
	ticker := time.NewTicker(7 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n=== KeyFlare Memcached Monitoring ===")
		fmt.Println("Metrics Server: http://localhost:9129/metrics")
		fmt.Println("Hot Keys API: http://localhost:9129/hot-keys")

		// Fetch and display hot keys
		resp, err := http.Get("http://localhost:9129/hot-keys?limit=5")
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

		// Show Prometheus metrics status
		resp2, err := http.Get("http://localhost:9129/metrics")
		if err == nil {
			defer resp2.Body.Close()
			fmt.Printf("Prometheus Metrics Status: %s\n", resp2.Status)
		}

		fmt.Println("===================================")
	}
}
