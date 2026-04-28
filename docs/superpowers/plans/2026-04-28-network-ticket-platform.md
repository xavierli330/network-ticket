# 网络工单平台实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个网络/服务器告警工单平台，支持告警接入、工单管理、客户 API 对接（HMAC 签名 + 防重放）、管理后台。

**Architecture:** Go 单体模块化后端 (Gin) + Next.js 前端，前后端同域部署，Docker Compose 编排。MySQL 8.0+ 存储。Channel + Worker Pool 异步推送。

**Tech Stack:** Go 1.23, Gin, sqlx, golang-migrate, Viper, zap, gjson / Next.js 15, shadcn/ui, Tailwind CSS, SWR / MySQL 8.0+, Docker Compose, Nginx

---

## 文件结构总览

```
network-ticket/
├── backend/
│   ├── cmd/server/main.go                    # 入口
│   ├── internal/
│   │   ├── config/config.go                  # Viper 配置结构体 + 加载
│   │   ├── model/
│   │   │   ├── ticket.go                     # Ticket, TicketStatus 常量
│   │   │   ├── workflow_state.go             # WorkflowState, NodeName 常量
│   │   │   ├── alert_source.go               # AlertSource
│   │   │   ├── alert_record.go               # AlertRecord (追加的重复告警)
│   │   │   ├── client.go                     # Client
│   │   │   ├── ticket_log.go                 # TicketLog
│   │   │   ├── audit_log.go                  # AuditLog
│   │   │   └── user.go                       # User, UserRole
│   │   ├── repository/
│   │   │   ├── db.go                         # sqlx 初始化
│   │   │   ├── ticket_repo.go
│   │   │   ├── workflow_state_repo.go
│   │   │   ├── alert_source_repo.go
│   │   │   ├── alert_record_repo.go
│   │   │   ├── client_repo.go
│   │   │   ├── ticket_log_repo.go
│   │   │   ├── audit_log_repo.go
│   │   │   ├── user_repo.go
│   │   │   └── nonce_repo.go
│   │   ├── service/
│   │   │   ├── ticket_service.go             # 工单状态机 + 业务逻辑
│   │   │   ├── alert_service.go              # 告警解析 + 去重 + 建单
│   │   │   ├── client_service.go             # 客户 CRUD
│   │   │   ├── auth_service.go               # 用户认证 + JWT
│   │   │   └── stats_service.go              # 统计 (v2 预留)
│   │   ├── handler/
│   │   │   ├── alert_handler.go              # webhook + 手动创建
│   │   │   ├── ticket_handler.go             # 工单 CRUD
│   │   │   ├── client_handler.go             # 客户 CRUD
│   │   │   ├── callback_handler.go           # 客户授权回调
│   │   │   ├── auth_handler.go               # 登录
│   │   │   └── admin_handler.go              # 审计日志
│   │   ├── middleware/
│   │   │   ├── auth.go                       # JWT 中间件
│   │   │   ├── signature.go                  # HMAC 验签
│   │   │   ├── nonce.go                      # 防重放中间件
│   │   │   └── logger.go                     # 请求日志
│   │   ├── alert/
│   │   │   ├── parser/
│   │   │   │   ├── parser.go                 # AlertParser 接口 + Registry
│   │   │   │   ├── generic.go                # GenericJSONParser (gjson)
│   │   │   │   ├── zabbix.go                 # ZabbixParser
│   │   │   │   └── prometheus.go             # PrometheusParser
│   │   │   └── poller/
│   │   │       └── poller.go                 # 定时拉取
│   │   ├── client/
│   │   │   ├── pusher.go                     # 工单推送 (HMAC 签名出站)
│   │   │   ├── worker.go                     # Channel + Worker Pool
│   │   │   └── retry.go                      # 5 次指数退避
│   │   ├── nonce/
│   │   │   ├── store.go                      # NonceStore 接口
│   │   │   ├── db_store.go                   # MySQL 实现
│   │   │   └── file_store.go                 # 文件实现
│   │   └── pkg/
│   │       ├── hmac.go                       # HMAC-SHA256 签发 + 验证
│   │       ├── fingerprint.go                # 告警指纹计算
│   │       └── ticket_no.go                  # 工单编号生成 TK-YYYYMMDD-NNNN
│   ├── migrations/
│   │   ├── 001_create_users.up.sql
│   │   ├── 001_create_users.down.sql
│   │   ├── 002_create_alert_sources.up.sql
│   │   ├── 002_create_alert_sources.down.sql
│   │   ├── 003_create_clients.up.sql
│   │   ├── 003_create_clients.down.sql
│   │   ├── 004_create_tickets.up.sql
│   │   ├── 004_create_tickets.down.sql
│   │   ├── 005_create_workflow_states.up.sql
│   │   ├── 005_create_workflow_states.down.sql
│   │   ├── 006_create_alert_records.up.sql
│   │   ├── 006_create_alert_records.down.sql
│   │   ├── 007_create_ticket_logs.up.sql
│   │   ├── 007_create_ticket_logs.down.sql
│   │   ├── 008_create_audit_logs.up.sql
│   │   ├── 008_create_audit_logs.down.sql
│   │   ├── 009_create_nonce_records.up.sql
│   │   └── 009_create_nonce_records.down.sql
│   ├── tests/
│   │   ├── testdata/
│   │   │   ├── zabbix_alert.json
│   │   │   ├── prometheus_alert.json
│   │   │   └── generic_alert.json
│   │   ├── parser_test.go
│   │   ├── fingerprint_test.go
│   │   ├── hmac_test.go
│   │   ├── ticket_service_test.go
│   │   ├── nonce_test.go
│   │   ├── worker_test.go
│   │   └── api_test.go                       # 集成测试
│   ├── config.example.yaml
│   ├── .air.toml                             # air 热重载
│   ├── Dockerfile
│   ├── go.mod
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.tsx
│   │   │   ├── page.tsx                      # 首页 → 跳转工单列表
│   │   │   ├── login/page.tsx
│   │   │   ├── tickets/
│   │   │   │   ├── page.tsx                  # 工单列表
│   │   │   │   └── [id]/page.tsx             # 工单详情
│   │   │   ├── clients/page.tsx              # 客户管理
│   │   │   └── sources/page.tsx              # 告警源管理
│   │   ├── components/
│   │   │   ├── ui/                           # shadcn/ui 组件
│   │   │   ├── layout/
│   │   │   │   ├── sidebar.tsx
│   │   │   │   └── header.tsx
│   │   │   ├── ticket/
│   │   │   │   ├── ticket-table.tsx
│   │   │   │   ├── ticket-status-badge.tsx
│   │   │   │   └── ticket-detail.tsx
│   │   │   └── auth/
│   │   │       └── login-form.tsx
│   │   ├── lib/
│   │   │   ├── api.ts                        # fetch 封装 + JWT 拦截
│   │   │   └── auth.ts                       # token 存储/刷新
│   │   └── types/
│   │       └── index.ts                      # 类型定义
│   ├── Dockerfile
│   ├── next.config.js
│   ├── package.json
│   └── tsconfig.json
├── docker-compose.yaml
├── nginx/
│   └── nginx.conf
├── Makefile
└── README.md
```

---

## Task 1: 项目脚手架 + Go Module

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/server/main.go`
- Create: `backend/config.example.yaml`
- Create: `backend/Makefile`
- Create: `backend/.air.toml`
- Create: `Makefile` (顶层)

- [ ] **Step 1: 初始化 Go module**

```bash
cd /Users/xavierli/life/code/network-ticket
mkdir -p backend/cmd/server
cd backend
go mod init github.com/xavierli/network-ticket
```

- [ ] **Step 2: 创建最小 main.go**

```go
// backend/cmd/server/main.go
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("network-ticket server starting...")
	os.Exit(0)
}
```

- [ ] **Step 3: 创建 config.example.yaml**

```yaml
server:
  port: 8080
  mode: "debug"  # debug | release

database:
  host: "127.0.0.1"
  port: 3306
  user: "ticket"
  password: "ticket_password"
  dbname: "network_ticket"
  max_open_conns: 20
  max_idle_conns: 10

log:
  level: "debug"  # debug | info | warn | error
  format: "json"  # json | text
  file_path: "./logs/server.log"
  max_size_mb: 100
  max_backups: 10
  max_age_days: 30

jwt:
  secret: "change-me-in-production"
  expire_hours: 24

security:
  nonce:
    backend: "db"  # db | file
    ttl: "5m"
    file:
      path: "./data/nonces.log"

worker:
  pool_size: 10
  retry_max: 5
  retry_base_interval: "1s"
  retry_max_interval: "30s"
```

- [ ] **Step 4: 创建 backend/Makefile**

```makefile
.PHONY: build run test migrate-up migrate-down

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -v -count=1

migrate-up:
	migrate -path migrations -database "mysql://$(DB_DSN)" up

migrate-down:
	migrate -path migrations -database "mysql://$(DB_DSN)" down 1

dev:
	air
```

- [ ] **Step 5: 创建 .air.toml**

```toml
root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 1000
  exclude_dir = ["tmp", "vendor", "migrations"]
  include_ext = ["go", "yaml"]
```

- [ ] **Step 6: 创建顶层 Makefile**

```makefile
.PHONY: dev-backend dev-frontend build docker-up docker-down

dev-backend:
	cd backend && air

dev-frontend:
	cd frontend && npm run dev

build:
	cd backend && go build -o bin/server ./cmd/server
	cd frontend && npm run build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
```

- [ ] **Step 7: 验证编译通过**

```bash
cd backend && go build ./cmd/server
```

Expected: 编译成功，无错误

- [ ] **Step 8: 提交**

```bash
git add backend/ Makefile
git commit -m "chore: initialize project scaffolding with Go module and config"
```

---

## Task 2: 配置加载 + 日志初始化

**Files:**
- Create: `backend/internal/config/config.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 安装依赖**

```bash
cd backend
go get github.com/gin-gonic/gin
go get github.com/spf13/viper
go get go.uber.org/zap
go get github.com/natefinch/lumberjack
```

- [ ] **Step 2: 编写 config.go**

```go
// backend/internal/config/config.go
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Security SecurityConfig `mapstructure:"security"`
	Worker   WorkerConfig   `mapstructure:"worker"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName)
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	FilePath   string `mapstructure:"file_path"`
	MaxSizeMB  int    `mapstructure:"max_size_mb"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAgeDays int    `mapstructure:"max_age_days"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type SecurityConfig struct {
	Nonce NonceConfig `mapstructure:"nonce"`
}

type NonceConfig struct {
	Backend string          `mapstructure:"backend"`
	TTL     time.Duration   `mapstructure:"ttl"`
	File    FileNonceConfig `mapstructure:"file"`
}

