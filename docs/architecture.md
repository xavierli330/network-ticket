# 网络工单系统 - 架构文档

## 整体架构

```
                          ┌──────────────────────────────────────────────────┐
                          │              网络工单平台                         │
                          │                                                    │
  ┌─────────┐  Webhook    │  ┌──────────┐   ┌──────────────┐   ┌───────────┐ │   HMAC+Nonce   ┌──────────┐
  │ Zabbix  │ ──────────► │  │          │   │              │   │           │ │ ─────────────► │          │
  └─────────┘             │  │  alert   │   │    ticket    │   │  client   │ │                │ 客户系统 │
  ┌──────────┐ Webhook    │  │  ingest  │   │    engine    │   │  adapter  │ │ ◄───────────── │          │
  │Prometheus│ ──────────► │  │          │   │              │   │           │ │    授权回调     └──────────┘
  └──────────┘             │  └──────────┘   └──────────────┘   └───────────┘ │
  ┌──────────┐ Poll        │       │               │                  │        │
  │  通用 JSON│ ──────────► │       │               │                  │        │
  └──────────┘             │       ▼               ▼                  ▼        │
                           │  ┌──────────────────────────────────────────┐    │
                           │  │              MySQL 8.0                   │    │
                           │  └──────────────────────────────────────────┘    │
                           │                                                    │
                           │  ┌──────────┐   ┌──────────┐                      │
                           │  │  admin   │   │  worker  │                      │
                           │  │   api    │   │  _pool   │                      │
                           │  └──────────┘   └──────────┘                      │
                           │       ▲                                            │
                           │       │                                            │
                           └───────┼────────────────────────────────────────────┘
                                   │
                            ┌──────┴──────┐
                            │   Nginx     │
                            │ (反向代理)   │
                            └──────┬──────┘
                                   │
                            ┌──────┴──────┐
                            │  Next.js    │
                            │  管理前端    │
                            └─────────────┘
```

## 模块说明

### alert_ingest — 告警接入模块

**位置**：`backend/internal/alert/parser/` + `backend/internal/service/alert_service.go`

**职责**：接收外部告警数据，解析、去重、创建工单。

**核心流程**：

1. 通过 Webhook（`POST /api/v1/alerts/webhook/:source_id`）或轮询接收原始 JSON
2. 根据告警源类型选择解析器（Zabbix / Prometheus / Generic）
3. 使用 gjson 从原始 JSON 中提取标题、描述、严重程度、源 IP、设备名称
4. 根据 `dedup_fields` 配置计算 SHA-256 指纹
5. 在去重窗口内查找相同指纹的已有工单，命中则追加告警记录，未命中则创建新工单

**解析器注册机制**：通过 `parser.Register()` 在 `init()` 阶段注册，支持扩展新的告警源类型。

### ticket_engine — 工单状态机

**位置**：`backend/internal/service/ticket_service.go` + `backend/internal/model/ticket.go`

**职责**：工单生命周期管理，状态转换校验，工作流节点追踪。

**两层状态模型**：

- **高层状态（ticket.status）**：面向业务的可观测状态 — pending / in_progress / completed / failed / cancelled / rejected
- **工作流节点（workflow_states）**：7 节点详细流程追踪 — alert_received → parsed → pushed → awaiting_auth → authorized → executing → completed

**状态机转换规则**：

```
pending    → in_progress, failed, cancelled
in_progress → completed, failed, rejected, cancelled
failed     → pending, cancelled
```

每次状态转换都记录 `ticket_logs` 表，包含 from_state、to_state、operator。

### client_adapter — 客户推送 + 回调

**位置**：`backend/internal/client/` + `backend/internal/handler/callback_handler.go` + `backend/internal/middleware/signature.go`

**职责**：向客户系统推送工单、接收客户授权回调、HMAC 签名验证。

**推送协议**：

- 请求头携带 X-Api-Key、X-Timestamp、X-Signature、X-Nonce
- 签名算法：`HMAC-SHA256(hmac_secret, timestamp + body)`
- 请求体包含 ticket_no、title、description、severity、alert_parsed、callback_url

**回调处理**：

- 客户调用 `POST /api/v1/callback/authorization`
- 签名验证中间件（`HMACSignature`）校验身份
- Nonce 中间件（`NonceCheck`）防重放
- 支持 `authorize` 和 `reject` 两种 action

### admin_api — 管理 API

**位置**：`backend/internal/handler/` + `backend/internal/middleware/auth.go`

**职责**：管理后台 REST API，JWT 认证与权限控制。

**认证中间件链**：

```
JWTAuth → RequireAdmin（仅客户管理接口）
```

**角色权限**：
- `admin`：全部接口
- `operator`：除客户增删改外的所有接口

### worker_pool — 异步推送

**位置**：`backend/internal/client/worker.go`

**职责**：管理异步工单推送任务，处理成功/失败、指数退避重试。

**工作方式**：

- 启动时创建 `pool_size` 个 goroutine（默认 10）
- 通过 channel 接收 `PushJob`
- 推送成功：更新工单状态为 `in_progress`，标记 `pushed` 节点为 `done`
- 推送失败：按指数退避调度重试（1s → 2s → 4s → 8s → 16s，上限 30s）
- 重试耗尽（默认 5 次）：标记工单为 `failed`

