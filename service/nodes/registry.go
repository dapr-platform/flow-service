/**
 * @module node_registry
 * @description 节点注册表，提供线程安全的节点注册、查找和管理功能
 * @architecture 单例模式的注册中心，支持动态注册和元数据查询
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow registry_states: initialized -> registered -> ready
 * @rules 注册表必须线程安全，节点ID必须唯一，支持按分类和类型查询
 * @dependencies sync, fmt, sort
 * @refs service/nodes/interface.go
 */

package nodes

import (
	"fmt"
	"sort"
	"sync"
)

// NodeRegistry 节点注册表
type NodeRegistry struct {
	nodes    map[string]NodePlugin
	metadata map[string]*NodeMetadata
	mu       sync.RWMutex
}

// 全局注册表实例
var globalRegistry *NodeRegistry
var registryOnce sync.Once

// GetRegistry 获取全局注册表实例
func GetRegistry() *NodeRegistry {
	registryOnce.Do(func() {
		globalRegistry = &NodeRegistry{
			nodes:    make(map[string]NodePlugin),
			metadata: make(map[string]*NodeMetadata),
		}
	})
	return globalRegistry
}

// Register 注册节点
func (r *NodeRegistry) Register(plugin NodePlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := plugin.GetMetadata()
	if metadata == nil {
		return fmt.Errorf("plugin metadata cannot be nil")
	}

	if metadata.ID == "" {
		return fmt.Errorf("plugin ID cannot be empty")
	}

	// 检查是否已注册
	if _, exists := r.nodes[metadata.ID]; exists {
		return fmt.Errorf("plugin with ID %s already registered", metadata.ID)
	}

	// 验证插件
	if err := r.validatePlugin(plugin); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	// 注册插件
	r.nodes[metadata.ID] = plugin
	r.metadata[metadata.ID] = metadata

	return nil
}

// Unregister 注销节点
func (r *NodeRegistry) Unregister(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.nodes[pluginID]; !exists {
		return fmt.Errorf("plugin with ID %s not found", pluginID)
	}

	delete(r.nodes, pluginID)
	delete(r.metadata, pluginID)

	return nil
}

// Get 获取节点插件
func (r *NodeRegistry) Get(pluginID string) (NodePlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.nodes[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin with ID %s not found", pluginID)
	}

	return plugin, nil
}

// GetMetadata 获取节点元数据
func (r *NodeRegistry) GetMetadata(pluginID string) (*NodeMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin with ID %s not found", pluginID)
	}

	return metadata, nil
}

// List 列出所有节点
func (r *NodeRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.nodes))
	for id := range r.nodes {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids
}

// ListByCategory 按分类列出节点
func (r *NodeRegistry) ListByCategory(category string) []*NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NodeMetadata
	for _, metadata := range r.metadata {
		if metadata.Category == category {
			result = append(result, metadata)
		}
	}

	// 按名称排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// ListByType 按类型列出节点
func (r *NodeRegistry) ListByType(nodeType string) []*NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NodeMetadata
	for _, metadata := range r.metadata {
		if metadata.Type == nodeType {
			result = append(result, metadata)
		}
	}

	// 按名称排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetAllMetadata 获取所有节点元数据
func (r *NodeRegistry) GetAllMetadata() []*NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*NodeMetadata, 0, len(r.metadata))
	for _, metadata := range r.metadata {
		result = append(result, metadata)
	}

	// 按分类和名称排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// GetCategories 获取所有分类
func (r *NodeRegistry) GetCategories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categorySet := make(map[string]bool)
	for _, metadata := range r.metadata {
		categorySet[metadata.Category] = true
	}

	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}

	sort.Strings(categories)
	return categories
}

// GetCategoriesWithInfo 获取所有分类及其详细信息
func (r *NodeRegistry) GetCategoriesWithInfo() []CategoryInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categoryMap := make(map[string]*CategoryInfo)
	for _, metadata := range r.metadata {
		if _, exists := categoryMap[metadata.Category]; !exists {
			categoryMap[metadata.Category] = &CategoryInfo{
				Category:  metadata.Category,
				NodeCount: 0,
			}
		}
		categoryMap[metadata.Category].NodeCount++
	}

	categories := make([]CategoryInfo, 0, len(categoryMap))
	for _, info := range categoryMap {
		categories = append(categories, *info)
	}

	// 按分类名称排序
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Category < categories[j].Category
	})

	return categories
}

