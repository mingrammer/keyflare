package policy

import (
	"fmt"
	"testing"
)

func TestManager_InvalidParameters(t *testing.T) {
	// Test invalid parameters for LocalCache
	config := Config{
		Type:       LocalCache,
		Parameters: "invalid-parameters", // Wrong type
	}

	_, err := New(config)
	if err == nil {
		t.Error("Expected error for invalid LocalCache parameters, got nil")
	}

	// Test invalid parameters for KeySplitting
	config = Config{
		Type:       KeySplitting,
		Parameters: 123, // Wrong type
	}

	_, err = New(config)
	if err == nil {
		t.Error("Expected error for invalid KeySplitting parameters, got nil")
	}

	// Test unsupported policy type
	config = Config{
		Type: "unsupported",
	}

	_, err = New(config)
	if err == nil {
		t.Error("Expected error for unsupported policy type, got nil")
	}
}

func TestManager_LocalCachePolicy(t *testing.T) {
	config := Config{
		Type: LocalCache,
		Parameters: LocalCacheConfig{
			TTL:          60,
			Jitter:       0.1,
			Capacity:     100,
			RefreshAhead: 0.8,
		},
		WhitelistKeys:     []string{"test-key"},
		WhitelistPatterns: []string{"user:.*"},
	}

	manager, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test whitelisted key
	policy := manager.GetPolicy("test-key")
	if policy == nil {
		t.Error("Expected policy for whitelisted key, got nil")
	}

	// Test pattern match
	policy = manager.GetPolicy("user:123")
	if policy == nil {
		t.Error("Expected policy for pattern-matched key, got nil")
	}

	// Test non-matching key
	policy = manager.GetPolicy("other-key")
	if policy != nil {
		t.Error("Expected nil policy for non-matching key")
	}

	// Test full flow with local cache policy
	testPolicy := manager.GetPolicy("test-key")
	if testPolicy == nil {
		t.Fatal("Expected policy for test-key")
	}

	// Test cache miss
	getCtx := Context{
		Key:  "test-key",
		Data: GetRequest{},
	}
	getResult := testPolicy.Apply(getCtx)
	if getResult.Error != nil {
		t.Errorf("Expected no error for cache miss, got: %v", getResult.Error)
	}
	if _, ok := getResult.Data.(CacheMiss); !ok {
		t.Errorf("Expected CacheMiss, got: %T", getResult.Data)
	}

	// Test set operation
	setCtx := Context{
		Key: "test-key",
		Data: SetRequest{
			Value: "test-value",
		},
	}
	setResult := testPolicy.Apply(setCtx)
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

	// Test cache hit
	getResult = testPolicy.Apply(getCtx)
	if getResult.Error != nil {
		t.Errorf("Expected successful get, got error: %v", getResult.Error)
	}
	cacheHit, ok := getResult.Data.(CacheHit)
	if !ok {
		t.Errorf("Expected CacheHit, got: %T", getResult.Data)
	}
	if cacheHit.Value != "test-value" {
		t.Errorf("Expected 'test-value', got: %v", cacheHit.Value)
	}
}

func TestManager_KeySplittingPolicy(t *testing.T) {
	config := Config{
		Type: KeySplitting,
		Parameters: KeySplittingConfig{
			Shards: 5,
		},
		WhitelistKeys: []string{"split-key"},
	}

	manager, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test whitelisted key
	policy := manager.GetPolicy("split-key")
	if policy == nil {
		t.Error("Expected policy for whitelisted key, got nil")
	}

	// Test non-matching key
	policy = manager.GetPolicy("other-key")
	if policy != nil {
		t.Error("Expected nil policy for non-matching key")
	}

	// Test full flow with key splitting policy
	testPolicy := manager.GetPolicy("split-key")
	if testPolicy == nil {
		t.Fatal("Expected policy for split-key")
	}

	// Test GET operation
	getCtx := Context{
		Key:  "split-key",
		Data: GetRequest{},
	}
	getResult := testPolicy.Apply(getCtx)
	if getResult.Error != nil {
		t.Errorf("Expected successful get operation, got error: %v", getResult.Error)
	}
	getAction, ok := getResult.Data.(KeySplittingGetAction)
	if !ok {
		t.Errorf("Expected KeySplittingGetAction, got: %T", getResult.Data)
	}
	if getAction.OriginalKey != "split-key" {
		t.Errorf("Expected original key 'split-key', got: %s", getAction.OriginalKey)
	}
	if len(getAction.ShardKeys) != 5 {
		t.Errorf("Expected 5 shard keys, got: %d", len(getAction.ShardKeys))
	}
	for i, shardKey := range getAction.ShardKeys {
		expected := fmt.Sprintf("split-key:shard:%d", i)
		if shardKey != expected {
			t.Errorf("Expected shard key %s, got: %s", expected, shardKey)
		}
	}

	// Test SET operation
	setCtx := Context{
		Key: "split-key",
		Data: SetRequest{
			Value: "split-value",
		},
	}
	setResult := testPolicy.Apply(setCtx)
	if setResult.Error != nil {
		t.Errorf("Expected successful set operation, got error: %v", setResult.Error)
	}
	setAction, ok := setResult.Data.(KeySplittingSetAction)
	if !ok {
		t.Errorf("Expected KeySplittingSetAction, got: %T", setResult.Data)
	}
	if setAction.OriginalKey != "split-key" {
		t.Errorf("Expected original key 'split-key', got: %s", setAction.OriginalKey)
	}
	if setAction.Value != "split-value" {
		t.Errorf("Expected value 'split-value', got: %v", setAction.Value)
	}
	if len(setAction.ShardKeys) != 5 {
		t.Errorf("Expected 5 shard keys, got: %d", len(setAction.ShardKeys))
	}
	for i, shardKey := range setAction.ShardKeys {
		expected := fmt.Sprintf("split-key:shard:%d", i)
		if shardKey != expected {
			t.Errorf("Expected shard key %s, got: %s", expected, shardKey)
		}
	}
}

