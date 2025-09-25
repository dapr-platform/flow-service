# Flow Service 节点开发指南

本文档介绍如何在 Flow Service 中开发新的节点插件。

## 节点系统架构

Flow Service 采用插件化架构，所有节点都实现`NodePlugin`接口，通过注册机制动态加载。节点系统支持：

- 📦 **插件化设计**：每个节点都是独立的插件
- 🔌 **动态注册**：通过 init 函数自动注册节点
- 📝 **元数据驱动**：通过元数据定义节点的所有特性
- 🎨 **前端友好**：提供完整的 UI 配置 Schema

## 新建节点的完整步骤

### 1. 创建节点文件

在相应的分类目录下创建节点文件：

```
service/nodes/
├── datasource/     # 数据源节点
├── transform/      # 数据处理节点
├── output/         # 输出节点
└── control/        # 控制节点
```

### 2. 实现 NodePlugin 接口

每个节点必须实现以下四个方法：

```go
type NodePlugin interface {
    // 获取节点元数据
    GetMetadata() *NodeMetadata

    // 获取配置模式定义（UI配置界面）
    GetSchema() *ConfigSchema

    // 验证节点配置
    Validate(config map[string]interface{}) error

    // 执行节点逻辑
    Execute(ctx context.Context, input *NodeInput) (*NodeOutput, error)
}
```

### 3. 定义节点元数据

节点元数据包含了节点的所有基础信息：

```go
func (n *YourNode) GetMetadata() *NodeMetadata {
    return &NodeMetadata{
        // === 基础信息 ===
        ID:          "your_node_id",        // 唯一标识符
        Name:        "您的节点名称",            // 显示名称
        Description: "节点功能描述",           // 功能说明
        Version:     "1.0.0",              // 版本号
        Category:    nodes.CategoryDataSource, // 节点分类
        Type:        nodes.TypeAPI,         // 节点类型
        Icon:        "api",                 // 图标名称
        Tags:        []string{"HTTP", "API"}, // 标签
        Capabilities: []string{"http_request"}, // 能力列表

        // === 输入输出端口定义 ===
        InputPorts:  []PortDefinition{...},  // 输入端口
        OutputPorts: []PortDefinition{...},  // 输出端口

        // === 资源需求 ===
        Resources: &ResourceRequirement{
            CPU:    "50m",
            Memory: "64Mi",
        },

        // === 创建信息 ===
        Author:    "您的名字",
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}
```

### 4. 定义输入输出端口

端口定义决定了节点在前端的连接点绘制：

```go
// 输入端口定义
InputPorts: []PortDefinition{
    {
        ID:          "data",              // 端口唯一标识
        Name:        "输入数据",           // 端口显示名称
        Description: "需要处理的数据",      // 端口描述
        DataType:    DataTypeArray,      // 数据类型
        Required:    true,               // 是否必需
        Multiple:    false,              // 是否支持多个连接
        Schema: &DataSchema{             // 数据结构定义
            Type: "array",
            Items: &DataSchema{
                Type: "object",
            },
        },
    },
},

// 输出端口定义
OutputPorts: []PortDefinition{
    {
        ID:          "result",
        Name:        "处理结果",
        Description: "处理后的数据",
        DataType:    DataTypeArray,
        Required:    true,
        Multiple:    false,
    },
    {
        ID:          "error_data",
        Name:        "错误数据",
        Description: "处理失败的数据",
        DataType:    DataTypeArray,
        Required:    false,              // 可选输出
        Multiple:    false,
    },
},
```

### 5. 定义配置 Schema

配置 Schema 决定了前端配置界面的生成：

```go
func (n *YourNode) GetSchema() *ConfigSchema {
    return &ConfigSchema{
        Type: "object",
        Properties: map[string]*ConfigField{
            "basic": {
                Type:        "object",
                Title:       "基础配置",
                Description: "节点基础配置",
                Order:       1,
                Properties: map[string]*ConfigField{
                    "url": {
                        Type:        "string",
                        Title:       "请求URL",
                        Description: "HTTP请求地址",
                        Widget:      WidgetText,
                        Placeholder: "https://api.example.com",
                        Order:       1,
                    },
                    "method": {
                        Type:    "string",
                        Title:   "HTTP方法",
                        Default: "GET",
                        Enum:    []interface{}{"GET", "POST", "PUT", "DELETE"},
                        Widget:  WidgetSelect,
                        Order:   2,
                    },
                },
                Required: []string{"url"},
            },
        },
        Required: []string{"basic"},
        Groups: []ConfigGroup{
            {
                ID:          "basic",
                Title:       "基础配置",
                Description: "配置基本参数",
                Fields:      []string{"basic"},
                Order:       1,
            },
        },
    }
}
```

