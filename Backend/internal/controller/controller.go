package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"echofs/internal/metadata"
	"echofs/internal/metrics"
)

type Controller struct {
	config            Config
	metricsClient     metrics.PrometheusClient
	store             *Store
	policy            *Policy
	mu                sync.RWMutex
	objectModes       map[string]*ObjectModeState
	confirmationCount int
	sampleWindow      time.Duration
	globalOverride    string
	criticalKeys      map[string]bool
	emergencyMode     bool
	modeChangeReasons map[string]string
}

type Config struct {
	MetricsClient      metrics.PrometheusClient
	PollInterval       time.Duration
	SampleWindow       time.Duration
	ConfirmationCount  int
	EmergencyThreshold float64
	CooldownPeriod     time.Duration
}

type ObjectModeState struct {
	ObjectID         string    `json:"object_id"`
	CurrentMode      string    `json:"mode"`
	LastChange       time.Time `json:"last_change"`
	TTL              int       `json:"ttl_seconds"`
	Reason           string    `json:"reason"`
	ConsecutiveVotes int       `json:"-"`
	CooldownUntil    time.Time `json:"-"`
}

type ModeRequest struct {
	ObjectID string `json:"object_id"`
}

type ModeResponse struct {
	Mode      string `json:"mode"`
	TTL       int    `json:"ttl_seconds"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

type HintRequest struct {
	ObjectID string `json:"object_id"`
	Hint     string `json:"hint"`
}

func New(config Config) *Controller {
	return &Controller{
		config:            config,
		metricsClient:     config.MetricsClient,
		store:             NewStore(),
		policy:            NewPolicy(),
		objectModes:       make(map[string]*ObjectModeState),
		confirmationCount: config.ConfirmationCount,
		sampleWindow:      config.SampleWindow,
		criticalKeys:      make(map[string]bool),
		modeChangeReasons: make(map[string]string),
	}
}

func (c *Controller) Start(ctx context.Context) {
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()

	log.Println("Controller started, polling metrics...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Controller stopping...")
			return
		case <-ticker.C:
			c.evaluateAllObjects(ctx)
		}
	}
}

func (c *Controller) evaluateAllObjects(ctx context.Context) {

	objects := c.store.GetAllObjects()
	
	for _, obj := range objects {
		c.evaluateObject(ctx, obj)
	}
}

func (c *Controller) evaluateObject(ctx context.Context, obj *metadata.ObjectMeta) {

	objMetrics, err := c.gatherObjectMetrics(ctx, obj.FileID)
	if err != nil {
		log.Printf("Failed to gather metrics for object %s: %v", obj.FileID, err)
		return
	}

	c.mu.Lock()
	currentState, exists := c.objectModes[obj.FileID]
	if !exists {
		currentState = &ObjectModeState{
			ObjectID:    obj.FileID,
			CurrentMode: "C",
			LastChange:  time.Now(),
			TTL:         30,
			Reason:      "initial",
		}
		c.objectModes[obj.FileID] = currentState
	}
	c.mu.Unlock()

	if time.Now().Before(currentState.CooldownUntil) {
		return
	}

	recommendedMode := c.policy.DecideMode(*obj, objMetrics, currentState)

	if recommendedMode != currentState.CurrentMode {
		currentState.ConsecutiveVotes++
		
		if currentState.ConsecutiveVotes >= 3 {
			c.transitionMode(obj, currentState, recommendedMode, objMetrics)
		}
	} else {
		currentState.ConsecutiveVotes = 0
	}
}

func (c *Controller) transitionMode(obj *metadata.ObjectMeta, state *ObjectModeState, newMode string, metrics ObjectMetrics) {
	oldMode := state.CurrentMode
	
	log.Printf("Transitioning object %s from %s to %s (reason: %s)", 
		obj.FileID, oldMode, newMode, metrics.TransitionReason)

	state.CurrentMode = newMode
	state.LastChange = time.Now()
	state.Reason = metrics.TransitionReason
	state.ConsecutiveVotes = 0
	
	state.CooldownUntil = time.Now().Add(30 * time.Second)

	obj.CurrentMode = newMode
	obj.LastModeChange = time.Now()
	c.store.UpdateObject(obj)

	c.emitModeChangeMetric(obj.FileID, oldMode, newMode, metrics.TransitionReason)
}

func (c *Controller) gatherObjectMetrics(ctx context.Context, objectID string) (ObjectMetrics, error) {

	return ObjectMetrics{
		PartitionRisk:     0.1,
		ReplicationLag:    50 * time.Millisecond,
		WriteRate:         10.0,
		NodeRTT:          map[string]time.Duration{
			"worker1": 5 * time.Millisecond,
			"worker2": 15 * time.Millisecond,
		},
		TransitionReason: "low_latency",
	}, nil
}

func (c *Controller) emitModeChangeMetric(objectID, fromMode, toMode, reason string) {

	log.Printf("METRIC: echofs_object_mode_change_total{object=%s,from=%s,to=%s,reason=%s} +1",
		objectID, fromMode, toMode, reason)
}

func (c *Controller) HandleGetMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	objectID := r.URL.Query().Get("object_id")
	if objectID == "" {
		http.Error(w, "object_id parameter required", http.StatusBadRequest)
		return
	}

	c.mu.RLock()
	state, exists := c.objectModes[objectID]
	c.mu.RUnlock()

	if !exists {

		state = &ObjectModeState{
			ObjectID:    objectID,
			CurrentMode: "C",
			TTL:         30,
			Reason:      "default",
		}
	}

	response := ModeResponse{
		Mode:      state.CurrentMode,
		TTL:       state.TTL,
		Reason:    state.Reason,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (c *Controller) HandleSetHint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req HintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ObjectID == "" || req.Hint == "" {
		http.Error(w, "object_id and hint are required", http.StatusBadRequest)
		return
	}

	validHints := map[string]bool{
		"Auto":      true,
		"Strong":    true,
		"Available": true,
	}
	if !validHints[req.Hint] {
		http.Error(w, "Invalid hint value. Must be Auto, Strong, or Available", http.StatusBadRequest)
		return
	}

	obj := c.store.GetObject(req.ObjectID)
	if obj == nil {
		http.Error(w, "Object not found", http.StatusNotFound)
		return
	}

	obj.ModeHint = req.Hint
	c.store.UpdateObject(obj)

	log.Printf("Updated hint for object %s to %s", req.ObjectID, req.Hint)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"object_id": req.ObjectID,
		"hint":      req.Hint,
	})
}

type RegisterRequest struct {
	ObjectID string `json:"object_id"`
	Name     string `json:"name,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

func (c *Controller) HandleRegisterObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ObjectID == "" {
		http.Error(w, "object_id is required", http.StatusBadRequest)
		return
	}

	// Check if object already exists
	if existing := c.store.GetObject(req.ObjectID); existing != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "already_exists",
			"object_id": req.ObjectID,
		})
		return
	}

	// Create new object metadata
	obj := &metadata.ObjectMeta{
		FileID:         req.ObjectID,
		Name:           req.Name,
		Size:           req.Size,
		CurrentMode:    "C", // Default to strong consistency
		ModeHint:       "Auto",
		LastModeChange: time.Now(),
	}

	c.store.RegisterObject(obj)

	log.Printf("Registered object %s with consistency controller", req.ObjectID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "registered",
		"object_id": req.ObjectID,
		"mode":      obj.CurrentMode,
	})
}

