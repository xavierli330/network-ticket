---
marp: true
theme: default
class: invert
paginate: true
---

<!-- _class: lead -->

# 网络工单平台

## 部署 · 测试 · 使用 演示指南

**Network Ticket Platform**

---

## 目录

1. 项目简介
2. 部署步骤
3. 功能演示
4. 测试验证
5. 使用入门

---

## 一、项目简介

### 什么是网络工单平台？

**告警接收 → 工单生成 → 推送给客户 → 客户处理** 的自动化系统

```
Zabbix 发现 "服务器 CPU 95%"
        ↓
   推送给本平台
        ↓
   生成工单 #NT001
        ↓
   推送给 XX 公司运维系统
        ↓
   XX 公司回调授权
        ↓
   处理完成，状态变为 "completed"
```

---

### 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.26 + Gin + sqlx + MySQL 8.0 |
| 前端 | Next.js 16 + React 19 + Tailwind CSS 4 |
| 部署 | Docker Compose + Nginx |
| 认证 | JWT + HMAC-SHA256 |

---

### 核心功能一览

| 功能 | 状态 |
|------|------|
| 告警源管理（Zabbix/Prometheus/Generic） | ✅ |
| 告警接收与自动创建工单 | ✅ |
| 工单去重（指纹算法） | ✅ |
| 工单类型管理 | ✅ |
| 手动创建工单 | ✅ |
| 客户管理与工单推送 | ✅ |
| 客户回调授权/拒绝 | ✅ |
| 用户管理（admin/operator） | ✅ |
| 审计日志 | ✅ |
| 轮询调度器 | ✅ |

---

## 二、部署步骤

### 环境要求

| 依赖 | 版本要求 |
|------|----------|
| Docker | 20.10+ |
| Docker Compose | v2+ |
| 端口 | 80, 8080, 3306 |

---

### 一键部署（推荐）

```bash
# 1. 克隆项目
git clone <repo-url>
cd network-ticket

# 2. 运行部署脚本
./deploy.sh
```

脚本会自动完成：
- ✅ 检查 Docker 环境
- ✅ 生成 `.env` 和 `config.yaml`
- ✅ 构建镜像
- ✅ 执行数据库迁移
- ✅ 健康检查

---

### 部署完成后

```bash
# 查看服务状态
./manage.sh status

# 查看日志
./manage.sh logs

# 查看后端日志
./manage.sh logs backend

# 停止服务
./manage.sh stop

# 启动服务
./manage.sh start
```

---

### 访问系统

| 入口 | 地址 | 默认账号 |
|------|------|----------|
| 前端页面 | `http://your-server/login` | admin / admin123 |
| 后端 API | `http://your-server/api/v1` | - |

> ⚠️ 首次登录后请立即修改默认密码

---

### 配置文件说明

