/**
 * @module data_aggregate
 * @description 数据聚合节点，提供分组聚合、统计计算等功能
 * @architecture 数据处理插件实现，支持多种聚合函数和分组规则
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow aggregate_states: configured -> grouping -> aggregating -> completed
 * @rules 支持多字段分组、多种聚合函数、自定义计算表达式
 * @dependencies context, time, math, sort, strconv
 * @refs service/nodes/interface.go
 */

package transform

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册数据聚合节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewDataAggregateNode()); err != nil {
		log.Printf("注册数据聚合节点失败: %v", err)
	} else {
		log.Println("数据聚合节点注册成功")
	}
}

// DataAggregateNode 数据聚合节点
type DataAggregateNode struct{}

// NewDataAggregateNode 创建数据聚合节点
func NewDataAggregateNode() *DataAggregateNode {
	return &DataAggregateNode{}
}

// GetMetadata 获取节点元数据
func (d *DataAggregateNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "data_aggregate",
		Name:        "数据聚合器",
		Description: "对数据进行分组聚合计算",
		Version:     "1.0.0",
		Category:    nodes.CategoryTransform,
		Type:        nodes.TypeDataAggregate,
		Icon:        "aggregate",
		Tags:        []string{"聚合", "分组", "统计"},

		InputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "输入数据",
				Description: "需要聚合的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "aggregated_data",
				Name:        "聚合结果",
				Description: "聚合计算后的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "group_by",
					Type:        "array",
					Title:       "分组字段",
					Description: "用于分组的字段列表",
					Items: &nodes.ConfigField{
						Type: "string",
					},
					Widget: nodes.WidgetJSON,
				},
				{
					Name:        "aggregations",
					Type:        "array",
					Title:       "聚合函数",
					Description: "聚合计算配置",
					Items: &nodes.ConfigField{
						Type: "object",
						Properties: []nodes.ConfigField{
							{
								Name:        "field",
								Type:        "string",
								Title:       "字段名",
								Description: "要聚合的字段",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "function",
								Type:        "string",
								Title:       "聚合函数",
								Description: "聚合计算函数",
								Default:     "count",
								Enum:        []interface{}{"count", "sum", "avg", "min", "max", "first", "last", "distinct_count", "concat", "median", "percentile"},
								Widget:      nodes.WidgetSelect,
							},
							{
								Name:        "alias",
								Type:        "string",
								Title:       "别名",
								Description: "结果字段别名",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "parameters",
								Type:        "object",
								Title:       "函数参数",
								Description: "聚合函数的额外参数",
								Widget:      nodes.WidgetJSON,
							},
						},
						Required: []string{"field", "function"},
					},
				},
				{
					Name:        "having",
					Type:        "array",
					Title:       "过滤条件",
					Description: "聚合后的过滤条件",
					Items: &nodes.ConfigField{
						Type: "object",
						Properties: []nodes.ConfigField{
							{
								Name:        "field",
								Type:        "string",
								Title:       "字段名",
								Description: "聚合结果字段名",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "operator",
								Type:        "string",
								Title:       "操作符",
								Description: "比较操作符",
								Default:     "greater",
								Enum:        []interface{}{"greater", "greater_equal", "less", "less_equal", "equals", "not_equals"},
								Widget:      nodes.WidgetSelect,
							},
							{
								Name:        "value",
								Type:        "string",
								Title:       "比较值",
								Description: "用于比较的值",
								Widget:      nodes.WidgetText,
							},
						},
						Required: []string{"field", "operator", "value"},
					},
				},
				{
					Name:        "order_by",
					Type:        "array",
					Title:       "排序配置",
					Description: "结果排序配置",
					Items: &nodes.ConfigField{
						Type: "object",
						Properties: []nodes.ConfigField{
							{
								Name:        "field",
								Type:        "string",
								Title:       "排序字段",
								Description: "用于排序的字段名",
								Widget:      nodes.WidgetText,
							},
							{
								Name:        "direction",
								Type:        "string",
								Title:       "排序方向",
								Description: "排序方向",
								Default:     "asc",
								Enum:        []interface{}{"asc", "desc"},
								Widget:      nodes.WidgetSelect,
							},
						},
						Required: []string{"field"},
					},
				},
				{
					Name:        "limit",
					Type:        "number",
					Title:       "结果限制",
					Description: "限制返回结果数量，0表示不限制",
					Default:     0,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "include_stats",
					Type:        "boolean",
					Title:       "包含统计信息",
					Description: "是否在输出中包含聚合统计信息",
					Default:     true,
					Widget:      nodes.WidgetBoolean,
				},
			},
			Required: []string{"aggregations"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证配置
func (d *DataAggregateNode) Validate(config map[string]interface{}) error {
	// 验证aggregations字段
	aggregationsRaw, exists := config["aggregations"]
	if !exists {
		return fmt.Errorf("缺少必需的字段: aggregations")
	}

	aggregations, ok := aggregationsRaw.([]interface{})
	if !ok {
		return fmt.Errorf("aggregations必须是数组")
	}

	if len(aggregations) == 0 {
		return fmt.Errorf("aggregations不能为空")
	}

	// 验证每个聚合配置
	for i, aggRaw := range aggregations {
		agg, ok := aggRaw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("aggregations[%d]必须是对象", i)
		}

		// 验证必需字段
		if _, exists := agg["field"]; !exists {
			return fmt.Errorf("aggregations[%d]缺少必需的字段: field", i)
		}
		if _, exists := agg["function"]; !exists {
			return fmt.Errorf("aggregations[%d]缺少必需的字段: function", i)
		}

		// 验证聚合函数
		function, _ := agg["function"].(string)
		validFunctions := []string{"count", "sum", "avg", "min", "max", "first", "last", "distinct_count", "concat", "median", "percentile"}
		valid := false
		for _, validFunc := range validFunctions {
			if function == validFunc {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("aggregations[%d]的function无效: %s", i, function)
		}
	}

	return nil
}

// Execute 执行聚合
func (d *DataAggregateNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
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

	// 获取配置
	groupBy := d.getGroupBy(input.Config)
	aggregations := d.getAggregations(input.Config)
	having := d.getHaving(input.Config)
	orderBy := d.getOrderBy(input.Config)
	limit := d.getLimit(input.Config)
	includeStats := d.getIncludeStats(input.Config)

	// 执行聚合
	aggregatedData, stats := d.aggregateData(data, groupBy, aggregations, having, orderBy, limit)

	// 构建输出
	output := &nodes.NodeOutput{
		Data: map[string]interface{}{
			"aggregated_data": aggregatedData,
		},
		Success:  true,
		Duration: time.Since(startTime),
	}

	if includeStats {
		output.Data["aggregate_stats"] = stats
	}

	return output, nil
}

// aggregateData 聚合数据
func (d *DataAggregateNode) aggregateData(data []interface{}, groupBy []string, aggregations []AggregationConfig, having []HavingConfig, orderBy []OrderByConfig, limit int) ([]interface{}, map[string]interface{}) {
	stats := map[string]interface{}{
		"total_records":    len(data),
		"groups_count":     0,
		"aggregated_count": 0,
		"filtered_count":   0,
	}

	// 分组
	groups := d.groupData(data, groupBy)
	stats["groups_count"] = len(groups)

	// 聚合
	aggregatedData := make([]interface{}, 0, len(groups))
	for groupKey, groupData := range groups {
		result := make(map[string]interface{})

		// 设置分组字段
		if len(groupBy) > 0 {
			groupValues := strings.Split(groupKey, "|||")
			for i, field := range groupBy {
				if i < len(groupValues) {
					result[field] = groupValues[i]
				}
			}
		}

		// 执行聚合函数
		for _, agg := range aggregations {
			value, err := d.executeAggregation(groupData, agg)
			if err != nil {
				continue
			}

			fieldName := agg.Alias
			if fieldName == "" {
				fieldName = fmt.Sprintf("%s_%s", agg.Function, agg.Field)
			}
			result[fieldName] = value
		}

		aggregatedData = append(aggregatedData, result)
	}

	stats["aggregated_count"] = len(aggregatedData)

	// Having过滤
	if len(having) > 0 {
		filteredData := make([]interface{}, 0, len(aggregatedData))
		for _, item := range aggregatedData {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if d.evaluateHaving(itemMap, having) {
					filteredData = append(filteredData, item)
				}
			}
		}
		aggregatedData = filteredData
		stats["filtered_count"] = len(aggregatedData)
	}

	// 排序
	if len(orderBy) > 0 {
		d.sortData(aggregatedData, orderBy)
	}

	// 限制结果数量
	if limit > 0 && len(aggregatedData) > limit {
		aggregatedData = aggregatedData[:limit]
	}

	return aggregatedData, stats
}

