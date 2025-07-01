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

	if err := keyflare.Start(); err != nil {
		t.Fatalf("Failed to start KeyFlare: %v", err)
	}
	defer keyflare.Stop()
}
