/**
 * @module node_interface
 * @description 简化的节点接口定义和元数据结构，为内置节点系统提供统一的接口规范，支持动态配置数据获取
 * @architecture 插件化节点系统设计，支持动态注册和配置，扩展了动态数据获取功能
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow node_states: created -> configured -> ready -> executing -> completed/failed
 * @rules 所有节点必须实现NodePlugin接口，元数据必须包含完整的配置定义，支持动态数据获取
 * @dependencies context, time
 * @refs service/models/node.go
 */

package nodes

import (
	"context"
	"time"
)

// NodePlugin 节点插件接口（简化版，支持动态数据）
type NodePlugin interface {
	// GetMetadata 获取节点元数据（包含配置模式）
	GetMetadata() *NodeMetadata

	// Validate 验证节点配置
	Validate(config map[string]interface{}) error

	// Execute 执行节点
	Execute(ctx context.Context, input *NodeInput) (*NodeOutput, error)

	// GetDynamicData 获取动态配置数据（新增）
	GetDynamicData(method string, params map[string]interface{}) (interface{}, error)
}

// NodeMetadata 节点元数据（简化版）
type NodeMetadata struct {
	// 基础信息
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Category    string   `json:"category"`
	Type        string   `json:"type"`
	Icon        string   `json:"icon"`
	Tags        []string `json:"tags"`

	// 输入输出定义
	InputPorts  []PortDefinition `json:"input_ports"`
	OutputPorts []PortDefinition `json:"output_ports"`

	// 配置定义
	ConfigSchema *ConfigSchema `json:"config_schema"`

	// 创建信息
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PortDefinition 端口定义
type PortDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DataType    string `json:"data_type"`
	Required    bool   `json:"required"`
	Multiple    bool   `json:"multiple"`
}

// ConfigSchema 配置模式定义（简化版）
type ConfigSchema struct {
	Type       string        `json:"type"`
	Properties []ConfigField `json:"properties"`
	Required   []string      `json:"required"`
}

// ConfigField 配置字段定义（简化版，支持动态数据源）
type ConfigField struct {
	Name        string        `json:"name"` // 字段名称（新增）
	Type        string        `json:"type"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Properties  []ConfigField `json:"properties,omitempty"`
	Items       *ConfigField  `json:"items,omitempty"`
	Required    []string      `json:"required,omitempty"`

	// 基础UI属性
	Widget      string `json:"widget,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`

	// 动态数据源配置（新增）
	DataSource *DataSourceConfig `json:"data_source,omitempty"`
}

// DataSourceConfig 动态数据源配置（新增）
type DataSourceConfig struct {
	Type         string                 `json:"type"`         // "method", "api", "static"
	Method       string                 `json:"method"`       // 方法名，如"GetTableNames"
	Parameters   map[string]interface{} `json:"parameters"`   // 方法参数配置
	Dependencies []string               `json:"dependencies"` // 依赖的其他配置字段
	CacheTime    int                    `json:"cache_time"`   // 缓存时间(秒)，0表示不缓存
	Fallback     interface{}            `json:"fallback"`     // 获取失败时的fallback值
}

// NodeInput 节点输入
type NodeInput struct {
	Data      map[string]interface{} `json:"data"`
	Config    map[string]interface{} `json:"config"`
	Context   map[string]interface{} `json:"context"`
	Variables map[string]interface{} `json:"variables"`
}

// NodeOutput 节点输出
type NodeOutput struct {
	Data     map[string]interface{} `json:"data"`
	Error    string                 `json:"error,omitempty"`
	Logs     []string               `json:"logs,omitempty"`
	Metrics  map[string]interface{} `json:"metrics,omitempty"`
	Success  bool                   `json:"success"`
	Duration time.Duration          `json:"duration"`
}

// DynamicDataRequest 动态数据请求结构（新增）
type DynamicDataRequest struct {
	NodeID     string                 `json:"node_id"`
	Method     string                 `json:"method"`
	Parameters map[string]interface{} `json:"parameters"`
}

// DynamicDataResponse 动态数据响应结构（新增）
type DynamicDataResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
	Cached  bool        `json:"cached"`
}
