# 部署手册

本文档面向运维人员和生产环境部署者。

---

## 环境要求

| 组件 | 版本要求 | 说明 |
|------|----------|------|
| Docker | 20.10+ | 容器运行时 |
| Docker Compose | v2+ | 服务编排 |

> 仅需 Docker 即可部署，无需在宿主机安装 Go 或 Node.js。

---

## 快速部署

```bash
# 1. 克隆项目
git clone <repo>
cd network-ticket

# 2. 一键部署
./deploy.sh
```

部署完成后：
- 前端：`http://localhost`
- 后端 API：`http://localhost/api/v1/`

---

## 日常运维

```bash
./manage.sh start      # 启动服务
./manage.sh stop       # 停止服务
./manage.sh restart    # 重启服务
./manage.sh status     # 查看服务状态
./manage.sh logs       # 查看实时日志
./manage.sh backup     # 备份数据库
./manage.sh update     # 更新代码后重新构建
./manage.sh uninstall  # 完全卸载（含数据）
```

---

## 配置说明

系统有两层配置：

1. **环境变量（`.env`）**：数据库密码、JWT Secret 等敏感信息通过 `.env` 文件管理，部署脚本会自动生成。
2. **后端配置文件（`backend/config.yaml`）**：服务行为配置，首次部署时自动从 `config.example.yaml` 复制。

> 修改 `.env` 或 `backend/config.yaml` 后需执行 `./manage.sh reload` 生效（`restart` 不会重新读取环境变量）。

同时支持通过环境变量覆盖配置项，前缀为 `NT_`，例如 `NT_DATABASE_HOST=mysql`、`NT_JWT_SECRET=xxx`。

### server - 服务器配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| port | int | 8080 | HTTP 监听端口 |
| mode | string | "debug" | Gin 运行模式 (`debug` / `release`) |

### database - 数据库配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| host | string | "127.0.0.1" | MySQL 主机地址 |
| port | int | 3306 | MySQL 端口 |
| user | string | "ticket" | 数据库用户名 |
| password | string | "ticket_password" | 数据库密码 |
| dbname | string | "network_ticket" | 数据库名称 |
| max_open_conns | int | 20 | 最大打开连接数 |
| max_idle_conns | int | 10 | 最大空闲连接数 |

### log - 日志配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| level | string | "debug" | 日志级别 (`debug`/`info`/`warn`/`error`) |
| format | string | "json" | 日志格式 (`json`/`text`) |
| file_path | string | "./logs/server.log" | 日志文件路径 |
| max_size_mb | int | 100 | 单个日志文件最大大小 (MB) |
| max_backups | int | 10 | 保留的旧日志文件最大数量 |
| max_age_days | int | 30 | 日志文件保留天数 |

### jwt - JWT 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| secret | string | "change-me-in-production" | JWT 签名密钥 |
| expire_hours | int | 24 | Token 过期时间 (小时) |

### security.nonce - Nonce 防重放配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| backend | string | "db" | Nonce 存储后端 (`db` / `file`) |
| ttl | string | "5m" | Nonce 有效期 |

### worker - Worker Pool 配置

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| pool_size | int | 10 | 并发 Worker 数量 |
| retry_max | int | 5 | 最大重试次数 |
| retry_base_interval | string | "1s" | 重试基础间隔 |
| retry_max_interval | string | "30s" | 重试最大间隔 (指数退避上限) |

---

## 默认账号

| 角色 | 用户名 | 密码 | 说明 |
|------|--------|------|------|
| 管理员 | admin | admin123 | **首次登录后请立即修改** |

---

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
./manage.sh restart
```

---

## 生产环境注意事项

### 安全配置

- **修改 JWT Secret**：编辑 `.env` 文件，将 `NT_JWT_SECRET` 替换为随机高强度字符串（建议 32 位以上）
- **修改数据库密码**：编辑 `.env` 文件，修改 `MYSQL_ROOT_PASSWORD` 和 `MYSQL_PASSWORD`（同时修改 `NT_DATABASE_PASSWORD` 保持一致），修改后执行 `./manage.sh reload`
- **修改默认管理员密码**：首次登录后立即在后台修改
- **限制端口访问**：安全组/防火墙只开放 80 和 443 端口，3306 不对外开放

### 运维配置

- **定期备份数据库**：建议每日定时执行 `./manage.sh backup`
- **配置日志轮转**：应用已内置 lumberjack 日志轮转，按 `max_size_mb` / `max_backups` / `max_age_days` 自动管理
- **监控服务状态**：关注 `./manage.sh status` 和后端日志输出
- **Gin 运行模式**：生产环境设置 `server.mode: "release"` 以获得更好的性能

### 资源建议

| 服务 | 最低配置 | 推荐配置 |
|------|----------|----------|
| MySQL | 1 核 / 1GB | 2 核 / 2GB |
| Backend | 1 核 / 512MB | 2 核 / 1GB |
| Frontend | 1 核 / 512MB | 1 核 / 512MB |
| Nginx | 0.5 核 / 256MB | 0.5 核 / 256MB |

---

## 常见问题

### Q: MySQL 连接失败

**排查步骤**：

1. 检查 MySQL 容器是否正常运行：`./manage.sh status`
2. 查看 MySQL 日志：`./manage.sh logs mysql`
3. 确认 MySQL healthcheck 已通过（`docker compose up` 会等待 healthcheck 通过后才启动 backend）
4. 如果是本地开发直连 MySQL，确认 `config.yaml` 中 `database.host` 为 `127.0.0.1`

### Q: 前端无法访问 API

**排查步骤**：

1. 检查 Nginx 容器状态：`./manage.sh status`
2. 查看 Nginx 日志：`./manage.sh logs nginx`
3. 查看后端日志：`./manage.sh logs backend`
4. 确认 `nginx.conf` 中 `upstream` 配置指向正确的服务名
5. 测试后端直连是否正常：`curl http://localhost:8080/api/v1/auth/login`

### Q: 工单推送失败

**排查步骤**：

1. 检查客户的 `api_endpoint` 是否可达（从后端容器内测试）
2. 查看后端日志中是否有推送错误信息
3. 确认客户 HMAC 签名配置正确
4. 检查 Worker Pool 配置是否合理（`worker.pool_size`）

### Q: 如何查看 Worker 运行状态

Worker Pool 的日志输出在后端容器日志中，可通过以下命令查看：

```bash
./manage.sh logs backend | grep -i worker
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