### 6. 实现配置验证

```go
func (n *YourNode) Validate(config map[string]interface{}) error {
    basic, ok := config["basic"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("missing basic config")
    }

    if url, ok := basic["url"].(string); !ok || url == "" {
        return fmt.Errorf("missing or invalid URL")
    }

    return nil
}
```

### 7. 实现执行逻辑

```go
func (n *YourNode) Execute(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
    startTime := time.Now()
    output := &NodeOutput{
        Data:    make(map[string]interface{}),
        Logs:    []string{},
        Metrics: make(map[string]interface{}),
        Success: false,
    }

    // 获取输入数据
    inputData, exists := input.Data["data"]
    if !exists {
        output.Error = "missing input data"
        output.Duration = time.Since(startTime)
        return output, nil
    }

    // 执行业务逻辑
    result, err := n.processData(inputData, input.Config)
    if err != nil {
        output.Error = err.Error()
        output.Duration = time.Since(startTime)
        return output, nil
    }

    // 设置输出
    output.Data["result"] = result
    output.Success = true
    output.Duration = time.Since(startTime)
    output.Logs = append(output.Logs, "处理完成")

    return output, nil
}
```

### 8. 注册节点

在 init 函数中自动注册节点：

```go
func init() {
    registry := nodes.GetRegistry()
    if err := registry.Register(NewYourNode()); err != nil {
        log.Printf("注册节点失败: %v", err)
    } else {
        log.Println("节点注册成功")
    }
}
```

## 节点输入输出模式

节点通过`InputPorts`和`OutputPorts`定义输入输出端口，支持以下模式：

### 1. 仅输出节点（数据源节点）

```go
InputPorts:  []PortDefinition{},  // 无输入端口
OutputPorts: []PortDefinition{
    {
        ID:       "data",
        Name:     "数据输出",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
```

**前端绘制**：节点左侧无连接点，右侧有一个输出连接点

### 2. 仅输入节点（输出节点）

```go
InputPorts: []PortDefinition{
    {
        ID:       "data",
        Name:     "数据输入",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
OutputPorts: []PortDefinition{},  // 无输出端口
```

**前端绘制**：节点左侧有一个输入连接点，右侧无连接点

### 3. 单输入单输出节点（处理节点）

```go
InputPorts: []PortDefinition{
    {
        ID:       "data",
        Name:     "输入数据",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
OutputPorts: []PortDefinition{
    {
        ID:       "result",
        Name:     "处理结果",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
```

**前端绘制**：节点左侧有一个输入连接点，右侧有一个输出连接点

### 4. 多输入单输出节点（聚合节点）

```go
InputPorts: []PortDefinition{
    {
        ID:       "data1",
        Name:     "数据源1",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
    {
        ID:       "data2",
        Name:     "数据源2",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
OutputPorts: []PortDefinition{
    {
        ID:       "merged_data",
        Name:     "合并结果",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
```

**前端绘制**：节点左侧有两个输入连接点，右侧有一个输出连接点

### 5. 单输入多输出节点（分流节点）

```go
InputPorts: []PortDefinition{
    {
        ID:       "data",
        Name:     "输入数据",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
},
OutputPorts: []PortDefinition{
    {
        ID:       "success_data",
        Name:     "成功数据",
        DataType: DataTypeArray,
        Required: true,
        Multiple: false,
    },
    {
        ID:       "error_data",
        Name:     "错误数据",
        DataType: DataTypeArray,
        Required: false,  // 可选输出
        Multiple: false,
    },
},
```

**前端绘制**：节点左侧有一个输入连接点，右侧有两个输出连接点

### 6. 支持多连接的端口

```go
InputPorts: []PortDefinition{
    {
        ID:       "data",
        Name:     "数据输入",
        DataType: DataTypeArray,
        Required: true,
        Multiple: true,    // 支持多个连接
    },
},
```

**前端绘制**：该连接点可以接受多个输入连线

