# 工单类型 + 手工建单 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现工单类型管理和手工建单功能，支持管理员定义工单类型，操作员手工创建工单并选择类型。

**Architecture:** 引入独立的 `ticket_types` 表，与告警源解耦。`tickets` 表添加 `ticket_type_id` 外键（可为空，兼容历史数据），`alert_source_id` 改为可为空。后端遵循 model → repo → service → handler 分层，前端在现有页面基础上添加管理页面和对话框。

**Tech Stack:** Go 1.24 + Gin + sqlx + MySQL, Next.js 16 + React 19 + Tailwind CSS, golang-migrate for DB migrations

---

## File Structure

### Backend — New Files
- `backend/migrations/010_add_ticket_types.up.sql` — 创建 ticket_types 表，修改 tickets 表
- `backend/migrations/010_add_ticket_types.down.sql` — 回滚迁移
- `backend/internal/model/ticket_type.go` — TicketType 模型
- `backend/internal/repository/ticket_type_repo.go` — TicketType 数据访问
- `backend/internal/service/ticket_type_service.go` — TicketType 业务逻辑
- `backend/internal/handler/ticket_type_handler.go` — TicketType HTTP API (admin-only)

### Backend — Modified Files
- `backend/internal/model/ticket.go` — 添加 TicketTypeID，AlertSourceID 改为 *int64
- `backend/internal/repository/ticket_repo.go` — Create SQL 添加 ticket_type_id，新增 UpdateTicketTypeID
- `backend/internal/service/ticket_service.go` — CreateTicket 处理 alertSourceID=0
- `backend/internal/handler/ticket_handler.go` — 添加 CreateManual 方法
- `backend/cmd/server/main.go` — 注册新组件和路由
- `backend/tests/api_test.go` — 更新 setupRouter 镜像新路由

### Frontend — Modified Files
- `frontend/src/types/index.ts` — 添加 TicketType 接口，修改 Ticket 接口
- `frontend/src/components/layout/sidebar.tsx` — 添加工单类型管理导航
- `frontend/src/app/ticket-types/page.tsx` — 工单类型管理页面（新建）
- `frontend/src/app/tickets/page.tsx` — 添加工单类型筛选 + 手工建单按钮/对话框
- `frontend/src/components/ticket/ticket-table.tsx` — 显示工单类型
- `frontend/src/app/tickets/[id]/page.tsx` — 显示工单类型信息

---

## Task 1: Database Migration

**Files:**
- Create: `backend/migrations/010_add_ticket_types.up.sql`
- Create: `backend/migrations/010_add_ticket_types.down.sql`

- [ ] **Step 1: Write up migration**

Create `backend/migrations/010_add_ticket_types.up.sql`:

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

-- 修改 tickets 表：alert_source_id 改为可为空，添加 ticket_type_id
ALTER TABLE tickets
    MODIFY alert_source_id BIGINT UNSIGNED NULL,
    ADD COLUMN ticket_type_id BIGINT UNSIGNED NULL AFTER source_type;

-- 添加外键
ALTER TABLE tickets ADD CONSTRAINT fk_tickets_ticket_type
    FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id);

-- 插入默认类型
INSERT INTO ticket_types (code, name, description, color) VALUES
    ('default', '默认', '未分类工单', '#6B7280'),
    ('network_fault', '网络故障', '网络设备或链路故障', '#EF4444'),
    ('server_alert', '服务器告警', '服务器性能或硬件告警', '#F59E0B');
```

- [ ] **Step 2: Write down migration**

Create `backend/migrations/010_add_ticket_types.down.sql`:

```sql
ALTER TABLE tickets DROP FOREIGN KEY fk_tickets_ticket_type;
ALTER TABLE tickets DROP COLUMN ticket_type_id;
ALTER TABLE tickets MODIFY alert_source_id BIGINT UNSIGNED NOT NULL;
DROP TABLE ticket_types;
```

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/010_add_ticket_types.up.sql backend/migrations/010_add_ticket_types.down.sql
git commit -m "feat: add ticket_types migration"
```

---

## Task 2: Backend Model — TicketType + Ticket Modifications

**Files:**
- Create: `backend/internal/model/ticket_type.go`
- Modify: `backend/internal/model/ticket.go`

- [ ] **Step 1: Create TicketType model**

Create `backend/internal/model/ticket_type.go`:

```go
package model

import "time"

// TicketType represents a category of ticket.
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

- [ ] **Step 2: Modify Ticket model**

Modify `backend/internal/model/ticket.go`. Change `AlertSourceID` from `int64` to `*int64`, and add `TicketTypeID`:

```go
// Ticket represents a network ticket record.
type Ticket struct {
	ID            int64      `db:"id"             json:"id"`
	TicketNo      string     `db:"ticket_no"      json:"ticket_no"`
	AlertSourceID *int64     `db:"alert_source_id" json:"alert_source_id"`
	SourceType    string     `db:"source_type"    json:"source_type"`
	TicketTypeID  *int64     `db:"ticket_type_id"  json:"ticket_type_id"`
	AlertRaw      JSON       `db:"alert_raw"      json:"alert_raw"`
	AlertParsed   JSON       `db:"alert_parsed"   json:"alert_parsed"`
	Title         string     `db:"title"          json:"title"`
	Description   *string    `db:"description"    json:"description"`
	Severity      string     `db:"severity"       json:"severity"`
	Status        string     `db:"status"         json:"status"`
	ClientID      *int64     `db:"client_id"      json:"client_id"`
	ExternalID    *string    `db:"external_id"    json:"external_id"`
	CallbackData  JSON       `db:"callback_data"  json:"callback_data"`
	Fingerprint   *string    `db:"fingerprint"    json:"fingerprint"`
	TimeoutAt     *time.Time `db:"timeout_at"     json:"timeout_at"`
	CreatedAt     time.Time  `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"     json:"updated_at"`
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/ticket_type.go backend/internal/model/ticket.go
git commit -m "feat: add TicketType model, make AlertSourceID nullable"
```

---

## Task 3: Backend Repository — TicketTypeRepo

**Files:**
- Create: `backend/internal/repository/ticket_type_repo.go`

- [ ] **Step 1: Implement TicketTypeRepo**

Create `backend/internal/repository/ticket_type_repo.go`:

```go
package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/xavierli/network-ticket/internal/model"
)

type TicketTypeRepo struct {
	db *sqlx.DB
}

func NewTicketTypeRepo(db *sqlx.DB) *TicketTypeRepo {
	return &TicketTypeRepo{db: db}
}