```
重试间隔：Backoff(attempt) = BaseInterval * 2^attempt，上限 MaxInterval
  attempt 0 → 1s
  attempt 1 → 2s
  attempt 2 → 4s
  attempt 3 → 8s
  attempt 4 → 16s（最后一次）
```

## 数据流

### 完整生命周期

```
告警接收       解析         去重          建单          推送客户       等待授权
  │            │            │             │              │             │
  ▼            ▼            ▼             ▼              ▼             ▼
┌──────┐  ┌────────┐  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│Webhook│  │Parser  │  │Finger-  │  │Create    │  │WorkerPool│  │Callback  │
│/Poll  │→ │Registry│→ │print +  │→ │Ticket +  │→ │→Push()  │→ │Handler   │
│      │  │        │  │Dedup    │  │Workflow  │  │  (retry) │  │          │
└──────┘  └────────┘  └─────────┘  └──────────┘  └──────────┘  └──────────┘
                                                                      │
                                                      ┌───────────────┘
                                                      │
                                              ┌───────┴────────┐
                                              │                 │
                                           authorize         reject
                                              │                 │
                                              ▼                 ▼
                                        in_progress        rejected
                                              │
                                              ▼
                                          completed
```

### 数据表关系

```
users                  ← 管理后台用户
  │
alert_sources          ← 告警源配置（类型、解析规则、去重规则）
  │
  ├── tickets          ← 工单主表（告警 → 工单 1:1，去重窗口内可能 N:1）
  │     │
  │     ├── workflow_states  ← 工单工作流节点（每工单 7 行）
  │     ├── ticket_logs      ← 工单状态变更日志
  │     └── alert_records    ← 追加的重复告警记录
  │
clients                ← 客户配置（推送地址、API Key、HMAC Secret）
  │
audit_logs             ← 审计日志（全局操作记录）
  │
nonce_records          ← Nonce 防重放记录（仅 backend=db 时使用）
```

## 安全机制

### HMAC-SHA256 签名流程

```
发送方（平台或客户）:
  1. 准备请求体 body
  2. 获取当前 Unix 时间戳 timestamp
  3. 生成随机 Nonce (UUID)
  4. 计算 signature = HMAC-SHA256(hmac_secret, timestamp + body)
  5. 发送请求，Header 携带 X-Api-Key, X-Timestamp, X-Signature, X-Nonce

接收方（平台中间件 HMACSignature）:
  1. 从 Header 提取 X-Api-Key → 查询数据库获取客户的 hmac_secret
  2. 从 Header 提取 X-Timestamp → 校验与当前时间差 <= 300s
  3. 读取请求体 body
  4. 计算 expected = HMAC-SHA256(hmac_secret, timestamp + body)
  5. 对比 expected 与 X-Signature → 不匹配则 401
```

### Nonce 防重放

```
中间件 NonceCheck:
  1. 从 Header 提取 X-Nonce
  2. 调用 Store.CheckAndSet(nonce, TTL=5min)
     - db 后端：INSERT 到 nonce_records 表（唯一约束），重复则返回 false
     - file 后端：写入本地文件并检查是否已存在
  3. 重复 → 返回 409 Conflict
```

### JWT 认证（管理后台）

```
登录流程:
  1. POST /api/v1/auth/login { username, password }
  2. 验证密码 → 生成 JWT Token（包含 user_id, username, role）
  3. 返回 Token

请求鉴权:
  1. 请求头 Authorization: Bearer <token>
  2. JWTAuth 中间件解析 Token → 注入 user_id, username, role 到 context
  3. RequireAdmin 中间件检查 role == "admin"（仅客户管理接口需要）
```

### 两级角色

| 角色     | 权限                         | 鉴权方式                        |
| -------- | ---------------------------- | ------------------------------- |
| admin    | 全部管理接口 + 客户增删改    | JWTAuth + RequireAdmin          |
| operator | 工单、告警源、审计日志       | JWTAuth                         |
| 外部客户 | 授权回调                     | HMACSignature + NonceCheck      |

## 扩展路径

### v1.5 — Dashboard + 超时通知

- **Dashboard**：工单统计面板（按状态/严重程度/客户的分布图表）
- **超时通知**：`timeout_at` 字段已预留，增加定时扫描逻辑，超时后触发通知
- **WebSocket 推送**：前端实时更新工单状态变化

### v2 — 动态流程引擎 + 多租户

- **workflow_definitions 表**：将硬编码的 7 节点流程改为数据库配置
- **独立流程引擎**：基于 workflow_definitions 驱动，支持自定义节点和分支
- **多租户隔离**：tenant_id 字段，数据行级隔离
- **RBAC 权限**：替代当前简单的 admin/operator 两级角色
- **告警升级**：超时未处理时自动升级严重程度或通知上级

### v3 — 告警关联分析

- **告警聚合**：基于拓扑关系关联同一故障源的多个告警
- **根因分析**：结合 CMDB 数据进行初步根因定位
- **SLA 管理**：工单响应/解决时间 SLA 跟踪与报表
- **开放 API**：完整的第三方集成 SDK（Go / Python / Java）
