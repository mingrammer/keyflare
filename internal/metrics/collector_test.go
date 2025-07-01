package metrics

import (
	"testing"
	"time"

	"github.com/mingrammer/keyflare/internal/detector"
)

func TestNew(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":9121",
		CollectionInterval:  15 * time.Second,
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	collector := New(config)

	if collector == nil {
		t.Fatal("New() returned nil")
	}

	// Verify it's actually a metricServer
	if _, ok := collector.(*metricServer); !ok {
		t.Error("Expected *metricServer, got different type")
	}
}

func TestNewNoop(t *testing.T) {
	collector := NewNoop()

	if collector == nil {
		t.Fatal("NewNoop() returned nil")
	}

	// Verify it's actually a noopCollector
	if _, ok := collector.(*noopCollector); !ok {
		t.Error("Expected *noopCollector, got different type")
	}

	// Test that all methods can be called without panic
	collector.RecordKeyAccess("test")
	collector.RecordPolicyApplication("local_cache", true)
	collector.UpdateHotKeys([]detector.KeyCount{})
	collector.SetDetector(nil)

	// Start and stop should not return errors
	if err := collector.Start(); err != nil {
		t.Errorf("Start() returned error: %v", err)
	}

	if err := collector.Stop(); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

func TestMetricServer_RecordKeyAccess(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0", // Use port 0 for testing
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// This should not panic
	server.RecordKeyAccess("test_key")
	server.RecordKeyAccess("another_key")
}

func TestMetricServer_RecordPolicyApplication(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Test different policy types and success states
	server.RecordPolicyApplication("local_cache", true)
	server.RecordPolicyApplication("local_cache", false)
	server.RecordPolicyApplication("key_splitting", true)
}

func TestMetricServer_UpdateHotKeys(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   3, // Limit to 3 for testing
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Create test hot keys
	hotKeys := []detector.KeyCount{
		{Key: "key1", Count: 100},
		{Key: "key2", Count: 75},
		{Key: "key3", Count: 50},
		{Key: "key4", Count: 25}, // Should be limited out
		{Key: "key5", Count: 10}, // Should be limited out
	}

	server.UpdateHotKeys(hotKeys)

	// Check that history was updated
	snapshot := server.hotKeyHistory.GetLatest()
	if snapshot == nil {
		t.Fatal("Expected snapshot after UpdateHotKeys")
	}

	if len(snapshot.keys) != 5 {
		t.Errorf("Expected 5 keys in snapshot, got %d", len(snapshot.keys))
	}

	// Check that all keys have metadata
	for _, kc := range snapshot.keys {
		if _, ok := snapshot.keyMeta[kc.Key]; !ok {
			t.Errorf("Missing metadata for key %s", kc.Key)
		}
	}
}

func TestMetricServer_SetDetector(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Create a mock detector
	detectorConfig := detector.Config{
		TopK:          10,
		HotThreshold:  50,
		DecayFactor:   0.95,
		DecayInterval: 60 * time.Second,
	}
	det := detector.New(detectorConfig)

	server.SetDetector(det)

	if server.detector != det {
		t.Error("Detector not properly set")
	}
}

func TestHotKeyHistory_Add(t *testing.T) {
	history := newHotKeyHistory(3) // Small size for testing

	// Add first snapshot
	keys1 := []detector.KeyCount{
		{Key: "key1", Count: 10},
		{Key: "key2", Count: 5},
	}
	history.Add(keys1)

	snapshot := history.GetLatest()
	if snapshot == nil {
		t.Fatal("Expected snapshot after Add")
	}

	if len(snapshot.keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(snapshot.keys))
	}

	// Add more snapshots to test circular buffer
	keys2 := []detector.KeyCount{{Key: "key3", Count: 15}}
	keys3 := []detector.KeyCount{{Key: "key4", Count: 20}}
	keys4 := []detector.KeyCount{{Key: "key5", Count: 25}} // Should wrap around

	history.Add(keys2)
	history.Add(keys3)
	history.Add(keys4)

	// Should still have a latest snapshot
	latestSnapshot := history.GetLatest()
	if latestSnapshot == nil {
		t.Fatal("Expected latest snapshot")
	}

	if len(latestSnapshot.keys) != 1 || latestSnapshot.keys[0].Key != "key5" {
		t.Error("Latest snapshot not correct after circular buffer wrap")
	}
}

