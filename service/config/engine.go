/*
 * @module: flow-service/service/config/engine
 * @description: 工作流执行引擎配置管理，提供DAG执行、任务调度和并发控制配置
 * @architecture: 工作流引擎配置层
 * @documentReference: /docs/engine-configuration.md
 * @stateFlow: /docs/engine-state-flow.md
 * @rules:
 *   - 所有超时配置必须大于0
 *   - 并发数配置必须合理，避免资源耗尽
 *   - 重试策略必须有上限，防止无限重试
 * @dependencies:
 *   - service/config/app.go
 *   - service/config/dapr.go
 * @refs:
 *   - pkg/engine/executor.go
 *   - pkg/engine/scheduler.go
 */

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// EngineConfig 工作流引擎配置结构
type EngineConfig struct {
	// 执行器配置
	Executor ExecutorConfig `json:"executor"`

	// 调度器配置
	Scheduler SchedulerConfig `json:"scheduler"`

	// 工作池配置
	WorkerPool WorkerPoolConfig `json:"worker_pool"`

	// 任务配置
	Task TaskConfig `json:"task"`

	// 重试配置
	Retry RetryConfig `json:"retry"`

	// 监控配置
	Monitor MonitorConfig `json:"monitor"`

	// 存储配置
	Storage StorageConfig `json:"storage"`

	// 性能配置
	Performance PerformanceConfig `json:"performance"`
}

// ExecutorConfig DAG执行器配置
type ExecutorConfig struct {
	// 最大并发执行的DAG数量
	MaxConcurrentDAGs int `json:"max_concurrent_dags" validate:"min=1"`

	// 单个DAG最大并发任务数
	MaxConcurrentTasks int `json:"max_concurrent_tasks" validate:"min=1"`

	// DAG执行超时时间
	DAGTimeout time.Duration `json:"dag_timeout"`

	// 任务执行超时时间
	TaskTimeout time.Duration `json:"task_timeout"`

	// 执行模式
	ExecutionMode string `json:"execution_mode" validate:"oneof=sync async"`

	// 是否启用并行执行
	ParallelExecution bool `json:"parallel_execution"`

	// 依赖检查间隔
	DependencyCheckInterval time.Duration `json:"dependency_check_interval"`

	// 状态同步间隔
	StateSyncInterval time.Duration `json:"state_sync_interval"`
}

// SchedulerConfig 任务调度器配置
type SchedulerConfig struct {
	// 调度器类型
	Type string `json:"type" validate:"oneof=cron event manual"`

	// 调度间隔
	Interval time.Duration `json:"interval"`

	// 最大调度队列长度
	MaxQueueSize int `json:"max_queue_size" validate:"min=1"`

	// 调度器工作线程数
	WorkerThreads int `json:"worker_threads" validate:"min=1"`

	// 是否启用优先级调度
	PriorityScheduling bool `json:"priority_scheduling"`

	// 默认优先级
	DefaultPriority int `json:"default_priority" validate:"min=0,max=10"`

	// 调度延迟容忍度
	ScheduleDelayTolerance time.Duration `json:"schedule_delay_tolerance"`

	// 是否启用负载均衡
	LoadBalancing bool `json:"load_balancing"`
}

// WorkerPoolConfig 工作池配置
type WorkerPoolConfig struct {
	// 核心工作线程数
	CoreWorkers int `json:"core_workers" validate:"min=1"`

	// 最大工作线程数
	MaxWorkers int `json:"max_workers" validate:"min=1"`

	// 工作线程空闲超时
	IdleTimeout time.Duration `json:"idle_timeout"`

	// 任务队列大小
	QueueSize int `json:"queue_size" validate:"min=1"`

	// 工作线程预热
	PrewarmWorkers bool `json:"prewarm_workers"`

	// 动态扩缩容
	AutoScaling AutoScalingConfig `json:"auto_scaling"`

	// 工作线程监控
	WorkerMonitoring bool `json:"worker_monitoring"`

	// 任务分发策略
	TaskDistribution string `json:"task_distribution" validate:"oneof=round_robin least_loaded random"`
}

