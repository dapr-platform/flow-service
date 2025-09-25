/**
 * @module state_manager
 * @description 状态管理器，实现工作流和执行的状态机管理
 * @architecture 状态机模式，统一管理状态转换规则和验证
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow workflow_states: inactive -> active -> paused -> disabled, execution_states: pending -> running -> completed/failed
 * @rules 所有状态变更必须通过状态管理器验证，遵循预定义的状态转换规则
 * @dependencies service/models/workflow.go, service/models/execution.go
 * @refs service/workflow_service.go, service/execution_service.go
 */

package service

import (
	"fmt"
	"sync"
	"time"

	"flow-service/service/models"
)

// StateTransition 状态转换记录
type StateTransition struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"` // workflow, execution
	EntityID   string    `json:"entity_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason"`
	Operator   string    `json:"operator"`
	Timestamp  time.Time `json:"timestamp"`
}

// StateManager 状态管理器
type StateManager struct {
	workflowTransitions  map[models.WorkflowStatus][]models.WorkflowStatus
	executionTransitions map[models.ExecutionStatus][]models.ExecutionStatus
	transitionHistory    []StateTransition
	mu                   sync.RWMutex
}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	sm := &StateManager{
		transitionHistory: make([]StateTransition, 0),
	}

	sm.initializeTransitionRules()
	return sm
}

// initializeTransitionRules 初始化状态转换规则
func (sm *StateManager) initializeTransitionRules() {
	// 工作流状态转换规则
	sm.workflowTransitions = map[models.WorkflowStatus][]models.WorkflowStatus{
		models.WorkflowStatusInactive: {
			models.WorkflowStatusActive,
			models.WorkflowStatusDisabled,
		},
		models.WorkflowStatusActive: {
			models.WorkflowStatusInactive,
			models.WorkflowStatusPaused,
			models.WorkflowStatusDisabled,
		},
		models.WorkflowStatusPaused: {
			models.WorkflowStatusActive,
			models.WorkflowStatusInactive,
			models.WorkflowStatusDisabled,
		},
		models.WorkflowStatusDisabled: {
			models.WorkflowStatusInactive,
		},
	}

	// 执行状态转换规则
	sm.executionTransitions = map[models.ExecutionStatus][]models.ExecutionStatus{
		models.ExecutionStatusPending: {
			models.ExecutionStatusRunning,
			models.ExecutionStatusCancelled,
		},
		models.ExecutionStatusRunning: {
			models.ExecutionStatusCompleted,
			models.ExecutionStatusFailed,
			models.ExecutionStatusCancelled,
		},
		models.ExecutionStatusFailed: {
			models.ExecutionStatusPending, // 允许重试
		},
		models.ExecutionStatusCompleted: {}, // 终态
		models.ExecutionStatusCancelled: {}, // 终态
	}
}

// ValidateWorkflowTransition 验证工作流状态转换
func (sm *StateManager) ValidateWorkflowTransition(from, to models.WorkflowStatus) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowedStates, exists := sm.workflowTransitions[from]
	if !exists {
		return fmt.Errorf("invalid from status: %s", from)
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return nil
		}
	}

	return fmt.Errorf("invalid workflow state transition from %s to %s", from, to)
}

// ValidateExecutionTransition 验证执行状态转换
func (sm *StateManager) ValidateExecutionTransition(from, to models.ExecutionStatus) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowedStates, exists := sm.executionTransitions[from]
	if !exists {
		return fmt.Errorf("invalid from status: %s", from)
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return nil
		}
	}

	return fmt.Errorf("invalid execution state transition from %s to %s", from, to)
}

// RecordWorkflowTransition 记录工作流状态转换
func (sm *StateManager) RecordWorkflowTransition(workflowID string, from, to models.WorkflowStatus, reason, operator string) error {
	// 验证状态转换
	if err := sm.ValidateWorkflowTransition(from, to); err != nil {
		return err
	}

	// 记录转换历史
	transition := StateTransition{
		ID:         fmt.Sprintf("wf_%s_%d", workflowID, time.Now().UnixNano()),
		EntityType: "workflow",
		EntityID:   workflowID,
		FromStatus: string(from),
		ToStatus:   string(to),
		Reason:     reason,
		Operator:   operator,
		Timestamp:  time.Now(),
	}

	sm.mu.Lock()
	sm.transitionHistory = append(sm.transitionHistory, transition)
	sm.mu.Unlock()

	return nil
}

