/*
 * @module: flow-service/service/models/node
 * @description: 节点模型定义，表示工作流中的单个执行单元，作为JSON存储在Workflow中
 * @architecture: 数据模型层 - 核心业务模型
 * @documentReference: /docs/node-model.md
 * @stateFlow: /docs/node-state-flow.md
 * @rules:
 *   - 节点ID在工作流内必须唯一
 *   - 节点必须指定有效的插件类型
 *   - 依赖关系必须指向存在的节点
 * @dependencies:
 *   - time
 *   - encoding/json
 *   - fmt
 * @refs:
 *   - service/models/workflow.go
 *   - service/models/edge.go
 *   - pkg/plugin/interface.go
 */

package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// NodeStatusEnum 节点状态枚举
type NodeStatusEnum string

const (
	// NodeStatusPending 待执行状态
	NodeStatusPending NodeStatusEnum = "pending"
	// NodeStatusRunning 运行中状态
	NodeStatusRunning NodeStatusEnum = "running"
	// NodeStatusCompleted 已完成状态
	NodeStatusCompleted NodeStatusEnum = "completed"
	// NodeStatusFailed 失败状态
	NodeStatusFailed NodeStatusEnum = "failed"
	// NodeStatusSkipped 跳过状态
	NodeStatusSkipped NodeStatusEnum = "skipped"
	// NodeStatusCancelled 已取消状态
	NodeStatusCancelled NodeStatusEnum = "cancelled"
	// NodeStatusRetrying 重试中状态
	NodeStatusRetrying NodeStatusEnum = "retrying"
	// NodeStatusInactive 非活跃状态
	NodeStatusInactive NodeStatusEnum = "inactive"
)

// String 返回状态字符串
func (s NodeStatusEnum) String() string {
	return string(s)
}

// IsValid 验证状态是否有效
func (s NodeStatusEnum) IsValid() bool {
	switch s {
	case NodeStatusPending, NodeStatusRunning, NodeStatusCompleted,
		NodeStatusFailed, NodeStatusSkipped, NodeStatusCancelled, NodeStatusRetrying, NodeStatusInactive:
		return true
	default:
		return false
	}
}

// CanTransitionTo 检查是否可以转换到目标状态
func (s NodeStatusEnum) CanTransitionTo(target NodeStatusEnum) bool {
	transitions := map[NodeStatusEnum][]NodeStatusEnum{
		NodeStatusPending:   {NodeStatusRunning, NodeStatusSkipped, NodeStatusCancelled},
		NodeStatusRunning:   {NodeStatusCompleted, NodeStatusFailed, NodeStatusCancelled, NodeStatusRetrying},
		NodeStatusRetrying:  {NodeStatusRunning, NodeStatusFailed, NodeStatusCancelled},
		NodeStatusCompleted: {},
		NodeStatusFailed:    {NodeStatusPending, NodeStatusRetrying}, // 允许重试
		NodeStatusSkipped:   {},
		NodeStatusCancelled: {NodeStatusPending}, // 允许重新执行
	}

	validTargets, exists := transitions[s]
	if !exists {
		return false
	}

	for _, validTarget := range validTargets {
		if validTarget == target {
			return true
		}
	}
	return false
}

// IsFinalState 检查是否为最终状态
func (s NodeStatusEnum) IsFinalState() bool {
	return s == NodeStatusCompleted || s == NodeStatusSkipped
}

// IsErrorState 检查是否为错误状态
func (s NodeStatusEnum) IsErrorState() bool {
	return s == NodeStatusFailed || s == NodeStatusCancelled
}

// NodeTypeEnum 节点类型枚举
type NodeTypeEnum string