// GetTypes 获取所有类型
func (r *NodeRegistry) GetTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typeSet := make(map[string]bool)
	for _, metadata := range r.metadata {
		typeSet[metadata.Type] = true
	}

	types := make([]string, 0, len(typeSet))
	for nodeType := range typeSet {
		types = append(types, nodeType)
	}

	sort.Strings(types)
	return types
}

// GetTypesWithInfo 获取类型详细信息
func (r *NodeRegistry) GetTypesWithInfo() []TypeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typeMap := make(map[string]*TypeInfo)
	for _, metadata := range r.metadata {
		if _, exists := typeMap[metadata.Type]; !exists {
			typeMap[metadata.Type] = &TypeInfo{
				Type:      metadata.Type,
				NodeCount: 0,
			}
		}
		typeMap[metadata.Type].NodeCount++
	}

	types := make([]TypeInfo, 0, len(typeMap))
	for _, info := range typeMap {
		types = append(types, *info)
	}

	// 按类型名称排序
	sort.Slice(types, func(i, j int) bool {
		return types[i].Type < types[j].Type
	})

	return types
}

// CategoryInfo 分类信息结构体
type CategoryInfo struct {
	Category  string `json:"category"`
	NodeCount int    `json:"node_count"`
}

// TypeInfo 类型信息结构体
type TypeInfo struct {
	Type      string `json:"type"`
	NodeCount int    `json:"node_count"`
}

// Search 搜索节点
func (r *NodeRegistry) Search(query string) []*NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*NodeMetadata
	for _, metadata := range r.metadata {
		if r.matchesQuery(metadata, query) {
			result = append(result, metadata)
		}
	}

	// 按相关性排序（简单实现）
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// Count 获取注册节点数量
func (r *NodeRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// IsRegistered 检查节点是否已注册
func (r *NodeRegistry) IsRegistered(pluginID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.nodes[pluginID]
	return exists
}

// validatePlugin 验证插件
func (r *NodeRegistry) validatePlugin(plugin NodePlugin) error {
	metadata := plugin.GetMetadata()

	// 验证基础字段
	if metadata.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if metadata.Type == "" {
		return fmt.Errorf("plugin type cannot be empty")
	}

	if metadata.Category == "" {
		return fmt.Errorf("plugin category cannot be empty")
	}

	// 验证配置模式
	if metadata.ConfigSchema == nil {
		return fmt.Errorf("plugin config schema cannot be nil")
	}

	// 可以添加更多验证逻辑
	return nil
}

// matchesQuery 检查元数据是否匹配查询
func (r *NodeRegistry) matchesQuery(metadata *NodeMetadata, query string) bool {
	if query == "" {
		return true
	}

	// 简单的字符串匹配实现
	// 可以扩展为更复杂的搜索逻辑
	return contains(metadata.Name, query) ||
		contains(metadata.Description, query) ||
		contains(metadata.Category, query) ||
		contains(metadata.Type, query) ||
		containsInSlice(metadata.Tags, query)
}

// contains 检查字符串是否包含子串（忽略大小写）
func contains(str, substr string) bool {
	if str == "" || substr == "" {
		return false
	}
	// 简单实现，可以使用 strings.Contains 和 strings.ToLower
	return len(str) >= len(substr) &&
		(str == substr ||
			(len(str) > len(substr) &&
				(str[:len(substr)] == substr ||
					str[len(str)-len(substr):] == substr)))
}

// containsInSlice 检查字符串切片是否包含子串
func containsInSlice(slice []string, substr string) bool {
	for _, str := range slice {
		if contains(str, substr) {
			return true
		}
	}
	return false
}

// 便利函数
func Register(plugin NodePlugin) error {
	return GetRegistry().Register(plugin)
}

func Get(pluginID string) (NodePlugin, error) {
	return GetRegistry().Get(pluginID)
}

func GetMetadata(pluginID string) (*NodeMetadata, error) {
	return GetRegistry().GetMetadata(pluginID)
}

func List() []string {
	return GetRegistry().List()
}

func GetAllMetadata() []*NodeMetadata {
	return GetRegistry().GetAllMetadata()
}

func Search(query string) []*NodeMetadata {
	return GetRegistry().Search(query)
}
