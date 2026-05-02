# 工单类型 + 手工建单设计文档

> 日期：2026-04-29
> 范围：工单类型管理、手工建单功能

---

## 背景

当前系统所有工单均由告警自动触发创建，无法手工录入。为了便于测试工单页面和流程，需要支持手工建单。同时引入"工单类型"概念，将工单分类，为后续不同流程扩展打基础。

---

## 目标

1. 支持管理员定义和管理工单类型（编码、名称、颜色等）
2. 支持手工创建工单，选择类型、填写信息
3. 保持与现有流程兼容（所有类型共用现有 7 节点工作流）
4. 历史工单兼容（`ticket_type_id` 可为 NULL）

---

## 非目标

- 不同工单类型走不同流程（v2 功能）
- 修改现有自动告警创建流程
- 工单类型的权限控制（如某类型只有某角色能看）

---

## 设计

### 1. 数据模型

#### 1.1 ticket_types 表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT UNSIGNED | PK, AUTO_INCREMENT | |
| code | VARCHAR(64) | NOT NULL, UNIQUE | 类型编码，如 `network_fault` |
| name | VARCHAR(128) | NOT NULL | 显示名称，如 "网络故障" |
| description | TEXT | NULL | 描述 |
| color | VARCHAR(7) | DEFAULT '#6B7280' | 前端显示颜色 |
| status | VARCHAR(20) | DEFAULT 'active' | `active` / `inactive` |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP | |

#### 1.2 tickets 表修改

添加字段：
- `ticket_type_id BIGINT UNSIGNED NULL`（可为空，兼容历史数据）
- 外键：`FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id)`

修改字段：
- `alert_source_id BIGINT UNSIGNED NULL`（原 NOT NULL，改为可为空，支持手工建单）

### 2. 后端 API

#### 2.1 工单类型管理（admin-only）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/ticket-types` | admin | 列表（所有状态） |
| POST | `/api/v1/ticket-types` | admin | 创建 |
| PUT | `/api/v1/ticket-types/:id` | admin | 更新 |
| DELETE | `/api/v1/ticket-types/:id` | admin | 删除（仅当无关联工单时） |

**请求/响应格式：**

```json
// POST /api/v1/ticket-types
{
  "code": "network_fault",
  "name": "网络故障",
  "description": "网络设备或链路故障",
  "color": "#FF6B6B",
  "status": "active"
}

// Response
{
  "id": 1,
  "code": "network_fault",
  "name": "网络故障",
  "description": "网络设备或链路故障",
  "color": "#FF6B6B",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### 2.2 手工建单

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| POST | `/api/v1/tickets/manual` | admin/operator | 手工创建工单 |

**请求体：**

```json
{
  "ticket_type_id": 1,
  "title": "核心交换机故障",
  "description": "机房 A 核心交换机无响应",
  "severity": "critical",
  "client_id": 2
}
```

| 字段 | 必填 | 说明 |
|------|:----:|------|
| ticket_type_id | ✅ | 工单类型 ID |
| title | ✅ | 标题 |
| description | ❌ | 描述 |
| severity | ✅ | `critical` / `warning` / `info` |
| client_id | ❌ | 关联客户 ID（可选） |

**处理逻辑：**

1. 校验 `ticket_type_id` 存在且 `status = 'active'`
2. 如果 `client_id` 提供，校验客户存在且 `status = 'active'`
3. 构造 `*parser.ParsedAlert`：
   ```go
   parsed := &parser.ParsedAlert{
       Title:       req.Title,
       Description: req.Description,
       Severity:    req.Severity,
   }
   ```
4. 调用 `TicketService.CreateTicket`：
   - `alertSourceID = 0`（无告警源）
   - `sourceType = "manual"`
   - `alertRaw = []byte("{}")`（空 JSON）
   - `parsedAlert = parsed`
   - `clientID = req.ClientID`
   - `fingerprint = nil`
5. 工单创建后，设置 `ticket_type_id`
6. 返回创建的工单

**响应：**

```json
{
  "id": 42,
  "ticket_no": "NT20240101120000a1b2c3",
  "ticket_type_id": 1,
  "title": "核心交换机故障",
  "description": "机房 A 核心交换机无响应",
  "severity": "critical",
  "status": "pending",
  "client_id": 2,
  "created_at": "2024-01-01T12:00:00Z"
}
```

### 3. 前端页面

#### 3.1 工单类型管理 `/ticket-types`

**权限**：admin only

**页面结构**：
- 表格：编码、名称（带颜色块）、描述、状态
- 新增/编辑按钮 → 对话框

**对话框字段**：
- 编码（必填，英文字母+下划线）
- 名称（必填）
- 描述（可选）
- 颜色（颜色选择器，默认 #6B7280）
- 状态（active/inactive）

#### 3.2 手工建单

不单独建页面，在「工单管理」页面添加「手工建单」按钮：

- 点击弹出对话框
- 字段：
  - 工单类型（下拉选择，显示颜色块）
  - 标题（文本输入）
  - 描述（多行文本）
  - 严重程度（下拉：critical/warning/info）
  - 关联客户（下拉选择，可选，显示客户名称）
- 提交后刷新工单列表

### 4. 数据库迁移

**迁移文件：`010_add_ticket_types.up.sql`**

```sql
-- 创建工单类型表
CREATE TABLE ticket_types (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#6B7280',
    status VARCHAR(20) DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 修改 tickets 表
ALTER TABLE tickets 
    MODIFY alert_source_id BIGINT UNSIGNED NULL,
    ADD COLUMN ticket_type_id BIGINT UNSIGNED NULL AFTER source_type;

-- 添加外键
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_ticket_type 
    FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id);