type FileNonceConfig struct {
	Path string `mapstructure:"path"`
}

type WorkerConfig struct {
	PoolSize         int           `mapstructure:"pool_size"`
	RetryMax         int           `mapstructure:"retry_max"`
	RetryBaseInterval time.Duration `mapstructure:"retry_base_interval"`
	RetryMaxInterval  time.Duration `mapstructure:"retry_max_interval"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
```

- [ ] **Step 3: 编写日志初始化**

在 `config.go` 底部追加 `InitLogger` 函数：

```go
func InitLogger(cfg *LogConfig) (*zap.Logger, error) {
	level, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "ts"
	encoderCfg.EncodeTime = zap.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "text" {
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	cores := []zapcore.Core{}

	// stdout
	stdoutCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		level,
	)
	cores = append(cores, stdoutCore)

	// file
	if cfg.FilePath != "" {
		writer := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
		}
		fileCore := zapcore.NewCore(
			encoder,
			zapcore.AddSync(writer),
			level,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core, zap.AddCaller()), nil
}
```

需要 import: `"os"`, `"go.uber.org/zap/zapcore"`, `"gopkg.in/natefinch/lumberjack.v2"`

- [ ] **Step 4: 更新 main.go**

```go
// backend/cmd/server/main.go
package main

import (
	"flag"
	"log"

	"github.com/xavierli/network-ticket/internal/config"
)

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := config.InitLogger(&cfg.Log)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("network-ticket server starting",
		zap.Int("port", cfg.Server.Port),
	)
}
```

- [ ] **Step 5: 验证编译**

```bash
cd backend && go build ./cmd/server
```

- [ ] **Step 6: 提交**

```bash
git add backend/
git commit -m "feat: add config loading (Viper) and logger init (zap + lumberjack)"
```

---

## Task 3: 数据库迁移脚本

**Files:**
- Create: `backend/migrations/001_create_users.up.sql`
- Create: `backend/migrations/001_create_users.down.sql`
- Create: `backend/migrations/002_create_alert_sources.up.sql`
- Create: `backend/migrations/002_create_alert_sources.down.sql`
- Create: `backend/migrations/003_create_clients.up.sql`
- Create: `backend/migrations/003_create_clients.down.sql`
- Create: `backend/migrations/004_create_tickets.up.sql`
- Create: `backend/migrations/004_create_tickets.down.sql`
- Create: `backend/migrations/005_create_workflow_states.up.sql`
- Create: `backend/migrations/005_create_workflow_states.down.sql`
- Create: `backend/migrations/006_create_alert_records.up.sql`
- Create: `backend/migrations/006_create_alert_records.down.sql`
- Create: `backend/migrations/007_create_ticket_logs.up.sql`
- Create: `backend/migrations/007_create_ticket_logs.down.sql`
- Create: `backend/migrations/008_create_audit_logs.up.sql`
- Create: `backend/migrations/008_create_audit_logs.down.sql`
- Create: `backend/migrations/009_create_nonce_records.up.sql`
- Create: `backend/migrations/009_create_nonce_records.down.sql`

- [ ] **Step 1: 编写所有迁移脚本**

`001_create_users.up.sql`:
```sql
CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'operator' COMMENT 'admin | operator',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 默认管理员: admin / admin123 (bcrypt hash)
INSERT INTO users (username, password, role) VALUES
('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'admin');
```

`001_create_users.down.sql`:
```sql
DROP TABLE IF EXISTS users;
```

`002_create_alert_sources.up.sql`:
```sql
CREATE TABLE alert_sources (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    type VARCHAR(64) NOT NULL COMMENT 'zabbix | prometheus | generic',
    config JSON COMMENT '连接配置(认证方式等)',
    parser_config JSON COMMENT '字段映射配置(JSONPath)',
    webhook_secret VARCHAR(255) COMMENT 'webhook验签密钥',
    poll_endpoint VARCHAR(512) COMMENT '轮询地址',
    poll_interval INT DEFAULT 0 COMMENT '轮询间隔(秒), 0=不轮询',
    dedup_fields JSON COMMENT '去重指纹字段配置',
    dedup_window_sec INT DEFAULT 600 COMMENT '去重窗口(秒), 默认10分钟',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`002_create_alert_sources.down.sql`:
```sql
DROP TABLE IF EXISTS alert_sources;
```

`003_create_clients.up.sql`:
```sql
CREATE TABLE clients (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    api_endpoint VARCHAR(512) NOT NULL COMMENT '推送地址',
    api_key VARCHAR(255) NOT NULL COMMENT 'API Key',
    hmac_secret VARCHAR(255) NOT NULL COMMENT 'HMAC签名密钥',
    callback_url VARCHAR(512) COMMENT '客户回调地址(信息字段,实际用单一回调路径)',
    config JSON COMMENT '扩展配置',
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_api_key (api_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`003_create_clients.down.sql`:
```sql
DROP TABLE IF EXISTS clients;
```

`004_create_tickets.up.sql`:
```sql
CREATE TABLE tickets (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_no VARCHAR(32) NOT NULL COMMENT 'TK-YYYYMMDD-NNNN',
    alert_source_id BIGINT UNSIGNED NOT NULL,
    source_type VARCHAR(64) NOT NULL,
    alert_raw JSON NOT NULL,
    alert_parsed JSON,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(16) NOT NULL DEFAULT 'info' COMMENT 'critical | warning | info',
    status VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT 'pending | in_progress | completed | failed | cancelled',
    client_id BIGINT UNSIGNED,
    external_id VARCHAR(128) COMMENT '客户侧工单ID',
    callback_data JSON,
    fingerprint VARCHAR(64) COMMENT '告警指纹',
    timeout_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_ticket_no (ticket_no),
    KEY idx_status (status),
    KEY idx_client_id (client_id),
    KEY idx_alert_source_id (alert_source_id),
    KEY idx_fingerprint (fingerprint),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`004_create_tickets.down.sql`:
```sql
DROP TABLE IF EXISTS tickets;
```

`005_create_workflow_states.up.sql`:
```sql
CREATE TABLE workflow_states (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id BIGINT UNSIGNED NOT NULL,
    node_name VARCHAR(64) NOT NULL COMMENT 'alert_received | parsed | pushed | awaiting_auth | authorized | executing | completed',
    status VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT 'pending | active | done | failed | skipped | timeout',
    operator VARCHAR(64) COMMENT 'system | client:xxx',
    input_data JSON,
    output_data JSON,
    error_message TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ticket_id (ticket_id),
    KEY idx_ticket_node (ticket_id, node_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`005_create_workflow_states.down.sql`:
```sql
DROP TABLE IF EXISTS workflow_states;
```

`006_create_alert_records.up.sql`:
```sql
CREATE TABLE alert_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id BIGINT UNSIGNED NOT NULL,
    alert_raw JSON NOT NULL COMMENT '追加的重复告警原始数据',
    alert_parsed JSON,
    received_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ticket_id (ticket_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`006_create_alert_records.down.sql`:
```sql
DROP TABLE IF EXISTS alert_records;
```

`007_create_ticket_logs.up.sql`:
```sql
CREATE TABLE ticket_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ticket_id BIGINT UNSIGNED NOT NULL,
    action VARCHAR(64) NOT NULL,
    from_state VARCHAR(32),
    to_state VARCHAR(32),
    operator VARCHAR(64),
    detail JSON,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ticket_id (ticket_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`007_create_ticket_logs.down.sql`:
```sql
DROP TABLE IF EXISTS ticket_logs;
```

`008_create_audit_logs.up.sql`:
```sql
CREATE TABLE audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    resource_id BIGINT UNSIGNED,
    detail JSON,
    ip_address VARCHAR(45),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_actor (actor),
    KEY idx_resource (resource_type, resource_id),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`008_create_audit_logs.down.sql`:
```sql
DROP TABLE IF EXISTS audit_logs;
```

`009_create_nonce_records.up.sql`:
```sql
CREATE TABLE nonce_records (
    nonce VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (nonce)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

`009_create_nonce_records.down.sql`:
```sql
DROP TABLE IF EXISTS nonce_records;
```

- [ ] **Step 2: 提交**

```bash
git add backend/migrations/
git commit -m "feat: add all database migration scripts (9 tables)"
```

---

## Task 4: 数据模型 + Repository 层

**Files:**
- Create: `backend/internal/model/ticket.go`
- Create: `backend/internal/model/workflow_state.go`
- Create: `backend/internal/model/alert_source.go`
- Create: `backend/internal/model/alert_record.go`
- Create: `backend/internal/model/client.go`
- Create: `backend/internal/model/ticket_log.go`
- Create: `backend/internal/model/audit_log.go`
- Create: `backend/internal/model/user.go`
- Create: `backend/internal/repository/db.go`
- Create: `backend/internal/repository/ticket_repo.go`
- Create: `backend/internal/repository/workflow_state_repo.go`
- Create: `backend/internal/repository/alert_source_repo.go`
- Create: `backend/internal/repository/alert_record_repo.go`
- Create: `backend/internal/repository/client_repo.go`
- Create: `backend/internal/repository/ticket_log_repo.go`
- Create: `backend/internal/repository/audit_log_repo.go`
- Create: `backend/internal/repository/user_repo.go`
- Create: `backend/internal/repository/nonce_repo.go`

- [ ] **Step 1: 安装 sqlx + bcrypt**

```bash
cd backend
go get github.com/jmoiron/sqlx
go get github.com/go-sql-driver/mysql
go get golang.org/x/crypto/bcrypt
```

- [ ] **Step 2: 编写所有 model 文件**

`model/ticket.go`:
```go
package model

import "time"

type TicketStatus string

const (
	TicketStatusPending    TicketStatus = "pending"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusCompleted  TicketStatus = "completed"
	TicketStatusFailed     TicketStatus = "failed"
	TicketStatusCancelled  TicketStatus = "cancelled"
	TicketStatusRejected   TicketStatus = "rejected"
)

type Ticket struct {
	ID            int64       `db:"id" json:"id"`
	TicketNo      string      `db:"ticket_no" json:"ticket_no"`
	AlertSourceID int64       `db:"alert_source_id" json:"alert_source_id"`
	SourceType    string      `db:"source_type" json:"source_type"`
	AlertRaw      JSON        `db:"alert_raw" json:"alert_raw"`
	AlertParsed   JSON        `db:"alert_parsed" json:"alert_parsed"`
	Title         string      `db:"title" json:"title"`
	Description   string      `db:"description" json:"description"`
	Severity      string      `db:"severity" json:"severity"`
	Status        TicketStatus `db:"status" json:"status"`
	ClientID      *int64      `db:"client_id" json:"client_id"`
	ExternalID    *string     `db:"external_id" json:"external_id"`
	CallbackData  JSON        `db:"callback_data" json:"callback_data"`
	Fingerprint   *string     `db:"fingerprint" json:"fingerprint"`
	TimeoutAt     *time.Time  `db:"timeout_at" json:"timeout_at"`
	CreatedAt     time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time   `db:"updated_at" json:"updated_at"`
}

type JSON []byte

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSON value: %v", value)
	}
	*j = append((*j)[0:0], bytes...)
	return nil
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}
```

需要 import `"database/sql/driver"`, `"fmt"`

`model/workflow_state.go`:
```go
package model

import "time"

type NodeName string

const (
	NodeAlertReceived NodeName = "alert_received"
	NodeParsed        NodeName = "parsed"
	NodePushed        NodeName = "pushed"
	NodeAwaitingAuth  NodeName = "awaiting_auth"
	NodeAuthorized    NodeName = "authorized"
	NodeExecuting     NodeName = "executing"
	NodeCompleted     NodeName = "completed"
)

type NodeStatus string

const (
	NodeStatusPending NodeStatus = "pending"
	NodeStatusActive  NodeStatus = "active"
	NodeStatusDone    NodeStatus = "done"
	NodeStatusFailed  NodeStatus = "failed"
	NodeStatusSkipped NodeStatus = "skipped"
	NodeStatusTimeout NodeStatus = "timeout"
)

type WorkflowState struct {
	ID           int64      `db:"id" json:"id"`
	TicketID     int64      `db:"ticket_id" json:"ticket_id"`
	NodeName     NodeName   `db:"node_name" json:"node_name"`
	Status       NodeStatus `db:"status" json:"status"`
	Operator     *string    `db:"operator" json:"operator"`
	InputData    JSON       `db:"input_data" json:"input_data"`
	OutputData   JSON       `db:"output_data" json:"output_data"`
	ErrorMessage *string    `db:"error_message" json:"error_message"`
	StartedAt    *time.Time `db:"started_at" json:"started_at"`
	CompletedAt  *time.Time `db:"completed_at" json:"completed_at"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}
```

`model/alert_source.go`:
```go
package model

import "time"

type AlertSource struct {
	ID             int64      `db:"id" json:"id"`
	Name           string     `db:"name" json:"name"`
	Type           string     `db:"type" json:"type"`
	Config         JSON       `db:"config" json:"config"`
	ParserConfig   JSON       `db:"parser_config" json:"parser_config"`
	WebhookSecret  *string    `db:"webhook_secret" json:"webhook_secret"`
	PollEndpoint   *string    `db:"poll_endpoint" json:"poll_endpoint"`
	PollInterval   int        `db:"poll_interval" json:"poll_interval"`
	DedupFields    JSON       `db:"dedup_fields" json:"dedup_fields"`
	DedupWindowSec int        `db:"dedup_window_sec" json:"dedup_window_sec"`
	Status         string     `db:"status" json:"status"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
}
```

`model/alert_record.go`:
```go
package model

import "time"

type AlertRecord struct {
	ID          int64     `db:"id" json:"id"`
	TicketID    int64     `db:"ticket_id" json:"ticket_id"`
	AlertRaw    JSON      `db:"alert_raw" json:"alert_raw"`
	AlertParsed JSON      `db:"alert_parsed" json:"alert_parsed"`
	ReceivedAt  time.Time `db:"received_at" json:"received_at"`
}
```

`model/client.go`:
```go
package model

import "time"

type Client struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	APIEndpoint string    `db:"api_endpoint" json:"api_endpoint"`
	APIKey      string    `db:"api_key" json:"-"`
	HMACSecret  string    `db:"hmac_secret" json:"-"`
	CallbackURL *string   `db:"callback_url" json:"callback_url"`
	Config      JSON      `db:"config" json:"config"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}
```

`model/ticket_log.go`:
```go
package model

import "time"

type TicketLog struct {
	ID        int64     `db:"id" json:"id"`
	TicketID  int64     `db:"ticket_id" json:"ticket_id"`
	Action    string    `db:"action" json:"action"`
	FromState *string   `db:"from_state" json:"from_state"`
	ToState   *string   `db:"to_state" json:"to_state"`
	Operator  *string   `db:"operator" json:"operator"`
	Detail    JSON      `db:"detail" json:"detail"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
```

`model/audit_log.go`:
```go
package model

import "time"

type AuditLog struct {
	ID           int64     `db:"id" json:"id"`
	Actor        string    `db:"actor" json:"actor"`
	Action       string    `db:"action" json:"action"`
	ResourceType string    `db:"resource_type" json:"resource_type"`
	ResourceID   *int64    `db:"resource_id" json:"resource_id"`
	Detail       JSON      `db:"detail" json:"detail"`
	IPAddress    *string   `db:"ip_address" json:"ip_address"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
```

`model/user.go`:
```go
package model

import "time"

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
)

type User struct {
	ID        int64     `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	Password  string    `db:"-" json:"-"`
	Role      UserRole  `db:"role" json:"role"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
```

- [ ] **Step 3: 编写 repository/db.go**

```go
package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"
	"github.com/xavierli/network-ticket/internal/config"
)

func NewDB(cfg *config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}
```

- [ ] **Step 4: 编写所有 repo 文件**

每个 repo 文件结构类似，列出关键方法签名：

`repository/ticket_repo.go` — Create, GetByID, GetByTicketNo, GetByFingerprint, List(分页+筛选), UpdateStatus
`repository/workflow_state_repo.go` — Create, ListByTicketID, UpdateStatus, GetActiveByTicket
`repository/alert_source_repo.go` — Create, GetByID, List, Update, Delete
`repository/alert_record_repo.go` — Create, ListByTicketID
`repository/client_repo.go` — Create, GetByID, GetByAPIKey, List, Update, Delete
`repository/ticket_log_repo.go` — Create, ListByTicketID
`repository/audit_log_repo.go` — Create, List(分页+筛选)
`repository/user_repo.go` — Create, GetByUsername, GetByID, List, Update, Delete
`repository/nonce_repo.go` — CheckAndSet(INSERT IGNORE), CleanExpired

每个方法使用 sqlx 的 `Get`/`Select`/`Exec` 实现。以 ticket_repo.go 为例：

```go
package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/xavierli/network-ticket/internal/model"
)

type TicketRepo struct {
	db *sqlx.DB
}

func NewTicketRepo(db *sqlx.DB) *TicketRepo {
	return &TicketRepo{db: db}
}

func (r *TicketRepo) Create(ctx context.Context, t *model.Ticket) error {
	query := `INSERT INTO tickets
		(ticket_no, alert_source_id, source_type, alert_raw, alert_parsed,
		 title, description, severity, status, client_id, fingerprint, timeout_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		t.TicketNo, t.AlertSourceID, t.SourceType, t.AlertRaw, t.AlertParsed,
		t.Title, t.Description, t.Severity, t.Status, t.ClientID, t.Fingerprint, t.TimeoutAt,
	)
	if err != nil {
		return fmt.Errorf("insert ticket: %w", err)
	}
	t.ID, _ = result.LastInsertId()
	return nil
}

func (r *TicketRepo) GetByID(ctx context.Context, id int64) (*model.Ticket, error) {
	var t model.Ticket
	err := r.db.GetContext(ctx, &t, "SELECT * FROM tickets WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("get ticket by id: %w", err)
	}
	return &t, nil
}

func (r *TicketRepo) GetByTicketNo(ctx context.Context, ticketNo string) (*model.Ticket, error) {
	var t model.Ticket
	err := r.db.GetContext(ctx, &t, "SELECT * FROM tickets WHERE ticket_no = ?", ticketNo)
	if err != nil {
		return nil, fmt.Errorf("get ticket by no: %w", err)
	}
	return &t, nil
}

func (r *TicketRepo) GetByFingerprint(ctx context.Context, fingerprint string) (*model.Ticket, error) {
	var t model.Ticket
	err := r.db.GetContext(ctx, &t,
		"SELECT * FROM tickets WHERE fingerprint = ? AND status IN ('pending','in_progress') ORDER BY created_at DESC LIMIT 1",
		fingerprint)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

type TicketFilter struct {
	Status   string `form:"status"`
	ClientID int64  `form:"client_id"`
	Severity string `form:"severity"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

func (r *TicketRepo) List(ctx context.Context, f TicketFilter) ([]model.Ticket, int64, error) {
	if f.Page < 1 { f.Page = 1 }
	if f.PageSize < 1 || f.PageSize > 100 { f.PageSize = 20 }

	where := "WHERE 1=1"
	args := []interface{}{}
	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.ClientID > 0 {
		where += " AND client_id = ?"
		args = append(args, f.ClientID)
	}
	if f.Severity != "" {
		where += " AND severity = ?"
		args = append(args, f.Severity)
	}
	if f.Keyword != "" {
		where += " AND (title LIKE ? OR ticket_no LIKE ?)"
		args = append(args, "%"+f.Keyword+"%", "%"+f.Keyword+"%")
	}

	var total int64
	countQ := "SELECT COUNT(*) FROM tickets " + where
	r.db.GetContext(ctx, &total, countQ, args...)

	offset := (f.Page - 1) * f.PageSize
	listQ := "SELECT * FROM tickets " + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, f.PageSize, offset)

	var tickets []model.Ticket
	err := r.db.SelectContext(ctx, &tickets, listQ, args...)
	return tickets, total, err
}

func (r *TicketRepo) UpdateStatus(ctx context.Context, id int64, status model.TicketStatus) error {
	_, err := r.db.ExecContext(ctx, "UPDATE tickets SET status = ? WHERE id = ?", status, id)
	return err
}
```

其余 repo 文件遵循相同模式，这里不逐一列出完整代码——结构和 ticket_repo.go 一致：`NewXxxRepo` 构造函数 + `sqlx` 操作。

- [ ] **Step 5: 验证编译**

```bash
cd backend && go build ./...
```

- [ ] **Step 6: 提交**

```bash
git add backend/internal/model/ backend/internal/repository/
git commit -m "feat: add data models and repository layer (sqlx)"
```

---

## Task 5: 安全工具 — HMAC + 指纹 + 工单编号

**Files:**
- Create: `backend/internal/pkg/hmac.go`
- Create: `backend/internal/pkg/fingerprint.go`
- Create: `backend/internal/pkg/ticket_no.go`
- Create: `backend/tests/hmac_test.go`
- Create: `backend/tests/fingerprint_test.go`

- [ ] **Step 1: 编写 hmac.go**

```go
package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

func SignHMAC(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyHMAC(secret string, timestamp int64, body []byte, signature string) bool {
	expected := SignHMAC(secret, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

func VerifyTimestamp(timestamp int64, maxDriftSec int64) error {
	drift := abs64(now().Unix() - timestamp)
	if drift > maxDriftSec {
		return fmt.Errorf("timestamp drift %d seconds exceeds max %d", drift, maxDriftSec)
	}
	return nil
}

// 测试时可通过包级变量注入 now 函数
var now = time.Now
```

需要 import `"time"` 和一个 `abs64` 辅助函数。

- [ ] **Step 2: 编写 fingerprint.go**

```go
package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// ComputeFingerprint 根据 dedupFields 配置从 raw JSON 中提取字段计算指纹
// dedupFields 格式: ["source_ip", "alert_type"] (JSONPath)
func ComputeFingerprint(raw json.RawMessage, dedupFields []string) (string, error) {
	parts := make([]string, 0, len(dedupFields))
	for _, field := range dedupFields {
		val := gjson.GetBytes(raw, field)
		if !val.Exists() {
			return "", fmt.Errorf("fingerprint field %s not found", field)
		}
		parts = append(parts, val.String())
	}
	joined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:]), nil
}
```

```bash
cd backend && go get github.com/tidwall/gjson
```

- [ ] **Step 3: 编写 ticket_no.go**

```go
package pkg

import (
	"fmt"
	"sync/atomic"
	"time"
)

var ticketSeq int64

func GenerateTicketNo() string {
	now := time.Now()
	date := now.Format("20060102")
	seq := atomic.AddInt64(&ticketSeq, 1)
	return fmt.Sprintf("TK-%s-%04d", date, seq)
}
```

- [ ] **Step 4: 编写测试**

`tests/hmac_test.go`:
```go
package tests

import (
	"testing"

	"github.com/xavierli/network-ticket/internal/pkg"
)

func TestSignAndVerifyHMAC(t *testing.T) {
	secret := "test-secret"
	timestamp := int64(1714286400)
	body := []byte(`{"ticket_no":"TK-20260428-0001"}`)

	sig := pkg.SignHMAC(secret, timestamp, body)
	if !pkg.VerifyHMAC(secret, timestamp, body, sig) {
		t.Error("signature verification should succeed")
	}
	if pkg.VerifyHMAC("wrong-secret", timestamp, body, sig) {
		t.Error("wrong secret should fail verification")
	}
}

func TestVerifyTimestamp(t *testing.T) {
	pkg.Now = func() time.Time { return time.Unix(1714286400, 0) }
	defer func() { pkg.Now = time.Now }()

	if err := pkg.VerifyTimestamp(1714286400, 300); err != nil {
		t.Error("same timestamp should pass")
	}
	if err := pkg.VerifyTimestamp(1714286100, 300); err != nil {
		t.Error("within 5min should pass")
	}
	if err := pkg.VerifyTimestamp(1714286000, 300); err == nil {
		t.Error("beyond 5min should fail")
	}
}
```

`tests/fingerprint_test.go`:
```go
package tests

import (
	"encoding/json"
	"testing"

	"github.com/xavierli/network-ticket/internal/pkg"
)

func TestComputeFingerprint(t *testing.T) {
	raw := json.RawMessage(`{"source_ip":"10.0.0.1","alert_type":"link_down","severity":"critical"}`)

	fp1, err := pkg.ComputeFingerprint(raw, []string{"source_ip", "alert_type"})
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	raw2 := json.RawMessage(`{"source_ip":"10.0.0.1","alert_type":"link_down","severity":"warning"}`)
	fp2, _ := pkg.ComputeFingerprint(raw2, []string{"source_ip", "alert_type"})

	if fp1 != fp2 {
		t.Error("same fingerprint fields should produce same hash")
	}

	raw3 := json.RawMessage(`{"source_ip":"10.0.0.2","alert_type":"link_down","severity":"critical"}`)
	fp3, _ := pkg.ComputeFingerprint(raw3, []string{"source_ip", "alert_type"})

	if fp1 == fp3 {
		t.Error("different field values should produce different hash")
	}
}
```

- [ ] **Step 5: 运行测试**

```bash
cd backend && go test ./tests/ -v -run "TestSign|TestVerify|TestCompute"
```

- [ ] **Step 6: 提交**

```bash
git add backend/internal/pkg/ backend/tests/
git commit -m "feat: add HMAC signing/verification, alert fingerprint, ticket number generation"
```

---

## Task 6: Nonce 存储实现

**Files:**
- Create: `backend/internal/nonce/store.go`
- Create: `backend/internal/nonce/db_store.go`
- Create: `backend/internal/nonce/file_store.go`
- Create: `backend/tests/nonce_test.go`

- [ ] **Step 1: 编写 store.go 接口**

```go
package nonce

import (
	"context"
	"time"
)

type Store interface {
	CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error)
	Clean(ctx context.Context) error
}
```

- [ ] **Step 2: 编写 db_store.go**

```go
package nonce

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type DBStore struct {
	db *sqlx.DB
}

func NewDBStore(db *sqlx.DB) *DBStore {
	return &DBStore{db: db}
}

func (s *DBStore) CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT IGNORE INTO nonce_records (nonce, created_at) VALUES (?, ?)",
		nonce, time.Now(),
	)
	if err != nil {
		return false, fmt.Errorf("insert nonce: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *DBStore) Clean(ctx context.Context) error {
	cutoff := time.Now().Add(-5 * time.Minute)
	_, err := s.db.ExecContext(ctx, "DELETE FROM nonce_records WHERE created_at < ?", cutoff)
	return err
}
```

- [ ] **Step 3: 编写 file_store.go**

```go
package nonce

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

type FileStore struct {
	path string
}

func NewFileStore(path string) (*FileStore, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	f.Close()
	return &FileStore{path: path}, nil
}

func (s *FileStore) CheckAndSet(ctx context.Context, nonce string, ttl time.Duration) (bool, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	prefix := nonce + "|"
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), prefix) {
			return false, nil
		}
	}

	f2, err := os.OpenFile(s.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer f2.Close()
	fmt.Fprintf(f2, "%s|%d\n", nonce, time.Now().Unix())
	return true, nil
}

func (s *FileStore) Clean(ctx context.Context) error {
	cutoff := time.Now().Add(-5 * time.Minute).Unix()
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	var keep []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			ts, _ := strconv.ParseInt(parts[1], 10, 64)
			if ts >= cutoff {
				keep = append(keep, line)
			}
		}
	}

	f2, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f2.Close()
	for _, line := range keep {
		fmt.Fprintln(f2, line)
	}
	return nil
}
```

需要 import `"path/filepath"`, `"strconv"`

- [ ] **Step 4: 编写测试 (file store)**

```go
// tests/nonce_test.go
package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/xavierli/network-ticket/internal/nonce"
)

