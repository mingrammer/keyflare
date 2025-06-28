package keyflare_test

import (
	"testing"

	"github.com/mingrammer/keyflare"
)

func TestNew_WithDefaultOptions(t *testing.T) {
	err := keyflare.New()
	if err != nil {
		t.Fatalf("Failed to create KeyFlare with default options: %v", err)
	}
	defer keyflare.Shutdown()

	if err := keyflare.Start(); err != nil {
		t.Fatalf("Failed to start KeyFlare: %v", err)
	}
	defer keyflare.Stop()
}

func TestNew_WithCustomDetectorOptions(t *testing.T) {
	err := keyflare.New(
		keyflare.WithDetectorOptions(keyflare.DetectorOptions{
			TopK:          50,
			DecayFactor:   0.95,
			DecayInterval: 30,
			HotThreshold:  10,
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create KeyFlare with custom detector options: %v", err)
	}
	defer keyflare.Shutdown()

	if err := keyflare.Start(); err != nil {
		t.Fatalf("Failed to start KeyFlare: %v", err)
	}
	defer keyflare.Stop()
}

func TestNew_WithLocalCachePolicy(t *testing.T) {
	err := keyflare.New(
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.LocalCache,
			Parameters: keyflare.LocalCacheParams{
				TTL:          120,
				Jitter:       0.1,
				Capacity:     500,
				RefreshAhead: 0.9,
			},
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create KeyFlare with local cache policy: %v", err)
	}
	defer keyflare.Shutdown()

	if err := keyflare.Start(); err != nil {
		t.Fatalf("Failed to start KeyFlare: %v", err)
	}
	defer keyflare.Stop()
}

func TestNew_WithKeySplittingPolicy(t *testing.T) {
	err := keyflare.New(
		keyflare.WithPolicyOptions(keyflare.PolicyOptions{
			Type: keyflare.KeySplitting,
			Parameters: keyflare.KeySplittingParams{
				Shards: 5,
			},
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create KeyFlare with key splitting policy: %v", err)
	}
	defer keyflare.Shutdown()

	if err := keyflare.Start(); err != nil {
		t.Fatalf("Failed to start KeyFlare: %v", err)
	}
	defer keyflare.Stop()
}

func TestNew_MultipleInstancesNotAllowed(t *testing.T) {
	err := keyflare.New()
	if err != nil {
		t.Fatalf("Failed to create first KeyFlare instance: %v", err)
	}

	// Try to create a second instance - should fail
	err = keyflare.New()
	if err == nil {
		t.Fatal("Expected error when creating second instance without shutdown")
	}

	// Cleanup
	keyflare.Shutdown()

	// Now we should be able to create a new instance
	err = keyflare.New()
	if err != nil {
		t.Fatalf("Failed to create new instance after shutdown: %v", err)
	}
	keyflare.Shutdown()
}

func TestLifecycle_StartStopShutdown(t *testing.T) {
	// Test starting without New()
	err := keyflare.Start()
	if err == nil {
		t.Fatal("Expected error when starting without New()")
	}

	// Create instance
	err = keyflare.New()
	if err != nil {
		t.Fatalf("Failed to create KeyFlare: %v", err)
	}

	// Test stopping without starting
	err = keyflare.Stop()
	if err == nil {
		t.Fatal("Expected error when stopping without starting")
	}

	// Start
	err = keyflare.Start()
	if err != nil {
		t.Fatalf("Failed to start KeyFlare: %v", err)
	}

	// Stop
	err = keyflare.Stop()
	if err != nil {
		t.Fatalf("Failed to stop KeyFlare: %v", err)
	}

	// Shutdown
	err = keyflare.Shutdown()
	if err != nil {
		t.Fatalf("Failed to shutdown KeyFlare: %v", err)
	}
}
