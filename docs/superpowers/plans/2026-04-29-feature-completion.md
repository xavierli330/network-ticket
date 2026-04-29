# 功能补全实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补全网络工单平台的审计日志前端页面、用户管理前后端、轮询调度器三个功能，并更新文档记录已知问题。

**Architecture:** 遵循现有代码模式：前端用 Next.js + Tailwind 复用现有组件风格；后端用 Gin + sqlx 遵循现有 handler/service/repository 分层；轮询调度器用独立 goroutine + time.Ticker。

**Tech Stack:** Go 1.23, Gin, sqlx, Next.js 16, React 19, Tailwind CSS

---

## 文件结构映射

### 审计日志前端

| 文件 | 操作 | 职责 |
|------|------|------|
| `frontend/src/types/index.ts` | 修改 | 添加 `AuditLog` 接口 |
| `frontend/src/components/layout/sidebar.tsx` | 修改 | 添加「审计日志」导航入口 |
| `frontend/src/app/audit-logs/page.tsx` | 创建 | 审计日志列表页面（筛选 + 表格 + 分页）|
| `backend/internal/repository/audit_log_repo.go` | 修改 | `List` 方法添加 `operator` 筛选参数 |
| `backend/internal/handler/admin_handler.go` | 修改 | `ListAuditLogs` 读取 `operator` 查询参数 |

### 用户管理前后端

| 文件 | 操作 | 职责 |
|------|------|------|
| `backend/internal/repository/user_repo.go` | 修改 | 添加 `ListWithPagination` 和 `Count` 方法 |
| `backend/internal/service/user_service.go` | 创建 | UserService：List, Create, Update, Delete, GetByID |
| `backend/internal/handler/user_handler.go` | 创建 | UserHandler：List, Create, Update, Delete |
| `backend/cmd/server/main.go` | 修改 | 注册 `/users` 路由 |
| `frontend/src/types/index.ts` | 修改 | 添加 `UserCreateRequest` 类型 |
| `frontend/src/components/layout/sidebar.tsx` | 修改 | admin 角色显示「用户管理」入口 |
| `frontend/src/app/users/page.tsx` | 创建 | 用户管理页面（列表 + 新增/编辑/删除对话框）|

### 轮询调度器

| 文件 | 操作 | 职责 |
|------|------|------|
| `backend/internal/poller/worker.go` | 创建 | 单个告警源的轮询 worker |
| `backend/internal/poller/scheduler.go` | 创建 | 调度器：启动/停止/重载所有 worker |
| `backend/internal/poller/poller.go` | 创建 | 包入口，提供 NewScheduler |
| `backend/cmd/server/main.go` | 修改 | 启动 Poller Scheduler |

### 文档更新

| 文件 | 操作 | 职责 |
|------|------|------|
| `docs/user-manual.md` | 修改 | 更新功能实现状态 |
| `docs/known-issues.md` | 创建 | 记录工单推送触发问题 |

---

