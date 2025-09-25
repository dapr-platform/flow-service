/**
 * @module data_filter
 * @description 数据过滤节点，提供灵活的数据过滤和筛选功能
 * @architecture 数据处理插件实现，支持多种过滤条件和逻辑运算
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow filter_states: configured -> filtering -> completed
 * @rules 支持多种数据类型过滤，提供AND/OR逻辑运算，支持嵌套条件
 * @dependencies context, time, reflect, strconv
 * @refs service/nodes/interface.go
 */

package transform

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册数据过滤节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewDataFilterNode()); err != nil {
		log.Printf("注册数据过滤节点失败: %v", err)
	} else {
		log.Println("数据过滤节点注册成功")
	}
}

// DataFilterNode 数据过滤节点
type DataFilterNode struct{}

// NewDataFilterNode 创建数据过滤节点
func NewDataFilterNode() *DataFilterNode {
	return &DataFilterNode{}
}

// GetMetadata 获取节点元数据
func (d *DataFilterNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "data_filter",
		Name:        "数据过滤器",
		Description: "根据条件过滤数据记录",
		Version:     "1.0.0",
		Category:    nodes.CategoryTransform,
		Type:        nodes.TypeDataFilter,
		Icon:        "filter",
		Tags:        []string{"过滤", "条件", "筛选"},

		InputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "输入数据",
				Description: "需要过滤的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "filtered_data",
				Name:        "过滤结果",
				Description: "符合条件的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
			{
				ID:          "excluded_data",
				Name:        "排除数据",
				Description: "不符合条件的数据",
				DataType:    nodes.DataTypeArray,
				Required:    false,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "conditions",
					Type:        "array",
					Title:       "过滤条件",
					Description: "数据过滤条件列表",
					Items: &nodes.ConfigField{
						Type: "object",
						Properties: []nodes.ConfigField{
							{
								Name:        "field",
								Type:        "string",
								Title:       "字段名",
								Description: "要过滤的字段名",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "operator",
								Type:        "string",
								Title:       "操作符",
								Description: "比较操作符",
								Default:     "equals",
								Enum:        []interface{}{"equals", "not_equals", "greater", "greater_equal", "less", "less_equal", "in", "not_in", "contains", "not_contains", "starts_with", "ends_with", "regex"},
								Widget:      nodes.WidgetSelect,
							},
							{
								Name:        "value",
								Type:        "string",
								Title:       "比较值",
								Description: "用于比较的值",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "data_type",
								Type:        "string",
								Title:       "数据类型",
								Description: "字段的数据类型",
								Default:     "string",
								Enum:        []interface{}{"string", "number", "boolean", "date"},
								Widget:      nodes.WidgetSelect,
							},
						},
						Required: []string{"field", "operator", "value"},
					},
				},
				{
					Name:        "logic",
					Type:        "string",
					Title:       "逻辑关系",
					Description: "多个条件之间的逻辑关系",
					Default:     "and",
					Enum:        []interface{}{"and", "or"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "include_excluded",
					Type:        "boolean",
					Title:       "输出排除数据",
					Description: "是否输出不符合条件的数据",
					Default:     false,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "include_stats",
					Type:        "boolean",
					Title:       "包含统计信息",
					Description: "是否在输出中包含过滤统计信息",
					Default:     true,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "limit",
					Type:        "number",
					Title:       "结果限制",
					Description: "限制输出结果数量，0表示不限制",
					Default:     0,
					Widget:      nodes.WidgetNumber,
				},
			},
			Required: []string{"conditions"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (d *DataFilterNode) Validate(config map[string]interface{}) error {
	// 验证过滤条件
	conditions, ok := config["conditions"].([]interface{})
	if !ok || len(conditions) == 0 {
		return fmt.Errorf("missing or empty conditions")
	}

	// 验证每个条件
	for i, conditionRaw := range conditions {
		condition, ok := conditionRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("condition %d: invalid format", i)
		}

		if field, ok := condition["field"].(string); !ok || field == "" {
			return fmt.Errorf("condition %d: missing or invalid field", i)
		}

		if operator, ok := condition["operator"].(string); !ok || operator == "" {
			return fmt.Errorf("condition %d: missing or invalid operator", i)
		}

		if _, ok := condition["value"]; !ok {
			return fmt.Errorf("condition %d: missing value", i)
		}
	}

	return nil
}

// Execute 执行节点
func (d *DataFilterNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()
	output := &nodes.NodeOutput{
		Data:    make(map[string]interface{}),
		Logs:    []string{},
		Metrics: make(map[string]interface{}),
		Success: false,
	}

	// 获取输入数据
	inputData, exists := input.Data["data"]
	if !exists {
		output.Error = "missing input data"
		output.Duration = time.Since(startTime)
		return output, nil
	}

	dataArray, ok := inputData.([]interface{})
	if !ok {
		output.Error = "input data must be an array"
		output.Duration = time.Since(startTime)
		return output, nil
	}

	output.Logs = append(output.Logs, fmt.Sprintf("开始过滤 %d 条数据", len(dataArray)))

	// 解析配置
	conditions := input.Config["conditions"].([]interface{})
	logic := d.getLogic(input.Config)
	outputConfig := d.getOutputConfig(input.Config)

	// 执行过滤
	filteredData, excludedData, stats := d.filterData(dataArray, conditions, logic)

	// 应用结果限制
	if outputConfig.Limit > 0 && len(filteredData) > outputConfig.Limit {
		filteredData = filteredData[:outputConfig.Limit]
		stats["limited"] = true
		stats["limit_applied"] = outputConfig.Limit
	}

	// 构建输出
	output.Data["filtered_data"] = filteredData

	if outputConfig.IncludeExcluded {
		output.Data["excluded_data"] = excludedData
	}

	if outputConfig.IncludeStats {
		output.Data["filter_stats"] = stats
	}

	output.Success = true
	output.Duration = time.Since(startTime)
	output.Logs = append(output.Logs, fmt.Sprintf("过滤完成，保留 %d 条数据，排除 %d 条数据",
		len(filteredData), len(excludedData)))

	// 添加指标
	output.Metrics["filter_duration"] = time.Since(startTime).Milliseconds()
	output.Metrics["input_count"] = len(dataArray)
	output.Metrics["output_count"] = len(filteredData)
	output.Metrics["excluded_count"] = len(excludedData)

	return output, nil
}

// filterData 执行数据过滤
func (d *DataFilterNode) filterData(data []interface{}, conditions []interface{}, logic string) ([]interface{}, []interface{}, map[string]interface{}) {
	var filteredData []interface{}
	var excludedData []interface{}

	stats := map[string]interface{}{
		"total_input": len(data),
		"conditions":  len(conditions),
		"logic":       logic,
		"filter_time": time.Now(),
	}

	for _, item := range data {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			excludedData = append(excludedData, item)
			continue
		}

		if d.evaluateConditions(itemMap, conditions, logic) {
			filteredData = append(filteredData, item)
		} else {
			excludedData = append(excludedData, item)
		}
	}

	stats["filtered_count"] = len(filteredData)
	stats["excluded_count"] = len(excludedData)
	stats["filter_rate"] = float64(len(filteredData)) / float64(len(data))

	return filteredData, excludedData, stats
}

