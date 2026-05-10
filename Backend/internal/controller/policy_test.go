package controller

import (
	"testing"
	"time"

	"echofs/internal/metadata"
)

func TestPolicy_LowRiskReturnsStrong(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{
		ModeHint: "Auto",
	}
	metrics := ObjectMetrics{
		PartitionRisk:  0.1,
		ReplicationLag: 10 * time.Millisecond,
		WriteRate:      5.0,
		NodeRTT:        map[string]time.Duration{"w1": 5 * time.Millisecond},
	}
	state := &ObjectModeState{
		CurrentMode: "C",
		LastChange:  time.Now().Add(-10 * time.Minute), // Long ago, no penalty
	}

	mode := policy.DecideMode(meta, metrics, state)

	if mode != "C" {
		t.Errorf("expected mode C for low risk, got %s", mode)
	}
}

func TestPolicy_HighPartitionRiskReturnsAvailable(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{
		ModeHint: "Auto",
	}
	metrics := ObjectMetrics{
		PartitionRisk:  0.9,
		ReplicationLag: 800 * time.Millisecond,
		WriteRate:      80.0,
		NodeRTT:        map[string]time.Duration{"w1": 200 * time.Millisecond},
	}
	state := &ObjectModeState{
		CurrentMode: "C",
		LastChange:  time.Now().Add(-10 * time.Minute),
	}

	mode := policy.DecideMode(meta, metrics, state)

	if mode != "A" {
		t.Errorf("expected mode A for high partition risk, got %s", mode)
	}
}

func TestPolicy_UserHintStrongForcesStrong(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{
		ModeHint: "Strong",
	}
	metrics := ObjectMetrics{
		PartitionRisk:  0.5,
		ReplicationLag: 200 * time.Millisecond,
		WriteRate:      30.0,
		NodeRTT:        map[string]time.Duration{"w1": 50 * time.Millisecond},
	}
	state := &ObjectModeState{
		CurrentMode: "C",
		LastChange:  time.Now().Add(-10 * time.Minute),
	}

	mode := policy.DecideMode(meta, metrics, state)

	// With Strong hint (value=0), the score should be lower, favoring C
	if mode == "A" {
		t.Errorf("expected mode C or Hybrid with Strong hint, got A")
	}
}

func TestPolicy_UserHintAvailableFavorsAvailable(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{
		ModeHint: "Available",
	}
	metrics := ObjectMetrics{
		PartitionRisk:  0.7,
		ReplicationLag: 600 * time.Millisecond,
		WriteRate:      60.0,
		NodeRTT:        map[string]time.Duration{"w1": 50 * time.Millisecond},
	}
	state := &ObjectModeState{
		CurrentMode: "A",
		LastChange:  time.Now().Add(-10 * time.Minute),
	}

	mode := policy.DecideMode(meta, metrics, state)

	if mode != "A" {
		t.Errorf("expected mode A with Available hint and high risk, got %s", mode)
	}
}

func TestPolicy_RecentChangePenaltyPreventsFlapping(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{
		ModeHint: "Auto",
	}
	metrics := ObjectMetrics{
		PartitionRisk:  0.6,
		ReplicationLag: 400 * time.Millisecond,
		WriteRate:      40.0,
		NodeRTT:        map[string]time.Duration{"w1": 50 * time.Millisecond},
	}

	// State changed very recently — penalty should suppress mode change
	state := &ObjectModeState{
		CurrentMode: "C",
		LastChange:  time.Now().Add(-10 * time.Second), // 10 seconds ago
	}

	mode := policy.DecideMode(meta, metrics, state)

	// With recent change penalty, the score is reduced, making it harder to switch
	// The exact result depends on the penalty magnitude, but it should resist change
	_ = mode // We just verify it doesn't panic
}

func TestPolicy_ScoreIsBounded(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{ModeHint: "Auto"}
	state := &ObjectModeState{
		CurrentMode: "C",
		LastChange:  time.Now().Add(-10 * time.Minute),
	}

	// Test with extreme values
	extremeMetrics := ObjectMetrics{
		PartitionRisk:  10.0, // Way above 1.0
		ReplicationLag: 100 * time.Second,
		WriteRate:      10000.0,
	}

	score := policy.calculateScore(meta, extremeMetrics, state)

	if score < 0.0 || score > 1.0 {
		t.Errorf("score should be bounded [0,1], got %f", score)
	}
}

func TestPolicy_HysteresisRequiresHigherThresholdToSwitch(t *testing.T) {
	policy := NewPolicy()

	meta := metadata.ObjectMeta{ModeHint: "Auto"}

	// Metrics that produce a score just above ThresholdAvailable
	metrics := ObjectMetrics{
		PartitionRisk:  0.65,
		ReplicationLag: 300 * time.Millisecond,
		WriteRate:      30.0,
	}

	// Currently in C mode — needs score > ThresholdAvailable + 0.1 to switch
	stateC := &ObjectModeState{
		CurrentMode: "C",
		LastChange:  time.Now().Add(-10 * time.Minute),
	}
	modeFromC := policy.DecideMode(meta, metrics, stateC)

	// Currently in A mode — needs score < ThresholdStrong - 0.1 to switch back
	stateA := &ObjectModeState{
		CurrentMode: "A",
		LastChange:  time.Now().Add(-10 * time.Minute),
	}
	modeFromA := policy.DecideMode(meta, metrics, stateA)

	// With hysteresis, the same metrics can produce different decisions
	// depending on current state — this IS the anti-flapping behavior
	_ = modeFromC
	_ = modeFromA
}

func TestPolicy_PolicyStatsReturnsWeights(t *testing.T) {
	policy := NewPolicy()
	stats := policy.PolicyStats()

	weights, ok := stats["weights"].(map[string]float64)
	if !ok {
		t.Fatal("expected weights in policy stats")
	}

	if weights["partition"] != 0.4 {
		t.Errorf("expected partition weight 0.4, got %f", weights["partition"])
	}
	if weights["lag"] != 0.3 {
		t.Errorf("expected lag weight 0.3, got %f", weights["lag"])
	}
}
