package policy

import (
	"fmt"
	"math/rand/v2"
)

// keySplittingPolicy implements a policy that splits a key into multiple keys
type keySplittingPolicy struct {
	config KeySplittingConfig
}

// newKeySplittingPolicy creates a new key splitting policy with the provided parameters
func newKeySplittingPolicy(config KeySplittingConfig) Policy {
	return &keySplittingPolicy{
		config: config,
	}
}

// Apply implements Policy.Apply for look-aside key splitting
// This method returns instructions for the client on how to handle the key
func (p *keySplittingPolicy) Apply(ctx Context) Result {
	key := ctx.Key

	switch ctx.Data.(type) {
	case GetRequest:
		return p.handleLookAsideGet(key)
	case SetRequest:
		return p.handleLookAsideSet(key, ctx.Data.(SetRequest))
	default:
		return Result{
			Error: fmt.Errorf("unsupported operation type: %T", ctx.Data),
		}
	}
}

// handleLookAsideGet handles GET operations with look-aside pattern
func (p *keySplittingPolicy) handleLookAsideGet(key string) Result {
	// Look-aside pattern: Try to read from a single shard first,
	// fallback to original key if no sharded data exists
	shardKeys := p.generateShardKeys(key)
	return Result{
		Data: KeySplittingGetAction{
			OriginalKey:  key,
			RandShardKey: shardKeys[rand.Int()%int(p.config.Shards)],
			ShardKeys:    shardKeys,
		},
	}
}

// handleLookAsideSet handles SET operations
func (p *keySplittingPolicy) handleLookAsideSet(key string, req SetRequest) Result {
	shardKeys := p.generateShardKeys(key)
	return Result{
		Data: KeySplittingSetAction{
			OriginalKey: key,
			ShardKeys:   shardKeys,
			Value:       req.Value,
			TTL:         req.TTL,
		},
	}
}

// generateShardKeys generates shard keys for the given key
func (p *keySplittingPolicy) generateShardKeys(key string) []string {
	// TODO: support auto detection for number of shards.
	shards := int(p.config.Shards)
	shardKeys := make([]string, shards)
	for i := range shards {
		shardKeys[i] = fmt.Sprintf("%s:shard:%d", key, i)
	}
	return shardKeys
}

// Action types for key splitting operations
type KeySplittingGetAction struct {
	OriginalKey  string   `json:"original_key"`
	RandShardKey string   `json:"rand_shard_key"`
	ShardKeys    []string `json:"shard_keys"`
}

type KeySplittingSetAction struct {
	OriginalKey string   `json:"original_key"`
	ShardKeys   []string `json:"shard_keys"`
	Value       any      `json:"value"`
	TTL         *float64 `json:"ttl,omitempty"`
	Action      string   `json:"action"`
}