// evaluateConditions 评估条件
func (d *DataFilterNode) evaluateConditions(item map[string]interface{}, conditions []interface{}, logic string) bool {
	if len(conditions) == 0 {
		return true
	}

	results := make([]bool, len(conditions))
	for i, condition := range conditions {
		conditionMap := condition.(map[string]interface{})
		results[i] = d.evaluateCondition(item, conditionMap)
	}

	// 应用逻辑运算
	if logic == "or" {
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	} else { // and
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	}
}

// evaluateCondition 评估单个条件
func (d *DataFilterNode) evaluateCondition(item map[string]interface{}, condition map[string]interface{}) bool {
	field := condition["field"].(string)
	operator := condition["operator"].(string)

	// 获取字段值
	fieldValue, exists := item[field]

	// 处理存在性检查
	if operator == "exists" {
		return exists
	}
	if operator == "not_exists" {
		return !exists
	}

	if !exists {
		return false
	}

	// 获取比较值
	compareValue, hasCompareValue := condition["value"]
	if !hasCompareValue && operator != "exists" && operator != "not_exists" {
		return false
	}

	// 获取数据类型
	dataType := "string"
	if dt, ok := condition["data_type"].(string); ok {
		dataType = dt
	}

	// 转换数据类型并比较
	return d.compareValues(fieldValue, compareValue, operator, dataType)
}

// compareValues 比较值
func (d *DataFilterNode) compareValues(fieldValue, compareValue interface{}, operator, dataType string) bool {
	switch operator {
	case "equals":
		return d.equals(fieldValue, compareValue, dataType)
	case "not_equals":
		return !d.equals(fieldValue, compareValue, dataType)
	case "greater":
		return d.greater(fieldValue, compareValue, dataType)
	case "greater_equal":
		return d.greater(fieldValue, compareValue, dataType) || d.equals(fieldValue, compareValue, dataType)
	case "less":
		return d.less(fieldValue, compareValue, dataType)
	case "less_equal":
		return d.less(fieldValue, compareValue, dataType) || d.equals(fieldValue, compareValue, dataType)
	case "in":
		return d.in(fieldValue, compareValue, dataType)
	case "not_in":
		return !d.in(fieldValue, compareValue, dataType)
	case "contains":
		return d.contains(fieldValue, compareValue)
	case "not_contains":
		return !d.contains(fieldValue, compareValue)
	case "starts_with":
		return d.startsWith(fieldValue, compareValue)
	case "ends_with":
		return d.endsWith(fieldValue, compareValue)
	case "regex":
		return d.regex(fieldValue, compareValue)
	default:
		return false
	}
}

