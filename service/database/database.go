/*
 * @module service/database/database
 * @description 数据库初始化和连接管理模块，提供数据库连接、迁移和基础数据初始化功能
 * @architecture 数据访问层 - 数据库连接管理
 * @documentReference docs/database-design.md
 * @stateFlow 应用启动时执行数据库初始化
 * @rules 确保数据库连接稳定性和迁移安全性
 * @dependencies gorm.io/gorm, gorm.io/driver/postgres, flow-service/service/models
 * @refs service/models/, service/config/
 */

package database

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DB 全局数据库连接实例
	DB *gorm.DB
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `json:"host" yaml:"host"`
	Port         int    `json:"port" yaml:"port"`
	Database     string `json:"database" yaml:"database"`
	Username     string `json:"username" yaml:"username"`
	Password     string `json:"password" yaml:"password"`
	MaxOpenConns int    `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns" yaml:"max_idle_conns"`
	SSLMode      string `json:"ssl_mode" yaml:"ssl_mode"`
	Schema       string `json:"schema" yaml:"schema"`
	LogLevel     string `json:"log_level" yaml:"log_level"`
}

// GetDefaultConfig 获取默认数据库配置
func GetDefaultConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:         "localhost",
		Port:         5432,
		Database:     "flow_service",
		Username:     "postgres",
		Password:     "password",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
		SSLMode:      "disable",
		Schema:       "public",
		LogLevel:     "info",
	}
}

// LoadConfigFromEnv 从环境变量加载数据库配置
func LoadConfigFromEnv() *DatabaseConfig {
	config := GetDefaultConfig()

	if host := os.Getenv("DB_HOST"); host != "" {
		config.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		if portInt := parseInt(port, 5432); portInt > 0 {
			config.Port = portInt
		}
	}
	if database := os.Getenv("DB_NAME"); database != "" {
		config.Database = database
	}
	if username := os.Getenv("DB_USER"); username != "" {
		config.Username = username
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		config.Password = password
	}
	if sslMode := os.Getenv("DB_SSLMODE"); sslMode != "" {
		config.SSLMode = sslMode
	}
	if schema := os.Getenv("DB_SCHEMA"); schema != "" {
		config.Schema = schema
	}
	if logLevel := os.Getenv("DB_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	return config
}

// InitDatabase 初始化数据库连接
func InitDatabase() error {
	config := LoadConfigFromEnv()
	return InitDatabaseWithConfig(config)
}

// InitDatabaseWithConfig 使用指定配置初始化数据库连接
func InitDatabaseWithConfig(config *DatabaseConfig) error {
	// 构建DSN连接字符串
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s TimeZone=Asia/Shanghai",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode, config.Schema)

	// 配置GORM日志级别
	logLevel := logger.Info
	switch config.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	}

	// 创建GORM配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	// 打开数据库连接
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 获取底层sql.DB对象并配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接池失败: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Printf("数据库连接成功: %s:%d/%s", config.Host, config.Port, config.Database)
	return nil
}

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// 辅助函数
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	// 简单的字符串转整数实现
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return defaultValue
		}
	}
	return result
}
