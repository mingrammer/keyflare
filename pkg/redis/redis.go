// Package redis provides a Redis client wrapper with KeyFlare hot key detection.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/mingrammer/keyflare/internal"
	"github.com/mingrammer/keyflare/internal/policy"
	"github.com/redis/go-redis/v9"
)

// Wrapper wraps a go-redis client with KeyFlare hot key detection.
type Wrapper struct {
	client *redis.ClusterClient
	kf     *internal.KeyFlare
}

// Wrap creates a new Redis client wrapper with the provided client.
// It uses the global KeyFlare instance which must be initialized and started first.
func Wrap(client *redis.ClusterClient) (*Wrapper, error) {
	kf, err := internal.GetInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to get KeyFlare instance: %w. Call keyflare.New() and keyflare.Start() first", err)
	}

	return &Wrapper{
		client: client,
		kf:     kf,
	}, nil
}

// Client returns the underlying Redis client.
func (w *Wrapper) Client() *redis.ClusterClient {
	return w.client
}

// incrementKey increments the key counter in the detector.
func (w *Wrapper) incrementKey(key string) {
	w.kf.Detector().Increment(key, 1)
}

// applyPolicyIfHot applies the policy if the key is hot.
func (w *Wrapper) applyPolicyIfHot(key string, operation string, value any) (any, error) {
	if w.kf.Detector().IsHot(key) {
		p := w.kf.PolicyManager().GetPolicy(key)
		if p != nil {
			var requestData any
			switch operation {
			case "get":
				requestData = policy.GetRequest{}
			case "set":
				requestData = policy.SetRequest{Value: value}
			default:
				return nil, nil
			}

			ctx := policy.Context{
				Key:  key,
				Data: requestData,
			}
			result := p.Apply(ctx)
			if result.Error != nil {
				return nil, fmt.Errorf("failed to apply policy for key %s: %w", key, result.Error)
			}
			return result.Data, nil
		}
	}

	return nil, nil
}

// Get wraps redis.Client.Get.
func (w *Wrapper) Get(ctx context.Context, key string) *redis.StringCmd {
	// Increment key counter
	w.incrementKey(key)

	// Try to apply policy if hot
	policyResult, err := w.applyPolicyIfHot(key, "get", nil)
	if policyResult == nil && err == nil {
		return w.client.Get(ctx, key)
	}

	if err != nil {
		cmd := redis.NewStringCmd(ctx, "get", key)
		cmd.SetErr(err)
		return cmd
	}

	// Handle different policy types
	switch result := policyResult.(type) {
	case policy.CacheHit:
		// Local cache hit
		cmd := redis.NewStringCmd(ctx, "get", key)
		cmd.SetVal(result.Value.(string))
		return cmd
	case policy.KeySplittingGetAction:
		// Look-aside key splitting: try shard first, fallback to original
		return w.handleLookAsideGet(ctx, result)
	case policy.CacheMiss:
		// Cache miss, get from Redis and async set to cache
		redisResult := w.client.Get(ctx, key)
		fmt.Printf("Cache miss for key %s, fetching from Redis. %v\n", key, redisResult)
		if redisResult.Err() == nil {
			// Data found in Redis, asynchronously cache it
			go w.asyncSetLocalCache(key, redisResult.Val())
		}
		return redisResult
	}
	return redis.NewStringCmd(ctx, "get", key)
}

// Set wraps redis.Client.Set.
func (w *Wrapper) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	// Increment key counter
	w.incrementKey(key)

	// Try to apply policy if hot
	if policyResult, err := w.applyPolicyIfHot(key, "set", value); err != nil || policyResult != nil {
		if err != nil {
			cmd := redis.NewStatusCmd(ctx, "set", key, value)
			cmd.SetErr(err)
			return cmd
		}

		// Handle different policy types
		switch result := policyResult.(type) {
		case policy.KeySplittingSetAction:
			// Multi-write to shards
			return w.handleKeySplittingSet(ctx, result, expiration)

		case policy.CacheSet:
			// Local cache set, continue to Redis
			break
		}
	}

	return w.client.Set(ctx, key, value, expiration)
}

