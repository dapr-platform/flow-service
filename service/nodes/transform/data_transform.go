/**
 * @module data_transform
 * @description 数据转换节点，提供字段映射、数据格式转换、值计算等功能
 * @architecture 数据处理插件实现，支持多种转换规则和表达式计算
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow transform_states: configured -> transforming -> completed
 * @rules 支持字段映射、类型转换、表达式计算、数据格式化
 * @dependencies context, time, reflect, strconv, regexp, math
 * @refs service/nodes/interface.go
 */

package transform

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册数据转换节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewDataTransformNode()); err != nil {
		log.Printf("注册数据转换节点失败: %v", err)
	} else {
		log.Println("数据转换节点注册成功")
	}
}

// DataTransformNode 数据转换节点
type DataTransformNode struct{}

// NewDataTransformNode 创建数据转换节点
func NewDataTransformNode() *DataTransformNode {
	return &DataTransformNode{}
}

// GetMetadata 获取节点元数据
func (d *DataTransformNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "data_transform",
		Name:        "数据转换器",
		Description: "对数据进行字段映射和转换",
		Version:     "1.0.0",
		Category:    nodes.CategoryTransform,
		Type:        nodes.TypeDataTransform,
		Icon:        "transform",
		Tags:        []string{"转换", "映射", "字段"},

		InputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "输入数据",
				Description: "需要转换的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "transformed_data",
				Name:        "转换结果",
				Description: "转换后的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "mappings",
					Type:        "array",
					Title:       "字段映射",
					Description: "字段转换映射规则",
					Items: &nodes.ConfigField{
						Type: "object",
						Properties: []nodes.ConfigField{
							{
								Name:        "source_field",
								Type:        "string",
								Title:       "源字段",
								Description: "源数据中的字段名",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "target_field",
								Type:        "string",
								Title:       "目标字段",
								Description: "转换后的字段名",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "transform_type",
								Type:        "string",
								Title:       "转换类型",
								Description: "数据转换类型",
								Default:     "copy",
								Enum:        []interface{}{"copy", "uppercase", "lowercase", "trim", "format_date", "to_number", "to_string", "to_boolean", "concat", "split", "replace", "substring"},
								Widget:      nodes.WidgetSelect,
							},
							{
								Name:        "transform_params",
								Type:        "object",
								Title:       "转换参数",
								Description: "转换操作的参数",
								Widget:      nodes.WidgetJSON,
							},
							{
								Name:        "default_value",
								Type:        "string",
								Title:       "默认值",
								Description: "当源字段不存在时的默认值",
								Widget:      nodes.WidgetText,
							},
						},
						Required: []string{"source_field", "target_field", "transform_type"},
					},
				},
				{
					Name:        "remove_unmapped",
					Type:        "boolean",
					Title:       "移除未映射字段",
					Description: "是否移除未在映射中定义的字段",
					Default:     false,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "preserve_original",
					Type:        "boolean",
					Title:       "保留原始字段",
					Description: "在添加新字段时是否保留原始字段",
					Default:     false,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "batch_size",
					Type:        "number",
					Title:       "批处理大小",
					Description: "每批处理的记录数量，0表示一次处理全部",
					Default:     0,
					Widget:      nodes.WidgetNumber,
				},
			},
			Required: []string{"mappings"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证配置
func (d *DataTransformNode) Validate(config map[string]interface{}) error {
	// 验证mappings字段
	mappingsRaw, exists := config["mappings"]
	if !exists {
		return fmt.Errorf("missing required field: mappings")
	}

	mappings, ok := mappingsRaw.([]interface{})
	if !ok {
		return fmt.Errorf("mappings must be an array")
	}

	if len(mappings) == 0 {
		return fmt.Errorf("mappings cannot be empty")
	}

	// 验证每个映射配置
	for i, mappingRaw := range mappings {
		mapping, ok := mappingRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("mappings[%d] must be an object", i)
		}

		// 验证必需字段
		if _, exists := mapping["source_field"]; !exists {
			return fmt.Errorf("mappings[%d] missing required field: source_field", i)
		}
		if _, exists := mapping["target_field"]; !exists {
			return fmt.Errorf("mappings[%d] missing required field: target_field", i)
		}
		if _, exists := mapping["transform_type"]; !exists {
			return fmt.Errorf("mappings[%d] missing required field: transform_type", i)
		}

		// 验证转换类型
		transformType, _ := mapping["transform_type"].(string)
		validTypes := []string{"copy", "uppercase", "lowercase", "trim", "format_date", "to_number", "to_string", "to_boolean", "concat", "split", "replace", "substring"}
		valid := false
		for _, vt := range validTypes {
			if transformType == vt {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("mappings[%d] invalid transform_type: %s", i, transformType)
		}
	}

	return nil
}

// Execute 执行转换
func (d *DataTransformNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()

	// 获取输入数据
	dataRaw, exists := input.Data["data"]
	if !exists {
		return &nodes.NodeOutput{
			Success: false,
			Error:   "缺少输入数据",
		}, nil
	}

	data, ok := dataRaw.([]interface{})
	if !ok {
		return &nodes.NodeOutput{
			Success: false,
			Error:   "输入数据必须是数组",
		}, nil
	}

	// 解析配置
	mappings := d.getMappings(input.Config)
	globalConfig := d.getGlobalConfig(input.Config)

	// 执行转换
	transformedData, stats := d.transformData(data, mappings, globalConfig)

	// 构建输出
	output := &nodes.NodeOutput{
		Data: map[string]interface{}{
			"transformed_data": transformedData,
		},
		Success:  true,
		Duration: time.Since(startTime),
	}

	if globalConfig.IncludeStats {
		output.Data["transform_stats"] = stats
	}

	return output, nil
}

// transformData 转换数据
func (d *DataTransformNode) transformData(data []interface{}, mappings []MappingConfig, globalConfig *GlobalConfig) ([]interface{}, map[string]interface{}) {
	transformedData := make([]interface{}, 0, len(data))
	stats := map[string]interface{}{
		"total_records":     len(data),
		"transformed_count": 0,
		"error_count":       0,
		"field_stats":       make(map[string]interface{}),
	}

	fieldStats := make(map[string]map[string]int)

	for _, itemRaw := range data {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			stats["error_count"] = stats["error_count"].(int) + 1
			continue
		}

		transformedItem := make(map[string]interface{})

		// 保留原始字段
		if globalConfig.PreserveOriginal {
			for k, v := range item {
				transformedItem[k] = v
			}
		}

		// 应用映射
		hasError := false
		for _, mapping := range mappings {
			if err := d.applyMapping(item, transformedItem, mapping); err != nil {
				if !globalConfig.IgnoreErrors {
					hasError = true
					break
				}
				// 记录字段统计
				if fieldStats[mapping.TargetField] == nil {
					fieldStats[mapping.TargetField] = make(map[string]int)
				}
				fieldStats[mapping.TargetField]["errors"]++
			} else {
				// 记录字段统计
				if fieldStats[mapping.TargetField] == nil {
					fieldStats[mapping.TargetField] = make(map[string]int)
				}
				fieldStats[mapping.TargetField]["success"]++
			}
		}

		if !hasError {
			transformedData = append(transformedData, transformedItem)
			stats["transformed_count"] = stats["transformed_count"].(int) + 1
		} else {
			stats["error_count"] = stats["error_count"].(int) + 1
		}
	}

	stats["field_stats"] = fieldStats
	return transformedData, stats
}

