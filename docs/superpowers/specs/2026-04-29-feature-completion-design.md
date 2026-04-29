# 功能补全设计文档

> 日期：2026-04-29
> 范围：审计日志前端页面、用户管理前后端、轮询调度器

---

## 背景

网络工单平台后端 API 和数据库已完整实现，但前端和部分后端功能存在缺失：

1. 审计日志 API 已就绪，但无前端查看页面
2. 用户管理后端 Repo 存在，但无 CRUD API 和前端页面
3. 告警源轮询模式配置字段已存在，但后端无调度器实现

---

## 目标

1. 提供审计日志前端查看能力
2. 提供完整的用户管理功能（前后端）
3. 实现告警源轮询调度器，使轮询模式真正可用

---

## 非目标

- 不修改现有工单状态机逻辑
- 不实现 Dashboard、WebSocket、超时通知等 v1.5+ 功能
- 不修改现有 Docker 部署流程

---

## 设计

### 1. 审计日志前端页面

**页面路径**：`/audit-logs`

**权限**：admin 和 operator 均可见

**布局**：与现有「工单管理」列表页保持一致（左侧筛选 + 右侧表格）

**筛选区**：
- 时间范围：开始时间 / 结束时间（日期时间选择器）
- 操作人：文本输入框
- 「查询」按钮 + 「重置」按钮

**表格列**：

| 列名 | 字段 | 说明 |
|------|------|------|
| 时间 | `created_at` | 格式化为本地时间 |
| 操作 | `action` | 如 ticket_created, status_updated |
| 资源类型 | `resource_type` | ticket / client / alert_source 等 |
| 资源ID | `resource_id` | |
| 操作人 | `operator` | |
| 详情 | `detail` | |

**分页**：默认 20 条/页，与工单列表一致

**侧边栏**：添加「审计日志」导航入口

**API**：`GET /api/v1/audit-logs?page=1&page_size=20&operator=xxx`

---

### 2. 用户管理（前后端）

#### 2.1 后端 API

