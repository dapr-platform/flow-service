/*
 * @module: flow-service/service/models/edge
 * @description: 边模型定义，表示工作流中节点之间的连接关系和数据流，作为JSON存储在Workflow中
 * @architecture: 数据模型层 - 核心业务模型
 * @documentReference: /docs/edge-model.md
 * @stateFlow: /docs/edge-state-flow.md
 * @rules:
 *   - 边必须连接两个不同的节点
 *   - 不能形成环路
 *   - 条件边必须指定条件表达式
 * @dependencies:
 *   - time
 *   - encoding/json
 *   - fmt
 * @refs:
 *   - service/models/workflow.go
 *   - service/models/node.go
 */

package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EdgeTypeEnum 边类型枚举
type EdgeTypeEnum string

const (
	// EdgeTypeNormal 普通边
	EdgeTypeNormal EdgeTypeEnum = "normal"
	// EdgeTypeConditional 条件边
	EdgeTypeConditional EdgeTypeEnum = "conditional"
	// EdgeTypeLoop 循环边
	EdgeTypeLoop EdgeTypeEnum = "loop"
	// EdgeTypeError 错误处理边
	EdgeTypeError EdgeTypeEnum = "error"
	// EdgeTypeTimeout 超时处理边
	EdgeTypeTimeout EdgeTypeEnum = "timeout"
	// EdgeTypeSkip 跳过边
	EdgeTypeSkip EdgeTypeEnum = "skip"
)

// String 返回类型字符串
func (t EdgeTypeEnum) String() string {
	return string(t)
}

// IsValid 验证类型是否有效
func (t EdgeTypeEnum) IsValid() bool {
	switch t {
	case EdgeTypeNormal, EdgeTypeConditional, EdgeTypeLoop,
		EdgeTypeError, EdgeTypeTimeout, EdgeTypeSkip:
		return true
	default:
		return false
	}
}

// EdgeStatusEnum 边状态枚举
type EdgeStatusEnum string

const (
	// EdgeStatusActive 激活状态
	EdgeStatusActive EdgeStatusEnum = "active"
	// EdgeStatusInactive 非激活状态
	EdgeStatusInactive EdgeStatusEnum = "inactive"
	// EdgeStatusDisabled 禁用状态
	EdgeStatusDisabled EdgeStatusEnum = "disabled"
)

// String 返回状态字符串
func (s EdgeStatusEnum) String() string {
	return string(s)
}

// IsValid 验证状态是否有效
func (s EdgeStatusEnum) IsValid() bool {
	switch s {
	case EdgeStatusActive, EdgeStatusInactive, EdgeStatusDisabled:
		return true
	default:
		return false
	}
}

// EdgeCondition 边条件配置
type EdgeCondition struct {
	// 条件表达式
	Expression string `json:"expression" validate:"required"`

	// 条件类型
	Type string `json:"type" validate:"oneof=javascript python go simple"`

	// 条件参数
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// 默认值
	DefaultValue bool `json:"default_value"`

	// 超时时间 (单位: 纳秒)
	Timeout time.Duration `json:"timeout" swaggertype:"integer" validate:"min=0"`

	// 是否启用
	Enabled bool `json:"enabled"`
}

// EdgeDataMapping 边数据映射配置
type EdgeDataMapping struct {
	// 源字段映射
	SourceMapping map[string]string `json:"source_mapping,omitempty"`

	// 目标字段映射
	TargetMapping map[string]string `json:"target_mapping,omitempty"`

	// 数据转换规则
	TransformRules []TransformRule `json:"transform_rules,omitempty"`

	// 数据过滤规则
	FilterRules []FilterRule `json:"filter_rules,omitempty"`

	// 是否传递所有数据
	PassThrough bool `json:"pass_through"`
}

