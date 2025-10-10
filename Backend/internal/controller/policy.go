package controller

import (
	"math"
	"time"

	"echofs/internal/metadata"
)

type Policy struct {
	// Weights for decision factors
	WeightPartition float64
	WeightLag       float64
	WeightWrite     float64
	WeightHint      float64
	WeightPenalty   float64

	// Thresholds for mode decisions
	ThresholdAvailable float64 // Score above this -> Available mode
	ThresholdStrong    float64 // Score below this -> Strong mode
	// Between thresholds -> Hybrid mode

	// Normalization constants
	MaxLagMs    float64
	MaxWriteRate float64
}

type ObjectMetrics struct {
	PartitionRisk    float64                       // 0.0-1.0, risk of network partition
	ReplicationLag   time.Duration                 // Current replication lag
	WriteRate        float64                       // Writes per second
	NodeRTT          map[string]time.Duration      // RTT to each worker node
	TransitionReason string                        // Human-readable reason for transition
}

func NewPolicy() *Policy {
	return &Policy{
		// Weights (sum should be ~1.0 for interpretability)
		WeightPartition: 0.4,  // Network partition risk is most important
		WeightLag:       0.3,  // Replication lag is critical
		WeightWrite:     0.2,  // Write rate affects consistency requirements
		WeightHint:      0.1,  // User/operator hints
		WeightPenalty:   0.2,  // Penalty for recent mode changes

		// Thresholds
		ThresholdAvailable: 0.6,  // Above 0.6 -> Available mode
		ThresholdStrong:    0.3,  // Below 0.3 -> Strong mode

		// Normalization
		MaxLagMs:     1000.0, // 1 second max lag for normalization
		MaxWriteRate: 100.0,  // 100 writes/sec max for normalization
	}
}

func (p *Policy) DecideMode(meta metadata.ObjectMeta, metrics ObjectMetrics, currentState *ObjectModeState) string {
	score := p.calculateScore(meta, metrics, currentState)
	
	// Apply hysteresis - make it harder to change modes
	if currentState.CurrentMode == "A" {
		// If currently Available, require lower score to go to Strong
		if score < p.ThresholdStrong-0.1 {
			return "C"
		}
	} else if currentState.CurrentMode == "C" {
		// If currently Strong, require higher score to go to Available
		if score > p.ThresholdAvailable+0.1 {
			return "A"
		}
	}

	// Standard thresholds
	if score > p.ThresholdAvailable {
		return "A"
	}
	if score < p.ThresholdStrong {
		return "C"
	}
	return "Hybrid"
}

func (p *Policy) calculateScore(meta metadata.ObjectMeta, metrics ObjectMetrics, currentState *ObjectModeState) float64 {
	score := 0.0

	// Factor 1: Network partition risk (higher = favor Available mode)
	score += p.WeightPartition * metrics.PartitionRisk

	// Factor 2: Replication lag (higher lag = favor Available mode)
	normalizedLag := math.Min(float64(metrics.ReplicationLag.Milliseconds())/p.MaxLagMs, 1.0)
	score += p.WeightLag * normalizedLag

	// Factor 3: Write rate (higher rate = favor Available mode for performance)
	normalizedWriteRate := math.Min(metrics.WriteRate/p.MaxWriteRate, 1.0)
	score += p.WeightWrite * normalizedWriteRate

	// Factor 4: User/operator hint
	hintValue := p.getHintValue(meta.ModeHint)
	score += p.WeightHint * hintValue

	// Factor 5: Penalty for recent mode changes (stability)
	recentChangePenalty := p.calculateRecentChangePenalty(currentState)
	score -= p.WeightPenalty * recentChangePenalty

	// Determine transition reason for logging
	metrics.TransitionReason = p.determineTransitionReason(metrics, hintValue, recentChangePenalty)

	return math.Max(0.0, math.Min(1.0, score)) // Clamp to [0,1]
}

func (p *Policy) getHintValue(hint string) float64 {
	switch hint {
	case "Available":
		return 1.0 // Strongly favor Available mode
	case "Strong":
		return 0.0 // Strongly favor Strong mode
	case "Auto":
		fallthrough
	default:
		return 0.5 // Neutral
	}
}

func (p *Policy) calculateRecentChangePenalty(currentState *ObjectModeState) float64 {
	// Penalize recent mode changes to prevent flapping
	timeSinceChange := time.Since(currentState.LastChange)
	
	// Full penalty for changes within 1 minute, decay over 5 minutes
	if timeSinceChange < time.Minute {
		return 1.0
	} else if timeSinceChange < 5*time.Minute {
		// Linear decay from 1.0 to 0.0 over 4 minutes
		return 1.0 - float64(timeSinceChange-time.Minute)/float64(4*time.Minute)
	}
	
	return 0.0 // No penalty after 5 minutes
}

func (p *Policy) determineTransitionReason(metrics ObjectMetrics, hintValue, penalty float64) string {
	// Determine primary reason for mode transition
	if hintValue == 1.0 {
		return "user_hint_available"
	}
	if hintValue == 0.0 {
		return "user_hint_strong"
	}
	if metrics.PartitionRisk > 0.7 {
		return "high_partition_risk"
	}
	if metrics.ReplicationLag > 500*time.Millisecond {
		return "high_replication_lag"
	}
	if metrics.WriteRate > 50.0 {
		return "high_write_rate"
	}
	if penalty > 0.5 {
		return "stability_penalty"
	}
	
	// Calculate average RTT
	totalRTT := time.Duration(0)
	nodeCount := 0
	for _, rtt := range metrics.NodeRTT {
		totalRTT += rtt
		nodeCount++
	}
	if nodeCount > 0 {
		avgRTT := totalRTT / time.Duration(nodeCount)
		if avgRTT < 10*time.Millisecond {
			return "low_latency"
		}
		if avgRTT > 100*time.Millisecond {
			return "high_latency"
		}
	}
	
	return "policy_evaluation"
}

// PolicyStats returns current policy configuration for debugging
func (p *Policy) PolicyStats() map[string]interface{} {
	return map[string]interface{}{
		"weights": map[string]float64{
			"partition": p.WeightPartition,
			"lag":       p.WeightLag,
			"write":     p.WeightWrite,
			"hint":      p.WeightHint,
			"penalty":   p.WeightPenalty,
		},
		"thresholds": map[string]float64{
			"available": p.ThresholdAvailable,
			"strong":    p.ThresholdStrong,
		},
		"normalization": map[string]float64{
			"max_lag_ms":     p.MaxLagMs,
			"max_write_rate": p.MaxWriteRate,
		},
	}
}