// GetSet wraps redis.Client.GetSet.
func (w *Wrapper) GetSet(ctx context.Context, key string, value any) *redis.StringCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.GetSet(ctx, key, value)
}

// Del wraps redis.Client.Del.
func (w *Wrapper) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	// Increment key counters
	for _, key := range keys {
		w.incrementKey(key)
	}

	return w.client.Del(ctx, keys...)
}

// MGet wraps redis.Client.MGet.
func (w *Wrapper) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	// Increment key counters
	for _, key := range keys {
		w.incrementKey(key)
	}

	return w.client.MGet(ctx, keys...)
}

// MSet wraps redis.Client.MSet.
func (w *Wrapper) MSet(ctx context.Context, values ...any) *redis.StatusCmd {
	// Increment key counters
	for i := 0; i < len(values); i += 2 {
		if key, ok := values[i].(string); ok {
			w.incrementKey(key)
		}
	}

	return w.client.MSet(ctx, values...)
}

// Incr wraps redis.Client.Incr.
func (w *Wrapper) Incr(ctx context.Context, key string) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Incr(ctx, key)
}

// IncrBy wraps redis.Client.IncrBy.
func (w *Wrapper) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.IncrBy(ctx, key, value)
}

// Decr wraps redis.Client.Decr.
func (w *Wrapper) Decr(ctx context.Context, key string) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Decr(ctx, key)
}

// DecrBy wraps redis.Client.DecrBy.
func (w *Wrapper) DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.DecrBy(ctx, key, value)
}

// Exists wraps redis.Client.Exists.
func (w *Wrapper) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	// Increment key counters
	for _, key := range keys {
		w.incrementKey(key)
	}

	return w.client.Exists(ctx, keys...)
}

// Expire wraps redis.Client.Expire.
func (w *Wrapper) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Expire(ctx, key, expiration)
}

// TTL wraps redis.Client.TTL.
func (w *Wrapper) TTL(ctx context.Context, key string) *redis.DurationCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.TTL(ctx, key)
}

// HSet wraps redis.Client.HSet.
func (w *Wrapper) HSet(ctx context.Context, key string, values ...any) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.HSet(ctx, key, values...)
}

// HGet wraps redis.Client.HGet.
func (w *Wrapper) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.HGet(ctx, key, field)
}

// HGetAll wraps redis.Client.HGetAll.
func (w *Wrapper) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.HGetAll(ctx, key)
}

// HMGet wraps redis.Client.HMGet.
func (w *Wrapper) HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.HMGet(ctx, key, fields...)
}

// HMSet wraps redis.Client.HMSet.
func (w *Wrapper) HMSet(ctx context.Context, key string, values ...any) *redis.BoolCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.HMSet(ctx, key, values...)
}

// HDel wraps redis.Client.HDel.
func (w *Wrapper) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.HDel(ctx, key, fields...)
}

// LPush wraps redis.Client.LPush.
func (w *Wrapper) LPush(ctx context.Context, key string, values ...any) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.LPush(ctx, key, values...)
}

// RPush wraps redis.Client.RPush.
func (w *Wrapper) RPush(ctx context.Context, key string, values ...any) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.RPush(ctx, key, values...)
}

// LPop wraps redis.Client.LPop.
func (w *Wrapper) LPop(ctx context.Context, key string) *redis.StringCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.LPop(ctx, key)
}

// RPop wraps redis.Client.RPop.
func (w *Wrapper) RPop(ctx context.Context, key string) *redis.StringCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.RPop(ctx, key)
}

// LLen wraps redis.Client.LLen.
func (w *Wrapper) LLen(ctx context.Context, key string) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.LLen(ctx, key)
}

// LRange wraps redis.Client.LRange.
func (w *Wrapper) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.LRange(ctx, key, start, stop)
}

// SAdd wraps redis.Client.SAdd.
func (w *Wrapper) SAdd(ctx context.Context, key string, members ...any) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.SAdd(ctx, key, members...)
}

// SMembers wraps redis.Client.SMembers.
func (w *Wrapper) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.SMembers(ctx, key)
}

// SRem wraps redis.Client.SRem.
func (w *Wrapper) SRem(ctx context.Context, key string, members ...any) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.SRem(ctx, key, members...)
}

