// Package detector provides hot key detection functionality
package detector

import (
	"sync"
	"time"

	"github.com/mingrammer/keyflare/internal/algorithm"
)

const (
	DefaultErrorRate     = 0.01
	DefaultTopK          = 100
	DefaultDecayFactor   = 0.98
	DefaultDecayInterval = 60 * time.Second
)

// Config contains configuration options for the detector
type Config struct {
	// ErrorRate is the acceptable error rate for probabilistic algorithms
	ErrorRate float64

	// TopK is the number of top hot keys to track
	TopK int

	// DecayFactor is used to decay old counts over time
	DecayFactor float64

	// DecayInterval is the interval at which decay is applied
	DecayInterval time.Duration

	// HotThreshold is the threshold for determining if a key is hot
	// If it's 0, then the threshold is dynamically determined based on the Top-K keys
	HotThreshold uint64
}

// KeyCount represents a key and its estimated count
type KeyCount struct {
	Key   string
	Count uint64
}

// Detector defines the interface for hot key detection
type Detector interface {
	// Increment increments the count for a key
	Increment(key string, count uint64)

	// GetCount returns the estimated count for a key
	GetCount(key string) uint64

	// TopK returns the top K hot keys
	TopK() []KeyCount

	// IsHot returns true if the key is considered hot
	IsHot(key string) bool

	// Reset resets the detector
	Reset()
}

// hotKeyDetector implements the Detector interface using a combination of
// Count-Min Sketch and Space-Saving algorithms
type hotKeyDetector struct {
	sketch        *algorithm.CountMinSketch
	topK          *algorithm.SpaceSaving
	mu            sync.RWMutex
	config        Config
	lastDecay     time.Time
	decayInterval time.Duration
}

// New creates a new detector with the provided configuration
func New(config Config) Detector {
	if config.ErrorRate <= 0 {
		config.ErrorRate = DefaultErrorRate
	}
	if config.TopK <= 0 {
		config.TopK = DefaultTopK
	}
	if config.DecayFactor <= 0 {
		config.DecayFactor = DefaultDecayFactor
	}
	if config.DecayInterval <= 0 {
		config.DecayInterval = DefaultDecayInterval
	}

	sketch := algorithm.NewCountMinSketch(config.ErrorRate, 0.01) // 99% confidence
	topK := algorithm.NewSpaceSaving(config.TopK)

	return &hotKeyDetector{
		sketch:        sketch,
		topK:          topK,
		mu:            sync.RWMutex{},
		config:        config,
		lastDecay:     time.Now(),
		decayInterval: config.DecayInterval,
	}
}

// Increment increments the count for a key
func (d *hotKeyDetector) Increment(key string, count uint64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if we need to apply decay
	now := time.Now()
	if now.Sub(d.lastDecay) >= d.decayInterval {
		d.sketch.Decay(d.config.DecayFactor)
		d.lastDecay = now
	}

	// Update the sketch and topK
	d.sketch.Add([]byte(key), count)
	d.topK.Add(key, count)
}

// GetCount returns the estimated count for a key
func (d *hotKeyDetector) GetCount(key string) uint64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.sketch.Estimate([]byte(key))
}

// TopK returns the top K hot keys
func (d *hotKeyDetector) TopK() []KeyCount {
	d.mu.RLock()
	defer d.mu.RUnlock()

	items := d.topK.TopK(d.config.TopK)
	result := make([]KeyCount, 0, len(items))

	for _, item := range items {
		accurateCount := d.sketch.Estimate([]byte(item.Key))
		result = append(result, KeyCount{
			Key:   item.Key,
			Count: accurateCount, // CMS count instead of Space-Saving count
		})
	}

	// Sort by accurate count (descending)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Count < result[j].Count {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// IsHot returns true if the key is considered hot
func (d *hotKeyDetector) IsHot(key string) bool {
	count := d.GetCount(key)

	// If a threshold is specified, use it
	if d.config.HotThreshold > 0 {
		return count >= d.config.HotThreshold
	}

	// Otherwise, check if the key is in the top-K
	topK := d.TopK()
	for _, kc := range topK {
		if kc.Key == key {
			return true
		}
	}

	return false
}

// Reset resets the detector
func (d *hotKeyDetector) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.sketch.Reset()
	d.topK = algorithm.NewSpaceSaving(d.config.TopK)
	d.lastDecay = time.Now()
}
