# Flow Service

## 项目简介

Flow Service 是智慧园区平台的流程服务，负责提供流程编排、执行和管理功能。

## 主要功能

- 流程定义和管理
- 流程实例执行
- 任务调度和管理
- 流程监控和日志

## 技术架构

- **框架**: 基于 Dapr 的微服务架构
- **语言**: Go 1.23.1
- **路由**: Chi v5
- **文档**: Swagger/OpenAPI
- **监控**: Prometheus
- **容器**: Docker

## 项目结构

```
flow-service/
├── api/                 # API层，路由和控制器
│   └── routes.go       # 路由配置
├── service/            # 业务逻辑层
│   └── init.go        # 服务初始化
├── docs/               # API文档
│   └── docs.go        # Swagger文档
├── main.go             # 程序入口
├── go.mod              # Go模块配置
├── Dockerfile          # Docker构建文件
├── .gitignore          # Git忽略配置
└── README.md           # 项目说明
```

## 开发环境

### 前置要求

- Go 1.23.1+
- Docker (可选)
- Dapr CLI (用于本地开发)

### 快速开始

1. 克隆项目

```bash
cd flow-service
```

2. 初始化依赖

```bash
go mod tidy
```

3. 运行服务

```bash
go run main.go
```

4. 访问 API 文档

打开浏览器访问: http://localhost/swagger/index.html

## API 接口

### 健康检查

- **GET** `/health` - 服务健康检查

### 流程管理

- **GET** `/api/v1/flows` - 获取流程列表
- **POST** `/api/v1/flows` - 创建新流程
- **GET** `/api/v1/flows/{id}` - 获取流程详情
- **PUT** `/api/v1/flows/{id}` - 更新流程
- **DELETE** `/api/v1/flows/{id}` - 删除流程

### 流程实例

- **GET** `/api/v1/instances` - 获取实例列表
- **POST** `/api/v1/instances` - 启动流程实例
- **GET** `/api/v1/instances/{id}` - 获取实例详情

### 任务管理

- **GET** `/api/v1/tasks` - 获取任务列表
- **POST** `/api/v1/tasks/{id}/complete` - 完成任务

## 环境变量

| 变量名       | 描述         | 默认值 |
| ------------ | ------------ | ------ |
| LISTEN_PORT  | 服务监听端口 | 80     |
| BASE_CONTEXT | 服务基础路径 | ""     |

## Docker 部署

### 构建镜像

```bash
docker build -t flow-service .
```

### 运行容器

```bash
docker run -p 80:80 flow-service
```

## 开发规范

- 代码必须包含完整的中文注释
- 遵循 SOLID 原则
- 单个文件不超过 500 行
- 使用统一的错误处理机制

## 许可证

本项目采用 MIT 许可证。
# flow-service
