/**
 * @module node_controller
 * @description 节点控制器，提供节点完整信息的API接口，支持动态配置数据获取
 * @architecture REST API控制器，支持节点注册、查询、完整信息获取和动态数据获取
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow controller_states: initialized -> registered -> serving
 * @rules 提供标准REST API，支持分页查询，返回统一格式响应，包含完整节点信息，支持动态数据获取
 * @dependencies chi, net/http, encoding/json, sync
 * @refs service/nodes/registry.go, service/nodes/interface.go
 */

package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"flow-service/service/nodes"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// NodeController 节点控制器
type NodeController struct {
	registry     *nodes.NodeRegistry
	dynamicCache map[string]*CacheEntry
	cacheMutex   sync.RWMutex
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
	TTL       int // 秒
}

// NewNodeController 创建节点控制器
func NewNodeController() *NodeController {
	return &NodeController{
		registry:     nodes.GetRegistry(),
		dynamicCache: make(map[string]*CacheEntry),
	}
}

func (nc *NodeController) getIntParam(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return num
}

// GetNodes 获取所有节点完整信息
// @Summary 获取所有节点完整信息
// @Description 获取系统中注册的所有节点的完整信息，包括元数据和配置模式
// @Tags nodes
// @Accept json
// @Produce json
// @Success 200 {object} controllers.APIResponse{data=NodesListResponse}
// @Failure 400 {object} controllers.APIResponse
// @Failure 500 {object} controllers.APIResponse
// @Router /nodes [get]
func (nc *NodeController) GetNodes(w http.ResponseWriter, r *http.Request) {

	// 获取节点元数据
	allMetadata := nc.registry.GetAllMetadata()

	// 构建完整响应，包含schema信息
	response := NodesListResponse{
		Nodes:          make([]NodeFullInfo, len(allMetadata)),
		CategoriesInfo: nc.registry.GetCategoriesWithInfo(),
		TypesInfo:      nc.registry.GetTypesWithInfo(),
	}

	for i, metadata := range allMetadata {
		response.Nodes[i] = NodeFullInfo{
			// 基础信息
			ID:          metadata.ID,
			Name:        metadata.Name,
			Description: metadata.Description,
			Version:     metadata.Version,
			Category:    metadata.Category,
			Type:        metadata.Type,
			Icon:        metadata.Icon,
			Tags:        metadata.Tags,
			Author:      metadata.Author,
			CreatedAt:   metadata.CreatedAt,
			UpdatedAt:   metadata.UpdatedAt,

			// 完整元数据
			Metadata: metadata,
		}
	}

	render.Render(w, r, SuccessResponse("获取节点列表成功", response))
}

// ValidateNodeConfig 验证节点配置
// @Summary 验证节点配置
// @Description 验证指定节点的配置是否正确
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "节点ID"
// @Param config body map[string]interface{} true "节点配置"
// @Success 200 {object} controllers.APIResponse{data=ValidationResult}
// @Failure 400 {object} controllers.APIResponse
// @Failure 404 {object} controllers.APIResponse
// @Failure 500 {object} controllers.APIResponse
// @Router /nodes/{id}/validate [post]
func (nc *NodeController) ValidateNodeConfig(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		render.Status(r, http.StatusBadRequest)
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "节点ID不能为空", nil))
		return
	}

	// 解析请求体
	var config map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "无效的JSON格式", err))
		return
	}

	// 获取节点插件
	plugin, err := nc.registry.Get(nodeID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.Render(w, r, ErrorResponse(http.StatusNotFound, "节点不存在", err))
		return
	}

	// 验证配置
	validationErr := plugin.Validate(config)

	result := ValidationResult{
		Valid:  validationErr == nil,
		NodeID: nodeID,
		Config: config,
	}

	if validationErr != nil {
		result.Error = validationErr.Error()
	}

	message := "配置验证成功"
	if !result.Valid {
		message = "配置验证失败"
	}

	render.Render(w, r, SuccessResponse(message, result))
}

