# 网络工单平台设计文档

> 日期: 2026-04-28
> 状态: 已确认

---

## 1. 项目背景

构建一个网络/服务器告警工单平台，核心流程：

1. 接收告警（webhook/轮询） → 解析并生成工单
2. 通过 API 将工单推送给客户系统（HMAC 签名 + 防重放）
3. 客户在其系统内授权后回调通知
4. 收到授权后触发外部执行动作（执行动作不在本平台范围内）

**约束与决策：**
- 少量客户，渐进对接，每个客户 API 协议可能不同
- 告警类型多样，字段差异大
- 工单流程先简单（线性状态机），架构预留扩展到独立流程引擎
- 前端最小起步，后续迭代加 Dashboard/日志追踪/数据分析
- 部署环境未定，要求灵活（支持私有部署和云部署）

---

## 2. 架构设计

### 2.1 方案选择：单体模块化

选择理由：少量客户场景，部署简单，开发快速，模块边界清晰，未来可按需拆分。

```
┌─────────────────────────────────────────────────────────┐
│                   Go Backend (Single Binary)              │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │  Alert    │  │  Ticket  │  │  Client  │  │  Admin  │ │
│  │  Ingest   │  │  Engine  │  │  Adapter │  │  API    │ │
│  │  Module   │  │  Module  │  │  Module  │  │  Module │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘ │
│       │             │             │              │       │
│  ┌────┴─────────────┴─────────────┴──────────────┴────┐ │
│  │              PostgreSQL (单库多表)                    │ │
│  └────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
        │                          │
   Webhook/Poll              API (HMAC签名)
   ┌────────┐                ┌────────────┐
   │ 监控平台 │                │ 客户系统 ×N │
   └────────┘                └────────────┘
```

### 2.2 端到端数据流

```
  监控平台                    工单平台                      客户系统
     │                          │                            │
     │  ① 告警 (webhook/poll)   │                            │
     ├─────────────────────────►│                            │
     │                          │ ② 解析告警, 生成工单        │
     │                          │    状态: CREATED            │
     │                          │                            │
     │                          │ ③ 推送工单 (HMAC签名)       │
     │                          ├───────────────────────────►│
     │                          │    状态: PUSHED             │
     │                          │                            │
     │                          │ ④ 授权回调 (验签+防重放)    │
     │                          │◄───────────────────────────┤
     │                          │    状态: AUTHORIZED         │
     │                          │                            │
     │                          │ ⑤ 触发执行动作 (回调通知)   │
     │                          │──────► 外部执行系统         │
     │                          │    状态: EXECUTING          │
     │                          │                            │
     │                          │ ⑥ 执行完成回调              │
     │                          │◄────── 外部执行系统         │
     │                          │    状态: COMPLETED          │
```

---

## 3. 数据模型

### 3.1 核心表

**tickets（工单主表）— 只维护高层可观测状态**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | 主键 |
| ticket_no | varchar | 业务编号 (如 TK-20260428-0001) |
| alert_source_id | int FK | 告警源 |
| source_type | varchar | 告警源类型 |
| alert_raw | jsonb | 原始告警 JSON |
| alert_parsed | jsonb | 解析后结构化数据 |
| title | varchar | 工单标题 |
| description | text | 工单描述 |
| severity | varchar | 严重级别: critical/warning/info |
| **status** | varchar | **高层状态: pending / in_progress / completed / failed / cancelled** |
| client_id | int FK | 客户 |
| external_id | varchar | 客户侧工单 ID |
| callback_data | jsonb | 回调数据 |
| timeout_at | timestamp | 超时时间 |
| created_at | timestamp | |
| updated_at | timestamp | |

**workflow_states（详细流程状态）— 每次流转一条记录**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | 主键 |
| ticket_id | int FK | 关联工单 |
| node_name | varchar | 流程节点: alert_received / parsed / pushed / awaiting_auth / authorized / executing / completed |
| status | varchar | 节点状态: pending / active / done / failed / skipped / timeout |
| operator | varchar | 操作方: system / client:xxx |
| input_data | jsonb | 节点输入 |
| output_data | jsonb | 节点输出 |
| error_message | text | 错误信息 |
| started_at | timestamp | |
| completed_at | timestamp | |
| created_at | timestamp | |

**alert_sources（告警源配置）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | 主键 |
| name | varchar | 告警源名称 |
| type | varchar | 类型: zabbix / prometheus / generic |
| config | jsonb | 告警源连接配置 |
| parser_config | jsonb | 字段映射配置 (JSONPath) |
| webhook_secret | varchar | Webhook 验签密钥 |
| poll_endpoint | varchar | 轮询地址 |
| poll_interval | int | 轮询间隔 (秒) |
| status | varchar | 启用状态 |
| created_at | timestamp | |
| updated_at | timestamp | |

**clients（客户对接配置）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | 主键 |
| name | varchar | 客户名称 |
| api_endpoint | varchar | 推送地址 |
| api_key | varchar | API Key (加密存储) |
| hmac_secret | varchar | HMAC 密钥 (加密存储) |
| callback_url | varchar | 客户回调地址 |
| config | jsonb | 扩展配置 |
| status | varchar | 启用状态 |
| created_at | timestamp | |

