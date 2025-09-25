/**
 * @module logger_output
 * @description 日志输出节点，将数据输出到不同级别的日志中
 * @architecture 输出插件实现，支持多种日志级别和格式化选项
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow logger_states: configured -> formatting -> logging -> completed
 * @rules 支持多种日志级别，提供格式化选项，便于调试和监控
 * @dependencies log, encoding/json, context, time
 * @refs service/nodes/interface.go
 */

package output

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册日志输出节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewLoggerNode()); err != nil {
		log.Printf("注册日志输出节点失败: %v", err)
	} else {
		log.Println("日志输出节点注册成功")
	}
}

// LoggerNode 日志输出节点
type LoggerNode struct{}

// NewLoggerNode 创建日志输出节点
func NewLoggerNode() *LoggerNode {
	return &LoggerNode{}
}

// GetMetadata 获取节点元数据
func (l *LoggerNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "logger_output",
		Name:        "日志输出器",
		Description: "将数据输出到日志文件或控制台",
		Version:     "1.0.0",
		Category:    nodes.CategoryOutput,
		Type:        nodes.TypeLogger,
		Icon:        "log",
		Tags:        []string{"日志", "输出", "调试"},

		InputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "输入数据",
				Description: "需要记录的数据",
				DataType:    nodes.DataTypeAny,
				Required:    true,
				Multiple:    false,
			},
		},

		OutputPorts: []nodes.PortDefinition{},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "level",
					Type:        "string",
					Title:       "日志级别",
					Description: "日志输出级别",
					Default:     "info",
					Enum:        []interface{}{"debug", "info", "warn", "error"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "format",
					Type:        "string",
					Title:       "输出格式",
					Description: "日志输出格式",
					Default:     "json",
					Enum:        []interface{}{"json", "text", "table"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "prefix",
					Type:        "string",
					Title:       "日志前缀",
					Description: "日志消息前缀",
					Default:     "[FLOW]",
					Widget:      nodes.WidgetText,
				},
				{
					Name:        "max_length",
					Type:        "number",
					Title:       "最大长度",
					Description: "单条日志最大长度，0表示不限制",
					Default:     1000,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "include_timestamp",
					Type:        "boolean",
					Title:       "包含时间戳",
					Description: "是否在日志中包含时间戳",
					Default:     true,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "include_metadata",
					Type:        "boolean",
					Title:       "包含元数据",
					Description: "是否包含数据类型、大小等元数据信息",
					Default:     true,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "log_empty_data",
					Type:        "boolean",
					Title:       "记录空数据",
					Description: "是否记录空数据或null值",
					Default:     false,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "log_arrays",
					Type:        "boolean",
					Title:       "记录数组数据",
					Description: "是否记录数组类型的数据",
					Default:     true,
					Widget:      nodes.WidgetBoolean,
				},
				{
					Name:        "max_array_items",
					Type:        "number",
					Title:       "数组最大项数",
					Description: "记录数组数据时的最大项数，0表示不限制",
					Default:     10,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "custom_message",
					Type:        "string",
					Title:       "自定义消息",
					Description: "自定义日志消息模板",
					Widget:      nodes.WidgetTextarea,
					Placeholder: "处理了 {count} 条数据",
				},
			},
			Required: []string{},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (l *LoggerNode) Validate(config map[string]interface{}) error {
	// 验证日志级别
	if level, ok := config["level"].(string); ok {
		validLevels := []string{"debug", "info", "warn", "error"}
		valid := false
		for _, validLevel := range validLevels {
			if level == validLevel {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid log level: %s", level)
		}
	}

	// 验证输出格式
	if format, ok := config["format"].(string); ok {
		validFormats := []string{"json", "text", "table"}
		valid := false
		for _, validFormat := range validFormats {
			if format == validFormat {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid format: %s", format)
		}
	}

	// 验证输出目标
	if target, ok := config["output_target"].(string); ok {
		validTargets := []string{"console", "file", "both"}
		valid := false
		for _, validTarget := range validTargets {
			if target == validTarget {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid output_target: %s", target)
		}
	}

	return nil
}

// Execute 执行节点
func (l *LoggerNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()
	output := &nodes.NodeOutput{
		Data:    make(map[string]interface{}),
		Logs:    []string{},
		Metrics: make(map[string]interface{}),
		Success: false,
	}

	// 解析配置
	loggingConfig := l.getLoggingConfig(input.Config)
	formatConfig := l.getFormatConfig(input.Config)
	filterConfig := l.getFilterConfig(input.Config)

	// 获取输入数据
	data, exists := input.Data["data"]
	if !exists && !filterConfig.LogEmptyData {
		output.Error = "no input data and empty data logging is disabled"
		output.Duration = time.Since(startTime)
		return output, nil
	}

	// 获取自定义消息
	customMessage := ""
	if msg, exists := input.Data["message"]; exists {
		if msgStr, ok := msg.(string); ok {
			customMessage = msgStr
		}
	}

	// 格式化数据
	formattedData, dataInfo := l.formatData(data, formatConfig, filterConfig)

	// 构建日志消息
	logMessage := l.buildLogMessage(formattedData, customMessage, loggingConfig, dataInfo)

	// 输出日志
	l.outputLog(logMessage, loggingConfig.Level)

	// 构建输出结果
	result := map[string]interface{}{
		"logged":       true,
		"log_level":    loggingConfig.Level,
		"message_size": len(logMessage),
		"data_type":    dataInfo.DataType,
		"data_size":    dataInfo.DataSize,
	}

	if loggingConfig.IncludeMetadata {
		result["logged_at"] = time.Now().Format(time.RFC3339)
		result["node_id"] = "logger_output"
		result["execution_time"] = time.Since(startTime).Milliseconds()
	}

	output.Data["result"] = result
	output.Success = true
	output.Duration = time.Since(startTime)
	output.Logs = append(output.Logs, fmt.Sprintf("数据已输出到%s级别日志", loggingConfig.Level))

	// 添加指标
	output.Metrics["log_duration"] = time.Since(startTime).Milliseconds()
	output.Metrics["message_length"] = len(logMessage)
	output.Metrics["data_items"] = dataInfo.ItemCount

	return output, nil
}

// formatData 格式化数据
func (l *LoggerNode) formatData(data interface{}, formatConfig *FormatConfig, filterConfig *FilterConfig) (string, *DataInfo) {
	dataInfo := l.analyzeData(data)

	// 检查是否应该记录
	if !l.shouldLogData(data, dataInfo, filterConfig) {
		return "", dataInfo
	}

	// 处理数组数据截断
	processedData := l.processArrayData(data, filterConfig)

	var formatted string
	var err error

	switch formatConfig.OutputFormat {
	case "json":
		formatted, err = l.formatAsJSON(processedData, false)
	case "pretty_json":
		formatted, err = l.formatAsJSON(processedData, true)
	case "text":
		formatted = l.formatAsText(processedData, formatConfig.ShowDataType)
	case "table":
		formatted = l.formatAsTable(processedData)
	default:
		formatted, err = l.formatAsJSON(processedData, false)
	}

	if err != nil {
		formatted = fmt.Sprintf("Format error: %v, Raw data: %v", err, data)
	}

	// 截断过长的消息
	if formatConfig.MaxLength > 0 && len(formatted) > formatConfig.MaxLength {
		formatted = formatted[:formatConfig.MaxLength] + formatConfig.TruncateMessage
	}

	return formatted, dataInfo
}

// formatAsJSON 格式化为JSON
func (l *LoggerNode) formatAsJSON(data interface{}, pretty bool) (string, error) {
	var bytes []byte
	var err error

	if pretty {
		bytes, err = json.MarshalIndent(data, "", "  ")
	} else {
		bytes, err = json.Marshal(data)
	}

	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// formatAsText 格式化为文本
func (l *LoggerNode) formatAsText(data interface{}, showType bool) string {
	if showType {
		return fmt.Sprintf("%T: %v", data, data)
	}
	return fmt.Sprintf("%v", data)
}

// formatAsTable 格式化为表格（简单实现）
func (l *LoggerNode) formatAsTable(data interface{}) string {
	switch v := data.(type) {
	case []interface{}:
		if len(v) == 0 {
			return "Empty array"
		}
		// 简单的表格格式
		var result strings.Builder
		result.WriteString(fmt.Sprintf("Array with %d items:\n", len(v)))
		for i, item := range v {
			result.WriteString(fmt.Sprintf("  [%d] %v\n", i, item))
		}
		return result.String()
	case map[string]interface{}:
		var result strings.Builder
		result.WriteString("Object:\n")
		for key, value := range v {
			result.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
		return result.String()
	default:
		return fmt.Sprintf("Value: %v", data)
	}
}

// buildLogMessage 构建日志消息
func (l *LoggerNode) buildLogMessage(formattedData, customMessage string, loggingConfig *LoggingConfig, dataInfo *DataInfo) string {
	var parts []string

	// 添加前缀
	if loggingConfig.Prefix != "" {
		parts = append(parts, loggingConfig.Prefix)
	}

	// 添加时间戳
	if loggingConfig.IncludeTimestamp {
		parts = append(parts, time.Now().Format("2006-01-02 15:04:05"))
	}

	// 添加自定义消息
	if customMessage != "" {
		parts = append(parts, customMessage)
	}

	// 添加元数据
	if loggingConfig.IncludeMetadata && dataInfo != nil {
		metaPart := fmt.Sprintf("(Type:%s, Size:%d)", dataInfo.DataType, dataInfo.DataSize)
		parts = append(parts, metaPart)
	}

	// 添加数据
	if formattedData != "" {
		parts = append(parts, formattedData)
	} else {
		parts = append(parts, "<empty data>")
	}

	return strings.Join(parts, " ")
}

// outputLog 输出日志
func (l *LoggerNode) outputLog(message, level string) {
	switch level {
	case "debug":
		log.Printf("[DEBUG] %s", message)
	case "info":
		log.Printf("[INFO] %s", message)
	case "warn":
		log.Printf("[WARN] %s", message)
	case "error":
		log.Printf("[ERROR] %s", message)
	default:
		log.Printf("[INFO] %s", message)
	}
}

// analyzeData 分析数据
func (l *LoggerNode) analyzeData(data interface{}) *DataInfo {
	info := &DataInfo{}

	if data == nil {
		info.DataType = "null"
		info.DataSize = 0
		info.ItemCount = 0
		return info
	}

	switch v := data.(type) {
	case []interface{}:
		info.DataType = "array"
		info.DataSize = len(v)
		info.ItemCount = len(v)
	case map[string]interface{}:
		info.DataType = "object"
		info.DataSize = len(v)
		info.ItemCount = 1
	case string:
		info.DataType = "string"
		info.DataSize = len(v)
		info.ItemCount = 1
	default:
		info.DataType = fmt.Sprintf("%T", data)
		info.DataSize = 1
		info.ItemCount = 1
	}

	return info
}

// shouldLogData 检查是否应该记录数据
func (l *LoggerNode) shouldLogData(data interface{}, dataInfo *DataInfo, filterConfig *FilterConfig) bool {
	if data == nil {
		return filterConfig.LogEmptyData
	}

	if dataInfo.DataType == "array" && !filterConfig.LogArrays {
		return false
	}

	return true
}

// processArrayData 处理数组数据
func (l *LoggerNode) processArrayData(data interface{}, filterConfig *FilterConfig) interface{} {
	if arr, ok := data.([]interface{}); ok && filterConfig.MaxArrayItems > 0 && len(arr) > filterConfig.MaxArrayItems {
		truncated := make([]interface{}, filterConfig.MaxArrayItems)
		copy(truncated, arr[:filterConfig.MaxArrayItems])
		return append(truncated, fmt.Sprintf("... and %d more items", len(arr)-filterConfig.MaxArrayItems))
	}
	return data
}

// 配置获取方法
func (l *LoggerNode) getLoggingConfig(config map[string]interface{}) *LoggingConfig {
	loggingConfig := &LoggingConfig{
		Level:            "info",
		Prefix:           "[FLOW]",
		IncludeTimestamp: true,
		IncludeMetadata:  true,
	}

	if level, ok := config["level"].(string); ok {
		loggingConfig.Level = level
	}

	if prefix, ok := config["prefix"].(string); ok {
		loggingConfig.Prefix = prefix
	}

	if includeTimestamp, ok := config["include_timestamp"].(bool); ok {
		loggingConfig.IncludeTimestamp = includeTimestamp
	}

	if includeMetadata, ok := config["include_metadata"].(bool); ok {
		loggingConfig.IncludeMetadata = includeMetadata
	}

	return loggingConfig
}

func (l *LoggerNode) getFormatConfig(config map[string]interface{}) *FormatConfig {
	formatConfig := &FormatConfig{
		OutputFormat:    "json",
		MaxLength:       1000,
		TruncateMessage: "...",
		ShowDataType:    true,
	}

	if format, ok := config["format"].(string); ok {
		formatConfig.OutputFormat = format
	}

	if maxLength, ok := config["max_length"].(float64); ok {
		formatConfig.MaxLength = int(maxLength)
	}

	// 暂时保持ShowDataType为默认值，可以根据需要添加配置项

	return formatConfig
}

func (l *LoggerNode) getFilterConfig(config map[string]interface{}) *FilterConfig {
	filterConfig := &FilterConfig{
		LogEmptyData:  false,
		LogArrays:     true,
		MaxArrayItems: 10,
	}

	if logEmptyData, ok := config["log_empty_data"].(bool); ok {
		filterConfig.LogEmptyData = logEmptyData
	}

	if logArrays, ok := config["log_arrays"].(bool); ok {
		filterConfig.LogArrays = logArrays
	}

	if maxArrayItems, ok := config["max_array_items"].(float64); ok {
		filterConfig.MaxArrayItems = int(maxArrayItems)
	}

	return filterConfig
}

// 配置结构体
type LoggingConfig struct {
	Level            string `json:"level"`
	Prefix           string `json:"prefix"`
	IncludeTimestamp bool   `json:"include_timestamp"`
	IncludeMetadata  bool   `json:"include_metadata"`
}

type FormatConfig struct {
	OutputFormat    string `json:"output_format"`
	MaxLength       int    `json:"max_length"`
	TruncateMessage string `json:"truncate_message"`
	ShowDataType    bool   `json:"show_data_type"`
}

type FilterConfig struct {
	LogEmptyData  bool `json:"log_empty_data"`
	LogArrays     bool `json:"log_arrays"`
	MaxArrayItems int  `json:"max_array_items"`
}

type DataInfo struct {
	DataType  string `json:"data_type"`
	DataSize  int    `json:"data_size"`
	ItemCount int    `json:"item_count"`
}

// GetDynamicData 获取动态配置数据（默认实现）
func (l *LoggerNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("日志输出节点暂不支持动态数据获取方法: %s", method)
}