func TestHotKeyHistory_GetLatest_Empty(t *testing.T) {
	history := newHotKeyHistory(5)

	snapshot := history.GetLatest()
	if snapshot != nil {
		t.Error("Expected nil snapshot for empty history")
	}
}

func TestHotKeyHistory_TrendCalculation(t *testing.T) {
	history := newHotKeyHistory(5)

	// Test trend calculation workflow step by step

	// Step 1: Add initial snapshot
	keys1 := []detector.KeyCount{
		{Key: "stable", Count: 100},
		{Key: "rising", Count: 50},
		{Key: "falling", Count: 200},
	}
	history.Add(keys1)

	snapshot1 := history.GetLatest()
	if snapshot1 == nil {
		t.Fatal("Expected first snapshot")
	}

	// First snapshot - all keys should have PrevCount = 0 (new keys)
	for _, kc := range keys1 {
		meta := snapshot1.keyMeta[kc.Key]
		if meta.prevCount != 0 {
			t.Errorf("Key %s: expected PrevCount 0 for first snapshot, got %d", kc.Key, meta.prevCount)
		}
	}

	// Step 2: Add second snapshot with changes
	time.Sleep(5 * time.Millisecond)
	keys2 := []detector.KeyCount{
		{Key: "stable", Count: 100},  // Same count
		{Key: "rising", Count: 80},   // Increased from 50 to 80
		{Key: "falling", Count: 120}, // Decreased from 200 to 120
		{Key: "new", Count: 30},      // New key
	}
	history.Add(keys2)

	snapshot2 := history.GetLatest()
	if snapshot2 == nil {
		t.Fatal("Expected second snapshot")
	}

	// Verify trend can be calculated correctly
	// PrevCount should contain actual values from previous snapshot
	testCases := []struct {
		key           string
		currentCount  uint64
		expectedPrev  uint64
		expectedTrend string
	}{
		{"stable", 100, 100, "stable"},   // 100 == 100
		{"rising", 80, 50, "rising"},     // 80 > 50
		{"falling", 120, 200, "falling"}, // 120 < 200
		{"new", 30, 0, "new"},            // new key, PrevCount = 0
	}

	for _, tc := range testCases {
		meta, ok := snapshot2.keyMeta[tc.key]
		if !ok {
			t.Errorf("Key %s not found in snapshot", tc.key)
			continue
		}

		if meta.prevCount != tc.expectedPrev {
			t.Errorf("Key %s: expected PrevCount %d, got %d", tc.key, tc.expectedPrev, meta.prevCount)
		}

		// Find current count in snapshot
		var currentCount uint64
		for _, kc := range snapshot2.keys {
			if kc.Key == tc.key {
				currentCount = kc.Count
				break
			}
		}

		if currentCount != tc.currentCount {
			t.Errorf("Key %s: expected current count %d, got %d", tc.key, tc.currentCount, currentCount)
		}

		// Verify trend logic
		var actualTrend string
		if meta.prevCount == 0 {
			actualTrend = "new"
		} else if currentCount > meta.prevCount {
			actualTrend = "rising"
		} else if currentCount < meta.prevCount {
			actualTrend = "falling"
		} else {
			actualTrend = "stable"
		}

		if actualTrend != tc.expectedTrend {
			t.Errorf("Key %s: expected trend %s, got %s (current: %d, prev: %d)",
				tc.key, tc.expectedTrend, actualTrend, currentCount, meta.prevCount)
		}
	}

	// Step 3: Add third snapshot to verify PrevCount uses actual counts from snapshot2
	time.Sleep(5 * time.Millisecond)
	keys3 := []detector.KeyCount{
		{Key: "stable", Count: 90}, // Now decreased from 100 to 90
		{Key: "rising", Count: 85}, // Continued rising from 80 to 85
	}
	history.Add(keys3)

	snapshot3 := history.GetLatest()
	meta3stable := snapshot3.keyMeta["stable"]
	meta3rising := snapshot3.keyMeta["rising"]

	// PrevCount should now be actual counts from snapshot2
	if meta3stable.prevCount != 100 {
		t.Errorf("Expected stable PrevCount 100 (from snapshot2), got %d", meta3stable.prevCount)
	}

	if meta3rising.prevCount != 80 {
		t.Errorf("Expected rising PrevCount 80 (from snapshot2), got %d", meta3rising.prevCount)
	}
}

