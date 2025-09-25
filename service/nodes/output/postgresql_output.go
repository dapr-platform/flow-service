/**
 * @module postgresql_output
 * @description PostgreSQL输出节点，将数据写入PostgreSQL数据库
 * @architecture 数据输出插件实现，支持批量插入、更新、删除操作
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow output_states: configured -> connecting -> writing -> completed
 * @rules 支持事务处理、批量操作、连接池管理
 * @dependencies context, database/sql, time
 * @refs service/nodes/interface.go
 */

package output

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"flow-service/service/nodes"

	_ "github.com/lib/pq"
)

// init 自动注册PostgreSQL输出节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewPostgreSQLOutputNode()); err != nil {
		log.Printf("注册PostgreSQL输出节点失败: %v", err)
	} else {
		log.Println("PostgreSQL输出节点注册成功")
	}
}

// PostgreSQLOutputNode PostgreSQL输出节点
type PostgreSQLOutputNode struct{}

// NewPostgreSQLOutputNode 创建PostgreSQL输出节点
func NewPostgreSQLOutputNode() *PostgreSQLOutputNode {
	return &PostgreSQLOutputNode{}
}

// GetMetadata 获取节点元数据
func (p *PostgreSQLOutputNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "postgresql_output",
		Name:        "PostgreSQL输出器",
		Description: "将数据写入PostgreSQL数据库",
		Version:     "1.0.0",
		Category:    nodes.CategoryOutput,
		Type:        nodes.TypePostgreSQL,
		Icon:        "database",
		Tags:        []string{"数据库", "PostgreSQL", "输出"},

		InputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "输入数据",
				Description: "需要写入数据库的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
		},

		OutputPorts: []nodes.PortDefinition{},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "host",
					Type:        "string",
					Title:       "数据库主机",
					Description: "PostgreSQL数据库主机地址",
					Widget:      nodes.WidgetText,
					Placeholder: "localhost",
				},
				{
					Name:        "port",
					Type:        "number",
					Title:       "端口",
					Description: "数据库端口号",
					Default:     5432,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "database",
					Type:        "string",
					Title:       "数据库名",
					Description: "目标数据库名称",
					Widget:      nodes.WidgetText,
				},
				{
					Name:        "username",
					Type:        "string",
					Title:       "用户名",
					Description: "数据库用户名",
					Widget:      nodes.WidgetText,
				},
				{
					Name:        "password",
					Type:        "string",
					Title:       "密码",
					Description: "数据库密码",
					Widget:      nodes.WidgetPassword,
				},
				{
					Name:        "sslmode",
					Type:        "string",
					Title:       "SSL模式",
					Description: "SSL连接模式",
					Default:     "disable",
					Enum:        []interface{}{"disable", "require", "verify-ca", "verify-full"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "table_name",
					Type:        "string",
					Title:       "表名",
					Description: "目标数据表名称",
					Widget:      nodes.WidgetText,
				},
				{
					Name:        "operation",
					Type:        "string",
					Title:       "操作类型",
					Description: "数据库操作类型",
					Default:     "insert",
					Enum:        []interface{}{"insert", "upsert", "update", "delete"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "batch_size",
					Type:        "number",
					Title:       "批处理大小",
					Description: "每批处理的记录数量",
					Default:     100,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "conflict_resolution",
					Type:        "string",
					Title:       "冲突处理",
					Description: "主键冲突时的处理方式",
					Default:     "ignore",
					Enum:        []interface{}{"ignore", "update", "error"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "primary_key",
					Type:        "array",
					Title:       "主键字段",
					Description: "用于冲突检测的主键字段列表",
					Items: &nodes.ConfigField{
						Type: "string",
					},
					Widget: nodes.WidgetJSON,
				},
			},
			Required: []string{"host", "database", "username", "password", "table_name"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (p *PostgreSQLOutputNode) Validate(config map[string]interface{}) error {
	// 验证必需的连接字段
	if _, ok := config["host"].(string); !ok {
		return fmt.Errorf("missing or invalid host")
	}

	if _, ok := config["database"].(string); !ok {
		return fmt.Errorf("missing or invalid database")
	}

	if _, ok := config["username"].(string); !ok {
		return fmt.Errorf("missing or invalid username")
	}

	if _, ok := config["password"].(string); !ok {
		return fmt.Errorf("missing or invalid password")
	}

	if _, ok := config["table_name"].(string); !ok {
		return fmt.Errorf("missing or invalid table_name")
	}

	// 验证操作类型
	if operation, ok := config["operation"].(string); ok {
		validOps := []string{"insert", "upsert", "update", "delete"}
		valid := false
		for _, validOp := range validOps {
			if operation == validOp {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid operation: %s", operation)
		}
	}

	return nil
}

// Execute 执行输出操作
func (p *PostgreSQLOutputNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
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

	// 解析配置
	config := p.parseConfig(input.Config)

	// 建立数据库连接
	db, err := p.connect(config.Connection)
	if err != nil {
		return &nodes.NodeOutput{
			Success: false,
			Error:   fmt.Sprintf("数据库连接失败: %v", err),
		}, nil
	}
	defer db.Close()

	// 执行操作
	result, err := p.executeOperation(ctx, db, data, config)
	if err != nil {
		return &nodes.NodeOutput{
			Success: false,
			Error:   fmt.Sprintf("操作执行失败: %v", err),
		}, nil
	}

	return &nodes.NodeOutput{
		Data: map[string]interface{}{
			"result": result,
		},
		Success:  true,
		Duration: time.Since(startTime),
	}, nil
}

// connect 建立数据库连接
func (p *PostgreSQLOutputNode) connect(config ConnectionConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		config.Host, config.Port, config.Database, config.Username, config.Password, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// executeOperation 执行数据库操作
func (p *PostgreSQLOutputNode) executeOperation(ctx context.Context, db *sql.DB, data []interface{}, config *OutputConfig) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"total_records":  len(data),
		"success_count":  0,
		"error_count":    0,
		"operation_type": config.Operation.Type,
		"table":          config.Operation.Table,
	}

	// 使用事务
	var tx *sql.Tx
	var err error
	if config.Options.UseTransaction {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return result, fmt.Errorf("开始事务失败: %v", err)
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			} else {
				tx.Commit()
			}
		}()
	}

	// 批量处理数据
	batchSize := config.Operation.BatchSize
	for i := 0; i < len(data); i += batchSize {
		end := i + batchSize
		if end > len(data) {
			end = len(data)
		}

		batch := data[i:end]
		successCount, errorCount, err := p.processBatch(ctx, db, tx, batch, config)
		if err != nil && !config.Options.IgnoreErrors {
			return result, err
		}

		result["success_count"] = result["success_count"].(int) + successCount
		result["error_count"] = result["error_count"].(int) + errorCount
	}

	return result, nil
}

// processBatch 处理数据批次
func (p *PostgreSQLOutputNode) processBatch(ctx context.Context, db *sql.DB, tx *sql.Tx, batch []interface{}, config *OutputConfig) (int, int, error) {
	switch config.Operation.Type {
	case "insert":
		return p.insertBatch(ctx, db, tx, batch, config)
	case "update":
		return p.updateBatch(ctx, db, tx, batch, config)
	case "delete":
		return p.deleteBatch(ctx, db, tx, batch, config)
	case "upsert":
		return p.upsertBatch(ctx, db, tx, batch, config)
	default:
		return 0, len(batch), fmt.Errorf("不支持的操作类型: %s", config.Operation.Type)
	}
}

// insertBatch 批量插入
func (p *PostgreSQLOutputNode) insertBatch(ctx context.Context, db *sql.DB, tx *sql.Tx, batch []interface{}, config *OutputConfig) (int, int, error) {
	if len(batch) == 0 {
		return 0, 0, nil
	}

	// 构建插入SQL
	tableName := fmt.Sprintf("%s.%s", config.Operation.Schema, config.Operation.Table)
	fields := make([]string, 0, len(config.Mapping.FieldMappings))
	placeholders := make([]string, 0, len(config.Mapping.FieldMappings))

	for i, mapping := range config.Mapping.FieldMappings {
		fields = append(fields, mapping.TargetField)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}

	sqlQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "))

	// 准备语句
	var stmt *sql.Stmt
	var err error
	if tx != nil {
		stmt, err = tx.PrepareContext(ctx, sqlQuery)
	} else {
		stmt, err = db.PrepareContext(ctx, sqlQuery)
	}
	if err != nil {
		return 0, len(batch), fmt.Errorf("准备插入语句失败: %v", err)
	}
	defer stmt.Close()

	// 执行插入
	successCount := 0
	errorCount := 0

	for _, itemRaw := range batch {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			errorCount++
			continue
		}

		values := make([]interface{}, 0, len(config.Mapping.FieldMappings))
		for _, mapping := range config.Mapping.FieldMappings {
			value, exists := item[mapping.SourceField]
			if !exists {
				value = nil
			}
			values = append(values, value)
		}

		_, err := stmt.ExecContext(ctx, values...)
		if err != nil {
			errorCount++
			if !config.Options.IgnoreErrors {
				return successCount, errorCount, fmt.Errorf("插入记录失败: %v", err)
			}
		} else {
			successCount++
		}
	}

	return successCount, errorCount, nil
}

