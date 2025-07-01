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

	"github.com/redis/rueidis"

	"github.com/mingrammer/keyflare"
	rueidisWrapper "github.com/mingrammer/keyflare/pkg/rueidis"
)

// MonitoringExample demonstrates KeyFlare monitoring capabilities with Rueidis
func MonitoringExample() {
	fmt.Println("=== Rueidis + KeyFlare Monitoring Example ===")

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
				TTL:      150,
				Capacity: 400,
			},
			WhitelistKeys: []string{
				"monitor:rueidis:stream_1",
				"monitor:rueidis:stream_2",
				"monitor:rueidis:analytics",
			},
		}),
		keyflare.WithMetricsOptions(keyflare.MetricsOptions{
			MetricServerAddress: ":9130", // Port for Rueidis monitoring
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

		// Set initial data using Rueidis command builder
		monitorKeys := []string{
			"monitor:rueidis:stream_1",
			"monitor:rueidis:stream_2",
			"monitor:rueidis:analytics",
			"monitor:rueidis:normal_1",
			"monitor:rueidis:normal_2",
		}

		for _, key := range monitorKeys {
			setCmd := wrappedClient.B().Set().Key(key).Value(fmt.Sprintf("value_for_%s", key)).Ex(time.Hour).Build()
			result := wrappedClient.Do(ctx, setCmd)
			if result.Error() != nil {
				log.Printf("Failed to set key %s: %v", key, result.Error())
			}
		}

		// Generate traffic with different patterns
		var wg sync.WaitGroup

		// Heavy traffic for stream 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 70; i++ {
				getCmd := wrappedClient.B().Get().Key("monitor:rueidis:stream_1").Build()
				wrappedClient.Do(ctx, getCmd)
				time.Sleep(120 * time.Millisecond)
			}
		}()

		// Medium traffic for stream 2
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				getCmd := wrappedClient.B().Get().Key("monitor:rueidis:stream_2").Build()
				wrappedClient.Do(ctx, getCmd)
				time.Sleep(200 * time.Millisecond)
			}
		}()

		// Light traffic for analytics
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 35; i++ {
				getCmd := wrappedClient.B().Get().Key("monitor:rueidis:analytics").Build()
				wrappedClient.Do(ctx, getCmd)
				time.Sleep(350 * time.Millisecond)
			}
		}()

		// Occasional access to normal keys
		wg.Add(1)
		go func() {
			defer wg.Done()
			normalKeys := []string{"monitor:rueidis:normal_1", "monitor:rueidis:normal_2"}
			for i := 0; i < 18; i++ {
				for _, key := range normalKeys {
					getCmd := wrappedClient.B().Get().Key(key).Build()
					wrappedClient.Do(ctx, getCmd)
				}
				time.Sleep(1 * time.Second)
			}
		}()

		// Multi-command operations to demonstrate Rueidis features
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 12; i++ {
				cmd1 := wrappedClient.B().Get().Key("monitor:rueidis:stream_1").Build()
				cmd2 := wrappedClient.B().Get().Key("monitor:rueidis:stream_2").Build()
				wrappedClient.DoMulti(ctx, cmd1, cmd2)
				time.Sleep(1500 * time.Millisecond)
			}
		}()

		// Some update operations
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 8; i++ {
				setCmd := wrappedClient.B().Set().Key("monitor:rueidis:stream_1").Value(fmt.Sprintf("updated_stream_data_%d", i)).Ex(time.Hour).Build()
				wrappedClient.Do(ctx, setCmd)
				time.Sleep(2500 * time.Millisecond)
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

	fmt.Println("\n✅ Rueidis Monitoring Demo completed!")
	fmt.Println("\n📊 Monitoring endpoints:")
	fmt.Println("• Prometheus metrics: http://localhost:9130/metrics")
	fmt.Println("• Hot keys API: http://localhost:9130/hot-keys")
}

func printMonitoringInfo() {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n=== KeyFlare Rueidis Monitoring ===")
		fmt.Println("Metrics Server: http://localhost:9130/metrics")
		fmt.Println("Hot Keys API: http://localhost:9130/hot-keys")

		// Fetch and display hot keys
		resp, err := http.Get("http://localhost:9130/hot-keys?limit=5")
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
		resp2, err := http.Get("http://localhost:9130/metrics")
		if err == nil {
			defer resp2.Body.Close()
			fmt.Printf("Prometheus Metrics Status: %s\n", resp2.Status)
		}

		fmt.Println("===================================")
	}
}
