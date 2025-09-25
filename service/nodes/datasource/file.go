/**
 * @module file_datasource
 * @description 文件数据源节点，提供文件读取和解析功能
 * @architecture 文件处理插件实现，支持多种文件格式
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow file_states: configured -> reading -> parsing -> completed
 * @rules 支持CSV、JSON、XML等格式，提供编码检测和错误处理
 * @dependencies os, encoding/csv, encoding/json, context, time
 * @refs service/nodes/interface.go
 */

package datasource

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册文件节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewFileNode()); err != nil {
		log.Printf("注册文件节点失败: %v", err)
	} else {
		log.Println("文件节点注册成功")
	}
}

// FileNode 文件数据源节点
type FileNode struct{}

// NewFileNode 创建文件节点
func NewFileNode() *FileNode {
	return &FileNode{}
}

// GetMetadata 获取节点元数据
func (f *FileNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "file_datasource",
		Name:        "文件数据源",
		Description: "从文件读取数据",
		Version:     "1.0.0",
		Category:    nodes.CategoryDataSource,
		Type:        nodes.TypeFile,
		Icon:        "file",
		Tags:        []string{"文件", "CSV", "JSON", "XML", "YAML"},

		InputPorts: []nodes.PortDefinition{},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "文件数据",
				Description: "从文件读取的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "path",
					Type:        "string",
					Title:       "文件路径",
					Description: "要读取的文件路径",
					Widget:      nodes.WidgetFile,
				},
				{
					Name:        "format",
					Type:        "string",
					Title:       "文件格式",
					Description: "文件数据格式",
					Default:     "csv",
					Enum:        []interface{}{"csv", "json", "txt", "xml", "yaml"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "encoding",
					Type:        "string",
					Title:       "文件编码",
					Description: "文件字符编码",
					Default:     "utf-8",
					Enum:        []interface{}{"utf-8", "gbk", "gb2312"},
					Widget:      nodes.WidgetSelect,
				},
			},
			Required: []string{"path", "format"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (f *FileNode) Validate(config map[string]interface{}) error {
	path, ok := config["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("missing or invalid file path")
	}

	format, ok := config["format"].(string)
	if !ok || format == "" {
		return fmt.Errorf("missing or invalid file format")
	}

	return nil
}

// Execute 执行节点
func (f *FileNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()
	output := &nodes.NodeOutput{
		Data:    make(map[string]interface{}),
		Logs:    []string{},
		Metrics: make(map[string]interface{}),
		Success: false,
	}

	// 解析配置
	path := input.Config["path"].(string)
	format := input.Config["format"].(string)

	output.Logs = append(output.Logs, fmt.Sprintf("开始读取文件: %s", path))

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		output.Error = fmt.Sprintf("文件不存在: %s", path)
		output.Duration = time.Since(startTime)
		return output, nil
	}

	// 读取文件
	data, err := f.readFile(path, format)
	if err != nil {
		output.Error = fmt.Sprintf("读取文件失败: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}

	output.Data["data"] = data
	output.Success = true
	output.Duration = time.Since(startTime)
	output.Logs = append(output.Logs, fmt.Sprintf("文件读取成功，共 %d 条记录", len(data)))

	// 添加指标
	output.Metrics["read_duration"] = time.Since(startTime).Milliseconds()
	output.Metrics["record_count"] = len(data)

	return output, nil
}

// readFile 读取文件
func (f *FileNode) readFile(path, format string) ([]map[string]interface{}, error) {
	switch strings.ToLower(format) {
	case "csv":
		return f.readCSV(path)
	case "json":
		return f.readJSON(path)
	case "txt":
		return f.readText(path)
	case "xml":
		return f.readXML(path)
	case "yaml", "yml":
		return f.readYAML(path)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", format)
	}
}

// readCSV 读取CSV文件
func (f *FileNode) readCSV(path string) ([]map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 第一行作为标题
	headers := records[0]
	var result []map[string]interface{}

	for i := 1; i < len(records); i++ {
		record := make(map[string]interface{})
		for j, value := range records[i] {
			if j < len(headers) {
				record[headers[j]] = value
			}
		}
		result = append(result, record)
	}

	return result, nil
}

// readJSON 读取JSON文件
func (f *FileNode) readJSON(path string) ([]map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	// 转换为统一格式
	switch v := data.(type) {
	case []interface{}:
		var result []map[string]interface{}
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				result = append(result, itemMap)
			}
		}
		return result, nil
	case map[string]interface{}:
		return []map[string]interface{}{v}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON structure")
	}
}

// readText 读取文本文件
func (f *FileNode) readText(path string) ([]map[string]interface{}, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var result []map[string]interface{}

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, map[string]interface{}{
				"line_number": i + 1,
				"content":     line,
			})
		}
	}

	return result, nil
}

// readXML 读取XML文件
func (f *FileNode) readXML(path string) ([]map[string]interface{}, error) {
	// 为了简化，目前返回XML文件的文本内容，后续可以扩展为真正的XML解析
	return f.readText(path)
}

// readYAML 读取YAML文件
func (f *FileNode) readYAML(path string) ([]map[string]interface{}, error) {
	// 为了简化，目前返回YAML文件的文本内容，后续可以扩展为真正的YAML解析
	return f.readText(path)
}

// GetDynamicData 获取动态配置数据（默认实现）
func (f *FileNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("文件节点暂不支持动态数据获取方法: %s", method)
}