// updateBatch 批量更新
func (p *PostgreSQLOutputNode) updateBatch(ctx context.Context, db *sql.DB, tx *sql.Tx, batch []interface{}, config *OutputConfig) (int, int, error) {
	// 获取主键字段
	keyFields := make([]FieldMapping, 0)
	updateFields := make([]FieldMapping, 0)

	for _, mapping := range config.Mapping.FieldMappings {
		if mapping.IsKey {
			keyFields = append(keyFields, mapping)
		} else {
			updateFields = append(updateFields, mapping)
		}
	}

	if len(keyFields) == 0 {
		return 0, len(batch), fmt.Errorf("更新操作需要指定主键字段")
	}

	// 构建更新SQL
	tableName := fmt.Sprintf("%s.%s", config.Operation.Schema, config.Operation.Table)
	setClauses := make([]string, 0, len(updateFields))
	whereClauses := make([]string, 0, len(keyFields))

	paramIndex := 1
	for _, mapping := range updateFields {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", mapping.TargetField, paramIndex))
		paramIndex++
	}

	for _, mapping := range keyFields {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", mapping.TargetField, paramIndex))
		paramIndex++
	}

	sqlQuery := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	// 准备语句
	var stmt *sql.Stmt
	var err error
	if tx != nil {
		stmt, err = tx.PrepareContext(ctx, sqlQuery)
	} else {
		stmt, err = db.PrepareContext(ctx, sqlQuery)
	}
	if err != nil {
		return 0, len(batch), fmt.Errorf("准备更新语句失败: %v", err)
	}
	defer stmt.Close()

	// 执行更新
	successCount := 0
	errorCount := 0

	for _, itemRaw := range batch {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			errorCount++
			continue
		}

		values := make([]interface{}, 0, len(updateFields)+len(keyFields))

		// 添加更新字段值
		for _, mapping := range updateFields {
			value, exists := item[mapping.SourceField]
			if !exists {
				value = nil
			}
			values = append(values, value)
		}

		// 添加主键字段值
		for _, mapping := range keyFields {
			value, exists := item[mapping.SourceField]
			if !exists {
				value = nil
			}
			values = append(values, value)
		}

		_, err := stmt.ExecContext(ctx, values...)
		if err != nil {
			errorCount++
			if !config.Options.IgnoreErrors {
				return successCount, errorCount, fmt.Errorf("更新记录失败: %v", err)
			}
		} else {
			successCount++
		}
	}

	return successCount, errorCount, nil
}

