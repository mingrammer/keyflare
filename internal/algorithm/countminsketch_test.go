package algorithm

import (
	"testing"
)

func TestCountMinSketch_Basic(t *testing.T) {
	// Test with reasonable epsilon and delta values
	cms := NewCountMinSketch(0.01, 0.01) // 1% error rate, 99% confidence

	// Add some items
	cms.Add([]byte("key1"), 5)
	cms.Add([]byte("key2"), 3)
	cms.Add([]byte("key1"), 2) // key1 total should be 7

	// Test estimates
	estimate1 := cms.Estimate([]byte("key1"))
	estimate2 := cms.Estimate([]byte("key2"))
	estimate3 := cms.Estimate([]byte("nonexistent"))

	// CMS guarantees overestimation, so estimate >= actual count
	if estimate1 < 7 {
		t.Errorf("Estimate(key1) = %d, want >= 7", estimate1)
	}

	if estimate2 < 3 {
		t.Errorf("Estimate(key2) = %d, want >= 3", estimate2)
	}

	if estimate3 != 0 {
		t.Errorf("Estimate(nonexistent) = %d, want 0", estimate3)
	}
}

func TestCountMinSketch_DifferentParameters(t *testing.T) {
	tests := []struct {
		name    string
		epsilon float64
		delta   float64
	}{
		{"high precision", 0.001, 0.001},
		{"medium precision", 0.01, 0.01},
		{"low precision", 0.1, 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cms := NewCountMinSketch(tt.epsilon, tt.delta)

			// Basic functionality test
			cms.Add([]byte("test"), 10)
			estimate := cms.Estimate([]byte("test"))

			if estimate < 10 {
				t.Errorf("Estimate should be >= 10, got %d", estimate)
			}
		})
	}
}

func TestCountMinSketch_MultipleKeys(t *testing.T) {
	cms := NewCountMinSketch(0.01, 0.01)

	// Add multiple different keys
	keys := []string{"apple", "banana", "cherry", "date", "elderberry"}
	counts := []uint64{10, 5, 15, 8, 3}

	for i, key := range keys {
		cms.Add([]byte(key), counts[i])
	}

	// Verify estimates
	for i, key := range keys {
		estimate := cms.Estimate([]byte(key))
		if estimate < counts[i] {
			t.Errorf("Key %s: estimate %d should be >= %d", key, estimate, counts[i])
		}
	}
}

func TestCountMinSketch_IncrementalAdds(t *testing.T) {
	cms := NewCountMinSketch(0.01, 0.01)

	key := []byte("incremental")

	// Add in increments
	cms.Add(key, 3)
	estimate1 := cms.Estimate(key)

	cms.Add(key, 2)
	estimate2 := cms.Estimate(key)

	cms.Add(key, 5)
	estimate3 := cms.Estimate(key)

	// Estimates should be non-decreasing
	if estimate2 < estimate1 {
		t.Errorf("Estimate decreased: %d < %d", estimate2, estimate1)
	}

	if estimate3 < estimate2 {
		t.Errorf("Estimate decreased: %d < %d", estimate3, estimate2)
	}

	// Final estimate should be >= 10
	if estimate3 < 10 {
		t.Errorf("Final estimate %d should be >= 10", estimate3)
	}
}

func TestCountMinSketch_Reset(t *testing.T) {
	cms := NewCountMinSketch(0.01, 0.01)

	// Add some data
	cms.Add([]byte("key1"), 100)
	cms.Add([]byte("key2"), 50)

	// Verify data exists
	if cms.Estimate([]byte("key1")) == 0 {
		t.Error("key1 should have non-zero estimate before reset")
	}

	// Reset
	cms.Reset()

	// Verify data is cleared
	if cms.Estimate([]byte("key1")) != 0 {
		t.Error("key1 should have zero estimate after reset")
	}

	if cms.Estimate([]byte("key2")) != 0 {
		t.Error("key2 should have zero estimate after reset")
	}
}

func TestCountMinSketch_Decay(t *testing.T) {
	cms := NewCountMinSketch(0.01, 0.01)

	// Add some data
	cms.Add([]byte("key1"), 100)
	cms.Add([]byte("key2"), 50)

	initialCount1 := cms.Estimate([]byte("key1"))
	initialCount2 := cms.Estimate([]byte("key2"))

	// Apply 50% decay
	cms.Decay(0.5)

	decayedCount1 := cms.Estimate([]byte("key1"))
	decayedCount2 := cms.Estimate([]byte("key2"))

	// Counts should be reduced
	if decayedCount1 >= initialCount1 {
		t.Errorf("Decay not applied to key1: before=%d, after=%d", initialCount1, decayedCount1)
	}

	if decayedCount2 >= initialCount2 {
		t.Errorf("Decay not applied to key2: before=%d, after=%d", initialCount2, decayedCount2)
	}

	// Should be approximately half (allowing for rounding)
	if decayedCount1 == 0 || decayedCount1 > initialCount1*6/10 {
		t.Errorf("Decay result unexpected for key1: %d (from %d)", decayedCount1, initialCount1)
	}
}