// ZAdd wraps redis.Client.ZAdd.
func (w *Wrapper) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.ZAdd(ctx, key, members...)
}

// ZRange wraps redis.Client.ZRange.
func (w *Wrapper) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.ZRange(ctx, key, start, stop)
}

// ZRangeWithScores wraps redis.Client.ZRangeWithScores.
func (w *Wrapper) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.ZRangeWithScores(ctx, key, start, stop)
}

// ZRank wraps redis.Client.ZRank.
func (w *Wrapper) ZRank(ctx context.Context, key, member string) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.ZRank(ctx, key, member)
}

// ZRem wraps redis.Client.ZRem.
func (w *Wrapper) ZRem(ctx context.Context, key string, members ...any) *redis.IntCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.ZRem(ctx, key, members...)
}

// ZScore wraps redis.Client.ZScore.
func (w *Wrapper) ZScore(ctx context.Context, key, member string) *redis.FloatCmd {
	// Increment key counter
	w.incrementKey(key)

	return w.client.ZScore(ctx, key, member)
}

// Ping wraps redis.Client.Ping.
func (w *Wrapper) Ping(ctx context.Context) *redis.StatusCmd {
	return w.client.Ping(ctx)
}

// Pipeline wraps redis.Client.Pipeline.
func (w *Wrapper) Pipeline() redis.Pipeliner {
	return w.client.Pipeline()
}

// TxPipeline wraps redis.Client.TxPipeline.
func (w *Wrapper) TxPipeline() redis.Pipeliner {
	return w.client.TxPipeline()
}

// Subscribe wraps redis.Client.Subscribe.
func (w *Wrapper) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return w.client.Subscribe(ctx, channels...)
}

// Publish wraps redis.Client.Publish.
func (w *Wrapper) Publish(ctx context.Context, channel string, message any) *redis.IntCmd {
	return w.client.Publish(ctx, channel, message)
}

// asyncSetLocalCache asynchronously sets value in local cache
func (w *Wrapper) asyncSetLocalCache(key, value string) {
	// Get policy manager and try to cache regardless of hot key status
	// This ensures cache miss data gets cached for future hits
	p := w.kf.PolicyManager().GetPolicy(key)
	if p != nil {
		ctx := policy.Context{
			Key:  key,
			Data: policy.SetRequest{Value: value},
		}
		result := p.Apply(ctx)
		_ = result // Cache set operation completed
	}
}

// handleKeySplittingSet implements multi-write for key splitting
func (w *Wrapper) handleKeySplittingSet(
	ctx context.Context, action policy.KeySplittingSetAction, ttl time.Duration,
) *redis.StatusCmd {
	// Write to original key
	originalCmd := w.client.Set(ctx, action.OriginalKey, action.Value, ttl)
	if originalCmd.Err() != nil {
		return originalCmd
	}

	// Asynchronously write to all target shards
	go w.replicateToShards(ctx, action.ShardKeys, action.Value, ttl)

	// Return success from original write
	return originalCmd
}

// replicateToShards writes to shard keys asynchronously
func (w *Wrapper) replicateToShards(
	ctx context.Context, shardKeys []string, value any, ttl time.Duration,
) {
	// Write to all shards
	for _, shardKey := range shardKeys {
		w.client.Set(ctx, shardKey, value, ttl)
	}
}

// handleLookAsideGet implements look-aside pattern for key splitting
func (w *Wrapper) handleLookAsideGet(
	ctx context.Context, action policy.KeySplittingGetAction,
) *redis.StringCmd {
	// Step 1: Try to read from primary shard
	shardResult := w.client.Get(ctx, action.RandShardKey)
	if shardResult.Err() == nil {
		// Shard data exists, return it
		return shardResult
	}

	// Step 2: Shard doesn't exist, try original key
	original := w.client.Get(ctx, action.OriginalKey)
	if original.Err() != nil {
		// Neither shard nor original exists
		return original
	}

	// Step 3: Original data exists, asynchronously replicate to shards
	go w.replicateToShards(ctx, action.ShardKeys, original.Val(), time.Hour)

	// Return original data immediately
	return original
}

// Close wraps redis.Client.Close.
func (w *Wrapper) Close() error {
	return w.client.Close()
}
