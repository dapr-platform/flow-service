/**
 * @module static_data_datasource
 * @description 静态数据源节点，提供配置化的JSON数据输出功能
 * @architecture 数据源插件实现，支持JSON格式数据配置和输出
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow static_states: configured -> parsing -> outputting -> completed
 * @rules 支持JSON数组格式数据配置，便于测试和演示使用
 * @dependencies encoding/json, context, time
 * @refs service/nodes/interface.go
 */

package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册静态数据节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewStaticDataNode()); err != nil {
		log.Printf("注册静态数据节点失败: %v", err)
	} else {
		log.Println("静态数据节点注册成功")
	}
}

// StaticDataNode 静态数据源节点
type StaticDataNode struct{}

// NewStaticDataNode 创建静态数据节点
func NewStaticDataNode() *StaticDataNode {
	return &StaticDataNode{}
}

// GetMetadata 获取节点元数据
func (s *StaticDataNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "static_data_datasource",
		Name:        "静态数据源",
		Description: "提供配置化的静态JSON数据输出",
		Version:     "1.0.0",
		Category:    nodes.CategoryDataSource,
		Type:        nodes.TypeStatic,
		Icon:        "data",
		Tags:        []string{"静态数据", "JSON", "测试", "配置"},

		InputPorts: []nodes.PortDefinition{},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "静态数据",
				Description: "配置的静态数据输出",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "json_data",
					Type:        "string",
					Title:       "JSON数据",
					Description: "JSON格式的数组数据",
					Widget:      nodes.WidgetCode,
					Placeholder: `[{"id": 1, "name": "张三", "age": 25}, {"id": 2, "name": "李四", "age": 30}]`,
				},
				{
					Name:        "data_name",
					Type:        "string",
					Title:       "数据名称",
					Description: "数据集的名称描述",
					Default:     "测试数据",
					Widget:      nodes.WidgetText,
				},
			},
			Required: []string{"json_data"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (s *StaticDataNode) Validate(config map[string]interface{}) error {
	jsonData, ok := config["json_data"].(string)
	if !ok || jsonData == "" {
		return fmt.Errorf("missing or invalid json_data")
	}

	// 验证JSON格式
	var testData []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &testData); err != nil {
		return fmt.Errorf("invalid JSON format: %v", err)
	}

	return nil
}

// Execute 执行节点
func (s *StaticDataNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()
	output := &nodes.NodeOutput{
		Data:    make(map[string]interface{}),
		Logs:    []string{},
		Metrics: make(map[string]interface{}),
		Success: false,
	}

	jsonData := input.Config["json_data"].(string)
	dataName := s.getDataName(input.Config)

	output.Logs = append(output.Logs, fmt.Sprintf("开始处理静态数据: %s", dataName))

	// 解析JSON数据
	var rawData []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &rawData); err != nil {
		output.Error = fmt.Sprintf("JSON解析失败: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}

	// 构建输出
	output.Data["data"] = rawData
	output.Success = true
	output.Duration = time.Since(startTime)
	output.Logs = append(output.Logs, fmt.Sprintf("静态数据处理完成，共 %d 条记录", len(rawData)))

	// 添加指标
	output.Metrics["processing_duration"] = time.Since(startTime).Milliseconds()
	output.Metrics["record_count"] = len(rawData)
	output.Metrics["data_size"] = len(jsonData)

	return output, nil
}

// getDataName 获取数据名称
func (s *StaticDataNode) getDataName(config map[string]interface{}) string {
	if name, ok := config["data_name"].(string); ok && name != "" {
		return name
	}
	return "测试数据"
}

// GetDynamicData 获取动态配置数据（默认实现）
func (s *StaticDataNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("静态数据节点暂不支持动态数据获取方法: %s", method)
}
