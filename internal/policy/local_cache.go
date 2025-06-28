package policy

import (
	"crypto/rand"
	"fmt"
	"math"
	"sync"
	"time"
)

// CacheItem represents an item stored in the local cache
type CacheItem struct {
	Key        string
	Value      any
	Expiration time.Time
	RefreshAt  time.Time // Time when refresh should be triggered
}

// IsExpired checks if the cache item has expired
func (c *CacheItem) IsExpired() bool {
	return time.Now().After(c.Expiration)
}

// ShouldRefresh checks if the cache item should be refreshed
func (c *CacheItem) ShouldRefresh() bool {
	return time.Now().After(c.RefreshAt)
}

// localCachePolicy implements the Policy interface for local cache
type localCachePolicy struct {
	config LocalCacheConfig
	// Consider using a dedicated caching package like ristretto for better performance
	// Alternatively, sync.Map could suffice since hot keys are typically few in number
	cache map[string]*CacheItem
	mu    sync.RWMutex
	size  int
}

// newLocalCachePolicy creates a new local cache policy
func newLocalCachePolicy(config LocalCacheConfig) Policy {
	return &localCachePolicy{
		config: config,
		cache:  make(map[string]*CacheItem),
		mu:     sync.RWMutex{},
		size:   0,
	}
}

// applies the policy on the given context and returns the result
func (p *localCachePolicy) Apply(ctx Context) Result {
	switch ctx.Data.(type) {
	case GetRequest:
		return p.handleGet(ctx)
	case SetRequest:
		return p.handleSet(ctx)
	default:
		return Result{
			Data:  nil,
			Error: fmt.Errorf("unsupported operation type: %T", ctx.Data),
		}
	}
}

// handleGet handles GET operations
func (p *localCachePolicy) handleGet(ctx Context) Result {
	p.mu.RLock()
	item, ok := p.cache[ctx.Key]
	p.mu.RUnlock()

	if !ok {
		return Result{
			Data: CacheMiss{Key: ctx.Key},
		}
	}

	// Check if item is expired
	if item.IsExpired() {
		// Remove expired item
		p.mu.Lock()
		delete(p.cache, ctx.Key)
		p.size--
		p.mu.Unlock()

		return Result{
			Data: CacheMiss{Key: ctx.Key},
		}
	}

	// Check if item should be refreshed
	shouldRefresh := item.ShouldRefresh()

	return Result{
		Data: CacheHit{
			Key:           ctx.Key,
			Value:         item.Value,
			ShouldRefresh: shouldRefresh,
		},
	}
}

// handleSet handles SET operations
func (p *localCachePolicy) handleSet(ctx Context) Result {
	req, ok := ctx.Data.(SetRequest)
	if !ok {
		return Result{
			Data:  nil,
			Error: fmt.Errorf("invalid set request type"),
		}
	}

	// Check capacity before adding new item
	p.mu.Lock()
	defer p.mu.Unlock()

	// If key doesn't exist and we're at capacity, evict LRU item
	if _, ok := p.cache[ctx.Key]; !ok && p.size >= int(p.config.Capacity) {
		p.evictLRU()
	}

	// Calculate TTL with jitter
	ttl := p.calculateTTLWithJitter()
	expiration := time.Now().Add(time.Duration(ttl) * time.Second)
	refreshAt := time.Now().Add(time.Duration(ttl*p.config.RefreshAhead) * time.Second)

	// Create cache item
	item := &CacheItem{
		Key:        ctx.Key,
		Value:      req.Value,
		Expiration: expiration,
		RefreshAt:  refreshAt,
	}

	// Store in cache
	if _, ok := p.cache[ctx.Key]; !ok {
		p.size++
	}
	p.cache[ctx.Key] = item

	return Result{
		Data: CacheSet{Key: ctx.Key, TTL: ttl},
	}
}

// calculateTTLWithJitter calculates TTL with random jitter
func (p *localCachePolicy) calculateTTLWithJitter() float64 {
	if p.config.Jitter <= 0 {
		return p.config.TTL
	}

	// Generate random jitter between -jitter and +jitter
	jitterRange := p.config.TTL * p.config.Jitter
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)

	// Convert bytes to float64 between -1 and 1
	randomValue := float64(int64(randomBytes[0])<<56|
		int64(randomBytes[1])<<48|
		int64(randomBytes[2])<<40|
		int64(randomBytes[3])<<32|
		int64(randomBytes[4])<<24|
		int64(randomBytes[5])<<16|
		int64(randomBytes[6])<<8|
		int64(randomBytes[7])) / float64(math.MaxInt64)

	jitter := randomValue * jitterRange
	return p.config.TTL + jitter
}

// evictLRU evicts the least recently used item from cache
// Note: This is a simplified LRU implementation
// In production, you might want to use a more sophisticated LRU algorithm
func (p *localCachePolicy) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, item := range p.cache {
		if first || item.Expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Expiration
			first = false
		}
	}

	if oldestKey != "" {
		delete(p.cache, oldestKey)
		p.size--
	}
}

// GetCacheStats returns cache statistics for monitoring
func (p *localCachePolicy) GetCacheStats() CacheStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	expiredCount := 0
	for _, item := range p.cache {
		if item.IsExpired() {
			expiredCount++
		}
	}

	return CacheStats{
		Size:         p.size,
		Capacity:     int(p.config.Capacity),
		ExpiredItems: expiredCount,
	}
}

// Request types for different operations
type GetRequest struct{}

type SetRequest struct {
	Value any
	TTL   *float64 // Optional TTL override
}

// Response types for different operations
type CacheHit struct {
	Key           string
	Value         any
	ShouldRefresh bool
}

type CacheMiss struct {
	Key string
}

type CacheSet struct {
	Key string
	TTL float64
}

type CacheStats struct {
	Size         int
	Capacity     int
	ExpiredItems int
}