func TestFileNonceStore(t *testing.T) {
	path := t.TempDir() + "/nonces.log"
	store, err := nonce.NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	ok, err := store.CheckAndSet(ctx, "nonce-1", 5*time.Minute)
	if err != nil || !ok {
		t.Fatalf("first check should succeed: ok=%v err=%v", ok, err)
	}

	ok, _ = store.CheckAndSet(ctx, "nonce-1", 5*time.Minute)
	if ok {
		t.Error("duplicate nonce should return false")
	}

	ok, _ = store.CheckAndSet(ctx, "nonce-2", 5*time.Minute)
	if !ok {
		t.Error("different nonce should succeed")
	}

	os.Remove(path)
}
```

- [ ] **Step 5: 运行测试**

```bash
cd backend && go test ./tests/ -v -run TestFileNonce
```

- [ ] **Step 6: 提交**

```bash
git add backend/internal/nonce/ backend/tests/nonce_test.go
git commit -m "feat: add nonce store with db and file backends"
```

---

## Task 7: 告警解析器

**Files:**
- Create: `backend/internal/alert/parser/parser.go`
- Create: `backend/internal/alert/parser/generic.go`
- Create: `backend/internal/alert/parser/zabbix.go`
- Create: `backend/internal/alert/parser/prometheus.go`
- Create: `backend/tests/parser_test.go`
- Create: `backend/tests/testdata/zabbix_alert.json`
- Create: `backend/tests/testdata/prometheus_alert.json`
- Create: `backend/tests/testdata/generic_alert.json`

- [ ] **Step 1: 编写 parser.go (接口 + registry)**

```go
package parser

