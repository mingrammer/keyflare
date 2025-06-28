package policy

import (
	"fmt"
	"testing"
)

func TestKeySplittingPolicy_Get(t *testing.T) {
	config := KeySplittingConfig{
		Shards: 3,
	}
	policy := newKeySplittingPolicy(config)

	ctx := Context{
		Key:  "test-key",
		Data: GetRequest{},
	}

	result := policy.Apply(ctx)

	if result.Error != nil {
		t.Errorf("Expected successful apply, got error: %v", result.Error)
	}

	action, ok := result.Data.(KeySplittingGetAction)
	if !ok {
		t.Errorf("Expected KeySplittingGetAction, got: %T", result.Data)
	}

	if action.OriginalKey != "test-key" {
		t.Errorf("Expected original key 'test-key', got %s", action.OriginalKey)
	}

	if len(action.ShardKeys) != 3 {
		t.Errorf("Expected 3 shard keys, got %d", len(action.ShardKeys))
	}

	expectedKeys := []string{"test-key:shard:0", "test-key:shard:1", "test-key:shard:2"}
	for i, key := range action.ShardKeys {
		if key != expectedKeys[i] {
			t.Errorf("Expected shard key %s, got %s", expectedKeys[i], key)
		}
	}
}

func TestKeySplittingPolicy_Set(t *testing.T) {
	config := KeySplittingConfig{
		Shards: 5,
	}
	policy := newKeySplittingPolicy(config)

	ctx := Context{
		Key: "user:123",
		Data: SetRequest{
			Value: "user-data",
			TTL:   nil,
		},
	}

	result := policy.Apply(ctx)

	if result.Error != nil {
		t.Errorf("Expected successful apply, got error: %v", result.Error)
	}

	action, ok := result.Data.(KeySplittingSetAction)
	if !ok {
		t.Errorf("Expected KeySplittingSetAction, got: %T", result.Data)
	}

	if action.OriginalKey != "user:123" {
		t.Errorf("Expected original key 'user:123', got %s", action.OriginalKey)
	}

	if action.Value != "user-data" {
		t.Errorf("Expected value 'user-data', got %v", action.Value)
	}

	if len(action.ShardKeys) != 5 {
		t.Errorf("Expected 5 target shards, got %d", len(action.ShardKeys))
	}

	for i, key := range action.ShardKeys {
		expected := fmt.Sprintf("user:123:shard:%d", i)
		if key != expected {
			t.Errorf("Expected shard key %s, got %s", expected, key)
		}
	}
}

func TestKeySplittingPolicy_InvalidOperation(t *testing.T) {
	config := KeySplittingConfig{
		Shards: 3,
	}
	policy := newKeySplittingPolicy(config)

	ctx := Context{
		Key:  "test-key",
		Data: "unsupported-operation",
	}

	result := policy.Apply(ctx)

	if result.Error == nil {
		t.Error("Expected error for unsupported operation")
	}
}

func TestKeySplittingPolicy_GenerateShardKeys(t *testing.T) {
	config := KeySplittingConfig{
		Shards: 7,
	}
	policy := newKeySplittingPolicy(config).(*keySplittingPolicy)

	shardKeys := policy.generateShardKeys("session:abc123")

	if len(shardKeys) != 7 {
		t.Errorf("Expected 7 shard keys, got %d", len(shardKeys))
	}

	for i, key := range shardKeys {
		expected := fmt.Sprintf("session:abc123:shard:%d", i)
		if key != expected {
			t.Errorf("Expected shard key %s, got %s", expected, key)
		}
	}
}