// TransformRule 数据转换规则
type TransformRule struct {
	// 字段名
	Field string `json:"field" validate:"required"`

	// 转换类型
	Type string `json:"type" validate:"oneof=format convert calculate"`

	// 转换表达式
	Expression string `json:"expression" validate:"required"`

	// 转换参数
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// FilterRule 数据过滤规则
type FilterRule struct {
	// 字段名
	Field string `json:"field" validate:"required"`

	// 操作符
	Operator string `json:"operator" validate:"oneof=eq ne gt lt ge le in nin contains"`

	// 比较值
	Value interface{} `json:"value"`

	// 逻辑操作符
	Logic string `json:"logic" validate:"oneof=and or"`
}

// EdgeConfig 边配置
type EdgeConfig struct {
	// 条件配置
	Condition *EdgeCondition `json:"condition,omitempty"`

	// 数据映射配置
	DataMapping *EdgeDataMapping `json:"data_mapping,omitempty"`

	// 权重（用于负载均衡）
	Weight int `json:"weight" validate:"min=1,max=100"`

	// 优先级
	Priority int `json:"priority" validate:"min=0,max=10"`

	// 延迟执行时间 (单位: 纳秒)
	Delay time.Duration `json:"delay" swaggertype:"integer" validate:"min=0"`

	// 重试配置
	RetryConfig *EdgeRetryConfig `json:"retry_config,omitempty"`

	// 标签
	Tags []string `json:"tags,omitempty"`

	// 描述
	Description string `json:"description,omitempty"`

	// 是否启用
	Enabled bool `json:"enabled"`
}

// EdgeRetryConfig 边重试配置
type EdgeRetryConfig struct {
	// 最大重试次数
	MaxRetries int `json:"max_retries" validate:"min=0,max=5"`

	// 重试间隔 (单位: 纳秒)
	RetryInterval time.Duration `json:"retry_interval" swaggertype:"integer" validate:"min=0"`

	// 退避策略
	BackoffStrategy string `json:"backoff_strategy" validate:"oneof=fixed linear exponential"`

	// 重试条件
	RetryConditions []string `json:"retry_conditions,omitempty"`
}

// EdgeExecution 边执行信息
type EdgeExecution struct {
	// 执行ID
	ExecutionID string `json:"execution_id"`

	// 执行次数
	ExecutionCount int64 `json:"execution_count"`

	// 成功次数
	SuccessCount int64 `json:"success_count"`

	// 失败次数
	FailureCount int64 `json:"failure_count"`

	// 最后执行时间
	LastExecutionTime *time.Time `json:"last_execution_time,omitempty"`

	// 最后成功时间
	LastSuccessTime *time.Time `json:"last_success_time,omitempty"`

	// 最后失败时间
	LastFailureTime *time.Time `json:"last_failure_time,omitempty"`

	// 平均执行时间 (单位: 纳秒)
	AverageExecutionTime time.Duration `json:"average_execution_time" swaggertype:"integer"`

	// 最后错误信息
	LastError string `json:"last_error,omitempty"`
}

// Edge 边模型定义 - 作为JSON存储在Workflow中
type Edge struct {
	// 基础字段
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// 连接字段
	FromNodeID string `json:"from_node_id" validate:"required"`
	ToNodeID   string `json:"to_node_id" validate:"required"`

	// 类型和状态
	Type   EdgeTypeEnum   `json:"type" validate:"required"`
	Status EdgeStatusEnum `json:"status"`

	// 配置字段
	Config *EdgeConfig `json:"config"`

	// 执行信息
	Execution *EdgeExecution `json:"execution"`

	UIConfig map[string]interface{} `json:"ui_config"`
}

// Validate 验证边
func (e *Edge) Validate() error {
	// 验证基础字段
	if e.ID == "" {
		return fmt.Errorf("edge ID cannot be empty")
	}

	if e.FromNodeID == "" {
		return fmt.Errorf("edge from node ID cannot be empty")
	}

	if e.ToNodeID == "" {
		return fmt.Errorf("edge to node ID cannot be empty")
	}

	// 验证不能连接自己
	if e.FromNodeID == e.ToNodeID {
		return fmt.Errorf("edge cannot connect node to itself")
	}

	if !e.Type.IsValid() {
		return fmt.Errorf("invalid edge type: %s", e.Type)
	}

	if !e.Status.IsValid() {
		return fmt.Errorf("invalid edge status: %s", e.Status)
	}

	// 验证条件边必须有条件配置
	if e.Type == EdgeTypeConditional {
		if e.Config == nil || e.Config.Condition == nil {
			return fmt.Errorf("conditional edge must have condition config")
		}

		if e.Config.Condition.Expression == "" {
			return fmt.Errorf("conditional edge must have condition expression")
		}
	}

	return nil
}

// UpdateStatus 更新状态
func (e *Edge) UpdateStatus(newStatus EdgeStatusEnum) error {
	if !newStatus.IsValid() {
		return fmt.Errorf("invalid edge status: %s", newStatus)
	}

	e.Status = newStatus
	return nil
}

// IsConditional 检查是否为条件边
func (e *Edge) IsConditional() bool {
	return e.Type == EdgeTypeConditional
}

// IsEnabled 检查边是否启用
func (e *Edge) IsEnabled() bool {
	if e.Status == EdgeStatusDisabled {
		return false
	}

	if e.Config != nil {
		return e.Config.Enabled
	}

	return true
}

// GetWeight 获取权重
func (e *Edge) GetWeight() int {
	if e.Config != nil && e.Config.Weight > 0 {
		return e.Config.Weight
	}
	return 1 // 默认权重
}

// GetPriority 获取优先级
func (e *Edge) GetPriority() int {
	if e.Config != nil {
		return e.Config.Priority
	}
	return 0 // 默认优先级
}

// GetDelay 获取延迟时间
func (e *Edge) GetDelay() time.Duration {
	if e.Config != nil {
		return e.Config.Delay
	}
	return 0 // 默认无延迟
}

// EvaluateCondition 评估条件（简化版本，实际应该调用表达式引擎）
func (e *Edge) EvaluateCondition(context map[string]interface{}) (bool, error) {
	if !e.IsConditional() {
		return true, nil // 非条件边总是返回true
	}

	if e.Config == nil || e.Config.Condition == nil {
		return false, fmt.Errorf("conditional edge missing condition config")
	}

	condition := e.Config.Condition
	if !condition.Enabled {
		return condition.DefaultValue, nil
	}

	// 简单的条件评估实现
	expr := condition.Expression
	switch condition.Type {
	case "simple":
		return e.evaluateSimpleCondition(expr, context)
	default:
		return condition.DefaultValue, nil
	}
}

// evaluateSimpleCondition 评估简单条件
func (e *Edge) evaluateSimpleCondition(expr string, context map[string]interface{}) (bool, error) {
	// 简单的条件解析：支持 "value > 10" 格式
	if expr == "true" {
		return true, nil
	}
	if expr == "false" {
		return false, nil
	}

	// 解析 "value > 10" 格式
	parts := strings.Fields(expr)
	if len(parts) != 3 {
		return false, fmt.Errorf("invalid expression format: %s", expr)
	}

	field := parts[0]
	operator := parts[1]
	valueStr := parts[2]

	contextValue, exists := context[field]
	if !exists {
		return false, nil
	}

	// 尝试转换为数字进行比较
	contextNum, ok1 := contextValue.(int)
	if !ok1 {
		if f, ok2 := contextValue.(float64); ok2 {
			contextNum = int(f)
		} else {
			return false, fmt.Errorf("cannot compare non-numeric value")
		}
	}

	targetNum := 0
	if _, err := fmt.Sscanf(valueStr, "%d", &targetNum); err != nil {
		return false, fmt.Errorf("invalid target value: %s", valueStr)
	}

	switch operator {
	case ">":
		return contextNum > targetNum, nil
	case ">=":
		return contextNum >= targetNum, nil
	case "<":
		return contextNum < targetNum, nil
	case "<=":
		return contextNum <= targetNum, nil
	case "==", "=":
		return contextNum == targetNum, nil
	case "!=":
		return contextNum != targetNum, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// TransformData 转换数据
func (e *Edge) TransformData(inputData map[string]interface{}) (map[string]interface{}, error) {
	if e.Config == nil || e.Config.DataMapping == nil {
		return inputData, nil // 无配置时直接返回原数据
	}

	mapping := e.Config.DataMapping

	// 如果启用了直通模式
	if mapping.PassThrough {
		return inputData, nil
	}

	result := make(map[string]interface{})

	// 应用源字段映射
	if mapping.SourceMapping != nil {
		for inputField, sourceField := range mapping.SourceMapping {
			if value, exists := inputData[sourceField]; exists {
				result[inputField] = value
			}
		}
	}

	// 应用目标字段映射
	if mapping.TargetMapping != nil {
		tempResult := make(map[string]interface{})
		for sourceField, targetField := range mapping.TargetMapping {
			if value, exists := result[sourceField]; exists {
				tempResult[targetField] = value
			} else if value, exists := inputData[sourceField]; exists {
				tempResult[targetField] = value
			}
		}
		result = tempResult
	}

	// 如果没有任何映射，复制所有数据
	if mapping.SourceMapping == nil && mapping.TargetMapping == nil {
		for key, value := range inputData {
			result[key] = value
		}
	}

	// 应用转换规则
	for _, rule := range mapping.TransformRules {
		if err := e.applyTransformRule(result, rule); err != nil {
			return nil, fmt.Errorf("failed to apply transform rule for field %s: %w", rule.Field, err)
		}
	}

	// 应用过滤规则
	if len(mapping.FilterRules) > 0 {
		if !e.applyFilterRules(result, mapping.FilterRules) {
			return nil, fmt.Errorf("data filtered out by filter rules")
		}
	}

	return result, nil
}

// applyTransformRule 应用转换规则
func (e *Edge) applyTransformRule(data map[string]interface{}, rule TransformRule) error {
	// 这里应该实现实际的转换逻辑
	// 目前只是简单的示例
	switch rule.Type {
	case "format":
		// 格式化转换
		if value, exists := data[rule.Field]; exists {
			data[rule.Field] = fmt.Sprintf(rule.Expression, value)
		}
	case "convert":
		// 类型转换
		// 实现类型转换逻辑
	case "calculate":
		// 计算转换
		// 实现计算逻辑
	}

	return nil
}

// applyFilterRules 应用过滤规则
func (e *Edge) applyFilterRules(data map[string]interface{}, rules []FilterRule) bool {
	// 这里应该实现实际的过滤逻辑
	// 目前只是简单返回true
	return true
}

// RecordExecution 记录执行信息
func (e *Edge) RecordExecution(success bool, duration time.Duration, err error) {
	if e.Execution == nil {
		e.Execution = &EdgeExecution{}
	}

	now := time.Now()
	e.Execution.ExecutionCount++
	e.Execution.LastExecutionTime = &now

	if success {
		e.Execution.SuccessCount++
		e.Execution.LastSuccessTime = &now
	} else {
		e.Execution.FailureCount++
		e.Execution.LastFailureTime = &now
		if err != nil {
			e.Execution.LastError = err.Error()
		}
	}

	// 更新平均执行时间
	if e.Execution.ExecutionCount > 0 {
		totalTime := time.Duration(e.Execution.ExecutionCount-1)*e.Execution.AverageExecutionTime + duration
		e.Execution.AverageExecutionTime = totalTime / time.Duration(e.Execution.ExecutionCount)
	}
}

// Clone 克隆边
func (e *Edge) Clone() *Edge {
	clone := &Edge{
		ID:          fmt.Sprintf("%s_copy_%d", e.ID, time.Now().Unix()),
		Name:        e.Name + " (Copy)",
		Description: e.Description,
		FromNodeID:  e.FromNodeID,
		ToNodeID:    e.ToNodeID,
		Type:        e.Type,
		Status:      e.Status,
		UIConfig:    e.UIConfig,
	}

	// 深拷贝配置
	if e.Config != nil {
		configJSON, _ := json.Marshal(e.Config)
		clone.Config = &EdgeConfig{}
		json.Unmarshal(configJSON, clone.Config)
	}

	// 深拷贝执行信息
	if e.Execution != nil {
		executionJSON, _ := json.Marshal(e.Execution)
		clone.Execution = &EdgeExecution{}
		json.Unmarshal(executionJSON, clone.Execution)
	}

	return clone
}

// GetSuccessRate 获取成功率
func (e *Edge) GetSuccessRate() float64 {
	if e.Execution == nil || e.Execution.ExecutionCount == 0 {
		return 0.0
	}

	return float64(e.Execution.SuccessCount) / float64(e.Execution.ExecutionCount)
}

// GetFailureRate 获取失败率
func (e *Edge) GetFailureRate() float64 {
	if e.Execution == nil || e.Execution.ExecutionCount == 0 {
		return 0.0
	}

	return float64(e.Execution.FailureCount) / float64(e.Execution.ExecutionCount)
}
