// Package memcached provides a Memcached client wrapper with KeyFlare hot key detection.
package memcached

import (
	"fmt"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/mingrammer/keyflare/internal"
	"github.com/mingrammer/keyflare/internal/policy"
)

// Wrapper wraps a gomemcache/memcache client with hot key detection.
type Wrapper struct {
	client *memcache.Client
	kf     *internal.KeyFlare
}

// Wrap creates a new Memcached client wrapper with the provided client.
// It uses the global KeyFlare instance which must be initialized and started first.
func Wrap(client *memcache.Client) (*Wrapper, error) {
	kf, err := internal.GetInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to get KeyFlare instance: %w. Call keyflare.New() and keyflare.Start() first", err)
	}

	return &Wrapper{
		client: client,
		kf:     kf,
	}, nil
}

// Client returns the underlying Memcached client.
func (w *Wrapper) Client() *memcache.Client {
	return w.client
}

// incrementKey increments the key counter in the detector.
func (w *Wrapper) incrementKey(key string) {
	w.kf.Detector().Increment(key, 1)
}

// applyPolicyIfHot applies the policy if the key is hot.
func (w *Wrapper) applyPolicyIfHot(key string) (any, error) {
	if w.kf.Detector().IsHot(key) {
		p := w.kf.PolicyManager().GetPolicy(key)
		if p != nil {
			ctx := policy.Context{
				Key: key,
			}
			result := p.Apply(ctx)

			if result.Error == nil {
				return result.Data, nil
			}
		}
	}

	return nil, nil
}

// Get wraps memcache.Client.Get.
func (w *Wrapper) Get(key string) (*memcache.Item, error) {
	// Increment key counter
	w.incrementKey(key)

	// Try to apply policy if hot
	if value, err := w.applyPolicyIfHot(key); err != nil || value != nil {
		// If policy was applied and returned a result
		if err != nil {
			return nil, err
		}

		if value != nil {
			switch v := value.(type) {
			case *memcache.Item:
				return v, nil
			case []byte:
				return &memcache.Item{
					Key:   key,
					Value: v,
				}, nil
			case string:
				return &memcache.Item{
					Key:   key,
					Value: []byte(v),
				}, nil
			}
		}
	}

	// If no policy was applied or policy returned nil, call the original method
	return w.client.Get(key)
}

// GetMulti wraps memcache.Client.GetMulti.
func (w *Wrapper) GetMulti(keys []string) (map[string]*memcache.Item, error) {
	// Increment key counters
	for _, key := range keys {
		w.incrementKey(key)
	}

	return w.client.GetMulti(keys)
}

// Set wraps memcache.Client.Set.
func (w *Wrapper) Set(item *memcache.Item) error {
	// Increment key counter
	w.incrementKey(item.Key)

	return w.client.Set(item)
}

// Add wraps memcache.Client.Add.
func (w *Wrapper) Add(item *memcache.Item) error {
	// Increment key counter
	w.incrementKey(item.Key)

	return w.client.Add(item)
}

// Replace wraps memcache.Client.Replace.
func (w *Wrapper) Replace(item *memcache.Item) error {
	// Increment key counter
	w.incrementKey(item.Key)

	return w.client.Replace(item)
}

// Delete wraps memcache.Client.Delete.
func (w *Wrapper) Delete(key string) error {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Delete(key)
}

// Increment wraps memcache.Client.Increment.
func (w *Wrapper) Increment(key string, delta uint64) (uint64, error) {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Increment(key, delta)
}

// Decrement wraps memcache.Client.Decrement.
func (w *Wrapper) Decrement(key string, delta uint64) (uint64, error) {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Decrement(key, delta)
}

// CompareAndSwap wraps memcache.Client.CompareAndSwap.
func (w *Wrapper) CompareAndSwap(item *memcache.Item) error {
	// Increment key counter
	w.incrementKey(item.Key)

	return w.client.CompareAndSwap(item)
}

// Touch wraps memcache.Client.Touch.
func (w *Wrapper) Touch(key string, seconds int32) error {
	// Increment key counter
	w.incrementKey(key)

	return w.client.Touch(key, seconds)
}

// FlushAll wraps memcache.Client.FlushAll.
func (w *Wrapper) FlushAll() error {
	return w.client.FlushAll()
}

// Ping wraps memcache.Client.Ping.
func (w *Wrapper) Ping() error {
	return w.client.Ping()
}