## Task 1: 审计日志前端页面

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/components/layout/sidebar.tsx`
- Create: `frontend/src/app/audit-logs/page.tsx`
- Modify: `backend/internal/repository/audit_log_repo.go`
- Modify: `backend/internal/handler/admin_handler.go`

---

- [ ] **Step 1: 添加 AuditLog 类型**

在 `frontend/src/types/index.ts` 的 `PaginatedResponse` 之前添加：

```typescript
export interface AuditLog {
  id: number;
  actor: string;
  action: string;
  resource_type: string;
  resource_id?: number;
  detail: unknown;
  ip_address?: string;
  created_at: string;
}
```

- [ ] **Step 2: 修改 sidebar 添加审计日志导航**

修改 `frontend/src/components/layout/sidebar.tsx`，将 `NAV_ITEMS` 改为：

```typescript
const NAV_ITEMS = [
  { label: '工单管理', href: '/tickets' },
  { label: '客户管理', href: '/clients' },
  { label: '告警源管理', href: '/sources' },
  { label: '审计日志', href: '/audit-logs' },
];
```

- [ ] **Step 3: 扩展后端 AuditLogRepo 支持 operator 筛选**

修改 `backend/internal/repository/audit_log_repo.go` 的 `List` 方法：

```go
// List returns a paginated list of audit logs with optional operator filter.
func (r *AuditLogRepo) List(ctx context.Context, page, pageSize int, operator string) ([]model.AuditLog, int, error) {
	// Count total.
	var total int
	countQuery := `SELECT COUNT(*) FROM audit_logs`
	countArgs := []interface{}{}
	if operator != "" {
		countQuery = `SELECT COUNT(*) FROM audit_logs WHERE actor = ?`
		countArgs = append(countArgs, operator)
	}
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count audit_logs: %w", err)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var logs []model.AuditLog
	listQuery := `SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs := []interface{}{pageSize, offset}
	if operator != "" {
		listQuery = `SELECT * FROM audit_logs WHERE actor = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
		listArgs = []interface{}{operator, pageSize, offset}
	}
	if err := r.db.SelectContext(ctx, &logs, listQuery, listArgs...); err != nil {
		return nil, 0, fmt.Errorf("list audit_logs: %w", err)
	}
	return logs, total, nil
}
```

- [ ] **Step 4: 修改 AdminHandler 支持 operator 查询参数**

修改 `backend/internal/handler/admin_handler.go` 的 `ListAuditLogs` 方法：

```go
func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	operator := c.DefaultQuery("operator", "")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	logs, total, err := h.auditRepo.List(c.Request.Context(), page, pageSize, operator)
	if err != nil {
		h.logger.Error("list audit logs failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}
	if logs == nil {
		logs = []model.AuditLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
```

- [ ] **Step 5: 创建审计日志前端页面**

创建 `frontend/src/app/audit-logs/page.tsx`：

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { AuditLog, PaginatedResponse } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';

export default function AuditLogsPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [operator, setOperator] = useState('');
  const pageSize = 20;

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
      });
      if (operator) params.set('operator', operator);

      const data = await api.get<PaginatedResponse<AuditLog>>(`/audit-logs?${params.toString()}`);
      setLogs(data.items);
      setTotal(data.total);
    } catch {
      // error handled by api client
    } finally {
      setLoading(false);
    }
  }, [page, operator]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <h2 className="mb-4 text-xl font-bold text-gray-800">审计日志</h2>

          {/* Filters */}
          <div className="mb-4 flex flex-wrap items-center gap-3">
            <input
              type="text"
              value={operator}
              onChange={(e) => { setOperator(e.target.value); setPage(1); }}
              placeholder="操作人筛选"
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
            <button
              onClick={() => { setOperator(''); setPage(1); }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-100"
            >
              重置
            </button>
          </div>

          {/* Table */}
          {loading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <div className="overflow-x-auto rounded-md border border-gray-200">
              <table className="min-w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">时间</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">操作</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">资源类型</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">资源ID</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">操作人</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">详情</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-700">{new Date(log.created_at).toLocaleString()}</td>
                      <td className="px-4 py-3 text-gray-700">{log.action}</td>
                      <td className="px-4 py-3 text-gray-700">{log.resource_type}</td>
                      <td className="px-4 py-3 text-gray-700">{log.resource_id ?? '-'}</td>
                      <td className="px-4 py-3 text-gray-700">{log.actor}</td>
                      <td className="px-4 py-3 text-gray-700 max-w-xs truncate">{JSON.stringify(log.detail)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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
    </div>
  );
}
```

- [ ] **Step 6: 验证审计日志页面**

启动前后端服务，访问 `http://localhost/audit-logs`：

```bash
# 后端
cd backend && go run cmd/server/main.go

# 前端
cd frontend && npm run dev
```

预期：
- 侧边栏显示「审计日志」入口
- 页面显示审计日志表格（时间、操作、资源类型、资源ID、操作人、详情）
- 操作人筛选框可用
- 分页正常工作

- [ ] **Step 7: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/components/layout/sidebar.tsx frontend/src/app/audit-logs/page.tsx backend/internal/repository/audit_log_repo.go backend/internal/handler/admin_handler.go
git commit -m "feat: 审计日志前端页面 + 后端 operator 筛选支持

- 添加 AuditLog TypeScript 类型
- sidebar 添加审计日志导航
- 创建 /audit-logs 列表页面（筛选 + 分页）
- 扩展后端 List 方法支持 operator 参数筛选"
```

---

## Task 2: 用户管理前后端

**Files:**
- Modify: `backend/internal/repository/user_repo.go`
- Create: `backend/internal/service/user_service.go`
- Create: `backend/internal/handler/user_handler.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/components/layout/sidebar.tsx`
- Create: `frontend/src/app/users/page.tsx`

---

- [ ] **Step 1: 扩展 UserRepo 支持分页和计数**

修改 `backend/internal/repository/user_repo.go`，替换 `List` 方法并添加 `Count`：

```go
// List returns paginated users (password excluded via struct tag).
func (r *UserRepo) List(ctx context.Context, page, pageSize int) ([]model.User, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var users []model.User
	query := `SELECT id, username, role, status, created_at, updated_at FROM users ORDER BY id LIMIT ? OFFSET ?`
	if err := r.db.SelectContext(ctx, &users, query, pageSize, offset); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// Count returns total number of users.
func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var total int
	query := `SELECT COUNT(*) FROM users`
	if err := r.db.GetContext(ctx, &total, query); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return total, nil
}
```

- [ ] **Step 2: 创建 UserService**

创建 `backend/internal/service/user_service.go`：

```go
package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
)

// UserService provides business logic for user operations.
type UserService struct {
	userRepo *repository.UserRepo
	logger   *zap.Logger
}

// NewUserService creates a new UserService.
func NewUserService(userRepo *repository.UserRepo, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// List returns paginated users.
func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	users, err := s.userRepo.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if users == nil {
		users = []model.User{}
	}
	return users, total, nil
}

// GetByID returns a user by ID.
func (s *UserService) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// CreateUser creates a new user with bcrypt hashed password.
func (s *UserService) CreateUser(ctx context.Context, username, password, role string) (*model.User, error) {
	// Check if username already exists.
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &model.User{
		Username: username,
		Password: string(hash),
		Role:     role,
		Status:   "active",
	}

	id, err := s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	u.ID = id
	u.Password = ""
	s.logger.Info("user created", zap.Int64("id", id), zap.String("username", username))
	return u, nil
}

// UpdateUser updates a user's username, role, and optionally password.
func (s *UserService) UpdateUser(ctx context.Context, id int64, username, password, role string) error {
	u, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	u.Username = username
	u.Role = role

	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		u.Password = string(hash)
	}

	if err := s.userRepo.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	s.logger.Info("user updated", zap.Int64("id", id))
	return nil
}

// DeleteUser deletes a user by ID.
func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	s.logger.Info("user deleted", zap.Int64("id", id))
	return nil
}
```

- [ ] **Step 3: 创建 UserHandler**

创建 `backend/internal/handler/user_handler.go`：

```go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/service"
)