**ticket_logs（工单状态变更日志）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | 主键 |
| ticket_id | int FK | 关联工单 |
| action | varchar | 操作类型 |
| from_state | varchar | 变更前状态 |
| to_state | varchar | 变更后状态 |
| operator | varchar | 操作人 |
| detail | jsonb | 详情 |
| created_at | timestamp | |

**audit_logs（审计日志）**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | 主键 |
| actor | varchar | 操作人 |
| action | varchar | 操作 |
| resource_type | varchar | 资源类型 |
| resource_id | int | 资源 ID |
| detail | jsonb | 详情 |
| ip_address | varchar | IP |
| created_at | timestamp | |

### 3.2 两层状态的关系

```
ticket.status = "pending" (高层: 待处理)
  └── workflow_states:
        ├── alert_received  status=done     ← 告警已接收
        ├── parsed          status=done     ← 解析完成
        └── pushed          status=active   ← 推送中...

ticket.status = "in_progress" (高层: 进行中)
  └── workflow_states:
        ├── alert_received  status=done
        ├── parsed          status=done
        ├── pushed          status=done
        ├── awaiting_auth   status=active   ← 等待客户授权
        └── authorized      status=pending  ← 待触发

ticket.status = "completed" (高层: 已完成)
  └── workflow_states:
        ├── alert_received  status=done
        ├── parsed          status=done
        ├── pushed          status=done
        ├── awaiting_auth   status=done
        ├── authorized      status=done
        ├── executing       status=done
        └── completed       status=done
```

### 3.3 流程定义（v2 可选扩展）

workflow_definitions 表作为未来流程编排预留，当前版本节点硬编码在代码中。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | serial PK | |
| name | varchar | 流程名称 |
| alert_source_type | varchar | 绑定告警源类型 |
| nodes | jsonb | 节点定义及流转关系 |
| version | int | 版本号 |
| status | varchar | 启用状态 |
| created_at | timestamp | |

### 3.4 关键设计决策

- 告警原始数据(alert_raw)和解析后数据(alert_parsed)分开存储，解耦存储和解析，方便回溯重解析
- 客户配置用 JSONB 灵活存储差异部分，无需为每个客户建表
- ticket_logs 独立记录状态变更，用于审计追踪，不污染工单主表
- workflow_states 详细流程状态独立于 tickets 主表高层状态，未来可拆到独立流程引擎

---

## 4. API 接口设计

### 4.1 告警接入 API

```
# Webhook 接收告警 (被动)
POST /api/v1/alerts/webhook/:source_id
  Headers: X-Signature, X-Timestamp
  Body: 原始告警 JSON
  Response: { ticket_id, ticket_no, status }

# 手动创建告警 (测试/补偿)
POST /api/v1/alerts
  Body: { source_type, raw_data, severity?, title? }
  Response: { ticket_id, ticket_no, status }

# 告警源管理
GET    /api/v1/alert-sources
POST   /api/v1/alert-sources
PUT    /api/v1/alert-sources/:id
DELETE /api/v1/alert-sources/:id
```

### 4.2 客户对接 API（核心）

**出站 — 推送工单给客户：**

```
POST {client.api_endpoint}
  Headers:
    X-Api-Key: {client.api_key}
    X-Timestamp: {unix_seconds}
    X-Signature: HMAC-SHA256(secret, timestamp + body)
    X-Nonce: {uuid}
  Body: {
    ticket_no, title, description,
    severity, alert_parsed,
    callback_url
  }
```

**入站 — 客户授权回调：**

```
POST /api/v1/callback/authorization
  Headers:
    X-Api-Key: {client.api_key}
    X-Timestamp: {unix_seconds}
    X-Signature: HMAC-SHA256(secret, timestamp + body)
    X-Nonce: {uuid}
  Body: {
    ticket_no,
    action: "authorize" | "reject",
    operator,
    comment?,
    authorized_at
  }
  Response: { code: 0, message: "ok", ticket_no, status }
```

### 4.3 管理后台 API

```
# 工单管理
GET    /api/v1/tickets
GET    /api/v1/tickets/:id
PUT    /api/v1/tickets/:id
POST   /api/v1/tickets/:id/retry
POST   /api/v1/tickets/:id/cancel

# 客户管理
GET    /api/v1/clients
POST   /api/v1/clients
PUT    /api/v1/clients/:id
DELETE /api/v1/clients/:id

# Dashboard (v2)
GET    /api/v1/stats/overview
GET    /api/v1/stats/trends
GET    /api/v1/stats/clients

# 审计日志
GET    /api/v1/audit-logs
```

### 4.4 安全机制

| 机制 | 说明 |
|------|------|
| HMAC-SHA256 签名 | timestamp + body 作为输入，secret 由双方保管 |
| Nonce 防重放 | 每次请求唯一 nonce，服务端窗口内去重 |
| Timestamp 时间窗口 | 偏差超过 ±5 分钟拒绝 |
| API Key 身份识别 | 识别客户身份，关联 secret 验签 |

