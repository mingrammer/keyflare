package policy

import (
	"fmt"
	"testing"
	"time"
)

func TestLocalCachePolicy_Get_Miss(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          60,
		Jitter:       0.1,
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config)

	// Test cache miss
	ctx := Context{
		Key:  "non-existent-key",
		Data: GetRequest{},
	}

	result := policy.Apply(ctx)

	if result.Error != nil {
		t.Errorf("Expected no error for cache miss, got: %v", result.Error)
	}

	cacheMiss, ok := result.Data.(CacheMiss)
	if !ok {
		t.Errorf("Expected CacheMiss, got: %T", result.Data)
	}

	if cacheMiss.Key != "non-existent-key" {
		t.Errorf("Expected key 'non-existent-key', got: %s", cacheMiss.Key)
	}
}

func TestLocalCachePolicy_Set_Get(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          60,
		Jitter:       0.0, // No jitter for predictable testing
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config)

	// Test SET operation
	setCtx := Context{
		Key: "test-key",
		Data: SetRequest{
			Value: "test-value",
		},
	}

	setResult := policy.Apply(setCtx)

	if setResult.Error != nil {
		t.Errorf("Expected successful set, got error: %v", setResult.Error)
	}

	cacheSet, ok := setResult.Data.(CacheSet)
	if !ok {
		t.Errorf("Expected CacheSet, got: %T", setResult.Data)
	}

	if cacheSet.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got: %s", cacheSet.Key)
	}

	if cacheSet.TTL != 60 {
		t.Errorf("Expected TTL 60, got: %f", cacheSet.TTL)
	}

	// Test GET operation
	getCtx := Context{
		Key:  "test-key",
		Data: GetRequest{},
	}

	getResult := policy.Apply(getCtx)

	if getResult.Error != nil {
		t.Errorf("Expected successful get, got error: %v", getResult.Error)
	}

	cacheHit, ok := getResult.Data.(CacheHit)
	if !ok {
		t.Errorf("Expected CacheHit, got: %T", getResult.Data)
	}

	if cacheHit.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got: %s", cacheHit.Key)
	}

	if cacheHit.Value != "test-value" {
		t.Errorf("Expected value 'test-value', got: %v", cacheHit.Value)
	}

	if cacheHit.ShouldRefresh {
		t.Error("Expected ShouldRefresh to be false for fresh item")
	}
}

func TestLocalCachePolicy_Expiration(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          0.1, // 100ms TTL for quick expiration
		Jitter:       0.0,
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config)

	// Set a value
	setCtx := Context{
		Key: "test-key",
		Data: SetRequest{
			Value: "test-value",
		},
	}
	policy.Apply(setCtx)

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Try to get the expired value
	getCtx := Context{
		Key:  "test-key",
		Data: GetRequest{},
	}

	getResult := policy.Apply(getCtx)

	if getResult.Error != nil {
		t.Errorf("Expected no error for expired item, got: %v", getResult.Error)
	}

	cacheMiss, ok := getResult.Data.(CacheMiss)
	if !ok {
		t.Errorf("Expected CacheMiss, got: %T", getResult.Data)
	}

	if cacheMiss.Key != "test-key" {
		t.Errorf("Expected key 'test-key', got: %s", cacheMiss.Key)
	}
}

func TestLocalCachePolicy_RefreshAhead(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          1.0, // 1 second TTL
		Jitter:       0.0,
		Capacity:     100,
		RefreshAhead: 0.5, // Refresh at 50% of TTL (500ms)
	}
	policy := newLocalCachePolicy(config)

	// Set a value
	setCtx := Context{
		Key: "test-key",
		Data: SetRequest{
			Value: "test-value",
		},
	}
	policy.Apply(setCtx)

	// Wait for refresh threshold
	time.Sleep(600 * time.Millisecond)

	// Get the value
	getCtx := Context{
		Key:  "test-key",
		Data: GetRequest{},
	}

	getResult := policy.Apply(getCtx)
	if getResult.Error != nil {
		t.Errorf("Expected successful get, got error: %v", getResult.Error)
	}

	cacheHit, ok := getResult.Data.(CacheHit)
	if !ok {
		t.Errorf("Expected CacheHit, got: %T", getResult.Data)
	}

	if !cacheHit.ShouldRefresh {
		t.Error("Expected ShouldRefresh to be true after refresh threshold")
	}
}

