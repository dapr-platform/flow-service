/**
 * @module execution
 * @description 简化执行记录模型，轻量级的执行实例管理，合并原 Instance 功能
 * @architecture 轻量化执行记录，专注于状态管理和执行追踪
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow execution_states: pending -> running -> completed/failed -> archived
 * @rules 执行记录必须关联工作流，状态变更必须可追溯
 * @dependencies gorm.io/gorm, time, encoding/json
 * @refs service/models/workflow.go
 */

package models

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ExecutionStatus 执行状态枚举
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"   // 等待执行
	ExecutionStatusRunning   ExecutionStatus = "running"   // 正在执行
	ExecutionStatusCompleted ExecutionStatus = "completed" // 执行完成
	ExecutionStatusFailed    ExecutionStatus = "failed"    // 执行失败
	ExecutionStatusCancelled ExecutionStatus = "cancelled" // 已取消
	ExecutionStatusTimeout   ExecutionStatus = "timeout"   // 执行超时
	ExecutionStatusArchived  ExecutionStatus = "archived"  // 已归档
)

// IsValid 验证执行状态是否有效
func (s ExecutionStatus) IsValid() bool {
	switch s {
	case ExecutionStatusPending, ExecutionStatusRunning, ExecutionStatusCompleted,
		ExecutionStatusFailed, ExecutionStatusCancelled, ExecutionStatusTimeout, ExecutionStatusArchived:
		return true
	default:
		return false
	}
}

// IsFinished 检查执行是否已结束
func (s ExecutionStatus) IsFinished() bool {
	return s == ExecutionStatusCompleted || s == ExecutionStatusFailed ||
		s == ExecutionStatusCancelled || s == ExecutionStatusTimeout
}

// TriggerType 触发类型枚举
type TriggerType string

const (
	TriggerTypeSchedule TriggerType = "schedule" // 定时触发
	TriggerTypeManual   TriggerType = "manual"   // 手动触发
	TriggerTypeAPI      TriggerType = "api"      // API触发
	TriggerTypeEvent    TriggerType = "event"    // 事件触发
)

// ExecutionContext 执行上下文
type ExecutionContext struct {
	// 执行变量
	Variables map[string]interface{} `json:"variables,omitempty"`

	// 输入参数
	Input map[string]interface{} `json:"input,omitempty"`

	// 输出结果
	Output map[string]interface{} `json:"output,omitempty"`

	// 执行环境
	Environment map[string]string `json:"environment,omitempty"`
}

// ExecutionNodeRecord 执行中节点记录
type ExecutionNodeRecord struct {
	NodeID     string                 `json:"node_id"`
	NodeName   string                 `json:"node_name"`
	Status     ExecutionStatus        `json:"status"`
	StartTime  *time.Time             `json:"start_time,omitempty"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Duration   time.Duration          `json:"duration" swaggertype:"integer"`
	RetryCount int                    `json:"retry_count"`
	Input      map[string]interface{} `json:"input,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"`
	ErrorMsg   string                 `json:"error_msg,omitempty"`
	Logs       []string               `json:"logs,omitempty"`
}

// ExecutionMetrics 执行指标
type ExecutionMetrics struct {
	TotalNodes     int           `json:"total_nodes"`
	CompletedNodes int           `json:"completed_nodes"`
	FailedNodes    int           `json:"failed_nodes"`
	SkippedNodes   int           `json:"skipped_nodes"`
	ExecutionTime  time.Duration `json:"execution_time" swaggertype:"integer"`
	QueueTime      time.Duration `json:"queue_time" swaggertype:"integer"`
	WaitTime       time.Duration `json:"wait_time" swaggertype:"integer"`
}

// Execution 轻量化执行记录模型
type Execution struct {
	// 基础字段
	ID          string `json:"id" gorm:"primaryKey;size:64"`
	WorkflowID  string `json:"workflow_id" gorm:"not null;size:64;index"`
	WorkflowVer string `json:"workflow_version" gorm:"not null;size:20"`

	// 执行信息
	Name        string          `json:"name" gorm:"size:255"`
	Description string          `json:"description" gorm:"type:text"`
	Status      ExecutionStatus `json:"status" gorm:"default:pending;size:20;index"`

	// 触发信息
	TriggerType TriggerType            `json:"trigger_type" gorm:"size:20"`
	TriggerBy   string                 `json:"trigger_by" gorm:"size:100"`
	TriggerData string                 `json:"-" gorm:"type:text;column:trigger_data"`
	Trigger     map[string]interface{} `json:"trigger,omitempty" gorm:"-"`

	// 执行上下文
	ContextData string            `json:"-" gorm:"type:text;column:context"`
	Context     *ExecutionContext `json:"context,omitempty" gorm:"-"`

	// 节点执行记录
	NodesData string                 `json:"-" gorm:"type:text;column:nodes"`
	Nodes     []*ExecutionNodeRecord `json:"nodes,omitempty" gorm:"-"`

	// 执行指标
	MetricsData string            `json:"-" gorm:"type:text;column:metrics"`
	Metrics     *ExecutionMetrics `json:"metrics,omitempty" gorm:"-"`

	// 时间信息
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// 重试信息
	RetryCount    int    `json:"retry_count" gorm:"default:0"`
	MaxRetries    int    `json:"max_retries" gorm:"default:3"`
	RetryStrategy string `json:"retry_strategy" gorm:"size:50;default:exponential"`

	// 错误信息
	ErrorMsg   string `json:"error_msg,omitempty" gorm:"type:text"`
	ErrorCode  string `json:"error_code,omitempty" gorm:"size:50"`
	StackTrace string `json:"stack_trace,omitempty" gorm:"type:text"`

	// 执行配置快照
	ConfigSnapshot string `json:"-" gorm:"type:text;column:config_snapshot"`

	// 优先级和标签
	Priority int      `json:"priority" gorm:"default:0"`
	Tags     []string `json:"tags" gorm:"-"`
	TagsData string   `json:"-" gorm:"type:text;column:tags"`

	// 创建信息
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`

	// 关联关系
	Workflow *Workflow `json:"workflow,omitempty" gorm:"foreignKey:WorkflowID"`
}