// Create inserts a new ticket type and returns the ID.
func (r *TicketTypeRepo) Create(ctx context.Context, tt *model.TicketType) (int64, error) {
	query := `INSERT INTO ticket_types (code, name, description, color, status)
		VALUES (?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		tt.Code, tt.Name, tt.Description, tt.Color, tt.Status,
	)
	if err != nil {
		return 0, fmt.Errorf("insert ticket_type: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// GetByID returns a ticket type by its primary key.
func (r *TicketTypeRepo) GetByID(ctx context.Context, id int64) (*model.TicketType, error) {
	var tt model.TicketType
	query := `SELECT * FROM ticket_types WHERE id = ?`
	if err := r.db.GetContext(ctx, &tt, query, id); err != nil {
		return nil, fmt.Errorf("get ticket_type by id: %w", err)
	}
	return &tt, nil
}

// GetByCode returns a ticket type by its code.
func (r *TicketTypeRepo) GetByCode(ctx context.Context, code string) (*model.TicketType, error) {
	var tt model.TicketType
	query := `SELECT * FROM ticket_types WHERE code = ?`
	if err := r.db.GetContext(ctx, &tt, query, code); err != nil {
		return nil, fmt.Errorf("get ticket_type by code: %w", err)
	}
	return &tt, nil
}

// List returns all ticket types ordered by id.
func (r *TicketTypeRepo) List(ctx context.Context) ([]model.TicketType, error) {
	var types []model.TicketType
	query := `SELECT * FROM ticket_types ORDER BY id`
	if err := r.db.SelectContext(ctx, &types, query); err != nil {
		return nil, fmt.Errorf("list ticket_types: %w", err)
	}
	return types, nil
}

// Update updates a ticket type.
func (r *TicketTypeRepo) Update(ctx context.Context, tt *model.TicketType) error {
	query := `UPDATE ticket_types SET
		code = ?, name = ?, description = ?, color = ?, status = ?
		WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query,
		tt.Code, tt.Name, tt.Description, tt.Color, tt.Status, tt.ID,
	); err != nil {
		return fmt.Errorf("update ticket_type: %w", err)
	}
	return nil
}

// Delete removes a ticket type by ID.
func (r *TicketTypeRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM ticket_types WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("delete ticket_type: %w", err)
	}
	return nil
}

// CountTicketsByType returns the number of tickets associated with a ticket type.
func (r *TicketTypeRepo) CountTicketsByType(ctx context.Context, ticketTypeID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tickets WHERE ticket_type_id = ?`
	if err := r.db.GetContext(ctx, &count, query, ticketTypeID); err != nil {
		return 0, fmt.Errorf("count tickets by type: %w", err)
	}
	return count, nil
}
```

- [ ] **Step 2: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS (no output = success)

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/ticket_type_repo.go
git commit -m "feat: add TicketTypeRepo"
```

---

## Task 4: Backend Repository — TicketRepo Modifications

**Files:**
- Modify: `backend/internal/repository/ticket_repo.go`

- [ ] **Step 1: Update Create to include ticket_type_id**

In `backend/internal/repository/ticket_repo.go`, modify the `Create` method. Update the query to include `ticket_type_id` as the last column:

```go
// Create inserts a new ticket and returns the ID.
func (r *TicketRepo) Create(ctx context.Context, t *model.Ticket) (int64, error) {
	query := `INSERT INTO tickets
		(ticket_no, alert_source_id, source_type, ticket_type_id, alert_raw, alert_parsed, title, description,
		 severity, status, client_id, external_id, callback_data, fingerprint, timeout_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result, err := r.db.ExecContext(ctx, query,
		t.TicketNo, t.AlertSourceID, t.SourceType, t.TicketTypeID, t.AlertRaw, t.AlertParsed, t.Title, t.Description,
		t.Severity, t.Status, t.ClientID, t.ExternalID, t.CallbackData, t.Fingerprint, t.TimeoutAt,
	)
	if err != nil {
		return 0, fmt.Errorf("insert ticket: %w", err)
	}
	id, _ := result.LastInsertId()
	return id, nil
}
```

- [ ] **Step 2: Add UpdateTicketTypeID method**

Add after the `Create` method in `backend/internal/repository/ticket_repo.go`:

```go
// UpdateTicketTypeID sets the ticket_type_id for a ticket.
func (r *TicketRepo) UpdateTicketTypeID(ctx context.Context, ticketID int64, ticketTypeID *int64) error {
	query := `UPDATE tickets SET ticket_type_id = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, ticketTypeID, ticketID); err != nil {
		return fmt.Errorf("update ticket type id: %w", err)
	}
	return nil
}
```

- [ ] **Step 3: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/repository/ticket_repo.go
git commit -m "feat: add ticket_type_id to TicketRepo.Create, add UpdateTicketTypeID"
```

---

## Task 5: Backend Service — TicketTypeService

**Files:**
- Create: `backend/internal/service/ticket_type_service.go`

- [ ] **Step 1: Implement TicketTypeService**

Create `backend/internal/service/ticket_type_service.go`:

```go
package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

type TicketTypeService struct {
	ticketTypeRepo *repository.TicketTypeRepo
	logger         *zap.Logger
}

func NewTicketTypeService(ticketTypeRepo *repository.TicketTypeRepo, logger *zap.Logger) *TicketTypeService {
	return &TicketTypeService{ticketTypeRepo: ticketTypeRepo, logger: logger}
}

// List returns all ticket types.
func (s *TicketTypeService) List(ctx context.Context) ([]model.TicketType, error) {
	types, err := s.ticketTypeRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	if types == nil {
		types = []model.TicketType{}
	}
	return types, nil
}

// Create creates a new ticket type.
func (s *TicketTypeService) Create(ctx context.Context, code, name string, description *string, color, status string) (*model.TicketType, error) {
	// Check for duplicate code.
	if _, err := s.ticketTypeRepo.GetByCode(ctx, code); err == nil {
		return nil, fmt.Errorf("code already exists")
	}

	if status == "" {
		status = "active"
	}
	if color == "" {
		color = "#6B7280"
	}

	 tt := &model.TicketType{
		Code:        code,
		Name:        name,
		Description: description,
		Color:       color,
		Status:      status,
	}
	id, err := s.ticketTypeRepo.Create(ctx, tt)
	if err != nil {
		return nil, fmt.Errorf("create ticket type: %w", err)
	}
	tt.ID = id
	s.logger.Info("ticket type created", zap.Int64("id", id), zap.String("code", code))
	return tt, nil
}

// Update updates an existing ticket type.
func (s *TicketTypeService) Update(ctx context.Context, id int64, code, name string, description *string, color, status string) error {
	// Verify the type exists.
	if _, err := s.ticketTypeRepo.GetByID(ctx, id); err != nil {
		return fmt.Errorf("ticket type not found")
	}

	// Check code uniqueness if changing code.
	existing, err := s.ticketTypeRepo.GetByCode(ctx, code)
	if err == nil && existing.ID != id {
		return fmt.Errorf("code already exists")
	}

	tt := &model.TicketType{
		ID:          id,
		Code:        code,
		Name:        name,
		Description: description,
		Color:       color,
		Status:      status,
	}
	if err := s.ticketTypeRepo.Update(ctx, tt); err != nil {
		return fmt.Errorf("update ticket type: %w", err)
	}
	s.logger.Info("ticket type updated", zap.Int64("id", id))
	return nil
}

// Delete deletes a ticket type if it has no associated tickets.
func (s *TicketTypeService) Delete(ctx context.Context, id int64) error {
	count, err := s.ticketTypeRepo.CountTicketsByType(ctx, id)
	if err != nil {
		return fmt.Errorf("check ticket type usage: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("ticket type is in use")
	}
	if err := s.ticketTypeRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete ticket type: %w", err)
	}
	s.logger.Info("ticket type deleted", zap.Int64("id", id))
	return nil
}
```