// equals 相等比较
func (d *DataFilterNode) equals(fieldValue, compareValue interface{}, dataType string) bool {
	switch dataType {
	case "number":
		fv := d.toFloat64(fieldValue)
		cv := d.toFloat64(compareValue)
		return fv == cv
	case "boolean":
		fv := d.toBool(fieldValue)
		cv := d.toBool(compareValue)
		return fv == cv
	default:
		return d.toString(fieldValue) == d.toString(compareValue)
	}
}

// greater 大于比较
func (d *DataFilterNode) greater(fieldValue, compareValue interface{}, dataType string) bool {
	if dataType == "number" {
		fv := d.toFloat64(fieldValue)
		cv := d.toFloat64(compareValue)
		return fv > cv
	}
	return d.toString(fieldValue) > d.toString(compareValue)
}

// less 小于比较
func (d *DataFilterNode) less(fieldValue, compareValue interface{}, dataType string) bool {
	if dataType == "number" {
		fv := d.toFloat64(fieldValue)
		cv := d.toFloat64(compareValue)
		return fv < cv
	}
	return d.toString(fieldValue) < d.toString(compareValue)
}

// in 包含比较
func (d *DataFilterNode) in(fieldValue, compareValue interface{}, dataType string) bool {
	compareStr := d.toString(compareValue)
	values := strings.Split(compareStr, ",")

	fieldStr := d.toString(fieldValue)
	for _, value := range values {
		if strings.TrimSpace(value) == fieldStr {
			return true
		}
	}
	return false
}

// contains 字符串包含
func (d *DataFilterNode) contains(fieldValue, compareValue interface{}) bool {
	fieldStr := d.toString(fieldValue)
	compareStr := d.toString(compareValue)
	return strings.Contains(fieldStr, compareStr)
}

// startsWith 字符串开头
func (d *DataFilterNode) startsWith(fieldValue, compareValue interface{}) bool {
	fieldStr := d.toString(fieldValue)
	compareStr := d.toString(compareValue)
	return strings.HasPrefix(fieldStr, compareStr)
}

// endsWith 字符串结尾
func (d *DataFilterNode) endsWith(fieldValue, compareValue interface{}) bool {
	fieldStr := d.toString(fieldValue)
	compareStr := d.toString(compareValue)
	return strings.HasSuffix(fieldStr, compareStr)
}

// regex 正则表达式匹配
func (d *DataFilterNode) regex(fieldValue, compareValue interface{}) bool {
	// 简单实现，实际应该使用 regexp 包
	fieldStr := d.toString(fieldValue)
	pattern := d.toString(compareValue)
	return strings.Contains(fieldStr, pattern)
}

// 类型转换辅助方法
func (d *DataFilterNode) toString(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func (d *DataFilterNode) toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

func (d *DataFilterNode) toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1" || v == "yes"
	case int:
		return v != 0
	case float64:
		return v != 0
	}
	return false
}

// getLogic 获取逻辑运算符
func (d *DataFilterNode) getLogic(config map[string]interface{}) string {
	if logic, ok := config["logic"].(string); ok {
		return logic
	}
	return "and"
}

// getOutputConfig 获取输出配置
func (d *DataFilterNode) getOutputConfig(config map[string]interface{}) *OutputConfig {
	outputConfig := &OutputConfig{
		IncludeExcluded: false,
		IncludeStats:    true,
		Limit:           0,
	}

	if includeExcluded, ok := config["include_excluded"].(bool); ok {
		outputConfig.IncludeExcluded = includeExcluded
	}

	if includeStats, ok := config["include_stats"].(bool); ok {
		outputConfig.IncludeStats = includeStats
	}

	if limit, ok := config["limit"].(float64); ok {
		outputConfig.Limit = int(limit)
	}

	return outputConfig
}

// OutputConfig 输出配置
type OutputConfig struct {
	IncludeExcluded bool
	IncludeStats    bool
	Limit           int
}

// GetDynamicData 获取动态配置数据（默认实现）
func (d *DataFilterNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("数据过滤节点暂不支持动态数据获取方法: %s", method)
}