// GetDynamicData 获取节点动态配置数据
// @Summary 获取节点动态配置数据
// @Description 获取指定节点的动态配置数据，如数据库表名、列名等
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path string true "节点ID"
// @Param method query string true "方法名"
// @Param request body map[string]interface{} false "方法参数"
// @Success 200 {object} controllers.APIResponse{data=nodes.DynamicDataResponse}
// @Failure 400 {object} controllers.APIResponse
// @Failure 404 {object} controllers.APIResponse
// @Failure 500 {object} controllers.APIResponse
// @Router /nodes/{id}/dynamic-data [post]
func (nc *NodeController) GetDynamicData(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		render.Status(r, http.StatusBadRequest)
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "节点ID不能为空", nil))
		return
	}

	method := r.URL.Query().Get("method")
	if method == "" {
		render.Status(r, http.StatusBadRequest)
		render.Render(w, r, ErrorResponse(http.StatusBadRequest, "方法名不能为空", nil))
		return
	}

	// 解析请求参数
	var params map[string]interface{}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			params = make(map[string]interface{})
		}
	}

	// 检查缓存
	cacheKey := nc.getCacheKey(nodeID, method, params)
	if cachedData := nc.getCachedData(cacheKey); cachedData != nil {
		response := &nodes.DynamicDataResponse{
			Success: true,
			Data:    cachedData,
			Cached:  true,
		}
		render.Render(w, r, SuccessResponse("获取动态数据成功（缓存）", response))
		return
	}

	// 获取节点插件
	plugin, err := nc.registry.Get(nodeID)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.Render(w, r, ErrorResponse(http.StatusNotFound, "节点不存在", err))
		return
	}

	// 调用动态数据获取方法
	data, err := plugin.GetDynamicData(method, params)

	response := &nodes.DynamicDataResponse{
		Success: err == nil,
		Data:    data,
		Cached:  false,
	}

	if err != nil {
		response.Error = err.Error()
		render.Render(w, r, SuccessResponse("获取动态数据失败", response))
		return
	}

	// 缓存结果
	nc.setCachedData(cacheKey, data, 300) // 默认缓存5分钟

	render.Render(w, r, SuccessResponse("获取动态数据成功", response))
}

// getCacheKey 生成缓存键
func (nc *NodeController) getCacheKey(nodeID, method string, params map[string]interface{}) string {
	paramsBytes, _ := json.Marshal(params)
	return nodeID + ":" + method + ":" + string(paramsBytes)
}

// getCachedData 获取缓存数据
func (nc *NodeController) getCachedData(key string) interface{} {
	nc.cacheMutex.RLock()
	defer nc.cacheMutex.RUnlock()

	entry, exists := nc.dynamicCache[key]
	if !exists {
		return nil
	}

	// 检查是否过期
	if time.Since(entry.Timestamp).Seconds() > float64(entry.TTL) {
		delete(nc.dynamicCache, key)
		return nil
	}

	return entry.Data
}

// setCachedData 设置缓存数据
func (nc *NodeController) setCachedData(key string, data interface{}, ttl int) {
	nc.cacheMutex.Lock()
	defer nc.cacheMutex.Unlock()

	nc.dynamicCache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// 响应结构体

// NodesListResponse 节点列表响应
type NodesListResponse struct {
	Nodes          []NodeFullInfo       `json:"nodes"`
	Pagination     PaginationInfo       `json:"pagination"`
	CategoriesInfo []nodes.CategoryInfo `json:"categories_info"`
	TypesInfo      []nodes.TypeInfo     `json:"types_info"`
}

// NodeFullInfo 节点完整信息
type NodeFullInfo struct {
	// 基础信息
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	Category    string    `json:"category"`
	Type        string    `json:"type"`
	Icon        string    `json:"icon"`
	Tags        []string  `json:"tags"`
	Author      string    `json:"author"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// 完整元数据
	Metadata *nodes.NodeMetadata `json:"metadata"`
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Page       int `json:"page"`
	Size       int `json:"size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool                   `json:"valid"`
	NodeID string                 `json:"node_id"`
	Config map[string]interface{} `json:"config"`
	Error  string                 `json:"error,omitempty"`
}

// CategoriesResponse 分类响应
type CategoriesResponse struct {
	Categories []CategoryInfo `json:"categories"`
	Total      int            `json:"total"`
}

// CategoryInfo 分类信息
type CategoryInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	NodeCount   int    `json:"node_count"`
	Icon        string `json:"icon"`
}

// TypesResponse 类型响应
type TypesResponse struct {
	Types []TypeInfo `json:"types"`
	Total int        `json:"total"`
}

// TypeInfo 类型信息
type TypeInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	NodeCount   int    `json:"node_count"`
	Icon        string `json:"icon"`
}

// RegistryStats 注册表统计信息
type RegistryStats struct {
	TotalNodes      int            `json:"total_nodes"`
	TotalCategories int            `json:"total_categories"`
	TotalTypes      int            `json:"total_types"`
	Categories      map[string]int `json:"categories"`
	Types           map[string]int `json:"types"`
}