- [ ] **Step 2: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/ticket_type_service.go
git commit -m "feat: add TicketTypeService"
```

---

## Task 6: Backend Service — TicketService Modifications

**Files:**
- Modify: `backend/internal/service/ticket_service.go`

- [ ] **Step 1: Modify CreateTicket to handle nullable alertSourceID**

In `backend/internal/service/ticket_service.go`, modify the `CreateTicket` method. Change the ticket initialization to handle `alertSourceID`:

Replace the ticket initialization block (lines 86-95 approximately):

```go
func (s *TicketService) CreateTicket(ctx context.Context, alertSourceID int64, sourceType string, alertRaw json.RawMessage, parsedAlert interface{}, clientID *int64, fingerprint *string) (*model.Ticket, error) {
	ticketNo := pkg.GenerateTicketNo()

	alertParsedJSON, err := json.Marshal(parsedAlert)
	if err != nil {
		return nil, fmt.Errorf("marshal parsed alert: %w", err)
	}

	var alertSourceIDPtr *int64
	if alertSourceID != 0 {
		alertSourceIDPtr = &alertSourceID
	}

	ticket := &model.Ticket{
		TicketNo:      ticketNo,
		AlertSourceID: alertSourceIDPtr,
		SourceType:    sourceType,
		AlertRaw:      model.JSON(alertRaw),
		AlertParsed:   model.JSON(alertParsedJSON),
		Status:        model.TicketStatusPending,
		ClientID:      clientID,
		Fingerprint:   fingerprint,
	}

	// Extract title and severity from parsed alert if it's a *parser.ParsedAlert.
	if pa, ok := parsedAlert.(*parser.ParsedAlert); ok {
		ticket.Title = pa.Title
		ticket.Severity = pa.Severity
		if pa.Description != "" {
			ticket.Description = &pa.Description
		}
	}
```

- [ ] **Step 2: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/ticket_service.go
git commit -m "feat: make AlertSourceID nullable in CreateTicket"
```

---

## Task 7: Backend Handler — TicketTypeHandler

**Files:**
- Create: `backend/internal/handler/ticket_type_handler.go`

- [ ] **Step 1: Implement TicketTypeHandler**

Create `backend/internal/handler/ticket_type_handler.go`:

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/service"
)

// TicketTypeHandler handles ticket type CRUD endpoints (admin-only).
type TicketTypeHandler struct {
	ticketTypeService *service.TicketTypeService
	logger            *zap.Logger
}

// NewTicketTypeHandler creates a new TicketTypeHandler.
func NewTicketTypeHandler(ts *service.TicketTypeService, l *zap.Logger) *TicketTypeHandler {
	return &TicketTypeHandler{
		ticketTypeService: ts,
		logger:            l,
	}
}

// List returns all ticket types.
func (h *TicketTypeHandler) List(c *gin.Context) {
	types, err := h.ticketTypeService.List(c.Request.Context())
	if err != nil {
		h.logger.Error("list ticket types failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ticket types"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": types})
}

type CreateTicketTypeRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	Status      string  `json:"status"`
}

// Create creates a new ticket type.
func (h *TicketTypeHandler) Create(c *gin.Context) {
	var req CreateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code and name are required"})
		return
	}

	tt, err := h.ticketTypeService.Create(c.Request.Context(), req.Code, req.Name, req.Description, req.Color, req.Status)
	if err != nil {
		h.logger.Error("create ticket type failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tt)
}

type UpdateTicketTypeRequest struct {
	Code        string  `json:"code" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	Color       string  `json:"color"`
	Status      string  `json:"status"`
}

// Update updates an existing ticket type.
func (h *TicketTypeHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req UpdateTicketTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.ticketTypeService.Update(c.Request.Context(), id, req.Code, req.Name, req.Description, req.Color, req.Status); err != nil {
		h.logger.Error("update ticket type failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete removes a ticket type.
func (h *TicketTypeHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.ticketTypeService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("delete ticket type failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
```

- [ ] **Step 2: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/ticket_type_handler.go
git commit -m "feat: add TicketTypeHandler"
```

---

## Task 8: Backend Handler — TicketHandler Modifications (Manual Creation)

**Files:**
- Modify: `backend/internal/handler/ticket_handler.go`

- [ ] **Step 1: Add imports and modify TicketHandler struct**

At the top of `backend/internal/handler/ticket_handler.go`, add the missing imports:

The file already imports:
- `net/http`
- `strconv`
- `github.com/gin-gonic/gin`
- `go.uber.org/zap`
- `github.com/xavierli/network-ticket/internal/model`
- `github.com/xavierli/network-ticket/internal/service`

Add these imports:
```go
import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/alert/parser"
	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)
```

Modify the `TicketHandler` struct to include additional dependencies:

```go
// TicketHandler handles ticket CRUD and status transition endpoints.
type TicketHandler struct {
	ticketService   *service.TicketService
	clientRepo      *repository.ClientRepo
	ticketTypeRepo  *repository.TicketTypeRepo
	logger          *zap.Logger
}

// NewTicketHandler creates a new TicketHandler.
func NewTicketHandler(ts *service.TicketService, clientRepo *repository.ClientRepo, ticketTypeRepo *repository.TicketTypeRepo, l *zap.Logger) *TicketHandler {
	return &TicketHandler{
		ticketService:  ts,
		clientRepo:     clientRepo,
		ticketTypeRepo: ticketTypeRepo,
		logger:         l,
	}
}
```

- [ ] **Step 2: Add CreateManual method**

Add the following methods at the end of `backend/internal/handler/ticket_handler.go` (after the `Cancel` method):

```go
// CreateManualRequest represents a manual ticket creation request.
type CreateManualRequest struct {
	TicketTypeID int64   `json:"ticket_type_id" binding:"required"`
	Title        string  `json:"title" binding:"required"`
	Description  *string `json:"description"`
	Severity     string  `json:"severity" binding:"required"`
	ClientID     *int64  `json:"client_id"`
}

// CreateManual handles manual ticket creation.
func (h *TicketHandler) CreateManual(c *gin.Context) {
	var req CreateManualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket_type_id, title and severity are required"})
		return
	}

	ctx := c.Request.Context()

	// 1. Validate ticket type exists and is active.
	ticketType, err := h.ticketTypeRepo.GetByID(ctx, req.TicketTypeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ticket type"})
		return
	}
	if ticketType.Status != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket type is not active"})
		return
	}

	// 2. Validate client if provided.
	if req.ClientID != nil {
		client, err := h.clientRepo.GetByID(ctx, *req.ClientID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client"})
			return
		}
		if client.Status != "active" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "client is not active"})
			return
		}
	}

	// 3. Construct ParsedAlert.
	var desc string
	if req.Description != nil {
		desc = *req.Description
	}
	parsed := &parser.ParsedAlert{
		Title:       req.Title,
		Description: desc,
		Severity:    req.Severity,
	}

	// 4. Create ticket via TicketService.
	ticket, err := h.ticketService.CreateTicket(ctx, 0, "manual", []byte("{}"), parsed, req.ClientID, nil)
	if err != nil {
		h.logger.Error("manual ticket creation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ticket"})
		return
	}

	// 5. Set ticket_type_id.
	if err := h.ticketService.GetTicketRepo().UpdateTicketTypeID(ctx, ticket.ID, &req.TicketTypeID); err != nil {
		h.logger.Error("set ticket type id failed", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set ticket type"})
		return
	}

	// Refresh ticket to include the type.
	ticket, err = h.ticketService.GetByID(ctx, ticket.ID)
	if err != nil {
		h.logger.Warn("refresh ticket after create failed", zap.Int64("ticket_id", ticket.ID), zap.Error(err))
	}

	c.JSON(http.StatusCreated, ticket)
}
```

Wait - there's a problem. `h.ticketService.GetTicketRepo()` doesn't exist. I need to expose the repo, or do the update differently.

Options:
1. Add a getter method to TicketService
2. Add a new method `CreateManualTicket` to TicketService that handles everything
3. Pass the ticketRepo to the handler directly

Option 3 is simplest. Let me modify the handler to accept ticketRepo directly, and use it.

Actually, looking at the handler pattern in this codebase, handlers typically only have services, not repos. But the existing AlertHandler does have a repo (`alertSourceRepo`). So it's acceptable to add a `ticketRepo` to TicketHandler.

Let me revise. I'll add `ticketRepo` to the handler and use it directly.

Revised struct:
```go
type TicketHandler struct {
	ticketService   *service.TicketService
	clientRepo      *repository.ClientRepo
	ticketTypeRepo  *repository.TicketTypeRepo
	ticketRepo      *repository.TicketRepo
	logger          *zap.Logger
}
```

Revised constructor:
```go
func NewTicketHandler(ts *service.TicketService, clientRepo *repository.ClientRepo, ticketTypeRepo *repository.TicketTypeRepo, ticketRepo *repository.TicketRepo, l *zap.Logger) *TicketHandler {
	return &TicketHandler{
		ticketService:  ts,
		clientRepo:     clientRepo,
		ticketTypeRepo: ticketTypeRepo,
		ticketRepo:     ticketRepo,
		logger:         l,
	}
}
```

Then in CreateManual, use `h.ticketRepo.UpdateTicketTypeID`.

- [ ] **Step 3: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/handler/ticket_handler.go
git commit -m "feat: add manual ticket creation endpoint"
```