func TestLocalCachePolicy_Capacity(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          60,
		Jitter:       0.0,
		Capacity:     2, // Small capacity for testing eviction
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config)

	// Fill cache to capacity
	for i := 0; i < 2; i++ {
		setCtx := Context{
			Key: testKey(i),
			Data: SetRequest{
				Value: testValue(i),
			},
		}
		result := policy.Apply(setCtx)
		if result.Error != nil {
			t.Errorf("Expected successful set for %s, got error: %v", testKey(i), result.Error)
		}
	}

	// Add one more item to trigger eviction
	setCtx := Context{
		Key: "key2",
		Data: SetRequest{
			Value: "value2",
		},
	}
	result := policy.Apply(setCtx)
	if result.Error != nil {
		t.Errorf("Expected successful set for key2, got error: %v", result.Error)
	}

	evictedCount := 0

	// Check that one of the first items was evicted (cache miss indicates eviction)
	getCtx1 := Context{
		Key:  "key0",
		Data: GetRequest{},
	}
	result1 := policy.Apply(getCtx1)
	if _, ok := result1.Data.(CacheMiss); ok {
		evictedCount++
	}

	getCtx2 := Context{
		Key:  "key1",
		Data: GetRequest{},
	}
	result2 := policy.Apply(getCtx2)
	if _, ok := result2.Data.(CacheMiss); ok {
		evictedCount++
	}

	if evictedCount == 0 {
		t.Error("Expected at least one item to be evicted when capacity is exceeded")
	}

	// The newest item should still be there
	getCtx3 := Context{
		Key:  "key2",
		Data: GetRequest{},
	}
	result3 := policy.Apply(getCtx3)
	if _, ok := result3.Data.(CacheHit); !ok {
		t.Error("Expected newest item to still be in cache")
	}
}

func TestLocalCachePolicy_InvalidOperation(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          60,
		Jitter:       0.0,
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config)

	// Test unsupported operation
	ctx := Context{
		Key:  "test-key",
		Data: "invalid-operation",
	}

	result := policy.Apply(ctx)
	if result.Error == nil {
		t.Error("Expected failure for invalid operation")
	}
}

func TestLocalCachePolicy_Jitter(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          60,
		Jitter:       0.2, // 20% jitter
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config).(*localCachePolicy)

	// Test TTL calculation with jitter multiple times
	ttls := make([]float64, 10)
	for i := 0; i < 10; i++ {
		ttls[i] = policy.calculateTTLWithJitter()
	}

	// Check that TTLs are within expected range
	minTTL := 60 * (1 - 0.2) // 48
	maxTTL := 60 * (1 + 0.2) // 72

	for i, ttl := range ttls {
		if ttl < minTTL || ttl > maxTTL {
			t.Errorf("TTL %d (%f) is outside expected range [%f, %f]", i, ttl, minTTL, maxTTL)
		}
	}

	// Check that not all TTLs are the same (testing randomness)
	allSame := true
	for i := 1; i < len(ttls); i++ {
		if ttls[i] != ttls[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("All TTLs are the same, jitter might not be working correctly")
	}
}

func TestLocalCachePolicy_GetCacheStats(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          0.1, // Short TTL for testing expired items
		Jitter:       0.0,
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config).(*localCachePolicy)

	// Add some items
	for i := 0; i < 3; i++ {
		setCtx := Context{
			Key: testKey(i),
			Data: SetRequest{
				Value: testValue(i),
			},
		}
		policy.Apply(setCtx)
	}

	// Wait for some items to expire
	time.Sleep(150 * time.Millisecond)

	// Add one fresh item
	setCtx := Context{
		Key: "fresh-key",
		Data: SetRequest{
			Value: "fresh-value",
		},
	}
	policy.Apply(setCtx)

	// Get cache stats
	stats := policy.GetCacheStats()

	if stats.Size != 4 {
		t.Errorf("Expected size 4, got: %d", stats.Size)
	}

	if stats.Capacity != 100 {
		t.Errorf("Expected capacity 100, got: %d", stats.Capacity)
	}

	if stats.ExpiredItems < 3 {
		t.Errorf("Expected at least 3 expired items, got: %d", stats.ExpiredItems)
	}
}

func TestLocalCachePolicy_SetOverwrite(t *testing.T) {
	config := LocalCacheConfig{
		TTL:          60,
		Jitter:       0.0,
		Capacity:     100,
		RefreshAhead: 0.8,
	}
	policy := newLocalCachePolicy(config)

	// Set initial value
	setCtx1 := Context{
		Key: "test-key",
		Data: SetRequest{
			Value: "initial-value",
		},
	}
	result1 := policy.Apply(setCtx1)
	if result1.Error != nil {
		t.Errorf("Expected successful set, got error: %v", result1.Error)
	}

	// Overwrite with new value
	setCtx2 := Context{
		Key: "test-key",
		Data: SetRequest{
			Value: "updated-value",
		},
	}
	result2 := policy.Apply(setCtx2)
	if result2.Error != nil {
		t.Errorf("Expected successful overwrite, got error: %v", result2.Error)
	}

	// Get the updated value
	getCtx := Context{
		Key:  "test-key",
		Data: GetRequest{},
	}
	getResult := policy.Apply(getCtx)
	if getResult.Error != nil {
		t.Errorf("Expected successful get, got error: %v", getResult.Error)
	}

	cacheHit, ok := getResult.Data.(CacheHit)
	if !ok {
		t.Errorf("Expected CacheHit, got: %T", getResult.Data)
	}

	if cacheHit.Value != "updated-value" {
		t.Errorf("Expected updated value 'updated-value', got: %v", cacheHit.Value)
	}
}

// Helper functions for testing
func testKey(i int) string {
	return fmt.Sprintf("key%d", i)
}

func testValue(i int) string {
	return fmt.Sprintf("value%d", i)
}