当前仅有 `POST /api/v1/auth/login`，需新增：

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/v1/users` | admin | 用户列表（支持分页） |
| POST | `/api/v1/users` | admin | 创建用户 |
| PUT | `/api/v1/users/:id` | admin | 更新用户信息（含密码修改） |
| DELETE | `/api/v1/users/:id` | admin | 删除用户 |

**请求/响应格式**：

创建/更新请求体：
```json
{
  "username": "operator1",
  "password": "newpass123",
  "role": "operator"
}
```
- `password`：创建时必填；更新时留空表示不改密码
- `role`：`admin` 或 `operator`

列表响应：
```json
{
  "items": [
    {
      "id": 1,
      "username": "admin",
      "role": "admin",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "total": 10,
  "page": 1,
  "page_size": 20
}
```

#### 2.2 前端页面

**页面路径**：`/users`

**权限控制**：
- 侧边栏：仅 `admin` 角色显示此入口
- 路由守卫：操作员直接访问 `/users` 时跳转到首页或显示无权限

**页面结构**：
- 用户列表表格（用户名、角色、创建时间、操作列）
- 「新增用户」按钮 → 弹出对话框
- 操作列：编辑（对话框）、删除（确认对话框）

**密码处理**：
- 创建用户时必须设置密码
- 编辑用户时密码输入框可选填，留空表示不修改密码

---

### 3. 轮询调度器

#### 3.1 架构

新建 `backend/internal/poller/` 包：

```
poller/
├── scheduler.go    # 调度器主逻辑：启动/停止/重载
├── worker.go       # 单个告警源的轮询 worker
└── poller.go       # 包入口，与 service 层交互
```

#### 3.2 生命周期

**启动流程**：
1. 后端启动时，从数据库加载所有 `status = 'active'` 且 `poll_endpoint != ''` 的告警源
2. 为每个告警源启动一个独立的 goroutine
3. 每个 goroutine 内使用 `time.Ticker` 按 `poll_interval` 定时触发

**动态重载**：
- 告警源被创建/更新/删除时，通过 channel 通知调度器
- 调度器停止旧的 worker，启动新的 worker（或更新配置）

**停止流程**：
- 后端关闭时，调用 `scheduler.Stop()`，等待所有 worker 优雅退出

#### 3.3 Worker 执行逻辑

```
1. 等待 ticker 触发
2. 发送 GET 请求到 poll_endpoint
3. 解析响应（期望 JSON 数组）
4. 遍历数组中的每个告警元素
5. 对每个元素：构造与 Webhook 相同的告警结构，调用 AlertService.Ingest()
6. 记录成功/失败日志
```

#### 3.4 错误处理（指数退避）

请求失败时（网络错误、非 200 状态码、解析失败）：

| 重试次数 | 等待间隔 |
|---------|---------|
| 第 1 次 | 1s |
| 第 2 次 | 2s |
| 第 3 次 | 4s |
| 第 4 次 | 8s |
| 第 5 次 | 16s |

连续失败超过 5 次后，放弃本次轮询，等待下一个正常周期。

> 重试仅在单次轮询内发生，不影响下一次正常周期的定时。

#### 3.5 与现有代码集成

- Worker 调用 `service.AlertService.Ingest()` 处理解析后的告警数据
- 复用现有的解析器（Zabbix / Prometheus / Generic）
- 复用现有的去重逻辑

#### 3.6 并发安全

- 每个告警源独立 goroutine，互不干扰
- 使用 `sync.WaitGroup` 管理 worker 生命周期
- 使用 `context.Context` 传递取消信号

---

## 数据流

### 审计日志页面

```
用户访问 /audit-logs
      ↓
前端发送 GET /api/v1/audit-logs?page=1&page_size=20
      ↓
后端返回审计日志列表
      ↓
前端渲染表格 + 分页
```

### 用户管理

```
管理员访问 /users
      ↓
前端检查 role === 'admin'，否则拒绝访问
      ↓
发送 GET /api/v1/users
      ↓
后端 JWTAuth → RequireAdmin → 返回用户列表
      ↓
前端渲染表格
```

### 轮询调度器

```
后端启动
      ↓
Scheduler.Start(): 加载 active + poll_endpoint 不为空的告警源
      ↓
为每个告警源启动 Worker goroutine
      ↓
Worker: ticker 触发 → GET poll_endpoint → 解析 JSON → Ingest()
      ↓
告警源变更 → Scheduler.Reload() → 停止旧 Worker，启动新 Worker
      ↓
后端关闭 → Scheduler.Stop() → 等待所有 Worker 退出
```

---

## 错误处理

| 场景 | 处理方式 |
|------|---------|
| 轮询请求超时 | 按指数退避重试，记录 warn 日志 |
| 轮询返回非 JSON | 记录 error 日志，跳过本次 |
| 轮询返回非数组 | 尝试作为单条告警处理，失败则记录 error |
| 调度器重载失败 | 记录 error，保持旧 worker 运行 |
| 用户创建时用户名重复 | 返回 400，提示用户名已存在 |
| 删除当前登录用户 | 返回 403，禁止删除自己 |

---

## 安全考虑

1. **用户管理 API**：仅 admin 可访问，通过 `RequireAdmin` 中间件控制
2. **密码存储**：新建/修改密码时使用 bcrypt 哈希，与现有登录逻辑一致
3. **轮询请求**：不携带认证信息（与 Webhook 模式一致，依赖网络层隔离）
4. **删除保护**：禁止删除当前登录的用户（防止把自己锁在外面）

---

## 测试策略

1. **审计日志页面**：手动测试筛选、分页、渲染
2. **用户管理**：
   - 单元测试：创建用户、更新密码、删除用户、列表分页
   - 集成测试：admin 可操作，operator 被拒绝
3. **轮询调度器**：
   - 单元测试：Worker 的指数退避逻辑、JSON 解析
   - 集成测试：Scheduler 的启动/停止/重载、定时触发

---

## 实现顺序

按复杂度递增：

1. **审计日志前端页面**（纯前端，1~2 小时）
2. **用户管理前后端**（前后端都有，但逻辑简单，2~3 小时）
3. **轮询调度器**（涉及定时任务、并发、错误处理，最复杂，3~4 小时）
