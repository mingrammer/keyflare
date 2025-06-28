package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/mingrammer/keyflare/internal/detector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HotKeyInfo contains detailed information about a hot key (for API responses)
type HotKeyInfo struct {
	Key       string    `json:"key"`
	Count     uint64    `json:"count"`
	Rank      int       `json:"rank"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Trend     string    `json:"trend"` // "rising", "falling", "stable", "new"
}

// HotKeysResponse is the API response for hot keys
type HotKeysResponse struct {
	Timestamp   time.Time        `json:"timestamp"`
	TopK        int              `json:"top_k"`
	TotalKeys   int              `json:"total_keys"`
	Keys        []HotKeyInfo     `json:"keys"`
	QueryLimit  int              `json:"query_limit"`
	ActualLimit int              `json:"actual_limit"`
	TimeSeries  []TimeSeriesData `json:"time_series,omitempty"` // 시계열 데이터 추가
}

// TimeSeriesData represents hot key counts over time
type TimeSeriesData struct {
	Timestamp time.Time          `json:"timestamp"`
	Keys      map[string]uint64  `json:"keys"`     // key -> cumulative count
	Rates     map[string]float64 `json:"rates"`    // key -> count per second
	Interval  float64            `json:"interval"` // seconds between this and previous measurement
}

// KeyMetadata stores metadata about keys
type KeyMetadata struct {
	FirstSeen time.Time
	LastSeen  time.Time
	PrevCount uint64
}

// HotKeySnapshot represents a snapshot of hot keys at a point in time
type HotKeySnapshot struct {
	Timestamp time.Time
	Keys      []detector.KeyCount
	KeyMeta   map[string]KeyMetadata
}

// hotKeyHistory maintains a history of hot key snapshots
type hotKeyHistory struct {
	mu        sync.RWMutex
	snapshots []HotKeySnapshot
	maxSize   int
	keyMeta   map[string]KeyMetadata // 전체 키 메타데이터
}

// newHotKeyHistory creates a new hot key history tracker
func newHotKeyHistory(maxSize int) *hotKeyHistory {
	if maxSize <= 0 {
		maxSize = 30 // default 30 snapshots
	}
	return &hotKeyHistory{
		snapshots: make([]HotKeySnapshot, 0, maxSize),
		maxSize:   maxSize,
		keyMeta:   make(map[string]KeyMetadata),
	}
}

// Add adds a new snapshot to the history
func (h *hotKeyHistory) Add(keys []detector.KeyCount) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// Update key metadata
	currentMeta := make(map[string]KeyMetadata)
	for _, kc := range keys {
		existing, exists := h.keyMeta[kc.Key]
		if !exists {
			// New key
			existing = KeyMetadata{
				FirstSeen: now,
				LastSeen:  now,
				PrevCount: 0,
			}
		} else {
			// Update existing key
			existing.LastSeen = now
			existing.PrevCount = existing.PrevCount // Will be updated after snapshot
		}
		currentMeta[kc.Key] = existing
		h.keyMeta[kc.Key] = existing
	}

	// Create snapshot
	snapshot := HotKeySnapshot{
		Timestamp: now,
		Keys:      keys,
		KeyMeta:   currentMeta,
	}

	// Add to snapshots
	h.snapshots = append(h.snapshots, snapshot)

	// Remove old snapshots if necessary
	if len(h.snapshots) > h.maxSize {
		h.snapshots = h.snapshots[1:]
	}

	// Update previous counts for next iteration
	for _, kc := range keys {
		if meta, exists := h.keyMeta[kc.Key]; exists {
			meta.PrevCount = kc.Count
			h.keyMeta[kc.Key] = meta
		}
	}
}

// GetLatest returns the latest snapshot
func (h *hotKeyHistory) GetLatest() *HotKeySnapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.snapshots) == 0 {
		return nil
	}
	return &h.snapshots[len(h.snapshots)-1]
}

// GetTimeSeries returns time series data for specified keys
func (h *hotKeyHistory) GetTimeSeries(keys []string, maxPoints int) []TimeSeriesData {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.snapshots) == 0 {
		return []TimeSeriesData{}
	}

	// Determine which snapshots to include
	startIdx := 0
	if maxPoints > 0 && len(h.snapshots) > maxPoints {
		startIdx = len(h.snapshots) - maxPoints
	}

	result := make([]TimeSeriesData, 0, len(h.snapshots)-startIdx)

	// Track previous counts for rate calculation
	prevCounts := make(map[string]uint64)
	var prevTimestamp time.Time

	for i := startIdx; i < len(h.snapshots); i++ {
		snapshot := h.snapshots[i]
		keyData := make(map[string]uint64)
		rateData := make(map[string]float64)

		// Calculate time interval
		var interval float64 = 0
		if i > startIdx {
			interval = snapshot.Timestamp.Sub(prevTimestamp).Seconds()
		}

		// Include data for all specified keys
		for _, key := range keys {
			// Find the key in this snapshot
			currentCount := uint64(0)
			for _, kc := range snapshot.Keys {
				if kc.Key == key {
					currentCount = kc.Count
					break
				}
			}
			keyData[key] = currentCount

			// Calculate rate (count per second)
			if i > startIdx && interval > 0 {
				prevCount, exists := prevCounts[key]
				if exists {
					// Calculate delta and rate
					delta := int64(currentCount) - int64(prevCount)
					if delta < 0 {
						// Handle decay case where count decreased
						delta = 0
					}
					rateData[key] = float64(delta) / interval
				} else {
					// First occurrence of this key
					rateData[key] = float64(currentCount) / interval
				}
			} else {
				// First data point, no rate calculation possible
				rateData[key] = 0
			}

			// Update previous count
			prevCounts[key] = currentCount
		}

		result = append(result, TimeSeriesData{
			Timestamp: snapshot.Timestamp,
			Keys:      keyData,
			Rates:     rateData,
			Interval:  interval,
		})

		prevTimestamp = snapshot.Timestamp
	}

	return result
}

// metricServer provides Prometheus metrics and hot key API
type metricServer struct {
	config           Config
	detector         detector.Detector
	registry         *prometheus.Registry
	server           *http.Server
	collectionTicker *time.Ticker
	stopChan         chan struct{}
	wg               sync.WaitGroup
	hotKeyHistory    *hotKeyHistory

	// Prometheus metrics
	keyAccessTotal         *prometheus.CounterVec
	policyApplicationTotal *prometheus.CounterVec
	hotKeys                *prometheus.GaugeVec
	topKKeysCount          prometheus.Gauge
}

// newCollectorServer creates a new metric server
func newMetricServer(config Config) *metricServer {
	registry := prometheus.NewRegistry()

	namespace := config.Namespace
	if namespace == "" {
		namespace = "keyflare"
	}

	// Create essential metrics
	keyAccessTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "key_access_total",
			Help:      "Total number of key accesses",
		},
		[]string{"operation"},
	)

	policyApplicationTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "policy_application_total",
			Help:      "Total number of policy applications",
		},
		[]string{"policy", "success"},
	)

	hotKeys := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "hot_keys",
			Help:      "Currently detected hot keys and their counts",
		},
		[]string{"key"},
	)

	topKKeysCount := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "top_k_keys_count",
			Help:      "Number of keys in the top K list",
		},
	)

	// Register metrics
	registry.MustRegister(keyAccessTotal)
	registry.MustRegister(policyApplicationTotal)
	registry.MustRegister(hotKeys)
	registry.MustRegister(topKKeysCount)

	return &metricServer{
		config:                 config,
		detector:               nil,
		registry:               registry,
		server:                 nil,
		collectionTicker:       nil,
		stopChan:               make(chan struct{}),
		wg:                     sync.WaitGroup{},
		hotKeyHistory:          newHotKeyHistory(config.HotKeyHistorySize),
		keyAccessTotal:         keyAccessTotal,
		policyApplicationTotal: policyApplicationTotal,
		hotKeys:                hotKeys,
		topKKeysCount:          topKKeysCount,
	}
}

// RecordKeyAccess records a key access
func (s *metricServer) RecordKeyAccess(key string) {
	s.keyAccessTotal.WithLabelValues("get").Inc()
}

// RecordPolicyApplication records a policy application
func (s *metricServer) RecordPolicyApplication(policy string, success bool) {
	successStr := "false"
	if success {
		successStr = "true"
	}
	s.policyApplicationTotal.WithLabelValues(policy, successStr).Inc()
}

// UpdateHotKeys updates the hot keys metric and history
func (s *metricServer) UpdateHotKeys(hotKeys []detector.KeyCount) {
	// Update history for API
	s.hotKeyHistory.Add(hotKeys)

	// Reset the hot keys metric
	s.hotKeys.Reset()

	// Only expose limited number of keys as metrics
	limit := s.config.HotKeyMetricLimit
	if limit <= 0 {
		limit = 10 // default
	}

	// Update metrics for top P keys only
	for i, kc := range hotKeys {
		if i >= limit {
			break
		}
		s.hotKeys.WithLabelValues(kc.Key).Set(float64(kc.Count))
	}

	// Update the total count
	s.topKKeysCount.Set(float64(len(hotKeys)))
}

// SetDetector sets the detector for metrics collection
func (s *metricServer) SetDetector(d detector.Detector) {
	s.detector = d
}

// collectMetrics collects metrics from the detector and updates Prometheus metrics
func (s *metricServer) collectMetrics() {
	// Update hot keys
	if s.detector != nil {
		hotKeys := s.detector.TopK()
		s.UpdateHotKeys(hotKeys)
	}
}

// handleHotKeys handles the hot keys API endpoint
func (s *metricServer) handleHotKeys(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 100 // default
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Check if time series data is requested
	includeTimeSeries := r.URL.Query().Get("include_timeseries") == "true"
	timeSeriesPoints := 50 // default number of time series points
	if tsp := r.URL.Query().Get("timeseries_points"); tsp != "" {
		if parsed, err := strconv.Atoi(tsp); err == nil && parsed > 0 {
			timeSeriesPoints = parsed
		}
	}

	// Get latest snapshot
	snapshot := s.hotKeyHistory.GetLatest()
	if snapshot == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HotKeysResponse{
			Timestamp: time.Now(),
			Keys:      []HotKeyInfo{},
		})
		return
	}

	// Convert to HotKeyInfo with enriched data
	hotKeys := make([]HotKeyInfo, 0, len(snapshot.Keys))
	topKeyNames := make([]string, 0, limit) // For time series
	for i, kc := range snapshot.Keys {
		// Apply limit
		if i >= limit {
			break
		}

		info := HotKeyInfo{
			Key:   kc.Key,
			Count: kc.Count,
			Rank:  i + 1,
		}

		// Add metadata
		if meta, ok := snapshot.KeyMeta[kc.Key]; ok {
			info.FirstSeen = meta.FirstSeen
			info.LastSeen = meta.LastSeen

			// Determine trend
			if meta.PrevCount == 0 {
				info.Trend = "new"
			} else if kc.Count > meta.PrevCount {
				info.Trend = "rising"
			} else if kc.Count < meta.PrevCount {
				info.Trend = "falling"
			} else {
				info.Trend = "stable"
			}
		}

		hotKeys = append(hotKeys, info)
		topKeyNames = append(topKeyNames, kc.Key)
	}

	// Create response
	response := HotKeysResponse{
		Timestamp:   snapshot.Timestamp,
		TopK:        len(snapshot.Keys),
		TotalKeys:   len(snapshot.Keys),
		Keys:        hotKeys,
		QueryLimit:  limit,
		ActualLimit: len(hotKeys),
	}

	// Add time series data if requested
	if includeTimeSeries && len(topKeyNames) > 0 {
		// Limit to top 10 keys for performance
		maxKeysForTimeSeries := 10
		if len(topKeyNames) > maxKeysForTimeSeries {
			topKeyNames = topKeyNames[:maxKeysForTimeSeries]
		}
		response.TimeSeries = s.hotKeyHistory.GetTimeSeries(topKeyNames, timeSeriesPoints)
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRoot handles the root endpoint
func (s *metricServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	html := `<html>
		<head><title>KeyFlare Metrics</title></head>
		<body>
		<h1>KeyFlare Metrics</h1>
		<ul>
			<li><a href="/metrics">Prometheus Metrics</a></li>
			<li><a href="/hot-keys">Hot Key Histories</a></li>
		</ul>
		</body>
		</html>`

	w.Write([]byte(html))
}

// Start starts the metric server
func (s *metricServer) Start() error {
	// Create HTTP mux
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleRoot)

	// Prometheus metrics endpoint
	mux.Handle("/metrics",
		promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}),
	)

	// Hot key list endpoint
	mux.HandleFunc("/hot-keys", s.handleHotKeys)

	s.server = &http.Server{
		Addr:    s.config.MetricServerAddress,
		Handler: mux,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting metric server: %v\n", err)
		}
	}()

	// Start metrics collection ticker
	s.collectionTicker = time.NewTicker(s.config.CollectionInterval)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.collectionTicker.C:
				s.collectMetrics()
			case <-s.stopChan:
				return
			}
		}
	}()

	return nil
}

// Stop stops the metric server
func (s *metricServer) Stop() error {
	// Stop collection ticker
	if s.collectionTicker != nil {
		s.collectionTicker.Stop()
	}

	// Signal collection goroutine to stop
	close(s.stopChan)

	// Shutdown HTTP server
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	// Wait for goroutines to finish
	s.wg.Wait()

	return nil
}