// groupData 分组数据
func (d *DataAggregateNode) groupData(data []interface{}, groupBy []string) map[string][]map[string]interface{} {
	groups := make(map[string][]map[string]interface{})

	for _, itemRaw := range data {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// 生成分组键
		var groupKey string
		if len(groupBy) == 0 {
			groupKey = "all"
		} else {
			keyParts := make([]string, len(groupBy))
			for i, field := range groupBy {
				if value, exists := item[field]; exists {
					keyParts[i] = fmt.Sprintf("%v", value)
				} else {
					keyParts[i] = ""
				}
			}
			groupKey = strings.Join(keyParts, "|||")
		}

		// 添加到对应分组
		if groups[groupKey] == nil {
			groups[groupKey] = make([]map[string]interface{}, 0)
		}
		groups[groupKey] = append(groups[groupKey], item)
	}

	return groups
}

// executeAggregation 执行聚合函数
func (d *DataAggregateNode) executeAggregation(data []map[string]interface{}, agg AggregationConfig) (interface{}, error) {
	switch agg.Function {
	case "count":
		return len(data), nil
	case "sum":
		return d.sum(data, agg.Field)
	case "avg":
		return d.avg(data, agg.Field)
	case "min":
		return d.min(data, agg.Field)
	case "max":
		return d.max(data, agg.Field)
	case "first":
		return d.first(data, agg.Field)
	case "last":
		return d.last(data, agg.Field)
	case "distinct_count":
		return d.distinctCount(data, agg.Field)
	case "concat":
		separator := ","
		if sep, exists := agg.Parameters["separator"]; exists {
			if s, ok := sep.(string); ok {
				separator = s
			}
		}
		return d.concat(data, agg.Field, separator)
	case "median":
		return d.median(data, agg.Field)
	case "percentile":
		percentile := 50.0
		if p, exists := agg.Parameters["percentile"]; exists {
			if pf, ok := p.(float64); ok {
				percentile = pf
			}
		}
		return d.percentile(data, agg.Field, percentile)
	default:
		return nil, fmt.Errorf("未知的聚合函数: %s", agg.Function)
	}
}