func (c *Controller) evaluateObjectWithConfirmation(ctx context.Context, obj *metadata.ObjectMeta) {

	objMetrics, err := c.gatherObjectMetrics(ctx, obj.FileID)
	if err != nil {
		log.Printf("Failed to gather metrics for object %s: %v", obj.FileID, err)
		return
	}
	
	if objMetrics.PartitionRisk > c.config.EmergencyThreshold {
		c.handleEmergencyCondition(obj, objMetrics)
		return
	}
	
	c.mu.Lock()
	currentState, exists := c.objectModes[obj.FileID]
	if !exists {
		currentState = &ObjectModeState{
			ObjectID:    obj.FileID,
			CurrentMode: c.getDefaultMode(obj.FileID),
			LastChange:  time.Now(),
			TTL:         30,
			Reason:      "initial",
		}
		c.objectModes[obj.FileID] = currentState
	}
	c.mu.Unlock()
	
	if overrideMode := c.getOperatorOverride(obj.FileID); overrideMode != "" {
		if overrideMode != currentState.CurrentMode {
			c.transitionModeWithReason(obj, currentState, overrideMode, "operator_override", objMetrics)
		}
		return
	}
	
	if time.Now().Before(currentState.CooldownUntil) {
		return
	}
	
	recommendedMode := c.policy.DecideMode(*obj, objMetrics, currentState)
	
	if recommendedMode != currentState.CurrentMode {
		currentState.ConsecutiveVotes++
		
		requiredVotes := c.confirmationCount
		if c.isHighRiskTransition(currentState.CurrentMode, recommendedMode) {
			requiredVotes *= 2
		}
		
		if currentState.ConsecutiveVotes >= requiredVotes {
			reason := c.determineTransitionReason(objMetrics, currentState.CurrentMode, recommendedMode)
			c.transitionModeWithReason(obj, currentState, recommendedMode, reason, objMetrics)
		}
	} else {
		currentState.ConsecutiveVotes = 0
	}
}

