/**
 * @module nodes_init
 * @description 节点系统初始化，注册所有内置节点并提供系统启动配置
 * @architecture 初始化模块，负责节点注册和系统配置
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow init_states: starting -> registering -> completed
 * @rules 必须在系统启动时调用，确保所有内置节点正确注册
 * @dependencies context, log
 * @refs service/nodes/registry.go, service/nodes/datasource/, service/nodes/transform/
 */

package nodes

import (
	"log"
)

// InitializeNodes 初始化所有内置节点
// 注意：具体的节点注册将在各个子包的init函数中完成，避免循环导入
func InitializeNodes() error {
	log.Println("开始初始化内置节点...")

	registry := GetRegistry()

	// 此时节点应该已经通过各个子包的init函数注册完成
	// 我们只需要验证注册状态
	nodeCount := registry.Count()
	if nodeCount == 0 {
		log.Println("警告：没有发现任何已注册的节点")
	} else {
		log.Printf("节点初始化完成，共注册 %d 个节点", nodeCount)
	}

	return nil
}

// GetAvailableNodes 获取可用节点列表
func GetAvailableNodes() []string {
	return GetRegistry().List()
}

// GetNodesByCategory 按分类获取节点
func GetNodesByCategory(category string) []*NodeMetadata {
	return GetRegistry().ListByCategory(category)
}

// GetNodesByType 按类型获取节点
func GetNodesByType(nodeType string) []*NodeMetadata {
	return GetRegistry().ListByType(nodeType)
}

// SearchNodes 搜索节点
func SearchNodes(query string) []*NodeMetadata {
	return GetRegistry().Search(query)
}

// GetNodeCount 获取注册节点数量
func GetNodeCount() int {
	return GetRegistry().Count()
}

// IsNodeRegistered 检查节点是否已注册
func IsNodeRegistered(nodeID string) bool {
	return GetRegistry().IsRegistered(nodeID)
}

// GetNodePlugin 获取节点插件
func GetNodePlugin(nodeID string) (NodePlugin, error) {
	return GetRegistry().Get(nodeID)
}

// ValidateNodeConfig 验证节点配置
func ValidateNodeConfig(nodeID string, config map[string]interface{}) error {
	plugin, err := GetRegistry().Get(nodeID)
	if err != nil {
		return err
	}
	return plugin.Validate(config)
}
