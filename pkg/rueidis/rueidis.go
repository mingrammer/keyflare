// Package rueidis provides a Rueidis client wrapper with KeyFlare hot key detection.
package rueidis

import (
	"context"
	"fmt"
	"time"

	"github.com/mingrammer/keyflare/internal"
	"github.com/redis/rueidis"
)

// Wrapper wraps a rueidis client with KeyFlare hot key detection.
type Wrapper struct {
	client rueidis.Client
	kf     *internal.KeyFlare
}

// Wrap creates a new Rueidis client wrapper with the provided client.
// It uses the global KeyFlare instance which must be initialized and started first.
func Wrap(client rueidis.Client) (*Wrapper, error) {
	kf, err := internal.GetInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to get KeyFlare instance: %w", err)
	}

	return &Wrapper{
		client: client,
		kf:     kf,
	}, nil
}

// Client returns the underlying Rueidis client.
func (w *Wrapper) Client() rueidis.Client {
	return w.client
}

// extractKeyFromCommand attempts to extract the key from a Redis command.
// It uses the Commands() method which returns the command as a slice of strings.
// For most Redis commands, the key is at index 1 (after the command name).
func extractKeyFromCommand(cmd rueidis.Completed) string {
	commands := cmd.Commands()
	if len(commands) > 1 {
		return commands[1] // Key is typically at index 1
	}
	return "" // No key found
}

// extractKeyFromCacheable attempts to extract the key from a cacheable command.
func extractKeyFromCacheable(cmd rueidis.Cacheable) string {
	// Cacheable commands also have Commands() method
	commands := cmd.Commands()
	if len(commands) > 1 {
		return commands[1]
	}
	return ""
}

// incrementKey increments the key counter in the detector.
func (w *Wrapper) incrementKey(key string) {
	if key != "" { // Only track non-empty keys
		w.kf.Detector().Increment(key, 1)
	}
}

// Do wraps rueidis.Client.Do.
func (w *Wrapper) Do(
	ctx context.Context, cmd rueidis.Completed,
) rueidis.RedisResult {
	// Extract and track key automatically using Commands() method
	key := extractKeyFromCommand(cmd)
	w.incrementKey(key)

	return w.client.Do(ctx, cmd)
}

// DoCache wraps rueidis.Client.DoCache.
func (w *Wrapper) DoCache(
	ctx context.Context, cmd rueidis.Cacheable, ttl time.Duration,
) rueidis.RedisResult {
	// Extract and track key automatically using Commands() method
	key := extractKeyFromCacheable(cmd)
	w.incrementKey(key)

	return w.client.DoCache(ctx, cmd, ttl)
}

// DoMulti wraps rueidis.Client.DoMulti.
func (w *Wrapper) DoMulti(
	ctx context.Context, multi ...rueidis.Completed,
) []rueidis.RedisResult {
	// Extract and track keys automatically for all commands
	for _, cmd := range multi {
		key := extractKeyFromCommand(cmd)
		w.incrementKey(key)
	}

	return w.client.DoMulti(ctx, multi...)
}

// DoMultiCache wraps rueidis.Client.DoMultiCache.
func (w *Wrapper) DoMultiCache(
	ctx context.Context, multi ...rueidis.CacheableTTL,
) []rueidis.RedisResult {
	// Extract and track keys automatically for all cacheable commands
	for _, cacheable := range multi {
		key := extractKeyFromCacheable(cacheable.Cmd)
		w.incrementKey(key)
	}

	return w.client.DoMultiCache(ctx, multi...)
}

// DoStream wraps rueidis.Client.DoStream.
func (w *Wrapper) DoStream(
	ctx context.Context, cmd rueidis.Completed,
) rueidis.RedisResultStream {
	// Extract and track key automatically
	key := extractKeyFromCommand(cmd)
	w.incrementKey(key)

	return w.client.DoStream(ctx, cmd)
}