func TestHotKeyHistory_Metadata(t *testing.T) {
	history := newHotKeyHistory(5)

	// Add initial snapshot
	keys1 := []detector.KeyCount{
		{Key: "key1", Count: 10},
	}
	history.Add(keys1)

	snapshot1 := history.GetLatest()
	if snapshot1 == nil {
		t.Fatal("Expected first snapshot")
	}

	// Check metadata was created
	meta, ok := snapshot1.keyMeta["key1"]
	if !ok {
		t.Fatal("Expected metadata for key1")
	}

	firstSeen := meta.firstSeen
	if meta.prevCount != 0 {
		t.Errorf("Expected PrevCount 0 after first snapshot, got %d", meta.prevCount)
	}

	// Wait a bit and add another snapshot with same key but different count
	time.Sleep(10 * time.Millisecond)
	keys2 := []detector.KeyCount{
		{Key: "key1", Count: 20}, // Increased count
	}
	history.Add(keys2)

	snapshot2 := history.GetLatest()
	if snapshot2 == nil {
		t.Fatal("Expected second snapshot")
	}

	meta2, ok := snapshot2.keyMeta["key1"]
	if !ok {
		t.Fatal("Expected metadata for key1 in second snapshot")
	}

	// FirstSeen should be preserved, LastSeen should be updated
	if !meta2.firstSeen.Equal(firstSeen) {
		t.Error("FirstSeen should be preserved across snapshots")
	}

	if !meta2.lastSeen.After(firstSeen) {
		t.Error("LastSeen should be updated")
	}

	// PrevCount should now reflect the previous snapshot's count (10)
	// This allows trend calculation to compare 20 (current) vs 10 (previous)
	if meta2.prevCount != 10 {
		t.Errorf("Expected PrevCount 10 for trend calculation, got %d", meta2.prevCount)
	}

	// Wait and add third snapshot to verify PrevCount gets updated
	time.Sleep(10 * time.Millisecond)
	keys3 := []detector.KeyCount{
		{Key: "key1", Count: 15}, // Decreased count
	}
	history.Add(keys3)

	snapshot3 := history.GetLatest()
	meta3 := snapshot3.keyMeta["key1"]

	// Now PrevCount should be 20 (from previous snapshot)
	if meta3.prevCount != 20 {
		t.Errorf("Expected PrevCount 20 for trend calculation, got %d", meta3.prevCount)
	}
}

func TestConfig_Defaults(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		check  func(t *testing.T, server *metricServer)
	}{
		{
			name:   "default hot key metric limit",
			config: Config{MetricServerAddress: ":0"},
			check: func(t *testing.T, server *metricServer) {
				if server.config.HotKeyMetricLimit != 10 {
					t.Errorf("Expected default HotKeyMetricLimit 10, got %d", server.config.HotKeyMetricLimit)
				}
			},
		},
		{
			name:   "default hot key history size",
			config: Config{MetricServerAddress: ":0"},
			check: func(t *testing.T, server *metricServer) {
				if server.config.HotKeyHistorySize != 10 {
					t.Errorf("Expected default HotKeyHistorySize 10, got %d", server.config.HotKeyHistorySize)
				}
			},
		},
		{
			name:   "default collection interval",
			config: Config{MetricServerAddress: ":0"},
			check: func(t *testing.T, server *metricServer) {
				if server.config.CollectionInterval != 15*time.Second {
					t.Errorf("Expected default CollectionInterval 15s, got %v", server.config.CollectionInterval)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := New(tt.config)
			server, ok := collector.(*metricServer)
			if !ok {
				t.Fatal("Expected *metricServer")
			}
			tt.check(t, server)
		})
	}
}

func TestMetricServer_CollectMetrics(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Create and set a detector with some data
	detectorConfig := detector.Config{
		TopK:          10,
		HotThreshold:  50,
		DecayFactor:   0.95,
		DecayInterval: 60 * time.Second,
	}
	det := detector.New(detectorConfig)

	// Add some test data
	det.Increment("hot_key", 100)
	det.Increment("warm_key", 25)

	server.SetDetector(det)

	// Call collectMetrics
	server.collectMetrics()

	// Check that history was updated
	snapshot := server.hotKeyHistory.GetLatest()
	if snapshot == nil {
		t.Fatal("Expected snapshot after collectMetrics")
	}

	if len(snapshot.keys) == 0 {
		t.Error("Expected some keys in snapshot")
	}
}
