/*
 * @module: flow-service/api/controllers/response
 * @description: 统一API响应结构和便利函数，简化控制器响应处理
 * @architecture: MVC架构，响应包装层
 * @documentReference: /docs/api-response-structure.md
 * @stateFlow: 无状态响应包装
 * @rules:
 *   - 所有API响应必须使用统一格式
 *   - 错误响应包含标准错误码和消息
 *   - 成功响应包含数据和状态
 * @dependencies:
 *   - github.com/go-chi/render
 * @refs:
 *   - datahub-service/api/controllers/response.go
 */

package controllers

import (
	"net/http"

	"github.com/go-chi/render"
)

// APIResponse 统一API响应结构
type APIResponse struct {
	Status int         `json:"status" example:"0"`
	Msg    string      `json:"msg" example:"操作成功"`
	Data   interface{} `json:"data,omitempty"`
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Status int         `json:"status" example:"0"`
	Msg    string      `json:"msg" example:"操作成功"`
	Data   interface{} `json:"data"`
	Total  int64       `json:"total" example:"100"`
	Page   int         `json:"page" example:"1"`
	Size   int         `json:"size" example:"10"`
}

// Response 实现render.Renderer接口
func (a *APIResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// PaginatedResponse 实现render.Renderer接口
func (p *PaginatedResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// SuccessResponse 创建成功响应
func SuccessResponse(msg string, data interface{}) render.Renderer {
	return &APIResponse{
		Status: 0,
		Msg:    msg,
		Data:   data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(httpStatus int, msg string, err error) render.Renderer {
	response := &APIResponse{
		Status: httpStatus,
		Msg:    msg,
	}

	if err != nil {
		response.Data = map[string]string{"error": err.Error()}
	}

	return response
}

// PaginatedSuccessResponse 创建分页成功响应
func PaginatedSuccessResponse(msg string, data interface{}, total int64, page, size int) render.Renderer {
	return &PaginatedResponse{
		Status: 0,
		Msg:    msg,
		Data:   data,
		Total:  total,
		Page:   page,
		Size:   size,
	}
}