-- 插入默认类型
INSERT INTO ticket_types (code, name, description) VALUES 
    ('default', '默认', '未分类工单'),
    ('network_fault', '网络故障', '网络设备或链路故障'),
    ('server_alert', '服务器告警', '服务器性能或硬件告警');
```

**迁移文件：`010_add_ticket_types.down.sql`**

```sql
ALTER TABLE tickets DROP FOREIGN KEY fk_tickets_ticket_type;
ALTER TABLE tickets DROP COLUMN ticket_type_id;
ALTER TABLE tickets MODIFY alert_source_id BIGINT UNSIGNED NOT NULL;
DROP TABLE ticket_types;
```

### 5. 模型定义

**`backend/internal/model/ticket_type.go`**

```go
package model

import "time"

type TicketType struct {
    ID          int64     `db:"id"          json:"id"`
    Code        string    `db:"code"        json:"code"`
    Name        string    `db:"name"        json:"name"`
    Description *string   `db:"description" json:"description"`
    Color       string    `db:"color"       json:"color"`
    Status      string    `db:"status"      json:"status"`
    CreatedAt   time.Time `db:"created_at"  json:"created_at"`
    UpdatedAt   time.Time `db:"updated_at"  json:"updated_at"`
}
```

**修改 `backend/internal/model/ticket.go`**

在 `Ticket` 结构体中添加：
```go
TicketTypeID *int64 `db:"ticket_type_id" json:"ticket_type_id"`
```

### 6. 与现有流程的兼容

- 自动告警创建的工单：`ticket_type_id = NULL`，保持现有行为不变
- 手工创建的工单：`ticket_type_id` 为用户选择的类型
- 工单列表/详情页：显示工单类型名称和颜色（如果 `ticket_type_id` 不为空）

---

## 数据流

```
用户访问 /ticket-types
      ↓
管理员创建/编辑工单类型
      ↓
用户访问 /tickets
      ↓
点击「手工建单」
      ↓
填写表单（选择类型、标题、描述、严重程度、客户）
      ↓
POST /api/v1/tickets/manual
      ↓
后端校验类型和客户 → 构造 ParsedAlert → CreateTicket
      ↓
返回工单 → 刷新列表
```

---

## 错误处理

| 场景 | 处理方式 |
|------|---------|
| 工单类型不存在或 inactive | 返回 400，提示"工单类型无效" |
| 客户不存在或 inactive | 返回 400，提示"客户无效" |
| 编码重复 | 返回 400，提示"编码已存在" |
| 删除有关联工单的类型 | 返回 400，提示"该类型已被使用，无法删除" |

---

## 实现顺序

1. **数据库迁移**（创建 ticket_types 表，修改 tickets 表）
2. **后端**：TicketTypeRepo + TicketTypeService + TicketTypeHandler
3. **后端**：手工建单 API（TicketHandler.CreateManual）
4. **前端**：工单类型管理页面 `/ticket-types`
5. **前端**：工单列表页添加「手工建单」按钮和对话框
6. **前端**：工单详情显示类型信息