func (c *Controller) handleEmergencyCondition(obj *metadata.ObjectMeta, metrics ObjectMetrics) {
	log.Printf("Emergency condition detected for object %s: partition_risk=%.2f", 
		obj.FileID, metrics.PartitionRisk)
	
	c.mu.Lock()
	c.emergencyMode = true
	c.mu.Unlock()
	
	currentState := c.objectModes[obj.FileID]
	if currentState == nil {
		currentState = &ObjectModeState{
			ObjectID:    obj.FileID,
			CurrentMode: "C",
			LastChange:  time.Now(),
		}
		c.objectModes[obj.FileID] = currentState
	}
	
	if currentState.CurrentMode != "A" {
		c.transitionModeWithReason(obj, currentState, "A", "emergency_partition", metrics)
	}
	
	c.persistState()
}

func (c *Controller) getDefaultMode(objectID string) string {

	if c.criticalKeys[objectID] {
		return "C"
	}
	
	if c.globalOverride != "" {
		return c.globalOverride
	}
	
	return "C"
}

func (c *Controller) getOperatorOverride(objectID string) string {

	if c.criticalKeys[objectID] {
		return "C"
	}
	
	return c.globalOverride
}

func (c *Controller) isHighRiskTransition(fromMode, toMode string) bool {

	return fromMode == "C" && toMode == "A"
}

func (c *Controller) determineTransitionReason(metrics ObjectMetrics, fromMode, toMode string) string {
	if metrics.PartitionRisk > 0.7 {
		return "high_partition_risk"
	}
	if metrics.ReplicationLag > 500*time.Millisecond {
		return "high_replication_lag"
	}
	if metrics.WriteRate > 50.0 {
		return "high_write_rate"
	}
	if fromMode == "C" && toMode == "A" {
		return "availability_optimization"
	}
	if fromMode == "A" && toMode == "C" {
		return "consistency_optimization"
	}
	return "policy_evaluation"
}

