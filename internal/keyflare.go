// Package internal provides the core implementation of KeyFlare
package internal

import (
	"fmt"
	"sync"

	"github.com/mingrammer/keyflare/internal/detector"
	"github.com/mingrammer/keyflare/internal/metrics"
	"github.com/mingrammer/keyflare/internal/policy"
)

var (
	// globalInstance is the singleton instance of KeyFlare
	globalInstance *KeyFlare
	// mu protects the singleton instance
	mu sync.RWMutex
)

// Config contains all configuration options for KeyFlare
type Config struct {
	// DetectorConfig configures the hot key detector
	DetectorConfig detector.Config

	// PolicyConfig configures the policy manager
	PolicyConfig policy.Config

	// MetricsConfig configures the metrics collector
	MetricsConfig metrics.Config

	// EnableMetrics determines whether to enable metrics collection
	EnableMetrics bool
}

// KeyFlare is the core implementation
type KeyFlare struct {
	detector  detector.Detector
	policy    policy.Manager
	metrics   metrics.Collector
	config    Config
	isRunning bool
}

// New creates and returns the global KeyFlare instance
func New(config Config) error {
	mu.Lock()
	defer mu.Unlock()

	if globalInstance != nil {
		return fmt.Errorf("KeyFlare is already initialized")
	}

	// Create detector
	d := detector.New(config.DetectorConfig)

	// Create policy manager
	p, err := policy.New(config.PolicyConfig)
	if err != nil {
		return fmt.Errorf("failed to create policy manager: %w", err)
	}

	// Create metrics collector
	var m metrics.Collector
	if config.EnableMetrics {
		m = metrics.New(config.MetricsConfig)
		// Set detector for metrics collection
		m.SetDetector(d)
	} else {
		m = metrics.NewNoop()
	}

	globalInstance = &KeyFlare{
		detector:  d,
		policy:    p,
		metrics:   m,
		config:    config,
		isRunning: false,
	}

	return nil
}

// Start starts the global KeyFlare instance
func Start() error {
	mu.Lock()
	defer mu.Unlock()

	if globalInstance == nil {
		return fmt.Errorf("KeyFlare is not initialized. Call New() first")
	}

	if globalInstance.isRunning {
		return fmt.Errorf("KeyFlare is already running")
	}

	// Start metrics collector
	if globalInstance.metrics != nil {
		if err := globalInstance.metrics.Start(); err != nil {
			return err
		}
	}

	globalInstance.isRunning = true
	return nil
}

// Stop stops and clears the global KeyFlare instance
func Stop() error {
	mu.Lock()
	defer mu.Unlock()

	if globalInstance == nil {
		return fmt.Errorf("KeyFlare is not initialized")
	}

	if globalInstance.isRunning {
		// Stop metrics collector
		if globalInstance.metrics != nil {
			if err := globalInstance.metrics.Stop(); err != nil {
				return err
			}
		}
		globalInstance.isRunning = false
	}

	globalInstance = nil
	return nil
}


// GetInstance returns the global KeyFlare instance for use by wrapper packages
func GetInstance() (*KeyFlare, error) {
	mu.RLock()
	defer mu.RUnlock()

	if globalInstance == nil {
		return nil, fmt.Errorf("KeyFlare is not initialized. Call New() first")
	}

	if !globalInstance.isRunning {
		return nil, fmt.Errorf("KeyFlare is not running. Call Start() first")
	}

	return globalInstance, nil
}

// Detector returns the hot key detector
func (kf *KeyFlare) Detector() detector.Detector {
	return kf.detector
}

// PolicyManager returns the policy manager
func (kf *KeyFlare) PolicyManager() policy.Manager {
	return kf.policy
}

// Metrics returns the metrics collector
func (kf *KeyFlare) Metrics() metrics.Collector {
	return kf.metrics
}
