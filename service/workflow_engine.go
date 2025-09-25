/**
 * @module workflow_engine
 * @description 工作流执行引擎，专为2层架构设计，负责工作流调度和协调
 * @architecture 轻量级执行引擎设计，专注于依赖管理、调度协调和状态管理
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow execution_states: created -> running -> completed/failed
 * @rules 执行状态变更必须遵循状态机规则，通过节点注册系统执行具体节点逻辑
 * @dependencies service/models/workflow.go, service/models/execution.go, service/nodes
 * @refs service/execution_service.go, service/nodes/registry.go
 */

package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"flow-service/service/models"
	"flow-service/service/nodes"
)

// EngineStatus 引擎状态
type EngineStatus int

const (
	EngineStatusStopped EngineStatus = iota
	EngineStatusRunning
	EngineStatusStopping
)

// WorkflowEngine 工作流执行引擎
type WorkflowEngine struct {
	status         EngineStatus
	executions     map[string]*ExecutionContext
	mu             sync.RWMutex
	maxConcurrency int
	nodeRegistry   *nodes.NodeRegistry
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	ExecutionID string
	WorkflowID  string
	Workflow    *models.Workflow
	Execution   *models.Execution
	Variables   map[string]interface{}
	NodeStates  map[string]models.ExecutionStatus

	// 新增字段用于依赖管理
	NodeDependencies map[string][]string // 节点依赖关系
	CompletedNodes   map[string]bool     // 已完成的节点
	ExecutingNodes   map[string]bool     // 正在执行的节点
	ReadyNodes       chan string         // 准备执行的节点队列

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewWorkflowEngine 创建工作流执行引擎
func NewWorkflowEngine() *WorkflowEngine {
	return &WorkflowEngine{
		status:         EngineStatusStopped,
		executions:     make(map[string]*ExecutionContext),
		maxConcurrency: 10, // 最大并发执行数
		nodeRegistry:   nodes.GetRegistry(),
	}
}

// Start 启动引擎
func (e *WorkflowEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status == EngineStatusRunning {
		return fmt.Errorf("engine is already running")
	}

	e.status = EngineStatusRunning
	log.Println("WorkflowEngine started")
	return nil
}

// Stop 停止引擎
func (e *WorkflowEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status != EngineStatusRunning {
		return fmt.Errorf("engine is not running")
	}

	e.status = EngineStatusStopping

	// 取消所有正在执行的任务
	for _, execCtx := range e.executions {
		if execCtx.cancel != nil {
			execCtx.cancel()
		}
	}

	e.status = EngineStatusStopped
	log.Println("WorkflowEngine stopped")
	return nil
}

// ExecuteWorkflow 执行工作流
func (e *WorkflowEngine) ExecuteWorkflow(ctx context.Context, workflow *models.Workflow, execution *models.Execution) error {
	if e.status != EngineStatusRunning {
		return fmt.Errorf("engine is not running")
	}

	// 检查并发限制
	e.mu.RLock()
	if len(e.executions) >= e.maxConcurrency {
		e.mu.RUnlock()
		return fmt.Errorf("maximum concurrent executions reached: %d", e.maxConcurrency)
	}
	e.mu.RUnlock()

	// 创建执行上下文
	execCtx := &ExecutionContext{
		ExecutionID:      execution.ID,
		WorkflowID:       workflow.ID,
		Workflow:         workflow,
		Execution:        execution,
		Variables:        make(map[string]interface{}),
		NodeStates:       make(map[string]models.ExecutionStatus),
		NodeDependencies: make(map[string][]string),
		CompletedNodes:   make(map[string]bool),
		ExecutingNodes:   make(map[string]bool),
		ReadyNodes:       make(chan string, len(workflow.Nodes)),
	}

	execCtx.ctx, execCtx.cancel = context.WithCancel(ctx)

	// 注册执行上下文
	e.mu.Lock()
	e.executions[execution.ID] = execCtx
	e.mu.Unlock()

	// 异步执行工作流
	go func() {
		defer func() {
			// 清理执行上下文
			e.mu.Lock()
			delete(e.executions, execution.ID)
			e.mu.Unlock()

			// 关闭通道
			close(execCtx.ReadyNodes)
		}()

		if err := e.executeWorkflowInternal(execCtx); err != nil {
			log.Printf("Workflow execution failed: %v", err)
		}
	}()

	return nil
}

// CancelExecution 取消执行
func (e *WorkflowEngine) CancelExecution(executionID string) error {
	e.mu.RLock()
	execCtx, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	if execCtx.cancel != nil {
		execCtx.cancel()
	}

	return nil
}

// GetExecutionStatus 获取执行状态
func (e *WorkflowEngine) GetExecutionStatus(executionID string) (*ExecutionStatus, error) {
	e.mu.RLock()
	execCtx, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	execCtx.mu.RLock()
	defer execCtx.mu.RUnlock()

	return &ExecutionStatus{
		ExecutionID: executionID,
		WorkflowID:  execCtx.WorkflowID,
		Status:      execCtx.Execution.Status,
		NodeStates:  execCtx.NodeStates,
		Variables:   execCtx.Variables,
	}, nil
}

// ExecutionStatus 执行状态
type ExecutionStatus struct {
	ExecutionID string                            `json:"execution_id"`
	WorkflowID  string                            `json:"workflow_id"`
	Status      models.ExecutionStatus            `json:"status"`
	NodeStates  map[string]models.ExecutionStatus `json:"node_states"`
	Variables   map[string]interface{}            `json:"variables"`
}

// executeWorkflowInternal 内部执行工作流
func (e *WorkflowEngine) executeWorkflowInternal(execCtx *ExecutionContext) error {
	// 检查工作流节点
	if execCtx.Workflow.Nodes == nil || len(execCtx.Workflow.Nodes) == 0 {
		return fmt.Errorf("workflow has no nodes")
	}

	log.Printf("Starting workflow execution: %s", execCtx.ExecutionID)

	// 构建依赖关系图
	if err := e.buildDependencyGraph(execCtx); err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// 找到起始节点（没有依赖的节点）
	startNodes := e.findStartNodes(execCtx)
	if len(startNodes) == 0 {
		return fmt.Errorf("no start nodes found in workflow")
	}

	// 将起始节点加入准备队列
	for _, nodeID := range startNodes {
		select {
		case execCtx.ReadyNodes <- nodeID:
		case <-execCtx.ctx.Done():
			return execCtx.ctx.Err()
		}
	}

	// 执行工作流
	return e.executeDAG(execCtx)
}

// buildDependencyGraph 构建依赖关系图
func (e *WorkflowEngine) buildDependencyGraph(execCtx *ExecutionContext) error {
	// 初始化所有节点的依赖列表
	for nodeID := range execCtx.Workflow.Nodes {
		execCtx.NodeDependencies[nodeID] = []string{}
		execCtx.NodeStates[nodeID] = models.ExecutionStatusPending
	}

	// 根据边构建依赖关系
	if execCtx.Workflow.Edges != nil {
		for _, edge := range execCtx.Workflow.Edges {
			// 检查边是否启用
			if !edge.IsEnabled() {
				continue
			}

			toNodeID := edge.ToNodeID
			fromNodeID := edge.FromNodeID

			// 检查节点是否存在
			if _, exists := execCtx.Workflow.Nodes[toNodeID]; !exists {
				log.Printf("Warning: Edge references non-existent node: %s", toNodeID)
				continue
			}
			if _, exists := execCtx.Workflow.Nodes[fromNodeID]; !exists {
				log.Printf("Warning: Edge references non-existent node: %s", fromNodeID)
				continue
			}

			// 添加依赖关系
			execCtx.NodeDependencies[toNodeID] = append(execCtx.NodeDependencies[toNodeID], fromNodeID)
		}
	}

	log.Printf("Built dependency graph: %+v", execCtx.NodeDependencies)
	return nil
}

// findStartNodes 找到起始节点
func (e *WorkflowEngine) findStartNodes(execCtx *ExecutionContext) []string {
	var startNodes []string

	for nodeID, dependencies := range execCtx.NodeDependencies {
		if len(dependencies) == 0 {
			startNodes = append(startNodes, nodeID)
		}
	}

	log.Printf("Found start nodes: %v", startNodes)
	return startNodes
}

// executeDAG 执行DAG
func (e *WorkflowEngine) executeDAG(execCtx *ExecutionContext) error {
	var wg sync.WaitGroup
	errorChan := make(chan error, len(execCtx.Workflow.Nodes))

	// 启动节点执行协程
	for i := 0; i < 3; i++ { // 最多3个并发执行节点
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.nodeExecutorWorker(execCtx, errorChan)
		}()
	}

	// 等待所有节点执行完成
	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// 检查执行结果
	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	log.Printf("Workflow execution completed: %s", execCtx.ExecutionID)
	return nil
}