func TestManager_AddRemoveWhitelistKey(t *testing.T) {
	config := Config{
		Type: LocalCache,
		Parameters: LocalCacheConfig{
			TTL:      60,
			Capacity: 100,
		},
		WhitelistKeys: []string{"initial-key"},
	}

	manager, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test initial whitelist key
	policy := manager.GetPolicy("initial-key")
	if policy == nil {
		t.Error("Expected policy for initial whitelist key")
	}

	// Add new whitelist key
	manager.AddWhitelistKey("new-key")
	policy = manager.GetPolicy("new-key")
	if policy == nil {
		t.Error("Expected policy for newly added whitelist key")
	}

	// Remove whitelist key
	manager.RemoveWhitelistKey("new-key")
	policy = manager.GetPolicy("new-key")
	if policy != nil {
		t.Error("Expected nil policy for removed whitelist key")
	}

	// Initial key should still work
	policy = manager.GetPolicy("initial-key")
	if policy == nil {
		t.Error("Expected policy for initial whitelist key to still work")
	}
}

func TestManager_RegisterPattern(t *testing.T) {
	config := Config{
		Type: LocalCache,
		Parameters: LocalCacheConfig{
			TTL:      60,
			Capacity: 100,
		},
		WhitelistKeys: []string{},
	}

	manager, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Register a new pattern
	err = manager.RegisterPattern("session:.*")
	if err != nil {
		t.Errorf("Expected no error registering pattern, got: %v", err)
	}

	// Test pattern match
	policy := manager.GetPolicy("session:abc123")
	if policy == nil {
		t.Error("Expected policy for pattern-matched key")
	}

	// Test non-matching key
	policy = manager.GetPolicy("user:123")
	if policy != nil {
		t.Error("Expected nil policy for non-matching key")
	}

	// Register invalid pattern
	err = manager.RegisterPattern("[invalid")
	if err == nil {
		t.Error("Expected error for invalid regex pattern")
	}
}

func TestManager_InitialPatterns(t *testing.T) {
	config := Config{
		Type: LocalCache,
		Parameters: LocalCacheConfig{
			TTL:      60,
			Capacity: 100,
		},
		WhitelistKeys:     []string{},
		WhitelistPatterns: []string{"user:.*", "session:.*"},
	}

	manager, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test valid patterns
	policy := manager.GetPolicy("user:123")
	if policy == nil {
		t.Error("Expected policy for user pattern match")
	}

	policy = manager.GetPolicy("session:abc")
	if policy == nil {
		t.Error("Expected policy for session pattern match")
	}

	// Test non-matching key
	policy = manager.GetPolicy("cache:123")
	if policy != nil {
		t.Error("Expected nil policy for non-matching key")
	}

	// Invalid pattern should be skipped, manager should still work
	policy = manager.GetPolicy("user:456")
	if policy == nil {
		t.Error("Expected manager to work despite invalid pattern")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	config := Config{
		Type: LocalCache,
		Parameters: LocalCacheConfig{
			TTL:      60,
			Capacity: 100,
		},
		WhitelistKeys: []string{"test-key"},
	}

	manager, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test concurrent read access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			policy := manager.GetPolicy("test-key")
			if policy == nil {
				t.Error("Expected policy in concurrent access")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent write access
	for i := 0; i < 5; i++ {
		go func(id int) {
			key := fmt.Sprintf("concurrent-key-%d", id)
			manager.AddWhitelistKey(key)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify writes worked
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("concurrent-key-%d", i)
		policy := manager.GetPolicy(key)
		if policy == nil {
			t.Errorf("Expected policy for concurrent key %s", key)
		}
	}
}