// TableName 返回表名
func (e *Execution) TableName() string {
	return "executions"
}

// BeforeCreate GORM 钩子，创建前执行
func (e *Execution) BeforeCreate(tx *gorm.DB) error {
	return e.serializeFields()
}

// BeforeSave GORM 钩子，保存前执行
func (e *Execution) BeforeSave(tx *gorm.DB) error {
	return e.serializeFields()
}

// AfterFind GORM 钩子，查询后执行
func (e *Execution) AfterFind(tx *gorm.DB) error {
	// 保存原始状态
	originalStatus := e.Status

	// 执行反序列化
	if err := e.deserializeFields(); err != nil {
		return err
	}

	// 确保状态字段不会被重置
	if e.Status == "" && originalStatus != "" {
		e.Status = originalStatus
	}

	// 如果状态仍然为空，设置默认值
	if e.Status == "" {
		e.Status = ExecutionStatusPending
	}

	return nil
}

// serializeFields 序列化字段
func (e *Execution) serializeFields() error {
	// 序列化触发数据
	if e.Trigger != nil {
		data, err := json.Marshal(e.Trigger)
		if err != nil {
			return err
		}
		e.TriggerData = string(data)
	}

	// 序列化执行上下文
	if e.Context != nil {
		data, err := json.Marshal(e.Context)
		if err != nil {
			return err
		}
		e.ContextData = string(data)
	}

	// 序列化节点执行记录
	if e.Nodes != nil {
		data, err := json.Marshal(e.Nodes)
		if err != nil {
			return err
		}
		e.NodesData = string(data)
	}

	// 序列化指标数据
	if e.Metrics != nil {
		data, err := json.Marshal(e.Metrics)
		if err != nil {
			return err
		}
		e.MetricsData = string(data)
	}

	// 序列化标签
	if e.Tags != nil {
		data, err := json.Marshal(e.Tags)
		if err != nil {
			return err
		}
		e.TagsData = string(data)
	}

	return nil
}

// deserializeFields 反序列化字段
func (e *Execution) deserializeFields() error {
	// 反序列化触发数据
	if e.TriggerData != "" {
		if err := json.Unmarshal([]byte(e.TriggerData), &e.Trigger); err != nil {
			return err
		}
	}

	// 反序列化执行上下文
	if e.ContextData != "" {
		if err := json.Unmarshal([]byte(e.ContextData), &e.Context); err != nil {
			return err
		}
	}

	// 反序列化节点执行记录
	if e.NodesData != "" {
		if err := json.Unmarshal([]byte(e.NodesData), &e.Nodes); err != nil {
			return err
		}
	}

	// 反序列化指标数据
	if e.MetricsData != "" {
		if err := json.Unmarshal([]byte(e.MetricsData), &e.Metrics); err != nil {
			return err
		}
	}

	// 反序列化标签
	if e.TagsData != "" {
		if err := json.Unmarshal([]byte(e.TagsData), &e.Tags); err != nil {
			return err
		}
	}

	return nil
}

