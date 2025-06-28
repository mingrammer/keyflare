package detector_test

import (
	"testing"
	"time"

	"github.com/mingrammer/keyflare/internal/detector"
)

func TestDetector_NewWithDefaults(t *testing.T) {
	config := detector.Config{}
	d := detector.New(config)
	if d == nil {
		t.Fatal("Expected detector to be created with default config")
	}
}

func TestDetector_NewWithCustomConfig(t *testing.T) {
	config := detector.Config{
		ErrorRate:     0.001,
		TopK:          50,
		DecayFactor:   0.95,
		DecayInterval: 30 * time.Second,
		HotThreshold:  100,
	}

	d := detector.New(config)
	if d == nil {
		t.Fatal("Expected detector to be created with custom config")
	}
}

func TestDetector_IncrementAndCount(t *testing.T) {
	config := detector.Config{
		TopK:          10,
		DecayInterval: 60 * time.Second,
	}
	d := detector.New(config)

	// Increment keys
	for i := 0; i < 100; i++ {
		d.Increment("key1", 1)
	}
	for i := 0; i < 50; i++ {
		d.Increment("key2", 1)
	}

	// Check counts
	count1 := d.GetCount("key1")
	count2 := d.GetCount("key2")

	if count1 < count2 {
		t.Errorf("Expected key1 count (%d) to be greater than key2 count (%d)", count1, count2)
	}
}

func TestDetector_TopKResults(t *testing.T) {
	config := detector.Config{
		TopK:          3,
		DecayInterval: 60 * time.Second,
	}
	d := detector.New(config)

	// Create keys with different frequencies
	for i := 0; i < 100; i++ {
		d.Increment("popular", 1)
	}
	for i := 0; i < 50; i++ {
		d.Increment("medium", 1)
	}
	for i := 0; i < 10; i++ {
		d.Increment("rare", 1)
	}

	topK := d.TopK()

	// Check that we got the correct number of top keys
	if len(topK) > config.TopK {
		t.Errorf("Expected at most %d keys, got %d", config.TopK, len(topK))
	}

	// Check that keys are sorted by count
	for i := 1; i < len(topK); i++ {
		if topK[i-1].Count < topK[i].Count {
			t.Errorf("Keys not sorted: %v", topK)
		}
	}

	// Check that the most popular key is first
	if len(topK) > 0 && topK[0].Key != "popular" {
		t.Errorf("Expected 'popular' to be the top key, got %s", topK[0].Key)
	}
}

func TestDetector_IsHotWithThreshold(t *testing.T) {
	config := detector.Config{
		TopK:          10,
		HotThreshold:  50,
		DecayInterval: 60 * time.Second,
	}
	d := detector.New(config)

	// Create a hot key
	for i := 0; i < 100; i++ {
		d.Increment("hot_key", 1)
	}

	// Create a cold key
	for i := 0; i < 10; i++ {
		d.Increment("cold_key", 1)
	}

	if !d.IsHot("hot_key") {
		t.Error("Expected hot_key to be hot")
	}

	if d.IsHot("cold_key") {
		t.Error("Expected cold_key to not be hot")
	}
}

func TestDetector_Reset(t *testing.T) {
	config := detector.Config{
		TopK:          10,
		DecayInterval: 60 * time.Second,
	}
	d := detector.New(config)

	// Add some keys
	d.Increment("key1", 100)
	d.Increment("key2", 50)

	// Verify counts before reset
	if d.GetCount("key1") == 0 {
		t.Error("Expected key1 to have non-zero count before reset")
	}

	// Reset
	d.Reset()

	// Verify counts after reset
	if d.GetCount("key1") != 0 {
		t.Error("Expected key1 to have zero count after reset")
	}
	if d.GetCount("key2") != 0 {
		t.Error("Expected key2 to have zero count after reset")
	}

	// Verify top K is empty
	topK := d.TopK()
	if len(topK) != 0 {
		t.Errorf("Expected empty top K after reset, got %d keys", len(topK))
	}
}