// 聚合函数实现
func (d *DataAggregateNode) sum(data []map[string]interface{}, field string) (float64, error) {
	sum := 0.0
	count := 0
	for _, item := range data {
		if value, exists := item[field]; exists {
			if numValue, err := d.toFloat64(value); err == nil {
				sum += numValue
				count++
			}
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("没有有效的数值")
	}
	return sum, nil
}

func (d *DataAggregateNode) avg(data []map[string]interface{}, field string) (float64, error) {
	sum, err := d.sum(data, field)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range data {
		if value, exists := item[field]; exists {
			if _, err := d.toFloat64(value); err == nil {
				count++
			}
		}
	}
	if count == 0 {
		return 0, fmt.Errorf("没有有效的数值")
	}
	return sum / float64(count), nil
}

func (d *DataAggregateNode) min(data []map[string]interface{}, field string) (interface{}, error) {
	var min interface{}
	for _, item := range data {
		if value, exists := item[field]; exists {
			if min == nil {
				min = value
			} else {
				if d.compareValues(value, min) < 0 {
					min = value
				}
			}
		}
	}
	if min == nil {
		return nil, fmt.Errorf("没有有效的值")
	}
	return min, nil
}

func (d *DataAggregateNode) max(data []map[string]interface{}, field string) (interface{}, error) {
	var max interface{}
	for _, item := range data {
		if value, exists := item[field]; exists {
			if max == nil {
				max = value
			} else {
				if d.compareValues(value, max) > 0 {
					max = value
				}
			}
		}
	}
	if max == nil {
		return nil, fmt.Errorf("没有有效的值")
	}
	return max, nil
}

func (d *DataAggregateNode) first(data []map[string]interface{}, field string) (interface{}, error) {
	for _, item := range data {
		if value, exists := item[field]; exists {
			return value, nil
		}
	}
	return nil, fmt.Errorf("没有有效的值")
}

func (d *DataAggregateNode) last(data []map[string]interface{}, field string) (interface{}, error) {
	for i := len(data) - 1; i >= 0; i-- {
		if value, exists := data[i][field]; exists {
			return value, nil
		}
	}
	return nil, fmt.Errorf("没有有效的值")
}

func (d *DataAggregateNode) distinctCount(data []map[string]interface{}, field string) (int, error) {
	distinct := make(map[string]bool)
	for _, item := range data {
		if value, exists := item[field]; exists {
			key := fmt.Sprintf("%v", value)
			distinct[key] = true
		}
	}
	return len(distinct), nil
}

func (d *DataAggregateNode) concat(data []map[string]interface{}, field string, separator string) (string, error) {
	var values []string
	for _, item := range data {
		if value, exists := item[field]; exists {
			values = append(values, fmt.Sprintf("%v", value))
		}
	}
	return strings.Join(values, separator), nil
}

func (d *DataAggregateNode) median(data []map[string]interface{}, field string) (float64, error) {
	var values []float64
	for _, item := range data {
		if value, exists := item[field]; exists {
			if numValue, err := d.toFloat64(value); err == nil {
				values = append(values, numValue)
			}
		}
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("没有有效的数值")
	}
	sort.Float64s(values)
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2, nil
	}
	return values[n/2], nil
}