// nodeExecutorWorker 节点执行工作协程
func (e *WorkflowEngine) nodeExecutorWorker(execCtx *ExecutionContext, errorChan chan<- error) {
	for {
		select {
		case <-execCtx.ctx.Done():
			return
		case nodeID, ok := <-execCtx.ReadyNodes:
			if !ok {
				return // 通道已关闭
			}

			// 检查节点是否已经在执行
			execCtx.mu.Lock()
			if execCtx.ExecutingNodes[nodeID] {
				execCtx.mu.Unlock()
				continue
			}
			execCtx.ExecutingNodes[nodeID] = true
			execCtx.mu.Unlock()

			// 执行节点
			node := execCtx.Workflow.Nodes[nodeID]
			if err := e.executeNode(execCtx, nodeID, node); err != nil {
				log.Printf("Node execution failed: %s, error: %v", nodeID, err)

				// 更新节点状态为失败
				execCtx.mu.Lock()
				execCtx.NodeStates[nodeID] = models.ExecutionStatusFailed
				delete(execCtx.ExecutingNodes, nodeID)
				execCtx.mu.Unlock()

				errorChan <- fmt.Errorf("node %s failed: %w", nodeID, err)
				return
			}

			// 节点执行成功
			execCtx.mu.Lock()
			execCtx.NodeStates[nodeID] = models.ExecutionStatusCompleted
			execCtx.CompletedNodes[nodeID] = true
			delete(execCtx.ExecutingNodes, nodeID)
			execCtx.mu.Unlock()

			// 检查并激活下游节点
			e.activateDownstreamNodes(execCtx, nodeID)
		}
	}
}