// deleteBatch 批量删除
func (p *PostgreSQLOutputNode) deleteBatch(ctx context.Context, db *sql.DB, tx *sql.Tx, batch []interface{}, config *OutputConfig) (int, int, error) {
	// 获取主键字段
	keyFields := make([]FieldMapping, 0)
	for _, mapping := range config.Mapping.FieldMappings {
		if mapping.IsKey {
			keyFields = append(keyFields, mapping)
		}
	}

	if len(keyFields) == 0 {
		return 0, len(batch), fmt.Errorf("删除操作需要指定主键字段")
	}

	// 构建删除SQL
	tableName := fmt.Sprintf("%s.%s", config.Operation.Schema, config.Operation.Table)
	whereClauses := make([]string, 0, len(keyFields))

	for i, mapping := range keyFields {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", mapping.TargetField, i+1))
	}

	sqlQuery := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "))

	// 准备语句
	var stmt *sql.Stmt
	var err error
	if tx != nil {
		stmt, err = tx.PrepareContext(ctx, sqlQuery)
	} else {
		stmt, err = db.PrepareContext(ctx, sqlQuery)
	}
	if err != nil {
		return 0, len(batch), fmt.Errorf("准备删除语句失败: %v", err)
	}
	defer stmt.Close()

	// 执行删除
	successCount := 0
	errorCount := 0

	for _, itemRaw := range batch {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			errorCount++
			continue
		}

		values := make([]interface{}, 0, len(keyFields))
		for _, mapping := range keyFields {
			value, exists := item[mapping.SourceField]
			if !exists {
				value = nil
			}
			values = append(values, value)
		}

		_, err := stmt.ExecContext(ctx, values...)
		if err != nil {
			errorCount++
			if !config.Options.IgnoreErrors {
				return successCount, errorCount, fmt.Errorf("删除记录失败: %v", err)
			}
		} else {
			successCount++
		}
	}

	return successCount, errorCount, nil
}