// RecordExecutionTransition 记录执行状态转换
func (sm *StateManager) RecordExecutionTransition(executionID string, from, to models.ExecutionStatus, reason, operator string) error {
	// 验证状态转换
	if err := sm.ValidateExecutionTransition(from, to); err != nil {
		return err
	}

	// 记录转换历史
	transition := StateTransition{
		ID:         fmt.Sprintf("exec_%s_%d", executionID, time.Now().UnixNano()),
		EntityType: "execution",
		EntityID:   executionID,
		FromStatus: string(from),
		ToStatus:   string(to),
		Reason:     reason,
		Operator:   operator,
		Timestamp:  time.Now(),
	}

	sm.mu.Lock()
	sm.transitionHistory = append(sm.transitionHistory, transition)
	sm.mu.Unlock()

	return nil
}

// GetTransitionHistory 获取状态转换历史
func (sm *StateManager) GetTransitionHistory(entityType, entityID string) []StateTransition {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var history []StateTransition
	for _, transition := range sm.transitionHistory {
		if (entityType == "" || transition.EntityType == entityType) &&
			(entityID == "" || transition.EntityID == entityID) {
			history = append(history, transition)
		}
	}

	return history
}

// GetAllowedWorkflowTransitions 获取工作流允许的状态转换
func (sm *StateManager) GetAllowedWorkflowTransitions(currentStatus models.WorkflowStatus) []models.WorkflowStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowed, exists := sm.workflowTransitions[currentStatus]
	if !exists {
		return []models.WorkflowStatus{}
	}

	result := make([]models.WorkflowStatus, len(allowed))
	copy(result, allowed)
	return result
}

// GetAllowedExecutionTransitions 获取执行允许的状态转换
func (sm *StateManager) GetAllowedExecutionTransitions(currentStatus models.ExecutionStatus) []models.ExecutionStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowed, exists := sm.executionTransitions[currentStatus]
	if !exists {
		return []models.ExecutionStatus{}
	}

	result := make([]models.ExecutionStatus, len(allowed))
	copy(result, allowed)
	return result
}

// IsWorkflowFinalState 判断是否为工作流最终状态
func (sm *StateManager) IsWorkflowFinalState(status models.WorkflowStatus) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowed, exists := sm.workflowTransitions[status]
	return exists && len(allowed) == 0
}

// IsExecutionFinalState 判断是否为执行最终状态
func (sm *StateManager) IsExecutionFinalState(status models.ExecutionStatus) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	allowed, exists := sm.executionTransitions[status]
	return exists && len(allowed) == 0
}

// CleanupHistory 清理历史记录
func (sm *StateManager) CleanupHistory(beforeTime time.Time) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var remaining []StateTransition
	cleanedCount := 0

	for _, transition := range sm.transitionHistory {
		if transition.Timestamp.After(beforeTime) {
			remaining = append(remaining, transition)
		} else {
			cleanedCount++
		}
	}

	sm.transitionHistory = remaining
	return cleanedCount
}

// GetStatistics 获取状态统计信息
func (sm *StateManager) GetStatistics() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	workflowStats := make(map[string]int)
	executionStats := make(map[string]int)

	for _, transition := range sm.transitionHistory {
		if transition.EntityType == "workflow" {
			workflowStats[transition.ToStatus]++
		} else if transition.EntityType == "execution" {
			executionStats[transition.ToStatus]++
		}
	}

	return map[string]interface{}{
		"total_transitions":     len(sm.transitionHistory),
		"workflow_transitions":  workflowStats,
		"execution_transitions": executionStats,
	}
}

// 全局状态管理器实例
var GlobalStateManager *StateManager

func init() {
	GlobalStateManager = NewStateManager()
}
