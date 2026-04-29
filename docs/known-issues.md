# 已知问题

## 工单创建后不会自动推送给客户

**状态**：待修复 🔴

**问题描述**：
平台成功接收告警并创建工单，但工单不会自动进入推送队列推送给关联客户。

**根因**：
Worker Pool 已完整实现（`backend/internal/client/worker.go`），但创建工单的代码（`AlertService.Ingest` 和 `TicketService.CreateTicket`）没有调用 `workerPool.Submit()` 提交推送任务。

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
