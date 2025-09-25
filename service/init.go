/*
 * @module: flow-service/service/init
 * @description: 流程服务的业务逻辑层初始化，负责核心组件的启动和配置
 * @architecture: 分层架构，服务层负责业务逻辑处理和组件协调
 * @documentReference: /docs/flow-service-service-layer.md
 * @stateFlow: 无
 * @rules:
 *   - 服务启动时进行必要的初始化操作
 *   - 按依赖顺序初始化各个组件
 *   - 提供优雅的错误处理和回滚机制
 * @dependencies:
 *   - 配置管理系统
 *   - 日志框架
 * @refs:
 *   - pkg/framework/logger.go
 *   - service/config/
 */

package service

import (
	"context"
	"fmt"
	"log"

	"flow-service/service/database"
)

var GlobalWorkflowService *WorkflowService
var GlobalExecutionService *ExecutionService

func init() {
	err := initDatabase()
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	err = initServices()
	if err != nil {
		log.Fatalf("服务初始化失败: %v", err)
	}
}

func initDatabase() error {
	// 初始化数据库连接
	if err := database.InitDatabase(); err != nil {
		return fmt.Errorf("数据库连接初始化失败: %w", err)
	}

	// 执行数据库迁移
	db := database.GetDB()
	if err := database.AutoMigrate(db); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 初始化基础数据
	if err := database.InitializeData(db); err != nil {
		return fmt.Errorf("基础数据初始化失败: %w", err)
	}

	// 创建数据库索引
	if err := database.CreateIndexes(db); err != nil {
		return fmt.Errorf("数据库索引创建失败: %w", err)
	}

	log.Println("数据库初始化完成")
	return nil
}

func initServices() error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 初始化全局组件实例（确保在使用前初始化）
	if GlobalEngine == nil {
		GlobalEngine = NewWorkflowEngine()
	}
	if GlobalSimpleScheduler == nil {
		GlobalSimpleScheduler = NewSimpleScheduler()
	}

	// 启动简化组件
	if err := GlobalEngine.Start(context.Background()); err != nil {
		log.Printf("启动执行引擎失败: %v", err)
	}

	if err := GlobalSimpleScheduler.Start(context.Background()); err != nil {
		log.Printf("启动调度器失败: %v", err)
	}

	// 创建服务实例，使用简化组件
	GlobalWorkflowService = NewWorkflowService(db, GlobalSimpleScheduler)
	GlobalExecutionService = NewExecutionService(db, GlobalWorkflowService, GlobalEngine)

	log.Println("服务初始化完成")
	return nil
}