const (
	// NodeTypeDataSource 数据源节点
	NodeTypeDataSource NodeTypeEnum = "datasource"
	// NodeTypeTransform 数据转换节点
	NodeTypeTransform NodeTypeEnum = "transform"
	// NodeTypeOutput 数据输出节点
	NodeTypeOutput NodeTypeEnum = "output"
	// NodeTypeControl 控制流节点
	NodeTypeControl NodeTypeEnum = "control"
	// NodeTypeCondition 条件判断节点
	NodeTypeCondition NodeTypeEnum = "condition"
	// NodeTypeLoop 循环节点
	NodeTypeLoop NodeTypeEnum = "loop"
	// NodeTypeSubDAG 子DAG节点
	NodeTypeSubDAG NodeTypeEnum = "subdag"
	// NodeTypeScript 脚本执行节点
	NodeTypeScript NodeTypeEnum = "script"
	// NodeTypeAPI API调用节点
	NodeTypeAPI NodeTypeEnum = "api"
	// NodeTypeTimer 定时器节点
	NodeTypeTimer NodeTypeEnum = "timer"
)

// String 返回类型字符串
func (t NodeTypeEnum) String() string {
	return string(t)
}

// IsValid 验证类型是否有效
func (t NodeTypeEnum) IsValid() bool {
	switch t {
	case NodeTypeDataSource, NodeTypeTransform, NodeTypeOutput, NodeTypeControl,
		NodeTypeCondition, NodeTypeLoop, NodeTypeSubDAG, NodeTypeScript,
		NodeTypeAPI, NodeTypeTimer:
		return true
	default:
		return false
	}
}

// NodeConfig 节点配置
type NodeConfig struct {
	// 插件配置
	PluginConfig map[string]interface{} `json:"plugin_config,omitempty"`

	// 输入配置
	InputConfig *InputConfig `json:"input_config,omitempty"`

	// 输出配置
	OutputConfig *OutputConfig `json:"output_config,omitempty"`

	// 重试配置
	RetryConfig *NodeRetryConfig `json:"retry_config,omitempty"`

	// 超时配置
	TimeoutConfig *TimeoutConfig `json:"timeout_config,omitempty"`

	// 资源配置
	ResourceConfig *ResourceConfig `json:"resource_config,omitempty"`

	// 条件配置（用于条件节点）
	ConditionConfig *ConditionConfig `json:"condition_config,omitempty"`

	// 循环配置（用于循环节点）
	LoopConfig *LoopConfig `json:"loop_config,omitempty"`

	// 环境变量
	Environment map[string]string `json:"environment,omitempty"`

	// 标签
	Tags []string `json:"tags,omitempty"`

	// 描述
	Description string `json:"description,omitempty"`
}

// InputConfig 输入配置
type InputConfig struct {
	// 输入映射
	Mapping map[string]string `json:"mapping,omitempty"`

	// 默认值
	Defaults map[string]interface{} `json:"defaults,omitempty"`

	// 验证规则
	Validation map[string]interface{} `json:"validation,omitempty"`

	// 是否必需
	Required []string `json:"required,omitempty"`
}

// OutputConfig 输出配置
type OutputConfig struct {
	// 输出映射
	Mapping map[string]string `json:"mapping,omitempty"`

	// 输出过滤
	Filter []string `json:"filter,omitempty"`

	// 输出格式化
	Format map[string]interface{} `json:"format,omitempty"`
}

// NodeRetryConfig 节点重试配置
type NodeRetryConfig struct {
	// 最大重试次数
	MaxRetries int `json:"max_retries" validate:"min=0,max=10"`

	// 重试间隔 (单位: 纳秒)
	RetryInterval time.Duration `json:"retry_interval" swaggertype:"integer" validate:"min=0"`

	// 退避策略
	BackoffStrategy string `json:"backoff_strategy" validate:"oneof=fixed linear exponential"`

	// 退避倍数
	BackoffMultiplier float64 `json:"backoff_multiplier" validate:"min=1"`

	// 最大重试间隔 (单位: 纳秒)
	MaxRetryInterval time.Duration `json:"max_retry_interval" swaggertype:"integer" validate:"min=0"`

	// 重试条件
	RetryConditions []string `json:"retry_conditions,omitempty"`

	// 停止重试条件
	StopConditions []string `json:"stop_conditions,omitempty"`
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	// 执行超时 (单位: 纳秒)
	ExecutionTimeout time.Duration `json:"execution_timeout" swaggertype:"integer" validate:"min=0"`

	// 连接超时 (单位: 纳秒)
	ConnectionTimeout time.Duration `json:"connection_timeout" swaggertype:"integer" validate:"min=0"`

	// 读取超时 (单位: 纳秒)
	ReadTimeout time.Duration `json:"read_timeout" swaggertype:"integer" validate:"min=0"`

	// 写入超时 (单位: 纳秒)
	WriteTimeout time.Duration `json:"write_timeout" swaggertype:"integer" validate:"min=0"`
}

