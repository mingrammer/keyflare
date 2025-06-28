// Package keyflare provides a client-side hot key detection engine for caching systems.
package keyflare

import (
	"time"

	"github.com/mingrammer/keyflare/internal"
	"github.com/mingrammer/keyflare/internal/detector"
	"github.com/mingrammer/keyflare/internal/metrics"
	"github.com/mingrammer/keyflare/internal/policy"
)

// Default configuration constants
const (
	// Detector defaults
	DefaultDetectorErrorRate     = 0.01
	DefaultDetectorTopK          = 100
	DefaultDetectorDecayFactor   = 0.98
	DefaultDetectorDecayInterval = 60 * time.Second
	DefaultDetectorHotThreshold  = 0

	// Policy defaults
	DefaultLocalCacheTTL          = 60.0
	DefaultLocalCacheJitter       = 0.2
	DefaultLocalCacheCapacity     = 1000.0
	DefaultLocalCacheRefreshAhead = 0.8

	DefaultKeySplittingShards = 10.0

	// Metrics defaults
	DefaultMetricsNamespace          = "keyflare"
	DefaultMetricsServerAddress      = ":9121"
	DefaultMetricsCollectionInterval = 15 * time.Second
	DefaultMetricsHotKeyLimit        = 10
	DefaultMetricsHotKeyHistorySize  = 10
	DefaultMetricsEnableAPI          = true
)

// PolicyType defines the type of policy
type PolicyType string

const (
	// LocalCache represents local in-memory cache policy
	LocalCache PolicyType = "local-cache"
	// KeySplitting represents key splitting policy
	KeySplitting PolicyType = "key-splitting"
)

// Options contains configuration options for KeyFlare
type Options struct {
	// DetectorOptions configures the hot key detector
	DetectorOptions DetectorOptions

	// PolicyOptions configures the policy manager
	PolicyOptions PolicyOptions

	// MetricsOptions configures the metrics collector
	MetricsOptions MetricsOptions

	// EnableMetrics determines whether to enable metrics collection
	EnableMetrics bool
}

// DetectorOptions contains configuration options for the detector
type DetectorOptions struct {
	// ErrorRate is the acceptable error rate for probabilistic algorithms
	ErrorRate float64

	// TopK is the number of top hot keys to track
	TopK int

	// DecayFactor is used to decay old counts over time
	DecayFactor float64

	// DecayInterval is the interval at which decay is applied (in seconds)
	DecayInterval time.Duration

	// HotThreshold is the threshold for determining if a key is hot
	// If it's 0, then the threshold is dynamically determined based on the Top-K keys
	HotThreshold uint64
}

// PolicyOptions contains configuration options for policy management
type PolicyOptions struct {
	// Type determines which policy to use
	Type PolicyType

	// Parameters holds the policy-specific parameters
	Parameters any

	// WhitelistKeys is a list of keys to whitelist
	// TODO: support auto whitelisting
	WhitelistKeys []string

	// WhitelistPatterns is a list of regex patterns to whitelist keys
	WhitelistPatterns []string
}

// MetricsOptions contains configuration options for metrics
type MetricsOptions struct {
	// Namespace is the namespace for metrics
	Namespace string

	// MetricServerAddress is the address for the metric server
	MetricServerAddress string

	// CollectionInterval is the interval at which metrics are collected (in seconds)
	CollectionInterval time.Duration

	// HotKeyMetricLimit is the number of hot keys to expose as metrics (default: 10)
	HotKeyMetricLimit int

	// HotKeyHistorySize is the number of historical snapshots to keep (default: 10)
	HotKeyHistorySize int

	// EnableAPI enables the hot keys API endpoint
	EnableAPI bool
}

// LocalCacheParams defines parameters for local cache policy
type LocalCacheParams struct {
	// TTL is the time-to-live for cached items in seconds
	TTL float64 `json:"ttl"`

	// Jitter is the randomness factor for TTL (0.0-1.0)
	Jitter float64 `json:"jitter"`

	// Capacity is the maximum number of items in the cache
	Capacity float64 `json:"capacity"`

	// RefreshAhead determines when to refresh items before expiration (0.0-1.0)
	RefreshAhead float64 `json:"refresh_ahead"`
}

// KeySplittingParams defines parameters for key splitting policy
type KeySplittingParams struct {
	// Shards is the number of shards to split keys into
	Shards int64 `json:"shards"`
}

// KeyCount represents a key and its estimated count
type KeyCount struct {
	Key   string
	Count uint64
}

