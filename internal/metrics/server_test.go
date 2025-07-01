package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mingrammer/keyflare/internal/detector"
)

func TestMetricServer_Start_Stop(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0", // Use port 0 for automatic assignment
		CollectionInterval:  100 * time.Millisecond,
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

func TestMetricServer_HandleRoot(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	server.handleRoot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "KeyFlare Metrics") {
		t.Error("Response should contain 'KeyFlare Metrics'")
	}

	if !strings.Contains(body, "/metrics") {
		t.Error("Response should contain link to /metrics")
	}

	if !strings.Contains(body, "/hot-keys") {
		t.Error("Response should contain link to /hot-keys")
	}
}

func TestMetricServer_HandleHotKeys_Empty(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	req := httptest.NewRequest("GET", "/hot-keys", nil)
	w := httptest.NewRecorder()

	server.handleHotKeys(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var response hotKeysResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if len(response.Keys) != 0 {
		t.Error("Expected empty keys for empty history")
	}
}

func TestMetricServer_HandleHotKeys_WithData(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Add test data to history
	hotKeys := []detector.KeyCount{
		{Key: "key1", Count: 100},
		{Key: "key2", Count: 75},
		{Key: "key3", Count: 50},
	}
	server.hotKeyHistory.Add(hotKeys)

	req := httptest.NewRequest("GET", "/hot-keys", nil)
	w := httptest.NewRecorder()

	server.handleHotKeys(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response hotKeysResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if len(response.Keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(response.Keys))
	}

	if response.TopK != 3 {
		t.Errorf("Expected TopK 3, got %d", response.TopK)
	}

	if response.TotalKeys != 3 {
		t.Errorf("Expected TotalKeys 3, got %d", response.TotalKeys)
	}

	// Check key data
	if response.Keys[0].Key != "key1" || response.Keys[0].Count != 100 {
		t.Errorf("Expected key1 with count 100, got %s with count %d", response.Keys[0].Key, response.Keys[0].Count)
	}

	if response.Keys[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", response.Keys[0].Rank)
	}

	// Check that all keys have required fields
	for i, keyInfo := range response.Keys {
		if keyInfo.Key == "" {
			t.Errorf("Key %d has empty key", i)
		}
		if keyInfo.Count == 0 {
			t.Errorf("Key %s has zero count", keyInfo.Key)
		}
		if keyInfo.Rank == 0 {
			t.Errorf("Key %s has zero rank", keyInfo.Key)
		}
		if keyInfo.FirstSeen.IsZero() {
			t.Errorf("Key %s has zero FirstSeen", keyInfo.Key)
		}
		if keyInfo.LastSeen.IsZero() {
			t.Errorf("Key %s has zero LastSeen", keyInfo.Key)
		}
	}
}

func TestMetricServer_HandleHotKeys_WithLimit(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Add many test keys
	hotKeys := []detector.KeyCount{}
	for i := 0; i < 20; i++ {
		hotKeys = append(hotKeys, detector.KeyCount{
			Key:   fmt.Sprintf("key%d", i),
			Count: uint64(100 - i),
		})
	}
	server.hotKeyHistory.Add(hotKeys)

	// Test with limit parameter
	req := httptest.NewRequest("GET", "/hot-keys?limit=5", nil)
	w := httptest.NewRecorder()

	server.handleHotKeys(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response hotKeysResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if len(response.Keys) != 5 {
		t.Errorf("Expected 5 keys due to limit, got %d", len(response.Keys))
	}

	if response.QueryLimit != 5 {
		t.Errorf("Expected QueryLimit 5, got %d", response.QueryLimit)
	}

	if response.ActualLimit != 5 {
		t.Errorf("Expected ActualLimit 5, got %d", response.ActualLimit)
	}
}

func TestMetricServer_HandleHotKeys_InvalidLimit(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Add test data
	hotKeys := []detector.KeyCount{{Key: "key1", Count: 100}}
	server.hotKeyHistory.Add(hotKeys)

	// Test with invalid limit
	req := httptest.NewRequest("GET", "/hot-keys?limit=invalid", nil)
	w := httptest.NewRecorder()

	server.handleHotKeys(w, req)

	// Should still return 200 and use default limit
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response hotKeysResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Should use default limit of 100
	if response.QueryLimit != 100 {
		t.Errorf("Expected QueryLimit 100 (default), got %d", response.QueryLimit)
	}
}

func TestMetricServer_HandleHotKeys_TrendDetection(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		HotKeyMetricLimit:   10,
		HotKeyHistorySize:   5,
	}

	server := newMetricServer(config)

	// Add first snapshot
	hotKeys1 := []detector.KeyCount{
		{Key: "stable_key", Count: 50},
		{Key: "rising_key", Count: 30},
		{Key: "falling_key", Count: 80},
	}
	server.hotKeyHistory.Add(hotKeys1)

	// Wait a bit and add second snapshot with changes
	time.Sleep(10 * time.Millisecond)
	hotKeys2 := []detector.KeyCount{
		{Key: "stable_key", Count: 50},  // Same count - should be "stable"
		{Key: "rising_key", Count: 60},  // Increased from 30 to 60 - should be "rising"
		{Key: "falling_key", Count: 40}, // Decreased from 80 to 40 - should be "falling"
		{Key: "new_key", Count: 25},     // New key - should be "new"
	}
	server.hotKeyHistory.Add(hotKeys2)

	req := httptest.NewRequest("GET", "/hot-keys", nil)
	w := httptest.NewRecorder()

	server.handleHotKeys(w, req)

	var response hotKeysResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check trends
	trendMap := make(map[string]string)
	for _, keyInfo := range response.Keys {
		trendMap[keyInfo.Key] = keyInfo.Trend
	}

	if trendMap["stable_key"] != "stable" {
		t.Errorf("Expected stable_key trend to be 'stable', got '%s'", trendMap["stable_key"])
	}

	if trendMap["rising_key"] != "rising" {
		t.Errorf("Expected rising_key trend to be 'rising', got '%s'", trendMap["rising_key"])
	}

	if trendMap["falling_key"] != "falling" {
		t.Errorf("Expected falling_key trend to be 'falling', got '%s'", trendMap["falling_key"])
	}

	if trendMap["new_key"] != "new" {
		t.Errorf("Expected new_key trend to be 'new', got '%s'", trendMap["new_key"])
	}
}

func TestMetricServer_CollectionTicker(t *testing.T) {
	config := Config{
		Namespace:           "test",
		MetricServerAddress: ":0",
		CollectionInterval:  50 * time.Millisecond, // Fast for testing
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
	det.Increment("test_key", 100)
	server.SetDetector(det)

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Wait for at least one collection cycle
	time.Sleep(100 * time.Millisecond)

	// Check that metrics were collected
	snapshot := server.hotKeyHistory.GetLatest()
	if snapshot == nil {
		t.Error("Expected snapshot after collection cycle")
	} else if len(snapshot.keys) == 0 {
		t.Error("Expected some keys in collected snapshot")
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}