// applyMapping 应用单个映射
func (d *DataTransformNode) applyMapping(sourceItem, targetItem map[string]interface{}, mapping MappingConfig) error {
	// 获取源值
	sourceValue, exists := sourceItem[mapping.SourceField]
	if !exists {
		if mapping.DefaultValue != nil {
			sourceValue = mapping.DefaultValue
		} else {
			return fmt.Errorf("源字段 %s 不存在", mapping.SourceField)
		}
	}

	// 根据转换类型处理
	var transformedValue interface{}
	var err error

	switch mapping.TransformType {
	case "direct":
		transformedValue = sourceValue
	case "format":
		transformedValue, err = d.formatValue(sourceValue, mapping.TransformConfig)
	case "calculate":
		transformedValue, err = d.calculateValue(sourceValue, mapping.TransformConfig)
	case "lookup":
		transformedValue, err = d.lookupValue(sourceValue, mapping.TransformConfig)
	case "regex":
		transformedValue, err = d.regexValue(sourceValue, mapping.TransformConfig)
	default:
		return fmt.Errorf("未知的转换类型: %s", mapping.TransformType)
	}

	if err != nil {
		return err
	}

	// 类型转换
	finalValue, err := d.convertType(transformedValue, mapping.DataType)
	if err != nil {
		return err
	}

	// 设置目标值
	targetItem[mapping.TargetField] = finalValue
	return nil
}

