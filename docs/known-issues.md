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

---

## 工单类型筛选未实际生效

**状态**：待修复 🟡

**问题描述**：
工单列表页的类型筛选下拉框显示正常（"默认"、"网络故障"、"服务器告警"），但选择后表格数据未按类型过滤，仍显示全部工单。

**根因推测**：
前端传值正确，但后端工单列表 API 未处理 `type` / `ticketType` 筛选参数，或参数名不匹配。

**影响**：
- 用户无法按工单类型快速定位工单
- 筛选器交互正常但功能无效，体验差

**临时解决方案**：
暂无。需后端支持按工单类型编码筛选。

**修复建议**：
1. 检查 `GET /api/v1/tickets` 是否接收并处理类型筛选参数
2. 确认前端传的参数名（如 `type_code`）与后端期望的一致
3. 在 SQL 查询中加入 `ticket_type_id` 或 `ticket_type_code` 过滤条件

---

## 工单详情页"完整流程状态"按钮无响应

**状态**：待修复 🟡

**问题描述**：
点击工单详情页的"完整流程状态 ▼"按钮，无展开/折叠效果，无法查看 7 节点工作流时间线。

**根因推测**：
按钮点击事件未绑定状态切换逻辑，或展开内容未正确渲染。

**影响**：
- 用户无法查看工单的完整处理流程
- 工作流追踪功能不可用

**临时解决方案**：
暂无。需前端修复展开/折叠交互。

**修复建议**：
1. 为按钮添加 `onClick` 事件，切换展开状态（如 `isExpanded`）
2. 根据展开状态条件渲染工作流节点列表
3. 可考虑用时间轴组件展示 7 节点状态（alert_received → parsed → pushed → awaiting_auth → authorized → executing → completed）
