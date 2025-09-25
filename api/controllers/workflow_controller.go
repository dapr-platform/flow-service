/**
 * @module workflow_controller
 * @description 统一工作流控制器，合并原 DAG、Task、Instance 控制器功能，提供完整的工作流和执行管理
 * @architecture 统一控制器设计，简化API层架构，提供RESTful接口
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow workflow_states: inactive -> active -> paused -> disabled
 * @rules 统一错误处理，标准化响应格式，支持完整的工作流生命周期管理
 * @dependencies service/workflow_service.go, service/execution_service.go, api/controllers/response.go
 * @refs api/routes.go
 */

package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"

	"flow-service/service"
	"flow-service/service/models"
)

// WorkflowController 统一工作流控制器
type WorkflowController struct {
	workflowService  *service.WorkflowService
	executionService *service.ExecutionService
}

// NewWorkflowController 创建工作流控制器实例
func NewWorkflowController() *WorkflowController {
	return &WorkflowController{
		workflowService:  service.GlobalWorkflowService,
		executionService: service.GlobalExecutionService,
	}
}

// ==================== 工作流管理 ====================

// CreateWorkflow 创建工作流
// @Summary 创建工作流
// @Description 创建新的工作流定义
// @Tags workflows
// @Accept json
// @Produce json
// @Param workflow body models.Workflow true "工作流定义"
// @Success 200 {object} APIResponse{data=models.Workflow}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows [post]
func (c *WorkflowController) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var workflow models.Workflow
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "无效的JSON格式", err))
		return
	}

	// 生成ID
	if workflow.ID == "" {
		workflow.ID = uuid.New().String()
	}

	// 创建工作流
	if err := c.workflowService.CreateWorkflow(&workflow); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "创建工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流创建成功", workflow))
}

// GetWorkflow 获取工作流
// @Summary 获取工作流
// @Description 根据ID获取工作流详情
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse{data=models.Workflow}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id} [get]
func (c *WorkflowController) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	workflow, err := c.workflowService.GetWorkflow(id)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusNotFound, "工作流不存在", err))
		return
	}

	render.Render(w, r, SuccessResponse("获取工作流成功", workflow))
}

// UpdateWorkflow 更新工作流
// @Summary 更新工作流
// @Description 更新工作流定义
// @Tags workflows
// @Accept json
// @Produce json
// @Param id path string true "工作流ID"
// @Param workflow body models.Workflow true "工作流定义"
// @Success 200 {object} APIResponse{data=models.Workflow}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id} [put]
func (c *WorkflowController) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	var workflow models.Workflow
	if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "无效的JSON格式", err))
		return
	}

	workflow.ID = id
	if err := c.workflowService.UpdateWorkflow(&workflow); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "更新工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流更新成功", workflow))
}

// DeleteWorkflow 删除工作流
// @Summary 删除工作流
// @Description 删除指定的工作流
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id} [delete]
func (c *WorkflowController) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	if err := c.workflowService.DeleteWorkflow(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "删除工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流删除成功", nil))
}

// ListWorkflows 列出工作流
// @Summary 列出工作流
// @Description 分页列出工作流
// @Tags workflows
// @Produce json
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页大小，默认10"
// @Param status query string false "状态筛选"
// @Success 200 {object} PaginatedResponse
// @Failure 500 {object} APIResponse
// @Router /workflows [get]
func (c *WorkflowController) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	statusStr := r.URL.Query().Get("status")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	status := models.WorkflowStatus(statusStr)

	workflows, total, err := c.workflowService.ListWorkflows(offset, pageSize, status)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "获取工作流列表失败", err))
		return
	}

	render.Render(w, r, PaginatedSuccessResponse("获取工作流列表成功", workflows, total, page, pageSize))
}

// ==================== 工作流状态控制 ====================

// ActivateWorkflow 激活工作流
// @Summary 激活工作流
// @Description 激活指定的工作流
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id}/activate [post]
func (c *WorkflowController) ActivateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	if err := c.workflowService.ActivateWorkflow(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "激活工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流激活成功", nil))
}

// DeactivateWorkflow 停用工作流
// @Summary 停用工作流
// @Description 停用指定的工作流
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id}/deactivate [post]
func (c *WorkflowController) DeactivateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	if err := c.workflowService.DeactivateWorkflow(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "停用工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流停用成功", nil))
}

// PauseWorkflow 暂停工作流
// @Summary 暂停工作流
// @Description 暂停指定的工作流
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id}/pause [post]
func (c *WorkflowController) PauseWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	if err := c.workflowService.PauseWorkflow(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "暂停工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流暂停成功", nil))
}

// ResumeWorkflow 恢复工作流
// @Summary 恢复工作流
// @Description 恢复指定的工作流
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id}/resume [post]
func (c *WorkflowController) ResumeWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	if err := c.workflowService.ResumeWorkflow(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "恢复工作流失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("工作流恢复成功", nil))
}

// ==================== 执行管理 ====================

