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

### 后端

```bash
cd backend

# 复制配置文件
cp config.example.yaml config.yaml

# 安装依赖
go mod download

# 启动热重载开发（需安装 air）
make dev

# 或手动编译运行
make build
./bin/server
```

### 前端

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

### 数据库迁移

项目使用 [golang-migrate](https://github.com/golang-migrate/migrate) 管理数据库迁移。

```bash
# 安装 migrate 工具
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# 执行迁移
cd backend
migrate -path migrations -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" up

# 回退一步
migrate -path migrations -database "mysql://ticket:ticket_password@tcp(localhost:3306)/network_ticket" down 1
```

### 运行测试

```bash
cd backend
make test
```

---

## 配置说明

配置文件路径：`backend/config.yaml`。同时支持通过环境变量覆盖，前缀为 `NT_`，例如 `NT_DATABASE_HOST=mysql`、`NT_JWT_SECRET=xxx`。

完整配置字段说明见 [部署手册](deployment.md#配置说明)。