// executeNode 执行节点 - 重构为使用节点插件系统
func (e *WorkflowEngine) executeNode(execCtx *ExecutionContext, nodeID string, node *models.Node) error {
	log.Printf("Executing node: %s (type: %s, plugin: %s)", nodeID, node.Type, node.Plugin)

	// 更新节点状态为运行中
	execCtx.mu.Lock()
	execCtx.NodeStates[nodeID] = models.ExecutionStatusRunning
	execCtx.mu.Unlock()

	// 获取节点执行超时时间
	timeout := node.GetExecutionTimeout()
	if timeout == 0 {
		timeout = time.Second * 30 // 默认30秒超时
	}

	// 创建节点执行上下文
	nodeCtx, cancel := context.WithTimeout(execCtx.ctx, timeout)
	defer cancel()

	// 准备输入数据
	inputData := e.prepareNodeInput(execCtx, nodeID, node)

	// 通过节点注册系统获取节点插件
	nodePlugin, err := e.nodeRegistry.Get(node.Plugin)
	if err != nil {
		return fmt.Errorf("failed to get node plugin %s: %w", node.Plugin, err)
	}

	// 准备节点输入
	nodeInput := &nodes.NodeInput{
		Data:      inputData,
		Config:    node.Config.PluginConfig,
		Context:   make(map[string]interface{}),
		Variables: execCtx.Variables,
	}

	// 执行节点
	nodeOutput, err := nodePlugin.Execute(nodeCtx, nodeInput)
	if err != nil {
		return fmt.Errorf("node plugin execution failed: %w", err)
	}

	// 检查执行结果
	if nodeOutput == nil {
		return fmt.Errorf("node plugin returned nil output")
	}

	if !nodeOutput.Success {
		return fmt.Errorf("node execution failed: %s", nodeOutput.Error)
	}

	// 处理输出数据
	e.processNodeOutput(execCtx, nodeID, nodeOutput.Data)

	log.Printf("Node %s executed successfully", nodeID)
	return nil
}