## 端口配置详解

### PortDefinition 字段说明

```go
type PortDefinition struct {
    ID          string      `json:"id"`           // 端口唯一标识
    Name        string      `json:"name"`         // 显示名称
    Description string      `json:"description"`  // 描述信息
    DataType    string      `json:"data_type"`    // 数据类型
    Required    bool        `json:"required"`     // 是否必需
    Multiple    bool        `json:"multiple"`     // 是否支持多连接
    Schema      *DataSchema `json:"schema"`       // 数据结构定义
}
```

### 数据类型常量

```go
const (
    DataTypeString  = "string"   // 字符串
    DataTypeNumber  = "number"   // 数字
    DataTypeBoolean = "boolean"  // 布尔值
    DataTypeArray   = "array"    // 数组
    DataTypeObject  = "object"   // 对象
    DataTypeAny     = "any"      // 任意类型
)
```

### Required 字段的含义

- `Required: true` - 必需端口，必须有连接才能执行
- `Required: false` - 可选端口，没有连接也能执行

### Multiple 字段的含义

- `Multiple: true` - 支持多个连接，可以从多个节点接收数据
- `Multiple: false` - 只能接受一个连接

## 前端节点绘制规则

前端根据节点元数据绘制节点和连接点：

1. **节点外观**：根据`Category`和`Icon`决定节点样式和图标
2. **输入连接点**：在节点左侧绘制`InputPorts`定义的连接点
3. **输出连接点**：在节点右侧绘制`OutputPorts`定义的连接点
4. **连接点颜色**：根据`DataType`显示不同颜色
5. **必需标识**：`Required: true`的端口会有特殊标识
6. **多连接支持**：`Multiple: true`的端口可以接受多条连线

## 节点分类和类型

### 节点分类（Category）

```go
const (
    CategoryDataSource = "datasource" // 数据源
    CategoryTransform  = "transform"  // 数据处理
    CategoryOutput     = "output"     // 输出
    CategoryControl    = "control"    // 控制
    CategoryLogic      = "logic"      // 逻辑
    CategoryUtility    = "utility"    // 工具
)
```

### 节点类型（Type）

类型比分类更具体，决定节点的具体功能：

```go
const (
    // 数据源类型
    TypePostgreSQL = "postgresql"
    TypeAPI        = "api"
    TypeFile       = "file"
    TypeStatic     = "static"

    // 数据处理类型
    TypeDataFilter    = "data_filter"
    TypeDataTransform = "data_transform"
    TypeDataAggregate = "data_aggregate"

    // 输出类型
    TypeLogger = "logger"

    // 控制类型
    TypeCondition = "condition"
    TypeLoop      = "loop"
)
```

## 配置 UI 组件类型

```go
const (
    WidgetText     = "text"        // 文本输入框
    WidgetNumber   = "number"      // 数字输入框
    WidgetBoolean  = "boolean"     // 开关/复选框
    WidgetSelect   = "select"      // 下拉选择
    WidgetTextarea = "textarea"    // 多行文本
    WidgetPassword = "password"    // 密码输入框
    WidgetFile     = "file"        // 文件选择
    WidgetCode     = "code"        // 代码编辑器
    WidgetJSON     = "json"        // JSON编辑器
)
```

## 示例：完整的节点实现

参考现有节点实现：

- **数据源节点**：`service/nodes/datasource/api.go` - API 数据源
- **处理节点**：`service/nodes/transform/data_filter.go` - 数据过滤
- **输出节点**：`service/nodes/output/logger.go` - 日志输出

## 最佳实践

1. **错误处理**：使用`NodeOutput.Error`字段返回错误信息，不要直接返回 error
2. **日志记录**：在`NodeOutput.Logs`中记录执行日志，便于调试
3. **性能监控**：在`NodeOutput.Metrics`中记录性能指标
4. **资源配置**：合理设置`Resources`字段，帮助系统进行资源调度
5. **文档注释**：为节点添加完整的文档注释，便于理解和维护

## 节点注册机制

所有节点通过 init 函数自动注册到全局注册表：

```go
func init() {
    registry := nodes.GetRegistry()
    if err := registry.Register(NewYourNode()); err != nil {
        log.Printf("注册节点失败: %v", err)
    }
}
```

系统启动时会自动加载所有注册的节点，无需手动配置。