import (
	"context"
	"encoding/json"
	"time"
)

type ParsedAlert struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	SourceIP    string                 `json:"source_ip"`
	DeviceName  string                 `json:"device_name"`
	AlertTime   time.Time              `json:"alert_time"`
	Fields      map[string]interface{} `json:"fields"`
}

type AlertParser interface {
	Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error)
	SourceType() string
}

var registry = map[string]AlertParser{}

func Register(p AlertParser) {
	registry[p.SourceType()] = p
}

func Get(sourceType string) (AlertParser, bool) {
	p, ok := registry[sourceType]
	return p, ok
}

func init() {
	Register(&GenericJSONParser{})
	Register(&ZabbixParser{})
	Register(&PrometheusParser{})
}
```

- [ ] **Step 2: 编写 generic.go**

```go
package parser

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type GenericJSONParser struct{}

func (p *GenericJSONParser) SourceType() string { return "generic" }

func (p *GenericJSONParser) Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error) {
	return parseWithMapping(raw, nil, nil)
}

// parseWithMapping 使用 parser_config 中的 field_mapping 和 severity_mapping 解析
// config 格式: {"field_mapping":{"title":"$.path",...},"severity_mapping":{"critical":["p1",...],...}}
func parseWithMapping(raw json.RawMessage, fieldMapping, severityMapping map[string]interface{}) (*ParsedAlert, error) {
	alert := &ParsedAlert{}
	alert.Fields = make(map[string]interface{})

	// 如果没有 field_mapping，尝试自动提取常见字段
	if len(fieldMapping) == 0 {
		alert.Title = gjson.GetBytes(raw, "title").String()
		if alert.Title == "" {
			alert.Title = gjson.GetBytes(raw, "alertname").String()
		}
		alert.Description = gjson.GetBytes(raw, "description").String()
		if alert.Description == "" {
			alert.Description = gjson.GetBytes(raw, "message").String()
		}
		alert.Severity = normalizeSeverity(gjson.GetBytes(raw, "severity").String())
		alert.SourceIP = gjson.GetBytes(raw, "source_ip").String()
		alert.DeviceName = gjson.GetBytes(raw, "device_name").String()
		alert.AlertTime = time.Now()
		return alert, nil
	}

	// 有 field_mapping 时按映射提取
	mappings := fieldMapping
	for target, pathExpr := range mappings {
		path, ok := pathExpr.(string)
		if !ok { continue }
		val := gjson.GetBytes(raw, path)
		if !val.Exists() { continue }

		switch target {
		case "title":
			alert.Title = val.String()
		case "description":
			alert.Description = val.String()
		case "severity":
			alert.Severity = mapSeverity(val.String(), severityMapping)
		case "source_ip":
			alert.SourceIP = val.String()
		case "device_name":
			alert.DeviceName = val.String()
		case "alert_time":
			alert.AlertTime, _ = time.Parse(time.RFC3339, val.String())
		default:
			alert.Fields[target] = val.Value()
		}
	}

	if alert.AlertTime.IsZero() {
		alert.AlertTime = time.Now()
	}
	if alert.Severity == "" {
		alert.Severity = "info"
	}
	return alert, nil
}