### 4.5 Nonce 存储设计

可插拔后端，通过配置切换：

```go
type NonceStore interface {
    CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error)
    Clean(ctx context.Context) error
}
```

| 后端 | 实现 | 适用场景 |
|------|------|----------|
| db (默认) | PostgreSQL 表，INSERT ON CONFLICT DO NOTHING | 已有数据库，多实例部署 |
| file | 追加写入文件，grep 检查 + 重写清理 | 单实例部署，零依赖 |
| redis | SET nonce 1 EX ttl NX | 高并发，多实例 |

配置方式：

```yaml
security:
  nonce:
    backend: "db"    # file | db | redis
    ttl: 5m
```

---

## 5. 告警解析器设计

### 5.1 接口定义

```go
type AlertParser interface {
    Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error)
    SourceType() string
}

type ParsedAlert struct {
    Title       string                 `json:"title"`
    Description string                 `json:"description"`
    Severity    string                 `json:"severity"`
    SourceIP    string                 `json:"source_ip"`
    DeviceName  string                 `json:"device_name"`
    AlertTime   time.Time              `json:"alert_time"`
    Fields      map[string]interface{} `json:"fields"`
}
```

### 5.2 注册机制

```go
var parsers = map[string]AlertParser{}

func Register(p AlertParser) {
    parsers[p.SourceType()] = p
}
```

### 5.3 内置解析器

- **ZabbixParser** — Zabbix Webhook 告警
- **PrometheusParser** — Prometheus Alertmanager
- **GenericJSONParser** — 配置化 JSONPath 字段映射，无需写代码

### 5.4 GenericJSONParser 配置

alert_sources.parser_config (JSONB) 示例：

```json
{
  "field_mapping": {
    "title": "$.alerts[0].labels.alertname",
    "description": "$.alerts[0].annotations.summary",
    "severity": "$.alerts[0].labels.severity",
    "source_ip": "$.alerts[0].labels.instance",
    "device_name": "$.alerts[0].labels.device",
    "alert_time": "$.alerts[0].startsAt"
  },
  "severity_mapping": {
    "critical": ["p1", "critical", "emergency"],
    "warning": ["p2", "warning", "warn"],
    "info": ["p3", "info", "informational"]
  }
}
```

### 5.5 告警接入流程

```
Webhook/Poll 接收原始数据
  → 1. 验签 (webhook_secret)
  → 2. 查找 Parser (按 alert_sources.type，找不到用 GenericJSONParser)
  → 3. 解析 (parser.Parse → ParsedAlert)
  → 4. 去重判断 (指纹: source_type + 关键字段 hash，窗口内更新已有工单)
  → 5. 创建工单 (ticket + workflow_states，异步触发客户推送)
```

---

## 6. 项目目录结构

```
network-ticket/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── config/          # Viper 配置
│   │   ├── model/           # 数据模型
│   │   ├── repository/      # sqlc 生成
│   │   ├── service/         # 业务逻辑
│   │   ├── handler/         # Gin HTTP Handler
│   │   ├── middleware/      # auth/signature/nonce/logger
│   │   ├── alert/parser/    # 告警解析器 (registry + 内置)
│   │   ├── alert/poller/    # 定时拉取
│   │   ├── client/          # 客户端推送 + 重试
│   │   ├── nonce/           # 防重放存储 (file/db/redis)
│   │   └── pkg/             # hmac / fingerprint
│   ├── migrations/          # golang-migrate
│   ├── config.yaml
│   ├── config.example.yaml
│   ├── go.mod
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── app/             # App Router
│   │   │   ├── tickets/     # 工单列表 + 详情
│   │   │   ├── clients/     # 客户管理
│   │   │   └── sources/     # 告警源管理
│   │   ├── components/
│   │   ├── lib/api.ts
│   │   └── types/
│   ├── package.json
│   └── next.config.js
├── docs/
│   ├── deployment.md
│   └── api.md
├── docker-compose.yaml
├── Makefile
└── README.md
```

---

## 7. 技术栈

| 层 | 选型 |
|----|------|
| 后端框架 | Gin |
| 数据库 | PostgreSQL 16 |
| 数据访问 | sqlx + sqlc |
| 迁移 | golang-migrate |
| 配置 | Viper |
| 日志 | zap |
| JSONPath | gjson |
| 前端框架 | Next.js 15 (App Router) |
| UI 组件 | shadcn/ui + Tailwind CSS |
| HTTP 客户端 | fetch + SWR |
| 图表 | recharts (v2) |
| API 文档 | swaggo/swag |
| 热重载 | air (Go) + next dev |
| 部署 | Docker + Docker Compose |
| 反向代理 | Nginx (可选) |

---

## 8. 扩展路径

- **v1.5**: Dashboard (趋势图/状态分布/按客户统计) + 超时自动升级/通知
- **v2**: workflow_definitions 动态流程配置 → 独立流程引擎
- **v2**: 多租户/权限体系
- **v3**: 告警关联分析 + 自动化工单聚合
