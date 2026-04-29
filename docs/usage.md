# 网络工单系统 - 使用指南

本文档面向平台管理员和操作员，介绍系统的各项功能使用方法。

---

## 目录

- [1. 登录系统](#1-登录系统)
- [2. 告警源管理](#2-告警源管理)
- [3. 客户管理](#3-客户管理)
- [4. 工单管理](#4-工单管理)
- [5. 客户 API 对接指南](#5-客户-api-对接指南)
- [6. 审计日志](#6-审计日志)

---

## 1. 登录系统

### 访问入口

浏览器打开 http://localhost/login 进入登录页面。

### 默认账号

| 角色   | 用户名 | 密码     | 权限范围                               |
| ------ | ------ | -------- | -------------------------------------- |
| 管理员 | admin  | admin123 | 全部功能，包括客户增删改、系统配置     |
| 操作员 | —      | —        | 工单查看/操作、告警源查看、审计日志查看 |

> 首次登录后请立即修改默认管理员密码。

### 角色权限对照

| 功能         | 管理员 (admin) | 操作员 (operator) |
| ------------ | :------------: | :---------------: |
| 工单列表     |        Y       |         Y         |
| 工单详情     |        Y       |         Y         |
| 工单重试/取消 |        Y       |         Y         |
| 告警源管理   |        Y       |         Y         |
| 客户管理     |        Y       |         N         |
| 审计日志     |        Y       |         Y         |

---

## 2. 告警源管理

告警源定义了外部监控系统的接入方式。系统支持 Webhook（主动推送）和轮询（被动拉取）两种模式。

### 2.1 创建告警源

进入「告警源管理」页面，点击「新建告警源」，填写以下信息：

| 字段              | 必填 | 说明                                       |
| ----------------- | ---- | ------------------------------------------ |
| 名称 (name)       | 是   | 告警源名称，如 "Zabbix 主监控"             |
| 类型 (type)       | 是   | `zabbix` / `prometheus` / `generic`        |
| Webhook Secret    | 否   | Webhook 验证密钥                           |
| 轮询端点          | 否   | 轮询模式的远程 URL                         |
| 轮询间隔          | 否   | 轮询间隔（秒）                             |
| 去重字段          | 否   | 用于指纹计算的 JSONPath 字段列表           |
| 去重时间窗口      | 否   | 去重窗口（秒），默认 300（5 分钟）         |

创建后会分配一个 `source_id`，Webhook URL 为：

```
POST http://your-server/api/v1/alerts/webhook/{source_id}
```

**完整配置示例**：

以下是一个 `generic` 类型告警源的完整配置（API 或数据库视角）：

```json
{
  "name": "内部监控系统",
  "type": "generic",
  "webhook_secret": "wh_secret_xxx",
  "poll_endpoint": null,
  "poll_interval": 0,
  "dedup_fields": ["host", "alertname"],
  "dedup_window_sec": 300,
  "status": "active"
}
```

各字段格式说明：

| 字段            | 格式示例                              | 说明                              |
| --------------- | ------------------------------------- | --------------------------------- |
| `type`          | `"zabbix"` / `"prometheus"` / `"generic"` | 一旦创建不建议修改类型            |
| `webhook_secret`| `"wh_secret_xxx"`                     | 任意字符串，建议 16 位以上随机值  |
| `dedup_fields`  | `["host", "alertname"]`               | JSON 数组，元素为 gjson 路径字符串 |
| `dedup_window_sec`| `300`                               | 整数，单位秒，范围建议 60~3600    |
| `poll_interval` | `60`                                  | 整数，单位秒，与 `poll_endpoint` 同时设置时生效 |
| `status`        | `"active"` / `"inactive"`             | `inactive` 时停止接收新告警       |

### 2.2 Zabbix Webhook 配置

在 Zabbix 管理端完成以下配置：

**步骤一：创建媒介类型**

1. 进入 Administration → Media types → Create media type
2. 类型选择 **Webhook**
3. URL 填入平台 Webhook 地址：
   ```
   http://your-server/api/v1/alerts/webhook/{source_id}
   ```
4. HTTP 方法选择 `POST`
5. 如配置了 Webhook Secret，添加 HTTP Header：`X-Webhook-Secret: <your_secret>`

**Zabbix Webhook 参数示例**（Parameters 配置）：

| 参数名       | 值示例                                     | 说明                  |
| ------------ | ------------------------------------------ | --------------------- |
| `URL`        | `http://your-server/api/v1/alerts/webhook/1`| Webhook 地址          |
| `to`         | `{ALERT.SENDTO}`                           | 收件人（可留空）      |
| `subject`    | `{EVENT.NAME}`                             | 告警主题              |
| `message`    | `{EVENT.OPDATA}`                           | 告警详情              |
| `event.id`   | `{EVENT.ID}`                               | 事件 ID               |
| `event.severity` | `{EVENT.SEVERITY}`                     | 严重等级              |
| `host.name`  | `{HOST.NAME}`                              | 主机名                |
| `host.ip`    | `{HOST.IP}`                                | 主机 IP               |

Zabbix 发送的 JSON 格式示例：

```json
{
  "subject": "CPU 使用率超过 90%",
  "message": "server-01 的 CPU 使用率已达 95%，持续 5 分钟",
  "event": {
    "id": "12345",
    "severity": "High"
  },
  "host": {
    "name": "server-01",
    "ip": "192.168.1.100"
  }
}
```

**步骤二：配置 Action**

1. 进入 Configuration → Actions → Create action
2. 配置触发条件（Trigger conditions）
3. Operations 中选择上面创建的 Webhook 媒介类型
4. 发送内容为 Zabbix 默认 JSON 格式即可，平台内置 Zabbix 解析器会提取 `subject`、`message`、`host.name`、`host.ip`、`event.severity` 等字段

### 2.3 Prometheus Alertmanager 配置

在 Alertmanager 的 `alertmanager.yml` 中添加 receiver：

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'network-ticket'
  routes:
    - match:
        severity: critical
      receiver: 'network-ticket'
    - match:
        severity: warning
      receiver: 'network-ticket'

receivers:
  - name: 'network-ticket'
    webhook_configs:
      - url: 'http://your-server/api/v1/alerts/webhook/{source_id}'
        send_resolved: true
```

**配置说明**：

| 参数             | 示例值               | 说明                              |
| ---------------- | -------------------- | --------------------------------- |
| `url`            | 平台 Webhook 地址    | 将 `{source_id}` 替换为实际告警源 ID |
| `send_resolved`  | `true` / `false`     | `true` 时告警恢复也会推送（平台解析为 `severity: info`） |
| `severity` match | `critical` / `warning` | 可根据需要调整路由匹配规则        |

平台内置 Prometheus 解析器会提取 `alerts.0.labels.alertname`、`alerts.0.annotations.summary`、`alerts.0.labels.severity`、`alerts.0.labels.instance` 等字段。

### 2.4 通用 JSON 告警

对于其他监控系统，使用 `generic` 类型。平台通过 [gjson](https://github.com/tidwall/gjson) 路径从原始 JSON 中提取字段。

**提取规则**：

| 提取字段   | gjson 路径（按优先级）                          | 说明                              |
| ---------- | ----------------------------------------------- | --------------------------------- |
| 标题       | `title` → `alertname`                          | 先取 `title`，不存在则取 `alertname` |
| 描述       | `description` → `message`                      | 先取 `description`，不存在则取 `message` |
| 严重程度   | `severity`（自动归一化为 critical/warning/info） | 含 `crit`/`p1`/`emerg` → `critical`；含 `warn`/`p2`/`high` → `warning`；其他 → `info` |
| 源 IP      | `source_ip`                                    | 直接取 `source_ip` 字段           |
| 设备名称   | `device_name`                                  | 直接取 `device_name` 字段         |

**gjson 路径语法速查**：

| 语法          | 示例                | 说明                          |
| ------------- | ------------------- | ----------------------------- |
| `.`           | `host.name`         | 访问嵌套对象字段              |
| `\.`          | `event\.severity`  | 字段名本身含点号时需转义      |
| `#`           | `alerts.0.labels`   | 数组索引（从 0 开始）         |
| 通配符 `*`    | `alerts.0.labels.*` | 匹配对象下所有字段（不常用）  |

**示例 1：扁平 JSON（最简单）**

```json
{
  "title": "CPU 使用率过高",
  "description": "server-01 CPU 使用率超过 90%",
  "severity": "critical",
  "source_ip": "192.168.1.100",
  "device_name": "server-01"
}
```

**示例 2：嵌套 JSON（使用点号路径）**

```json
{
  "alert": {
    "name": "磁盘空间不足",
    "summary": "/data 分区剩余空间 < 10%"
  },
  "meta": {
    "host": "db-master-01",
    "ip": "10.0.0.5"
  },
  "level": "warning"
}
```

> 若需适配此格式，可通过 `parser_config` 配置自定义映射（当前 `generic` 固定使用上述路径，如需自定义请使用特定解析器或联系开发团队）。

**示例 3：含数组的 JSON**

```json
{
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "MemoryHigh",
        "instance": "192.168.1.20:9100",
        "severity": "warning"
      },
      "annotations": {
        "summary": "内存使用率超过 85%"
      }
    }
  ]
}
```

> 数组访问方式：`alerts.0.labels.alertname` 取第一个 alert 的 `alertname`。这与 Prometheus 解析器使用的路径一致。

### 2.5 轮询配置

当外部系统不支持 Webhook 时，可配置轮询模式。平台会定时向 `poll_endpoint` 发送 `GET` 请求拉取告警列表。

**启用条件**：`poll_endpoint` 和 `poll_interval` 必须同时设置。

**完整配置示例**：

```json
{
  "name": "内部监控 API",
  "type": "generic",
  "poll_endpoint": "http://monitor.internal/api/alerts",
  "poll_interval": 60,
  "dedup_fields": ["id"],
  "dedup_window_sec": 300,
  "status": "active"
}
```

**轮询端点响应格式要求**：

轮询端点应返回 JSON 数组，每个元素为一条告警原始数据：

```json
[
  {
    "title": "CPU 使用率过高",
    "description": "server-01 CPU 超过 90%",
    "severity": "critical",
    "source_ip": "192.168.1.100",
    "device_name": "server-01",
    "id": "alert-001"
  },
  {
    "title": "内存不足",
    "description": "server-02 内存使用率 95%",
    "severity": "warning",
    "source_ip": "192.168.1.101",
    "device_name": "server-02",
    "id": "alert-002"
  }
]
```

> 平台会遍历数组中的每个元素，逐条解析并创建/关联工单。若返回非数组 JSON，则尝试将其作为单条告警处理。

### 2.6 去重配置

防止重复告警产生重复工单。平台对 `dedup_fields` 指定的字段值拼接后计算 SHA-256 指纹，在同一时间窗口内相同指纹的告警会追加到已有工单，而非创建新工单。

**配置字段**：

| 字段               | 示例值                  | 说明                              |
| ------------------ | ----------------------- | --------------------------------- |
| `dedup_fields`     | `["host", "alertname"]` | JSON 数组，元素为 gjson 路径字符串 |
| `dedup_window_sec` | `300`                   | 去重时间窗口（秒），默认 300（5 分钟） |

**完整配置示例**：

```json
{
  "name": "Zabbix 主监控",
  "type": "zabbix",
  "dedup_fields": ["host.name", "event.name"],
  "dedup_window_sec": 600,
  "status": "active"
}
```

**去重效果示例**：

假设 `dedup_fields` 配置为 `["host", "alertname"]`，`dedup_window_sec` 为 `300`：

| 时间   | 收到告警                          | 处理结果                    |
| ------ | --------------------------------- | --------------------------- |
| T+0s   | `{"host":"srv01","alertname":"CPU高","severity":"critical"}` | 创建新工单 #NT001           |
| T+60s  | `{"host":"srv01","alertname":"CPU高","severity":"warning"}` | 追加到工单 #NT001（指纹相同） |
| T+120s | `{"host":"srv02","alertname":"CPU高","severity":"critical"}` | 创建新工单 #NT002（host 不同） |
| T+400s | `{"host":"srv01","alertname":"CPU高","severity":"critical"}` | 创建新工单 #NT003（窗口已过） |

**指纹计算逻辑**：

```
指纹 = SHA256( host值 + "|" + alertname值 )
      = SHA256( "srv01|CPU高" )
```

> 若 `dedup_fields` 中指定的字段在告警 JSON 中不存在，该次去重计算会失败，平台将跳过去重直接创建新工单。

---

## 3. 客户管理

客户是指接收工单推送的外部系统。仅管理员可操作。

### 3.1 创建客户

进入「客户管理」页面，点击「新建客户」，填写以下信息：

| 字段            | 必填 | 说明                                     |
| --------------- | ---- | ---------------------------------------- |
| 名称 (name)     | 是   | 客户名称，如 "XX公司运维系统"             |
| 推送地址        | 是   | 客户接收工单推送的 HTTP 端点             |
| API Key         | 是   | 客户身份标识，用于 HMAC 签名             |
| HMAC Secret     | 是   | HMAC-SHA256 签名密钥                     |
| 回调地址        | 否   | 客户授权回调 URL（平台推送给客户时携带）  |
| 状态 (status)   | 否   | `active`（默认）/ `inactive`             |

**完整配置示例**：

```json
{
  "name": "XX公司运维系统",
  "api_endpoint": "https://ops.xxx.com/api/tickets",
  "api_key": "ak_xxx_company_2024",
  "hmac_secret": "hs_xxx_company_secret",
  "callback_url": "https://ops.xxx.com/callback/ticket",
  "config": "{}",
  "status": "active"
}
```

各字段格式说明：

| 字段           | 格式示例                              | 说明                              |
| -------------- | ------------------------------------- | --------------------------------- |
| `api_key`      | `"ak_xxx"`                            | 建议前缀 `ak_`，长度 16~64 位     |
| `hmac_secret`  | `"hs_xxx"`                            | 建议前缀 `hs_`，长度 32~128 位    |
| `api_endpoint` | `"https://client.com/api/tickets"`    | 必须可被平台服务器访问            |
| `callback_url` | `"https://client.com/callback"`       | 客户回调平台时，平台会将其回传    |
| `config`       | `"{}"` 或 `"{\"custom\":true}"`      | 扩展配置，JSON 字符串格式          |
| `status`       | `"active"` / `"inactive"`             | `inactive` 时停止向该客户推送工单 |

### 3.2 API Key 与 HMAC Secret 说明

- **API Key** (`api_key`)：客户身份标识符。平台推送工单和客户回调时通过此值识别客户身份。
- **HMAC Secret** (`hmac_secret`)：用于计算 HMAC-SHA256 签名的密钥。平台向客户推送时自动签名，客户回调平台时也需使用相同密钥签名。两端的 `hmac_secret` 必须一致。

### 3.3 测试连通性

创建客户后，可使用 curl 测试推送地址是否可达：

```bash
curl -X POST https://client-server/api/tickets \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: your_api_key" \
  -H "X-Timestamp: $(date +%s)" \
  -H "X-Signature: test_signature" \
  -H "X-Nonce: $(uuidgen)" \
  -d '{"ticket_no":"TEST001","title":"连通性测试","severity":"info"}'
```

---

## 4. 工单管理

### 4.1 工单列表

进入「工单管理」页面，可查看所有工单。

**筛选条件**：
- 按状态筛选：pending / in_progress / completed / failed / cancelled / rejected
- 按客户筛选
- 按严重程度筛选：critical / warning / info
- 关键词搜索

**分页**：默认每页 20 条，支持翻页。

### 4.2 工单详情

点击工单编号进入详情页，包含以下信息：

**基本信息**：工单编号、标题、描述、严重程度、来源类型、关联客户、创建时间。

**Workflow 时间线**：7 个节点的状态追踪：

| 节点             | 说明                       |
| ---------------- | -------------------------- |
| alert_received   | 告警已接收                 |
| parsed           | 告警已解析                 |
| pushed           | 已推送至客户               |
| awaiting_auth    | 等待客户授权               |
| authorized       | 客户已授权                 |
| executing        | 执行中                     |
| completed        | 已完成                     |

每个节点的状态为：`pending` / `active` / `done` / `failed` / `skipped` / `timeout`。

**Workflow 时间线示例**：

一个正常流转的工单，其 workflow 状态如下：

| 时间                | 节点            | 状态   | 说明                    |
| ------------------- | --------------- | ------ | ----------------------- |
| 2024-01-01 12:00:00 | alert_received  | done   | 收到 Zabbix 告警        |
| 2024-01-01 12:00:01 | parsed          | done   | 解析出 title/severity   |
| 2024-01-01 12:00:02 | pushed          | done   | 推送至客户 A            |
| 2024-01-01 12:00:02 | awaiting_auth   | active | 等待客户授权            |
| 2024-01-01 12:05:00 | authorized      | done   | 客户 A 回调授权         |
| 2024-01-01 12:05:01 | executing       | done   | 进入执行阶段            |
| 2024-01-01 12:30:00 | completed       | done   | 处理完成                |

异常场景示例：

| 场景         | 节点状态变化                                |
| ------------ | ------------------------------------------- |
| 推送失败     | pushed → failed，工单状态变为 `failed`      |
| 客户拒绝     | awaiting_auth → skipped, authorized → skipped, 工单状态变为 `rejected` |
| 管理员取消   | 任意节点 → cancelled，工单状态变为 `cancelled` |

**告警追加记录**：同一指纹的重复告警在去重窗口内会追加到已有工单，可在详情页查看追加历史。例如：

| 时间                | 告警内容           | 操作         |
| ------------------- | ------------------ | ------------ |
| 2024-01-01 12:00:00 | CPU 使用率 90%     | 创建工单     |
| 2024-01-01 12:02:00 | CPU 使用率 95%     | 追加记录     |
| 2024-01-01 12:04:00 | CPU 使用率 98%     | 追加记录     |

### 4.3 手动操作

- **重试**：当工单状态为 `failed` 时，点击「重试」将状态重置为 `pending`，重新进入推送队列。
- **取消**：将工单状态置为 `cancelled`。

### 4.4 工单状态说明

| 状态         | 说明                               |
| ------------ | ---------------------------------- |
| pending      | 新创建，等待推送至客户             |
| in_progress  | 客户已接收/已授权，处理中          |
| completed    | 工单处理完成                       |
| failed       | 推送失败（重试耗尽后标记）         |
| cancelled    | 管理员手动取消                     |
| rejected     | 客户拒绝授权                       |

**合法的状态转换**：

```
pending → in_progress  (客户授权)
pending → failed       (推送失败)
pending → cancelled    (管理员取消)
in_progress → completed (管理员完成)
in_progress → failed    (系统标记)
in_progress → rejected  (客户拒绝)
in_progress → cancelled (管理员取消)
failed → pending       (管理员重试)
failed → cancelled     (管理员取消)
```

---

## 5. 客户 API 对接指南

本节面向客户的开发人员，说明如何接收平台推送的工单以及如何回调授权。

### 5.1 推送协议（我方 → 客户）

当新工单创建后，平台会向客户的 `api_endpoint` 发送 HTTP POST 请求。

**请求头**：

| 请求头       | 说明                          |
| ------------ | ----------------------------- |
| Content-Type | application/json              |
| X-Api-Key    | 客户的 API Key                |
| X-Timestamp  | Unix 时间戳（秒）             |
| X-Signature  | HMAC-SHA256 签名              |
| X-Nonce      | UUID，防止重放                 |

**请求体**：

```json
{
  "ticket_no": "NT20240101120000a1b2c3",
  "title": "CPU 使用率过高",
  "description": "server-01 CPU 使用率超过 90%",
  "severity": "critical",
  "alert_parsed": {
    "source_ip": "192.168.1.100",
    "device_name": "server-01"
  },
  "callback_url": "https://client-server/callback"
}
```

`alert_parsed` 字段包含平台从原始告警中提取的关键信息，内容取决于告警源类型和原始数据结构。常见字段包括：`source_ip`、`device_name`、`alert_time` 等。

**客户验证签名的步骤**：

1. 从请求头获取 `X-Timestamp`、`X-Signature`、`X-Nonce`
2. 检查 Timestamp 与当前时间差是否在 300 秒以内
3. 检查 Nonce 是否已使用过（防重放）
4. 计算 `HMAC-SHA256(hmac_secret, timestamp + request_body)`
5. 对比计算结果与 `X-Signature` 是否一致

### 5.2 回调协议（客户 → 我方）

客户处理工单后，调用平台回调接口进行授权或拒绝。

**接口地址**：`POST http://your-server/api/v1/callback/authorization`

**请求头**：

| 请求头       | 说明                          |
| ------------ | ----------------------------- |
| Content-Type | application/json              |
| X-Api-Key    | 客户的 API Key                |
| X-Timestamp  | Unix 时间戳（秒）             |
| X-Signature  | HMAC-SHA256 签名              |
| X-Nonce      | UUID，防重放                   |

**请求体**：

```json
{
  "ticket_no": "NT20240101120000a1b2c3",
  "action": "authorize",
  "operator": "张三",
  "comment": "已确认，开始处理",
  "authorized_at": "2024-01-01T12:05:00Z"
}
```

| 字段          | 必填 | 说明                                    |
| ------------- | ---- | --------------------------------------- |
| ticket_no     | 是   | 工单编号                                |
| action        | 是   | `authorize`（授权）或 `reject`（拒绝）   |
| operator      | 否   | 操作人                                  |
| comment       | 否   | 备注                                    |
| authorized_at | 否   | 授权时间 (ISO 8601)                     |

**拒绝回调示例**：

```json
{
  "ticket_no": "NT20240101120000a1b2c3",
  "action": "reject",
  "operator": "李四",
  "comment": "非我方负责范围，请转交网络组",
  "authorized_at": "2024-01-01T12:10:00Z"
}
```

**成功响应**：

```json
{
  "code": 0,
  "message": "ok",
  "ticket_no": "NT20240101120000a1b2c3",
  "status": "authorized"
}
```

### 5.3 HMAC 签名计算方法

签名算法：`HMAC-SHA256(hmac_secret, timestamp_string + request_body)`

Python 示例：

```python
import hmac
import hashlib
import time
import uuid
import json

def sign_request(hmac_secret: str, body: str) -> dict:
    """计算 HMAC 签名并返回所需的请求头"""
    timestamp = str(int(time.time()))
    nonce = str(uuid.uuid4())

    # 签名内容 = timestamp 字符串 + request body 字符串
    message = timestamp.encode('utf-8') + body.encode('utf-8')
    signature = hmac.new(
        hmac_secret.encode('utf-8'),
        message,
        hashlib.sha256
    ).hexdigest()

    return {
        'X-Api-Key': 'your_api_key',
        'X-Timestamp': timestamp,
        'X-Signature': signature,
        'X-Nonce': nonce,
    }
```

### 5.4 防重放说明

平台使用 **Nonce + Timestamp** 双重防重放机制：

- **Timestamp 校验**：请求时间戳与服务器时间差超过 300 秒（5 分钟）则拒绝
- **Nonce 唯一性**：每个 Nonce 值在 TTL（5 分钟）内只能使用一次，重复则返回 409 Conflict

Nonce 存储后端支持两种模式（在 `config.yaml` 的 `security.nonce.backend` 中配置）：

| 后端 | 说明                         |
| ---- | ---------------------------- |
| db   | 存储在 `nonce_records` 表    |
| file | 存储在本地文件（适合单实例）  |

### 5.5 完整对接示例 (Python Flask)

以下示例展示客户系统如何接收工单推送并发送授权回调：

```python
import hmac
import hashlib
import json
import time
import uuid

from flask import Flask, request, jsonify
import requests

app = Flask(__name__)

API_KEY = "ak_xxx"
HMAC_SECRET = "hs_xxx"
PLATFORM_URL = "http://your-server/api/v1/callback/authorization"


def verify_signature(secret: str, timestamp: str, body: bytes, signature: str) -> bool:
    """验证平台推送请求的 HMAC 签名"""
    message = timestamp.encode('utf-8') + body
    expected = hmac.new(secret.encode('utf-8'), message, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)


def make_signed_request(secret: str, api_key: str, body: dict) -> dict:
    """构造带 HMAC 签名的请求头"""
    body_str = json.dumps(body, ensure_ascii=False)
    timestamp = str(int(time.time()))
    nonce = str(uuid.uuid4())

    message = timestamp.encode('utf-8') + body_str.encode('utf-8')
    signature = hmac.new(secret.encode('utf-8'), message, hashlib.sha256).hexdigest()

    return {
        'headers': {
            'Content-Type': 'application/json',
            'X-Api-Key': api_key,
            'X-Timestamp': timestamp,
            'X-Signature': signature,
            'X-Nonce': nonce,
        },
        'data': body_str,
    }


@app.route('/api/tickets', methods=['POST'])
def receive_ticket():
    """接收平台推送的工单"""
    # 1. 验证签名
    timestamp = request.headers.get('X-Timestamp', '')
    signature = request.headers.get('X-Signature', '')
    body = request.get_data()

    if not verify_signature(HMAC_SECRET, timestamp, body, signature):
        return jsonify({'error': 'invalid signature'}), 401

    # 2. 解析工单
    ticket = json.loads(body)
    ticket_no = ticket['ticket_no']
    print(f"收到工单: {ticket_no} - {ticket['title']}")

    # 3. 业务处理（存库、通知等）
    # ...

    # 4. 回调授权
    callback_body = {
        'ticket_no': ticket_no,
        'action': 'authorize',
        'operator': '值班工程师',
        'comment': '已确认，开始处理',
        'authorized_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
    }

    req = make_signed_request(HMAC_SECRET, API_KEY, callback_body)
    resp = requests.post(PLATFORM_URL, **req)

    print(f"回调结果: {resp.status_code} {resp.text}")
    return jsonify({'code': 0, 'message': 'received'})


if __name__ == '__main__':
    app.run(port=5000)
```

---

## 6. 审计日志

### 查看操作记录

进入「审计日志」页面，查看系统所有操作的记录，包括：

- 工单创建、状态变更
- 客户授权/拒绝回调
- 管理员手动操作
- 告警源和客户的增删改

### 筛选

支持按以下条件筛选：

- **时间范围**：按操作时间筛选
- **操作人**：按 operator 筛选
- **分页浏览**：默认每页 20 条

审计日志由后端自动记录，不可编辑或删除。