---

## Task 9: Wire Everything in main.go

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add repository, service, handler instantiation**

In `backend/cmd/server/main.go`, in section 4 (Create repositories), add:

```go
	ticketTypeRepo := repository.NewTicketTypeRepo(db)
```

In section 6 (Create services), add:

```go
	ticketTypeService := service.NewTicketTypeService(ticketTypeRepo, logger)
```

In section 9 (Create handlers), add:

```go
	ticketTypeHandler := handler.NewTicketTypeHandler(ticketTypeService, logger)
```

And update the ticketHandler instantiation:

```go
	ticketHandler := handler.NewTicketHandler(ticketService, clientRepo, ticketTypeRepo, ticketRepo, logger)
```

- [ ] **Step 2: Add routes**

In the router setup, add ticket type routes (admin-only) and manual creation route:

After the ticket endpoints block, add:

```go
		// Ticket type endpoints (admin-only).
		ticketTypesAdmin := api.Group("")
		ticketTypesAdmin.Use(middleware.JWTAuth(authService), middleware.RequireAdmin())
		{
			ticketTypesAdmin.GET("/ticket-types", ticketTypeHandler.List)
			ticketTypesAdmin.POST("/ticket-types", ticketTypeHandler.Create)
			ticketTypesAdmin.PUT("/ticket-types/:id", ticketTypeHandler.Update)
			ticketTypesAdmin.DELETE("/ticket-types/:id", ticketTypeHandler.Delete)
		}

		// Manual ticket creation (admin/operator).
		tickets.POST("/tickets/manual", ticketHandler.CreateManual)
```

Note: the `tickets` group already has JWTAuth applied, so `/tickets/manual` will be authenticated. Both admin and operator can access it.

- [ ] **Step 3: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git commit -m "feat: wire ticket type and manual creation APIs"
```

---

## Task 10: Update Integration Test Router

**Files:**
- Modify: `backend/tests/api_test.go`

- [ ] **Step 1: Update setupRouter to mirror main.go changes**

In `backend/tests/api_test.go`, in the `setupRouter` function:

1. Add to repositories section:
```go
	ticketTypeRepo := repository.NewTicketTypeRepo(db)
```

2. Add to services section:
```go
	ticketTypeService := service.NewTicketTypeService(ticketTypeRepo, logger)
```

3. Update handlers section:
```go
	ticketTypeHandler := handler.NewTicketTypeHandler(ticketTypeService, logger)
	// ... update existing handlers
	userHandler := handler.NewUserHandler(userService, logger)
```

4. Update ticketHandler:
```go
	ticketHandler := handler.NewTicketHandler(ticketService, clientRepo, ticketTypeRepo, ticketRepo, logger)
```

5. Add routes in the router setup (after the existing ticket routes block):

```go
		// Ticket type endpoints (admin-only).
		ticketTypesAdmin := api.Group("")
		ticketTypesAdmin.Use(middleware.JWTAuth(authService), middleware.RequireAdmin())
		{
			ticketTypesAdmin.GET("/ticket-types", ticketTypeHandler.List)
			ticketTypesAdmin.POST("/ticket-types", ticketTypeHandler.Create)
			ticketTypesAdmin.PUT("/ticket-types/:id", ticketTypeHandler.Update)
			ticketTypesAdmin.DELETE("/ticket-types/:id", ticketTypeHandler.Delete)
		}

		// Manual ticket creation.
		tickets.POST("/tickets/manual", ticketHandler.CreateManual)
```

6. Also add the user management routes that were added in a previous implementation but are missing from the test router:

```go
		// User management endpoints (admin-only).
		usersAdmin := api.Group("")
		usersAdmin.Use(middleware.JWTAuth(authService), middleware.RequireAdmin())
		{
			usersAdmin.GET("/users", userHandler.List)
			usersAdmin.POST("/users", userHandler.Create)
			usersAdmin.PUT("/users/:id", userHandler.Update)
			usersAdmin.DELETE("/users/:id", userHandler.Delete)
		}
