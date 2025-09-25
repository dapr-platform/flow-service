/**
 * @module api_datasource
 * @description API数据源节点，提供HTTP API调用功能
 * @architecture HTTP客户端插件实现，支持多种HTTP方法和认证方式
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow api_states: configured -> connecting -> requesting -> completed
 * @rules 必须支持常用HTTP方法，提供错误重试机制，支持多种认证方式
 * @dependencies net/http, context, time, encoding/json
 * @refs service/nodes/interface.go
 */

package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"flow-service/service/nodes"
)

// init 自动注册API节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewAPINode()); err != nil {
		log.Printf("注册API节点失败: %v", err)
	} else {
		log.Println("API节点注册成功")
	}
}

// APINode API数据源节点
type APINode struct {
	client *http.Client
}

// NewAPINode 创建API节点
func NewAPINode() *APINode {
	return &APINode{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMetadata 获取节点元数据
func (a *APINode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "api_datasource",
		Name:        "API数据源",
		Description: "从HTTP API获取数据",
		Version:     "1.0.0",
		Category:    nodes.CategoryDataSource,
		Type:        nodes.TypeAPI,
		Icon:        "api",
		Tags:        []string{"HTTP", "API", "REST"},

		InputPorts: []nodes.PortDefinition{},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "response_data",
				Name:        "响应数据",
				Description: "API响应的数据",
				DataType:    nodes.DataTypeAny,
				Required:    true,
				Multiple:    false,
			},
			{
				ID:          "response_headers",
				Name:        "响应头",
				Description: "API响应头信息",
				DataType:    nodes.DataTypeObject,
				Required:    false,
				Multiple:    false,
			},
			{
				ID:          "status_info",
				Name:        "状态信息",
				Description: "HTTP状态码和相关信息",
				DataType:    nodes.DataTypeObject,
				Required:    false,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "url",
					Type:        "string",
					Title:       "请求URL",
					Description: "HTTP请求的完整URL地址",
					Widget:      nodes.WidgetText,
					Placeholder: "https://api.example.com/data",
				},
				{
					Name:        "method",
					Type:        "string",
					Title:       "HTTP方法",
					Description: "HTTP请求方法",
					Default:     "GET",
					Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "headers",
					Type:        "object",
					Title:       "请求头",
					Description: "HTTP请求头设置",
					Properties: []nodes.ConfigField{
						{
							Name:        "content_type",
							Type:        "string",
							Title:       "Content-Type",
							Description: "请求内容类型",
							Default:     "application/json",
							Widget:      nodes.WidgetSelect,
							Enum:        []interface{}{"application/json", "application/xml", "application/x-www-form-urlencoded", "text/plain"},
						},
						{
							Name:        "accept",
							Type:        "string",
							Title:       "Accept",
							Description: "接受的响应类型",
							Default:     "application/json",
							Widget:      nodes.WidgetText,
						},
						{
							Name:        "custom_headers",
							Type:        "object",
							Title:       "自定义请求头",
							Description: "额外的HTTP请求头",
							Widget:      nodes.WidgetJSON,
						},
					},
				},
				{
					Name:        "body",
					Type:        "string",
					Title:       "请求体",
					Description: "HTTP请求体数据(JSON格式)",
					Widget:      nodes.WidgetCode,
					Placeholder: `{"key": "value"}`,
				},
				{
					Name:        "timeout",
					Type:        "number",
					Title:       "超时时间(秒)",
					Description: "请求超时时间",
					Default:     30,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "auth",
					Type:        "object",
					Title:       "认证配置",
					Description: "API认证相关配置",
					Properties: []nodes.ConfigField{
						{
							Name:        "type",
							Type:        "string",
							Title:       "认证类型",
							Description: "API认证方式",
							Default:     "none",
							Enum:        []interface{}{"none", "basic", "bearer", "api_key"},
							Widget:      nodes.WidgetSelect,
						},
						{
							Name:        "username",
							Type:        "string",
							Title:       "用户名",
							Description: "Basic认证用户名",
							Widget:      nodes.WidgetText,
						},
						{
							Name:        "password",
							Type:        "string",
							Title:       "密码",
							Description: "Basic认证密码",
							Widget:      nodes.WidgetPassword,
						},
						{
							Name:        "token",
							Type:        "string",
							Title:       "Token",
							Description: "Bearer Token或API Key",
							Widget:      nodes.WidgetPassword,
						},
						{
							Name:        "api_key_header",
							Type:        "string",
							Title:       "API Key头名称",
							Description: "API Key在HTTP头中的名称",
							Default:     "X-API-Key",
							Widget:      nodes.WidgetText,
						},
					},
				},
			},
			Required: []string{"url", "method"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (a *APINode) Validate(config map[string]interface{}) error {
	// 验证URL
	if url, ok := config["url"].(string); !ok || url == "" {
		return fmt.Errorf("missing or invalid URL")
	}

	// 验证HTTP方法
	if method, ok := config["method"].(string); !ok || method == "" {
		return fmt.Errorf("missing or invalid HTTP method")
	}

	// 验证认证配置
	if auth, ok := config["auth"].(map[string]interface{}); ok {
		if authType, ok := auth["type"].(string); ok {
			switch authType {
			case "basic":
				if _, ok := auth["username"].(string); !ok {
					return fmt.Errorf("basic auth requires username")
				}
				if _, ok := auth["password"].(string); !ok {
					return fmt.Errorf("basic auth requires password")
				}
			case "bearer", "api_key":
				if _, ok := auth["token"].(string); !ok {
					return fmt.Errorf("%s auth requires token", authType)
				}
			}
		}
	}

	return nil
}

// Execute 执行节点
func (a *APINode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()
	output := &nodes.NodeOutput{
		Data:    make(map[string]interface{}),
		Logs:    []string{},
		Metrics: make(map[string]interface{}),
		Success: false,
	}

	// 解析配置
	url := input.Config["url"].(string)
	method := strings.ToUpper(input.Config["method"].(string))

	// 设置超时
	if timeout, ok := input.Config["timeout"].(float64); ok {
		a.client.Timeout = time.Duration(timeout) * time.Second
	}

	output.Logs = append(output.Logs, fmt.Sprintf("开始请求: %s %s", method, url))

	// 执行请求
	resp, err := a.executeRequest(ctx, input, method, url)
	if err != nil {
		output.Error = fmt.Sprintf("request failed: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		output.Error = fmt.Sprintf("failed to read response body: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}

	// 解析响应数据
	responseData, err := a.parseResponseData(body, resp.Header.Get("Content-Type"))
	if err != nil {
		output.Error = fmt.Sprintf("failed to parse response: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}

	// 构建输出
	output.Data["response_data"] = responseData
	output.Data["response_headers"] = a.headersToMap(resp.Header)
	output.Data["status_info"] = map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	output.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
	output.Duration = time.Since(startTime)
	output.Logs = append(output.Logs, fmt.Sprintf("请求完成，状态码: %d", resp.StatusCode))

	// 添加指标
	output.Metrics["request_duration"] = time.Since(startTime).Milliseconds()
	output.Metrics["status_code"] = resp.StatusCode
	output.Metrics["response_size"] = len(body)

	return output, nil
}

// executeRequest 执行单次请求
func (a *APINode) executeRequest(ctx context.Context, input *nodes.NodeInput, method, url string) (*http.Response, error) {
	// 准备请求体
	var reqBody io.Reader
	if bodyData, exists := input.Config["body"]; exists && bodyData != nil {
		if bodyStr, ok := bodyData.(string); ok && bodyStr != "" {
			reqBody = strings.NewReader(bodyStr)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	a.setHeaders(req, input.Config)

	// 设置认证
	a.setAuth(req, input.Config)

	// 执行请求
	return a.client.Do(req)
}

// setHeaders 设置请求头
func (a *APINode) setHeaders(req *http.Request, config map[string]interface{}) {
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		if contentType, ok := headers["content_type"].(string); ok {
			req.Header.Set("Content-Type", contentType)
		}
		if accept, ok := headers["accept"].(string); ok {
			req.Header.Set("Accept", accept)
		}
		if customHeaders, ok := headers["custom_headers"].(map[string]interface{}); ok {
			for key, value := range customHeaders {
				if strValue, ok := value.(string); ok {
					req.Header.Set(key, strValue)
				}
			}
		}
	}
}

// setAuth 设置认证
func (a *APINode) setAuth(req *http.Request, config map[string]interface{}) {
	if auth, ok := config["auth"].(map[string]interface{}); ok {
		if authType, ok := auth["type"].(string); ok {
			switch authType {
			case "basic":
				if username, ok := auth["username"].(string); ok {
					if password, ok := auth["password"].(string); ok {
						req.SetBasicAuth(username, password)
					}
				}
			case "bearer":
				if token, ok := auth["token"].(string); ok {
					req.Header.Set("Authorization", "Bearer "+token)
				}
			case "api_key":
				if token, ok := auth["token"].(string); ok {
					headerName := "X-API-Key"
					if name, ok := auth["api_key_header"].(string); ok {
						headerName = name
					}
					req.Header.Set(headerName, token)
				}
			}
		}
	}
}

// parseResponseData 解析响应数据
func (a *APINode) parseResponseData(body []byte, contentType string) (interface{}, error) {
	if len(body) == 0 {
		return nil, nil
	}

	// 根据Content-Type解析数据
	if strings.Contains(strings.ToLower(contentType), "application/json") {
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			// 如果JSON解析失败，返回原始字符串
			return string(body), nil
		}
		return data, nil
	}

	// 其他类型返回字符串
	return string(body), nil
}

// headersToMap 将HTTP头转换为map
func (a *APINode) headersToMap(headers http.Header) map[string]interface{} {
	result := make(map[string]interface{})
	for key, values := range headers {
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			result[key] = values
		}
	}
	return result
}

// GetDynamicData 获取动态配置数据（默认实现）
func (a *APINode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("API节点暂不支持动态数据获取方法: %s", method)
}