`config.yaml` 关键配置项：

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.port` | 后端端口 | 8080 |
| `database.*` | MySQL 连接信息 | - |
| `jwt.secret` | JWT 签名密钥 | 随机生成 |
| `jwt.expire_hours` | Token 有效期 | 24 |
| `security.nonce.backend` | 防重放存储 | db |
| `worker_pool.size` | Worker 数量 | 10 |

---

## 三、功能演示

### 3.1 告警源管理

**操作路径**：告警源管理 → 新建告警源

| 字段 | 示例值 |
|------|--------|
| 名称 | Zabbix 生产监控 |
| 类型 | zabbix |
| Webhook Secret | wh_zabbix_prod_2024 |
| 去重字段 | ["host.name", "event.name"] |

**保存后得到专属 Webhook 地址**：
```
POST http://your-server/api/v1/alerts/webhook/1
```

---

### 3.2 工单类型管理

**操作路径**：工单类型管理 → 新建工单类型

**常用类型**：

| 编码 | 名称 | 颜色 |
|------|------|------|
| incident | 故障工单 | 🔴 红色 |
| request | 服务请求 | 🔵 蓝色 |
| maintenance | 维护工单 | 🟢 绿色 |

> 工单类型用于分类，创建工单时必须选择

---

### 3.3 手动创建工单

**操作路径**：工单管理 → 新建工单

**适用场景**：
- 内部故障申报（不走监控系统）
- 服务请求
- 功能测试

**填写内容**：
- 工单类型（必选）
- 标题、描述
- 严重程度
- 关联客户（可选）

---

### 3.4 客户管理

**操作路径**：客户管理 → 新建客户

| 字段 | 说明 |
|------|------|
| 名称 | 客户名称 |
| 推送地址 | 客户接收工单的 API 端点 |
| API Key | 客户身份标识 |
| HMAC Secret | 签名密钥（需保密） |

> 创建后，工单可推送给该客户，客户通过回调接口返回授权/拒绝

---

### 3.5 用户管理

**操作路径**：用户管理 → 新增用户

| 角色 | 权限 |
|------|------|
| **admin** | 全部功能 |
| **operator** | 工单操作、查看，不能管理客户/用户/工单类型 |

> 管理员不能删除自己的账号

---

### 3.6 审计日志

**操作路径**：审计日志

记录所有操作：
- 工单创建、状态变更
- 客户授权/拒绝回调
- 管理员操作
- 增删改操作

> 不可编辑或删除

---

## 四、测试验证

### 4.1 运行单元测试

```bash
cd backend
make test
```

**要求**：MySQL 必须正在运行（测试依赖真实数据库）

---

### 4.2 功能验证清单

| 验证项 | 操作 | 预期结果 |
|--------|------|----------|
| 登录 | 访问 /login，输入 admin/admin123 | 登录成功，进入首页 |
| 创建告警源 | 告警源管理 → 新建 → 保存 | 列表中显示新告警源 |
| 创建工单类型 | 工单类型管理 → 新建 → 保存 | 列表中显示新类型 |
| 手动创建工单 | 工单管理 → 新建工单 → 填写 → 保存 | 工单列表中出现新工单 |
| 创建客户 | 客户管理 → 新建 → 保存 | 列表中显示新客户 |
| 创建用户 | 用户管理 → 新增 → 保存 | 列表中显示新用户 |
| 审计日志 | 审计日志页面 | 显示所有操作记录 |

---

### 4.3 API 连通性测试

```bash
# 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.token')

# 获取告警源列表
curl http://localhost/api/v1/alert-sources \
  -H "Authorization: Bearer $TOKEN"

# 获取工单列表
curl http://localhost/api/v1/tickets \
  -H "Authorization: Bearer $TOKEN"
```

---

### 4.4 Webhook 测试

```bash
# 发送测试告警（替换 {source_id}）
curl -X POST http://localhost/api/v1/alerts/webhook/1 \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "CPU 测试告警",
    "message": "server-01 CPU 超过 90%",
    "event": {"severity": "High"},
    "host": {"name": "server-01", "ip": "192.168.1.100"}
  }'
```

预期：工单列表中出现新工单

---

## 五、使用入门

### 首次使用三步走

**第一步：创建告警源**
- 路径：告警源管理 → 新建
- 填写名称、类型（zabbix/prometheus/generic）
- 保存后获得 Webhook 地址

**第二步：配置监控系统**
- 在 Zabbix/Prometheus 中填入平台 Webhook 地址
- 测试告警是否能正常推送到平台

**第三步：创建客户并测试推送**
- 创建客户，填写推送地址
- 触发告警，观察工单是否生成
- （当前版本需手动触发推送，详见已知问题）

---

### 完整配置流程

```
1. 创建告警源（平台侧）
        ↓
2. 在 Zabbix 中配置 Webhook（监控侧）
        ↓
3. 创建客户（平台侧）
        ↓
4. 提供 API Key + HMAC Secret 给客户
        ↓
5. 客户开发接收和回调接口
        ↓
6. 触发测试告警，验证全流程
```

---

### 已知问题

| 问题 | 影响 | 临时方案 |
|------|------|----------|
| 工单创建后不会自动推送给客户 | 工单停留在 pending 状态 | 手动通过 API 触发推送，或联系开发团队修复 |
| 工单超时功能 | 设置了超时时间不生效 | 暂不支持，v1.5 计划实现 |

> 详见 `docs/known-issues.md`

---

### 文档导航

| 文档 | 内容 | 适合谁 |
|------|------|--------|
| `README.md` | 项目概览、快速开始 | 所有人 |
| `docs/deployment.md` | 生产部署详细指南 | 运维 |
| `docs/development.md` | 本地开发环境搭建 | 开发者 |
| `docs/api.md` | 完整 API 文档 | 集成开发者 |
| `docs/usage.md` | 功能使用指南 | 管理员/操作员 |
| `docs/user-manual.md` | 非技术用户手册 | 非技术人员 |
| `docs/architecture.md` | 系统架构设计 | 技术读者 |

---

<!-- _class: lead -->

## 感谢使用

### 网络工单平台

**文档完整 · 功能就绪 · 欢迎使用**

如有问题，请联系开发团队。