func (c *Controller) transitionModeWithReason(obj *metadata.ObjectMeta, state *ObjectModeState, newMode, reason string, metrics ObjectMetrics) {
	oldMode := state.CurrentMode
	
	log.Printf("Transitioning object %s from %s to %s (reason: %s, partition_risk: %.2f, lag: %v)", 
		obj.FileID, oldMode, newMode, reason, metrics.PartitionRisk, metrics.ReplicationLag)

	state.CurrentMode = newMode
	state.LastChange = time.Now()
	state.Reason = reason
	state.ConsecutiveVotes = 0
	
	cooldownPeriod := c.config.CooldownPeriod
	if c.isHighRiskTransition(oldMode, newMode) {
		cooldownPeriod *= 2
	}
	state.CooldownUntil = time.Now().Add(cooldownPeriod)

	obj.CurrentMode = newMode
	obj.LastModeChange = time.Now()
	c.store.UpdateObject(obj)
	
	c.modeChangeReasons[fmt.Sprintf("%s_%d", obj.FileID, time.Now().Unix())] = reason

	c.emitModeChangeMetric(obj.FileID, oldMode, newMode, reason)
	
	c.persistState()
}

func (c *Controller) SetGlobalOverride(mode string) error {
	validModes := map[string]bool{"C": true, "A": true, "": true}
	if !validModes[mode] {
		return fmt.Errorf("invalid global override mode: %s", mode)
	}
	
	c.mu.Lock()
	c.globalOverride = mode
	c.mu.Unlock()
	
	log.Printf("Global override set to: %s", mode)
	
	return c.persistState()
}

func (c *Controller) AddCriticalKey(objectID string) error {
	c.mu.Lock()
	c.criticalKeys[objectID] = true
	c.mu.Unlock()
	
	log.Printf("Added critical key: %s", objectID)
	
	if obj := c.store.GetObject(objectID); obj != nil {
		if state, exists := c.objectModes[objectID]; exists && state.CurrentMode != "C" {
			metrics := ObjectMetrics{TransitionReason: "critical_key_designation"}
			c.transitionModeWithReason(obj, state, "C", "critical_key_designation", metrics)
		}
	}
	
	return c.persistState()
}

func (c *Controller) RemoveCriticalKey(objectID string) error {
	c.mu.Lock()
	delete(c.criticalKeys, objectID)
	c.mu.Unlock()
	
	log.Printf("Removed critical key: %s", objectID)
	return c.persistState()
}

func (c *Controller) persistState() error {

	return nil
}

func (c *Controller) GetCriticalKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	keys := make([]string, 0, len(c.criticalKeys))
	for key := range c.criticalKeys {
		keys = append(keys, key)
	}
	return keys
}

func (c *Controller) HandleSetGlobalOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Mode string `json:"mode"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := c.SetGlobalOverride(req.Mode); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"mode":   req.Mode,
	})
}

func (c *Controller) HandleCriticalKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		keys := c.GetCriticalKeys()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"critical_keys": keys,
			"count":         len(keys),
		})

	case http.MethodPost:
		var req struct {
			ObjectID string `json:"object_id"`
			Action   string `json:"action"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.ObjectID == "" || req.Action == "" {
			http.Error(w, "object_id and action are required", http.StatusBadRequest)
			return
		}

		var err error
		switch req.Action {
		case "add":
			err = c.AddCriticalKey(req.ObjectID)
		case "remove":
			err = c.RemoveCriticalKey(req.ObjectID)
		default:
			http.Error(w, "Invalid action. Must be 'add' or 'remove'", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "success",
			"object_id": req.ObjectID,
			"action":    req.Action,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (c *Controller) HandleControllerStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	c.mu.RLock()
	status := map[string]interface{}{
		"global_override":    c.globalOverride,
		"critical_keys":      len(c.criticalKeys),
		"emergency_mode":     c.emergencyMode,
		"total_objects":      len(c.objectModes),
		"sample_window":      c.sampleWindow.String(),
		"confirmation_count": c.confirmationCount,
		"policy_stats":       c.policy.PolicyStats(),
	}
	c.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}