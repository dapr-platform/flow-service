/**
 * @module workflow_service
 * @description 统一工作流服务，合并原 DAG 和 Task 服务功能，提供完整的工作流管理
 * @architecture 统一服务层设计，简化三层架构为两层架构的关键组件
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow workflow_states: inactive -> active -> paused -> disabled
 * @rules 工作流状态变更必须遵循状态机规则，调度配置必须经过验证
 * @dependencies service/models/workflow.go, service/database/database.go
 * @refs service/execution_service.go, pkg/scheduler/scheduler.go
 */

package service

import (
	"errors"
	"fmt"
	"time"

	"flow-service/service/models"

	"gorm.io/gorm"
)

// WorkflowService 统一工作流服务
type WorkflowService struct {
	db        *gorm.DB
	scheduler *SimpleScheduler
}

// NewWorkflowService 创建工作流服务实例
func NewWorkflowService(db *gorm.DB, scheduler *SimpleScheduler) *WorkflowService {
	return &WorkflowService{
		db:        db,
		scheduler: scheduler,
	}
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(workflow *models.Workflow) error {
	// 验证工作流
	if err := workflow.Validate(); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// 设置默认值
	if workflow.Status == "" {
		workflow.Status = models.WorkflowStatusInactive
	}

	if workflow.Version == "" {
		workflow.Version = "1.0.0"
	}

	// 初始化统计信息
	if workflow.Statistics == nil {
		workflow.Statistics = &models.WorkflowStatistics{
			TotalExecutions: 0,
			SuccessfulRuns:  0,
			FailedRuns:      0,
			SuccessRate:     0.0,
		}
	}

	// 保存到数据库
	if err := s.db.Create(workflow).Error; err != nil {
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	return nil
}

// GetWorkflow 获取工作流
func (s *WorkflowService) GetWorkflow(id string) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := s.db.Where("id = ?", id).First(&workflow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// 确保状态字段有默认值
	if workflow.Status == "" {
		workflow.Status = models.WorkflowStatusInactive
	}

	return &workflow, nil
}

// GetWorkflowByName 按名称获取工作流
func (s *WorkflowService) GetWorkflowByName(name string) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := s.db.Where("name = ?", name).First(&workflow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return &workflow, nil
}

// UpdateWorkflow 更新工作流
func (s *WorkflowService) UpdateWorkflow(workflow *models.Workflow) error {
	// 先获取现有工作流
	existingWorkflow, err := s.GetWorkflow(workflow.ID)
	if err != nil {
		return err
	}

	// 只更新提供的字段，保留其他字段
	if workflow.Name != "" {
		existingWorkflow.Name = workflow.Name
	}
	if workflow.Description != "" {
		existingWorkflow.Description = workflow.Description
	}
	if workflow.Version != "" {
		existingWorkflow.Version = workflow.Version
	}
	if workflow.Nodes != nil {
		existingWorkflow.Nodes = workflow.Nodes
	}
	if workflow.Edges != nil {
		existingWorkflow.Edges = workflow.Edges
	}
	if workflow.Schedule != nil {
		existingWorkflow.Schedule = workflow.Schedule
	}
	if workflow.Config != nil {
		existingWorkflow.Config = workflow.Config
	}
	if workflow.TagsList != nil {
		existingWorkflow.TagsList = workflow.TagsList
	}
	if workflow.Priority != 0 {
		existingWorkflow.Priority = workflow.Priority
	}

	// 验证更新后的工作流（使用更新验证，不强制要求节点）
	if err := existingWorkflow.ValidateForUpdate(); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	// 更新时间
	existingWorkflow.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(existingWorkflow).Error; err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	// 如果是激活状态且有调度配置，重新调度
	if existingWorkflow.IsActive() && existingWorkflow.CanExecute() {
		if err := s.scheduleWorkflow(existingWorkflow); err != nil {
			return fmt.Errorf("failed to schedule workflow: %w", err)
		}
	}

	// 将更新后的数据复制回传入的工作流对象
	*workflow = *existingWorkflow

	return nil
}

// DeleteWorkflow 删除工作流
func (s *WorkflowService) DeleteWorkflow(id string) error {
	// 先检查工作流是否存在
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 如果工作流处于活跃状态，先停止调度
	if workflow.IsActive() {
		if err := s.DeactivateWorkflow(id); err != nil {
			return fmt.Errorf("failed to deactivate workflow before deletion: %w", err)
		}
	}

	// 使用事务确保数据一致性
	tx := s.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 先删除所有相关的执行记录
	if err := tx.Where("workflow_id = ?", id).Delete(&models.Execution{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete related executions: %w", err)
	}

	// 再删除工作流
	if err := tx.Delete(&models.Workflow{}, "id = ?", id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListWorkflows 列出工作流
func (s *WorkflowService) ListWorkflows(offset, limit int, status models.WorkflowStatus) ([]*models.Workflow, int64, error) {
	var workflows []*models.Workflow
	var total int64

	query := s.db.Model(&models.Workflow{})

	// 按状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count workflows: %w", err)
	}

	// 查询数据
	if err := query.Offset(offset).Limit(limit).Find(&workflows).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list workflows: %w", err)
	}

	return workflows, total, nil
}

// ActivateWorkflow 激活工作流
func (s *WorkflowService) ActivateWorkflow(id string) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateWorkflowTransition(workflow.Status, models.WorkflowStatusActive); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	oldStatus := workflow.Status
	// 更新状态
	workflow.Status = models.WorkflowStatusActive
	workflow.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(workflow).Error; err != nil {
		return fmt.Errorf("failed to activate workflow: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordWorkflowTransition(id, oldStatus, models.WorkflowStatusActive, "manually activated", "system"); err != nil {
		// 记录失败不影响主流程，只记录日志
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	// 开始调度
	if workflow.CanExecute() {
		if err := s.scheduleWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to schedule workflow: %w", err)
		}
	}

	return nil
}

// DeactivateWorkflow 停用工作流
func (s *WorkflowService) DeactivateWorkflow(id string) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateWorkflowTransition(workflow.Status, models.WorkflowStatusInactive); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	// 停止调度
	if err := s.unscheduleWorkflow(workflow); err != nil {
		return fmt.Errorf("failed to unschedule workflow: %w", err)
	}

	oldStatus := workflow.Status
	// 更新状态
	workflow.Status = models.WorkflowStatusInactive
	workflow.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(workflow).Error; err != nil {
		return fmt.Errorf("failed to deactivate workflow: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordWorkflowTransition(id, oldStatus, models.WorkflowStatusInactive, "manually deactivated", "system"); err != nil {
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	return nil
}

// PauseWorkflow 暂停工作流
func (s *WorkflowService) PauseWorkflow(id string) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateWorkflowTransition(workflow.Status, models.WorkflowStatusPaused); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	// 暂停调度
	if err := s.unscheduleWorkflow(workflow); err != nil {
		return fmt.Errorf("failed to unschedule workflow: %w", err)
	}

	oldStatus := workflow.Status
	// 更新状态
	workflow.Status = models.WorkflowStatusPaused
	workflow.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(workflow).Error; err != nil {
		return fmt.Errorf("failed to pause workflow: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordWorkflowTransition(id, oldStatus, models.WorkflowStatusPaused, "manually paused", "system"); err != nil {
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	return nil
}

// ResumeWorkflow 恢复工作流
func (s *WorkflowService) ResumeWorkflow(id string) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 使用状态管理器验证状态转换
	if err := GlobalStateManager.ValidateWorkflowTransition(workflow.Status, models.WorkflowStatusActive); err != nil {
		return fmt.Errorf("invalid state transition: %w", err)
	}

	oldStatus := workflow.Status
	// 更新状态
	workflow.Status = models.WorkflowStatusActive
	workflow.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(workflow).Error; err != nil {
		return fmt.Errorf("failed to resume workflow: %w", err)
	}

	// 记录状态转换
	if err := GlobalStateManager.RecordWorkflowTransition(id, oldStatus, models.WorkflowStatusActive, "manually resumed", "system"); err != nil {
		fmt.Printf("Failed to record state transition: %v\n", err)
	}

	// 重新调度
	if workflow.CanExecute() {
		if err := s.scheduleWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to schedule workflow: %w", err)
		}
	}

	return nil
}

// UpdateWorkflowSchedule 更新工作流调度配置
func (s *WorkflowService) UpdateWorkflowSchedule(id string, schedule *models.WorkflowSchedule) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 验证调度配置
	if err := s.validateSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule configuration: %w", err)
	}

	// 如果工作流处于活跃状态，先停止调度
	if workflow.IsActive() {
		if err := s.unscheduleWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to unschedule workflow: %w", err)
		}
	}

	// 更新调度配置
	workflow.Schedule = schedule
	workflow.UpdatedAt = time.Now()

	// 保存到数据库
	if err := s.db.Save(workflow).Error; err != nil {
		return fmt.Errorf("failed to update workflow schedule: %w", err)
	}

	// 如果工作流处于活跃状态，重新调度
	if workflow.IsActive() && workflow.CanExecute() {
		if err := s.scheduleWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to reschedule workflow: %w", err)
		}
	}

	return nil
}

