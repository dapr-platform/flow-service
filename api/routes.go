/*
 * @module: flow-service/api/routes
 * @description: 统一流程服务路由配置，使用简化的两层架构（Workflow、Execution）
 * @architecture: RESTful API架构，基于Chi路由框架，统一工作流管理
 * @documentReference: ai_docs/refactor_plan.md
 * @stateFlow: 无状态路由配置，支持工作流生命周期管理
 * @rules:
 *   - 统一响应格式和错误处理
 *   - 支持CORS跨域请求
 *   - 集成请求日志和错误处理中间件
 *   - 简化路由结构，减少冗余
 * @dependencies:
 *   - Chi路由框架
 *   - CORS中间件
 *   - WorkflowController统一控制器
 * @refs:
 *   - api/controllers/workflow_controller.go
 *   - api/controllers/health.go
 */

package api

import (
	"net/http"

	"flow-service/api/controllers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// InitRoute 初始化路由配置
func InitRoute(r chi.Router) {
	// 基础中间件
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// CORS配置
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 创建控制器实例
	healthController := controllers.NewHealthController()
	workflowController := controllers.NewWorkflowController()
	nodeController := controllers.NewNodeController()

	// 基础健康检查路由
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 系统监控路由
	r.Get("/ready", healthController.GetReadiness)
	r.Get("/live", healthController.GetLiveness)
	r.Get("/info", healthController.GetSystemInfo)
	r.Get("/metrics", healthController.GetMetrics)
	r.Get("/version", healthController.GetVersion)

	// ==================== 统一工作流管理路由 ====================

	// 工作流管理路由
	r.Route("/workflows", func(r chi.Router) {
		// 工作流 CRUD
		r.Post("/", workflowController.CreateWorkflow)
		r.Get("/", workflowController.ListWorkflows)
		r.Get("/{id}", workflowController.GetWorkflow)
		r.Put("/{id}", workflowController.UpdateWorkflow)
		r.Delete("/{id}", workflowController.DeleteWorkflow)

		// 工作流状态控制
		r.Post("/{id}/activate", workflowController.ActivateWorkflow)
		r.Post("/{id}/deactivate", workflowController.DeactivateWorkflow)
		r.Post("/{id}/pause", workflowController.PauseWorkflow)
		r.Post("/{id}/resume", workflowController.ResumeWorkflow)

		// 工作流执行管理
		r.Post("/{id}/trigger", workflowController.TriggerExecution)
		r.Get("/{id}/executions", workflowController.ListExecutions)
		r.Get("/{id}/statistics", workflowController.GetWorkflowStatistics)
	})

	// 执行记录管理路由
	r.Route("/executions", func(r chi.Router) {
		r.Get("/", workflowController.ListExecutions)
		r.Get("/{id}", workflowController.GetExecution)
		r.Post("/{id}/cancel", workflowController.CancelExecution)
		r.Post("/{id}/retry", workflowController.RetryExecution)
		r.Get("/{id}/progress", workflowController.GetExecutionProgress)
	})

	// 节点管理路由
	r.Route("/nodes", func(r chi.Router) {
		r.Get("/", nodeController.GetNodes)
		r.Post("/{id}/validate", nodeController.ValidateNodeConfig)
		r.Post("/{id}/dynamic-data", nodeController.GetDynamicData)
	})

}