// formatValue 格式化值
func (d *DataTransformNode) formatValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	format, exists := config["format"]
	if !exists {
		return nil, fmt.Errorf("格式化配置缺少format字段")
	}

	formatStr, ok := format.(string)
	if !ok {
		return nil, fmt.Errorf("format必须是字符串")
	}

	// 简单的模板替换
	result := strings.ReplaceAll(formatStr, "{value}", fmt.Sprintf("%v", value))
	return result, nil
}

// calculateValue 计算值
func (d *DataTransformNode) calculateValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	expression, exists := config["expression"]
	if !exists {
		return nil, fmt.Errorf("计算配置缺少expression字段")
	}

	exprStr, ok := expression.(string)
	if !ok {
		return nil, fmt.Errorf("expression必须是字符串")
	}

	// 简单的数学计算
	numValue, err := d.toFloat64(value)
	if err != nil {
		return nil, err
	}

	// 替换占位符
	exprStr = strings.ReplaceAll(exprStr, "{value}", fmt.Sprintf("%f", numValue))

	// 简单的表达式计算（这里只支持基本运算）
	result, err := d.evaluateExpression(exprStr)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// lookupValue 查找值
func (d *DataTransformNode) lookupValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	lookupTable, exists := config["lookup_table"]
	if !exists {
		return nil, fmt.Errorf("查找配置缺少lookup_table字段")
	}

	table, ok := lookupTable.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("lookup_table必须是对象")
	}

	key := fmt.Sprintf("%v", value)
	if result, exists := table[key]; exists {
		return result, nil
	}

	return value, nil // 找不到则返回原值
}

// regexValue 正则处理值
func (d *DataTransformNode) regexValue(value interface{}, config map[string]interface{}) (interface{}, error) {
	pattern, exists := config["regex_pattern"]
	if !exists {
		return nil, fmt.Errorf("正则配置缺少regex_pattern字段")
	}

	patternStr, ok := pattern.(string)
	if !ok {
		return nil, fmt.Errorf("regex_pattern必须是字符串")
	}

	replacement, _ := config["regex_replacement"].(string)

	valueStr := fmt.Sprintf("%v", value)
	regex, err := regexp.Compile(patternStr)
	if err != nil {
		return nil, fmt.Errorf("正则表达式编译失败: %v", err)
	}

	if replacement != "" {
		return regex.ReplaceAllString(valueStr, replacement), nil
	} else {
		matches := regex.FindStringSubmatch(valueStr)
		if len(matches) > 1 {
			return matches[1], nil // 返回第一个捕获组
		}
		return valueStr, nil
	}
}

// convertType 类型转换
func (d *DataTransformNode) convertType(value interface{}, targetType string) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch targetType {
	case "string":
		return fmt.Sprintf("%v", value), nil
	case "number":
		return d.toFloat64(value)
	case "boolean":
		return d.toBool(value), nil
	case "array":
		if reflect.TypeOf(value).Kind() == reflect.Slice {
			return value, nil
		}
		return []interface{}{value}, nil
	case "object":
		if reflect.TypeOf(value).Kind() == reflect.Map {
			return value, nil
		}
		return map[string]interface{}{"value": value}, nil
	default:
		return value, nil
	}
}