// upsertBatch 批量更新插入
func (p *PostgreSQLOutputNode) upsertBatch(ctx context.Context, db *sql.DB, tx *sql.Tx, batch []interface{}, config *OutputConfig) (int, int, error) {
	// PostgreSQL使用ON CONFLICT实现upsert
	keyFields := make([]FieldMapping, 0)
	allFields := config.Mapping.FieldMappings

	for _, mapping := range config.Mapping.FieldMappings {
		if mapping.IsKey {
			keyFields = append(keyFields, mapping)
		}
	}

	if len(keyFields) == 0 {
		return 0, len(batch), fmt.Errorf("upsert操作需要指定主键字段")
	}

	// 构建upsert SQL
	tableName := fmt.Sprintf("%s.%s", config.Operation.Schema, config.Operation.Table)
	fields := make([]string, 0, len(allFields))
	placeholders := make([]string, 0, len(allFields))
	updateClauses := make([]string, 0)
	conflictFields := make([]string, 0, len(keyFields))

	for i, mapping := range allFields {
		fields = append(fields, mapping.TargetField)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		if !mapping.IsKey {
			updateClauses = append(updateClauses, fmt.Sprintf("%s = EXCLUDED.%s", mapping.TargetField, mapping.TargetField))
		}
	}

	for _, mapping := range keyFields {
		conflictFields = append(conflictFields, mapping.TargetField)
	}

	sqlQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (%s) DO UPDATE SET %s",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(conflictFields, ", "),
		strings.Join(updateClauses, ", "))

	// 准备语句
	var stmt *sql.Stmt
	var err error
	if tx != nil {
		stmt, err = tx.PrepareContext(ctx, sqlQuery)
	} else {
		stmt, err = db.PrepareContext(ctx, sqlQuery)
	}
	if err != nil {
		return 0, len(batch), fmt.Errorf("准备upsert语句失败: %v", err)
	}
	defer stmt.Close()

	// 执行upsert
	successCount := 0
	errorCount := 0

	for _, itemRaw := range batch {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			errorCount++
			continue
		}

		values := make([]interface{}, 0, len(allFields))
		for _, mapping := range allFields {
			value, exists := item[mapping.SourceField]
			if !exists {
				value = nil
			}
			values = append(values, value)
		}

		_, err := stmt.ExecContext(ctx, values...)
		if err != nil {
			errorCount++
			if !config.Options.IgnoreErrors {
				return successCount, errorCount, fmt.Errorf("upsert记录失败: %v", err)
			}
		} else {
			successCount++
		}
	}

	return successCount, errorCount, nil
}