// AutoScalingConfig 自动扩缩容配置
type AutoScalingConfig struct {
	// 是否启用自动扩缩容
	Enabled bool `json:"enabled"`

	// 扩容阈值（队列使用率）
	ScaleUpThreshold float64 `json:"scale_up_threshold" validate:"min=0,max=1"`

	// 缩容阈值（队列使用率）
	ScaleDownThreshold float64 `json:"scale_down_threshold" validate:"min=0,max=1"`

	// 扩容步长
	ScaleUpStep int `json:"scale_up_step" validate:"min=1"`

	// 缩容步长
	ScaleDownStep int `json:"scale_down_step" validate:"min=1"`

	// 扩缩容冷却时间
	CooldownPeriod time.Duration `json:"cooldown_period"`

	// 监控窗口大小
	MetricsWindow time.Duration `json:"metrics_window"`
}

// TaskConfig 任务配置
type TaskConfig struct {
	// 默认任务超时
	DefaultTimeout time.Duration `json:"default_timeout"`

	// 最大任务超时
	MaxTimeout time.Duration `json:"max_timeout"`

	// 任务心跳间隔
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`

	// 任务状态检查间隔
	StatusCheckInterval time.Duration `json:"status_check_interval"`

	// 任务结果缓存时间
	ResultCacheTTL time.Duration `json:"result_cache_ttl"`

	// 是否启用任务预处理
	PreprocessingEnabled bool `json:"preprocessing_enabled"`

	// 是否启用任务后处理
	PostprocessingEnabled bool `json:"postprocessing_enabled"`

	// 任务资源限制
	ResourceLimits ResourceLimitsConfig `json:"resource_limits"`
}

// ResourceLimitsConfig 资源限制配置
type ResourceLimitsConfig struct {
	// CPU限制（毫核）
	CPULimit int `json:"cpu_limit" validate:"min=0"`

	// 内存限制（MB）
	MemoryLimit int `json:"memory_limit" validate:"min=0"`

	// 磁盘限制（MB）
	DiskLimit int `json:"disk_limit" validate:"min=0"`

	// 网络带宽限制（KB/s）
	NetworkLimit int `json:"network_limit" validate:"min=0"`

	// 文件句柄限制
	FileHandleLimit int `json:"file_handle_limit" validate:"min=0"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	// 默认最大重试次数
	DefaultMaxRetries int `json:"default_max_retries" validate:"min=0"`

	// 最大重试次数上限
	MaxRetriesLimit int `json:"max_retries_limit" validate:"min=0"`

	// 初始重试延迟
	InitialDelay time.Duration `json:"initial_delay"`

	// 最大重试延迟
	MaxDelay time.Duration `json:"max_delay"`

	// 退避倍数
	BackoffMultiplier float64 `json:"backoff_multiplier" validate:"min=1"`

	// 是否启用抖动
	Jitter bool `json:"jitter"`

	// 重试条件
	RetryConditions []string `json:"retry_conditions"`

	// 不重试条件
	NoRetryConditions []string `json:"no_retry_conditions"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 是否启用监控
	Enabled bool `json:"enabled"`

	// 指标收集间隔
	MetricsInterval time.Duration `json:"metrics_interval"`

	// 指标保留时间
	MetricsRetention time.Duration `json:"metrics_retention"`

	// 是否启用性能分析
	ProfilingEnabled bool `json:"profiling_enabled"`

	// 是否启用链路追踪
	TracingEnabled bool `json:"tracing_enabled"`

	// 采样率
	SamplingRate float64 `json:"sampling_rate" validate:"min=0,max=1"`

	// 告警配置
	Alerting AlertingConfig `json:"alerting"`
}

// AlertingConfig 告警配置
type AlertingConfig struct {
	// 是否启用告警
	Enabled bool `json:"enabled"`

	// 告警规则
	Rules []AlertRule `json:"rules"`

	// 告警通道
	Channels []string `json:"channels"`

	// 告警抑制时间
	SuppressionTime time.Duration `json:"suppression_time"`
}

// AlertRule 告警规则
type AlertRule struct {
	// 规则名称
	Name string `json:"name"`

	// 指标名称
	Metric string `json:"metric"`

	// 阈值
	Threshold float64 `json:"threshold"`

	// 比较操作符
	Operator string `json:"operator" validate:"oneof=gt lt eq gte lte"`

	// 持续时间
	Duration time.Duration `json:"duration"`

	// 严重级别
	Severity string `json:"severity" validate:"oneof=low medium high critical"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	// 状态存储类型
	StateStoreType string `json:"state_store_type" validate:"oneof=memory redis postgresql"`

	// 结果存储类型
	ResultStoreType string `json:"result_store_type" validate:"oneof=memory redis postgresql s3"`

	// 状态持久化间隔
	StatePersistInterval time.Duration `json:"state_persist_interval"`

	// 结果清理间隔
	ResultCleanupInterval time.Duration `json:"result_cleanup_interval"`

	// 状态压缩
	StateCompression bool `json:"state_compression"`

	// 结果压缩
	ResultCompression bool `json:"result_compression"`

	// 批量操作大小
	BatchSize int `json:"batch_size" validate:"min=1"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	// 是否启用性能优化
	OptimizationEnabled bool `json:"optimization_enabled"`

	// GC调优
	GCTuning GCTuningConfig `json:"gc_tuning"`

	// 内存池配置
	MemoryPool MemoryPoolConfig `json:"memory_pool"`

	// 连接池配置
	ConnectionPool ConnectionPoolConfig `json:"connection_pool"`

	// 缓存配置
	Cache CacheConfig `json:"cache"`
}

// GCTuningConfig GC调优配置
type GCTuningConfig struct {
	// 目标GC百分比
	TargetPercent int `json:"target_percent" validate:"min=1,max=1000"`

	// 内存限制
	MemoryLimit int64 `json:"memory_limit" validate:"min=0"`

	// 是否启用并发GC
	ConcurrentGC bool `json:"concurrent_gc"`
}

// MemoryPoolConfig 内存池配置
type MemoryPoolConfig struct {
	// 是否启用内存池
	Enabled bool `json:"enabled"`

	// 初始池大小
	InitialSize int `json:"initial_size" validate:"min=0"`

	// 最大池大小
	MaxSize int `json:"max_size" validate:"min=0"`

	// 对象生命周期
	ObjectLifetime time.Duration `json:"object_lifetime"`
}

// ConnectionPoolConfig 连接池配置
type ConnectionPoolConfig struct {
	// 最大空闲连接数
	MaxIdleConns int `json:"max_idle_conns" validate:"min=0"`

	// 最大打开连接数
	MaxOpenConns int `json:"max_open_conns" validate:"min=0"`

	// 连接最大生命周期
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`

	// 连接最大空闲时间
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// 是否启用缓存
	Enabled bool `json:"enabled"`

	// 缓存类型
	Type string `json:"type" validate:"oneof=memory redis"`

	// 缓存大小（条目数）
	Size int `json:"size" validate:"min=0"`

	// 缓存TTL
	TTL time.Duration `json:"ttl"`

	// 缓存清理间隔
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultEngineConfig 默认引擎配置
var DefaultEngineConfig = &EngineConfig{
	Executor: ExecutorConfig{
		MaxConcurrentDAGs:       10,
		MaxConcurrentTasks:      50,
		DAGTimeout:              30 * time.Minute,
		TaskTimeout:             5 * time.Minute,
		ExecutionMode:           "async",
		ParallelExecution:       true,
		DependencyCheckInterval: 1 * time.Second,
		StateSyncInterval:       5 * time.Second,
	},

	Scheduler: SchedulerConfig{
		Type:                   "cron",
		Interval:               1 * time.Minute,
		MaxQueueSize:           1000,
		WorkerThreads:          5,
		PriorityScheduling:     true,
		DefaultPriority:        5,
		ScheduleDelayTolerance: 30 * time.Second,
		LoadBalancing:          true,
	},

	WorkerPool: WorkerPoolConfig{
		CoreWorkers:      10,
		MaxWorkers:       100,
		IdleTimeout:      5 * time.Minute,
		QueueSize:        1000,
		PrewarmWorkers:   true,
		WorkerMonitoring: true,
		TaskDistribution: "least_loaded",
		AutoScaling: AutoScalingConfig{
			Enabled:            true,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.2,
			ScaleUpStep:        5,
			ScaleDownStep:      2,
			CooldownPeriod:     2 * time.Minute,
			MetricsWindow:      5 * time.Minute,
		},
	},

	Task: TaskConfig{
		DefaultTimeout:        5 * time.Minute,
		MaxTimeout:            30 * time.Minute,
		HeartbeatInterval:     30 * time.Second,
		StatusCheckInterval:   10 * time.Second,
		ResultCacheTTL:        1 * time.Hour,
		PreprocessingEnabled:  true,
		PostprocessingEnabled: true,
		ResourceLimits: ResourceLimitsConfig{
			CPULimit:        1000, // 1 CPU core
			MemoryLimit:     512,  // 512MB
			DiskLimit:       1024, // 1GB
			NetworkLimit:    1024, // 1MB/s
			FileHandleLimit: 100,
		},
	},

	Retry: RetryConfig{
		DefaultMaxRetries: 3,
		MaxRetriesLimit:   10,
		InitialDelay:      1 * time.Second,
		MaxDelay:          5 * time.Minute,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		RetryConditions:   []string{"timeout", "network_error", "temporary_failure"},
		NoRetryConditions: []string{"validation_error", "permission_denied", "not_found"},
	},

	Monitor: MonitorConfig{
		Enabled:          true,
		MetricsInterval:  30 * time.Second,
		MetricsRetention: 24 * time.Hour,
		ProfilingEnabled: false,
		TracingEnabled:   true,
		SamplingRate:     0.1,
		Alerting: AlertingConfig{
			Enabled:         true,
			Rules:           []AlertRule{},
			Channels:        []string{"log", "webhook"},
			SuppressionTime: 5 * time.Minute,
		},
	},

	Storage: StorageConfig{
		StateStoreType:        "redis",
		ResultStoreType:       "redis",
		StatePersistInterval:  10 * time.Second,
		ResultCleanupInterval: 1 * time.Hour,
		StateCompression:      true,
		ResultCompression:     true,
		BatchSize:             100,
	},

	Performance: PerformanceConfig{
		OptimizationEnabled: true,
		GCTuning: GCTuningConfig{
			TargetPercent: 100,
			MemoryLimit:   0,
			ConcurrentGC:  true,
		},
		MemoryPool: MemoryPoolConfig{
			Enabled:        true,
			InitialSize:    100,
			MaxSize:        1000,
			ObjectLifetime: 10 * time.Minute,
		},
		ConnectionPool: ConnectionPoolConfig{
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: 1 * time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
		Cache: CacheConfig{
			Enabled:         true,
			Type:            "memory",
			Size:            10000,
			TTL:             1 * time.Hour,
			CleanupInterval: 10 * time.Minute,
		},
	},
}

// LoadEngineConfig 加载引擎配置
func LoadEngineConfig() *EngineConfig {
	config := *DefaultEngineConfig // 复制默认配置

	// 执行器配置
	if maxDAGs := os.Getenv("ENGINE_MAX_CONCURRENT_DAGS"); maxDAGs != "" {
		if val, err := strconv.Atoi(maxDAGs); err == nil && val > 0 {
			config.Executor.MaxConcurrentDAGs = val
		}
	}

	if maxTasks := os.Getenv("ENGINE_MAX_CONCURRENT_TASKS"); maxTasks != "" {
		if val, err := strconv.Atoi(maxTasks); err == nil && val > 0 {
			config.Executor.MaxConcurrentTasks = val
		}
	}

	if dagTimeout := os.Getenv("ENGINE_DAG_TIMEOUT"); dagTimeout != "" {
		if val, err := time.ParseDuration(dagTimeout); err == nil {
			config.Executor.DAGTimeout = val
		}
	}

	if taskTimeout := os.Getenv("ENGINE_TASK_TIMEOUT"); taskTimeout != "" {
		if val, err := time.ParseDuration(taskTimeout); err == nil {
			config.Executor.TaskTimeout = val
		}
	}

	// 工作池配置
	if coreWorkers := os.Getenv("ENGINE_CORE_WORKERS"); coreWorkers != "" {
		if val, err := strconv.Atoi(coreWorkers); err == nil && val > 0 {
			config.WorkerPool.CoreWorkers = val
		}
	}

	if maxWorkers := os.Getenv("ENGINE_MAX_WORKERS"); maxWorkers != "" {
		if val, err := strconv.Atoi(maxWorkers); err == nil && val > 0 {
			config.WorkerPool.MaxWorkers = val
		}
	}

	// 重试配置
	if maxRetries := os.Getenv("ENGINE_MAX_RETRIES"); maxRetries != "" {
		if val, err := strconv.Atoi(maxRetries); err == nil && val >= 0 {
			config.Retry.DefaultMaxRetries = val
		}
	}

	// 监控配置
	if monitorEnabled := os.Getenv("ENGINE_MONITOR_ENABLED"); monitorEnabled != "" {
		if val, err := strconv.ParseBool(monitorEnabled); err == nil {
			config.Monitor.Enabled = val
		}
	}

	return &config
}

// Validate 验证引擎配置
func (c *EngineConfig) Validate() error {
	// 验证执行器配置
	if err := c.Executor.Validate(); err != nil {
		return fmt.Errorf("executor config error: %w", err)
	}

	// 验证调度器配置
	if err := c.Scheduler.Validate(); err != nil {
		return fmt.Errorf("scheduler config error: %w", err)
	}

	// 验证工作池配置
	if err := c.WorkerPool.Validate(); err != nil {
		return fmt.Errorf("worker pool config error: %w", err)
	}

	// 验证任务配置
	if err := c.Task.Validate(); err != nil {
		return fmt.Errorf("task config error: %w", err)
	}

	return nil
}

// Validate 验证执行器配置
func (c *ExecutorConfig) Validate() error {
	if c.MaxConcurrentDAGs < 1 {
		return fmt.Errorf("max concurrent DAGs must be positive")
	}

	if c.MaxConcurrentTasks < 1 {
		return fmt.Errorf("max concurrent tasks must be positive")
	}

	if c.DAGTimeout <= 0 {
		return fmt.Errorf("DAG timeout must be positive")
	}

	if c.TaskTimeout <= 0 {
		return fmt.Errorf("task timeout must be positive")
	}

	validModes := map[string]bool{"sync": true, "async": true}
	if !validModes[c.ExecutionMode] {
		return fmt.Errorf("invalid execution mode: %s", c.ExecutionMode)
	}

	return nil
}

// Validate 验证调度器配置
func (c *SchedulerConfig) Validate() error {
	validTypes := map[string]bool{"cron": true, "event": true, "manual": true}
	if !validTypes[c.Type] {
		return fmt.Errorf("invalid scheduler type: %s", c.Type)
	}

	if c.MaxQueueSize < 1 {
		return fmt.Errorf("max queue size must be positive")
	}

	if c.WorkerThreads < 1 {
		return fmt.Errorf("worker threads must be positive")
	}

	if c.DefaultPriority < 0 || c.DefaultPriority > 10 {
		return fmt.Errorf("default priority must be between 0 and 10")
	}

	return nil
}

// Validate 验证工作池配置
func (c *WorkerPoolConfig) Validate() error {
	if c.CoreWorkers < 1 {
		return fmt.Errorf("core workers must be positive")
	}

	if c.MaxWorkers < c.CoreWorkers {
		return fmt.Errorf("max workers must be >= core workers")
	}

	if c.QueueSize < 1 {
		return fmt.Errorf("queue size must be positive")
	}

	validDistributions := map[string]bool{"round_robin": true, "least_loaded": true, "random": true}
	if !validDistributions[c.TaskDistribution] {
		return fmt.Errorf("invalid task distribution: %s", c.TaskDistribution)
	}

	return nil
}

// Validate 验证任务配置
func (c *TaskConfig) Validate() error {
	if c.DefaultTimeout <= 0 {
		return fmt.Errorf("default timeout must be positive")
	}

	if c.MaxTimeout < c.DefaultTimeout {
		return fmt.Errorf("max timeout must be >= default timeout")
	}

	return nil
}

// IsAsyncMode 判断是否为异步执行模式
func (c *EngineConfig) IsAsyncMode() bool {
	return c.Executor.ExecutionMode == "async"
}

// IsAutoScalingEnabled 判断是否启用自动扩缩容
func (c *EngineConfig) IsAutoScalingEnabled() bool {
	return c.WorkerPool.AutoScaling.Enabled
}

// IsMonitoringEnabled 判断是否启用监控
func (c *EngineConfig) IsMonitoringEnabled() bool {
	return c.Monitor.Enabled
}

// GetEffectiveWorkerCount 获取有效工作线程数
func (c *EngineConfig) GetEffectiveWorkerCount() int {
	if c.WorkerPool.AutoScaling.Enabled {
		return c.WorkerPool.MaxWorkers
	}
	return c.WorkerPool.CoreWorkers
}