```

- [ ] **Step 2: Verify Go compilation**

Run: `cd backend && go build ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/tests/api_test.go
git commit -m "test: update integration test router for ticket types"
```

---

## Task 11: Frontend Types

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Add TicketType interface and update Ticket interface**

In `frontend/src/types/index.ts`, add the `TicketType` interface before the `Ticket` interface:

```typescript
export interface TicketType {
  id: number;
  code: string;
  name: string;
  description?: string;
  color: string;
  status: string;
  created_at: string;
  updated_at: string;
}
```

Update the `Ticket` interface to add `ticket_type_id`, `alert_source_id`, and `ticket_type`:

```typescript
export interface Ticket {
  id: number;
  ticket_no: string;
  alert_source_id?: number;
  source_type: string;
  ticket_type_id?: number;
  ticket_type?: TicketType;
  title: string;
  description: string;
  severity: string;
  status: string;
  client_id?: number;
  fingerprint?: string;
  created_at: string;
  updated_at: string;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/types/index.ts
git commit -m "feat: add TicketType type, update Ticket type"
```

---

## Task 12: Frontend Sidebar

**Files:**
- Modify: `frontend/src/components/layout/sidebar.tsx`

- [ ] **Step 1: Add ticket-types to navigation**

In `frontend/src/components/layout/sidebar.tsx`, modify `ALL_NAV_ITEMS` to include the ticket-types page (admin-only):

```typescript
const ALL_NAV_ITEMS = [
  { label: '工单管理', href: '/tickets' },
  { label: '告警源管理', href: '/sources' },
  { label: '审计日志', href: '/audit-logs' },
  { label: '客户管理', href: '/clients', adminOnly: true },
  { label: '工单类型', href: '/ticket-types', adminOnly: true },
  { label: '用户管理', href: '/users', adminOnly: true },
];
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/layout/sidebar.tsx
git commit -m "feat: add ticket-types to sidebar navigation"
```

---

## Task 13: Frontend Ticket Types Management Page

**Files:**
- Create: `frontend/src/app/ticket-types/page.tsx`

- [ ] **Step 1: Create the ticket types management page**

Create `frontend/src/app/ticket-types/page.tsx`:

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import type { TicketType } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';

interface TicketTypeForm {
  code: string;
  name: string;
  description: string;
  color: string;
  status: 'active' | 'inactive';
}

const EMPTY_FORM: TicketTypeForm = {
  code: '',
  name: '',
  description: '',
  color: '#6B7280',
  status: 'active',
};

export default function TicketTypesPage() {
  const router = useRouter();
  const [types, setTypes] = useState<TicketType[]>([]);
  const [loading, setLoading] = useState(true);
  const [showDialog, setShowDialog] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<TicketTypeForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);

  // Admin-only route guard
  useEffect(() => {
    try {
      const raw = localStorage.getItem('user');
      if (raw) {
        const user = JSON.parse(raw);
        if (user.role !== 'admin') {
          router.push('/tickets');
        }
      } else {
        router.push('/login');
      }
    } catch {
      router.push('/login');
    }
  }, [router]);

  const fetchTypes = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<{ items: TicketType[] }>('/ticket-types');
      setTypes(data.items);
    } catch {
      // error handled by api client
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTypes();
  }, [fetchTypes]);

  function openCreate() {
    setEditingId(null);
    setForm(EMPTY_FORM);
    setShowDialog(true);
  }

  function openEdit(tt: TicketType) {
    setEditingId(tt.id);
    setForm({
      code: tt.code,
      name: tt.name,
      description: tt.description || '',
      color: tt.color,
      status: tt.status as 'active' | 'inactive',
    });
    setShowDialog(true);
  }

  async function handleSave() {
    if (!form.code || !form.name) return;
    setSaving(true);
    try {
      const body = {
        code: form.code,
        name: form.name,
        description: form.description || null,
        color: form.color,
        status: form.status,
      };
      if (editingId) {
        await api.put(`/ticket-types/${editingId}`, body);
      } else {
        await api.post('/ticket-types', body);
      }
      setShowDialog(false);
      fetchTypes();
    } catch {
      // error handled by api client
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await api.delete(`/ticket-types/${id}`);
      setDeleteConfirm(null);
      fetchTypes();
    } catch {
      // error handled by api client
    }
  }

  function updateForm<K extends keyof TicketTypeForm>(field: K, value: TicketTypeForm[K]) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-bold text-gray-800">工单类型管理</h2>
            <button
              onClick={openCreate}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              新增类型
            </button>
          </div>

          {loading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-gray-200 bg-gray-50">
                    <th className="px-4 py-3 text-left font-medium text-gray-600">ID</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">编码</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">名称</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">颜色</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">状态</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">创建时间</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {!types || types.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="px-4 py-8 text-center text-gray-500">
                        暂无工单类型数据
                      </td>
                    </tr>
                  ) : (
                    types.map((tt) => (
                      <tr key={tt.id} className="border-b border-gray-100">
                        <td className="px-4 py-3 text-gray-500">{tt.id}</td>
                        <td className="px-4 py-3 font-mono text-xs">{tt.code}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <span
                              className="inline-block h-3 w-3 rounded-full"
                              style={{ backgroundColor: tt.color }}
                            />
                            <span className="font-medium">{tt.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <span className="font-mono text-xs text-gray-500">{tt.color}</span>
                        </td>
                        <td className="px-4 py-3">
                          <span
                            className={`inline-block rounded border px-2 py-0.5 text-xs font-medium ${
                              tt.status === 'active'
                                ? 'border-green-300 bg-green-100 text-green-800'
                                : 'border-gray-300 bg-gray-100 text-gray-800'
                            }`}
                          >
                            {tt.status === 'active' ? '活跃' : '停用'}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-500">
                          {new Date(tt.created_at).toLocaleString('zh-CN')}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex gap-2">
                            <button
                              onClick={() => openEdit(tt)}
                              className="text-blue-600 hover:text-blue-800"
                            >
                              编辑
                            </button>
                            <button
                              onClick={() => setDeleteConfirm(tt.id)}
                              className="text-red-600 hover:text-red-800"
                            >
                              删除
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}

          {/* Create/Edit Dialog */}
          {showDialog && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
              <div className="w-full max-w-lg rounded-lg bg-white p-6 shadow-lg">
                <h3 className="mb-4 text-lg font-bold text-gray-800">
                  {editingId ? '编辑工单类型' : '新增工单类型'}
                </h3>
                <div className="space-y-3">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">编码</label>
                    <input
                      value={form.code}
                      onChange={(e) => updateForm('code', e.target.value)}
                      placeholder="如 network_fault"
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">名称</label>
                    <input
                      value={form.name}
                      onChange={(e) => updateForm('name', e.target.value)}
                      placeholder="如 网络故障"
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">描述</label>
                    <input
                      value={form.description}
                      onChange={(e) => updateForm('description', e.target.value)}
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div className="flex items-center gap-3">
                    <div className="flex-1">
                      <label className="mb-1 block text-sm font-medium text-gray-700">颜色</label>
                      <input
                        type="color"
                        value={form.color}
                        onChange={(e) => updateForm('color', e.target.value)}
                        className="h-9 w-full cursor-pointer rounded-md border border-gray-300"
                      />
                    </div>
                    <div className="flex-1">
                      <label className="mb-1 block text-sm font-medium text-gray-700">状态</label>
                      <select
                        value={form.status}
                        onChange={(e) => updateForm('status', e.target.value as 'active' | 'inactive')}
                        className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                      >
                        <option value="active">活跃</option>
                        <option value="inactive">停用</option>
                      </select>
                    </div>
                  </div>
                </div>
                <div className="mt-6 flex justify-end gap-2">
                  <button
                    onClick={() => setShowDialog(false)}
                    className="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-100"
                  >
                    取消
                  </button>
                  <button
                    onClick={handleSave}
                    disabled={saving || !form.code || !form.name}
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
                  >
                    {saving ? '保存中...' : '保存'}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Delete Confirmation */}
          {deleteConfirm !== null && (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
              <div className="w-full max-w-sm rounded-lg bg-white p-6 shadow-lg">
                <h3 className="mb-2 text-lg font-bold text-gray-800">确认删除</h3>
                <p className="mb-4 text-sm text-gray-600">确定要删除该工单类型吗？仅当没有关联工单时可删除。</p>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={() => setDeleteConfirm(null)}
                    className="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-100"
                  >
                    取消
                  </button>
                  <button
                    onClick={() => handleDelete(deleteConfirm)}
                    className="rounded-md bg-red-600 px-4 py-2 text-sm text-white transition-colors hover:bg-red-700"
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          )}
        </main>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/app/ticket-types/page.tsx
git commit -m "feat: add ticket types management page"
```

---

## Task 14: Frontend Tickets Page — Manual Creation + Type Filter

**Files:**
- Modify: `frontend/src/app/tickets/page.tsx`

- [ ] **Step 1: Add manual creation dialog and ticket type filter**

Replace the contents of `frontend/src/app/tickets/page.tsx`:

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { Ticket, PaginatedResponse, TicketType, Client } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';
import TicketTable from '@/components/ticket/ticket-table';

const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'pending', label: '待处理' },
  { value: 'in_progress', label: '处理中' },
  { value: 'completed', label: '已完成' },
  { value: 'failed', label: '失败' },
  { value: 'cancelled', label: '已取消' },
  { value: 'rejected', label: '已拒绝' },
];

