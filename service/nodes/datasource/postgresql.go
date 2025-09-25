/**
 * @module postgresql_datasource
 * @description PostgreSQL数据源节点，提供数据库连接和查询功能
 * @architecture 数据源插件实现，支持连接池和查询优化
 * @documentReference ai_docs/refactor_plan.md
 * @stateFlow datasource_states: configured -> connected -> querying -> completed
 * @rules 必须验证数据库连接，支持参数化查询防止SQL注入
 * @dependencies database/sql, lib/pq, context, time
 * @refs service/nodes/interface.go
 */

package datasource

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"flow-service/service/nodes"

	_ "github.com/lib/pq" // PostgreSQL 驱动
)

// init 自动注册PostgreSQL节点
func init() {
	registry := nodes.GetRegistry()
	if err := registry.Register(NewPostgreSQLNode()); err != nil {
		log.Printf("注册PostgreSQL节点失败: %v", err)
	} else {
		log.Println("PostgreSQL节点注册成功")
	}
}

// PostgreSQLNode PostgreSQL数据源节点
type PostgreSQLNode struct {
	db *sql.DB
}

// NewPostgreSQLNode 创建PostgreSQL节点
func NewPostgreSQLNode() *PostgreSQLNode {
	return &PostgreSQLNode{}
}