// ResourceConfig 资源配置
type ResourceConfig struct {
	// CPU限制
	CPULimit string `json:"cpu_limit,omitempty"`

	// 内存限制
	MemoryLimit string `json:"memory_limit,omitempty"`

	// 磁盘限制
	DiskLimit string `json:"disk_limit,omitempty"`

	// 网络限制
	NetworkLimit string `json:"network_limit,omitempty"`

	// 并发限制
	ConcurrencyLimit int `json:"concurrency_limit" validate:"min=1"`
}

// ConditionConfig 条件配置
type ConditionConfig struct {
	// 条件表达式
	Expression string `json:"expression" validate:"required"`

	// 条件类型
	Type string `json:"type" validate:"oneof=javascript python go"`

	// 真分支
	TrueBranch string `json:"true_branch,omitempty"`

	// 假分支
	FalseBranch string `json:"false_branch,omitempty"`

	// 默认分支
	DefaultBranch string `json:"default_branch,omitempty"`
}

// LoopConfig 循环配置
type LoopConfig struct {
	// 循环类型
	Type string `json:"type" validate:"oneof=for while foreach"`

	// 循环条件
	Condition string `json:"condition,omitempty"`

	// 循环变量
	Variable string `json:"variable,omitempty"`

	// 循环数据
	Data interface{} `json:"data,omitempty"`

	// 最大迭代次数
	MaxIterations int `json:"max_iterations" validate:"min=1"`

	// 并行执行
	Parallel bool `json:"parallel"`

	// 并发数
	Concurrency int `json:"concurrency" validate:"min=1"`
}

// NodeExecution 节点执行信息
type NodeExecution struct {
	// 执行ID
	ExecutionID string `json:"execution_id"`

	// 开始时间
	StartTime *time.Time `json:"start_time,omitempty"`

	// 结束时间
	EndTime *time.Time `json:"end_time,omitempty"`

	// 执行时长 (单位: 纳秒)
	Duration time.Duration `json:"duration" swaggertype:"integer"`

	// 重试次数
	RetryCount int `json:"retry_count"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 输入数据
	InputData map[string]interface{} `json:"input_data,omitempty"`

	// 输出数据
	OutputData map[string]interface{} `json:"output_data,omitempty"`

	// 日志
	Logs []string `json:"logs,omitempty"`

	// 指标
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// Node 节点模型定义 - 作为JSON存储在Workflow中
type Node struct {
	// 基础字段
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description"`

	// 类型和插件
	Type   NodeTypeEnum `json:"type" validate:"required"`
	Plugin string       `json:"plugin" validate:"required"`

	// 状态字段
	Status     NodeStatusEnum `json:"status"`
	RetryCount int            `json:"retry_count"`

	// 依赖关系
	Dependencies []string `json:"dependencies"`

	// 配置字段
	Config *NodeConfig `json:"config"`

	// 执行信息
	Execution *NodeExecution `json:"execution"`

	UIConfig map[string]interface{} `json:"ui_config"`
}

// Validate 验证节点
func (n *Node) Validate() error {
	// 验证基础字段
	if n.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	if n.Name == "" {
		return fmt.Errorf("node name is required")
	}

	if !n.Type.IsValid() {
		return fmt.Errorf("invalid node type: %s", n.Type)
	}

	if !n.Status.IsValid() {
		return fmt.Errorf("invalid node status: %s", n.Status)
	}

	if n.Plugin == "" {
		return fmt.Errorf("node plugin cannot be empty")
	}

	// 验证重试次数
	if n.RetryCount < 0 {
		return fmt.Errorf("retry count cannot be negative")
	}

	// 验证依赖关系
	for _, dep := range n.Dependencies {
		if dep == "" {
			return fmt.Errorf("empty dependency not allowed")
		}
	}

	return nil
}

