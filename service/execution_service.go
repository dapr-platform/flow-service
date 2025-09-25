/**
 * @module execution_service
 * @description 轻量化执行服务，合并原 Instance 服务功能，提供简化的执行管理
 * @architecture 轻量化执行管理，专注于执行状态和记录管理
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow execution_states: pending -> running -> completed/failed -> archived
 * @rules 执行记录必须关联工作流，状态变更必须可追溯，支持重试和取消
 * @dependencies service/models/execution.go, service/models/workflow.go
 * @refs service/workflow_service.go, pkg/engine/dag_engine.go
 */

package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"flow-service/service/models"

	"gorm.io/gorm"
)

// ExecutionService 执行服务
type ExecutionService struct {
	db              *gorm.DB
	workflowService *WorkflowService
	engine          *WorkflowEngine
}

// NewExecutionService 创建执行服务实例
func NewExecutionService(db *gorm.DB, workflowService *WorkflowService, engine *WorkflowEngine) *ExecutionService {
	return &ExecutionService{
		db:              db,
		workflowService: workflowService,
		engine:          engine,
	}
}

// CreateExecution 创建执行记录
func (s *ExecutionService) CreateExecution(execution *models.Execution) error {
	// 检查工作流是否存在
	workflow, err := s.workflowService.GetWorkflow(execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	// 设置默认值
	if execution.Status == "" {
		execution.Status = models.ExecutionStatusPending
	}

	if execution.WorkflowVer == "" {
		execution.WorkflowVer = workflow.Version
	}

	if execution.MaxRetries == 0 {
		execution.MaxRetries = 3
	}

	if execution.RetryStrategy == "" {
		execution.RetryStrategy = "exponential"
	}

	// 设置默认触发类型
	if execution.TriggerType == "" {
		execution.TriggerType = models.TriggerTypeManual
	}

	// 初始化执行上下文
	if execution.Context == nil {
		execution.Context = &models.ExecutionContext{}
	}

	// 初始化执行指标
	if execution.Metrics == nil {
		execution.Metrics = &models.ExecutionMetrics{}
	}

	// 保存执行配置快照
	if workflow.Config != nil {
		execution.ConfigSnapshot = fmt.Sprintf("workflow_config_%s", workflow.Version)
	}

	// 验证执行记录
	if err := execution.Validate(); err != nil {
		return fmt.Errorf("execution validation failed: %w", err)
	}

	// 保存到数据库
	if err := s.db.Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// GetExecution 获取执行记录
func (s *ExecutionService) GetExecution(id string) (*models.Execution, error) {
	var execution models.Execution
	if err := s.db.Preload("Workflow").Where("id = ?", id).First(&execution).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

// UpdateExecution 更新执行记录
func (s *ExecutionService) UpdateExecution(execution *models.Execution) error {
	// 验证执行记录
	if err := execution.Validate(); err != nil {
		return fmt.Errorf("execution validation failed: %w", err)
	}

	// 更新时间
	execution.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(execution).Error; err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	return nil
}

// DeleteExecution 删除执行记录
func (s *ExecutionService) DeleteExecution(id string) error {
	// 先检查执行记录是否存在
	execution, err := s.GetExecution(id)
	if err != nil {
		return err
	}

	// 如果执行正在运行，先取消
	if execution.IsRunning() {
		if err := s.CancelExecution(id); err != nil {
			return fmt.Errorf("failed to cancel execution before deletion: %w", err)
		}
	}

	// 软删除执行记录
	if err := s.db.Delete(&models.Execution{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	return nil
}

// ListExecutions 列出执行记录
func (s *ExecutionService) ListExecutions(workflowID string, status models.ExecutionStatus, offset, limit int) ([]*models.Execution, int64, error) {
	var executions []*models.Execution
	var total int64

	query := s.db.Model(&models.Execution{}).Preload("Workflow")

	// 按工作流筛选
	if workflowID != "" {
		query = query.Where("workflow_id = ?", workflowID)
	}

	// 按状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	// 查询数据，按创建时间倒序
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&executions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, total, nil
}

// StartExecution 开始执行
func (s *ExecutionService) StartExecution(id string) error {
	execution, err := s.GetExecution(id)
	if err != nil {
		return err
	}

	// 检查状态
	if err := execution.Start(); err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	// 更新到数据库
	if err := s.UpdateExecution(execution); err != nil {
		return err
	}

	// 使用简化引擎执行工作流
	if s.engine != nil {
		workflow, err := s.workflowService.GetWorkflow(execution.WorkflowID)
		if err != nil {
			return fmt.Errorf("failed to get workflow: %w", err)
		}

		if err := s.engine.ExecuteWorkflow(context.Background(), workflow, execution); err != nil {
			return fmt.Errorf("failed to start workflow execution: %w", err)
		}
	}

	return nil
}

// CompleteExecution 完成执行
func (s *ExecutionService) CompleteExecution(id string) error {
	execution, err := s.GetExecution(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateExecutionTransition(execution.Status, models.ExecutionStatusCompleted); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	oldStatus := execution.Status
	// 检查状态并完成
	if err := execution.Complete(); err != nil {
		return fmt.Errorf("failed to complete execution: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordExecutionTransition(id, oldStatus, models.ExecutionStatusCompleted, "execution completed", "system"); err != nil {
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	// 更新统计信息
	if execution.StartedAt != nil {
		execTime := execution.GetDuration()
		if err := s.workflowService.UpdateWorkflowStatistics(execution.WorkflowID, execTime, true); err != nil {
			// 记录错误但不影响主流程
			fmt.Printf("Failed to update workflow statistics: %v\n", err)
		}
	}

	return s.UpdateExecution(execution)
}

// FailExecution 执行失败
func (s *ExecutionService) FailExecution(id string, errorMsg string, errorCode string) error {
	execution, err := s.GetExecution(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateExecutionTransition(execution.Status, models.ExecutionStatusFailed); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	oldStatus := execution.Status
	// 检查状态并失败
	if err := execution.Fail(errorMsg, errorCode); err != nil {
		return fmt.Errorf("failed to fail execution: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordExecutionTransition(id, oldStatus, models.ExecutionStatusFailed, fmt.Sprintf("execution failed: %s", errorMsg), "system"); err != nil {
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	// 更新统计信息
	if execution.StartedAt != nil {
		execTime := execution.GetDuration()
		if err := s.workflowService.UpdateWorkflowStatistics(execution.WorkflowID, execTime, false); err != nil {
			// 记录错误但不影响主流程
			fmt.Printf("Failed to update workflow statistics: %v\n", err)
		}
	}

	return s.UpdateExecution(execution)
}

// CancelExecution 取消执行
func (s *ExecutionService) CancelExecution(id string) error {
	execution, err := s.GetExecution(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateExecutionTransition(execution.Status, models.ExecutionStatusCancelled); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	oldStatus := execution.Status
	// 检查状态并取消
	if err := execution.Cancel(); err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordExecutionTransition(id, oldStatus, models.ExecutionStatusCancelled, "execution cancelled", "system"); err != nil {
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	// 通知简化引擎停止执行
	if s.engine != nil {
		if err := s.engine.CancelExecution(id); err != nil {
			fmt.Printf("Failed to cancel execution in engine: %v\n", err)
		}
	}

	return s.UpdateExecution(execution)
}

// RetryExecution 重试执行
func (s *ExecutionService) RetryExecution(id string) error {
	execution, err := s.GetExecution(id)
	if err != nil {
		return err
	}

	// 检查是否可以重试
	if err := execution.Retry(); err != nil {
		return fmt.Errorf("failed to retry execution: %w", err)
	}

	// 更新到数据库
	if err := s.UpdateExecution(execution); err != nil {
		return err
	}

	// 重新开始执行
	return s.StartExecution(id)
}

// GetExecutionProgress 获取执行进度
func (s *ExecutionService) GetExecutionProgress(id string) (float64, error) {
	execution, err := s.GetExecution(id)
	if err != nil {
		return 0, err
	}

	return execution.GetProgress(), nil
}

// GetExecutionLogs 获取执行日志
func (s *ExecutionService) GetExecutionLogs(id string, nodeID string) ([]string, error) {
	execution, err := s.GetExecution(id)
	if err != nil {
		return nil, err
	}

	// 如果指定了节点ID，返回该节点的日志
	if nodeID != "" {
		for _, node := range execution.Nodes {
			if node.NodeID == nodeID {
				return node.Logs, nil
			}
		}
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	// 返回所有节点的日志
	var allLogs []string
	for _, node := range execution.Nodes {
		allLogs = append(allLogs, node.Logs...)
	}

	return allLogs, nil
}

// UpdateNodeExecution 更新节点执行状态
func (s *ExecutionService) UpdateNodeExecution(executionID string, nodeID string, status models.ExecutionStatus, errorMsg string) error {
	execution, err := s.GetExecution(executionID)
	if err != nil {
		return err
	}

	// 更新节点执行状态
	execution.UpdateNodeExecution(nodeID, status, errorMsg)

	// 更新执行指标
	s.updateExecutionMetrics(execution)

	return s.UpdateExecution(execution)
}

// GetExecutionsByStatus 按状态获取执行记录
func (s *ExecutionService) GetExecutionsByStatus(status models.ExecutionStatus, limit int) ([]*models.Execution, error) {
	var executions []*models.Execution

	query := s.db.Where("status = ?", status).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to get executions by status: %w", err)
	}

	return executions, nil
}

// CleanupExecutions 清理旧的执行记录
func (s *ExecutionService) CleanupExecutions(beforeTime time.Time, keepCount int) (int64, error) {
	// 删除指定时间之前的已完成执行记录
	result := s.db.Where("status IN ? AND created_at < ?",
		[]models.ExecutionStatus{models.ExecutionStatusCompleted, models.ExecutionStatusFailed, models.ExecutionStatusCancelled},
		beforeTime).
		Limit(keepCount).
		Delete(&models.Execution{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup executions: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// updateExecutionMetrics 更新执行指标（私有方法）
func (s *ExecutionService) updateExecutionMetrics(execution *models.Execution) {
	if execution.Metrics == nil {
		execution.Metrics = &models.ExecutionMetrics{}
	}

	var completed, failed, skipped int
	for _, node := range execution.Nodes {
		switch node.Status {
		case models.ExecutionStatusCompleted:
			completed++
		case models.ExecutionStatusFailed:
			failed++
		case models.ExecutionStatusCancelled:
			skipped++
		}
	}

	execution.Metrics.TotalNodes = len(execution.Nodes)
	execution.Metrics.CompletedNodes = completed
	execution.Metrics.FailedNodes = failed
	execution.Metrics.SkippedNodes = skipped

	if execution.StartedAt != nil {
		execution.Metrics.ExecutionTime = execution.GetDuration()
	}
}