// GetMetadata 获取节点元数据
func (p *PostgreSQLNode) GetMetadata() *nodes.NodeMetadata {
	return &nodes.NodeMetadata{
		ID:          "postgresql_datasource",
		Name:        "PostgreSQL数据源",
		Description: "从PostgreSQL数据库查询数据",
		Version:     "1.0.0",
		Category:    nodes.CategoryDataSource,
		Type:        nodes.TypePostgreSQL,
		Icon:        "database",
		Tags:        []string{"数据库", "SQL", "查询"},

		InputPorts: []nodes.PortDefinition{},

		OutputPorts: []nodes.PortDefinition{
			{
				ID:          "data",
				Name:        "查询结果",
				Description: "数据库查询返回的数据",
				DataType:    nodes.DataTypeArray,
				Required:    true,
				Multiple:    false,
			},
			{
				ID:          "metadata",
				Name:        "查询元数据",
				Description: "查询执行的元数据信息",
				DataType:    nodes.DataTypeObject,
				Required:    false,
				Multiple:    false,
			},
		},

		ConfigSchema: &nodes.ConfigSchema{
			Type: "object",
			Properties: []nodes.ConfigField{
				{
					Name:        "host",
					Type:        "string",
					Title:       "主机地址",
					Description: "数据库服务器地址",
					Default:     "localhost",
					Widget:      nodes.WidgetText,
					Placeholder: "localhost",
				},
				{
					Name:        "port",
					Type:        "number",
					Title:       "端口",
					Description: "数据库服务器端口",
					Default:     5432,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "database",
					Type:        "string",
					Title:       "数据库名",
					Description: "要连接的数据库名称",
					Widget:      nodes.WidgetText,
					Placeholder: "database_name",
				},
				{
					Name:        "username",
					Type:        "string",
					Title:       "用户名",
					Description: "数据库用户名",
					Widget:      nodes.WidgetText,
					Placeholder: "username",
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
					Enum:        []interface{}{"disable", "require", "prefer"},
					Widget:      nodes.WidgetSelect,
				},
				{
					Name:        "sql",
					Type:        "string",
					Title:       "SQL语句",
					Description: "要执行的SQL查询语句，支持参数占位符 $1, $2...",
					Widget:      nodes.WidgetCode,
					Placeholder: "SELECT * FROM table_name WHERE id = $1",
				},
				{
					Name:        "params",
					Type:        "array",
					Title:       "查询参数",
					Description: "SQL查询参数列表，按$1, $2...顺序提供",
					Items: &nodes.ConfigField{
						Type: "string",
					},
					Widget:      nodes.WidgetJSON,
					Placeholder: `["value1", "value2"]`,
				},
				{
					Name:        "timeout",
					Type:        "number",
					Title:       "查询超时(秒)",
					Description: "查询执行超时时间",
					Default:     30,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "limit",
					Type:        "number",
					Title:       "结果限制",
					Description: "限制返回结果的数量，0表示不限制",
					Default:     1000,
					Widget:      nodes.WidgetNumber,
				},
				{
					Name:        "table_name",
					Type:        "string",
					Title:       "表名",
					Description: "目标数据表名称",
					Widget:      nodes.WidgetSelect,
					Placeholder: "table_name",
					DataSource: &nodes.DataSourceConfig{
						Type:         "method",
						Method:       "GetTableNames",
						Dependencies: []string{"host", "port", "database", "username", "password"},
						CacheTime:    300, // 缓存5分钟
						Fallback:     []string{},
					},
				},
			},
			Required: []string{"host", "port", "database", "username", "password", "sql"},
		},

		Author:    "Flow Service Team",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Validate 验证节点配置
func (p *PostgreSQLNode) Validate(config map[string]interface{}) error {
	// 验证连接配置
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

	// 验证查询配置
	if sql, ok := config["sql"].(string); !ok || sql == "" {
		return fmt.Errorf("missing or invalid sql statement")
	}

	return nil
}

// Execute 执行节点
func (p *PostgreSQLNode) Execute(ctx context.Context, input *nodes.NodeInput) (*nodes.NodeOutput, error) {
	startTime := time.Now()
	output := &nodes.NodeOutput{
		Data:    make(map[string]interface{}),
		Logs:    []string{},
		Metrics: make(map[string]interface{}),
		Success: false,
	}

	// 解析配置
	connectionConfig := map[string]interface{}{
		"host":     input.Config["host"],
		"port":     input.Config["port"],
		"database": input.Config["database"],
		"username": input.Config["username"],
		"password": input.Config["password"],
		"sslmode":  input.Config["sslmode"],
	}

	queryConfig := map[string]interface{}{
		"sql":     input.Config["sql"],
		"params":  input.Config["params"],
		"timeout": input.Config["timeout"],
		"limit":   input.Config["limit"],
	}

	// 建立数据库连接
	if err := p.connect(connectionConfig); err != nil {
		output.Error = fmt.Sprintf("database connection failed: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}
	defer p.close()

	output.Logs = append(output.Logs, "数据库连接成功")

	// 执行查询
	sql := queryConfig["sql"].(string)
	timeout := p.getTimeout(queryConfig)

	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 获取查询参数
	var params []interface{}
	if paramsConfig, exists := queryConfig["params"]; exists {
		if paramArray, ok := paramsConfig.([]interface{}); ok {
			for _, param := range paramArray {
				params = append(params, param)
			}
		}
	}

	// 执行查询
	rows, err := p.db.QueryContext(queryCtx, sql, params...)
	if err != nil {
		output.Error = fmt.Sprintf("query execution failed: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}
	defer rows.Close()

	// 处理结果
	result, metadata, err := p.processResults(rows, queryConfig)
	if err != nil {
		output.Error = fmt.Sprintf("result processing failed: %v", err)
		output.Duration = time.Since(startTime)
		return output, nil
	}

	output.Data["data"] = result
	output.Data["metadata"] = metadata
	output.Success = true
	output.Duration = time.Since(startTime)
	output.Logs = append(output.Logs, fmt.Sprintf("查询成功，返回 %d 行数据", len(result)))

	// 添加指标
	output.Metrics["query_duration"] = time.Since(startTime).Milliseconds()
	output.Metrics["rows_count"] = len(result)

	return output, nil
}

// connect 建立数据库连接
func (p *PostgreSQLNode) connect(config map[string]interface{}) error {
	host := config["host"].(string)
	port := int(config["port"].(float64))
	database := config["database"].(string)
	username := config["username"].(string)
	password := ""
	if pwd, ok := config["password"].(string); ok {
		password = pwd
	}
	sslmode := "disable"
	if ssl, ok := config["sslmode"].(string); ok {
		sslmode = ssl
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, username, password, database, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}

	p.db = db
	return nil
}

// close 关闭数据库连接
func (p *PostgreSQLNode) close() {
	if p.db != nil {
		p.db.Close()
		p.db = nil
	}
}

// getTimeout 获取查询超时时间
func (p *PostgreSQLNode) getTimeout(query map[string]interface{}) time.Duration {
	if timeout, ok := query["timeout"].(float64); ok {
		return time.Duration(timeout) * time.Second
	}
	return 30 * time.Second
}

// processResults 处理查询结果
func (p *PostgreSQLNode) processResults(rows *sql.Rows, query map[string]interface{}) ([]map[string]interface{}, map[string]interface{}, error) {
	// 获取列信息
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}

	// 准备结果容器
	var result []map[string]interface{}
	limit := p.getLimit(query)
	count := 0

	// 处理每一行
	for rows.Next() {
		if limit > 0 && count >= limit {
			break
		}

		// 创建扫描目标
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, err
		}

		// 转换为map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				// 转换字节切片为字符串
				if b, ok := val.([]byte); ok {
					row[col] = string(b)
				} else {
					row[col] = val
				}
			} else {
				row[col] = nil
			}
		}

		result = append(result, row)
		count++
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// 创建元数据
	metadata := map[string]interface{}{
		"columns":      columns,
		"column_types": p.getColumnTypeNames(columnTypes),
		"row_count":    len(result),
		"truncated":    limit > 0 && count >= limit,
	}

	return result, metadata, nil
}

// getLimit 获取结果限制
func (p *PostgreSQLNode) getLimit(query map[string]interface{}) int {
	if limit, ok := query["limit"].(float64); ok {
		return int(limit)
	}
	return 1000
}

// getColumnTypeNames 获取列类型名称
func (p *PostgreSQLNode) getColumnTypeNames(columnTypes []*sql.ColumnType) []string {
	var typeNames []string
	for _, ct := range columnTypes {
		typeNames = append(typeNames, ct.DatabaseTypeName())
	}
	return typeNames
}

// GetDynamicData 获取动态配置数据
func (p *PostgreSQLNode) GetDynamicData(method string, params map[string]interface{}) (interface{}, error) {
	switch method {
	case "GetTableNames":
		return p.getTableNames(params)
	case "GetColumnNames":
		return p.getColumnNames(params)
	default:
		return nil, fmt.Errorf("不支持的方法: %s", method)
	}
}

// getTableNames 获取数据库表名列表
func (p *PostgreSQLNode) getTableNames(params map[string]interface{}) ([]string, error) {
	// 解析连接参数
	host, ok := params["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("缺少主机地址")
	}

	port := 5432
	if p, ok := params["port"].(float64); ok {
		port = int(p)
	}

	database, ok := params["database"].(string)
	if !ok || database == "" {
		return nil, fmt.Errorf("缺少数据库名")
	}

	username, ok := params["username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("缺少用户名")
	}

	password, ok := params["password"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少密码")
	}

	sslmode := "disable"
	if ssl, ok := params["sslmode"].(string); ok {
		sslmode = ssl
	}

	// 建立数据库连接
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, username, password, database, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	// 查询表名
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询表名失败: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取查询结果失败: %v", err)
	}

	return tables, nil
}

// getColumnNames 获取指定表的列名列表
func (p *PostgreSQLNode) getColumnNames(params map[string]interface{}) ([]string, error) {
	tableName, ok := params["table_name"].(string)
	if !ok || tableName == "" {
		return nil, fmt.Errorf("缺少表名")
	}

	// 解析连接参数（与getTableNames相同的逻辑）
	host, ok := params["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("缺少主机地址")
	}

	port := 5432
	if p, ok := params["port"].(float64); ok {
		port = int(p)
	}

	database, ok := params["database"].(string)
	if !ok || database == "" {
		return nil, fmt.Errorf("缺少数据库名")
	}

	username, ok := params["username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("缺少用户名")
	}

	password, ok := params["password"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少密码")
	}

	sslmode := "disable"
	if ssl, ok := params["sslmode"].(string); ok {
		sslmode = ssl
	}

	// 建立数据库连接
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, username, password, database, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	// 查询列名
	query := `
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("查询列名失败: %v", err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			continue
		}
		columns = append(columns, columnName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取查询结果失败: %v", err)
	}

	return columns, nil
}
