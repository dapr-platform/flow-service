/*
 * @module: flow-service/service/config/dapr
 * @description: Dapr 组件配置管理，提供微服务通信和状态存储配置
 * @architecture: Dapr 微服务架构配置层
 * @documentReference: /docs/dapr-configuration.md
 * @stateFlow: 无
 * @rules:
 *   - 所有 Dapr 组件必须配置健康检查
 *   - 支持多环境配置切换
 *   - 组件配置必须验证有效性
 * @dependencies:
 *   - Dapr Runtime
 *   - Redis/PostgreSQL (State Store)
 *   - Redis Streams (Pub/Sub)
 * @refs:
 *   - service/config/app.go
 *   - service/config/engine.go
 */

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DaprConfig Dapr 组件配置结构
type DaprConfig struct {
	// 基础配置
	AppID       string `json:"app_id"`
	AppPort     int    `json:"app_port"`
	AppProtocol string `json:"app_protocol" validate:"oneof=http grpc"`

	// State Store 配置
	StateStore StateStoreConfig `json:"state_store"`

	// Pub/Sub 配置
	PubSub PubSubConfig `json:"pub_sub"`

	// Service Invocation 配置
	ServiceInvocation ServiceInvocationConfig `json:"service_invocation"`

	// 组件健康检查配置
	HealthCheck DaprHealthCheckConfig `json:"health_check"`

	// 重试配置
	Retry DaprRetryConfig `json:"retry"`

	// 超时配置
	Timeout DaprTimeoutConfig `json:"timeout"`
}

// StateStoreConfig State Store 配置
type StateStoreConfig struct {
	Name             string        `json:"name"`
	Type             string        `json:"type" validate:"oneof=redis postgresql mongodb"`
	ConnectionString string        `json:"connection_string"`
	KeyPrefix        string        `json:"key_prefix"`
	MaxRetries       int           `json:"max_retries" validate:"min=0"`
	RetryDelay       time.Duration `json:"retry_delay"`
	TTL              time.Duration `json:"ttl"`
	ConsistencyLevel string        `json:"consistency_level" validate:"oneof=eventual strong"`
	OperationTimeout time.Duration `json:"operation_timeout"`
}

// PubSubConfig Pub/Sub 配置
type PubSubConfig struct {
	Name             string                 `json:"name"`
	Type             string                 `json:"type" validate:"oneof=redis-streams kafka pulsar"`
	ConnectionString string                 `json:"connection_string"`
	Topics           map[string]TopicConfig `json:"topics"`
	MaxRetries       int                    `json:"max_retries" validate:"min=0"`
	RetryDelay       time.Duration          `json:"retry_delay"`
	DeadLetterTopic  string                 `json:"dead_letter_topic"`
}

// TopicConfig 主题配置
type TopicConfig struct {
	Name           string        `json:"name"`
	ConsumerGroup  string        `json:"consumer_group"`
	MaxConcurrency int           `json:"max_concurrency" validate:"min=1"`
	AckTimeout     time.Duration `json:"ack_timeout"`
	RetryAttempts  int           `json:"retry_attempts" validate:"min=0"`
}

// ServiceInvocationConfig 服务调用配置
type ServiceInvocationConfig struct {
	Timeout        time.Duration            `json:"timeout"`
	MaxRetries     int                      `json:"max_retries" validate:"min=0"`
	RetryDelay     time.Duration            `json:"retry_delay"`
	Services       map[string]ServiceConfig `json:"services"`
	LoadBalancing  string                   `json:"load_balancing" validate:"oneof=round_robin random"`
	CircuitBreaker CircuitBreakerConfig     `json:"circuit_breaker"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	AppID      string        `json:"app_id"`
	Method     string        `json:"method" validate:"oneof=http grpc"`
	Namespace  string        `json:"namespace"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries" validate:"min=0"`
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	Enabled          bool          `json:"enabled"`
	FailureThreshold int           `json:"failure_threshold" validate:"min=1"`
	RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	HalfOpenRequests int           `json:"half_open_requests" validate:"min=1"`
}

// DaprHealthCheckConfig Dapr 健康检查配置
type DaprHealthCheckConfig struct {
	Enabled          bool          `json:"enabled"`
	Interval         time.Duration `json:"interval"`
	Timeout          time.Duration `json:"timeout"`
	FailureThreshold int           `json:"failure_threshold" validate:"min=1"`
	Components       []string      `json:"components"`
}