func (d *DataAggregateNode) percentile(data []map[string]interface{}, field string, p float64) (float64, error) {
	var values []float64
	for _, item := range data {
		if value, exists := item[field]; exists {
			if numValue, err := d.toFloat64(value); err == nil {
				values = append(values, numValue)
			}
		}
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("没有有效的数值")
	}
	sort.Float64s(values)
	index := (p / 100) * float64(len(values)-1)
	if index == float64(int(index)) {
		return values[int(index)], nil
	}
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	weight := index - float64(lower)
	return values[lower]*(1-weight) + values[upper]*weight, nil
}

// evaluateHaving 评估Having条件
func (d *DataAggregateNode) evaluateHaving(item map[string]interface{}, having []HavingConfig) bool {
	for _, condition := range having {
		if !d.evaluateHavingCondition(item, condition) {
			return false
		}
	}
	return true
}

func (d *DataAggregateNode) evaluateHavingCondition(item map[string]interface{}, condition HavingConfig) bool {
	fieldValue, exists := item[condition.Field]
	if !exists {
		return false
	}

	compareValue, err := d.parseValue(condition.Value)
	if err != nil {
		return false
	}

	switch condition.Operator {
	case "eq":
		return d.compareValues(fieldValue, compareValue) == 0
	case "ne":
		return d.compareValues(fieldValue, compareValue) != 0
	case "gt":
		return d.compareValues(fieldValue, compareValue) > 0
	case "gte":
		return d.compareValues(fieldValue, compareValue) >= 0
	case "lt":
		return d.compareValues(fieldValue, compareValue) < 0
	case "lte":
		return d.compareValues(fieldValue, compareValue) <= 0
	default:
		return false
	}
}

// sortData 排序数据
func (d *DataAggregateNode) sortData(data []interface{}, orderBy []OrderByConfig) {
	sort.Slice(data, func(i, j int) bool {
		itemI, okI := data[i].(map[string]interface{})
		itemJ, okJ := data[j].(map[string]interface{})
		if !okI || !okJ {
			return false
		}

		for _, order := range orderBy {
			valueI, existsI := itemI[order.Field]
			valueJ, existsJ := itemJ[order.Field]

			if !existsI && !existsJ {
				continue
			}
			if !existsI {
				return order.Direction == "asc"
			}
			if !existsJ {
				return order.Direction == "desc"
			}

			cmp := d.compareValues(valueI, valueJ)
			if cmp != 0 {
				if order.Direction == "desc" {
					return cmp > 0
				}
				return cmp < 0
			}
		}
		return false
	})
}

