/**
 * @module workflow
 * @description 统一工作流模型，合并 DAG 和 Task 功能，提供完整的流程定义和调度配置
 * @architecture 统一模型设计，简化三层架构为两层架构的核心组件
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow workflow_states: inactive -> active -> paused -> disabled
 * @rules 工作流必须包含至少一个节点，状态转换必须遵循状态机规则
 * @dependencies gorm.io/gorm, time, encoding/json
 * @refs service/models/node.go, service/models/edge.go
 */

package models

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// WorkflowStatus 工作流状态枚举
type WorkflowStatus string

const (
	WorkflowStatusInactive WorkflowStatus = "inactive" // 未激活
	WorkflowStatusActive   WorkflowStatus = "active"   // 已激活
	WorkflowStatusPaused   WorkflowStatus = "paused"   // 已暂停
	WorkflowStatusDisabled WorkflowStatus = "disabled" // 已禁用
)

// IsValid 验证工作流状态是否有效
func (s WorkflowStatus) IsValid() bool {
	switch s {
	case WorkflowStatusInactive, WorkflowStatusActive, WorkflowStatusPaused, WorkflowStatusDisabled:
		return true
	default:
		return false
	}
}

// ScheduleType 调度类型枚举
type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"     // Cron 表达式调度
	ScheduleTypeInterval ScheduleType = "interval" // 间隔调度
	ScheduleTypeOnce     ScheduleType = "once"     // 一次性调度
	ScheduleTypeManual   ScheduleType = "manual"   // 手动调度
)