// DaprRetryConfig Dapr 重试配置
type DaprRetryConfig struct {
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier" validate:"min=1"`
	MaxRetries   int           `json:"max_retries" validate:"min=0"`
	Jitter       bool          `json:"jitter"`
}

// DaprTimeoutConfig Dapr 超时配置
type DaprTimeoutConfig struct {
	StateStore        time.Duration `json:"state_store"`
	PubSub            time.Duration `json:"pub_sub"`
	ServiceInvocation time.Duration `json:"service_invocation"`
	Binding           time.Duration `json:"binding"`
}

// DefaultDaprConfig 默认 Dapr 配置
var DefaultDaprConfig = &DaprConfig{
	AppID:       "flow-service",
	AppPort:     80,
	AppProtocol: "http",

	StateStore: StateStoreConfig{
		Name:             "statestore",
		Type:             "redis",
		ConnectionString: "localhost:6379",
		KeyPrefix:        "flow:",
		MaxRetries:       3,
		RetryDelay:       1 * time.Second,
		TTL:              24 * time.Hour,
		ConsistencyLevel: "eventual",
		OperationTimeout: 5 * time.Second,
	},

	PubSub: PubSubConfig{
		Name:             "pubsub",
		Type:             "redis-streams",
		ConnectionString: "localhost:6379",
		Topics: map[string]TopicConfig{
			"flow-events": {
				Name:           "flow-events",
				ConsumerGroup:  "flow-service",
				MaxConcurrency: 10,
				AckTimeout:     30 * time.Second,
				RetryAttempts:  3,
			},
			"task-events": {
				Name:           "task-events",
				ConsumerGroup:  "flow-service",
				MaxConcurrency: 5,
				AckTimeout:     30 * time.Second,
				RetryAttempts:  3,
			},
		},
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		DeadLetterTopic: "dead-letter",
	},

	ServiceInvocation: ServiceInvocationConfig{
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		RetryDelay:    1 * time.Second,
		Services:      make(map[string]ServiceConfig),
		LoadBalancing: "round_robin",
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			RecoveryTimeout:  30 * time.Second,
			HalfOpenRequests: 3,
		},
	},

	HealthCheck: DaprHealthCheckConfig{
		Enabled:          true,
		Interval:         30 * time.Second,
		Timeout:          5 * time.Second,
		FailureThreshold: 3,
		Components:       []string{"statestore", "pubsub"},
	},

	Retry: DaprRetryConfig{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		MaxRetries:   5,
		Jitter:       true,
	},

	Timeout: DaprTimeoutConfig{
		StateStore:        5 * time.Second,
		PubSub:            10 * time.Second,
		ServiceInvocation: 30 * time.Second,
		Binding:           15 * time.Second,
	},
}

// LoadDaprConfig 加载 Dapr 配置
func LoadDaprConfig() *DaprConfig {
	config := *DefaultDaprConfig // 复制默认配置

	// 从环境变量加载基础配置
	if appID := os.Getenv("DAPR_APP_ID"); appID != "" {
		config.AppID = appID
	}

	if appPort := os.Getenv("DAPR_APP_PORT"); appPort != "" {
		if port, err := strconv.Atoi(appPort); err == nil {
			config.AppPort = port
		}
	}

	if protocol := os.Getenv("DAPR_APP_PROTOCOL"); protocol != "" {
		config.AppProtocol = protocol
	}

	// State Store 配置
	if stateStoreType := os.Getenv("DAPR_STATE_STORE_TYPE"); stateStoreType != "" {
		config.StateStore.Type = stateStoreType
	}

	if stateStoreConn := os.Getenv("DAPR_STATE_STORE_CONNECTION"); stateStoreConn != "" {
		config.StateStore.ConnectionString = stateStoreConn
	}

	if keyPrefix := os.Getenv("DAPR_STATE_STORE_KEY_PREFIX"); keyPrefix != "" {
		config.StateStore.KeyPrefix = keyPrefix
	}

	// Pub/Sub 配置
	if pubsubType := os.Getenv("DAPR_PUBSUB_TYPE"); pubsubType != "" {
		config.PubSub.Type = pubsubType
	}

	if pubsubConn := os.Getenv("DAPR_PUBSUB_CONNECTION"); pubsubConn != "" {
		config.PubSub.ConnectionString = pubsubConn
	}

	// 健康检查配置
	if healthEnabled := os.Getenv("DAPR_HEALTH_CHECK_ENABLED"); healthEnabled != "" {
		if enabled, err := strconv.ParseBool(healthEnabled); err == nil {
			config.HealthCheck.Enabled = enabled
		}
	}

	if healthInterval := os.Getenv("DAPR_HEALTH_CHECK_INTERVAL"); healthInterval != "" {
		if interval, err := time.ParseDuration(healthInterval); err == nil {
			config.HealthCheck.Interval = interval
		}
	}

	return &config
}

