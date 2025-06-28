// Package metrics provides metrics collection and export functionality
package metrics

import (
	"time"

	"github.com/mingrammer/keyflare/internal/detector"
)

const (
	DefaultHotKeyMetricLimit  = 10
	DefaultHotKeyHistorySize  = 10
	DefaultCollectionInterval = 15 * time.Second
)

// Config contains configuration options for metrics
type Config struct {
	// Namespace is the namespace for metrics
	Namespace string

	// MetricServerAddress is the address for the metric server
	MetricServerAddress string

	// CollectionInterval is the interval at which metrics are collected
	CollectionInterval time.Duration

	// HotKeyMetricLimit is the number of hot keys to expose as metrics (default: 10)
	HotKeyMetricLimit int

	// HotKeyHistorySize is the number of historical snapshots to keep (default: 10)
	HotKeyHistorySize int
}

// Collector defines the interface for metrics collection
type Collector interface {
	// RecordKeyAccess records a key access (for hot key detection)
	RecordKeyAccess(key string)

	// RecordPolicyApplication records a policy application
	RecordPolicyApplication(policy string, success bool)

	// UpdateHotKeys updates the hot keys metric
	UpdateHotKeys(hotKeys []detector.KeyCount)

	// SetDetector sets the detector for metrics collection
	SetDetector(d detector.Detector)

	// Start starts the metrics collector
	Start() error

	// Stop stops the metrics collector
	Stop() error
}

// hotKeySnapshot represents a point-in-time snapshot of hot keys
type hotKeySnapshot struct {
	Timestamp time.Time           `json:"timestamp"`
	Keys      []detector.KeyCount `json:"keys"`
	KeyMeta   map[string]*keyMeta `json:"-"` // Internal metadata
}

// keyMeta tracks metadata for each key
type keyMeta struct {
	FirstSeen time.Time
	LastSeen  time.Time
	PrevCount uint64
}

// New creates a new metrics collector with the provided configuration
func New(config Config) Collector {
	if config.HotKeyMetricLimit <= 0 {
		config.HotKeyMetricLimit = DefaultHotKeyMetricLimit
	}
	if config.HotKeyHistorySize <= 0 {
		config.HotKeyHistorySize = DefaultHotKeyHistorySize
	}
	if config.CollectionInterval <= 0 {
		config.CollectionInterval = DefaultCollectionInterval
	}

	return newMetricServer(config)
}

// NewNoop creates a new no-op collector
func NewNoop() Collector {
	return &noopCollector{}
}

// noopCollector is a no-op implementation of Collector
type noopCollector struct{}

func (c *noopCollector) RecordKeyAccess(key string)                          {}
func (c *noopCollector) RecordPolicyApplication(policy string, success bool) {}
func (c *noopCollector) UpdateHotKeys(hotKeys []detector.KeyCount)           {}
func (c *noopCollector) SetDetector(d detector.Detector)                     {}
func (c *noopCollector) Start() error                                        { return nil }
func (c *noopCollector) Stop() error                                         { return nil }
