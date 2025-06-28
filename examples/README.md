# KeyFlare Examples

This directory contains comprehensive examples demonstrating KeyFlare integration with different cache clients.

## Quick Start

Each example directory includes:

- **Policy examples**: Local cache and key splitting policies
- **Monitoring examples**: Prometheus metrics and REST API
- **Docker setup**: Complete environment for testing
- **Makefile**: Easy commands for cluster management

## Available Examples

### Redis (go-redis)

- **Location**: `redis/`
- **Cache Type**: Redis Cluster (3 nodes)
- **Monitoring Port**: 9122 (local cache), 9123 (key splitting)
- **Features**: Local cache policy, key splitting policy, monitoring dashboard

```bash
cd redis/
make demo  # Start cluster and run example
```

### Memcached

- **Location**: `memcached/`
- **Cache Type**: 3 Memcached instances
- **Monitoring Port**: 9124 (local cache), 9125 (key splitting)
- **Features**: Local cache policy, key splitting policy, monitoring dashboard

```bash
cd memcached/
make demo  # Start instances and run example
```

### Rueidis

- **Location**: `rueidis/`
- **Cache Type**: Redis Cluster (3 nodes)
- **Monitoring Port**: 9126 (local cache), 9127 (key splitting)
- **Features**: Local cache policy, key splitting policy, monitoring dashboard

```bash
cd rueidis/
make demo  # Start cluster and run example
```

## Running Examples

### Prerequisites

- Docker and Docker Compose
- Go 1.21+

### Step 1: Choose an Example

```bash
# Redis example
cd redis/

# Memcached example
cd memcached/

# Rueidis example
cd rueidis/
```

### Step 2: Start Infrastructure

```bash
# Start cache instances/cluster
make cluster-up

# Check status
make cluster-status

# View logs
make cluster-logs
```

### Step 3: Run KeyFlare Examples

```bash
# Run the interactive example
make run-example

# Or run full demo (starts cluster + runs example)
make demo
```

### Step 4: Monitor Results

Access monitoring endpoints:

- **Prometheus Metrics**: `http://localhost:XXXX/metrics`
- **Hot Keys API**: `http://localhost:XXXX/hot-keys`

Port numbers:

- Redis: 9121 (local cache), 9122 (key splitting)
- Memcached: 9124 (local cache), 9125 (key splitting)
- Rueidis: 9126 (local cache), 9127 (key splitting)

### Step 5: Cleanup

```bash
# Stop and clean up
make cluster-down
```

## Example Scenarios

### 1. Local Cache Policy

Demonstrates hot key detection and local caching:

1. Start with whitelisted keys
2. Generate traffic to trigger hot key detection
3. Observe local cache hits reducing backend load
4. Monitor via `/hot-keys` API and Prometheus metrics

### 2. Key Splitting Policy

Shows how hot keys are split across multiple shards:

1. Configure key splitting parameters
2. Generate concentrated traffic on specific keys
3. Watch keys get automatically split into shards
4. Monitor shard distribution and load balancing

### 3. Monitoring

Real-time observability features:

- **Metrics**: Access counts, policy applications, cache hits/misses
- **Hot Keys API**: Top-K hot keys with trend analysis
- **Time Series**: Historical data for pattern analysis

## Customization

### Adjusting Detection Parameters

```go
keyflare.WithDetectorOptions(keyflare.DetectorOptions{
    Capacity:      10000,  // Max keys to track
    TopK:          100,    // Number of top hot keys
    HotThreshold:  1000,   // Threshold for hot key detection
})
```

### Configuring Policies

```go
keyflare.WithPolicyOptions(keyflare.PolicyOptions{
    Type: keyflare.LocalCache,
    WhitelistKeys: []string{
        "your:hot:key",
        "another:key",
    },
})
```

### Monitoring Settings

```go
keyflare.WithMetricsOptions(keyflare.MetricsOptions{
    MetricServerAddress: ":9999",
    HotKeyMetricLimit:   50,
    EnableAPI:           true,
})
```