// UpdateWorkflowStatistics 更新工作流统计信息
func (s *WorkflowService) UpdateWorkflowStatistics(id string, execTime time.Duration, success bool) error {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return err
	}

	// 更新统计信息
	workflow.UpdateStatistics(execTime, success)

	// 保存到数据库
	if err := s.db.Save(workflow).Error; err != nil {
		return fmt.Errorf("failed to update workflow statistics: %w", err)
	}

	return nil
}

// GetWorkflowStatistics 获取工作流统计信息
func (s *WorkflowService) GetWorkflowStatistics(id string) (*models.WorkflowStatistics, error) {
	workflow, err := s.GetWorkflow(id)
	if err != nil {
		return nil, err
	}

	return workflow.Statistics, nil
}

// scheduleWorkflow 调度工作流
func (s *WorkflowService) scheduleWorkflow(workflow *models.Workflow) error {
	if workflow.Schedule == nil || !workflow.Schedule.Enabled {
		return nil
	}

	// 创建任务对象用于调度（临时解决方案，后续需要调整）
	// 注意：这里需要根据实际的调度器接口进行调整
	// 目前暂时跳过实际调度，返回nil
	return nil
}

// unscheduleWorkflow 取消工作流调度
func (s *WorkflowService) unscheduleWorkflow(workflow *models.Workflow) error {
	// 目前暂时跳过实际取消调度，返回nil
	// 后续需要根据调度器接口调整
	return nil
}