// Validate 验证 Dapr 配置
func (c *DaprConfig) Validate() error {
	if c.AppID == "" {
		return fmt.Errorf("app_id cannot be empty")
	}

	if c.AppPort < 1 || c.AppPort > 65535 {
		return fmt.Errorf("invalid app_port: %d", c.AppPort)
	}

	if c.AppProtocol != "http" && c.AppProtocol != "grpc" {
		return fmt.Errorf("invalid app_protocol: %s", c.AppProtocol)
	}

	// 验证 State Store 配置
	if err := c.StateStore.Validate(); err != nil {
		return fmt.Errorf("state store config error: %w", err)
	}

	// 验证 Pub/Sub 配置
	if err := c.PubSub.Validate(); err != nil {
		return fmt.Errorf("pubsub config error: %w", err)
	}

	return nil
}

// Validate 验证 State Store 配置
func (c *StateStoreConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("state store name cannot be empty")
	}

	validTypes := map[string]bool{"redis": true, "postgresql": true, "mongodb": true}
	if !validTypes[c.Type] {
		return fmt.Errorf("invalid state store type: %s", c.Type)
	}

	if c.ConnectionString == "" {
		return fmt.Errorf("state store connection string cannot be empty")
	}

	validConsistency := map[string]bool{"eventual": true, "strong": true}
	if !validConsistency[c.ConsistencyLevel] {
		return fmt.Errorf("invalid consistency level: %s", c.ConsistencyLevel)
	}

	return nil
}

// Validate 验证 Pub/Sub 配置
func (c *PubSubConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("pubsub name cannot be empty")
	}

	validTypes := map[string]bool{"redis-streams": true, "kafka": true, "pulsar": true}
	if !validTypes[c.Type] {
		return fmt.Errorf("invalid pubsub type: %s", c.Type)
	}

	if c.ConnectionString == "" {
		return fmt.Errorf("pubsub connection string cannot be empty")
	}

	// 验证主题配置
	for topicName, topicConfig := range c.Topics {
		if err := topicConfig.Validate(); err != nil {
			return fmt.Errorf("topic %s config error: %w", topicName, err)
		}
	}

	return nil
}

// Validate 验证主题配置
func (c *TopicConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("topic name cannot be empty")
	}

	if c.ConsumerGroup == "" {
		return fmt.Errorf("consumer group cannot be empty")
	}

	if c.MaxConcurrency < 1 {
		return fmt.Errorf("max concurrency must be positive")
	}

	return nil
}

// GetStateStoreComponentName 获取 State Store 组件名称
func (c *DaprConfig) GetStateStoreComponentName() string {
	return c.StateStore.Name
}

// GetPubSubComponentName 获取 Pub/Sub 组件名称
func (c *DaprConfig) GetPubSubComponentName() string {
	return c.PubSub.Name
}

// IsHealthCheckEnabled 判断是否启用健康检查
func (c *DaprConfig) IsHealthCheckEnabled() bool {
	return c.HealthCheck.Enabled
}

// GetTopicConfig 获取指定主题配置
func (c *DaprConfig) GetTopicConfig(topicName string) (TopicConfig, bool) {
	config, exists := c.PubSub.Topics[topicName]
	return config, exists
}

// AddServiceConfig 添加服务配置
func (c *DaprConfig) AddServiceConfig(serviceName string, config ServiceConfig) {
	if c.ServiceInvocation.Services == nil {
		c.ServiceInvocation.Services = make(map[string]ServiceConfig)
	}
	c.ServiceInvocation.Services[serviceName] = config
}