// 工具函数
func (d *DataAggregateNode) toFloat64(value interface{}) (float64, error) {
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

func (d *DataAggregateNode) parseValue(value string) (interface{}, error) {
	// 尝试解析为数字
	if numValue, err := strconv.ParseFloat(value, 64); err == nil {
		return numValue, nil
	}
	// 尝试解析为布尔值
	if boolValue, err := strconv.ParseBool(value); err == nil {
		return boolValue, nil
	}
	// 返回字符串
	return value, nil
}

func (d *DataAggregateNode) compareValues(a, b interface{}) int {
	// 尝试数字比较
	if aNum, err1 := d.toFloat64(a); err1 == nil {
		if bNum, err2 := d.toFloat64(b); err2 == nil {
			if aNum < bNum {
				return -1
			} else if aNum > bNum {
				return 1
			}
			return 0
		}
	}

	// 字符串比较
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// 配置获取函数
func (d *DataAggregateNode) getGroupBy(config map[string]interface{}) []string {
	groupByRaw, exists := config["group_by"]
	if !exists {
		return []string{}
	}

	groupBy, ok := groupByRaw.([]interface{})
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(groupBy))
	for _, field := range groupBy {
		if fieldStr, ok := field.(string); ok {
			result = append(result, fieldStr)
		}
	}

	return result
}

func (d *DataAggregateNode) getAggregations(config map[string]interface{}) []AggregationConfig {
	aggregationsRaw, exists := config["aggregations"]
	if !exists {
		return []AggregationConfig{}
	}

	aggregations, ok := aggregationsRaw.([]interface{})
	if !ok {
		return []AggregationConfig{}
	}

	result := make([]AggregationConfig, 0, len(aggregations))
	for _, aggRaw := range aggregations {
		agg, ok := aggRaw.(map[string]interface{})
		if !ok {
			continue
		}

		config := AggregationConfig{
			Field:      agg["field"].(string),
			Function:   agg["function"].(string),
			Parameters: make(map[string]interface{}),
		}

		if alias, exists := agg["alias"]; exists {
			config.Alias = alias.(string)
		}

		if params, exists := agg["parameters"]; exists {
			if paramsMap, ok := params.(map[string]interface{}); ok {
				config.Parameters = paramsMap
			}
		}

		result = append(result, config)
	}

	return result
}

func (d *DataAggregateNode) getHaving(config map[string]interface{}) []HavingConfig {
	havingRaw, exists := config["having"]
	if !exists {
		return []HavingConfig{}
	}

	having, ok := havingRaw.([]interface{})
	if !ok {
		return []HavingConfig{}
	}

	result := make([]HavingConfig, 0, len(having))
	for _, condRaw := range having {
		cond, ok := condRaw.(map[string]interface{})
		if !ok {
			continue
		}

		config := HavingConfig{
			Field:    cond["field"].(string),
			Operator: cond["operator"].(string),
			Value:    cond["value"].(string),
		}

		result = append(result, config)
	}

	return result
}

func (d *DataAggregateNode) getOrderBy(config map[string]interface{}) []OrderByConfig {
	orderByRaw, exists := config["order_by"]
	if !exists {
		return []OrderByConfig{}
	}

	orderBy, ok := orderByRaw.([]interface{})
	if !ok {
		return []OrderByConfig{}
	}

	result := make([]OrderByConfig, 0, len(orderBy))
	for _, orderRaw := range orderBy {
		order, ok := orderRaw.(map[string]interface{})
		if !ok {
			continue
		}

		config := OrderByConfig{
			Field:     order["field"].(string),
			Direction: "asc",
		}

		if direction, exists := order["direction"]; exists {
			config.Direction = direction.(string)
		}

		result = append(result, config)
	}

	return result
}

func (d *DataAggregateNode) getLimit(config map[string]interface{}) int {
	limitRaw, exists := config["limit"]
	if !exists {
		return 0
	}

	if limit, ok := limitRaw.(float64); ok {
		return int(limit)
	}

	return 0
}

func (d *DataAggregateNode) getIncludeStats(config map[string]interface{}) bool {
	includeStatsRaw, exists := config["include_stats"]
	if !exists {
		return true
	}

	if includeStats, ok := includeStatsRaw.(bool); ok {
		return includeStats
	}

	return true
}

// 配置结构体
type AggregationConfig struct {
	Field      string                 `json:"field"`
	Function   string                 `json:"function"`
	Alias      string                 `json:"alias,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type HavingConfig struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type OrderByConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// GetDynamicData 获取动态配置数据（默认实现）
func (d *DataAggregateNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("数据聚合节点暂不支持动态数据获取方法: %s", method)
}