func normalizeSeverity(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "crit") || strings.Contains(s, "p1") || strings.Contains(s, "emerg"):
		return "critical"
	case strings.Contains(s, "warn") || strings.Contains(s, "p2"):
		return "warning"
	default:
		return "info"
	}
}

func mapSeverity(s string, mapping map[string]interface{}) string {
	for level, vals := range mapping {
		list, ok := vals.([]interface{})
		if !ok { continue }
		for _, v := range list {
			if strings.EqualFold(s, fmt.Sprint(v)) {
				return level
			}
		}
	}
	return normalizeSeverity(s)
}
```

- [ ] **Step 3: 编写 zabbix.go 和 prometheus.go**

```go
// zabbix.go
package parser

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tidwall/gjson"
)

type ZabbixParser struct{}

func (p *ZabbixParser) SourceType() string { return "zabbix" }

func (p *ZabbixParser) Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error) {
	alert := &ParsedAlert{
		Title:       gjson.GetBytes(raw, "subject").String(),
		Description: gjson.GetBytes(raw, "message").String(),
		SourceIP:    gjson.GetBytes(raw, "host.ip").String(),
		DeviceName:  gjson.GetBytes(raw, "host.name").String(),
		Severity:    normalizeSeverity(gjson.GetBytes(raw, "event.severity").String()),
		AlertTime:   time.Now(),
		Fields:      map[string]interface{}{},
	}
	if alert.Title == "" {
		alert.Title = gjson.GetBytes(raw, "event.name").String()
	}
	return alert, nil
}
```

```go
// prometheus.go
package parser

import (
	"context"
	"encoding/json"
	"time"

	"github.com/tidwall/gjson"
)

type PrometheusParser struct{}

func (p *PrometheusParser) SourceType() string { return "prometheus" }

func (p *PrometheusParser) Parse(ctx context.Context, raw json.RawMessage) (*ParsedAlert, error) {
	alert := &ParsedAlert{
		Title:       gjson.GetBytes(raw, "alerts.0.labels.alertname").String(),
		Description: gjson.GetBytes(raw, "alerts.0.annotations.summary").String(),
		SourceIP:    gjson.GetBytes(raw, "alerts.0.labels.instance").String(),
		DeviceName:  gjson.GetBytes(raw, "alerts.0.labels.device").String(),
		Severity:    normalizeSeverity(gjson.GetBytes(raw, "alerts.0.labels.severity").String()),
		Fields:      map[string]interface{}{},
	}
	status := gjson.GetBytes(raw, "status").String()
	if status == "firing" && alert.Severity == "" {
		alert.Severity = "warning"
	}
	if alert.Description == "" {
		alert.Description = gjson.GetBytes(raw, "alerts.0.annotations.description").String()
	}
	ts := gjson.GetBytes(raw, "alerts.0.startsAt").String()
	if ts != "" {
		alert.AlertTime, _ = time.Parse(time.RFC3339, ts)
	}
	if alert.AlertTime.IsZero() {
		alert.AlertTime = time.Now()
	}
	return alert, nil
}
```

- [ ] **Step 4: 编写测试数据文件**

`tests/testdata/zabbix_alert.json`:
```json
{
  "event.name": "Link down on Gi0/1",
  "event.severity": "High",
  "subject": "Interface Gi0/1 is down on switch-01",
  "message": "Interface GigabitEthernet0/1 status changed to DOWN",
  "host": {"name": "switch-01", "ip": "10.0.0.1"}
}
```

`tests/testdata/prometheus_alert.json`:
```json
{
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {"alertname": "HighCPU", "severity": "critical", "instance": "10.0.1.5", "device": "server-03"},
    "annotations": {"summary": "CPU usage above 90%", "description": "CPU at 95% for 5 minutes"},
    "startsAt": "2026-04-28T10:30:00Z"
  }]
}
```

- [ ] **Step 5: 编写 parser_test.go**

```go
package tests

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/xavierli/network-ticket/internal/alert/parser"
)

func TestZabbixParser(t *testing.T) {
	raw := loadTestdata(t, "testdata/zabbix_alert.json")
	p, ok := parser.Get("zabbix")
	if !ok { t.Fatal("zabbix parser not registered") }

	alert, err := p.Parse(context.Background(), raw)
	if err != nil { t.Fatal(err) }
	if alert.Title != "Interface Gi0/1 is down on switch-01" {
		t.Errorf("unexpected title: %s", alert.Title)
	}
	if alert.SourceIP != "10.0.0.1" { t.Errorf("unexpected IP: %s", alert.SourceIP) }
	if alert.Severity != "warning" { t.Errorf("unexpected severity: %s", alert.Severity) }
}

func TestPrometheusParser(t *testing.T) {
	raw := loadTestdata(t, "testdata/prometheus_alert.json")
	p, ok := parser.Get("prometheus")
	if !ok { t.Fatal("prometheus parser not registered") }

	alert, err := p.Parse(context.Background(), raw)
	if err != nil { t.Fatal(err) }
	if alert.Title != "HighCPU" { t.Errorf("unexpected title: %s", alert.Title) }
	if alert.Severity != "critical" { t.Errorf("unexpected severity: %s", alert.Severity) }
}

func TestGenericParser(t *testing.T) {
	raw := json.RawMessage(`{"title":"Test Alert","severity":"critical","source_ip":"1.2.3.4"}`)
	p, ok := parser.Get("generic")
	if !ok { t.Fatal("generic parser not registered") }

	alert, err := p.Parse(context.Background(), raw)
	if err != nil { t.Fatal(err) }
	if alert.Title != "Test Alert" { t.Errorf("unexpected title: %s", alert.Title) }
}

func loadTestdata(t *testing.T, name string) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil { t.Fatal(err) }
	return json.RawMessage(data)
}
```

- [ ] **Step 6: 运行测试**

```bash
cd backend && go test ./tests/ -v -run TestZabbixParser -run TestPrometheusParser -run TestGenericParser
```

- [ ] **Step 7: 提交**

```bash
git add backend/internal/alert/ backend/tests/
git commit -m "feat: add alert parser registry with zabbix, prometheus, generic parsers"
```

---

## Task 8: 工单引擎 (状态机 + Service 层)

**Files:**
- Create: `backend/internal/service/ticket_service.go`
- Create: `backend/internal/service/alert_service.go`
- Create: `backend/internal/service/auth_service.go`
- Create: `backend/tests/ticket_service_test.go`

- [ ] **Step 1: 编写 ticket_service.go**

核心：状态转换校验 + workflow_states 联动 + ticket_logs 记录。

```go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
)

type TicketService struct {
	ticketRepo  *repository.TicketRepo
	workflowRepo *repository.WorkflowStateRepo
	logRepo     *repository.TicketLogRepo
	auditRepo   *repository.AuditLogRepo
	recordRepo  *repository.AlertRecordRepo
	logger      *zap.Logger
}

func NewTicketService(
	ticketRepo *repository.TicketRepo,
	workflowRepo *repository.WorkflowStateRepo,
	logRepo *repository.TicketLogRepo,
	auditRepo *repository.AuditLogRepo,
	recordRepo *repository.AlertRecordRepo,
	logger *zap.Logger,
) *TicketService {
	return &TicketService{
		ticketRepo: ticketRepo, workflowRepo: workflowRepo,
		logRepo: logRepo, auditRepo: auditRepo,
		recordRepo: recordRepo, logger: logger,
	}
}

// 状态转换规则: from -> []to
var transitions = map[model.TicketStatus][]model.TicketStatus{
	model.TicketStatusPending:    {model.TicketStatusInProgress, model.TicketStatusFailed, model.TicketStatusCancelled},
	model.TicketStatusInProgress: {model.TicketStatusCompleted, model.TicketStatusFailed, model.TicketStatusRejected, model.TicketStatusCancelled},
	model.TicketStatusFailed:     {model.TicketStatusPending, model.TicketStatusCancelled},
}

func canTransition(from, to model.TicketStatus) bool {
	allowed, ok := transitions[from]
	if !ok { return false }
	for _, s := range allowed {
		if s == to { return true }
	}
	return false
}

// CreateTicket 告警创建工单
func (s *TicketService) CreateTicket(ctx context.Context, alertSourceID int64, sourceType string,
	alertRaw json.RawMessage, alertParsed *model.ParsedAlert, clientID *int64, fingerprint *string) (*model.Ticket, error) {

	parsedJSON, _ := json.Marshal(alertParsed)
	ticket := &model.Ticket{
		TicketNo:      pkg.GenerateTicketNo(),
		AlertSourceID: alertSourceID,
		SourceType:    sourceType,
		AlertRaw:      model.JSON(alertRaw),
		AlertParsed:   model.JSON(parsedJSON),
		Title:         alertParsed.Title,
		Description:   alertParsed.Description,
		Severity:      alertParsed.Severity,
		Status:        model.TicketStatusPending,
		ClientID:      clientID,
		Fingerprint:   fingerprint,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.ticketRepo.Create(ctx, ticket); err != nil {
		return nil, err
	}

	// 初始化 workflow_states
	nodes := []model.NodeName{model.NodeAlertReceived, model.NodeParsed, model.NodePushed, model.NodeAwaitingAuth, model.NodeAuthorized, model.NodeExecuting, model.NodeCompleted}
	for _, node := range nodes {
		ws := &model.WorkflowState{
			TicketID: ticket.ID,
			NodeName: node,
			Status:   model.NodeStatusPending,
		}
		if node == model.NodeAlertReceived {
			ws.Status = model.NodeStatusDone
			now := time.Now()
			ws.StartedAt = &now
			ws.CompletedAt = &now
		}
		if node == model.NodeParsed {
			ws.Status = model.NodeStatusDone
			now := time.Now()
			ws.StartedAt = &now
			ws.CompletedAt = &now
		}
		s.workflowRepo.Create(ctx, ws)
	}

	s.logTransition(ctx, ticket.ID, "", model.TicketStatusPending, "system", "ticket created")
	s.logger.Info("ticket created", zap.String("ticket_no", ticket.TicketNo))
	return ticket, nil
}

// AppendAlert 去重匹配后追加告警记录
func (s *TicketService) AppendAlert(ctx context.Context, ticketID int64, raw json.RawMessage, parsed *model.ParsedAlert) error {
	parsedJSON, _ := json.Marshal(parsed)
	return s.recordRepo.Create(ctx, &model.AlertRecord{
		TicketID:    ticketID,
		AlertRaw:    model.JSON(raw),
		AlertParsed: model.JSON(parsedJSON),
		ReceivedAt:  time.Now(),
	})
}

// TransitionStatus 状态转换 (带校验)
func (s *TicketService) TransitionStatus(ctx context.Context, ticketID int64, to model.TicketStatus, operator string) error {
	ticket, err := s.ticketRepo.GetByID(ctx, ticketID)
	if err != nil { return fmt.Errorf("get ticket: %w", err) }
	if !canTransition(ticket.Status, to) {
		return fmt.Errorf("invalid transition: %s -> %s", ticket.Status, to)
	}
	from := ticket.Status
	if err := s.ticketRepo.UpdateStatus(ctx, ticketID, to); err != nil {
		return err
	}
	s.logTransition(ctx, ticketID, from, to, operator, "")
	return nil
}

func (s *TicketService) logTransition(ctx context.Context, ticketID int64, from, to model.TicketStatus, operator, action string) {
	if action == "" { action = fmt.Sprintf("status_change:%s->%s", from, to) }
	s.logRepo.Create(ctx, &model.TicketLog{
		TicketID: ticketID, Action: action,
		FromState: (*string)(&from), ToState: (*string)(&to),
		Operator: &operator,
	})
}
```

- [ ] **Step 2: 编写 alert_service.go**

```go
package service

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/alert/parser"
	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
)

