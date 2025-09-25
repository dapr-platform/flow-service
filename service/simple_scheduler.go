/**
 * @module simple_scheduler
 * @description 简化调度器，专为2层架构设计，替代复杂的pkg/scheduler
 * @architecture 轻量级调度器设计，专注于基本调度功能
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow scheduler_states: stopped -> running -> stopping -> stopped
 * @rules 调度器状态变更必须遵循状态机规则，支持基本的定时任务调度
 * @dependencies service/models/workflow.go
 * @refs service/workflow_service.go
 */

package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"flow-service/service/models"
)

// SchedulerStatus 调度器状态
type SchedulerStatus int

const (
	SchedulerStatusStopped SchedulerStatus = iota
	SchedulerStatusRunning
	SchedulerStatusStopping
)

// SimpleScheduler 简化调度器
type SimpleScheduler struct {
	status       SchedulerStatus
	tasks        map[string]*ScheduledTask
	ticker       *time.Ticker
	stopChan     chan struct{}
	mu           sync.RWMutex
	wg           sync.WaitGroup
	tickInterval time.Duration
}

// ScheduledTask 调度任务
type ScheduledTask struct {
	ID         string
	WorkflowID string
	Name       string
	Schedule   *models.WorkflowSchedule
	NextRun    time.Time
	LastRun    *time.Time
	RunCount   int64
	Enabled    bool
	mu         sync.RWMutex
}

// NewSimpleScheduler 创建简化调度器
func NewSimpleScheduler() *SimpleScheduler {
	return &SimpleScheduler{
		status:       SchedulerStatusStopped,
		tasks:        make(map[string]*ScheduledTask),
		tickInterval: 30 * time.Second, // 30秒检查一次
	}
}

// Start 启动调度器
func (s *SimpleScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == SchedulerStatusRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.status = SchedulerStatusRunning
	s.stopChan = make(chan struct{})
	s.ticker = time.NewTicker(s.tickInterval)

	s.wg.Add(1)
	go s.schedulingLoop(ctx)

	log.Println("SimpleScheduler started")
	return nil
}

// Stop 停止调度器
func (s *SimpleScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status != SchedulerStatusRunning {
		return fmt.Errorf("scheduler is not running")
	}

	s.status = SchedulerStatusStopping
	close(s.stopChan)

	if s.ticker != nil {
		s.ticker.Stop()
	}

	s.wg.Wait()
	s.status = SchedulerStatusStopped

	log.Println("SimpleScheduler stopped")
	return nil
}

// AddTask 添加调度任务
func (s *SimpleScheduler) AddTask(workflowID string, workflow *models.Workflow) error {
	if workflow.Schedule == nil || !workflow.Schedule.Enabled {
		return fmt.Errorf("workflow schedule is not enabled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task := &ScheduledTask{
		ID:         fmt.Sprintf("task_%s", workflowID),
		WorkflowID: workflowID,
		Name:       workflow.Name,
		Schedule:   workflow.Schedule,
		Enabled:    workflow.Schedule.Enabled,
	}

	if err := s.calculateNextRun(task); err != nil {
		return fmt.Errorf("failed to calculate next run time: %w", err)
	}

	s.tasks[task.ID] = task
	log.Printf("Added scheduled task: %s for workflow: %s", task.ID, workflowID)
	return nil
}

// RemoveTask 移除调度任务
func (s *SimpleScheduler) RemoveTask(workflowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskID := fmt.Sprintf("task_%s", workflowID)
	if _, exists := s.tasks[taskID]; !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	delete(s.tasks, taskID)
	log.Printf("Removed scheduled task: %s", taskID)
	return nil
}

// GetStatus 获取调度器状态
func (s *SimpleScheduler) GetStatus() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// GetTasks 获取所有调度任务
func (s *SimpleScheduler) GetTasks() map[string]*ScheduledTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make(map[string]*ScheduledTask)
	for k, v := range s.tasks {
		tasks[k] = v
	}
	return tasks
}

// schedulingLoop 调度循环
func (s *SimpleScheduler) schedulingLoop(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-s.ticker.C:
			s.checkAndTriggerTasks()
		}
	}
}

// checkAndTriggerTasks 检查并触发任务
func (s *SimpleScheduler) checkAndTriggerTasks() {
	s.mu.RLock()
	tasks := make([]*ScheduledTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		if task.Enabled && time.Now().After(task.NextRun) {
			tasks = append(tasks, task)
		}
	}
	s.mu.RUnlock()

	for _, task := range tasks {
		s.triggerTask(task)
	}
}

// triggerTask 触发任务执行
func (s *SimpleScheduler) triggerTask(task *ScheduledTask) {
	task.mu.Lock()
	defer task.mu.Unlock()

	// 更新执行统计
	now := time.Now()
	task.LastRun = &now
	task.RunCount++

	// 计算下次执行时间
	if err := s.calculateNextRun(task); err != nil {
		log.Printf("Failed to calculate next run time for task %s: %v", task.ID, err)
		return
	}

	// 触发工作流执行
	// 这里通过回调函数或者事件总线来触发工作流执行
	// 目前先记录日志
	log.Printf("Triggering workflow execution: %s (task: %s)", task.WorkflowID, task.ID)

	// TODO: 实际触发工作流执行的逻辑
	// 可以通过全局的WorkflowService来触发执行
	if GlobalWorkflowService != nil {
		go func() {
			// 异步执行，避免阻塞调度器
			// 这里需要创建执行记录并触发执行
			log.Printf("Scheduled trigger for workflow: %s", task.WorkflowID)
			// 实际的触发逻辑将在WorkflowService中实现
		}()
	}
}

// calculateNextRun 计算下次执行时间
func (s *SimpleScheduler) calculateNextRun(task *ScheduledTask) error {
	if task.Schedule == nil {
		return fmt.Errorf("schedule is nil")
	}

	now := time.Now()
	var nextRun time.Time

	switch task.Schedule.Type {
	case models.ScheduleTypeCron:
		// 简化的cron处理，仅支持基本的时间间隔
		if task.Schedule.CronExpression == "" {
			return fmt.Errorf("cron expression is empty")
		}

		// 基本的cron表达式处理（简化版）
		// 这里只是一个简单的示例，实际应该使用更完善的cron解析
		nextRun = now.Add(time.Hour) // 默认1小时后执行

	case models.ScheduleTypeInterval:
		if task.Schedule.Interval <= 0 {
			return fmt.Errorf("interval must be positive")
		}
		nextRun = now.Add(task.Schedule.Interval)

	case models.ScheduleTypeOnce:
		if task.Schedule.ExecuteAt == nil {
			return fmt.Errorf("execute_at is required for once schedule")
		}
		nextRun = *task.Schedule.ExecuteAt

	case models.ScheduleTypeManual:
		// 手动调度不设置下次执行时间
		nextRun = time.Time{}

	default:
		return fmt.Errorf("unsupported schedule type: %s", task.Schedule.Type)
	}

	// 检查时间窗口限制
	if task.Schedule.StartTime != nil && nextRun.Before(*task.Schedule.StartTime) {
		nextRun = *task.Schedule.StartTime
	}

	if task.Schedule.EndTime != nil && nextRun.After(*task.Schedule.EndTime) {
		// 如果超过结束时间，禁用任务
		task.Enabled = false
		return nil
	}

	task.NextRun = nextRun
	return nil
}

// GlobalSimpleScheduler 全局简化调度器实例
var GlobalSimpleScheduler *SimpleScheduler