// WorkflowSchedule 工作流调度配置
type WorkflowSchedule struct {
	// 调度类型
	Type ScheduleType `json:"type" validate:"required"`

	// Cron 调度配置
	CronExpression string `json:"cron_expression,omitempty"`
	Timezone       string `json:"timezone" validate:"required"`

	// 间隔调度配置
	Interval time.Duration `json:"interval,omitempty" swaggertype:"integer"`

	// 一次性调度配置
	ExecuteAt *time.Time `json:"execute_at,omitempty"`

	// 调度控制
	Enabled      bool       `json:"enabled"`
	StartTime    *time.Time `json:"start_time,omitempty"`
	EndTime      *time.Time `json:"end_time,omitempty"`
	MaxInstances int        `json:"max_instances" validate:"min=1,max=10"`

	// 错过执行策略
	MissedRunPolicy string `json:"missed_run_policy" validate:"oneof=skip run_once"`
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	// 执行配置
	Timeout        time.Duration `json:"timeout" swaggertype:"integer"`
	MaxRetries     int           `json:"max_retries" validate:"min=0,max=10"`
	MaxConcurrency int           `json:"max_concurrency" validate:"min=1,max=100"`

	// 变量配置
	Variables map[string]interface{} `json:"variables,omitempty"`

	// 通知配置
	Notifications *NotificationConfig `json:"notifications,omitempty"`

	// 其他配置
	Priority    int    `json:"priority" validate:"min=0,max=10"`
	Description string `json:"description,omitempty"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	OnSuccess bool     `json:"on_success"`
	OnFailure bool     `json:"on_failure"`
	OnRetry   bool     `json:"on_retry"`
	Channels  []string `json:"channels"`
}

// WorkflowStatistics 工作流统计信息
type WorkflowStatistics struct {
	TotalExecutions   int64         `json:"total_executions"`
	SuccessfulRuns    int64         `json:"successful_runs"`
	FailedRuns        int64         `json:"failed_runs"`
	SuccessRate       float64       `json:"success_rate"`
	AverageExecTime   time.Duration `json:"average_exec_time" swaggertype:"integer"`
	LastExecutionTime *time.Time    `json:"last_execution_time,omitempty"`
	NextExecutionTime *time.Time    `json:"next_execution_time,omitempty"`
}

// Workflow 统一工作流模型
type Workflow struct {
	// 基础字段
	ID          string `json:"id" gorm:"primaryKey;size:64"`
	Name        string `json:"name" gorm:"not null;size:255"`
	Type        string `json:"type" gorm:"not null;size:255"`
	Description string `json:"description" gorm:"type:text"`
	Version     string `json:"version" gorm:"not null;size:20"`

	// 流程定义 (原 DAG 功能)
	NodesData string           `json:"-" gorm:"type:text;column:nodes"`
	EdgesData string           `json:"-" gorm:"type:text;column:edges"`
	Nodes     map[string]*Node `json:"nodes,omitempty" gorm:"-"`
	Edges     []*Edge          `json:"edges,omitempty" gorm:"-"`

	// 调度配置 (原 Task 功能)
	ScheduleData string            `json:"-" gorm:"type:text;column:schedule"`
	Schedule     *WorkflowSchedule `json:"schedule" gorm:"-"`

	// 执行配置 (统一配置)
	ConfigData string          `json:"-" gorm:"type:text;column:config"`
	Config     *WorkflowConfig `json:"config" gorm:"-"`

	// 统一状态管理
	Status WorkflowStatus `json:"status" gorm:"default:inactive;size:20"`

	// 统计信息
	StatisticsData string              `json:"-" gorm:"type:text;column:statistics"`
	Statistics     *WorkflowStatistics `json:"statistics" gorm:"-"`

	// 元数据
	Tags     string   `json:"-" gorm:"type:text;column:tags"`
	TagsList []string `json:"tags" gorm:"-"`
	Priority int      `json:"priority" gorm:"default:0"`

	// 创建信息
	CreatedBy string     `json:"created_by" gorm:"size:100"`
	UpdatedBy string     `json:"updated_by" gorm:"size:100"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName 返回表名
func (w *Workflow) TableName() string {
	return "workflows"
}

// BeforeCreate GORM 钩子，创建前执行
func (w *Workflow) BeforeCreate(tx *gorm.DB) error {
	return w.serializeFields()
}

// BeforeSave GORM 钩子，保存前执行
func (w *Workflow) BeforeSave(tx *gorm.DB) error {
	return w.serializeFields()
}

// AfterFind GORM 钩子，查询后执行
func (w *Workflow) AfterFind(tx *gorm.DB) error {
	// 保存原始状态
	originalStatus := w.Status

	// 执行反序列化
	if err := w.deserializeFields(); err != nil {
		return err
	}

	// 确保状态字段不会被重置
	if w.Status == "" && originalStatus != "" {
		w.Status = originalStatus
	}

	// 如果状态仍然为空，设置默认值
	if w.Status == "" {
		w.Status = WorkflowStatusInactive
	}

	return nil
}

// serializeFields 序列化字段
func (w *Workflow) serializeFields() error {
	// 序列化节点
	if w.Nodes != nil {
		data, err := json.Marshal(w.Nodes)
		if err != nil {
			return err
		}
		w.NodesData = string(data)
	}

	// 序列化边
	if w.Edges != nil {
		data, err := json.Marshal(w.Edges)
		if err != nil {
			return err
		}
		w.EdgesData = string(data)
	}

	// 序列化调度配置
	if w.Schedule != nil {
		data, err := json.Marshal(w.Schedule)
		if err != nil {
			return err
		}
		w.ScheduleData = string(data)
	}

	// 序列化配置
	if w.Config != nil {
		data, err := json.Marshal(w.Config)
		if err != nil {
			return err
		}
		w.ConfigData = string(data)
	}

	// 序列化统计信息
	if w.Statistics != nil {
		data, err := json.Marshal(w.Statistics)
		if err != nil {
			return err
		}
		w.StatisticsData = string(data)
	}

	// 序列化标签
	if w.TagsList != nil {
		data, err := json.Marshal(w.TagsList)
		if err != nil {
			return err
		}
		w.Tags = string(data)
	}

	return nil
}

// deserializeFields 反序列化字段
func (w *Workflow) deserializeFields() error {
	// 反序列化节点
	if w.NodesData != "" {
		if err := json.Unmarshal([]byte(w.NodesData), &w.Nodes); err != nil {
			return err
		}
	}

	// 反序列化边
	if w.EdgesData != "" {
		if err := json.Unmarshal([]byte(w.EdgesData), &w.Edges); err != nil {
			return err
		}
	}

	// 反序列化调度配置
	if w.ScheduleData != "" {
		if err := json.Unmarshal([]byte(w.ScheduleData), &w.Schedule); err != nil {
			return err
		}
	}

	// 反序列化配置
	if w.ConfigData != "" {
		if err := json.Unmarshal([]byte(w.ConfigData), &w.Config); err != nil {
			return err
		}
	}

	// 反序列化统计信息
	if w.StatisticsData != "" {
		if err := json.Unmarshal([]byte(w.StatisticsData), &w.Statistics); err != nil {
			return err
		}
	}

	// 反序列化标签
	if w.Tags != "" {
		if err := json.Unmarshal([]byte(w.Tags), &w.TagsList); err != nil {
			return err
		}
	}

	return nil
}

// Validate 验证工作流
func (w *Workflow) Validate() error {
	if w.Name == "" {
		return errors.New("workflow name is required")
	}

	if !w.Status.IsValid() {
		return errors.New("invalid workflow status")
	}

	return nil
}

// ValidateForUpdate 验证工作流更新（不要求节点）
func (w *Workflow) ValidateForUpdate() error {
	if w.Name == "" {
		return errors.New("workflow name is required")
	}

	if w.Version == "" {
		return errors.New("workflow version is required")
	}

	if !w.Status.IsValid() {
		return errors.New("invalid workflow status")
	}

	// 更新时不强制要求节点，允许部分更新
	return nil
}

// IsActive 检查工作流是否处于活跃状态
func (w *Workflow) IsActive() bool {
	return w.Status == WorkflowStatusActive
}

// CanExecute 检查工作流是否可以执行
func (w *Workflow) CanExecute() bool {
	return w.Status == WorkflowStatusActive && w.Schedule != nil && w.Schedule.Enabled
}

// UpdateStatistics 更新统计信息
func (w *Workflow) UpdateStatistics(execTime time.Duration, success bool) {
	if w.Statistics == nil {
		w.Statistics = &WorkflowStatistics{}
	}

	w.Statistics.TotalExecutions++
	if success {
		w.Statistics.SuccessfulRuns++
	} else {
		w.Statistics.FailedRuns++
	}

	// 计算成功率
	if w.Statistics.TotalExecutions > 0 {
		w.Statistics.SuccessRate = float64(w.Statistics.SuccessfulRuns) / float64(w.Statistics.TotalExecutions)
	}

	// 更新平均执行时间
	if w.Statistics.TotalExecutions == 1 {
		w.Statistics.AverageExecTime = execTime
	} else {
		// 使用移动平均算法
		w.Statistics.AverageExecTime = time.Duration(
			(int64(w.Statistics.AverageExecTime)*int64(w.Statistics.TotalExecutions-1) + int64(execTime)) /
				int64(w.Statistics.TotalExecutions),
		)
	}

	now := time.Now()
	w.Statistics.LastExecutionTime = &now
}
