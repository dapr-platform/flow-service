/*
 * @module service/database/migrate
 * @description 数据库迁移模块，负责创建和更新数据库表结构
 * @architecture 数据访问层 - 迁移管理
 * @documentReference docs/database-design.md
 * @stateFlow 应用启动时执行数据库迁移
 * @rules 确保数据库结构与模型定义保持一致
 * @dependencies flow-service/service/models, gorm.io/gorm
 * @refs service/models/, service/database/database.go
 */

package database

import (
	"flow-service/service/models"
	"log"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	log.Println("开始数据库迁移...")

	err := db.AutoMigrate(
		&models.Workflow{},
		&models.Execution{},
	)
	if err != nil {
		return err
	}

	log.Println("数据库迁移完成")
	return nil
}

// InitializeData 初始化基础数据
func InitializeData(db *gorm.DB) error {
	log.Println("开始初始化基础数据...")

	// 工作流状态类型
	workflowStatuses := []string{
		"inactive", // 未激活
		"active",   // 激活
		"paused",   // 暂停
		"disabled", // 禁用
	}

	// 执行状态类型
	executionStatuses := []string{
		"pending",   // 待执行
		"running",   // 运行中
		"completed", // 已完成
		"failed",    // 失败
		"cancelled", // 已取消
	}

	// 节点类型（存储在Workflow的JSON中）
	nodeTypes := []string{
		"datasource", "transform", "output", "control",
		"condition", "loop", "subdag", "script", "api", "timer",
	}

	// 边类型（存储在Workflow的JSON中）
	edgeTypes := []string{
		"normal", "conditional", "loop", "error", "timeout", "skip",
	}

	log.Printf("支持的工作流状态: %v", workflowStatuses)
	log.Printf("支持的执行状态: %v", executionStatuses)
	log.Printf("支持的节点类型: %v", nodeTypes)
	log.Printf("支持的边类型: %v", edgeTypes)

	log.Println("基础数据初始化完成")
	return nil
}

// CreateIndexes 创建数据库索引
func CreateIndexes(db *gorm.DB) error {
	log.Println("开始创建数据库索引...")

	// Workflow相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status)").Error; err != nil {
		log.Printf("创建Workflow状态索引失败: %v", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_workflows_created_at ON workflows(created_at)").Error; err != nil {
		log.Printf("创建Workflow创建时间索引失败: %v", err)
	}

	// Node和Edge数据存储在Workflow的JSON字段中，无需独立索引
	// 如果需要查询特定节点或边，可以通过Workflow查询后在应用层处理

	// Execution相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id)").Error; err != nil {
		log.Printf("创建Execution WorkflowID索引失败: %v", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status)").Error; err != nil {
		log.Printf("创建Execution状态索引失败: %v", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_executions_started_at ON executions(started_at)").Error; err != nil {
		log.Printf("创建Execution开始时间索引失败: %v", err)
	}

	log.Println("数据库索引创建完成")
	return nil
}
