/*
 * @module: flow-service/service/config
 * @description: 应用配置管理，提供服务运行所需的各种配置参数
 * @architecture: 配置层，支持环境变量和默认值
 * @documentReference: /docs/config-management.md
 * @stateFlow: 无
 * @rules:
 *   - 所有配置必须有默认值
 *   - 支持环境变量覆盖
 *   - 配置验证确保合法性
 * @dependencies:
 *   - 环境变量系统
 *   - 验证框架
 * @refs:
 *   - service/config/dapr.go
 *   - service/config/engine.go
 */

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// AppConfig 应用配置结构
type AppConfig struct {
	// 服务配置
	ListenPort  int    `json:"listen_port" validate:"min=1,max=65535"`
	BaseContext string `json:"base_context"`
	ServiceName string `json:"service_name"`
	Environment string `json:"environment" validate:"oneof=dev test prod"`

	// 超时配置
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`

	// 并发控制配置
	MaxConcurrentRequests int `json:"max_concurrent_requests" validate:"min=1"`
	MaxWorkerGoRoutines   int `json:"max_worker_goroutines" validate:"min=1"`
	RequestQueueSize      int `json:"request_queue_size" validate:"min=1"`

	// 健康检查配置
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`

	// 优雅关闭配置
	GracefulShutdownTimeout time.Duration `json:"graceful_shutdown_timeout"`
}

// DefaultAppConfig 默认应用配置
var DefaultAppConfig = &AppConfig{
	ListenPort:              80,
	BaseContext:             "",
	ServiceName:             "flow-service",
	Environment:             "dev",
	ReadTimeout:             30 * time.Second,
	WriteTimeout:            30 * time.Second,
	IdleTimeout:             60 * time.Second,
	MaxConcurrentRequests:   1000,
	MaxWorkerGoRoutines:     100,
	RequestQueueSize:        10000,
	HealthCheckInterval:     30 * time.Second,
	HealthCheckTimeout:      5 * time.Second,
	GracefulShutdownTimeout: 30 * time.Second,
}

// LoadAppConfig 加载应用配置
func LoadAppConfig() *AppConfig {
	config := *DefaultAppConfig // 复制默认配置

	// 从环境变量加载配置
	if port := os.Getenv("LISTEN_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.ListenPort = p
		}
	}

	if baseContext := os.Getenv("BASE_CONTEXT"); baseContext != "" {
		config.BaseContext = baseContext
	}

	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		config.ServiceName = serviceName
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Environment = env
	}

	// 超时配置
	if timeout := os.Getenv("READ_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			config.ReadTimeout = t
		}
	}

	if timeout := os.Getenv("WRITE_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			config.WriteTimeout = t
		}
	}

	if timeout := os.Getenv("IDLE_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			config.IdleTimeout = t
		}
	}

	// 并发控制配置
	if maxReq := os.Getenv("MAX_CONCURRENT_REQUESTS"); maxReq != "" {
		if m, err := strconv.Atoi(maxReq); err == nil && m > 0 {
			config.MaxConcurrentRequests = m
		}
	}

	if maxWorkers := os.Getenv("MAX_WORKER_GOROUTINES"); maxWorkers != "" {
		if m, err := strconv.Atoi(maxWorkers); err == nil && m > 0 {
			config.MaxWorkerGoRoutines = m
		}
	}

	if queueSize := os.Getenv("REQUEST_QUEUE_SIZE"); queueSize != "" {
		if q, err := strconv.Atoi(queueSize); err == nil && q > 0 {
			config.RequestQueueSize = q
		}
	}

	// 健康检查配置
	if interval := os.Getenv("HEALTH_CHECK_INTERVAL"); interval != "" {
		if i, err := time.ParseDuration(interval); err == nil {
			config.HealthCheckInterval = i
		}
	}

	if timeout := os.Getenv("HEALTH_CHECK_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			config.HealthCheckTimeout = t
		}
	}

	// 优雅关闭配置
	if timeout := os.Getenv("GRACEFUL_SHUTDOWN_TIMEOUT"); timeout != "" {
		if t, err := time.ParseDuration(timeout); err == nil {
			config.GracefulShutdownTimeout = t
		}
	}

	return &config
}

// Validate 验证配置的合法性
func (c *AppConfig) Validate() error {
	if c.ListenPort < 1 || c.ListenPort > 65535 {
		return fmt.Errorf("invalid listen port: %d", c.ListenPort)
	}

	if c.MaxConcurrentRequests < 1 {
		return fmt.Errorf("max concurrent requests must be positive")
	}

	if c.MaxWorkerGoRoutines < 1 {
		return fmt.Errorf("max worker goroutines must be positive")
	}

	if c.RequestQueueSize < 1 {
		return fmt.Errorf("request queue size must be positive")
	}

	validEnvs := map[string]bool{"dev": true, "test": true, "prod": true}
	if !validEnvs[c.Environment] {
		return fmt.Errorf("invalid environment: %s", c.Environment)
	}

	return nil
}

// IsDevelopment 判断是否为开发环境
func (c *AppConfig) IsDevelopment() bool {
	return c.Environment == "dev"
}

// IsProduction 判断是否为生产环境
func (c *AppConfig) IsProduction() bool {
	return c.Environment == "prod"
}

// GetServerAddress 获取服务器监听地址
func (c *AppConfig) GetServerAddress() string {
	return fmt.Sprintf(":%d", c.ListenPort)
}
