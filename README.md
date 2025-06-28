# KeyFlare

<p align="center">
  <img src="images/logo-min.png" alt="KeyFlare Logo" width="200"/>
</p>

**KeyFlare** is a client-side hot key detection engine designed to identify and mitigate hot key problems in caching systems in real-time.

[![Go Reference](https://pkg.go.dev/badge/github.com/mingrammer/keyflare.svg)](https://pkg.go.dev/github.com/mingrammer/keyflare)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Real-time Hot Key Detection**: Uses Count-Min Sketch and Space-Saving algorithms for efficient hot key identification
- **Memory-Efficient**: Unlike traditional full-tracking approaches, KeyFlare uses probabilistic algorithms that provide excellent accuracy with minimal memory overhead
- **Policy-Based Mitigation**: Automatic application of mitigation strategies (local caching, key splitting) when hot keys are detected
- **Non-Intrusive Integration**: Easy integration with existing cache clients without code changes
- **Comprehensive Monitoring**: Prometheus metrics and REST API for hot key insights
- **Multi-Client Support**: Works with Redis (go-redis), Memcached (gomemcache), and Rueidis clients

## Why KeyFlare?

Traditional hot key detection approaches often track every single key access, leading to significant memory overhead and scalability issues. KeyFlare takes a different approach:

- **Memory Efficiency**: Uses constant memory regardless of the number of unique keys
- **High Accuracy**: Probabilistic algorithms provide >99% accuracy for hot key detection
- **Real-time Processing**: Immediate detection and mitigation without batch processing delays

## Installation

Install the core library:

```bash
go get github.com/mingrammer/keyflare
```

Install client wrappers as needed:

```bash
# For Redis (go-redis)
go get github.com/mingrammer/keyflare/pkg/redis

# For Redis (rueidis)
go get github.com/mingrammer/keyflare/pkg/rueidis

# For Memcached
go get github.com/mingrammer/keyflare/pkg/memcached
```

## Quick Start

### 1. Initialize KeyFlare

```go
import "github.com/mingrammer/keyflare"

// Initialize with local cache policy
err := keyflare.New(
    keyflare.WithPolicyOptions(keyflare.PolicyOptions{
        Type: keyflare.LocalCache,
        Parameters: keyflare.LocalCacheParams{
            TTL:          300,
            Jitter:       0.2,
            Capacity:     1000,
            RefreshAhead: 0.8,
        },
        WhitelistKeys: []string{
            "user:popular",
            "config:global",
        },
    }),
)
if err != nil {
    log.Fatal(err)
}

// Start the detection engine
err = keyflare.Start()
if err != nil {
    log.Fatal(err)
}
defer keyflare.Stop()
defer keyflare.Shutdown()
```

### 2. Wrap Your Cache Client

#### Redis (go-redis) Example

```go
import (
    "github.com/redis/go-redis/v9"
    redisWrapper "github.com/mingrammer/keyflare/pkg/redis"
)

// Create Redis Cluster client (required)
rdb := redis.NewClusterClient(&redis.ClusterOptions{
    Addrs: []string{"localhost:7000", "localhost:7001", "localhost:7002"},
})

// Wrap with KeyFlare
client, err := redisWrapper.Wrap(rdb)
if err != nil {
    log.Fatal(err)
}

// Use exactly like the original client
err = client.Set(ctx, "my-key", "my-value", time.Minute).Err()
val, err := client.Get(ctx, "my-key").Result()
```

#### Redis (rueidis) Example

```go
import (
    "github.com/redis/rueidis"
    rueidisWrapper "github.com/mingrammer/keyflare/pkg/rueidis"
)

// Create Rueidis client
client, err := rueidis.NewClient(rueidis.ClientOption{
    InitAddress: []string{"localhost:6379"},
})
if err != nil {
    log.Fatal(err)
}

// Wrap with KeyFlare
wrappedClient, err := rueidisWrapper.Wrap(client)
if err != nil {
    log.Fatal(err)
}

// Use with command builder pattern
cmd := wrappedClient.B().Get().Key("my-key").Build()
result := wrappedClient.Do(ctx, cmd)
```

#### Memcached Example

```go
import (
    "github.com/bradfitz/gomemcache/memcache"
    memcachedWrapper "github.com/mingrammer/keyflare/pkg/memcached"
)

// Create Memcached client
mc := memcache.New("localhost:11211")

// Wrap with KeyFlare
client, err := memcachedWrapper.Wrap(mc)
if err != nil {
    log.Fatal(err)
}

// Use exactly like the original client
err = client.Set(&memcache.Item{Key: "my-key", Value: []byte("my-value")})
item, err := client.Get("my-key")
```

> **📚 Complete Examples:** For comprehensive integration examples with monitoring and policy demonstrations, see the [examples/](examples/) directory.

## Configuration

### Custom Detector Settings

```go
err := keyflare.New(
    keyflare.WithDetectorOptions(keyflare.DetectorOptions{
        ErrorRate:     0.001,  // Acceptable error rate for probabilistic algorithms
        TopK:          100,    // Number of top hot keys to track
        DecayFactor:   0.98,   // Decay rate for aging data
        DecayInterval: 60,     // Decay interval in seconds
        HotThreshold:  1000,   // Threshold for hot key detection (0 means automatic)
    }),
)
```

### Policy Configuration

Policies are applied via whitelist - only specified keys can be mitigated.

#### Local Cache Policy

```go
err := keyflare.New(
    keyflare.WithPolicyOptions(keyflare.PolicyOptions{
        Type: keyflare.LocalCache,
        Parameters: keyflare.LocalCacheParams{
            TTL:          300,   // Cache TTL in seconds
            Jitter:       0.2,   // TTL randomization factor
            Capacity:     1000,  // Max cached items
            RefreshAhead: 0.8,   // Refresh threshold
        },
        WhitelistKeys: []string{
            "user:popular",
            "config:global",
            "leaderboard:top",
        },
    }),
)
```

#### Key Splitting Policy

```go
err := keyflare.New(
    keyflare.WithPolicyOptions(keyflare.PolicyOptions{
        Type: keyflare.KeySplitting,
        Parameters: keyflare.KeySplittingParams{
            Shards: 10,  // Number of shards to split keys into
        },
        WhitelistKeys: []string{
            "counter:global",
            "analytics:realtime",
        },
    }),
)
```

## Monitoring

### Prometheus Metrics

KeyFlare exposes metrics at `http://localhost:9121/metrics`:

- `keyflare_key_access_total`: Total key access count
- `keyflare_policy_application_total`: Policy application statistics
- `keyflare_hot_keys`: Current hot key counts
- `keyflare_top_k_keys_count`: Number of keys in top-K list

### Hot Keys API

Get real-time hot key information:

```bash
# Get top 20 hot keys
curl "http://localhost:9121/hot-keys?limit=20"

# Get hot keys with time series data
curl "http://localhost:9121/hot-keys?include_timeseries=true&timeseries_points=100"
```

Response format:

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "top_k": 100,
  "total_keys": 45,
  "keys": [
    {
      "key": "user:12345",
      "count": 15420,
      "rank": 1,
      "first_seen": "2025-01-15T09:00:00Z",
      "last_seen": "2025-01-15T10:29:59Z",
      "trend": "rising"
    }
  ]
}
```

## Architecture

KeyFlare employs a sophisticated multi-algorithm approach for efficient hot key detection:

### Core Concepts

- **Count-Min Sketch (CMS)**: Provides accurate frequency estimation with constant memory usage
- **Space-Saving (SS)**: Efficiently maintains top-K frequent items
- **Exponential Decay**: Ages old data to adapt to changing access patterns
- **Policy Engine**: Automatically applies mitigation strategies

### Detection Pipeline

1. **Key Access Tracking**: Every cache operation increments counters in both CMS and SS
2. **Frequency Estimation**: CMS provides accurate count estimates with minimal memory
3. **Top-K Maintenance**: SS algorithm maintains the most frequent keys
4. **Hot Key Identification**: Keys exceeding thresholds or appearing in top-K are marked as hot
5. **Policy Application**: Mitigation policies are automatically applied to hot keys

### Memory Efficiency

Traditional approaches that track every key can consume gigabytes of memory in high-traffic systems. KeyFlare uses:

- **Constant Memory**: Memory usage doesn't grow with the number of unique keys
- **Configurable Precision**: Adjust error rates vs. memory usage based on requirements
- **Efficient Data Structures**: Optimized implementations of probabilistic algorithms

## How It Works

### 1. Detection Phase

When a key is accessed, KeyFlare:

- Updates the Count-Min Sketch with the key
- Adds/updates the key in the Space-Saving structure
- Applies time-based decay to prevent stale data

### 2. Classification Phase

Keys are classified as "hot" if they:

- Exceed a configured count threshold, OR
- Appear in the top-K most frequent keys

### 3. Mitigation Phase

Hot keys trigger automatic mitigation:

- **Local Cache**: Frequently accessed data is cached locally
- **Key Splitting**: Hot keys are split across multiple cache entries

### 4. Monitoring Phase

Real-time insights are provided through:

- Prometheus metrics for alerting and dashboards
- REST API for programmatic access
- Time-series data for trend analysis

## Benefits Over Traditional Approaches

| Aspect               | Traditional Full Tracking        | KeyFlare                           |
| -------------------- | -------------------------------- | ---------------------------------- |
| Memory Usage         | O(n) - grows with unique keys    | O(1) - constant memory             |
| Detection Accuracy   | 100%                             | >99% with configurable precision   |
| Performance Impact   | High - scales with key diversity | Low - constant overhead            |
| Production Readiness | Memory concerns at scale         | Designed for high-scale production |

## License

KeyFlare is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