// evaluateExpression 简单表达式计算
func (d *DataTransformNode) evaluateExpression(expr string) (float64, error) {
	// 这里只是一个简单的实现，实际应用中可以使用更强大的表达式引擎
	expr = strings.TrimSpace(expr)

	// 支持基本的数学运算
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return a + b, nil
			}
		}
	}

	if strings.Contains(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return a - b, nil
			}
		}
	}

	if strings.Contains(expr, "*") {
		parts := strings.Split(expr, "*")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return a * b, nil
			}
		}
	}

	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil && b != 0 {
				return a / b, nil
			}
		}
	}

	// 尝试直接解析为数字
	return strconv.ParseFloat(expr, 64)
}

// 工具函数
func (d *DataTransformNode) toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("无法转换为数字: %v", value)
	}
}

func (d *DataTransformNode) toBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "0" && v != "false"
	case int, int64:
		return v != 0
	case float64, float32:
		return v != 0
	default:
		return false
	}
}

// 配置获取函数
func (d *DataTransformNode) getMappings(config map[string]interface{}) []MappingConfig {
	mappingsRaw, exists := config["mappings"]
	if !exists {
		return []MappingConfig{}
	}

	mappings, ok := mappingsRaw.([]interface{})
	if !ok {
		return []MappingConfig{}
	}

	result := make([]MappingConfig, 0, len(mappings))
	for _, mappingRaw := range mappings {
		mapping, ok := mappingRaw.(map[string]interface{})
		if !ok {
			continue
		}

		config := MappingConfig{
			SourceField:   mapping["source_field"].(string),
			TargetField:   mapping["target_field"].(string),
			TransformType: mapping["transform_type"].(string),
			DataType:      "string",
		}

		if dataType, exists := mapping["data_type"]; exists {
			config.DataType = dataType.(string)
		}

		if defaultValue, exists := mapping["default_value"]; exists {
			config.DefaultValue = defaultValue
		}

		if transformConfig, exists := mapping["transform_config"]; exists {
			if tc, ok := transformConfig.(map[string]interface{}); ok {
				config.TransformConfig = tc
			}
		}

		result = append(result, config)
	}

	return result
}

// getGlobalConfig 获取全局配置
func (d *DataTransformNode) getGlobalConfig(config map[string]interface{}) *GlobalConfig {
	globalConfig := &GlobalConfig{
		PreserveOriginal: false,
		IgnoreErrors:     true,
		IncludeStats:     true,
	}

	if preserveOriginal, ok := config["preserve_original"].(bool); ok {
		globalConfig.PreserveOriginal = preserveOriginal
	}

	// 由于简化了配置，移除了ignore_errors配置项，保持默认值
	// if ignoreErrors, ok := config["ignore_errors"].(bool); ok {
	//     globalConfig.IgnoreErrors = ignoreErrors
	// }

	// 移除了include_stats配置项，保持默认值
	// if includeStats, ok := config["include_stats"].(bool); ok {
	//     globalConfig.IncludeStats = includeStats
	// }

	return globalConfig
}

// 配置结构体
type MappingConfig struct {
	SourceField     string                 `json:"source_field"`
	TargetField     string                 `json:"target_field"`
	TransformType   string                 `json:"transform_type"`
	TransformConfig map[string]interface{} `json:"transform_config"`
	DataType        string                 `json:"data_type"`
	DefaultValue    interface{}            `json:"default_value"`
}

type GlobalConfig struct {
	PreserveOriginal bool `json:"preserve_original"`
	IgnoreErrors     bool `json:"ignore_errors"`
	IncludeStats     bool `json:"include_stats"`
}

// GetDynamicData 获取动态配置数据（默认实现）
func (d *DataTransformNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("数据转换节点暂不支持动态数据获取方法: %s", method)
}
