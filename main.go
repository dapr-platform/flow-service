/*
 * @module: flow-service
 * @description: 流程服务的主入口，提供统一的工作流编排和执行功能
 * @architecture: 基于Dapr的微服务架构，使用简化的两层架构（Workflow、Execution）
 * @documentReference: ai_docs/refactor_plan.md
 * @stateFlow: 无
 * @rules:
 *   - 使用Dapr HTTP服务模式
 *   - 支持Prometheus监控
 *   - 集成Swagger文档
 *   - 统一服务初始化和依赖注入
 * @dependencies:
 *   - Dapr sidecar
 *   - Chi路由框架
 *   - Swagger文档系统
 *   - 统一服务层
 * @refs:
 *   - service/workflow_service.go
 *   - service/execution_service.go
 */

package main

import (
	"flow-service/api"
	_ "flow-service/docs"
	"log"
	"net/http"
	"os"
	"strconv"

	// 导入节点包以触发init函数
	_ "flow-service/service/nodes/datasource"
	_ "flow-service/service/nodes/output"
	_ "flow-service/service/nodes/transform"

	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

var (
	PORT         = 80
	BASE_CONTEXT = ""
)

func init() {
	if val := os.Getenv("LISTEN_PORT"); val != "" {
		PORT, _ = strconv.Atoi(val)
	}

	if val := os.Getenv("BASE_CONTEXT"); val != "" {
		BASE_CONTEXT = val
	}
}

// @title 流程服务 API
// @version 1.0
// @description 流程服务，提供流程编排、执行、调度功能
// @BasePath /swagger/flow-service
func main() {

	mux := chi.NewRouter()

	// 如果有BASE_CONTEXT，则在该路径下挂载所有路由
	if BASE_CONTEXT != "" {
		log.Println("BASE_CONTEXT", BASE_CONTEXT)
		mux.Route(BASE_CONTEXT, func(r chi.Router) {
			// 创建子路由器并初始化路由
			subMux := r.(*chi.Mux)
			api.InitRoute(subMux)
			r.Handle("/metrics", promhttp.Handler())
			r.Handle("/swagger*", httpSwagger.WrapHandler)
		})
	} else {
		api.InitRoute(mux)
		mux.Handle("/metrics", promhttp.Handler())
		mux.Handle("/swagger*", httpSwagger.WrapHandler)
	}

	s := daprd.NewServiceWithMux(":"+strconv.Itoa(PORT), mux)
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error: %v", err)
	}
}