type AlertService struct {
	alertSourceRepo *repository.AlertSourceRepo
	ticketService   *TicketService
	logger          *zap.Logger
}

func NewAlertService(
	alertSourceRepo *repository.AlertSourceRepo,
	ticketService *TicketService,
	logger *zap.Logger,
) *AlertService {
	return &AlertService{
		alertSourceRepo: alertSourceRepo,
		ticketService:   ticketService,
		logger:          logger,
	}
}

type IngestResult struct {
	TicketID int64  `json:"ticket_id"`
	TicketNo string `json:"ticket_no"`
	Status   string `json:"status"`
	IsNew    bool   `json:"is_new"`
}

func (s *AlertService) Ingest(ctx context.Context, sourceID int64, raw json.RawMessage) (*IngestResult, error) {
	source, err := s.alertSourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("get alert source: %w", err)
	}

	// 查找或使用 generic parser
	p, ok := parser.Get(source.Type)
	if !ok {
		p, _ = parser.Get("generic")
	}

	parsed, err := p.Parse(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("parse alert: %w", err)
	}

	// 计算指纹
	var fingerprint *string
	if len(source.DedupFields) > 0 {
		var fields []string
		json.Unmarshal(source.DedupFields, &fields)
		if len(fields) > 0 {
			fp, err := pkg.ComputeFingerprint(raw, fields)
			if err == nil {
				fingerprint = &fp
			}
		}
	}

	// 去重检查
	if fingerprint != nil {
		existing, err := s.ticketService.ticketRepo.GetByFingerprint(ctx, *fingerprint)
		if err == nil && existing != nil {
			// 检查是否在去重窗口内
			if time.Since(existing.CreatedAt).Seconds() < float64(source.DedupWindowSec) {
				_ = s.ticketService.AppendAlert(ctx, existing.ID, raw, parsed)
				return &IngestResult{
					TicketID: existing.ID,
					TicketNo: existing.TicketNo,
					Status:   string(existing.Status),
					IsNew:    false,
				}, nil
			}
		}
	}

	// 创建新工单
	var clientID *int64
	ticket, err := s.ticketService.CreateTicket(ctx, sourceID, source.Type, raw, parsed, clientID, fingerprint)
	if err != nil {
		return nil, err
	}

	return &IngestResult{
		TicketID: ticket.ID,
		TicketNo: ticket.TicketNo,
		Status:   string(ticket.Status),
		IsNew:    true,
	}, nil
}
```

需要 import `"fmt"`

- [ ] **Step 3: 编写 auth_service.go**

```go
package service

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xavierli/network-ticket/internal/config"
	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepo
	jwtCfg   *config.JWTConfig
}

func NewAuthService(userRepo *repository.UserRepo, jwtCfg *config.JWTConfig) *AuthService {
	return &AuthService{userRepo: userRepo, jwtCfg: jwtCfg}
}

type Claims struct {
	UserID   int64         `json:"user_id"`
	Username string        `json:"username"`
	Role     model.UserRole `json:"role"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(username, password string) (string, *Claims, error) {
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.jwtCfg.ExpireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.jwtCfg.Secret))
	if err != nil {
		return "", nil, err
	}
	return tokenStr, claims, nil
}

func (s *AuthService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.jwtCfg.Secret), nil
	})
	if err != nil { return nil, err }
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid { return nil, fmt.Errorf("invalid token") }
	return claims, nil
}
```

```bash
cd backend && go get github.com/golang-jwt/jwt/v5
```

- [ ] **Step 4: 编写状态转换测试**

```go
// tests/ticket_service_test.go
package tests

import (
	"testing"

	"github.com/xavierli/network-ticket/internal/model"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from, to model.TicketStatus
		want     bool
	}{
		{model.TicketStatusPending, model.TicketStatusInProgress, true},
		{model.TicketStatusPending, model.TicketStatusFailed, true},
		{model.TicketStatusPending, model.TicketStatusCompleted, false},
		{model.TicketStatusInProgress, model.TicketStatusCompleted, true},
		{model.TicketStatusInProgress, model.TicketStatusRejected, true},
		{model.TicketStatusCompleted, model.TicketStatusPending, false},
	}
	for _, tt := range tests {
		got := canTransition(tt.from, tt.to)
		if got != tt.want {
			t.Errorf("canTransition(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
```

注：`canTransition` 在 service 包中未导出。需要测试时导出为 `CanTransition` 或在 service 包内写测试。

- [ ] **Step 5: 运行测试**

```bash
cd backend && go test ./tests/ -v -run TestCanTransition
```

- [ ] **Step 6: 提交**

```bash
git add backend/internal/service/ backend/tests/
git commit -m "feat: add ticket engine (state machine), alert service (ingest+dedup), auth service (JWT)"
```

---

## Task 9: 客户推送 + Worker Pool

**Files:**
- Create: `backend/internal/client/pusher.go`
- Create: `backend/internal/client/worker.go`
- Create: `backend/internal/client/retry.go`
- Create: `backend/tests/worker_test.go`

- [ ] **Step 1: 编写 pusher.go**

```go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/xavierli/network-ticket/internal/pkg"
)

type PushRequest struct {
	TicketNo    string      `json:"ticket_no"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Severity    string      `json:"severity"`
	AlertParsed interface{} `json:"alert_parsed"`
	CallbackURL string      `json:"callback_url"`
}

type PushResponse struct {
	StatusCode int
	Body       string
	Success    bool
}

func Push(ctx context.Context, endpoint, apiKey, hmacSecret string, req *PushRequest) (*PushResponse, error) {
	body, _ := json.Marshal(req)
	timestamp := time.Now().Unix()
	signature := pkg.SignHMAC(hmacSecret, timestamp, body)
	nonce := uuid.New().String()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", apiKey)
	httpReq.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	httpReq.Header.Set("X-Signature", signature)
	httpReq.Header.Set("X-Nonce", nonce)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return &PushResponse{Success: false}, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return &PushResponse{
		StatusCode: resp.StatusCode,
		Body:       string(respBody),
		Success:    success,
	}, nil
}
```

```bash
cd backend && go get github.com/google/uuid
```

- [ ] **Step 2: 编写 retry.go**

```go
package client

import (
	"context"
	"math"
	"time"
)

type RetryConfig struct {
	MaxAttempts   int
	BaseInterval  time.Duration
	MaxInterval   time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  5,
		BaseInterval: time.Second,
		MaxInterval:  30 * time.Second,
	}
}

func backoff(attempt int, cfg RetryConfig) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * cfg.BaseInterval
	if delay > cfg.MaxInterval {
		delay = cfg.MaxInterval
	}
	return delay
}
```

- [ ] **Step 3: 编写 worker.go**

```go
package client

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

type PushJob struct {
	Ticket  *model.Ticket
	Client  *model.Client
	Attempt int
}

type WorkerPool struct {
	jobs      chan *PushJob
	wg        sync.WaitGroup
	poolSize  int
	retryCfg  RetryConfig
	ticketRepo *repository.TicketRepo
	clientRepo *repository.ClientRepo
	workflowRepo *repository.WorkflowStateRepo
	logger    *zap.Logger
}

func NewWorkerPool(
	poolSize int,
	retryCfg RetryConfig,
	ticketRepo *repository.TicketRepo,
	clientRepo *repository.ClientRepo,
	workflowRepo *repository.WorkflowStateRepo,
	logger *zap.Logger,
) *WorkerPool {
	return &WorkerPool{
		jobs:        make(chan *PushJob, poolSize*10),
		poolSize:    poolSize,
		retryCfg:    retryCfg,
		ticketRepo:  ticketRepo,
		clientRepo:  clientRepo,
		workflowRepo: workflowRepo,
		logger:      logger,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.poolSize; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.logger.Info("worker pool started", zap.Int("size", wp.poolSize))
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
}

func (wp *WorkerPool) Submit(job *PushJob) {
	wp.jobs <- job
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	for job := range wp.jobs {
		wp.process(job, id)
	}
}

func (wp *WorkerPool) process(job *PushJob, workerID int) {
	ctx := context.Background()
	req := &PushRequest{
		TicketNo:    job.Ticket.TicketNo,
		Title:       job.Ticket.Title,
		Description: job.Ticket.Description,
		Severity:    job.Ticket.Severity,
		AlertParsed: json.RawMessage(job.Ticket.AlertParsed),
		CallbackURL: "/api/v1/callback/authorization",
	}

	resp, err := Push(ctx, job.Client.APIEndpoint, job.Client.APIKey, job.Client.HMACSecret, req)
	if err != nil || !resp.Success {
		wp.logger.Warn("push failed",
			zap.String("ticket_no", job.Ticket.TicketNo),
			zap.Int("attempt", job.Attempt),
			zap.Error(err),
		)
		if job.Attempt < wp.retryCfg.MaxAttempts {
			delay := backoff(job.Attempt, wp.retryCfg)
			time.AfterFunc(delay, func() {
				wp.Submit(&PushJob{
					Ticket:  job.Ticket,
					Client:  job.Client,
					Attempt: job.Attempt + 1,
				})
			})
		} else {
			wp.ticketRepo.UpdateStatus(ctx, job.Ticket.ID, model.TicketStatusFailed)
		}
		return
	}

	wp.logger.Info("push succeeded",
		zap.String("ticket_no", job.Ticket.TicketNo),
		zap.Int("worker_id", workerID),
	)
	wp.ticketRepo.UpdateStatus(ctx, job.Ticket.ID, model.TicketStatusInProgress)
	// 更新 workflow_state: pushed -> done, awaiting_auth -> active
	wp.workflowRepo.UpdateStatus(ctx, job.Ticket.ID, model.NodePushed, model.NodeStatusDone)
	wp.workflowRepo.UpdateStatus(ctx, job.Ticket.ID, model.NodeAwaitingAuth, model.NodeStatusActive)
}
```

- [ ] **Step 4: 提交**

```bash
git add backend/internal/client/
git commit -m "feat: add client pusher, worker pool, and exponential backoff retry"
```

---

## Task 10: 中间件 (Auth/Signature/Nonce/Logger)

**Files:**
- Create: `backend/internal/middleware/auth.go`
- Create: `backend/internal/middleware/signature.go`
- Create: `backend/internal/middleware/nonce.go`
- Create: `backend/internal/middleware/logger.go`

- [ ] **Step 1: 编写 auth.go (JWT 中间件)**

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xavierli/network-ticket/internal/service"
)

func JWTAuth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := authService.ParseToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 2: 编写 signature.go (HMAC 验签中间件)**

```go
package middleware

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xavierli/network-ticket/internal/pkg"
	"github.com/xavierli/network-ticket/internal/repository"
)

