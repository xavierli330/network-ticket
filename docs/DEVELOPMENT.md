# 开发者文档

本文档面向希望本地开发或二次开发的贡献者。

---

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23, Gin, sqlx, MySQL 8.0, Viper, zap |
| 前端 | Next.js 16, React 19, Tailwind CSS |
| 部署 | Docker Compose, Nginx |

---

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
│   │   ├── middleware/         # JWT / HMAC 签名 / Nonce 中间件
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
├── docs/                       # 文档
├── docker-compose.yaml
├── deploy.sh                   # 一键部署脚本
├── manage.sh                   # 日常管理脚本
└── Makefile
```

---

## 本地开发环境

本项目仅支持 **Docker Compose 开发**。所有服务（MySQL、后端、前端、Nginx）通过 `docker-compose.dev.yaml` 一键启动，代码通过 bind mount 挂载进容器，前后端均支持热重载，修改后即时生效。

**前置条件**：仅需安装 [Docker](https://docs.docker.com/get-docker/) 和 Docker Compose。

### 快速开始

```bash
# 1. 准备环境变量和配置文件
cp .env.example .env
cp backend/config.example.yaml backend/config.yaml

# 2. 启动开发环境（首次会自动构建镜像并安装依赖）
make dev
# 或
./manage.sh dev
```

启动成功后会看到：

```
✓ 开发环境已启动

  前端访问: http://localhost:3000
  后端 API: http://localhost:8080
  Nginx 代理: http://localhost
```

### 访问地址

| 端点 | 地址 | 用途 |
|------|------|------|
| 前端 dev server | http://localhost:3000 | Next.js HMR，浏览器直接访问前端 |
| 后端 API | http://localhost:8080 | 调试 API |
| Nginx 代理 | http://localhost | 模拟生产路由，`/api/*` 走后端，其余走前端 |

### 常用操作

```bash
# 查看后端实时日志（Air 编译、请求日志）
docker compose -f docker-compose.dev.yaml logs -f backend

# 查看前端实时日志（HMR、编译信息）
docker compose -f docker-compose.dev.yaml logs -f frontend

# 重启单个服务（不需要全部重建）
docker compose -f docker-compose.dev.yaml restart backend

# 停止整个开发环境（数据不丢失）
make dev-down
```

### 技术原理

**后端热重载**

- 容器镜像：`golang:1.26-alpine` + [Air](https://github.com/air-verse/air)
- 代码挂载：`./backend:/app`
- Air 监听 `.go`、`.yaml`、`.toml` 文件变更，自动执行 `go build` 并重启进程
- 编译产物输出到 `./backend/tmp/main`（已在 `.gitignore` 中）

**前端热重载**

- 容器镜像：`node:20-alpine`
- 代码挂载：`./frontend:/app`
- 运行 `npm run dev`，Next.js 提供 HMR（热模块替换）
- `node_modules` 安装在容器内，同时同步回宿主机目录

**数据库**

- 使用 MySQL 8.0 容器，数据保存在 Docker 命名卷 `mysql_data`
- 停止开发环境不会丢失数据，但 `docker compose down -v` 会清除
- 如需从生产数据导入：`docker compose -f docker-compose.dev.yaml exec -T mysql mysql -u root -p network_ticket < backup.sql`

### 数据库迁移

开发环境启动后，MySQL 容器已就绪。首次使用全新数据库时需要手动执行迁移。

**方式 A：使用宿主机上的 migrate CLI（推荐）**

```bash
# 安装工具（仅需一次）
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移（端口 3306 已映射到宿主机）
migrate -path backend/migrations \
  -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" up

# 回退一步
migrate -path backend/migrations \
  -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" down 1

# 查看当前版本
migrate -path backend/migrations \
  -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" version
```

**方式 B：使用临时容器（无需本地安装 migrate）**

```bash
# 先获取 .env 中的 root 密码
source .env

docker run --rm --network network-ticket_default \
  -v "$(pwd)/backend/migrations:/migrations" \
  migrate/migrate:latest \
  -path /migrations \
  -database "mysql://root:${MYSQL_ROOT_PASSWORD}@tcp(mysql:3306)/network_ticket?multiStatements=true" \
  up
```

### 开发环境配置

`backend/config.yaml` 中的 `database.host` 默认是 `127.0.0.1`，但在 Docker 开发环境中会被环境变量 `NT_DATABASE_HOST=mysql` 自动覆盖，无需手动修改。

如需调整其他配置（如 JWT Secret、日志级别），直接编辑 `backend/config.yaml` 即可，Air 会自动重启后端使配置生效。

### 运行测试

```bash
cd backend
make test
```

测试会执行单元测试并开启竞态检测（`-race`）。确保开发环境已启动且 MySQL 可访问。

### 常见问题

**Q: `make dev` 后后端一直重启，日志显示 "connection refused"**

A: 通常是 MySQL 还没就绪。Docker Compose 已配置 `depends_on` + `condition: service_healthy`，但首次拉取镜像可能较慢。等待 10~30 秒后 Air 会自动重试，或手动重启后端容器：`docker compose -f docker-compose.dev.yaml restart backend`。

**Q: 前端 HMR 不生效，浏览器需要手动刷新**

A: 检查是否通过 `http://localhost:3000` 直接访问前端。如果通过 Nginx (`http://localhost`) 访问，确保 `nginx.conf` 已配置 WebSocket 代理（`Upgrade` / `Connection` 头），当前配置已包含。

**Q: 修改了 `backend/config.yaml`，需要重启吗？**

A: Air 会检测 `.yaml` 变更并自动重启后端。若未自动生效，手动重启即可：`docker compose -f docker-compose.dev.yaml restart backend`。

**Q: 如何清理开发环境的所有数据？**

A: `docker compose -f docker-compose.dev.yaml down -v`，这会删除容器和命名卷 `mysql_data`（包括数据库数据）。代码文件不受影响。

---

## 配置说明

配置文件路径：`backend/config.yaml`。同时支持通过环境变量覆盖，前缀为 `NT_`，例如 `NT_DATABASE_HOST=mysql`、`NT_JWT_SECRET=xxx`。

完整配置字段说明见 [部署手册](deployment.md#配置说明)。
