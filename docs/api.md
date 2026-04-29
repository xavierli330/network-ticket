# 网络工单系统 - API 文档

> Base URL: `/api/v1`
>
> 所有接口统一返回 JSON 格式。错误响应格式：`{"error": "错误描述"}`

---

## 目录

- [认证](#认证)
- [告警接入](#告警接入)
- [告警源管理](#告警源管理)
- [客户对接](#客户对接)
- [工单管理](#工单管理)
- [客户管理](#客户管理)
- [审计日志](#审计日志)
- [安全机制](#安全机制)

---

## 认证

### POST /api/v1/auth/login

用户登录，获取 JWT Token。

**认证方式**: 无需认证

**请求体**:

```json
{
  "username": "admin",
  "password": "admin123"
}
```

| 字段     | 类型   | 必填 | 说明     |
| -------- | ------ | ---- | -------- |
| username | string | 是   | 用户名   |
| password | string | 是   | 密码     |

**成功响应** (200):

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

| 字段           | 类型   | 说明                          |
| -------------- | ------ | ----------------------------- |
| token          | string | JWT Token，后续请求使用       |
| user.id        | int64  | 用户 ID                       |
| user.username  | string | 用户名                        |
| user.role      | string | 角色 (`admin` / `operator`)   |

**错误响应**:

| 状态码 | 说明                   |
| ------ | ---------------------- |
| 400    | 缺少 username 或 password |
| 401    | 用户名或密码错误       |

---

## 告警接入

### POST /api/v1/alerts/webhook/:source_id

Webhook 方式接收外部告警并自动创建/关联工单。系统根据告警源配置的解析器解析原始数据，通过指纹去重，自动生成工单并推送至关联客户。

**认证方式**: 无需认证（Webhook 入口，建议通过网络层面限制访问）

**路径参数**:

| 参数      | 类型  | 说明                           |
| --------- | ----- | ------------------------------ |
| source_id | int64 | 告警源 ID，对应 alert_sources 表 |

**请求体**: JSON 格式，结构取决于告警源配置的解析器（Zabbix / Prometheus / 通用等）。

**成功响应** (200):

```json
{
  "ticket_id": 42,
  "ticket_no": "NT20240101120000a1b2c3",
  "status": "pending",
  "is_new": true
}
```

| 字段      | 类型    | 说明                              |
| --------- | ------- | --------------------------------- |
| ticket_id | int64   | 工单 ID                           |
| ticket_no | string  | 工单编号                          |
| status    | string  | 工单当前状态                      |
| is_new    | boolean | 是否为新创建（false 表示关联已有工单） |

**错误响应**:

| 状态码 | 说明                      |
| ------ | ------------------------- |
| 400    | 无效的 source_id          |
| 500    | 告警解析或入库失败        |

---

### POST /api/v1/alerts

手动创建告警，用于测试或手动录入。

**认证方式**: JWT Bearer Token

**请求体**:

```json
{
  "source_type": "manual",
  "raw_data": {
    "host": "server-01",
    "alert": "CPU usage > 90%",
    "severity": "high"
  },
  "severity": "high",
  "title": "CPU 告警"
}
```

| 字段        | 类型   | 必填 | 说明                        |
| ----------- | ------ | ---- | --------------------------- |
| source_type | string | 是   | 告警源类型                  |
| raw_data    | object | 是   | 原始告警数据 (JSON)         |
| severity    | string | 否   | 严重程度                    |
| title       | string | 否   | 告警标题                    |

**成功响应** (201):

```json
{
  "ticket_id": 43,
  "ticket_no": "NT20240101120100d4e5f6",
  "status": "pending",
  "is_new": true
}
```

**错误响应**:

| 状态码 | 说明                        |
| ------ | --------------------------- |
| 400    | 缺少 source_type 或 raw_data |
| 401    | 未认证                      |
| 500    | 创建失败                    |

---

## 告警源管理

### GET /api/v1/alert-sources

获取所有告警源列表。

**认证方式**: JWT Bearer Token

**查询参数**: 无

**成功响应** (200):

```json
{
  "items": [
    {
      "id": 1,
      "name": "Zabbix 主监控",
      "type": "zabbix",
      "config": {},
      "parser_config": {},
      "webhook_secret": "wh_secret_xxx",
      "poll_endpoint": null,
      "poll_interval": 0,
      "dedup_fields": ["host", "trigger_name"],
      "dedup_window_sec": 300,
      "status": "active"
    }
  ]
}
```

---

### POST /api/v1/alert-sources

创建新的告警源。

**认证方式**: JWT Bearer Token

**请求体**:

```json
{
  "name": "Zabbix 主监控",
  "type": "zabbix",
  "config": {},
  "parser_config": {},
  "webhook_secret": "wh_secret_xxx",
  "poll_endpoint": null,
  "poll_interval": 0,
  "dedup_fields": ["host", "trigger_name"],
  "dedup_window_sec": 300,
  "status": "active"
}
```

| 字段              | 类型    | 必填 | 说明                          |
| ----------------- | ------- | ---- | ----------------------------- |
| name              | string  | 是   | 告警源名称                    |
| type              | string  | 是   | 告警源类型 (zabbix/prometheus/...) |
| config            | object  | 否   | 告警源连接配置 (JSON)         |
| parser_config     | object  | 否   | 解析器配置 (JSON)             |
| webhook_secret    | string  | 否   | Webhook 验证密钥              |
| poll_endpoint     | string  | 否   | 轮询端点 URL                  |
| poll_interval     | int     | 否   | 轮询间隔 (秒)                 |
| dedup_fields      | array   | 否   | 去重字段配置，JSON 数组，如 `["host", "alertname"]` |
| dedup_window_sec  | int     | 否   | 去重时间窗口 (秒)             |
| status            | string  | 否   | 状态，默认 "active"           |

**成功响应** (201): 返回创建的告警源对象。

**错误响应**:

| 状态码 | 说明                    |
| ------ | ----------------------- |
| 400    | 缺少 name 或 type       |
| 401    | 未认证                  |
| 500    | 创建失败                |

---

### PUT /api/v1/alert-sources/:id

更新告警源。

**认证方式**: JWT Bearer Token

**路径参数**:

| 参数 | 类型  | 说明     |
| ---- | ----- | -------- |
| id   | int64 | 告警源 ID |

**请求体**: 与创建相同。

**成功响应** (200):

```json
{
  "message": "updated"
}
```

---

### DELETE /api/v1/alert-sources/:id

删除告警源。

**认证方式**: JWT Bearer Token

**路径参数**:

| 参数 | 类型  | 说明     |
| ---- | ----- | -------- |
| id   | int64 | 告警源 ID |

**成功响应** (200):

```json
{
  "message": "deleted"
}
```

---

## 客户对接

### POST /api/v1/callback/authorization

授权回调接口。客户系统收到工单推送后，调用此接口对工单进行授权或拒绝。

**认证方式**: HMAC-SHA256 签名（详见 [安全机制](#安全机制)）

**请求头**:

| 请求头       | 说明                  |
| ------------ | --------------------- |
| X-Api-Key    | 客户 API Key          |
| X-Timestamp  | Unix 时间戳（秒）     |
| X-Signature  | HMAC-SHA256 签名      |
| X-Nonce      | 随机唯一标识（防重放）|

**请求体**:

```json
{
  "ticket_no": "NT20240101120000a1b2c3",
  "action": "authorize",
  "operator": "张三",
  "comment": "已确认，开始处理",
  "authorized_at": "2024-01-01T12:05:00Z"
}
```

| 字段          | 类型   | 必填 | 说明                                  |
| ------------- | ------ | ---- | ------------------------------------- |
| ticket_no     | string | 是   | 工单编号                              |
| action        | string | 是   | 操作：`authorize`（授权）或 `reject`（拒绝） |
| operator      | string | 否   | 操作人                                |
| comment       | string | 否   | 备注                                  |
| authorized_at | string | 否   | 授权时间 (ISO 8601)                   |

**成功响应** (200):

```json
{
  "code": 0,
  "message": "ok",
  "ticket_no": "NT20240101120000a1b2c3",
  "status": "authorized"
}
```

| 字段      | 类型   | 说明                                    |
| --------- | ------ | --------------------------------------- |
| code      | int    | 0 表示成功                              |
| message   | string | 响应消息                                |
| ticket_no | string | 工单编号                                |
| status    | string | 操作后状态 (`authorized` / `rejected`)  |

**错误响应**:

| 状态码 | 说明                                |
| ------ | ----------------------------------- |
| 400    | 请求体无效或缺少必要字段            |
| 401    | HMAC 签名验证失败                   |
| 403    | Nonce 重复（请求重放）或工单不属于该客户 |
| 404    | 工单不存在                          |

---

## 工单管理

### GET /api/v1/tickets

获取工单列表，支持分页和多条件筛选。

**认证方式**: JWT Bearer Token

**查询参数**:

| 参数      | 类型   | 必填 | 说明                          |
| --------- | ------ | ---- | ----------------------------- |
| page      | int    | 否   | 页码，默认 1                  |
| page_size | int    | 否   | 每页数量，默认 20             |
| status    | string | 否   | 按状态筛选                    |
| client_id | int64  | 否   | 按客户 ID 筛选                |
| severity  | string | 否   | 按严重程度筛选                |
| keyword   | string | 否   | 关键词搜索                    |

**工单状态值**: `pending` / `in_progress` / `completed` / `failed` / `cancelled` / `rejected`

> ⚠️ **已知问题**：当前版本工单创建后不会自动推送给客户，工单状态会停留在 `pending`。详见 [`docs/known-issues.md`](known-issues.md)。

**成功响应** (200):

```json
{
  "items": [
    {
      "id": 42,
      "ticket_no": "NT20240101120000a1b2c3",
      "alert_source_id": 1,
      "source_type": "zabbix",
      "alert_raw": {},
      "alert_parsed": {},
      "title": "CPU 告警",
      "description": "server-01 CPU 超过 90%",
      "severity": "high",
      "status": "pending",
      "client_id": 1,
      "external_id": null,
      "callback_data": null,
      "fingerprint": "abc123def456",
      "timeout_at": "2024-01-01T13:00:00Z",
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

---

### GET /api/v1/tickets/:id

获取单个工单详情，包含工作流状态历史。

**认证方式**: JWT Bearer Token

**路径参数**:

| 参数 | 类型  | 说明   |
| ---- | ----- | ------ |
| id   | int64 | 工单 ID |

**成功响应** (200):

```json
{
  "ticket": {
    "id": 42,
    "ticket_no": "NT20240101120000a1b2c3",
    "alert_source_id": 1,
    "source_type": "zabbix",
    "alert_raw": {},
    "alert_parsed": {},
    "title": "CPU 告警",
    "description": "server-01 CPU 超过 90%",
    "severity": "high",
    "status": "in_progress",
    "client_id": 1,
    "external_id": "ext-123",
    "callback_data": {},
    "fingerprint": "abc123def456",
    "timeout_at": "2024-01-01T13:00:00Z",
    "created_at": "2024-01-01T12:00:00Z",
    "updated_at": "2024-01-01T12:05:00Z"
  },
  "workflow_states": [
    {
      "id": 1,
      "ticket_id": 42,
      "from_status": "pending",
      "to_status": "in_progress",
      "operator": "client:张三",
      "created_at": "2024-01-01T12:05:00Z"
    }
  ]
}
```

**错误响应**:

| 状态码 | 说明         |
| ------ | ------------ |
| 400    | 无效的 ID    |
| 404    | 工单不存在   |

---

### PUT /api/v1/tickets/:id

手动更新工单状态（触发状态机转换）。

**认证方式**: JWT Bearer Token

**路径参数**:

| 参数 | 类型  | 说明   |
| ---- | ----- | ------ |
| id   | int64 | 工单 ID |

**请求体**:

```json
{
  "status": "completed",
  "operator": "admin"
}
```

| 字段     | 类型   | 必填 | 说明                    |
| -------- | ------ | ---- | ----------------------- |
| status   | string | 是   | 目标状态                |
| operator | string | 否   | 操作人，默认 "manual"   |

**成功响应** (200):

```json
{
  "message": "status updated"
}
```

**错误响应**:

| 状态码 | 说明                           |
| ------ | ------------------------------ |
| 400    | 无效 ID、缺少 status 或状态转换不合法 |
| 401    | 未认证                         |

---

### POST /api/v1/tickets/:id/retry

将失败的工单重新加入队列进行重试。将工单状态重置为 `pending`。

**认证方式**: JWT Bearer Token

**路径参数**:

| 参数 | 类型  | 说明   |
| ---- | ----- | ------ |
| id   | int64 | 工单 ID |

**请求体**: 无

**成功响应** (200):

```json
{
  "message": "ticket queued for retry"
}
```

**错误响应**:

| 状态码 | 说明                      |
| ------ | ------------------------- |
| 400    | 无效 ID 或状态转换不合法  |

---

### POST /api/v1/tickets/:id/cancel

取消工单，将状态置为 `cancelled`。

**认证方式**: JWT Bearer Token

**路径参数**:

| 参数 | 类型  | 说明   |
| ---- | ----- | ------ |
| id   | int64 | 工单 ID |

**请求体**: 无

**成功响应** (200):

```json
{
  "message": "ticket cancelled"
}
```

**错误响应**:

| 状态码 | 说明                      |
| ------ | ------------------------- |
| 400    | 无效 ID 或状态转换不合法  |

---

## 客户管理

### GET /api/v1/clients

获取所有客户列表。

**认证方式**: JWT Bearer Token

**成功响应** (200):

```json
{
  "items": [
    {
      "id": 1,
      "name": "客户A",
      "api_endpoint": "https://client-a.example.com/api/tickets",
      "api_key": "ak_xxx",
      "hmac_secret": "hs_xxx",
      "callback_url": "https://client-a.example.com/callback",
      "config": {},
      "status": "active"
    }
  ]
}
```

---

### POST /api/v1/clients

创建新客户。**仅管理员可用。**

**认证方式**: JWT Bearer Token（需 admin 角色）

**请求体**:

```json
{
  "name": "客户A",
  "api_endpoint": "https://client-a.example.com/api/tickets",
  "api_key": "ak_xxx",
  "hmac_secret": "hs_xxx",
  "callback_url": "https://client-a.example.com/callback",
  "config": "{}",
  "status": "active"
}
```

| 字段         | 类型   | 必填 | 说明                       |
| ------------ | ------ | ---- | -------------------------- |
| name         | string | 是   | 客户名称                   |
| api_endpoint | string | 是   | 客户接收推送的 API 端点    |
| api_key      | string | 是   | 客户 API Key               |
| hmac_secret  | string | 是   | HMAC 签名密钥              |
| callback_url | string | 否   | 授权回调 URL               |
| config       | string | 否   | 扩展配置 (JSON 字符串)     |
| status       | string | 否   | 状态，默认 "active"        |

**成功响应** (201): 返回创建的客户对象。

**错误响应**:

| 状态码 | 说明                                     |
| ------ | ---------------------------------------- |
| 400    | 缺少必填字段                             |
| 401    | 未认证                                   |
| 403    | 非 admin 角色                            |

---

### PUT /api/v1/clients/:id

更新客户信息。**仅管理员可用。**

**认证方式**: JWT Bearer Token（需 admin 角色）

**路径参数**:

| 参数 | 类型  | 说明   |
| ---- | ----- | ------ |
| id   | int64 | 客户 ID |

**请求体**: 与创建相同。

**成功响应** (200):

```json
{
  "message": "updated"
}
```

---

### DELETE /api/v1/clients/:id

删除客户。**仅管理员可用。**

**认证方式**: JWT Bearer Token（需 admin 角色）

**路径参数**:

| 参数 | 类型  | 说明   |
| ---- | ----- | ------ |
| id   | int64 | 客户 ID |

**成功响应** (200):

```json
{
  "message": "deleted"
}
```

---

## 审计日志

### GET /api/v1/audit-logs

获取审计日志列表，支持分页。

**认证方式**: JWT Bearer Token

**查询参数**:

| 参数      | 类型 | 必填 | 说明              |
| --------- | ---- | ---- | ----------------- |
| page      | int  | 否   | 页码，默认 1      |
| page_size | int  | 否   | 每页数量，默认 20 |

**成功响应** (200):

```json
{
  "items": [
    {
      "id": 1,
      "action": "ticket_created",
      "resource_type": "ticket",
      "resource_id": 42,
      "operator": "system",
      "detail": "Ticket NT20240101120000a1b2c3 created",
      "created_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "page_size": 20
}
```

---

## 安全机制

### HMAC-SHA256 签名验证

客户端调用回调接口 (`POST /api/v1/callback/authorization`) 时，需要使用 HMAC-SHA256 对请求进行签名。

#### 签名流程

1. **构造签名内容**: 将 `timestamp` 和 `request body` 拼接
2. **计算签名**: 使用 HMAC-SHA256 算法，以客户的 `hmac_secret` 为密钥
3. **发送请求**: 在请求头中携带签名相关信息

#### 请求头

| 请求头      | 说明                                     |
| ----------- | ---------------------------------------- |
| X-Api-Key   | 客户的 API Key，用于系统识别客户身份     |
| X-Timestamp | 当前 Unix 时间戳（秒）                   |
| X-Signature | HMAC-SHA256 签名值                       |
| X-Nonce     | 随机唯一字符串，防止请求重放             |

#### 签名示例 (Python)

```python
import hmac
import hashlib
import time
import uuid
import json
import requests

api_key = "ak_xxx"
hmac_secret = "hs_xxx"

body = json.dumps({
    "ticket_no": "NT20240101120000a1b2c3",
    "action": "authorize",
    "operator": "张三",
    "comment": "已确认",
    "authorized_at": "2024-01-01T12:05:00Z"
})

timestamp = str(int(time.time()))
nonce = str(uuid.uuid4())

# 签名内容 = timestamp + body
message = timestamp.encode() + body.encode()
signature = hmac.new(
    hmac_secret.encode(),
    message,
    hashlib.sha256
).hexdigest()

response = requests.post(
    "http://your-server/api/v1/callback/authorization",
    data=body,
    headers={
        "Content-Type": "application/json",
        "X-Api-Key": api_key,
        "X-Timestamp": timestamp,
        "X-Signature": signature,
        "X-Nonce": nonce,
    }
)
```

#### 验证规则

| 规则                  | 说明                                                    |
| --------------------- | ------------------------------------------------------- |
| Timestamp 窗口        | 请求时间戳与服务器时间差不得超过 300 秒（5 分钟）       |
| Nonce 唯一性          | 每个 Nonce 值在 TTL（5 分钟）内只能使用一次             |
| API Key 识别          | 通过 X-Api-Key 查找客户及其 HMAC Secret                 |
| 签名算法              | HMAC-SHA256(timestamp + body, hmac_secret)              |

### JWT Token 认证

管理端 API 使用 JWT Bearer Token 认证。

#### 使用方式

在请求头中携带 Token：

```
Authorization: Bearer <token>
```

#### Token 获取

通过 `POST /api/v1/auth/login` 登录获取。

#### Token 有效期

默认 24 小时（可在 `config.yaml` 的 `jwt.expire_hours` 配置）。

#### 权限控制

| 角色     | 权限                                    |
| -------- | --------------------------------------- |
| admin    | 所有接口，包括客户的增删改              |
| operator | 除客户增删改外的所有管理接口            |

### 工单状态机

```
                 ┌──────────┐
                 │ pending  │ ← (retry)
                 └────┬─────┘
                      │
              ┌───────┼────────┐
              ▼       ▼        ▼
      ┌──────────┐ ┌──────────┐ ┌──────────┐
      │in_progress│ │ rejected │ │cancelled │
      └────┬─────┘ └──────────┘ └──────────┘
           │
     ┌─────┼──────┐
     ▼            ▼
┌──────────┐ ┌──────────┐
│completed │ │  failed  │ → (retry → pending)
└──────────┘ └──────────┘
```

**合法的状态转换**:

| 从           | 到             | 触发方式               |
| ------------ | -------------- | ---------------------- |
| pending      | in_progress    | 客户授权回调           |
| pending      | rejected       | 客户拒绝回调           |
| pending      | cancelled      | 管理员手动取消         |
| in_progress  | completed      | 管理员手动完成         |
| in_progress  | failed         | 推送失败 / 系统标记    |
| failed       | pending        | 管理员手动重试         |