// DoMultiStream wraps rueidis.Client.DoMultiStream.
func (w *Wrapper) DoMultiStream(
	ctx context.Context, multi ...rueidis.Completed,
) rueidis.MultiRedisResultStream {
	// Extract and track keys automatically for all commands
	for _, cmd := range multi {
		key := extractKeyFromCommand(cmd)
		w.incrementKey(key)
	}

	return w.client.DoMultiStream(ctx, multi...)
}

// B wraps rueidis.Client.B.
func (w *Wrapper) B() rueidis.Builder {
	return w.client.B()
}

// Dedicated wraps rueidis.Client.Dedicated.
func (w *Wrapper) Dedicated(fn func(rueidis.DedicatedClient) error) error {
	return w.client.Dedicated(fn)
}

// Dedicate wraps rueidis.Client.Dedicate.
func (w *Wrapper) Dedicate() (rueidis.DedicatedClient, func()) {
	return w.client.Dedicate()
}

// Nodes wraps rueidis.Client.Nodes.
func (w *Wrapper) Nodes() map[string]rueidis.Client {
	return w.client.Nodes()
}

// Mode wraps rueidis.Client.Mode.
func (w *Wrapper) Mode() rueidis.ClientMode {
	return w.client.Mode()
}

// Close wraps rueidis.Client.Close.
func (w *Wrapper) Close() {
	w.client.Close()
}

// DedicatedWrapper wraps a rueidis dedicated client with KeyFlare.
type DedicatedWrapper struct {
	client rueidis.DedicatedClient
	kf     *internal.KeyFlare
}

// WrapDedicated creates a new Rueidis dedicated client wrapper.
// It uses the global KeyFlare instance which must be initialized and started first.
func WrapDedicated(client rueidis.DedicatedClient) (*DedicatedWrapper, error) {
	kf, err := internal.GetInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to get KeyFlare instance: %w", err)
	}

	return &DedicatedWrapper{
		client: client,
		kf:     kf,
	}, nil
}

// Client returns the underlying Rueidis dedicated client.
func (w *DedicatedWrapper) Client() rueidis.DedicatedClient {
	return w.client
}

// incrementKey increments the key counter in the detector.
func (w *DedicatedWrapper) incrementKey(key string) {
	if key != "" { // Only track non-empty keys
		w.kf.Detector().Increment(key, 1)
	}
}

// Do wraps rueidis.DedicatedClient.Do.
func (w *DedicatedWrapper) Do(
	ctx context.Context, cmd rueidis.Completed,
) rueidis.RedisResult {
	// Extract and track key automatically
	key := extractKeyFromCommand(cmd)
	w.incrementKey(key)

	return w.client.Do(ctx, cmd)
}

// DoMulti wraps rueidis.DedicatedClient.DoMulti.
func (w *DedicatedWrapper) DoMulti(
	ctx context.Context, multi ...rueidis.Completed,
) []rueidis.RedisResult {
	// Extract and track keys automatically for all commands
	for _, cmd := range multi {
		key := extractKeyFromCommand(cmd)
		w.incrementKey(key)
	}

	return w.client.DoMulti(ctx, multi...)
}

// B wraps rueidis.DedicatedClient.B.
func (w *DedicatedWrapper) B() rueidis.Builder {
	return w.client.B()
}

// Receive wraps rueidis.DedicatedClient.Receive.
func (w *DedicatedWrapper) Receive(
	ctx context.Context, cmd rueidis.Completed, fn func(msg rueidis.PubSubMessage),
) error {
	return w.client.Receive(ctx, cmd, fn)
}

// SetPubSubHooks wraps rueidis.DedicatedClient.SetPubSubHooks.
func (w *DedicatedWrapper) SetPubSubHooks(hooks rueidis.PubSubHooks) <-chan error {
	return w.client.SetPubSubHooks(hooks)
}

// Close wraps rueidis.DedicatedClient.Close.
func (w *DedicatedWrapper) Close() {
	w.client.Close()
}
