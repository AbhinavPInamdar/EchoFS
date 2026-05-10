# ADR-002: Adaptive Consistency Model

## Status
Accepted

## Context
Traditional distributed file systems force a static choice between consistency (CP) and availability (AP). EchoFS's core innovation is dynamically switching between modes based on observed network conditions.

## Decision
Implement a **policy-based consistency orchestrator** that:
1. Reads real-time metrics from Prometheus (partition risk, replication lag, write rate, node RTT)
2. Computes a weighted score to determine optimal mode
3. Uses hysteresis (3 consecutive samples) to prevent mode flapping
4. Supports emergency mode (immediate switch when partition risk > 80%)

## Modes
- **Strong (C)**: Quorum writes — N/2+1 nodes must acknowledge before success
- **Available (A)**: Single-node write with async replication — fast but may serve stale reads
- **Hybrid**: 2-node synchronous write — balance between latency and durability

## Write Path Integration
The consistency mode is queried **per-object** before each write operation:
1. Ingest pipeline calls `consistency.Client.GetMode(objectID)`
2. Client caches mode with TTL (avoids per-request orchestrator calls)
3. `WriteCoordinator` executes the appropriate strategy
4. If orchestrator is unreachable, defaults to Strong (safe fallback)

## Consequences
- Mode transitions are observable via Prometheus metrics
- Stale reads are bounded by the async replication queue drain time
- Emergency mode may cause brief write latency spikes as the system transitions
- Critical keys can be pinned to Strong mode regardless of conditions
