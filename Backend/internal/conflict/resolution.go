package conflict

import (
	"fmt"
	"log"
	"time"

	"echofs/internal/metadata"
)

type ConflictResolver struct {
	strategy ResolutionStrategy
}

type ResolutionStrategy interface {
	Resolve(obj1, obj2 *metadata.ObjectMeta) (*metadata.ObjectMeta, error)
	GetStrategyName() string
	GetDescription() string
}

type ConflictResult struct {
	ResolvedObject *metadata.ObjectMeta `json:"resolved_object"`
	Strategy       string               `json:"strategy"`
	Reason         string               `json:"reason"`
	ConflictType   string               `json:"conflict_type"`
	Timestamp      time.Time            `json:"timestamp"`
	RequiresManual bool                 `json:"requires_manual"`
}

func NewConflictResolver(strategy ResolutionStrategy) *ConflictResolver {
	return &ConflictResolver{
		strategy: strategy,
	}
}

func (cr *ConflictResolver) ResolveConflict(obj1, obj2 *metadata.ObjectMeta) (*ConflictResult, error) {
	if obj1 == nil || obj2 == nil {
		return nil, fmt.Errorf("cannot resolve conflict with nil objects")
	}
	
	if !obj1.HasConflictWith(obj2) {

		if obj1.IsNewerThan(obj2) {
			return &ConflictResult{
				ResolvedObject: obj1,
				Strategy:       "no_conflict",
				Reason:         "obj1_is_newer",
				ConflictType:   "none",
				Timestamp:      time.Now(),
				RequiresManual: false,
			}, nil
		}
		return &ConflictResult{
			ResolvedObject: obj2,
			Strategy:       "no_conflict",
			Reason:         "obj2_is_newer",
			ConflictType:   "none",
			Timestamp:      time.Now(),
			RequiresManual: false,
		}, nil
	}
	
	conflictType := cr.determineConflictType(obj1, obj2)
	
	resolved, err := cr.strategy.Resolve(obj1, obj2)
	if err != nil {
		return &ConflictResult{
			ResolvedObject: nil,
			Strategy:       cr.strategy.GetStrategyName(),
			Reason:         fmt.Sprintf("resolution_failed: %v", err),
			ConflictType:   conflictType,
			Timestamp:      time.Now(),
			RequiresManual: true,
		}, err
	}
	
	return &ConflictResult{
		ResolvedObject: resolved,
		Strategy:       cr.strategy.GetStrategyName(),
		Reason:         "automatic_resolution",
		ConflictType:   conflictType,
		Timestamp:      time.Now(),
		RequiresManual: false,
	}, nil
}

func (cr *ConflictResolver) determineConflictType(obj1, obj2 *metadata.ObjectMeta) string {

	if obj1.Size != obj2.Size {
		return "size_conflict"
	}
	
	if len(obj1.Chunks) != len(obj2.Chunks) {
		return "chunk_count_conflict"
	}
	
	if obj1.CurrentMode != obj2.CurrentMode {
		return "consistency_mode_conflict"
	}
	
	if cr.hasVectorClockConflict(obj1, obj2) {
		return "vector_clock_conflict"
	}
	
	return "metadata_conflict"
}

func (cr *ConflictResolver) hasVectorClockConflict(obj1, obj2 *metadata.ObjectMeta) bool {
	if obj1.VectorClock == nil || obj2.VectorClock == nil {
		return true
	}
	
	obj1Dominates := true
	obj2Dominates := true
	
	allNodes := make(map[string]bool)
	for node := range obj1.VectorClock {
		allNodes[node] = true
	}
	for node := range obj2.VectorClock {
		allNodes[node] = true
	}
	
	for node := range allNodes {
		clock1, exists1 := obj1.VectorClock[node]
		clock2, exists2 := obj2.VectorClock[node]
		
		if !exists1 {
			clock1 = 0
		}
		if !exists2 {
			clock2 = 0
		}
		
		if clock1 < clock2 {
			obj1Dominates = false
		}
		if clock2 < clock1 {
			obj2Dominates = false
		}
	}
	
	return !obj1Dominates && !obj2Dominates
}

type LastWriterWinsStrategy struct{}