// UpdateStatus 更新状态
func (n *Node) UpdateStatus(newStatus NodeStatusEnum) error {
	if !n.Status.CanTransitionTo(newStatus) {
		return fmt.Errorf("cannot transition from %s to %s", n.Status, newStatus)
	}

	n.Status = newStatus
	return nil
}

// IncrementRetryCount 增加重试次数
func (n *Node) IncrementRetryCount() {
	n.RetryCount++
}

// ResetRetryCount 重置重试次数
func (n *Node) ResetRetryCount() {
	n.RetryCount = 0
}

// CanRetry 检查是否可以重试
func (n *Node) CanRetry() bool {
	if n.Config == nil || n.Config.RetryConfig == nil {
		return n.RetryCount < 3 // 默认最大重试3次
	}

	return n.RetryCount < n.Config.RetryConfig.MaxRetries
}

// GetMaxRetries 获取最大重试次数
func (n *Node) GetMaxRetries() int {
	if n.Config == nil || n.Config.RetryConfig == nil {
		return 3 // 默认最大重试3次
	}

	return n.Config.RetryConfig.MaxRetries
}

// GetExecutionTimeout 获取执行超时时间
func (n *Node) GetExecutionTimeout() time.Duration {
	if n.Config == nil || n.Config.TimeoutConfig == nil {
		return 5 * time.Minute // 默认5分钟超时
	}

	if n.Config.TimeoutConfig.ExecutionTimeout > 0 {
		return n.Config.TimeoutConfig.ExecutionTimeout
	}

	return 5 * time.Minute
}

// IsReady 检查节点是否准备就绪（所有依赖都已完成）
func (n *Node) IsReady(completedNodes map[string]bool) bool {
	for _, dep := range n.Dependencies {
		if !completedNodes[dep] {
			return false
		}
	}
	return true
}

// Clone 克隆节点
func (n *Node) Clone() *Node {
	clone := &Node{
		ID:          fmt.Sprintf("%s_copy_%d", n.ID, time.Now().Unix()),
		Name:        n.Name + " (Copy)",
		Description: n.Description,
		Type:        n.Type,
		Plugin:      n.Plugin,
		Status:      NodeStatusPending, // 克隆的节点始终为待执行状态
		RetryCount:  0,                 // 重置重试次数
		UIConfig:    n.UIConfig,
	}

	// 深拷贝依赖关系
	if n.Dependencies != nil {
		clone.Dependencies = make([]string, len(n.Dependencies))
		copy(clone.Dependencies, n.Dependencies)
	}

	// 深拷贝配置
	if n.Config != nil {
		configJSON, _ := json.Marshal(n.Config)
		clone.Config = &NodeConfig{}
		json.Unmarshal(configJSON, clone.Config)
	}

	// 重置执行信息（克隆的节点应该从新开始）
	clone.Execution = &NodeExecution{
		ExecutionID: "",
		RetryCount:  0,
	}

	return clone
}

// StartExecution 开始执行
func (n *Node) StartExecution(executionID string) {
	if n.Execution == nil {
		n.Execution = &NodeExecution{}
	}

	now := time.Now()
	n.Execution.ExecutionID = executionID
	n.Execution.StartTime = &now
	n.Status = NodeStatusRunning
}

// CompleteExecution 完成执行
func (n *Node) CompleteExecution(outputData map[string]interface{}) {
	if n.Execution == nil {
		n.Execution = &NodeExecution{}
	}

	now := time.Now()
	n.Execution.EndTime = &now
	n.Execution.OutputData = outputData

	if n.Execution.StartTime != nil {
		n.Execution.Duration = now.Sub(*n.Execution.StartTime)
	}

	n.Status = NodeStatusCompleted
}

// FailExecution 执行失败
func (n *Node) FailExecution(err error) {
	if n.Execution == nil {
		n.Execution = &NodeExecution{}
	}

	now := time.Now()
	n.Execution.EndTime = &now
	n.Execution.Error = err.Error()

	if n.Execution.StartTime != nil {
		n.Execution.Duration = now.Sub(*n.Execution.StartTime)
	}

	n.Status = NodeStatusFailed
}