// activateDownstreamNodes 激活下游节点
func (e *WorkflowEngine) activateDownstreamNodes(execCtx *ExecutionContext, completedNodeID string) {
	if execCtx.Workflow.Edges == nil {
		return
	}

	for _, edge := range execCtx.Workflow.Edges {
		if edge.FromNodeID != completedNodeID || !edge.IsEnabled() {
			continue
		}

		toNodeID := edge.ToNodeID

		// 检查是否为条件边
		if edge.IsConditional() {
			shouldExecute, err := e.evaluateEdgeCondition(execCtx, edge)
			if err != nil {
				log.Printf("Failed to evaluate edge condition: %v", err)
				continue
			}
			if !shouldExecute {
				log.Printf("Edge condition not met, skipping node: %s", toNodeID)
				continue
			}
		}

		// 检查目标节点的所有依赖是否都已完成
		if e.areAllDependenciesCompleted(execCtx, toNodeID) {
			// 将节点加入准备队列
			select {
			case execCtx.ReadyNodes <- toNodeID:
				log.Printf("Node %s is ready for execution", toNodeID)
			case <-execCtx.ctx.Done():
				return
			default:
				// 通道满了，忽略（节点可能已经在队列中）
			}
		}
	}
}

// areAllDependenciesCompleted 检查所有依赖是否都已完成
func (e *WorkflowEngine) areAllDependenciesCompleted(execCtx *ExecutionContext, nodeID string) bool {
	execCtx.mu.RLock()
	defer execCtx.mu.RUnlock()

	dependencies := execCtx.NodeDependencies[nodeID]
	for _, depNodeID := range dependencies {
		if !execCtx.CompletedNodes[depNodeID] {
			return false
		}
	}
	return true
}

// evaluateEdgeCondition 评估边条件
func (e *WorkflowEngine) evaluateEdgeCondition(execCtx *ExecutionContext, edge *models.Edge) (bool, error) {
	if !edge.IsConditional() {
		return true, nil
	}

	// 使用当前的执行变量作为上下文
	execCtx.mu.RLock()
	context := make(map[string]interface{})
	for k, v := range execCtx.Variables {
		context[k] = v
	}
	execCtx.mu.RUnlock()

	// 调用边的条件评估方法
	return edge.EvaluateCondition(context)
}

// prepareNodeInput 准备节点输入数据
func (e *WorkflowEngine) prepareNodeInput(execCtx *ExecutionContext, nodeID string, node *models.Node) map[string]interface{} {
	execCtx.mu.RLock()
	defer execCtx.mu.RUnlock()

	inputData := make(map[string]interface{})

	// 复制全局变量
	for k, v := range execCtx.Variables {
		inputData[k] = v
	}

	// 应用输入映射
	if node.Config != nil && node.Config.InputConfig != nil {
		inputConfig := node.Config.InputConfig

		// 应用映射
		if inputConfig.Mapping != nil {
			for targetKey, sourceKey := range inputConfig.Mapping {
				if value, exists := execCtx.Variables[sourceKey]; exists {
					inputData[targetKey] = value
				}
			}
		}

		// 应用默认值
		if inputConfig.Defaults != nil {
			for key, defaultValue := range inputConfig.Defaults {
				if _, exists := inputData[key]; !exists {
					inputData[key] = defaultValue
				}
			}
		}
	}

	return inputData
}

// processNodeOutput 处理节点输出数据
func (e *WorkflowEngine) processNodeOutput(execCtx *ExecutionContext, nodeID string, outputData map[string]interface{}) {
	if outputData == nil {
		return
	}

	execCtx.mu.Lock()
	defer execCtx.mu.Unlock()

	// 将输出数据添加到全局变量中
	for key, value := range outputData {
		// 使用节点ID作为前缀避免冲突
		globalKey := fmt.Sprintf("%s_%s", nodeID, key)
		execCtx.Variables[globalKey] = value
	}

	// 同时保存原始输出
	execCtx.Variables[fmt.Sprintf("%s_output", nodeID)] = outputData
}

// GetActiveExecutions 获取活跃的执行列表
func (e *WorkflowEngine) GetActiveExecutions() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	executions := make([]string, 0, len(e.executions))
	for id := range e.executions {
		executions = append(executions, id)
	}
	return executions
}

// GetStatus 获取引擎状态
func (e *WorkflowEngine) GetStatus() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.status
}

// 全局引擎实例
var GlobalEngine *WorkflowEngine