func (lws *LastWriterWinsStrategy) Resolve(obj1, obj2 *metadata.ObjectMeta) (*metadata.ObjectMeta, error) {

	if obj1.UpdatedAt.After(obj2.UpdatedAt) {
		log.Printf("LWW: Choosing obj1 (updated: %v) over obj2 (updated: %v)", 
			obj1.UpdatedAt, obj2.UpdatedAt)
		return obj1, nil
	}
	
	log.Printf("LWW: Choosing obj2 (updated: %v) over obj1 (updated: %v)", 
		obj2.UpdatedAt, obj1.UpdatedAt)
	return obj2, nil
}

func (lws *LastWriterWinsStrategy) GetStrategyName() string {
	return "last_writer_wins"
}

func (lws *LastWriterWinsStrategy) GetDescription() string {
	return "Resolves conflicts by choosing the object with the latest timestamp. Risk: May lose concurrent updates."
}

type VectorClockMergeStrategy struct{}

func (vcms *VectorClockMergeStrategy) Resolve(obj1, obj2 *metadata.ObjectMeta) (*metadata.ObjectMeta, error) {

	merged := *obj1
	
	if merged.VectorClock == nil {
		merged.VectorClock = make(map[string]int64)
	}
	
	for node, clock := range obj2.VectorClock {
		if existingClock, exists := merged.VectorClock[node]; !exists || clock > existingClock {
			merged.VectorClock[node] = clock
		}
	}
	
	if obj2.Size > merged.Size {
		merged.Size = obj2.Size
	}
	
	mergedChunks := make(map[string]metadata.ChunkRef)
	
	for _, chunk := range obj1.Chunks {
		mergedChunks[chunk.ChunkID] = chunk
	}
	
	for _, chunk := range obj2.Chunks {
		if existing, exists := mergedChunks[chunk.ChunkID]; !exists || chunk.Version > existing.Version {
			mergedChunks[chunk.ChunkID] = chunk
		}
	}
	
	merged.Chunks = make([]metadata.ChunkRef, 0, len(mergedChunks))
	for _, chunk := range mergedChunks {
		merged.Chunks = append(merged.Chunks, chunk)
	}
	
	merged.LastVersion = max(obj1.LastVersion, obj2.LastVersion) + 1
	merged.UpdatedAt = time.Now()
	
	if obj1.CurrentMode == "C" || obj2.CurrentMode == "C" {
		merged.CurrentMode = "C"
	} else if obj1.CurrentMode == "Hybrid" || obj2.CurrentMode == "Hybrid" {
		merged.CurrentMode = "Hybrid"
	} else {
		merged.CurrentMode = "A"
	}
	
	log.Printf("Vector clock merge: combined %d and %d chunks into %d chunks", 
		len(obj1.Chunks), len(obj2.Chunks), len(merged.Chunks))
	
	return &merged, nil
}

func (vcms *VectorClockMergeStrategy) GetStrategyName() string {
	return "vector_clock_merge"
}

func (vcms *VectorClockMergeStrategy) GetDescription() string {
	return "Merges conflicting objects using vector clocks and semantic merge rules. Preserves all updates when possible."
}

type ManualResolutionStrategy struct {
	pendingConflicts map[string]*PendingConflict
}

type PendingConflict struct {
	ID        string                `json:"id"`
	Object1   *metadata.ObjectMeta  `json:"object1"`
	Object2   *metadata.ObjectMeta  `json:"object2"`
	Timestamp time.Time             `json:"timestamp"`
	Priority  string                `json:"priority"`
}

func NewManualResolutionStrategy() *ManualResolutionStrategy {
	return &ManualResolutionStrategy{
		pendingConflicts: make(map[string]*PendingConflict),
	}
}

func (mrs *ManualResolutionStrategy) Resolve(obj1, obj2 *metadata.ObjectMeta) (*metadata.ObjectMeta, error) {

	conflictID := fmt.Sprintf("%s_%d", obj1.FileID, time.Now().UnixNano())
	
	priority := mrs.determinePriority(obj1, obj2)
	
	conflict := &PendingConflict{
		ID:        conflictID,
		Object1:   obj1,
		Object2:   obj2,
		Timestamp: time.Now(),
		Priority:  priority,
	}
	
	mrs.pendingConflicts[conflictID] = conflict
	
	log.Printf("Conflict queued for manual resolution: %s (priority: %s)", conflictID, priority)
	
	return nil, fmt.Errorf("conflict requires manual resolution: %s", conflictID)
}

func (mrs *ManualResolutionStrategy) determinePriority(obj1, obj2 *metadata.ObjectMeta) string {

	if obj1.Size > 100*1024*1024 || obj2.Size > 100*1024*1024 {
		return "high"
	}
	
	if obj1.CurrentMode == "C" || obj2.CurrentMode == "C" {
		return "medium"
	}
	
	return "low"
}

