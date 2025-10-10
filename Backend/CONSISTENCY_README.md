# EchoFS Adaptive Consistency System

This document describes the adaptive consistency system implemented in EchoFS, which dynamically switches between strong and eventual consistency based on network conditions and workload patterns.

## Architecture Overview

```
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client    │───▶│ Consistency     │───▶│ Replication     │
│             │    │ Controller      │    │ Manager         │
└─────────────┘    └─────────────────┘    └─────────────────┘
                            │                       │
                            ▼                       ▼
                   ┌─────────────────┐    ┌─────────────────┐
                   │ Policy Engine   │    │ Sync/Async      │
                   │ (Decision Logic)│    │ Strategies      │
                   └─────────────────┘    └─────────────────┘
```

## Components

### 1. Consistency Controller (`cmd/consistency-controller/`)

**Purpose**: Central decision-making service that determines optimal consistency mode for each object.

**Key Features**:
- Monitors system metrics (latency, partition risk, write rates)
- Applies policy-based decisions with hysteresis to prevent flapping
- Exposes HTTP API for mode queries and hint setting
- Emits metrics for observability

**API Endpoints**:
```
GET  /v1/mode?object_id=<id>     # Get current consistency mode
POST /v1/hint                    # Set consistency hint
GET  /health                     # Health check
```

### 2. Replication Manager (`internal/replication/`)

**Purpose**: Manages different replication strategies based on consistency requirements.

**Strategies**:
- **Sync Strategy**: Quorum-based writes, waits for majority acknowledgment
- **Async Strategy**: Immediate local write, background replication
- **Worker Pool**: Health monitoring and load balancing across workers

### 3. Extended Metadata (`internal/metadata/model.go`)

**Enhanced Object Metadata**:
```go
type ObjectMeta struct {
    FileID         string
    Name           string
    Size           int64
    Chunks         []ChunkRef
    ModeHint       string               // "Auto", "Strong", "Available"
    CurrentMode    string               // "C", "A", "Hybrid"
    LastVersion    int64
    VectorClock    map[string]int64     // For conflict resolution
    LastSyncTs     time.Time
    LastModeChange time.Time
    // ... other fields
}
```

### 4. Policy Engine (`internal/controller/policy.go`)

**Decision Factors**:
- **Partition Risk** (40% weight): Network partition probability
- **Replication Lag** (30% weight): Current replication delay
- **Write Rate** (20% weight): Operations per second
- **User Hints** (10% weight): Explicit consistency preferences

**Decision Logic**:
```go
score := wPartition * partitionRisk +
         wLag * normalize(replicationLag) +
         wWrite * normalize(writeRate) +
         wHint * hintValue(modeHint) -
         wPenalty * recentSwitchPenalty(lastModeChange)

if score > 0.6 { return "A" }      // Available mode
if score < 0.3 { return "C" }      // Strong consistency
return "Hybrid"                    // Hybrid mode
```

## Usage

### 1. Starting the System

```bash
# Start consistency controller
./consistency-controller --port=8082 --metrics=localhost:9090

# Start master with consistency support
./master --consistency-controller=http://localhost:8082

# Start workers
./worker1 --port=8091
./worker2 --port=8092
./worker3 --port=8093
```

### 2. Client API Usage

**Upload with Consistency Control**:
```bash
# Strong consistency (waits for quorum)
curl -X POST http://localhost:8080/api/v1/files/upload/consistency \
  -F "file=@test.txt" \
  -F "user_id=user123" \
  -F "consistency=strong"

# Available consistency (immediate return)
curl -X POST http://localhost:8080/api/v1/files/upload/consistency \
  -F "file=@test.txt" \
  -F "user_id=user123" \
  -F "consistency=available"

# Auto mode (controller decides)
curl -X POST http://localhost:8080/api/v1/files/upload/consistency \
  -F "file=@test.txt" \
  -F "user_id=user123" \
  -F "consistency=auto" \
  -F "hint=Strong"
```

**Query Current Mode**:
```bash
curl "http://localhost:8082/v1/mode?object_id=file123"
```

**Set Consistency Hint**:
```bash
curl -X POST http://localhost:8082/v1/hint \
  -H "Content-Type: application/json" \
  -d '{"object_id": "file123", "hint": "Available"}'
```

### 3. Response Format

```json
{
  "success": true,
  "message": "File uploaded successfully with sync replication",
  "data": {
    "file_id": "file_1697123456789",
    "chunks": 1,
    "compressed": false,
    "file_size": 1024,
    "metadata": { /* ObjectMeta */ },
    "session_id": "session_1697123456789"
  },
  "consistency": {
    "mode_used": "C",
    "version": 1,
    "replicas": 3,
    "latency": "45.2ms",
    "timestamp": "2023-10-12T10:30:45Z"
  }
}
```

## Consistency Modes

