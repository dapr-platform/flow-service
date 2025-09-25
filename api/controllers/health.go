/*
 * @module: flow-service/api/controllers/health
 * @description: 健康检查和系统监控控制器，提供服务健康状态、系统信息和监控指标
 * @architecture: MVC架构，控制器层负责处理健康检查和监控相关的HTTP请求
 * @documentReference: /docs/api-health-controller.md
 * @stateFlow: 无状态，实时查询系统状态
 * @rules: 健康检查接口必须快速响应，系统信息不包含敏感数据，监控指标格式标准化
 * @dependencies: controllers/response.go
 * @refs: api/controllers/
 */

package controllers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/render"
)

// HealthController 健康检查控制器
type HealthController struct{}

// NewHealthController 创建健康检查控制器
func NewHealthController() *HealthController {
	return &HealthController{}
}

// HealthCheck 健康检查接口
// @Summary 健康检查
// @Description 检查服务健康状态
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}} "健康状态"
// @Router /health [get]
func (hc *HealthController) HealthCheck(w http.ResponseWriter, r *http.Request) {
	healthStatus := map[string]interface{}{
		"status":    "healthy",
		"version":   "2.0-refactored",
		"timestamp": time.Now().Unix(),
		"service":   "flow-service",
		"arch":      "simplified-2-layer",
	}

	response := SuccessResponse("健康检查成功", healthStatus)
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}

// GetSystemInfo 获取系统信息
// @Summary 获取系统信息
// @Description 获取服务的基本信息和运行状态
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}} "系统信息"
// @Router /info [get]
func (hc *HealthController) GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	systemInfo := map[string]interface{}{
		"service": map[string]interface{}{
			"name":    "flow-service",
			"version": "2.0-refactored",
			"arch":    "simplified-2-layer",
		},
		"runtime": map[string]interface{}{
			"goroutines":   runtime.NumGoroutine(),
			"memory_alloc": memStats.Alloc,
			"memory_total": memStats.TotalAlloc,
			"memory_sys":   memStats.Sys,
			"gc_runs":      memStats.NumGC,
			"last_gc_time": time.Unix(0, int64(memStats.LastGC)).Format("2006-01-02 15:04:05"),
		},
	}

	response := SuccessResponse("获取系统信息成功", systemInfo)
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}

// GetVersion 获取版本信息
// @Summary 获取版本信息
// @Description 获取服务的版本和构建信息
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}} "版本信息"
// @Router /version [get]
func (hc *HealthController) GetVersion(w http.ResponseWriter, r *http.Request) {
	versionInfo := map[string]interface{}{
		"name":         "flow-service",
		"version":      "2.0-refactored",
		"architecture": "simplified-2-layer",
		"build_time":   "2024-12-06",
		"go_version":   runtime.Version(),
		"models":       []string{"Workflow", "Execution", "Node", "Edge"},
		"components":   []string{"SimpleScheduler", "SimpleEngine", "StateManager"},
	}

	response := SuccessResponse("获取版本信息成功", versionInfo)
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}

// GetReadiness 就绪检查
// @Summary 就绪检查
// @Description 检查服务是否准备好接收请求
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}} "服务就绪"
// @Router /ready [get]
func (hc *HealthController) GetReadiness(w http.ResponseWriter, r *http.Request) {
	readinessStatus := map[string]interface{}{
		"ready":     true,
		"timestamp": time.Now().Unix(),
		"service":   "flow-service",
		"version":   "2.0-refactored",
	}

	response := SuccessResponse("服务已就绪", readinessStatus)
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}

// GetLiveness 存活检查
// @Summary 存活检查
// @Description 检查服务是否存活
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}} "服务存活"
// @Router /live [get]
func (hc *HealthController) GetLiveness(w http.ResponseWriter, r *http.Request) {
	livenessStatus := map[string]interface{}{
		"alive":     true,
		"timestamp": time.Now().Unix(),
		"pid":       runtime.GOMAXPROCS(0),
	}

	response := SuccessResponse("服务存活", livenessStatus)
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}

// GetMetrics 获取监控指标
// @Summary 获取监控指标
// @Description 获取服务的性能指标和统计信息
// @Tags 系统监控
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse{data=map[string]interface{}} "监控指标"
// @Router /metrics [get]
func (hc *HealthController) GetMetrics(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"service": map[string]interface{}{
			"name":    "flow-service",
			"version": "2.0-refactored",
		},
		"runtime": map[string]interface{}{
			"goroutines":      runtime.NumGoroutine(),
			"memory_alloc_mb": float64(memStats.Alloc) / 1024 / 1024,
			"memory_total_mb": float64(memStats.TotalAlloc) / 1024 / 1024,
			"memory_sys_mb":   float64(memStats.Sys) / 1024 / 1024,
			"gc_runs":         memStats.NumGC,
		},
		"system": map[string]interface{}{
			"cpu_count":  runtime.NumCPU(),
			"go_version": runtime.Version(),
			"go_os":      runtime.GOOS,
			"go_arch":    runtime.GOARCH,
		},
	}

	response := SuccessResponse("获取监控指标成功", metrics)
	render.Status(r, http.StatusOK)
	render.Render(w, r, response)
}
