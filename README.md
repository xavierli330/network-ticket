# 网络工单平台 (Network Ticket Platform)

告警驱动的工单管理平台，支持多告警源接入、客户 API 对接、工单状态流转。

## 功能特性

- **多告警源接入** — Webhook / 轮询两种模式，内置 Zabbix、Prometheus、通用 JSON 解析器
- **可配置告警解析与去重** — JSONPath 字段映射、SHA-256 指纹去重、可配置时间窗口
- **工单状态管理** — 两层状态模型：高层可观测状态 + 7 节点 workflow_states 详细流程追踪
- **客户 API 对接** — HMAC-SHA256 签名、Nonce 防重放、5 次指数退避重试
- **管理后台** — 工单管理、客户管理、告警源管理、审计日志
- **Docker Compose 一键部署** — MySQL + Backend + Frontend + Nginx

## 技术栈

| 层级     | 技术                                        |
| -------- | ------------------------------------------- |
| 后端     | Go 1.23, Gin, sqlx, MySQL 8.0, Viper, zap  |
| 前端     | Next.js 16, React 19, Tailwind CSS          |
| 部署     | Docker Compose, Nginx                       |

## 快速开始

```bash
git clone <repo>
cd network-ticket
cp backend/config.example.yaml backend/config.yaml
docker compose up -d
```

启动完成后访问 http://localhost，默认管理员账号 `admin` / `admin123`。

> 首次部署后请立即修改默认密码，详见 [部署手册](docs/deployment.md)。

## 项目结构

```
network-ticket/
├── backend/                    # Go 后端
│   ├── cmd/server/             # 入口
│   ├── internal/
│   │   ├── alert/parser/       # 告警解析器 (zabbix/prometheus/generic)
│   │   ├── client/             # 客户推送 + Worker Pool
│   │   ├── config/             # Viper 配置加载
│   │   ├── handler/            # Gin HTTP Handler
│   │   ├── middleware/          # JWT / HMAC 签名 / Nonce 中间件
│   │   ├── model/              # 数据模型
│   │   ├── nonce/              # Nonce 防重放存储 (db/file)
│   │   ├── pkg/                # HMAC 签名、指纹计算、工单号生成
│   │   ├── repository/         # sqlx 数据访问层
│   │   └── service/            # 业务逻辑层
│   ├── migrations/             # 数据库迁移 (001-009)
│   ├── config.example.yaml     # 配置模板
│   └── Dockerfile
├── frontend/                   # Next.js 前端
│   ├── src/
│   │   ├── app/                # App Router 页面
│   │   │   ├── login/          # 登录页
│   │   │   ├── tickets/        # 工单列表 + 详情
│   │   │   ├── clients/        # 客户管理
│   │   │   ├── sources/        # 告警源管理
│   │   │   └── layout/         # 布局组件 (Header/Sidebar)
│   │   ├── components/         # 公共组件
│   │   ├── lib/                # API 请求封装
│   │   └── types/              # TypeScript 类型
│   └── Dockerfile
├── nginx/
│   └── nginx.conf              # 反向代理配置
├── docs/
│   ├── deployment.md           # 部署手册
│   ├── api.md                  # API 文档
│   ├── usage.md                # 使用指南
│   └── architecture.md         # 架构文档
├── docker-compose.yaml
└── Makefile
```

## 文档

| 文档             | 说明                       |
| ---------------- | -------------------------- |
| [部署手册](docs/deployment.md)   | 环境要求、配置说明、生产注意事项 |
| [API 文档](docs/api.md)         | 完整 REST API 参考         |
| [使用指南](docs/usage.md)       | 登录、告警源、客户、工单操作指南 |
| [架构文档](docs/architecture.md) | 系统架构、数据流、安全机制    |

## License

MIT