### Strong Consistency ("C")
- **Guarantees**: Read-your-writes, monotonic reads
- **Implementation**: Quorum writes (majority of replicas)
- **Use Cases**: Critical data, financial transactions
- **Trade-offs**: Higher latency, may block on network partitions

### Available Consistency ("A")
- **Guarantees**: Eventual consistency, high availability
- **Implementation**: Async replication, immediate acknowledgment
- **Use Cases**: Social media, content distribution, analytics
- **Trade-offs**: Potential stale reads, conflict resolution needed

### Hybrid Mode
- **Behavior**: Adaptive based on operation type and current conditions
- **Implementation**: Mix of sync/async based on heuristics
- **Use Cases**: General-purpose applications with mixed requirements

## Monitoring and Metrics

### New Prometheus Metrics

```prometheus
# Mode transitions
echofs_object_mode_changes_total{object_id,from_mode,to_mode,reason}

# Replication performance
echofs_replication_latency_seconds{strategy,operation}
echofs_quorum_failures_total{reason}
echofs_async_queue_size

# Node health
echofs_node_health_status{node_id}
echofs_consistency_mode{object_id}
```

### Grafana Dashboard Additions

- **Consistency Mode Distribution**: Pie chart of current modes
- **Mode Transition Timeline**: Time series of mode changes
- **Replication Latency**: Histogram of sync vs async latencies
- **Quorum Success Rate**: Success rate of quorum operations
- **Node Health Matrix**: Health status of all worker nodes

## Configuration

### Controller Configuration
```go
Config{
    MetricsClient: prometheusClient,
    PollInterval:  10 * time.Second,
}
```

### Replication Configuration
```go
ReplicationConfig{
    QuorumSize:        2,              // Required replicas for quorum
    WriteTimeout:      5 * time.Second, // Timeout for sync writes
    ReplicationFactor: 3,              // Total number of replicas
    AsyncQueueSize:    100,            // Background replication queue
    AsyncBatchSize:    10,             // Batch size for async ops
    WorkerNodes:       []string{"worker1:8091", "worker2:8092"},
}
```

### Policy Tuning
```go
Policy{
    WeightPartition: 0.4,  // Network partition importance
    WeightLag:       0.3,  // Replication lag importance
    WeightWrite:     0.2,  // Write rate importance
    WeightHint:      0.1,  // User hint importance
    
    ThresholdAvailable: 0.6,  // Score threshold for Available mode
    ThresholdStrong:    0.3,  // Score threshold for Strong mode
}
```

## Testing

### Integration Tests
```bash
cd Backend/test/integration
go test -v ./...
```

### Load Testing
```bash
# Test consistency under load
./load_test.sh --consistency=auto --concurrent=50 --duration=60s

# Test mode transitions
./mode_transition_test.sh --partition-simulation=true
```

### Network Partition Simulation
```bash
# Using tc (traffic control) to simulate network conditions
sudo tc qdisc add dev eth0 root netem delay 100ms loss 5%
```

## Operational Considerations

### Deployment
1. Deploy consistency controller first
2. Update master nodes with controller endpoint
3. Ensure worker nodes are healthy
4. Monitor metrics for proper operation

### Monitoring
- Watch for excessive mode transitions (flapping)
- Monitor quorum failure rates
- Track replication lag across workers
- Alert on controller unavailability

### Troubleshooting

**Common Issues**:
1. **Controller Unavailable**: Falls back to strong consistency
2. **Quorum Failures**: Check worker health and network connectivity
3. **Mode Flapping**: Adjust policy weights or increase hysteresis
4. **High Async Queue**: Scale workers or increase processing capacity

**Debug Commands**:
```bash
# Check controller health
curl http://localhost:8082/health

# Get replication stats
curl http://localhost:8080/api/v1/replication/stats

# View current object modes
curl "http://localhost:8082/v1/mode?object_id=*"
```

## Future Enhancements

### Planned Features
1. **Machine Learning**: ML-based policy decisions
2. **Geographic Awareness**: Location-based consistency decisions
3. **Application-Level Hints**: Per-operation consistency requirements
4. **Conflict Resolution**: Automated merge strategies for conflicts
5. **Cross-Datacenter**: Multi-region consistency coordination

### Performance Optimizations
1. **Batched Operations**: Batch multiple operations for efficiency
2. **Predictive Scaling**: Anticipate consistency mode changes
3. **Caching**: Cache controller decisions with TTL
4. **Connection Pooling**: Reuse connections to workers

## References

- [CAP Theorem](https://en.wikipedia.org/wiki/CAP_theorem)
- [Eventual Consistency](https://en.wikipedia.org/wiki/Eventual_consistency)
- [Vector Clocks](https://en.wikipedia.org/wiki/Vector_clock)
- [Quorum Systems](https://en.wikipedia.org/wiki/Quorum_(distributed_computing))