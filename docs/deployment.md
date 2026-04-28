# 网络工单系统 - 部署手册

## 环境要求

| 组件       | 版本要求      | 说明                       |
| ---------- | ------------- | -------------------------- |
| Docker     | 20.10+        | 容器运行时                 |
| Docker Compose | v2+       | 服务编排                   |
| MySQL      | 8.0+          | 或使用 Docker 内置 MySQL   |
| Go         | 1.23+         | 仅开发时需要               |
| Node.js    | 20+           | 仅开发时需要               |

## 快速启动

```bash
# 克隆项目
git clone <repo>
cd network-ticket

# 配置
cp backend/config.example.yaml backend/config.yaml
# 编辑 config.yaml 修改数据库密码、JWT secret 等敏感配置

# 启动所有服务
docker compose up -d

# 查看后端日志
docker compose logs -f backend

# 查看所有服务状态
docker compose ps
```

启动完成后：
- 前端：`http://localhost` (Nginx 代理)
- 后端 API：`http://localhost/api/v1/` (Nginx 代理至后端 8080 端口)
- 直连后端：`http://localhost:8080/api/v1/`

## 配置说明

配置文件路径：`backend/config.yaml`。同时支持通过环境变量覆盖，前缀为 `NT_`，例如 `NT_DATABASE_HOST=mysql`、`NT_JWT_SECRET=xxx`。

### server - 服务器配置

| 字段   | 类型   | 默认值   | 说明                     |
| ------ | ------ | -------- | ------------------------ |
| port   | int    | 8080     | HTTP 监听端口            |
| mode   | string | "debug"  | Gin 运行模式 (`debug` / `release`) |

### database - 数据库配置

| 字段           | 类型   | 默认值          | 说明               |
| -------------- | ------ | --------------- | ------------------ |
| host           | string | "127.0.0.1"     | MySQL 主机地址     |
| port           | int    | 3306            | MySQL 端口         |
| user           | string | "ticket"        | 数据库用户名       |
| password       | string | "ticket_password" | 数据库密码       |
| dbname         | string | "network_ticket" | 数据库名称        |
| max_open_conns | int    | 20              | 最大打开连接数     |
| max_idle_conns | int    | 10              | 最大空闲连接数     |

### log - 日志配置

| 字段          | 类型   | 默认值            | 说明                           |
| ------------- | ------ | ----------------- | ------------------------------ |
| level         | string | "debug"           | 日志级别 (`debug`/`info`/`warn`/`error`) |
| format        | string | "json"            | 日志格式 (`json`/`text`)       |
| file_path     | string | "./logs/server.log" | 日志文件路径                 |
| max_size_mb   | int    | 100               | 单个日志文件最大大小 (MB)      |
| max_backups   | int    | 10                | 保留的旧日志文件最大数量       |
| max_age_days  | int    | 30                | 日志文件保留天数               |

### jwt - JWT 配置

| 字段          | 类型   | 默认值                   | 说明               |
| ------------- | ------ | ------------------------ | ------------------ |
| secret        | string | "change-me-in-production" | JWT 签名密钥      |
| expire_hours  | int    | 24                       | Token 过期时间 (小时) |

### security.nonce - Nonce 防重放配置

| 字段    | 类型   | 默认值 | 说明                                      |
| ------- | ------ | ------ | ----------------------------------------- |
| backend | string | "db"   | Nonce 存储后端 (`db` / `file`)            |
| ttl     | string | "5m"   | Nonce 有效期                              |

**security.nonce.file** (仅当 backend=file 时生效):

| 字段 | 类型   | 默认值              | 说明               |
| ---- | ------ | ------------------- | ------------------ |
| path | string | "./data/nonces.log" | Nonce 文件存储路径 |

### worker - Worker Pool 配置

| 字段                | 类型   | 默认值 | 说明                          |
| ------------------- | ------ | ------ | ----------------------------- |
| pool_size           | int    | 10     | 并发 Worker 数量              |
| retry_max           | int    | 5      | 最大重试次数                  |
| retry_base_interval | string | "1s"   | 重试基础间隔                  |
| retry_max_interval  | string | "30s"  | 重试最大间隔 (指数退避上限)   |

## 数据库迁移

