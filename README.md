# AI Learning Platform

> 高并发 AI 多智能体学习资源生成平台
> Go + Gin + gRPC + Kafka + Redis + MySQL

## 项目结构

```
ai-learning-platform/
├── task-service/           # 对外 HTTP 服务（Gin）
│   ├── internal/
│   │   ├── handler/        # HTTP 处理层
│   │   ├── service/        # 业务逻辑层
│   │   ├── repository/     # 数据访问层
│   │   ├── middleware/     # 中间件（限流/JWT/链路追踪）
│   │   └── model/          # 数据模型
│   └── pkg/
│       └── response/       # 统一响应格式
│
├── scheduler-service/      # 任务调度（gRPC Server）
│   └── internal/
│       ├── worker/         # Goroutine Worker Pool ← 面试核心
│       ├── fsm/            # 任务状态机
│       └── lock/           # Redis 分布式锁 ← 面试核心
│
├── agent-service/          # AI 智能体（gRPC Server）
│   └── internal/
│       ├── orchestrator/   # 编排器（errgroup fan-out）← 面试核心
│       └── agents/         # Search / Generate / Eval
│
├── proto/                  # gRPC 接口定义
│   └── scheduler.proto
│
├── configs/
│   └── mysql/init.sql      # 数据库初始化
│
├── docker-compose.yml      # 一键启动基础设施
└── Makefile

```

## 快速开始

### 第一步：启动基础设施

```bash
make up
# 启动 MySQL / Redis / Kafka / Prometheus / Grafana / Jaeger
```

### 第二步：初始化依赖

```bash
make tidy
```

### 第三步：启动服务

```bash
# 三个终端分别运行
make run-task        # :8080
make run-scheduler   # :9001 (gRPC)
make run-agent       # :9002 (gRPC)
```

### 验证

```bash
# 健康检查
curl http://localhost:8080/health

# 创建任务（需要先登录拿 JWT）
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title":"学Go并发","goal":"掌握goroutine和channel","deadline_days":14}'
```

## 核心技术亮点（面试可讲）

| 模块 | 技术点 | 面试深度 |
|------|--------|---------|
| Worker Pool | Goroutine Pool + buffered channel | ⭐⭐⭐⭐⭐ |
| 分布式锁 | Redis SETNX + Lua + Watchdog | ⭐⭐⭐⭐⭐ |
| 限流 | 令牌桶 + Redis Lua 脚本 | ⭐⭐⭐⭐ |
| Agent 协同 | errgroup + fan-out/fan-in | ⭐⭐⭐⭐ |
| 状态机 | FSM + 并发安全 | ⭐⭐⭐ |
| gRPC | Protobuf + 流式接口 | ⭐⭐⭐ |

## 监控地址

- Grafana: http://localhost:3000 (admin/admin123)
- Prometheus: http://localhost:9090
- Jaeger: http://localhost:16686