// validateSchedule 验证调度配置
func (s *WorkflowService) validateSchedule(schedule *models.WorkflowSchedule) error {
	if schedule == nil {
		return errors.New("schedule cannot be nil")
	}

	switch schedule.Type {
	case models.ScheduleTypeCron:
		if schedule.CronExpression == "" {
			return errors.New("cron_expression is required for cron schedule")
		}
		// 这里可以添加 cron 表达式验证逻辑
	case models.ScheduleTypeInterval:
		if schedule.Interval <= 0 {
			return errors.New("interval must be positive for interval schedule")
		}
	case models.ScheduleTypeOnce:
		if schedule.ExecuteAt == nil {
			return errors.New("execute_at is required for once schedule")
		}
		if schedule.ExecuteAt.Before(time.Now()) {
			return errors.New("execute_at must be in the future")
		}
	case models.ScheduleTypeManual:
		// 手动调度不需要额外验证
	default:
		return fmt.Errorf("unsupported schedule type: %s", schedule.Type)
	}

	if schedule.MaxInstances <= 0 {
		return errors.New("max_instances must be positive")
	}

	if schedule.StartTime != nil && schedule.EndTime != nil {
		if schedule.EndTime.Before(*schedule.StartTime) {
			return errors.New("end_time must be after start_time")
		}
	}

	return nil
}