项目使用 [golang-migrate](https://github.com/golang-migrate/migrate) 管理数据库迁移，迁移文件位于 `backend/migrations/` 目录。

### 安装 migrate 工具

```bash
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 执行迁移

```bash
cd backend

# 执行所有待运行的迁移 (up)
migrate -path migrations -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" up

# 回退一步 (down)
migrate -path migrations -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" down 1

# 查看当前迁移版本
migrate -path migrations -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" version
```

### 迁移文件清单

| 编号 | 文件                               | 说明          |
| ---- | ---------------------------------- | ------------- |
| 001  | 001_create_users                   | 用户表        |
| 002  | 002_create_alert_sources           | 告警源表      |
| 003  | 003_create_clients                 | 客户表        |
| 004  | 004_create_tickets                 | 工单表        |
| 005  | 005_create_workflow_states         | 工作流状态表  |
| 006  | 006_create_alert_records           | 告警记录表    |
| 007  | 007_create_ticket_logs             | 工单日志表    |
| 008  | 008_create_audit_logs              | 审计日志表    |
| 009  | 009_create_nonce_records           | Nonce 记录表  |

## 默认账号

| 角色   | 用户名 | 密码     | 说明             |
| ------ | ------ | -------- | ---------------- |
| 管理员 | admin  | admin123 | **请立即修改**   |

> 首次部署后务必修改默认管理员密码，避免安全风险。

## HTTPS 配置

生产环境建议启用 HTTPS。

### 操作步骤

1. 将 SSL 证书文件放置到 `nginx/certs/` 目录：

```bash
mkdir -p nginx/certs
cp your-cert.pem nginx/certs/cert.pem
cp your-key.pem nginx/certs/key.pem
```

2. 编辑 `nginx/nginx.conf`，取消 SSL 相关注释：

```nginx
server {
    listen 443 ssl;
    ssl_certificate /etc/nginx/certs/cert.pem;
    ssl_certificate_key /etc/nginx/certs/key.pem;
    # ...
}
```

3. 在 `docker-compose.yaml` 中取消证书卷挂载注释：

```yaml
volumes:
  - ./nginx/certs:/etc/nginx/certs
```

4. 重启 Nginx 容器：

```bash
docker compose restart nginx
```

## 生产环境注意事项

### 安全配置

- **修改 JWT Secret**：将 `jwt.secret` 替换为随机高强度字符串（建议 32 位以上）
- **修改默认管理员密码**：首次登录后立即修改
- **数据库密码**：修改 MySQL root 密码和业务用户密码
- **限制端口访问**：安全组/防火墙只开放 80 和 443 端口，3306 不对外开放

### 运维配置

- **定期备份数据库**：建议每日定时 `mysqldump` 备份
- **配置日志轮转**：应用已内置 lumberjack 日志轮转，按 `max_size_mb` / `max_backups` / `max_age_days` 自动管理
- **监控服务状态**：关注 `docker compose ps` 和后端日志输出
- **Gin 运行模式**：生产环境设置 `server.mode: "release"` 以获得更好的性能

### 资源建议

| 服务    | 最低配置       | 推荐配置       |
| ------- | -------------- | -------------- |
| MySQL   | 1 核 / 1GB     | 2 核 / 2GB     |
| Backend | 1 核 / 512MB   | 2 核 / 1GB     |
| Frontend| 1 核 / 512MB   | 1 核 / 512MB   |
| Nginx   | 0.5 核 / 256MB | 0.5 核 / 256MB |

## 常见问题

### Q: MySQL 连接失败

**排查步骤**：

1. 检查 MySQL 容器是否正常运行：`docker compose ps mysql`
2. 查看 MySQL 健康检查日志：`docker compose logs mysql`
3. 确认 MySQL healthcheck 已通过（`docker compose up` 会等待 healthcheck 通过后才启动 backend）
4. 如果是本地开发直连 MySQL，确认 `config.yaml` 中 `database.host` 为 `127.0.0.1`

### Q: 前端无法访问 API

**排查步骤**：

1. 检查 Nginx 容器状态：`docker compose ps nginx`
2. 查看 Nginx 日志：`docker compose logs nginx`
3. 查看后端日志：`docker compose logs backend`
4. 确认 `nginx.conf` 中 `upstream` 配置指向正确的服务名
5. 测试后端直连是否正常：`curl http://localhost:8080/api/v1/auth/login`

### Q: 工单推送失败

**排查步骤**：

1. 检查客户的 `api_endpoint` 是否可达（从后端容器内测试）
2. 查看后端日志中是否有推送错误信息
3. 确认客户 HMAC 签名配置正确
4. 检查 Worker Pool 配置是否合理（`worker.pool_size`）

### Q: 数据库迁移失败

**排查步骤**：

1. 检查数据库连接参数是否正确
2. 确认 migrate 工具版本兼容
3. 查看当前迁移状态：`migrate version`
4. 如需强制修正版本号：`migrate force <version>`

### Q: 如何查看 Worker 运行状态

Worker Pool 的日志输出在后端容器日志中，可通过以下命令查看：

```bash
docker compose logs -f backend | grep -i worker
```

### Q: 如何重启单个服务

```bash
# 重启后端
docker compose restart backend

# 重启 Nginx
docker compose restart nginx

# 重新构建并启动后端（代码更新后）
docker compose up -d --build backend
```