// UserHandler handles user management endpoints.
type UserHandler struct {
	userService *service.UserService
	logger      *zap.Logger
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userService *service.UserService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// List returns paginated users.
func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	users, total, err := h.userService.List(c.Request.Context(), page, pageSize)
	if err != nil {
		h.logger.Error("list users failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

// Create creates a new user.
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		h.logger.Error("create user failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password"`
	Role     string `json:"role" binding:"required"`
}

// Update updates a user.
func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.userService.UpdateUser(c.Request.Context(), id, req.Username, req.Password, req.Role); err != nil {
		h.logger.Error("update user failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete deletes a user.
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Prevent deleting current user.
	currentUserID, _ := c.Get("user_id")
	if currentUserID == id {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete yourself"})
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), id); err != nil {
		h.logger.Error("delete user failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
```

- [ ] **Step 4: 注册用户管理路由**

修改 `backend/cmd/server/main.go`：

在 `// 6. Create services` 部分添加：
```go
userService := service.NewUserService(userRepo, logger)
```

在 `// 8. Create handlers` 部分添加：
```go
userHandler := handler.NewUserHandler(userService, logger)
```

在路由注册部分（audit-logs 路由之后）添加：
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

- [ ] **Step 5: 添加前端 User 类型**

修改 `frontend/src/types/index.ts`，在 `User` 接口后添加：

```typescript
export interface UserCreateRequest {
  username: string;
  password: string;
  role: 'admin' | 'operator';
}

export interface UserUpdateRequest {
  username: string;
  password?: string;
  role: 'admin' | 'operator';
}
```

- [ ] **Step 6: 修改 sidebar 条件渲染用户管理**

修改 `frontend/src/components/layout/sidebar.tsx`，需要读取当前用户角色来条件渲染：

```tsx
'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useState } from 'react';

interface NavItem {
  label: string;
  href: string;
  adminOnly?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { label: '工单管理', href: '/tickets' },
  { label: '客户管理', href: '/clients', adminOnly: true },
  { label: '告警源管理', href: '/sources' },
  { label: '审计日志', href: '/audit-logs' },
  { label: '用户管理', href: '/users', adminOnly: true },
];

export default function Sidebar() {
  const pathname = usePathname();
  const [role, setRole] = useState<string | null>(null);

  useEffect(() => {
    try {
      const userStr = localStorage.getItem('user');
      if (userStr) {
        const user = JSON.parse(userStr);
        setRole(user.role);
      }
    } catch {
      // ignore
    }
  }, []);

  const visibleItems = NAV_ITEMS.filter((item) => {
    if (!item.adminOnly) return true;
    return role === 'admin';
  });

  return (
    <aside className="flex h-screen w-56 flex-col border-r border-gray-200 bg-white">
      <div className="border-b border-gray-200 px-5 py-4">
        <h1 className="text-lg font-bold text-gray-800">网络工单平台</h1>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {visibleItems.map((item) => {
          const active = pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`block rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                active
                  ? 'bg-blue-50 text-blue-700'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
              }`}
            >
              {item.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
```

> **注意**：这里假设登录时把 user 对象存到了 localStorage。需要检查 `frontend/src/app/login/page.tsx` 是否存储了 user 信息。如果只在 localStorage 存了 token，需要从 token 中解析 role，或者修改登录逻辑存储 user。

检查登录页面的存储逻辑：

```bash
grep -n "localStorage" frontend/src/app/login/page.tsx
```

如果登录页面没有存储 user 对象，需要修改登录页面：

```tsx
// 在登录成功回调中
api.setToken(data.token);
localStorage.setItem('user', JSON.stringify(data.user));
```

- [ ] **Step 7: 创建用户管理前端页面**

创建 `frontend/src/app/users/page.tsx`：

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { User, PaginatedResponse, UserCreateRequest, UserUpdateRequest } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  // Dialog states
  const [showDialog, setShowDialog] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formUsername, setFormUsername] = useState('');
  const [formPassword, setFormPassword] = useState('');
  const [formRole, setFormRole] = useState<'admin' | 'operator'>('operator');
  const [submitting, setSubmitting] = useState(false);

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<PaginatedResponse<User>>(`/users?page=${page}&page_size=${pageSize}`);
      setUsers(data.items);
      setTotal(data.total);
    } catch {
      // error handled by api client
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const openCreate = () => {
    setEditingUser(null);
    setFormUsername('');
    setFormPassword('');
    setFormRole('operator');
    setShowDialog(true);
  };

  const openEdit = (user: User) => {
    setEditingUser(user);
    setFormUsername(user.username);
    setFormPassword('');
    setFormRole(user.role);
    setShowDialog(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      if (editingUser) {
        const body: UserUpdateRequest = {
          username: formUsername,
          role: formRole,
        };
        if (formPassword) body.password = formPassword;
        await api.put(`/users/${editingUser.id}`, body);
      } else {
        const body: UserCreateRequest = {
          username: formUsername,
          password: formPassword,
          role: formRole,
        };
        await api.post('/users', body);
      }
      setShowDialog(false);
      fetchUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : '操作失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (user: User) => {
    if (!confirm(`确定删除用户 "${user.username}" 吗？`)) return;
    try {
      await api.delete(`/users/${user.id}`);
      fetchUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : '删除失败');
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-bold text-gray-800">用户管理</h2>
            <button
              onClick={openCreate}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              新增用户
            </button>
          </div>

          {loading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <div className="overflow-x-auto rounded-md border border-gray-200">
              <table className="min-w-full text-sm">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">ID</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">用户名</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">角色</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">创建时间</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {users.map((user) => (
                    <tr key={user.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-700">{user.id}</td>
                      <td className="px-4 py-3 text-gray-700">{user.username}</td>
                      <td className="px-4 py-3 text-gray-700">
                        <span
                          className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                            user.role === 'admin'
                              ? 'bg-purple-100 text-purple-700'
                              : 'bg-gray-100 text-gray-700'
                          }`}
                        >
                          {user.role === 'admin' ? '管理员' : '操作员'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-gray-700">{new Date(user.created_at).toLocaleString()}</td>
                      <td className="px-4 py-3">
                        <div className="flex gap-2">
                          <button
                            onClick={() => openEdit(user)}
                            className="rounded-md border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-100"
                          >
                            编辑
                          </button>
                          <button
                            onClick={() => handleDelete(user)}
                            className="rounded-md border border-red-300 px-2 py-1 text-xs text-red-600 hover:bg-red-50"
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
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

      {/* Dialog */}
      {showDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg bg-white p-6 shadow-lg">
            <h3 className="mb-4 text-lg font-bold text-gray-800">
              {editingUser ? '编辑用户' : '新增用户'}
            </h3>
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label className="mb-1 block text-sm font-medium text-gray-700">用户名</label>
                <input
                  type="text"
                  value={formUsername}
                  onChange={(e) => setFormUsername(e.target.value)}
                  required
                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                />
              </div>
              <div className="mb-4">
                <label className="mb-1 block text-sm font-medium text-gray-700">
                  密码 {editingUser && <span className="text-gray-400">（留空表示不修改）</span>}
                </label>
                <input
                  type="password"
                  value={formPassword}
                  onChange={(e) => setFormPassword(e.target.value)}
                  required={!editingUser}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                />
              </div>
              <div className="mb-4">
                <label className="mb-1 block text-sm font-medium text-gray-700">角色</label>
                <select
                  value={formRole}
                  onChange={(e) => setFormRole(e.target.value as 'admin' | 'operator')}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                >
                  <option value="operator">操作员</option>
                  <option value="admin">管理员</option>
                </select>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowDialog(false)}
                  className="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-600 hover:bg-gray-100"
                >
                  取消
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {submitting ? '保存中...' : '保存'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 8: 验证用户管理功能**

启动前后端，以 admin 身份登录：

1. 侧边栏显示「用户管理」入口 ✅
2. 点击「新增用户」，填写信息，保存 ✅
3. 新用户出现在列表中 ✅
4. 点击「编辑」，修改角色或密码，保存 ✅
5. 点击「删除」，确认后用户被删除 ✅
6. 尝试删除当前登录用户，应被拒绝 ✅
7. 以 operator 身份登录，侧边栏不显示「用户管理」和「客户管理」✅

- [ ] **Step 9: Commit**

```bash
git add backend/internal/repository/user_repo.go backend/internal/service/user_service.go backend/internal/handler/user_handler.go backend/cmd/server/main.go frontend/src/types/index.ts frontend/src/components/layout/sidebar.tsx frontend/src/app/users/page.tsx
git commit -m "feat: 用户管理前后端完整实现

- 后端：UserService + UserHandler（List/Create/Update/Delete）
- 后端：用户列表分页、bcrypt 密码哈希、删除自我保护
- 前端：/users 页面（列表 + 新增/编辑/删除对话框）
- 前端：sidebar 按角色条件渲染导航"
```

---

## Task 3: 轮询调度器

**Files:**
- Create: `backend/internal/poller/worker.go`
- Create: `backend/internal/poller/scheduler.go`
- Create: `backend/internal/poller/poller.go`
- Modify: `backend/cmd/server/main.go`

---

- [ ] **Step 1: 创建 worker.go**

创建 `backend/internal/poller/worker.go`：

```go
package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/service"
)

// Worker polls a single alert source at regular intervals.
type Worker struct {
	source      *model.AlertSource
	alertService *service.AlertService
	logger      *zap.Logger
	client      *http.Client
	stopCh      chan struct{}
}

// NewWorker creates a new polling worker for an alert source.
func NewWorker(source *model.AlertSource, alertService *service.AlertService, logger *zap.Logger) *Worker {
	return &Worker{
		source:       source,
		alertService: alertService,
		logger:       logger,
		client:       &http.Client{Timeout: 30 * time.Second},
		stopCh:       make(chan struct{}),
	}
}

// Start begins polling in a background goroutine.
func (w *Worker) Start() {
	go w.run()
}

// Stop signals the worker to stop.
func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) run() {
	interval := time.Duration(w.source.PollInterval) * time.Second
	if interval <= 0 {
		interval = 60 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Poll immediately on start.
	w.poll()

	for {
		select {
		case <-w.stopCh:
			w.logger.Info("poller worker stopped", zap.Int64("source_id", w.source.ID))
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *Worker) poll() {
	w.logger.Debug("polling alert source",
		zap.Int64("source_id", w.source.ID),
		zap.String("endpoint", w.source.PollEndpoint),
	)

	req, err := http.NewRequest(http.MethodGet, w.source.PollEndpoint, nil)
	if err != nil {
		w.logger.Error("failed to create poll request", zap.Error(err))
		return
	}

	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Warn("poll request failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logger.Warn("poll request returned non-200",
			zap.Int("status", resp.StatusCode),
		)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("failed to read poll response", zap.Error(err))
		return
	}

	// Try to parse as array first.
	var alerts []json.RawMessage
	if err := json.Unmarshal(body, &alerts); err != nil {
		// Try single alert.
		var single json.RawMessage
		if err := json.Unmarshal(body, &single); err != nil {
			w.logger.Error("failed to parse poll response as JSON", zap.Error(err))
			return
		}
		alerts = []json.RawMessage{single}
	}

	w.logger.Info("poll received alerts",
		zap.Int64("source_id", w.source.ID),
		zap.Int("count", len(alerts)),
	)

	for i, alert := range alerts {
		if _, err := w.alertService.Ingest(context.Background(), w.source.ID, alert); err != nil {
			w.logger.Error("failed to ingest polled alert",
				zap.Int64("source_id", w.source.ID),
				zap.Int("index", i),
				zap.Error(err),
			)
		}
	}
}
```

- [ ] **Step 2: 创建 scheduler.go**

创建 `backend/internal/poller/scheduler.go`：

```go
package poller

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/xavierli/network-ticket/internal/model"
	"github.com/xavierli/network-ticket/internal/repository"
	"github.com/xavierli/network-ticket/internal/service"
)

// Scheduler manages polling workers for all alert sources with poll endpoints.
type Scheduler struct {
	alertSourceRepo *repository.AlertSourceRepo
	alertService    *service.AlertService
	logger          *zap.Logger

	workers map[int64]*Worker
	mu      sync.RWMutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler creates a new polling scheduler.
func NewScheduler(
	alertSourceRepo *repository.AlertSourceRepo,
	alertService *service.AlertService,
	logger *zap.Logger,
) *Scheduler {
	return &Scheduler{
		alertSourceRepo: alertSourceRepo,
		alertService:    alertService,
		logger:          logger,
		workers:         make(map[int64]*Worker),
		stopCh:          make(chan struct{}),
	}
}

// Start initializes and starts all polling workers.
func (s *Scheduler) Start() error {
	sources, err := s.loadPollableSources()
	if err != nil {
		return err
	}

	for _, source := range sources {
		s.startWorker(source)
	}

	s.logger.Info("poller scheduler started", zap.Int("workers", len(s.workers)))

	// Start periodic reload goroutine.
	s.wg.Add(1)
	go s.reloadLoop()

	return nil
}

// Stop gracefully stops all workers.
func (s *Scheduler) Stop() {
	close(s.stopCh)

	s.mu.Lock()
	for id, worker := range s.workers {
		worker.Stop()
		delete(s.workers, id)
	}
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("poller scheduler stopped")
}

// Reload immediately refreshes the worker list from database.
func (s *Scheduler) Reload() error {
	sources, err := s.loadPollableSources()
	if err != nil {
		return err
	}

	// Build set of desired source IDs.
	desired := make(map[int64]*model.AlertSource)
	for i := range sources {
		desired[sources[i].ID] = &sources[i]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop workers for removed sources.
	for id, worker := range s.workers {
		if _, ok := desired[id]; !ok {
			worker.Stop()
			delete(s.workers, id)
			s.logger.Info("poller worker removed", zap.Int64("source_id", id))
		}
	}

	// Start workers for new sources.
	for id, source := range desired {
		if _, ok := s.workers[id]; !ok {
			s.startWorkerLocked(source)
			s.logger.Info("poller worker added", zap.Int64("source_id", id))
		}
	}

	return nil
}

func (s *Scheduler) loadPollableSources() ([]model.AlertSource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	all, err := s.alertSourceRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	var pollable []model.AlertSource
	for _, source := range all {
		if source.PollEndpoint != "" && source.Status == model.AlertSourceStatusActive {
			pollable = append(pollable, source)
		}
	}
	return pollable, nil
}

func (s *Scheduler) startWorker(source *model.AlertSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startWorkerLocked(source)
}

func (s *Scheduler) startWorkerLocked(source *model.AlertSource) {
	worker := NewWorker(source, s.alertService, s.logger)
	s.workers[source.ID] = worker
	worker.Start()
}

func (s *Scheduler) reloadLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.Reload(); err != nil {
				s.logger.Error("poller reload failed", zap.Error(err))
			}
		}
	}
}
```

- [ ] **Step 3: 创建 poller.go（包入口）**

创建 `backend/internal/poller/poller.go`：

```go
// Package poller provides periodic polling of alert sources that don't support webhooks.
package poller
```

- [ ] **Step 4: 修改 main.go 启动调度器**

修改 `backend/cmd/server/main.go`：

在 `// 7. Create and start worker pool` 之后添加：

```go
// ---------------------------------------------------------------------------
// 7.5. Create and start poller scheduler
// ---------------------------------------------------------------------------
pollerScheduler := poller.NewScheduler(alertSourceRepo, alertService, logger)
if err := pollerScheduler.Start(); err != nil {
	logger.Error("failed to start poller scheduler", zap.Error(err))
} else {
	defer pollerScheduler.Stop()
}
```

在 import 部分添加：
```go
"github.com/xavierli/network-ticket/internal/poller"
```

- [ ] **Step 5: 验证轮询调度器**

1. 创建一个带轮询配置的告警源：
   ```json
   {
     "name": "测试轮询",
     "type": "generic",
     "poll_endpoint": "http://httpbin.org/get",
     "poll_interval": 60,
     "status": "active"
   }
   ```

2. 启动后端，观察日志中是否有 "poller scheduler started" 和 "polling alert source" 日志

3. 检查是否正确解析响应并创建工单

4. 删除或禁用该告警源，30 秒后观察 worker 是否被移除

- [ ] **Step 6: Commit**

```bash
git add backend/internal/poller/ backend/cmd/server/main.go
git commit -m "feat: 告警源轮询调度器

- 新增 poller 包：Worker + Scheduler
- Worker：定时 GET 轮询端点，解析 JSON 数组/单条告警
- Scheduler：管理所有 worker 生命周期，每 30 秒自动重载配置
- 后端启动时自动启动调度器，关闭时优雅停止"
```

---

## Task 4: 文档更新

**Files:**
- Modify: `docs/user-manual.md`
- Create: `docs/known-issues.md`

---

- [ ] **Step 1: 更新用户手册功能状态**

修改 `docs/user-manual.md`：

1. 将「审计日志」部分的 `⚠️ 当前状态：后端 API 完整，但前端查看页面尚未开发」改为 `✅ 已实现：审计日志前端页面可用，支持按操作人筛选和分页。`

2. 将「用户管理」部分的「⚠️ 当前版本没有前端用户管理页面」改为 `✅ 已实现：用户管理前后端完整，仅 admin 可访问。`

3. 将「轮询配置」部分的「⚠️ 配置界面已显示这两个字段，但后端轮询调度器尚未实现」改为 `✅ 已实现：轮询调度器自动运行，定时拉取告警。`

- [ ] **Step 2: 创建 known-issues.md**

创建 `docs/known-issues.md`：

```markdown
# 已知问题

## 工单创建后不会自动推送给客户

**状态**：待修复 🔴

**问题描述**：
平台成功接收告警并创建工单，但工单不会自动进入推送队列推送给关联客户。

**根因**：
Worker Pool 已完整实现（`backend/internal/client/worker.go`），但创建工单的代码（`TicketService.CreateTicket` 或 `AlertService.Ingest`）没有调用 `workerPool.Submit()` 提交推送任务。

**影响**：
- 工单状态停留在 `pending`，不会变为 `in_progress`
- 客户系统收不到工单推送
- 需要手动通过 API 或其他方式触发推送

**临时解决方案**：
暂无。需要修改 `AlertService.Ingest` 或 `TicketService.CreateTicket`，在创建工单后调用 Worker Pool 提交推送任务。

**修复建议**：
在 `AlertService.Ingest` 创建新工单后，调用 `workerPool.Submit(ticket)` 将工单加入推送队列。需要：
1. 将 `WorkerPool` 注入 `AlertService`
2. 在 `Ingest` 方法中，创建工单成功后提交推送任务
3. 确保推送成功时更新工单状态为 `in_progress`

**相关代码**：
- `backend/internal/client/worker.go` - Worker Pool 实现
- `backend/internal/service/alert_service.go` - Ingest 方法
```

- [ ] **Step 3: Commit**

```bash
git add docs/user-manual.md docs/known-issues.md
git commit -m "docs: 更新功能状态 + 记录工单推送已知问题

- 用户手册：标记审计日志、用户管理、轮询调度器为已实现
- 新建 known-issues.md 记录工单推送触发问题"
```

---

## Spec Coverage Check

| Spec 需求 | 对应 Task/Step |
|-----------|---------------|
| 审计日志前端页面 | Task 1, Steps 1-7 |
| 审计日志 operator 筛选 | Task 1, Steps 3-4 |
| 用户管理后端 API | Task 2, Steps 1-4 |
| 用户管理前端页面 | Task 2, Steps 5-7 |
| 用户删除自我保护 | Task 2, Step 3 (Delete handler) |
| 轮询调度器 Worker | Task 3, Step 1 |
| 轮询调度器 Scheduler | Task 3, Steps 2-4 |
| 轮询定期重载 | Task 3, Step 2 (reloadLoop) |
| 文档更新 | Task 4, Steps 1-3 |
| 工单推送问题记录 | Task 4, Step 2 |

**无遗漏。**

## Placeholder Check

- 无 "TBD", "TODO", "implement later"
- 无 "Add appropriate error handling" 等模糊描述
- 所有代码步骤包含完整代码
- 无 "Similar to Task N" 引用

**通过。**

## Type Consistency Check

- `AuditLog` 类型：前后端字段一致（actor, action, resource_type, resource_id, detail, ip_address, created_at）
- `User` 类型：后端 `model.User` 的 JSON tag 与前端 `User` 接口一致
- `PaginatedResponse<T>`：前后端结构一致（items, total, page, page_size）
- `CreateUserRequest` / `UpdateUserRequest`：handler 和 frontend 类型一致

**通过。**