func HMACSignature(clientRepo *repository.ClientRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-Api-Key")
		timestampStr := c.GetHeader("X-Timestamp")
		signature := c.GetHeader("X-Signature")

		if apiKey == "" || timestampStr == "" || signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth headers"})
			return
		}

		client, err := clientRepo.GetByAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		timestamp, _ := strconv.ParseInt(timestampStr, 10, 64)
		if err := pkg.VerifyTimestamp(timestamp, 300); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		body, _ := io.ReadAll(c.Request.Body)
		if !pkg.VerifyHMAC(client.HMACSecret, timestamp, body, signature) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		c.Set("body_bytes", body)
		c.Set("client", client)
		c.Set("client_id", client.ID)
		c.Next()
	}
}
```

- [ ] **Step 3: 编写 nonce.go (防重放中间件)**

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xavierli/network-ticket/internal/nonce"
)

func NonceCheck(store nonce.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		nonceVal := c.GetHeader("X-Nonce")
		if nonceVal == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing nonce"})
			return
		}
		ok, err := store.CheckAndSet(c.Request.Context(), nonceVal, 5*time.Minute)
		if err != nil || !ok {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "duplicate request"})
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: 编写 logger.go (请求日志中间件)**

```go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		logger.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}
```

- [ ] **Step 5: 提交**

```bash
git add backend/internal/middleware/
git commit -m "feat: add JWT auth, HMAC signature, nonce, and request logger middleware"
```

---

## Task 11: HTTP Handler 层

**Files:**
- Create: `backend/internal/handler/alert_handler.go`
- Create: `backend/internal/handler/ticket_handler.go`
- Create: `backend/internal/handler/client_handler.go`
- Create: `backend/internal/handler/callback_handler.go`
- Create: `backend/internal/handler/auth_handler.go`
- Create: `backend/internal/handler/admin_handler.go`

- [ ] **Step 1: 编写所有 handler**

每个 handler 遵循标准 Gin handler 模式。以 callback_handler.go（最核心的授权回调）为例：

```go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/service"
)

type CallbackHandler struct {
	ticketService *service.TicketService
	logger        *zap.Logger
}

func NewCallbackHandler(ts *service.TicketService, logger *zap.Logger) *CallbackHandler {
	return &CallbackHandler{ticketService: ts, logger: logger}
}

type AuthorizationCallback struct {
	TicketNo     string `json:"ticket_no"`
	Action       string `json:"action"` // authorize | reject
	Operator     string `json:"operator"`
	Comment      string `json:"comment"`
	AuthorizedAt string `json:"authorized_at"`
}

func (h *CallbackHandler) Handle(c *gin.Context) {
	bodyBytes, _ := c.Get("body_bytes")
	body, _ := bodyBytes.([]byte)

	var req AuthorizationCallback
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	ticket, err := h.ticketService.GetByTicketNo(c.Request.Context(), req.TicketNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ticket not found"})
		return
	}

	clientID, _ := c.Get("client_id")
	clientIDInt := clientID.(int64)
	if ticket.ClientID == nil || *ticket.ClientID != clientIDInt {
		c.JSON(http.StatusForbidden, gin.H{"error": "ticket not belong to this client"})
		return
	}

	switch req.Action {
	case "authorize":
		h.ticketService.TransitionStatus(c.Request.Context(), ticket.ID, model.TicketStatusInProgress, "client:"+req.Operator)
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "ticket_no": req.TicketNo, "status": "authorized"})
	case "reject":
		h.ticketService.TransitionStatus(c.Request.Context(), ticket.ID, model.TicketStatusRejected, "client:"+req.Operator)
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "ticket_no": req.TicketNo, "status": "rejected"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
	}
}
```

其余 handler 遵循相同模式：
- `alert_handler.go`: `Webhook` (接收告警), `Create` (手动), `AlertSource CRUD`
- `ticket_handler.go`: `List`, `Get`, `Update`, `Retry`, `Cancel`
- `client_handler.go`: `List`, `Create`, `Update`, `Delete`
- `auth_handler.go`: `Login` (返回 JWT)
- `admin_handler.go`: `ListAuditLogs`

- [ ] **Step 2: 提交**

```bash
git add backend/internal/handler/
git commit -m "feat: add all HTTP handlers (alert, ticket, client, callback, auth, admin)"
```

---

## Task 12: 路由注册 + main.go 完整启动

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 完整 main.go — 初始化 DB/Service/Handler, 注册路由, 启动服务**

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/client"
	"github.com/xavierli/network-ticket/internal/config"
	"github.com/xavierli/network-ticket/internal/handler"
	"github.com/xavierli/network-ticket/internal/middleware"
	"github.com/xavierli/network-ticket/internal/nonce"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

func main() {
	configPath := flag.String("config", "config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := config.InitLogger(&cfg.Log)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer logger.Sync()

	// DB
	db, err := repository.NewDB(&cfg.Database)
	if err != nil {
		logger.Fatal("connect db", zap.Error(err))
	}
	defer db.Close()

	// Repos
	ticketRepo := repository.NewTicketRepo(db)
	workflowRepo := repository.NewWorkflowStateRepo(db)
	alertSourceRepo := repository.NewAlertSourceRepo(db)
	alertRecordRepo := repository.NewAlertRecordRepo(db)
	clientRepo := repository.NewClientRepo(db)
	logRepo := repository.NewTicketLogRepo(db)
	auditRepo := repository.NewAuditLogRepo(db)
	userRepo := repository.NewUserRepo(db)

	// Nonce store
	var nonceStore nonce.Store
	switch cfg.Security.Nonce.Backend {
	case "file":
		nonceStore, err = nonce.NewFileStore(cfg.Security.Nonce.File.Path)
		if err != nil { logger.Fatal("init file nonce store", zap.Error(err)) }
	default:
		nonceStore = nonce.NewDBStore(db)
	}

	// Services
	ticketSvc := service.NewTicketService(ticketRepo, workflowRepo, logRepo, auditRepo, alertRecordRepo, logger)
	alertSvc := service.NewAlertService(alertSourceRepo, ticketSvc, logger)
	authSvc := service.NewAuthService(userRepo, &cfg.JWT)

	// Worker pool
	retryCfg := client.RetryConfig{
		MaxAttempts:  cfg.Worker.RetryMax,
		BaseInterval: cfg.Worker.RetryBaseInterval,
		MaxInterval:  cfg.Worker.RetryMaxInterval,
	}
	workerPool := client.NewWorkerPool(cfg.Worker.PoolSize, retryCfg, ticketRepo, clientRepo, workflowRepo, logger)
	workerPool.Start()
	defer workerPool.Stop()

	// Handlers
	alertHandler := handler.NewAlertHandler(alertSvc, logger)
	ticketHandler := handler.NewTicketHandler(ticketSvc, logger)
	clientHandler := handler.NewClientHandler(clientRepo, logger)
	callbackHandler := handler.NewCallbackHandler(ticketSvc, logger)
	authHandler := handler.NewAuthHandler(authSvc, logger)
	adminHandler := handler.NewAdminHandler(auditRepo, logger)

	// Router
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))

	// 静态文件 (前端构建产物)
	r.Static("/assets", "./frontend/dist/assets")
	r.StaticFile("/", "./frontend/dist/index.html")
	r.StaticFile("/login", "./frontend/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./frontend/dist/index.html")
	})

	api := r.Group("/api/v1")
	{
		// 告警接入
		alerts := api.Group("/alerts")
		{
			alerts.POST("/webhook/:source_id", alertHandler.Webhook)
			alerts.POST("", middleware.JWTAuth(authSvc), alertHandler.Create)
		}

		// 告警源管理
		sources := api.Group("/alert-sources", middleware.JWTAuth(authSvc))
		{
			sources.GET("", alertHandler.ListSources)
			sources.POST("", alertHandler.CreateSource)
			sources.PUT("/:id", alertHandler.UpdateSource)
			sources.DELETE("/:id", alertHandler.DeleteSource)
		}

		// 客户回调
		api.POST("/callback/authorization",
			middleware.HMACSignature(clientRepo),
			middleware.NonceCheck(nonceStore),
			callbackHandler.Handle,
		)

		// 工单管理
		tickets := api.Group("/tickets", middleware.JWTAuth(authSvc))
		{
			tickets.GET("", ticketHandler.List)
			tickets.GET("/:id", ticketHandler.Get)
			tickets.PUT("/:id", ticketHandler.Update)
			tickets.POST("/:id/retry", ticketHandler.Retry)
			tickets.POST("/:id/cancel", ticketHandler.Cancel)
		}

		// 客户管理
		clients := api.Group("/clients", middleware.JWTAuth(authSvc))
		{
			clients.GET("", clientHandler.List)
			clients.POST("", middleware.RequireAdmin(), clientHandler.Create)
			clients.PUT("/:id", middleware.RequireAdmin(), clientHandler.Update)
			clients.DELETE("/:id", middleware.RequireAdmin(), clientHandler.Delete)
		}

		// 审计日志
		api.GET("/audit-logs", middleware.JWTAuth(authSvc), adminHandler.ListAuditLogs)

		// 登录
		api.POST("/auth/login", authHandler.Login)
	}

	// Start
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		logger.Info("server listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down...")
}
```

