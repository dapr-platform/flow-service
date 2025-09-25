# Flow Service 复杂工作流测试用例文档

## 测试概述

本文档描述了 Flow Service 复杂工作流的端到端测试用例，主要测试工作流引擎的逻辑处理能力，包括静态数据源、数据处理、控制流程和输出节点的综合应用。

**API 基础信息:**

- 基础 URL: `http://localhost:8800`
- 认证方式: 无需认证（测试环境）
- 内容类型: `application/json`

## 测试用例列表

### 1. 基础数据流工作流测试

#### TC-CWF-001: 静态数据 -> 数据过滤 -> Logger 输出

- **测试 ID**: TC-CWF-001
- **描述**: 创建包含静态数据源、数据过滤和 Logger 输出的基础工作流
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "基础数据过滤工作流",
    "description": "测试静态数据 -> 数据过滤 -> Logger输出的基础流程",
    "version": "1.0.0",
    "nodes": {
      "static_data": {
        "id": "static_data",
        "name": "员工数据源",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [
              {
                "id": 1,
                "name": "张三",
                "age": 28,
                "department": "技术部",
                "salary": 8000
              },
              {
                "id": 2,
                "name": "李四",
                "age": 32,
                "department": "销售部",
                "salary": 6000
              },
              {
                "id": 3,
                "name": "王五",
                "age": 25,
                "department": "技术部",
                "salary": 7000
              },
              {
                "id": 4,
                "name": "赵六",
                "age": 35,
                "department": "管理部",
                "salary": 12000
              },
              {
                "id": 5,
                "name": "钱七",
                "age": 29,
                "department": "技术部",
                "salary": 9000
              }
            ]
          }
        }
      },
      "data_filter": {
        "id": "data_filter",
        "name": "技术部员工过滤",
        "type": "data_filter",
        "plugin": "data_filter",
        "config": {
          "conditions": [
            {
              "field": "department",
              "operator": "equals",
              "value": "技术部",
              "data_type": "string"
            },
            {
              "field": "salary",
              "operator": "greater",
              "value": "7500",
              "data_type": "number"
            }
          ],
          "logic": "and"
        }
      },
      "logger_output": {
        "id": "logger_output",
        "name": "过滤结果输出",
        "type": "logger",
        "plugin": "logger",
        "config": {
          "logging": {
            "level": "info",
            "prefix": "[过滤结果]",
            "include_timestamp": true,
            "include_metadata": true
          },
          "format": {
            "output_format": "json",
            "show_data_type": true
          }
        }
      }
    },
    "edges": [
      {
        "id": "edge1",
        "source": "static_data",
        "target": "data_filter",
        "source_port": "data",
        "target_port": "data"
      },
      {
        "id": "edge2",
        "source": "data_filter",
        "target": "logger_output",
        "source_port": "filtered_data",
        "target_port": "input_data"
      }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 201
  - 响应体: 包含创建的工作流信息和工作流 ID

#### TC-CWF-002: 静态数据 -> 数据转换 -> Logger 输出

- **测试 ID**: TC-CWF-002
- **描述**: 创建包含数据转换功能的工作流
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "数据转换工作流",
    "description": "测试静态数据转换处理",
    "version": "1.0.0",
    "nodes": {
      "product_data": {
        "id": "product_data",
        "name": "产品数据源",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [
              {
                "product_id": "P001",
                "name": "笔记本电脑",
                "price": 5999.99,
                "category": "电子产品"
              },
              {
                "product_id": "P002",
                "name": "无线鼠标",
                "price": 129.99,
                "category": "电子产品"
              },
              {
                "product_id": "P003",
                "name": "机械键盘",
                "price": 399.99,
                "category": "电子产品"
              }
            ]
          }
        }
      },
      "data_transform": {
        "id": "data_transform",
        "name": "价格转换",
        "type": "data_transform",
        "plugin": "data_transform",
        "config": {
          "mappings": [
            {
              "source_field": "product_id",
              "target_field": "id",
              "transform_type": "direct"
            },
            {
              "source_field": "name",
              "target_field": "product_name",
              "transform_type": "direct"
            },
            {
              "source_field": "price",
              "target_field": "price_yuan",
              "transform_type": "calculate",
              "transform_config": {
                "expression": "value",
                "decimal_places": 2
              }
            },
            {
              "source_field": "price",
              "target_field": "price_formatted",
              "transform_type": "format",
              "transform_config": {
                "format": "currency",
                "currency": "CNY"
              }
            }
          ],
          "global": {
            "preserve_original": false,
            "ignore_errors": false
          }
        }
      },
      "logger_output": {
        "id": "logger_output",
        "name": "转换结果输出",
        "type": "logger",
        "plugin": "logger",
        "config": {
          "logging": {
            "level": "info",
            "prefix": "[转换结果]",
            "include_timestamp": true
          },
          "format": {
            "output_format": "table"
          }
        }
      }
    },
    "edges": [
      {
        "id": "edge1",
        "source": "product_data",
        "target": "data_transform",
        "source_port": "data",
        "target_port": "data"
      },
      {
        "id": "edge2",
        "source": "data_transform",
        "target": "logger_output",
        "source_port": "transformed_data",
        "target_port": "input_data"
      }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 201
  - 响应体: 包含创建的工作流信息

#### TC-CWF-003: 静态数据 -> 数据聚合 -> Logger 输出

- **测试 ID**: TC-CWF-003
- **描述**: 创建包含数据聚合功能的工作流
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "数据聚合工作流",
    "description": "测试静态数据聚合统计",
    "version": "1.0.0",
    "nodes": {
      "sales_data": {
        "id": "sales_data",
        "name": "销售数据源",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [
              {
                "region": "华北",
                "salesperson": "张三",
                "amount": 15000,
                "month": "2024-01"
              },
              {
                "region": "华北",
                "salesperson": "李四",
                "amount": 18000,
                "month": "2024-01"
              },
              {
                "region": "华南",
                "salesperson": "王五",
                "amount": 12000,
                "month": "2024-01"
              },
              {
                "region": "华南",
                "salesperson": "赵六",
                "amount": 16000,
                "month": "2024-01"
              },
              {
                "region": "华北",
                "salesperson": "张三",
                "amount": 17000,
                "month": "2024-02"
              },
              {
                "region": "华南",
                "salesperson": "王五",
                "amount": 14000,
                "month": "2024-02"
              }
            ]
          }
        }
      },
      "data_aggregate": {
        "id": "data_aggregate",
        "name": "区域销售统计",
        "type": "data_aggregate",
        "plugin": "data_aggregate",
        "config": {
          "group_by": ["region"],
          "aggregations": [
            {
              "field": "amount",
              "function": "sum",
              "alias": "total_amount"
            },
            {
              "field": "amount",
              "function": "avg",
              "alias": "avg_amount"
            },
            {
              "field": "amount",
              "function": "count",
              "alias": "sales_count"
            },
            {
              "field": "amount",
              "function": "max",
              "alias": "max_amount"
            }
          ],
          "order_by": [
            {
              "field": "total_amount",
              "direction": "desc"
            }
          ]
        }
      },
      "logger_output": {
        "id": "logger_output",
        "name": "聚合结果输出",
        "type": "logger",
        "plugin": "logger",
        "config": {
          "logging": {
            "level": "info",
            "prefix": "[聚合统计]",
            "include_timestamp": true,
            "include_metadata": true
          },
          "format": {
            "output_format": "json",
            "show_data_type": true
          }
        }
      }
    },
    "edges": [
      {
        "id": "edge1",
        "source": "sales_data",
        "target": "data_aggregate",
        "source_port": "data",
        "target_port": "data"
      },
      {
        "id": "edge2",
        "source": "data_aggregate",
        "target": "logger_output",
        "source_port": "aggregated_data",
        "target_port": "input_data"
      }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 201
  - 响应体: 包含创建的工作流信息

### 2. 复杂数据处理工作流测试

#### TC-CWF-004: 多数据源合并处理工作流

- **测试 ID**: TC-CWF-004
- **描述**: 创建包含多个静态数据源、数据合并和处理的复杂工作流
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "多数据源合并工作流",
    "description": "测试多个数据源的合并处理",
    "version": "1.0.0",
    "nodes": {
      "user_data": {
        "id": "user_data",
        "name": "用户基础数据",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [
              {
                "user_id": 1,
                "name": "张三",
                "email": "zhang@example.com",
                "department": "技术部"
              },
              {
                "user_id": 2,
                "name": "李四",
                "email": "li@example.com",
                "department": "销售部"
              },
              {
                "user_id": 3,
                "name": "王五",
                "email": "wang@example.com",
                "department": "技术部"
              }
            ]
          }
        }
      },
      "performance_data": {
        "id": "performance_data",
        "name": "用户绩效数据",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [
              {
                "user_id": 1,
                "score": 95,
                "projects_completed": 5,
                "rating": "A"
              },
              {
                "user_id": 2,
                "score": 78,
                "projects_completed": 3,
                "rating": "B"
              },
              {
                "user_id": 3,
                "score": 88,
                "projects_completed": 4,
                "rating": "A"
              }
            ]
          }
        }
      },
      "user_filter": {
        "id": "user_filter",
        "name": "技术部用户过滤",
        "type": "data_filter",
        "plugin": "data_filter",
        "config": {
          "conditions": [
            {
              "field": "department",
              "operator": "equals",
              "value": "技术部",
              "data_type": "string"
            }
          ]
        }
      },
      "performance_filter": {
        "id": "performance_filter",
        "name": "高绩效过滤",
        "type": "data_filter",
        "plugin": "data_filter",
        "config": {
          "conditions": [
            {
              "field": "score",
              "operator": "greater_equal",
              "value": "85",
              "data_type": "number"
            }
          ]
        }
      },
      "data_transform": {
        "id": "data_transform",
        "name": "数据合并转换",
        "type": "data_transform",
        "plugin": "data_transform",
        "config": {
          "mappings": [
            {
              "source_field": "name",
              "target_field": "employee_name",
              "transform_type": "direct"
            },
            {
              "source_field": "email",
              "target_field": "contact_email",
              "transform_type": "direct"
            },
            {
              "source_field": "department",
              "target_field": "dept",
              "transform_type": "direct"
            }
          ]
        }
      },
      "logger_output": {
        "id": "logger_output",
        "name": "合并结果输出",
        "type": "logger",
        "plugin": "logger",
        "config": {
          "logging": {
            "level": "info",
            "prefix": "[合并结果]",
            "include_timestamp": true
          },
          "format": {
            "output_format": "json"
          }
        }
      }
    },
    "edges": [
      {
        "id": "edge1",
        "source": "user_data",
        "target": "user_filter",
        "source_port": "data",
        "target_port": "data"
      },
      {
        "id": "edge2",
        "source": "performance_data",
        "target": "performance_filter",
        "source_port": "data",
        "target_port": "data"
      },
      {
        "id": "edge3",
        "source": "user_filter",
        "target": "data_transform",
        "source_port": "filtered_data",
        "target_port": "data"
      },
      {
        "id": "edge4",
        "source": "data_transform",
        "target": "logger_output",
        "source_port": "transformed_data",
        "target_port": "input_data"
      }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 201
  - 响应体: 包含创建的工作流信息

#### TC-CWF-005: 数据管道处理工作流

- **测试 ID**: TC-CWF-005
- **描述**: 创建完整的数据管道：过滤 -> 转换 -> 聚合 -> 输出
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "完整数据管道工作流",
    "description": "测试完整的数据处理管道",
    "version": "1.0.0",
    "nodes": {
      "order_data": {
        "id": "order_data",
        "name": "订单数据源",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [
              {
                "order_id": "O001",
                "customer": "客户A",
                "product": "产品1",
                "quantity": 10,
                "price": 99.99,
                "status": "completed",
                "date": "2024-01-15"
              },
              {
                "order_id": "O002",
                "customer": "客户B",
                "product": "产品2",
                "quantity": 5,
                "price": 199.99,
                "status": "completed",
                "date": "2024-01-16"
              },
              {
                "order_id": "O003",
                "customer": "客户A",
                "product": "产品1",
                "quantity": 3,
                "price": 99.99,
                "status": "pending",
                "date": "2024-01-17"
              },
              {
                "order_id": "O004",
                "customer": "客户C",
                "product": "产品3",
                "quantity": 8,
                "price": 299.99,
                "status": "completed",
                "date": "2024-01-18"
              },
              {
                "order_id": "O005",
                "customer": "客户B",
                "product": "产品2",
                "quantity": 12,
                "price": 199.99,
                "status": "completed",
                "date": "2024-01-19"
              }
            ]
          }
        }
      },
      "status_filter": {
        "id": "status_filter",
        "name": "已完成订单过滤",
        "type": "data_filter",
        "plugin": "data_filter",
        "config": {
          "conditions": [
            {
              "field": "status",
              "operator": "equals",
              "value": "completed",
              "data_type": "string"
            }
          ]
        }
      },
      "order_transform": {
        "id": "order_transform",
        "name": "订单金额计算",
        "type": "data_transform",
        "plugin": "data_transform",
        "config": {
          "mappings": [
            {
              "source_field": "customer",
              "target_field": "customer_name",
              "transform_type": "direct"
            },
            {
              "source_field": "quantity",
              "target_field": "qty",
              "transform_type": "direct"
            },
            {
              "source_field": "price",
              "target_field": "unit_price",
              "transform_type": "direct"
            },
            {
              "source_field": "quantity",
              "target_field": "total_amount",
              "transform_type": "calculate",
              "transform_config": {
                "expression": "quantity * price"
              }
            }
          ]
        }
      },
      "customer_aggregate": {
        "id": "customer_aggregate",
        "name": "客户订单统计",
        "type": "data_aggregate",
        "plugin": "data_aggregate",
        "config": {
          "group_by": ["customer_name"],
          "aggregations": [
            {
              "field": "total_amount",
              "function": "sum",
              "alias": "customer_total"
            },
            {
              "field": "qty",
              "function": "sum",
              "alias": "total_quantity"
            },
            {
              "field": "order_id",
              "function": "count",
              "alias": "order_count"
            }
          ],
          "order_by": [
            {
              "field": "customer_total",
              "direction": "desc"
            }
          ]
        }
      },
      "summary_logger": {
        "id": "summary_logger",
        "name": "统计结果输出",
        "type": "logger",
        "plugin": "logger",
        "config": {
          "logging": {
            "level": "info",
            "prefix": "[客户统计]",
            "include_timestamp": true,
            "include_metadata": true
          },
          "format": {
            "output_format": "table",
            "show_data_type": true
          }
        }
      }
    },
    "edges": [
      {
        "id": "edge1",
        "source": "order_data",
        "target": "status_filter",
        "source_port": "data",
        "target_port": "data"
      },
      {
        "id": "edge2",
        "source": "status_filter",
        "target": "order_transform",
        "source_port": "filtered_data",
        "target_port": "data"
      },
      {
        "id": "edge3",
        "source": "order_transform",
        "target": "customer_aggregate",
        "source_port": "transformed_data",
        "target_port": "data"
      },
      {
        "id": "edge4",
        "source": "customer_aggregate",
        "target": "summary_logger",
        "source_port": "aggregated_data",
        "target_port": "input_data"
      }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 201
  - 响应体: 包含创建的工作流信息

### 3. 工作流执行测试

#### TC-CWF-006: 执行基础数据流工作流

- **测试 ID**: TC-CWF-006
- **描述**: 激活并执行基础数据流工作流
- **端点**: `/workflows/{workflow_id}/activate` 和 `/workflows/{workflow_id}/trigger`
- **方法**: POST
- **步骤**:
  1. 激活工作流
  2. 触发工作流执行
  3. 检查执行状态
- **预期响应**:
  - 激活状态码: 200
  - 触发状态码: 200
  - 执行状态: success

#### TC-CWF-007: 执行数据转换工作流

- **测试 ID**: TC-CWF-007
- **描述**: 激活并执行数据转换工作流
- **端点**: `/workflows/{workflow_id}/activate` 和 `/workflows/{workflow_id}/trigger`
- **方法**: POST
- **预期响应**:
  - 激活状态码: 200
  - 触发状态码: 200
  - 执行状态: success

#### TC-CWF-008: 执行数据聚合工作流

- **测试 ID**: TC-CWF-008
- **描述**: 激活并执行数据聚合工作流
- **端点**: `/workflows/{workflow_id}/activate` 和 `/workflows/{workflow_id}/trigger`
- **方法**: POST
- **预期响应**:
  - 激活状态码: 200
  - 触发状态码: 200
  - 执行状态: success

#### TC-CWF-009: 执行多数据源合并工作流

- **测试 ID**: TC-CWF-009
- **描述**: 激活并执行多数据源合并工作流
- **端点**: `/workflows/{workflow_id}/activate` 和 `/workflows/{workflow_id}/trigger`
- **方法**: POST
- **预期响应**:
  - 激活状态码: 200
  - 触发状态码: 200
  - 执行状态: success

#### TC-CWF-010: 执行完整数据管道工作流

- **测试 ID**: TC-CWF-010
- **描述**: 激活并执行完整数据管道工作流
- **端点**: `/workflows/{workflow_id}/activate` 和 `/workflows/{workflow_id}/trigger`
- **方法**: POST
- **预期响应**:
  - 激活状态码: 200
  - 触发状态码: 200
  - 执行状态: success

### 4. 工作流状态和执行监控测试

#### TC-CWF-011: 监控工作流执行进度

- **测试 ID**: TC-CWF-011
- **描述**: 监控工作流的执行进度和状态变化
- **端点**: `/executions/{execution_id}/progress`
- **方法**: GET
- **预期响应**:
  - 状态码: 200
  - 响应体: 包含执行进度信息

#### TC-CWF-012: 获取工作流执行结果

- **测试 ID**: TC-CWF-012
- **描述**: 获取工作流执行的详细结果
- **端点**: `/executions/{execution_id}`
- **方法**: GET
- **预期响应**:
  - 状态码: 200
  - 响应体: 包含完整的执行结果和日志

#### TC-CWF-013: 获取工作流统计信息

- **测试 ID**: TC-CWF-013
- **描述**: 获取工作流的执行统计信息
- **端点**: `/workflows/{workflow_id}/statistics`
- **方法**: GET
- **预期响应**:
  - 状态码: 200
  - 响应体: 包含执行统计数据

### 5. 错误处理和边界测试

#### TC-CWF-014: 无效节点配置工作流

- **测试 ID**: TC-CWF-014
- **描述**: 创建包含无效配置的工作流，测试错误处理
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "无效配置工作流",
    "description": "测试无效节点配置的错误处理",
    "version": "1.0.0",
    "nodes": {
      "invalid_static": {
        "id": "invalid_static",
        "name": "无效静态数据",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "invalid_source"
          }
        }
      },
      "logger_output": {
        "id": "logger_output",
        "name": "错误输出",
        "type": "logger",
        "plugin": "logger",
        "config": {}
      }
    },
    "edges": [
      {
        "id": "edge1",
        "source": "invalid_static",
        "target": "logger_output",
        "source_port": "data",
        "target_port": "input_data"
      }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 400
  - 响应体: 包含配置验证错误信息

#### TC-CWF-015: 循环依赖工作流

- **测试 ID**: TC-CWF-015
- **描述**: 创建包含循环依赖的工作流，测试 DAG 验证
- **端点**: `/workflows`
- **方法**: POST
- **请求数据**:
  ```json
  {
    "name": "循环依赖工作流",
    "description": "测试循环依赖检测",
    "version": "1.0.0",
    "nodes": {
      "node_a": {
        "id": "node_a",
        "name": "节点A",
        "type": "static_data",
        "plugin": "static_data",
        "config": {
          "data": {
            "source": "manual",
            "records": [{ "id": 1, "name": "test" }]
          }
        }
      },
      "node_b": {
        "id": "node_b",
        "name": "节点B",
        "type": "data_filter",
        "plugin": "data_filter",
        "config": {
          "conditions": [{ "field": "id", "operator": "equals", "value": "1" }]
        }
      },
      "node_c": {
        "id": "node_c",
        "name": "节点C",
        "type": "logger",
        "plugin": "logger",
        "config": {
          "logging": { "level": "info" }
        }
      }
    },
    "edges": [
      { "id": "edge1", "source": "node_a", "target": "node_b" },
      { "id": "edge2", "source": "node_b", "target": "node_c" },
      { "id": "edge3", "source": "node_c", "target": "node_a" }
    ],
    "status": "inactive"
  }
  ```
- **预期响应**:
  - 状态码: 400
  - 响应体: 包含循环依赖错误信息

### 6. 清理测试

#### TC-CWF-016: 删除测试工作流

- **测试 ID**: TC-CWF-016
- **描述**: 删除所有创建的测试工作流
- **端点**: `/workflows/{workflow_id}`
- **方法**: DELETE
- **预期响应**:
  - 状态码: 200
  - 响应体: 包含删除成功信息

## 测试数据说明

- 所有测试用例使用静态数据，不依赖外部数据库
- 测试工作流涵盖了主要的数据处理节点类型
- 测试执行顺序：创建 -> 激活 -> 执行 -> 监控 -> 清理
- 错误测试验证系统的健壮性和错误处理能力

## 注意事项

1. 测试前确保服务已启动并正常运行
2. 节点插件系统需要正确注册所有测试节点类型
3. 工作流引擎需要支持 DAG 验证和执行调度
4. Logger 节点输出可以在服务日志中查看
5. 测试完成后及时清理创建的工作流资源