// parseConfig 解析配置
func (p *PostgreSQLOutputNode) parseConfig(config map[string]interface{}) *OutputConfig {
	result := &OutputConfig{
		Connection: ConnectionConfig{
			Host:    "localhost",
			Port:    5432,
			SSLMode: "disable",
		},
		Operation: OperationConfig{
			Type:      "insert",
			Schema:    "public",
			BatchSize: 1000,
		},
		Options: OptionsConfig{
			UseTransaction: true,
			IgnoreErrors:   false,
			Timeout:        30,
		},
	}

	// 解析连接配置
	if conn, exists := config["connection"]; exists {
		if connMap, ok := conn.(map[string]interface{}); ok {
			if host, exists := connMap["host"]; exists {
				result.Connection.Host = host.(string)
			}
			if port, exists := connMap["port"]; exists {
				if p, ok := port.(float64); ok {
					result.Connection.Port = int(p)
				}
			}
			if database, exists := connMap["database"]; exists {
				result.Connection.Database = database.(string)
			}
			if username, exists := connMap["username"]; exists {
				result.Connection.Username = username.(string)
			}
			if password, exists := connMap["password"]; exists {
				result.Connection.Password = password.(string)
			}
			if sslmode, exists := connMap["sslmode"]; exists {
				result.Connection.SSLMode = sslmode.(string)
			}
		}
	}

	// 解析操作配置
	if op, exists := config["operation"]; exists {
		if opMap, ok := op.(map[string]interface{}); ok {
			if opType, exists := opMap["type"]; exists {
				result.Operation.Type = opType.(string)
			}
			if table, exists := opMap["table"]; exists {
				result.Operation.Table = table.(string)
			}
			if schema, exists := opMap["schema"]; exists {
				result.Operation.Schema = schema.(string)
			}
			if batchSize, exists := opMap["batch_size"]; exists {
				if bs, ok := batchSize.(float64); ok {
					result.Operation.BatchSize = int(bs)
				}
			}
		}
	}

	// 解析字段映射
	if mapping, exists := config["mapping"]; exists {
		if mappingMap, ok := mapping.(map[string]interface{}); ok {
			if fieldMappings, exists := mappingMap["field_mappings"]; exists {
				if mappings, ok := fieldMappings.([]interface{}); ok {
					result.Mapping.FieldMappings = make([]FieldMapping, 0, len(mappings))
					for _, mappingRaw := range mappings {
						if m, ok := mappingRaw.(map[string]interface{}); ok {
							mapping := FieldMapping{
								DataType: "text",
								IsKey:    false,
							}
							if sourceField, exists := m["source_field"]; exists {
								mapping.SourceField = sourceField.(string)
							}
							if targetField, exists := m["target_field"]; exists {
								mapping.TargetField = targetField.(string)
							}
							if dataType, exists := m["data_type"]; exists {
								mapping.DataType = dataType.(string)
							}
							if isKey, exists := m["is_key"]; exists {
								if ik, ok := isKey.(bool); ok {
									mapping.IsKey = ik
								}
							}
							result.Mapping.FieldMappings = append(result.Mapping.FieldMappings, mapping)
						}
					}
				}
			}
		}
	}

	// 解析选项配置
	if options, exists := config["options"]; exists {
		if optionsMap, ok := options.(map[string]interface{}); ok {
			if useTransaction, exists := optionsMap["use_transaction"]; exists {
				if ut, ok := useTransaction.(bool); ok {
					result.Options.UseTransaction = ut
				}
			}
			if ignoreErrors, exists := optionsMap["ignore_errors"]; exists {
				if ie, ok := ignoreErrors.(bool); ok {
					result.Options.IgnoreErrors = ie
				}
			}
			if timeout, exists := optionsMap["timeout"]; exists {
				if t, ok := timeout.(float64); ok {
					result.Options.Timeout = int(t)
				}
			}
		}
	}

	return result
}

// 配置结构体
type OutputConfig struct {
	Connection ConnectionConfig `json:"connection"`
	Operation  OperationConfig  `json:"operation"`
	Mapping    MappingConfig    `json:"mapping"`
	Options    OptionsConfig    `json:"options"`
}

type ConnectionConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"sslmode"`
}

type OperationConfig struct {
	Type      string `json:"type"`
	Table     string `json:"table"`
	Schema    string `json:"schema"`
	BatchSize int    `json:"batch_size"`
}

type MappingConfig struct {
	FieldMappings []FieldMapping `json:"field_mappings"`
}

type FieldMapping struct {
	SourceField string `json:"source_field"`
	TargetField string `json:"target_field"`
	DataType    string `json:"data_type"`
	IsKey       bool   `json:"is_key"`
}

type OptionsConfig struct {
	UseTransaction bool `json:"use_transaction"`
	IgnoreErrors   bool `json:"ignore_errors"`
	Timeout        int  `json:"timeout"`
}

// GetDynamicData 获取动态配置数据（默认实现）
func (p *PostgreSQLOutputNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("PostgreSQL输出节点暂不支持动态数据获取方法: %s", method)
}