const SEVERITY_OPTIONS = [
  { value: '', label: '全部级别' },
  { value: 'critical', label: '严重' },
  { value: 'warning', label: '警告' },
  { value: 'info', label: '信息' },
];

interface ManualForm {
  ticket_type_id: number;
  title: string;
  description: string;
  severity: 'critical' | 'warning' | 'info';
  client_id: number | null;
}

const EMPTY_MANUAL_FORM: ManualForm = {
  ticket_type_id: 0,
  title: '',
  description: '',
  severity: 'warning',
  client_id: null,
};

export default function TicketsPage() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const [page, setPage] = useState(1);
  const [status, setStatus] = useState('');
  const [severity, setSeverity] = useState('');
  const [ticketTypeID, setTicketTypeID] = useState('');
  const [keyword, setKeyword] = useState('');
  const pageSize = 20;

  const [ticketTypes, setTicketTypes] = useState<TicketType[]>([]);
  const [clients, setClients] = useState<Client[]>([]);
  const [showManualDialog, setShowManualDialog] = useState(false);
  const [manualForm, setManualForm] = useState<ManualForm>(EMPTY_MANUAL_FORM);
  const [saving, setSaving] = useState(false);

  const fetchTickets = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
      });
      if (status) params.set('status', status);
      if (severity) params.set('severity', severity);
      if (ticketTypeID) params.set('ticket_type_id', ticketTypeID);
      if (keyword) params.set('keyword', keyword);

      const data = await api.get<PaginatedResponse<Ticket>>(`/tickets?${params.toString()}`);
      setTickets(data.items);
      setTotal(data.total);
    } catch {
      // error handled by api client
    } finally {
      setLoading(false);
    }
  }, [page, status, severity, ticketTypeID, keyword]);

  const fetchTicketTypes = useCallback(async () => {
    try {
      const data = await api.get<{ items: TicketType[] }>('/ticket-types');
      setTicketTypes(data.items);
    } catch {
      // ignore
    }
  }, []);

  const fetchClients = useCallback(async () => {
    try {
      const data = await api.get<PaginatedResponse<Client>>('/clients');
      setClients(data.items);
    } catch {
      // ignore
    }
  }, []);

  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  useEffect(() => {
    fetchTicketTypes();
    fetchClients();
  }, [fetchTicketTypes, fetchClients]);

  function openManualDialog() {
    setManualForm({
      ...EMPTY_MANUAL_FORM,
      ticket_type_id: ticketTypes.length > 0 ? ticketTypes[0].id : 0,
    });
    setShowManualDialog(true);
  }

  async function handleManualCreate() {
    if (!manualForm.ticket_type_id || !manualForm.title || !manualForm.severity) return;
    setSaving(true);
    try {
      const body: Record<string, unknown> = {
        ticket_type_id: manualForm.ticket_type_id,
        title: manualForm.title,
        severity: manualForm.severity,
      };
      if (manualForm.description) body.description = manualForm.description;
      if (manualForm.client_id) body.client_id = manualForm.client_id;

      await api.post('/tickets/manual', body);
      setShowManualDialog(false);
      fetchTickets();
    } catch {
      // error handled by api client
    } finally {
      setSaving(false);
    }
  }

  function updateManualForm<K extends keyof ManualForm>(field: K, value: ManualForm[K]) {
    setManualForm((prev) => ({ ...prev, [field]: value }));
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-bold text-gray-800">工单管理</h2>
            <button
              onClick={openManualDialog}
              className="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-green-700"
            >
              手工建单
            </button>
          </div>

          {/* Filters */}
          <div className="mb-4 flex flex-wrap items-center gap-3">
            <select
              value={status}
              onChange={(e) => { setStatus(e.target.value); setPage(1); }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>

            <select
              value={severity}
              onChange={(e) => { setSeverity(e.target.value); setPage(1); }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            >
              {SEVERITY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>

            <select
              value={ticketTypeID}
              onChange={(e) => { setTicketTypeID(e.target.value); setPage(1); }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            >
              <option value="">全部类型</option>
              {ticketTypes.map((tt) => (
                <option key={tt.id} value={tt.id}>{tt.name}</option>
              ))}
            </select>

            <input
              type="text"
              value={keyword}
              onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
              placeholder="关键词搜索"
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
          </div>

          {/* Table */}
          {loading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <TicketTable tickets={tickets} ticketTypes={ticketTypes} />
          )}

          {/* Pagination */}
          <div className="mt-4 flex items-center justify-between">
            <span className="text-sm text-gray-500">
              共 {total} 条记录，第 {page}/{totalPages} 页
            </span>
            <div className="flex gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                上一页
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50"
              >
                下一页
              </button>
            </div>
          </div>
        </main>
      </div>

      {/* Manual Creation Dialog */}
      {showManualDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="w-full max-w-lg rounded-lg bg-white p-6 shadow-lg">
            <h3 className="mb-4 text-lg font-bold text-gray-800">手工建单</h3>
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">工单类型 *</label>
                <select
                  value={manualForm.ticket_type_id}
                  onChange={(e) => updateManualForm('ticket_type_id', Number(e.target.value))}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                >
                  {ticketTypes.filter((t) => t.status === 'active').map((tt) => (
                    <option key={tt.id} value={tt.id}>
                      {tt.name}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">标题 *</label>
                <input
                  value={manualForm.title}
                  onChange={(e) => updateManualForm('title', e.target.value)}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">描述</label>
                <textarea
                  value={manualForm.description}
                  onChange={(e) => updateManualForm('description', e.target.value)}
                  rows={3}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div className="flex gap-3">
                <div className="flex-1">
                  <label className="mb-1 block text-sm font-medium text-gray-700">严重级别 *</label>
                  <select
                    value={manualForm.severity}
                    onChange={(e) => updateManualForm('severity', e.target.value as 'critical' | 'warning' | 'info')}
                    className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                  >
                    <option value="critical">严重</option>
                    <option value="warning">警告</option>
                    <option value="info">信息</option>
                  </select>
                </div>
                <div className="flex-1">
                  <label className="mb-1 block text-sm font-medium text-gray-700">关联客户</label>
                  <select
                    value={manualForm.client_id ?? ''}
                    onChange={(e) => updateManualForm('client_id', e.target.value ? Number(e.target.value) : null)}
                    className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                  >
                    <option value="">无</option>
                    {clients.map((c) => (
                      <option key={c.id} value={c.id}>{c.name}</option>
                    ))}
                  </select>
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button
                onClick={() => setShowManualDialog(false)}
                className="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-100"
              >
                取消
              </button>
              <button
                onClick={handleManualCreate}
                disabled={saving || !manualForm.ticket_type_id || !manualForm.title}
                className="rounded-md bg-green-600 px-4 py-2 text-sm text-white transition-colors hover:bg-green-700 disabled:opacity-50"
              >
                {saving ? '创建中...' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/app/tickets/page.tsx
git commit -m "feat: add manual ticket creation dialog and type filter"
```

---

## Task 15: Frontend Ticket Table — Show Type

**Files:**
- Modify: `frontend/src/components/ticket/ticket-table.tsx`

- [ ] **Step 1: Add ticket type display to table**

Replace `frontend/src/components/ticket/ticket-table.tsx`:

```tsx
'use client';

import { useRouter } from 'next/navigation';
import type { Ticket, TicketType } from '@/types';
import TicketStatusBadge from './ticket-status-badge';
import SeverityBadge from './severity-badge';

interface TicketTableProps {
  tickets: Ticket[];
  ticketTypes?: TicketType[];
}

export default function TicketTable({ tickets, ticketTypes }: TicketTableProps) {
  const router = useRouter();

  const typeMap = new Map(ticketTypes?.map((t) => [t.id, t]) ?? []);

  if (!tickets || tickets.length === 0) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-8 text-center text-gray-500">
        暂无工单数据
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-200 bg-gray-50">
            <th className="px-4 py-3 text-left font-medium text-gray-600">工单编号</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">标题</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">类型</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">严重级别</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">状态</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">告警源</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">创建时间</th>
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => {
            const tt = ticket.ticket_type_id ? typeMap.get(ticket.ticket_type_id) : undefined;
            return (
              <tr
                key={ticket.id}
                onClick={() => router.push(`/tickets/${ticket.id}`)}
                className="cursor-pointer border-b border-gray-100 transition-colors hover:bg-blue-50"
              >
                <td className="px-4 py-3 font-mono text-blue-600">{ticket.ticket_no}</td>
                <td className="max-w-xs truncate px-4 py-3">{ticket.title}</td>
                <td className="px-4 py-3">
                  {tt ? (
                    <div className="flex items-center gap-1.5">
                      <span
                        className="inline-block h-2.5 w-2.5 rounded-full"
                        style={{ backgroundColor: tt.color }}
                      />
                      <span className="text-xs text-gray-700">{tt.name}</span>
                    </div>
                  ) : (
                    <span className="text-xs text-gray-400">-</span>
                  )}
                </td>
                <td className="px-4 py-3">
                  <SeverityBadge severity={ticket.severity} />
                </td>
                <td className="px-4 py-3">
                  <TicketStatusBadge status={ticket.status} />
                </td>
                <td className="px-4 py-3 text-gray-500">{ticket.source_type}</td>
                <td className="px-4 py-3 text-gray-500">
                  {new Date(ticket.created_at).toLocaleString('zh-CN')}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/ticket/ticket-table.tsx
git commit -m "feat: show ticket type in ticket table"
```

---

## Task 16: Frontend Ticket Detail — Show Type

**Files:**
- Modify: `frontend/src/app/tickets/[id]/page.tsx`

- [ ] **Step 1: Add ticket type info to detail page**

In `frontend/src/app/tickets/[id]/page.tsx`, we need to fetch ticket types to display the type name. Add ticket types state and fetch:

Add to imports:
```typescript
import type { Ticket, WorkflowState, AlertRecord, TicketType } from '@/types';
```

Add state:
```typescript
  const [ticketTypes, setTicketTypes] = useState<TicketType[]>([]);
```

Add fetch in useEffect:
```typescript
  useEffect(() => {
    async function load() {
      try {
        const [data, typesData] = await Promise.all([
          api.get<TicketDetail>(`/tickets/${id}`),
          api.get<{ items: TicketType[] }>('/ticket-types').catch(() => ({ items: [] })),
        ]);
        setTicket(data);
        setTicketTypes(typesData.items);
      } catch {
        // error handled by api client
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);
```

In the Basic info section, add a type row after the title row:

```tsx
              <div>
                <dt className="text-gray-500">工单类型</dt>
                <dd className="mt-0.5">
                  {(() => {
                    const tt = ticket.ticket_type_id ? ticketTypes.find((t) => t.id === ticket.ticket_type_id) : undefined;
                    return tt ? (
                      <div className="flex items-center gap-1.5">
                        <span
                          className="inline-block h-3 w-3 rounded-full"
                          style={{ backgroundColor: tt.color }}
                        />
                        <span>{tt.name}</span>
                      </div>
                    ) : (
                      <span className="text-gray-400">-</span>
                    );
                  })()}
                </dd>
              </div>
```

The full modified file should be:

```tsx
'use client';

import { useState, useEffect, use } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import type { Ticket, WorkflowState, AlertRecord, TicketType } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';
import TicketStatusBadge from '@/components/ticket/ticket-status-badge';
import SeverityBadge from '@/components/ticket/severity-badge';

interface TicketDetail extends Ticket {
  workflow_states: WorkflowState[];
  alert_records: AlertRecord[];
}

const WORKFLOW_STATUS_STYLES: Record<string, string> = {
  pending: 'border-yellow-400 bg-yellow-50',
  running: 'border-blue-400 bg-blue-50',
  completed: 'border-green-400 bg-green-50',
  failed: 'border-red-400 bg-red-50',
  skipped: 'border-gray-400 bg-gray-50',
};

const WORKFLOW_STATUS_LABELS: Record<string, string> = {
  pending: '待执行',
  running: '执行中',
  completed: '已完成',
  failed: '失败',
  skipped: '已跳过',
};

export default function TicketDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [ticket, setTicket] = useState<TicketDetail | null>(null);
  const [ticketTypes, setTicketTypes] = useState<TicketType[]>([]);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    async function load() {
      try {
        const [data, typesData] = await Promise.all([
          api.get<TicketDetail>(`/tickets/${id}`),
          api.get<{ items: TicketType[] }>('/ticket-types').catch(() => ({ items: [] })),
        ]);
        setTicket(data);
        setTicketTypes(typesData.items);
      } catch {
        // error handled by api client
      } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);

  async function handleAction(action: string) {
    if (!ticket) return;
    setActionLoading(true);
    try {
      await api.post(`/tickets/${ticket.id}/${action}`, {});
      const data = await api.get<TicketDetail>(`/tickets/${id}`);
      setTicket(data);
    } catch {
      // error handled by api client
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="flex h-screen">
        <Sidebar />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header />
          <main className="flex-1 items-center justify-center p-6 text-center text-gray-500">
            加载中...
          </main>
        </div>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="flex h-screen">
        <Sidebar />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header />
          <main className="flex flex-1 items-center justify-center p-6 text-center text-gray-500">
            工单不存在
          </main>
        </div>
      </div>
    );
  }

  const canRetry = ['failed'].includes(ticket.status);
  const canCancel = ['pending', 'in_progress'].includes(ticket.status);

  const ticketType = ticket.ticket_type_id
    ? ticketTypes.find((t) => t.id === ticket.ticket_type_id)
    : undefined;

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          {/* Top bar */}
          <div className="mb-6 flex items-center justify-between">
            <button
              onClick={() => router.push('/tickets')}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-100"
            >
              返回列表
            </button>
            <div className="flex gap-2">
              {canRetry && (
                <button
                  onClick={() => handleAction('retry')}
                  disabled={actionLoading}
                  className="rounded-md bg-blue-600 px-4 py-1.5 text-sm text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
                >
                  重试
                </button>
              )}
              {canCancel && (
                <button
                  onClick={() => handleAction('cancel')}
                  disabled={actionLoading}
                  className="rounded-md bg-gray-600 px-4 py-1.5 text-sm text-white transition-colors hover:bg-gray-700 disabled:opacity-50"
                >
                  取消
                </button>
              )}
            </div>
          </div>

          {/* Basic info */}
          <div className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
            <h2 className="mb-4 text-lg font-bold text-gray-800">工单信息</h2>
            <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
              <div>
                <dt className="text-gray-500">工单编号</dt>
                <dd className="mt-0.5 font-mono text-blue-600">{ticket.ticket_no}</dd>
              </div>
              <div>
                <dt className="text-gray-500">状态</dt>
                <dd className="mt-0.5"><TicketStatusBadge status={ticket.status} /></dd>
              </div>
              <div>
                <dt className="text-gray-500">标题</dt>
                <dd className="mt-0.5">{ticket.title}</dd>
              </div>
              <div>
                <dt className="text-gray-500">工单类型</dt>
                <dd className="mt-0.5">
                  {ticketType ? (
                    <div className="flex items-center gap-1.5">
                      <span
                        className="inline-block h-3 w-3 rounded-full"
                        style={{ backgroundColor: ticketType.color }}
                      />
                      <span>{ticketType.name}</span>
                    </div>
                  ) : (
                    <span className="text-gray-400">-</span>
                  )}
                </dd>
              </div>
              <div>
                <dt className="text-gray-500">严重级别</dt>
                <dd className="mt-0.5"><SeverityBadge severity={ticket.severity} /></dd>
              </div>
              <div>
                <dt className="text-gray-500">告警源类型</dt>
                <dd className="mt-0.5">{ticket.source_type}</dd>
              </div>
              <div className="col-span-2">
                <dt className="text-gray-500">描述</dt>
                <dd className="mt-0.5 whitespace-pre-wrap">{ticket.description || '-'}</dd>
              </div>
              <div>
                <dt className="text-gray-500">创建时间</dt>
                <dd className="mt-0.5">{new Date(ticket.created_at).toLocaleString('zh-CN')}</dd>
              </div>
            </dl>
          </div>

          {/* Workflow timeline */}
          <div className="mb-6 rounded-lg border border-gray-200 bg-white p-6">
            <h2 className="mb-4 text-lg font-bold text-gray-800">处理流程</h2>
            {ticket.workflow_states && ticket.workflow_states.length > 0 ? (
              <div className="relative ml-4 border-l-2 border-gray-200 pl-6">
                {ticket.workflow_states.map((ws) => {
                  const style = WORKFLOW_STATUS_STYLES[ws.status] ?? 'border-gray-400 bg-gray-50';
                  const label = WORKFLOW_STATUS_LABELS[ws.status] ?? ws.status;
                  return (
                    <div key={ws.id} className="relative mb-6 last:mb-0">
                      <div className={`absolute -left-[31px] top-1 h-4 w-4 rounded-full border-2 ${style}`} />
                      <div className="rounded-md border border-gray-200 p-3">
                        <div className="flex items-center justify-between">
                          <span className="font-medium text-gray-800">{ws.node_name}</span>
                          <span className={`rounded border px-2 py-0.5 text-xs ${style}`}>{label}</span>
                        </div>
                        <div className="mt-1 flex gap-4 text-xs text-gray-500">
                          {ws.started_at && <span>开始: {new Date(ws.started_at).toLocaleString('zh-CN')}</span>}
                          {ws.completed_at && <span>完成: {new Date(ws.completed_at).toLocaleString('zh-CN')}</span>}
                        </div>
                        {ws.error_message && (
                          <div className="mt-1 text-xs text-red-600">错误: {ws.error_message}</div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <p className="text-sm text-gray-500">暂无流程记录</p>
            )}
          </div>

          {/* Alert records */}
          <div className="rounded-lg border border-gray-200 bg-white p-6">
            <h2 className="mb-4 text-lg font-bold text-gray-800">关联告警</h2>
            {ticket.alert_records && ticket.alert_records.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-gray-200 bg-gray-50">
                      <th className="px-4 py-2 text-left font-medium text-gray-600">ID</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-600">接收时间</th>
                      <th className="px-4 py-2 text-left font-medium text-gray-600">告警内容</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ticket.alert_records.map((ar) => (
                      <tr key={ar.id} className="border-b border-gray-100">
                        <td className="px-4 py-2 text-gray-500">{ar.id}</td>
                        <td className="px-4 py-2 text-gray-500">
                          {new Date(ar.received_at).toLocaleString('zh-CN')}
                        </td>
                        <td className="max-w-md truncate px-4 py-2">
                          {ar.alert_parsed
                            ? JSON.stringify(ar.alert_parsed)
                            : ar.alert_raw
                              ? JSON.stringify(ar.alert_raw)
                              : '-'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-sm text-gray-500">暂无关联告警</p>
            )}
          </div>
        </main>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/app/tickets/\[id\]/page.tsx
git commit -m "feat: show ticket type in ticket detail page"
```

---

## Task 17: Build Verification

**Files:**
- None (verification only)

- [ ] **Step 1: Build backend**

Run: `cd backend && go build ./...`
Expected: PASS (no output)

- [ ] **Step 2: Build frontend**

Run: `cd frontend && npm run build`
Expected: Build completes successfully

- [ ] **Step 3: Run unit tests**

Run: `cd backend && go test ./tests/... -v`
Expected: All unit tests pass (integration tests may be skipped if NT_TEST_DSN not set)

- [ ] **Step 4: Final commit if clean**

```bash
git status
# Should show no uncommitted changes
git log --oneline -5
```

---

## Spec Coverage Check

| Spec Requirement | Implementing Task |
|-----------------|-------------------|
| ticket_types 表创建 | Task 1 |
| tickets 表修改 (alert_source_id nullable, ticket_type_id) | Task 1, 2, 4 |
| TicketType 模型 | Task 2 |
| TicketTypeRepo CRUD | Task 3 |
| TicketTypeService 业务逻辑 | Task 5 |
| TicketTypeHandler admin API | Task 7 |
| 手工建单 API POST /tickets/manual | Task 8 |
| 校验工单类型存在且 active | Task 8 |
| 校验客户存在且 active | Task 8 |
| 构造 ParsedAlert → CreateTicket | Task 8 |
| 创建后设置 ticket_type_id | Task 8 |
| 前端工单类型管理页面 | Task 13 |
| 前端手工建单对话框 | Task 14 |
| 前端工单列表显示类型 | Task 15 |
| 前端工单详情显示类型 | Task 16 |
| 侧边栏导航 | Task 12 |
| 默认类型数据 | Task 1 |
| 编码重复校验 | Task 5 |
| 删除时检查关联工单 | Task 5 |

## Placeholder Scan

- No "TBD", "TODO", "implement later" found.
- All code steps contain complete code.
- All test commands include expected output.
- No vague requirements.

## Type Consistency Check

- `TicketTypeID` used as `*int64` in model, passed as pointer throughout.
- `AlertSourceID` changed to `*int64` consistently.
- `CreateTicket` signature unchanged (alertSourceID still `int64` param, converted to pointer internally).
- `TicketType` model fields match DB schema.
- Frontend `TicketType` interface matches backend JSON tags.
- Handler request struct fields match spec.

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-29-ticket-type.md`.**

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
