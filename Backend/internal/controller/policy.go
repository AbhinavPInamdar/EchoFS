package controller

import (
	"math"
	"time"

	"echofs/internal/metadata"
)

type Policy struct {

	WeightPartition float64
	WeightLag       float64
	WeightWrite     float64
	WeightHint      float64
	WeightPenalty   float64

	ThresholdAvailable float64
	ThresholdStrong    float64

	MaxLagMs    float64
	MaxWriteRate float64
}

type ObjectMetrics struct {
	PartitionRisk    float64
	ReplicationLag   time.Duration
	WriteRate        float64
	NodeRTT          map[string]time.Duration
	TransitionReason string
}

func NewPolicy() *Policy {
	return &Policy{

		WeightPartition: 0.4,
		WeightLag:       0.3,
		WeightWrite:     0.2,
		WeightHint:      0.1,
		WeightPenalty:   0.2,

		ThresholdAvailable: 0.6,
		ThresholdStrong:    0.3,

		MaxLagMs:     1000.0,
		MaxWriteRate: 100.0,
	}
}

func (p *Policy) DecideMode(meta metadata.ObjectMeta, metrics ObjectMetrics, currentState *ObjectModeState) string {
	score := p.calculateScore(meta, metrics, currentState)
	
	if currentState.CurrentMode == "A" {

		if score < p.ThresholdStrong-0.1 {
			return "C"
		}
	} else if currentState.CurrentMode == "C" {

		if score > p.ThresholdAvailable+0.1 {
			return "A"
		}
	}

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

	score += p.WeightPartition * metrics.PartitionRisk

	normalizedLag := math.Min(float64(metrics.ReplicationLag.Milliseconds())/p.MaxLagMs, 1.0)
	score += p.WeightLag * normalizedLag

	normalizedWriteRate := math.Min(metrics.WriteRate/p.MaxWriteRate, 1.0)
	score += p.WeightWrite * normalizedWriteRate

	hintValue := p.getHintValue(meta.ModeHint)
	score += p.WeightHint * hintValue

	recentChangePenalty := p.calculateRecentChangePenalty(currentState)
	score -= p.WeightPenalty * recentChangePenalty

	metrics.TransitionReason = p.determineTransitionReason(metrics, hintValue, recentChangePenalty)

	return math.Max(0.0, math.Min(1.0, score))
}

func (p *Policy) getHintValue(hint string) float64 {
	switch hint {
	case "Available":
		return 1.0
	case "Strong":
		return 0.0
	case "Auto":
		fallthrough
	default:
		return 0.5
	}
}

func (p *Policy) calculateRecentChangePenalty(currentState *ObjectModeState) float64 {

	timeSinceChange := time.Since(currentState.LastChange)
	
	if timeSinceChange < time.Minute {
		return 1.0
	} else if timeSinceChange < 5*time.Minute {

		return 1.0 - float64(timeSinceChange-time.Minute)/float64(4*time.Minute)
	}
	
	return 0.0
}

func (p *Policy) determineTransitionReason(metrics ObjectMetrics, hintValue, penalty float64) string {

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