func (mrs *ManualResolutionStrategy) GetPendingConflicts() map[string]*PendingConflict {
	return mrs.pendingConflicts
}

func (mrs *ManualResolutionStrategy) ResolveManually(conflictID string, chosenObject *metadata.ObjectMeta) error {
	conflict, exists := mrs.pendingConflicts[conflictID]
	if !exists {
		return fmt.Errorf("conflict %s not found", conflictID)
	}
	
	if chosenObject.FileID != conflict.Object1.FileID {
		return fmt.Errorf("chosen object does not match conflict")
	}
	
	delete(mrs.pendingConflicts, conflictID)
	
	log.Printf("Manual conflict resolution completed: %s", conflictID)
	return nil
}

func (mrs *ManualResolutionStrategy) GetStrategyName() string {
	return "manual_resolution"
}

func (mrs *ManualResolutionStrategy) GetDescription() string {
	return "Queues conflicts for manual resolution by operators. Safest but requires human intervention."
}

type CRDTStrategy struct{}

func (cs *CRDTStrategy) Resolve(obj1, obj2 *metadata.ObjectMeta) (*metadata.ObjectMeta, error) {

	merged := *obj1
	
	if merged.VectorClock == nil {
		merged.VectorClock = make(map[string]int64)
	}
	
	for node, clock := range obj2.VectorClock {
		if existingClock, exists := merged.VectorClock[node]; !exists || clock > existingClock {
			merged.VectorClock[node] = clock
		}
	}
	
	merged.Size = max(obj1.Size, obj2.Size)
	
	chunkMap := make(map[int]metadata.ChunkRef)
	
	for _, chunk := range obj1.Chunks {
		chunkMap[chunk.Index] = chunk
	}
	
	for _, chunk := range obj2.Chunks {
		if existing, exists := chunkMap[chunk.Index]; !exists || chunk.Version > existing.Version {
			chunkMap[chunk.Index] = chunk
		}
	}
	
	merged.Chunks = make([]metadata.ChunkRef, 0, len(chunkMap))
	for _, chunk := range chunkMap {
		merged.Chunks = append(merged.Chunks, chunk)
	}
	
	merged.LastVersion = max(obj1.LastVersion, obj2.LastVersion) + 1
	merged.UpdatedAt = time.Now()
	
	log.Printf("CRDT merge: resolved conflict for object %s", merged.FileID)
	return &merged, nil
}

func (cs *CRDTStrategy) GetStrategyName() string {
	return "crdt_merge"
}

func (cs *CRDTStrategy) GetDescription() string {
	return "Uses CRDT (Conflict-free Replicated Data Type) semantics for automatic conflict resolution."
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

type ConflictResolutionManager struct {
	strategies map[string]ResolutionStrategy
	defaultStrategy string
}

func NewConflictResolutionManager() *ConflictResolutionManager {
	manager := &ConflictResolutionManager{
		strategies: make(map[string]ResolutionStrategy),
		defaultStrategy: "vector_clock_merge",
	}
	
	manager.RegisterStrategy("last_writer_wins", &LastWriterWinsStrategy{})
	manager.RegisterStrategy("vector_clock_merge", &VectorClockMergeStrategy{})
	manager.RegisterStrategy("manual_resolution", NewManualResolutionStrategy())
	manager.RegisterStrategy("crdt_merge", &CRDTStrategy{})
	
	return manager
}

func (crm *ConflictResolutionManager) RegisterStrategy(name string, strategy ResolutionStrategy) {
	crm.strategies[name] = strategy
	log.Printf("Registered conflict resolution strategy: %s", name)
}

func (crm *ConflictResolutionManager) GetStrategy(name string) ResolutionStrategy {
	if strategy, exists := crm.strategies[name]; exists {
		return strategy
	}
	return crm.strategies[crm.defaultStrategy]
}

func (crm *ConflictResolutionManager) ListStrategies() map[string]string {
	strategies := make(map[string]string)
	for name, strategy := range crm.strategies {
		strategies[name] = strategy.GetDescription()
	}
	return strategies
}

func (crm *ConflictResolutionManager) SetDefaultStrategy(name string) error {
	if _, exists := crm.strategies[name]; !exists {
		return fmt.Errorf("strategy %s not found", name)
	}
	crm.defaultStrategy = name
	log.Printf("Default conflict resolution strategy set to: %s", name)
	return nil
}