// Validate 验证执行记录
func (e *Execution) Validate() error {
	if e.WorkflowID == "" {
		return errors.New("workflow_id is required")
	}

	if e.WorkflowVer == "" {
		return errors.New("workflow_version is required")
	}

	if !e.Status.IsValid() {
		return errors.New("invalid execution status")
	}

	if e.RetryCount < 0 {
		return errors.New("retry_count cannot be negative")
	}

	if e.MaxRetries < 0 {
		return errors.New("max_retries cannot be negative")
	}

	return nil
}

// IsRunning 检查执行是否正在运行
func (e *Execution) IsRunning() bool {
	return e.Status == ExecutionStatusRunning
}

// IsFinished 检查执行是否已结束
func (e *Execution) IsFinished() bool {
	return e.Status.IsFinished()
}

// IsSuccessful 检查执行是否成功
func (e *Execution) IsSuccessful() bool {
	return e.Status == ExecutionStatusCompleted
}

// CanRetry 检查是否可以重试
func (e *Execution) CanRetry() bool {
	return e.Status == ExecutionStatusFailed && e.RetryCount < e.MaxRetries
}

// Start 开始执行
func (e *Execution) Start() error {
	if e.Status != ExecutionStatusPending {
		return errors.New("execution is not in pending status")
	}

	e.Status = ExecutionStatusRunning
	now := time.Now()
	e.StartedAt = &now

	// 初始化指标
	if e.Metrics == nil {
		e.Metrics = &ExecutionMetrics{}
	}

	return nil
}

// Complete 完成执行
func (e *Execution) Complete() error {
	if e.Status != ExecutionStatusRunning {
		return errors.New("execution is not running")
	}

	e.Status = ExecutionStatusCompleted
	now := time.Now()
	e.CompletedAt = &now

	// 更新执行时间
	if e.StartedAt != nil {
		e.Metrics.ExecutionTime = now.Sub(*e.StartedAt)
	}

	return nil
}

// Fail 执行失败
func (e *Execution) Fail(errorMsg string, errorCode string) error {
	if e.Status != ExecutionStatusRunning {
		return errors.New("execution is not running")
	}

	e.Status = ExecutionStatusFailed
	e.ErrorMsg = errorMsg
	e.ErrorCode = errorCode
	now := time.Now()
	e.CompletedAt = &now

	// 更新执行时间
	if e.StartedAt != nil {
		e.Metrics.ExecutionTime = now.Sub(*e.StartedAt)
	}

	return nil
}

// Cancel 取消执行
func (e *Execution) Cancel() error {
	if e.IsFinished() {
		return errors.New("execution is already finished")
	}

	e.Status = ExecutionStatusCancelled
	now := time.Now()
	e.CompletedAt = &now

	return nil
}

// Retry 重试执行
func (e *Execution) Retry() error {
	if !e.CanRetry() {
		return errors.New("execution cannot be retried")
	}

	e.RetryCount++
	e.Status = ExecutionStatusPending
	e.StartedAt = nil
	e.CompletedAt = nil
	e.ErrorMsg = ""
	e.ErrorCode = ""
	e.StackTrace = ""

	return nil
}

// UpdateNodeExecution 更新节点执行状态
func (e *Execution) UpdateNodeExecution(nodeID string, status ExecutionStatus, errorMsg string) {
	if e.Nodes == nil {
		e.Nodes = make([]*ExecutionNodeRecord, 0)
	}

	// 查找现有节点执行记录
	for _, node := range e.Nodes {
		if node.NodeID == nodeID {
			node.Status = status
			if errorMsg != "" {
				node.ErrorMsg = errorMsg
			}
			if status.IsFinished() && node.StartTime != nil {
				now := time.Now()
				node.EndTime = &now
				node.Duration = now.Sub(*node.StartTime)
			}
			return
		}
	}

	// 创建新的节点执行记录
	nodeExec := &ExecutionNodeRecord{
		NodeID:     nodeID,
		Status:     status,
		RetryCount: 0,
	}

	if status == ExecutionStatusRunning {
		now := time.Now()
		nodeExec.StartTime = &now
	}

	if errorMsg != "" {
		nodeExec.ErrorMsg = errorMsg
	}

	e.Nodes = append(e.Nodes, nodeExec)
}

// GetProgress 获取执行进度
func (e *Execution) GetProgress() float64 {
	if e.Metrics == nil || e.Metrics.TotalNodes == 0 {
		return 0.0
	}

	return float64(e.Metrics.CompletedNodes) / float64(e.Metrics.TotalNodes) * 100.0
}

// GetDuration 获取执行时长
func (e *Execution) GetDuration() time.Duration {
	if e.StartedAt == nil {
		return 0
	}

	if e.CompletedAt != nil {
		return e.CompletedAt.Sub(*e.StartedAt)
	}

	return time.Since(*e.StartedAt)
}