// HotKeyInfo contains detailed information about a hot key (for API responses)
type HotKeyInfo struct {
	Key       string `json:"key"`
	Count     uint64 `json:"count"`
	Rank      int    `json:"rank"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
	Trend     string `json:"trend"` // "rising", "falling", "stable"
}

// HotKeysResponse is the API response for hot keys
type HotKeysResponse struct {
	Timestamp   string       `json:"timestamp"`
	TopK        int          `json:"top_k"`
	TotalKeys   int          `json:"total_keys"`
	Keys        []HotKeyInfo `json:"keys"`
	QueryLimit  int          `json:"query_limit"`
	ActualLimit int          `json:"actual_limit"`
}

// Option is a function that modifies KeyFlare options
type Option func(*Options)

// DefaultOptions returns the default configuration for KeyFlare
func DefaultOptions() Options {
	return Options{
		DetectorOptions: DefaultDetectorOptions(),
		PolicyOptions:   DefaultPolicyOptions(),
		MetricsOptions:  DefaultMetricsOptions(),
		EnableMetrics:   true,
	}
}

// DefaultDetectorOptions returns the default configuration for the detector
func DefaultDetectorOptions() DetectorOptions {
	return DetectorOptions{
		ErrorRate:     DefaultDetectorErrorRate,
		TopK:          DefaultDetectorTopK,
		DecayFactor:   DefaultDetectorDecayFactor,
		DecayInterval: DefaultDetectorDecayInterval,
		HotThreshold:  DefaultDetectorHotThreshold,
	}
}

// DefaultPolicyOptions returns the default configuration for policy management
func DefaultPolicyOptions() PolicyOptions {
	return PolicyOptions{
		Type:              LocalCache,
		Parameters:        DefaultLocalCacheParams(),
		WhitelistKeys:     []string{},
		WhitelistPatterns: []string{},
	}
}

// DefaultMetricsOptions returns the default configuration for metrics
func DefaultMetricsOptions() MetricsOptions {
	return MetricsOptions{
		Namespace:           DefaultMetricsNamespace,
		MetricServerAddress: DefaultMetricsServerAddress,
		CollectionInterval:  DefaultMetricsCollectionInterval,
		HotKeyMetricLimit:   DefaultMetricsHotKeyLimit,
		HotKeyHistorySize:   DefaultMetricsHotKeyHistorySize,
		EnableAPI:           DefaultMetricsEnableAPI,
	}
}

// DefaultLocalCacheParams returns default parameters for local cache policy
func DefaultLocalCacheParams() LocalCacheParams {
	return LocalCacheParams{
		TTL:          DefaultLocalCacheTTL,
		Jitter:       DefaultLocalCacheJitter,
		Capacity:     DefaultLocalCacheCapacity,
		RefreshAhead: DefaultLocalCacheRefreshAhead,
	}
}

// DefaultKeySplittingParams returns default parameters for key splitting policy
func DefaultKeySplittingParams() KeySplittingParams {
	return KeySplittingParams{
		Shards: DefaultKeySplittingShards,
	}
}

// WithDetectorOptions sets the detector options
func WithDetectorOptions(opts DetectorOptions) Option {
	return func(o *Options) {
		o.DetectorOptions = opts
	}
}

// WithPolicyOptions sets policy options
func WithPolicyOptions(opts PolicyOptions) Option {
	return func(o *Options) {
		o.PolicyOptions = opts
	}
}

// WithMetricsOptions sets the metrics options
func WithMetricsOptions(opts MetricsOptions) Option {
	return func(o *Options) {
		o.MetricsOptions = opts
	}
}

// WithMetricsEnabled sets whether metrics are enabled
func WithMetricsEnabled(enabled bool) Option {
	return func(o *Options) {
		o.EnableMetrics = enabled
	}
}

// New creates and returns the global KeyFlare instance
func New(opts ...Option) error {
	// Start with default options
	options := DefaultOptions()

	// Apply all provided options
	for _, opt := range opts {
		opt(&options)
	}

	// Apply defaults to any unset fields
	options = applyOptionsDefaults(options)

	// Convert to internal config
	config := internal.Config{
		DetectorConfig: detector.Config{
			ErrorRate:     options.DetectorOptions.ErrorRate,
			TopK:          options.DetectorOptions.TopK,
			DecayFactor:   options.DetectorOptions.DecayFactor,
			DecayInterval: time.Duration(options.DetectorOptions.DecayInterval) * time.Second,
			HotThreshold:  options.DetectorOptions.HotThreshold,
		},
		PolicyConfig: policy.Config{
			Type:              policy.Type(options.PolicyOptions.Type),
			Parameters:        convertPolicyParams(options.PolicyOptions.Type, options.PolicyOptions.Parameters),
			WhitelistKeys:     options.PolicyOptions.WhitelistKeys,
			WhitelistPatterns: options.PolicyOptions.WhitelistPatterns,
		},
		MetricsConfig: metrics.Config{
			Namespace:           options.MetricsOptions.Namespace,
			MetricServerAddress: options.MetricsOptions.MetricServerAddress,
			CollectionInterval:  time.Duration(options.MetricsOptions.CollectionInterval) * time.Second,
			HotKeyMetricLimit:   options.MetricsOptions.HotKeyMetricLimit,
			HotKeyHistorySize:   options.MetricsOptions.HotKeyHistorySize,
		},
		EnableMetrics: options.EnableMetrics,
	}

	return internal.New(config)
}

// Start starts the global KeyFlare instance
func Start() error {
	return internal.Start()
}

// Stop stops the global KeyFlare instance
func Stop() error {
	return internal.Stop()
}

// Shutdown stops and clears the global KeyFlare instance
func Shutdown() error {
	return internal.Shutdown()
}

// applyOptionsDefaults applies default values to missing fields in the provided options
func applyOptionsDefaults(opts Options) Options {
	opts.DetectorOptions = applyDetectorDefaults(opts.DetectorOptions)
	opts.PolicyOptions = applyPolicyDefaults(opts.PolicyOptions)
	opts.MetricsOptions = applyMetricsDefaults(opts.MetricsOptions)
	return opts
}

func applyDetectorDefaults(opts DetectorOptions) DetectorOptions {
	if opts.ErrorRate <= 0 {
		opts.ErrorRate = DefaultDetectorErrorRate
	}
	if opts.TopK <= 0 {
		opts.TopK = DefaultDetectorTopK
	}
	if opts.DecayFactor <= 0 {
		opts.DecayFactor = DefaultDetectorDecayFactor
	}
	if opts.DecayInterval <= 0 {
		opts.DecayInterval = DefaultDetectorDecayInterval
	}
	// HotThreshold can be 0, so no default override needed
	return opts
}

func applyLocalCacheDefaults(params LocalCacheParams) LocalCacheParams {
	if params.TTL <= 0 {
		params.TTL = DefaultLocalCacheTTL
	}
	if params.Jitter <= 0 {
		params.Jitter = DefaultLocalCacheJitter
	}
	if params.Capacity <= 0 {
		params.Capacity = DefaultLocalCacheCapacity
	}
	if params.RefreshAhead <= 0 {
		params.RefreshAhead = DefaultLocalCacheRefreshAhead
	}
	return params
}

func applyKeySplittingDefaults(params KeySplittingParams) KeySplittingParams {
	if params.Shards <= 0 {
		params.Shards = DefaultKeySplittingShards
	}
	return params
}

func applyPolicyDefaults(opts PolicyOptions) PolicyOptions {
	if opts.Type == "" {
		opts.Type = LocalCache
	}

	// Apply parameter defaults based on policy type
	switch opts.Type {
	case LocalCache:
		if opts.Parameters == nil {
			opts.Parameters = DefaultLocalCacheParams()
		} else if params, ok := opts.Parameters.(LocalCacheParams); ok {
			opts.Parameters = applyLocalCacheDefaults(params)
		}
	case KeySplitting:
		if opts.Parameters == nil {
			opts.Parameters = DefaultKeySplittingParams()
		} else if params, ok := opts.Parameters.(KeySplittingParams); ok {
			opts.Parameters = applyKeySplittingDefaults(params)
		}
	}

	if opts.WhitelistKeys == nil {
		opts.WhitelistKeys = []string{}
	}
	if opts.WhitelistPatterns == nil {
		opts.WhitelistPatterns = []string{}
	}
	return opts
}

func applyMetricsDefaults(opts MetricsOptions) MetricsOptions {
	if opts.Namespace == "" {
		opts.Namespace = DefaultMetricsNamespace
	}
	if opts.MetricServerAddress == "" {
		opts.MetricServerAddress = DefaultMetricsServerAddress
	}
	if opts.CollectionInterval <= 0 {
		opts.CollectionInterval = DefaultMetricsCollectionInterval
	}
	if opts.HotKeyMetricLimit <= 0 {
		opts.HotKeyMetricLimit = DefaultMetricsHotKeyLimit
	}
	if opts.HotKeyHistorySize <= 0 {
		opts.HotKeyHistorySize = DefaultMetricsHotKeyHistorySize
	}
	// EnableAPI defaults to true, handled in default options
	return opts
}

// convertPolicyParams converts public policy parameters to internal types
func convertPolicyParams(policyType PolicyType, params any) any {
	switch policyType {
	case LocalCache:
		if p, ok := params.(LocalCacheParams); ok {
			return policy.LocalCacheConfig{
				TTL:          p.TTL,
				Jitter:       p.Jitter,
				Capacity:     p.Capacity,
				RefreshAhead: p.RefreshAhead,
			}
		}
	case KeySplitting:
		if p, ok := params.(KeySplittingParams); ok {
			return policy.KeySplittingConfig{
				Shards: p.Shards,
			}
		}
	}
	return nil
}
