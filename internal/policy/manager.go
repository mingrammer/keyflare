// Package policy provides policy management and execution functionality
package policy

import (
	"fmt"
	"regexp"
	"sync"
)

// Type defines the type of policy
type Type string

const (
	// LocalCache represents local in-memory cache policy
	LocalCache Type = "local-cache"
	// KeySplitting represents key splitting policy
	KeySplitting Type = "key-splitting"
)

// Config contains configuration options for policy management
type Config struct {
	// Type determines which policy to use
	Type Type

	// Parameters holds the policy-specific parameters
	Parameters any

	// WhitelistKeys is a list of keys to whitelist
	WhitelistKeys []string

	// WhitelistPatterns is a list of regex patterns to whitelist keys
	WhitelistPatterns []string
}

// LocalCacheConfig defines parameters for local cache policy
type LocalCacheConfig struct {
	// TTL is the time-to-live for cached items in seconds
	TTL float64

	// Jitter is the randomness factor for TTL (0.0-1.0)
	Jitter float64

	// Capacity is the maximum number of items in the cache
	Capacity float64

	// RefreshAhead determines when to refresh items before expiration (0.0-1.0)
	RefreshAhead float64
}

// KeySplittingConfig defines parameters for key splitting policy
type KeySplittingConfig struct {
	// Shards is the number of shards to split keys into
	Shards int64
}

// Context contains runtime context for policy execution
type Context struct {
	Key  string
	Data any
}

// Result contains the result of policy execution
type Result struct {
	Data  any
	Error error
}

// Policy defines the interface for a policy
type Policy interface {
	// Apply applies the policy on the given context and returns the result
	Apply(ctx Context) Result
}

// Manager defines the interface for policy management
type Manager interface {
	// GetPolicy returns the policy for a given key
	GetPolicy(key string) Policy

	// RegisterPattern registers a pattern-based policy selection rule
	RegisterPattern(pattern string) error

	// AddWhitelistKey adds a key to the whitelist
	AddWhitelistKey(key string)

	// RemoveWhitelistKey removes a key from the whitelist
	RemoveWhitelistKey(key string)
}

// manager implements the Manager interface
type manager struct {
	policy         Policy
	patternRegexps map[string]*regexp.Regexp
	whitelistKeys  map[string]bool
	mu             sync.RWMutex
}

// New creates a new policy manager with the provided configuration
func New(config Config) (Manager, error) {
	var p Policy

	switch config.Type {
	case LocalCache:
		params, ok := config.Parameters.(LocalCacheConfig)
		if !ok {
			return nil, fmt.Errorf("invalid parameters type for LocalCache policy: expected LocalCacheConfig, got %T", config.Parameters)
		}
		p = newLocalCachePolicy(params)
	case KeySplitting:
		params, ok := config.Parameters.(KeySplittingConfig)
		if !ok {
			return nil, fmt.Errorf("invalid parameters type for KeySplitting policy: expected KeySplittingConfig, got %T", config.Parameters)
		}
		p = newKeySplittingPolicy(params)
	default:
		return nil, fmt.Errorf("unsupported policy type: %s", config.Type)
	}

	m := &manager{
		policy:         p,
		patternRegexps: make(map[string]*regexp.Regexp),
		whitelistKeys:  make(map[string]bool),
		mu:             sync.RWMutex{},
	}

	// Add whitelist keys
	for _, key := range config.WhitelistKeys {
		m.whitelistKeys[key] = true
	}

	// Add whitelist patterns
	for _, pattern := range config.WhitelistPatterns {
		if err := m.RegisterPattern(pattern); err != nil {
			return nil, fmt.Errorf("invalid whitelist pattern '%s': %w", pattern, err)
		}
	}

	return m, nil
}

// GetPolicy returns the policy for a given key
func (m *manager) GetPolicy(key string) Policy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if key is in whitelist
	if m.whitelistKeys[key] {
		return m.policy
	}

	// Check if any registered pattern matches the key
	for _, r := range m.patternRegexps {
		if r.MatchString(key) {
			return m.policy
		}
	}

	// Return nil if key doesn't match any pattern or whitelist
	return nil
}

// RegisterPattern registers a pattern-based policy selection rule
func (m *manager) RegisterPattern(pattern string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Compile the pattern into a regular expression
	r, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	// Register the pattern
	m.patternRegexps[pattern] = r
	return nil
}

// AddWhitelistKey adds a key to the whitelist
func (m *manager) AddWhitelistKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelistKeys[key] = true
}

// RemoveWhitelistKey removes a key from the whitelist
func (m *manager) RemoveWhitelistKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.whitelistKeys, key)
}
