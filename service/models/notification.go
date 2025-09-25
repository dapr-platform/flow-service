/*
 * @module: flow-service/service/models/notification
 * @description: 通知渠道模型定义，支持多种通知方式如邮件、短信、Webhook等
 * @architecture: 数据模型层 - 核心业务模型
 * @documentReference: /docs/notification-model.md
 * @stateFlow: /docs/notification-state-flow.md
 * @rules:
 *   - 通知渠道必须指定有效的类型
 *   - 配置参数必须符合渠道类型要求
 *   - 支持模板化消息内容
 * @dependencies:
 *   - time
 *   - encoding/json
 *   - fmt
 * @refs:
 *   - service/models/task.go
 */

package models

import (
	"time"
)

// NotificationTypeEnum 通知类型枚举
type NotificationTypeEnum string

const (
	// NotificationTypeEmail 邮件通知
	NotificationTypeEmail NotificationTypeEnum = "email"
	// NotificationTypeSMS 短信通知
	NotificationTypeSMS NotificationTypeEnum = "sms"
	// NotificationTypeWebhook Webhook通知
	NotificationTypeWebhook NotificationTypeEnum = "webhook"
	// NotificationTypeSlack Slack通知
	NotificationTypeSlack NotificationTypeEnum = "slack"
	// NotificationTypeDingTalk 钉钉通知
	NotificationTypeDingTalk NotificationTypeEnum = "dingtalk"
	// NotificationTypeWechat 微信通知
	NotificationTypeWechat NotificationTypeEnum = "wechat"
	// NotificationTypeTelegram Telegram通知
	NotificationTypeTelegram NotificationTypeEnum = "telegram"
)

// String 返回类型字符串
func (t NotificationTypeEnum) String() string {
	return string(t)
}

// IsValid 验证类型是否有效
func (t NotificationTypeEnum) IsValid() bool {
	switch t {
	case NotificationTypeEmail, NotificationTypeSMS, NotificationTypeWebhook,
		NotificationTypeSlack, NotificationTypeDingTalk, NotificationTypeWechat,
		NotificationTypeTelegram:
		return true
	default:
		return false
	}
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
	// 渠道ID
	ID string `json:"id" validate:"required"`

	// 渠道名称
	Name string `json:"name" validate:"required"`

	// 渠道类型
	Type NotificationTypeEnum `json:"type" validate:"required"`

	// 渠道配置
	Config map[string]interface{} `json:"config,omitempty"`

	// 消息模板
	Template string `json:"template,omitempty"`

	// 是否启用
	Enabled bool `json:"enabled"`

	// 重试配置
	RetryConfig *NotificationRetryConfig `json:"retry_config,omitempty"`

	// 限流配置
	RateLimitConfig *NotificationRateLimitConfig `json:"rate_limit_config,omitempty"`
}

// NotificationRetryConfig 通知重试配置
type NotificationRetryConfig struct {
	// 最大重试次数
	MaxRetries int `json:"max_retries" validate:"min=0,max=5"`

	// 重试间隔 (单位: 纳秒)
	RetryInterval time.Duration `json:"retry_interval" swaggertype:"integer" validate:"min=0"`

	// 退避策略
	BackoffStrategy string `json:"backoff_strategy" validate:"oneof=fixed linear exponential"`

	// 退避倍数
	BackoffMultiplier float64 `json:"backoff_multiplier" validate:"min=1"`
}

// NotificationRateLimitConfig 通知限流配置
type NotificationRateLimitConfig struct {
	// 限流窗口时间 (单位: 纳秒)
	WindowDuration time.Duration `json:"window_duration" swaggertype:"integer" validate:"min=0"`

	// 窗口内最大请求数
	MaxRequests int `json:"max_requests" validate:"min=1"`

	// 突发请求数
	BurstSize int `json:"burst_size" validate:"min=1"`
}

// NotificationMessage 通知消息
type NotificationMessage struct {
	// 消息ID
	ID string `json:"id"`

	// 渠道ID
	ChannelID string `json:"channel_id"`

	// 消息标题
	Title string `json:"title"`

	// 消息内容
	Content string `json:"content"`

	// 接收者
	Recipients []string `json:"recipients"`

	// 消息数据
	Data map[string]interface{} `json:"data,omitempty"`

	// 优先级
	Priority int `json:"priority" validate:"min=0,max=10"`

	// 发送时间
	SendTime *time.Time `json:"send_time,omitempty"`

	// 过期时间
	ExpireTime *time.Time `json:"expire_time,omitempty"`

	// 创建时间
	CreatedAt time.Time `json:"created_at"`
}

// NotificationRecord 通知记录
type NotificationRecord struct {
	// 记录ID
	ID string `json:"id"`

	// 消息ID
	MessageID string `json:"message_id"`

	// 渠道ID
	ChannelID string `json:"channel_id"`

	// 发送状态
	Status string `json:"status" validate:"oneof=pending sending sent failed"`

	// 发送时间
	SentAt *time.Time `json:"sent_at,omitempty"`

	// 重试次数
	RetryCount int `json:"retry_count"`

	// 错误信息
	Error string `json:"error,omitempty"`

	// 响应数据
	Response map[string]interface{} `json:"response,omitempty"`

	// 发送耗时 (单位: 纳秒)
	Duration time.Duration `json:"duration" swaggertype:"integer"`

	// 创建时间
	CreatedAt time.Time `json:"created_at"`

	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}