// TriggerExecution 触发执行
// @Summary 触发工作流执行
// @Description 手动触发工作流执行
// @Tags executions
// @Accept json
// @Produce json
// @Param id path string true "工作流ID"
// @Param request body TriggerExecutionRequest true "触发执行请求"
// @Success 200 {object} APIResponse{data=models.Execution}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id}/trigger [post]
func (c *WorkflowController) TriggerExecution(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "id")
	if workflowID == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	var request TriggerExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "无效的JSON格式", err))
		return
	}

	// 检查工作流是否存在
	workflow, err := c.workflowService.GetWorkflow(workflowID)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusNotFound, "工作流不存在", err))
		return
	}

	// 创建执行记录
	execution := &models.Execution{
		ID:          uuid.New().String(),
		WorkflowID:  workflowID,
		WorkflowVer: workflow.Version,
		Name:        request.Name,
		Description: request.Description,
		Status:      models.ExecutionStatusPending,
		TriggerType: models.TriggerTypeManual,
		TriggerBy:   request.TriggerBy,
		Context: &models.ExecutionContext{
			Variables: request.Variables,
			Input:     request.Input,
		},
		Priority: request.Priority,
	}

	if err := c.executionService.CreateExecution(execution); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "创建执行记录失败", err))
		return
	}

	// 开始执行
	if err := c.executionService.StartExecution(execution.ID); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "启动执行失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("执行触发成功", execution))
}

// GetExecution 获取执行记录
// @Summary 获取执行记录
// @Description 根据ID获取执行记录详情
// @Tags executions
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} APIResponse{data=models.Execution}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /executions/{id} [get]
func (c *WorkflowController) GetExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "执行ID不能为空", nil))
		return
	}

	execution, err := c.executionService.GetExecution(id)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusNotFound, "执行记录不存在", err))
		return
	}

	render.Render(w, r, SuccessResponse("获取执行记录成功", execution))
}

// ListExecutions 列出执行记录
// @Summary 列出执行记录
// @Description 分页列出执行记录
// @Tags executions
// @Produce json
// @Param workflow_id query string false "工作流ID筛选"
// @Param status query string false "状态筛选"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页大小，默认10"
// @Success 200 {object} PaginatedResponse
// @Failure 500 {object} APIResponse
// @Router /executions [get]
func (c *WorkflowController) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	workflowID := r.URL.Query().Get("workflow_id")
	statusStr := r.URL.Query().Get("status")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	status := models.ExecutionStatus(statusStr)

	executions, total, err := c.executionService.ListExecutions(workflowID, status, offset, pageSize)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "获取执行记录列表失败", err))
		return
	}

	render.Render(w, r, PaginatedSuccessResponse("获取执行记录列表成功", executions, total, page, pageSize))
}

// CancelExecution 取消执行
// @Summary 取消执行
// @Description 取消指定的执行
// @Tags executions
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /executions/{id}/cancel [post]
func (c *WorkflowController) CancelExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "执行ID不能为空", nil))
		return
	}

	if err := c.executionService.CancelExecution(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "取消执行失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("执行取消成功", nil))
}

// RetryExecution 重试执行
// @Summary 重试执行
// @Description 重试失败的执行
// @Tags executions
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} APIResponse{data=models.Execution}
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /executions/{id}/retry [post]
func (c *WorkflowController) RetryExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "执行ID不能为空", nil))
		return
	}

	if err := c.executionService.RetryExecution(id); err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "重试执行失败", err))
		return
	}

	// 获取更新后的执行记录
	execution, err := c.executionService.GetExecution(id)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "重试后获取执行记录失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("执行重试成功", execution))
}

// GetExecutionProgress 获取执行进度
// @Summary 获取执行进度
// @Description 获取执行的当前进度
// @Tags executions
// @Produce json
// @Param id path string true "执行ID"
// @Success 200 {object} APIResponse{data=ProgressResponse}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /executions/{id}/progress [get]
func (c *WorkflowController) GetExecutionProgress(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "执行ID不能为空", nil))
		return
	}

	progress, err := c.executionService.GetExecutionProgress(id)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "获取执行进度失败", err))
		return
	}

	progressResponse := ProgressResponse{
		ExecutionID: id,
		Progress:    progress,
		UpdatedAt:   time.Now(),
	}

	render.Render(w, r, SuccessResponse("获取执行进度成功", progressResponse))
}

// GetWorkflowStatistics 获取工作流统计信息
// @Summary 获取工作流统计信息
// @Description 获取工作流的统计信息
// @Tags workflows
// @Produce json
// @Param id path string true "工作流ID"
// @Success 200 {object} APIResponse{data=models.WorkflowStatistics}
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /workflows/{id}/statistics [get]
func (c *WorkflowController) GetWorkflowStatistics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "工作流ID不能为空", nil))
		return
	}

	statistics, err := c.workflowService.GetWorkflowStatistics(id)
	if err != nil {
		render.Render(w, r, ErrorResponse(http.StatusInternalServerError, "获取工作流统计信息失败", err))
		return
	}

	render.Render(w, r, SuccessResponse("获取工作流统计信息成功", statistics))
}

// ==================== 请求和响应结构体 ====================

// TriggerExecutionRequest 触发执行请求
type TriggerExecutionRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	TriggerBy   string                 `json:"trigger_by,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
}

// ProgressResponse 进度响应
type ProgressResponse struct {
	ExecutionID string    `json:"execution_id"`
	Progress    float64   `json:"progress"`
	UpdatedAt   time.Time `json:"updated_at"`
}