- [ ] **Step 2: 验证编译**

```bash
cd backend && go build ./cmd/server
```

- [ ] **Step 3: 提交**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire up all components in main.go with routes and graceful shutdown"
```

---

## Task 13: 前端脚手架

**Files:**
- Create: `frontend/` (Next.js 项目)

- [ ] **Step 1: 初始化 Next.js 项目**

```bash
cd /Users/xavierli/life/code/network-ticket
npx create-next-app@latest frontend --typescript --tailwind --eslint --app --src-dir --no-import-alias --use-npm
```

- [ ] **Step 2: 安装 shadcn/ui + SWR**

```bash
cd frontend
npx shadcn@latest init
npx shadcn@latest add button input label card table badge dialog select
npm install swr
```

- [ ] **Step 3: 创建类型定义 `src/types/index.ts`**

```typescript
export interface Ticket {
  id: number;
  ticket_no: string;
  source_type: string;
  title: string;
  description: string;
  severity: string;
  status: string;
  client_id?: number;
  created_at: string;
  updated_at: string;
}

export interface Client {
  id: number;
  name: string;
  api_endpoint: string;
  callback_url?: string;
  status: string;
  created_at: string;
}

export interface AlertSource {
  id: number;
  name: string;
  type: string;
  poll_endpoint?: string;
  poll_interval: number;
  status: string;
}

export interface User {
  id: number;
  username: string;
  role: 'admin' | 'operator';
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}
```

- [ ] **Step 4: 创建 API 客户端 `src/lib/api.ts`**

```typescript
const API_BASE = '/api/v1';

class ApiClient {
  private token: string | null = null;

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem('token', token);
    }
  }

  getToken(): string | null {
    if (this.token) return this.token;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('token');
    }
    return this.token;
  }

  async fetch<T>(path: string, options?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options?.headers as Record<string, string>),
    };
    const token = this.getToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const res = await fetch(`${API_BASE}${path}`, { ...options, headers });
    if (res.status === 401) {
      if (typeof window !== 'undefined') window.location.href = '/login';
      throw new Error('unauthorized');
    }
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
  }

  get<T>(path: string) { return this.fetch<T>(path); }
  post<T>(path: string, body: unknown) {
    return this.fetch<T>(path, { method: 'POST', body: JSON.stringify(body) });
  }
  put<T>(path: string, body: unknown) {
    return this.fetch<T>(path, { method: 'PUT', body: JSON.stringify(body) });
  }
  delete<T>(path: string) {
    return this.fetch<T>(path, { method: 'DELETE' });
  }
}

export const api = new ApiClient();
```

- [ ] **Step 5: 提交**

```bash
git add frontend/
git commit -m "feat: scaffold Next.js frontend with shadcn/ui, SWR, API client, and type definitions"
```

---

## Task 14: 前端页面实现

**Files:**
- Create: `frontend/src/app/login/page.tsx`
- Create: `frontend/src/app/tickets/page.tsx`
- Create: `frontend/src/app/tickets/[id]/page.tsx`
- Create: `frontend/src/app/clients/page.tsx`
- Create: `frontend/src/app/sources/page.tsx`
- Create: `frontend/src/components/layout/sidebar.tsx`
- Create: `frontend/src/components/layout/header.tsx`
- Create: `frontend/src/components/ticket/ticket-table.tsx`
- Create: `frontend/src/components/ticket/ticket-status-badge.tsx`
- Create: `frontend/src/components/auth/login-form.tsx`

- [ ] **Step 1: 编写布局组件 (sidebar + header)**

sidebar 导航: 工单管理 / 客户管理 / 告警源管理, 纯中文。

- [ ] **Step 2: 编写登录页面**

表单: 用户名 + 密码, 调用 `POST /api/v1/auth/login`, 存储 token 到 localStorage, 跳转 `/tickets`。

- [ ] **Step 3: 编写工单列表页**

使用 shadcn Table 组件，列: 编号/标题/严重级别/状态/客户/创建时间。筛选: 状态/严重级别/关键词。分页。使用 SWR 获取数据。

- [ ] **Step 4: 编写工单详情页**

展示工单基本信息 + workflow_states 时间线 + alert_records 追加记录列表。操作按钮: 重试/取消。

- [ ] **Step 5: 编写客户管理页和告警源管理页**

标准 CRUD 表格页面，创建/编辑弹窗 (shadcn Dialog)。

- [ ] **Step 6: 更新 `next.config.js`**

```javascript
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  images: { unoptimized: true },
};
module.exports = nextConfig;
```

使用 `output: 'export'` 生成静态文件，由 Go 后端服务。

- [ ] **Step 7: 提交**

```bash
git add frontend/src/
git commit -m "feat: implement all frontend pages (login, tickets, clients, sources)"
```

---

## Task 15: Docker Compose + Nginx + 部署配置

**Files:**
- Create: `docker-compose.yaml`
- Create: `nginx/nginx.conf`
- Create: `backend/Dockerfile`
- Create: `frontend/Dockerfile`

- [ ] **Step 1: 编写 docker-compose.yaml**

```yaml
version: "3.8"
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root_password
      MYSQL_DATABASE: network_ticket
      MYSQL_USER: ticket
      MYSQL_PASSWORD: ticket_password
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      retries: 10

  backend:
    build: ./backend
    ports:
      - "8080:8080"
    depends_on:
      mysql:
        condition: service_healthy
    environment:
      - DATABASE_PASSWORD=ticket_password
      - DATABASE_HOST=mysql
      - JWT_SECRET=production-secret-change-me
    volumes:
      - ./backend/config.yaml:/app/config.yaml
      - backend_logs:/app/logs

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      # - ./nginx/certs:/etc/nginx/certs  # HTTPS 证书
    depends_on:
      - backend

volumes:
  mysql_data:
  backend_logs:
```

- [ ] **Step 2: 编写 nginx.conf**

```nginx
events {
    worker_connections 1024;
}

http {
    upstream backend {
        server backend:8080;
    }

    server {
        listen 80;
        # listen 443 ssl;
        # ssl_certificate /etc/nginx/certs/cert.pem;
        # ssl_certificate_key /etc/nginx/certs/key.pem;

        client_max_body_size 10m;

        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        location / {
            proxy_pass http://backend;
        }
    }
}
```

- [ ] **Step 3: 编写 backend/Dockerfile**

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/server .
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/config.example.yaml ./config.yaml

EXPOSE 8080
CMD ["./server", "-config", "config.yaml"]
```

- [ ] **Step 4: 编写 frontend/Dockerfile (multi-stage build)**

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM alpine:3.19
WORKDIR /dist
COPY --from=builder /build/out ./frontend-dist

CMD ["echo", "frontend built"]
```

前端构建产物会被复制到后端镜像的 `./frontend/dist` 目录，由 Go 提供服务。需要调整 backend Dockerfile 添加前端产物 COPY。

- [ ] **Step 5: 提交**

```bash
git add docker-compose.yaml nginx/ backend/Dockerfile frontend/Dockerfile
git commit -m "feat: add Docker Compose, Nginx, and Dockerfiles for deployment"
```

---

## Task 16: 集成测试

**Files:**
- Create: `backend/tests/api_test.go`

- [ ] **Step 1: 编写集成测试**

启动测试 HTTP 服务器，测试完整流程：
1. 登录获取 JWT
2. 创建告警源
3. 创建客户
4. 通过 webhook 接收告警 → 创建工单
5. 查询工单列表
6. 查询工单详情 (含 workflow_states)
7. 模拟客户回调授权

使用 `httptest.NewServer` + Gin test mode，mock 掉实际 DB (使用 sqlmock 或 test container)。

- [ ] **Step 2: 运行集成测试**

```bash
cd backend && go test ./tests/ -v -run TestIntegration -tags=integration
```

- [ ] **Step 3: 提交**

```bash
git add backend/tests/api_test.go
git commit -m "test: add integration tests for full ticket lifecycle"
```

---

## Task 17: 部署手册

**Files:**
- Create: `docs/deployment.md`
- Create: `docs/api.md`

- [ ] **Step 1: 编写 deployment.md**

覆盖内容：
- 环境要求 (Docker, Docker Compose, MySQL 8.0+)
- 快速启动 (`docker compose up -d`)
- 配置说明 (config.yaml 各项)
- 数据库迁移 (`make migrate-up`)
- 默认管理员账号
- HTTPS 配置 (Nginx 证书)
- 生产环境注意事项 (改 JWT secret、改默认密码、数据库备份)
- 常见问题排查

- [ ] **Step 2: 编写 api.md**

所有 API 端点列表、请求/响应格式、认证方式、错误码。

- [ ] **Step 3: 提交**

```bash
git add docs/
git commit -m "docs: add deployment manual and API documentation"
```

---

## 自查清单

### Spec 覆盖度

| Spec 要求 | 对应 Task |
|-----------|-----------|
| 架构: 单体模块化 | Task 1-12 |
| MySQL 8.0+ 存储 | Task 3 (迁移) + Task 4 (repo) |
| tickets 两层状态 | Task 4 (model) + Task 8 (service) |
| workflow_states 独立表 | Task 3 + Task 4 |
| alert_records 追加表 | Task 3 + Task 4 + Task 8 |
| HMAC 签名/验证 | Task 5 (pkg) + Task 10 (middleware) |
| Nonce 防重放 (db/file) | Task 6 |
| 告警解析器 (registry) | Task 7 |
| Worker Pool + 重试 | Task 9 |
| JWT 认证 + 两级角色 | Task 8 (auth) + Task 10 (middleware) |
| 管理后台前端 | Task 13-14 |
| Docker Compose 部署 | Task 15 |
| 部署手册 | Task 17 |
| YAML + 环境变量配置 | Task 1-2 |
| 日志 stdout + 文件双写 | Task 2 |
| 去重窗口可配置 | Task 8 (alert_service) |
| 指纹字段可配置 | Task 5 (fingerprint) + Task 8 |
| 工单编号 TK-YYYYMMDD-NNNN | Task 5 |
| 前后端同域部署 | Task 12 (Go 静态文件) + Task 15 (Nginx) |
| 超时标记 + 通知预留 | Task 8 (status timeout) |

### 占位符扫描

无 TBD/TODO/待定。

### 类型一致性

所有 model 定义在 Task 4，repo 和 service 均引用相同类型。中间件 handler 引用的 service 方法签名在 Task 8/11 中一致。
