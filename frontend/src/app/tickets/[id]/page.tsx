'use client';

import { useState, useEffect, use } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import type { Ticket, WorkflowState, AlertRecord, Client } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';
import TicketStatusBadge from '@/components/ticket/ticket-status-badge';
import SeverityBadge from '@/components/ticket/severity-badge';

interface TicketDetail extends Ticket {
  workflow_states: WorkflowState[];
  alert_records: AlertRecord[];
}

interface TicketResponse {
  ticket: TicketDetail;
  workflow_states: WorkflowState[];
  alert_records?: AlertRecord[];
}

const WORKFLOW_LABELS: Record<string, string> = {
  alert_received: '告警接收',
  parsed: '告警解析',
  pushed: '推送客户',
  awaiting_auth: '等待授权',
  authorized: '客户授权',
  executing: '执行动作',
  completed: '处理完成',
};

const WORKFLOW_STATUS_ICON: Record<string, string> = {
  done: '✅',
  pending: '⏳',
  active: '🔵',
  failed: '❌',
  skipped: '⏭️',
};

function fmtTime(iso?: string | null) {
  if (!iso) return '-';
  return new Date(iso).toLocaleString('zh-CN');
}

export default function TicketDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const [ticket, setTicket] = useState<TicketDetail | null>(null);
  const [client, setClient] = useState<Client | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [showWorkflow, setShowWorkflow] = useState(false);

  useEffect(() => {
    load();
  }, [id]);

  async function load() {
    try {
      const data = await api.get<TicketResponse>(`/tickets/${id}`);
      const detail: TicketDetail = { ...data.ticket, workflow_states: data.workflow_states, alert_records: data.alert_records ?? [] };
      setTicket(detail);
      if (detail.client_id) {
        try {
          const clients = await api.get<{ items: Client[] }>('/clients');
          setClient(clients.items.find((c) => c.id === detail.client_id) ?? null);
        } catch { /* ignore */ }
      }
    } catch {
      // error handled by api client
    } finally {
      setLoading(false);
    }
  }

  async function transitionStatus(status: string, operator?: string) {
    if (!ticket) return;
    setActionLoading(true);
    try {
      await api.put(`/tickets/${ticket.id}`, { status, operator: operator ?? 'manual' });
      await load();
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
          <main className="flex flex-1 items-center justify-center p-6 text-gray-500">加载中...</main>
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
          <main className="flex flex-1 items-center justify-center p-6 text-gray-500">工单不存在</main>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          {/* Top bar */}
          <div className="mb-6 flex items-center gap-4">
            <button onClick={() => router.push('/tickets')} className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-100">
              ← 返回列表
            </button>
            <TicketStatusBadge status={ticket.status} />
          </div>

          <div className="space-y-6">
            {/* 工单信息 */}
            <InfoCard title="工单信息">
              <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
                <Field label="编号" value={<span className="font-mono text-blue-600">{ticket.ticket_no}</span>} />
                <Field label="严重级别" value={<SeverityBadge severity={ticket.severity} />} />
                <Field label="工单类型" value={
                  ticket.ticket_type
                    ? <span className="rounded px-2 py-0.5 text-xs text-white" style={{ backgroundColor: ticket.ticket_type.color }}>{ticket.ticket_type.name}</span>
                    : '-'
                } />
                <Field label="告警源类型" value={ticket.source_type} />
                <Field label="创建时间" value={fmtTime(ticket.created_at)} />
                <Field label="更新时间" value={fmtTime(ticket.updated_at)} />
                <div className="col-span-2">
                  <dt className="text-gray-500">标题</dt>
                  <dd className="mt-0.5 font-medium">{ticket.title}</dd>
                </div>
                {ticket.description && (
                  <div className="col-span-2">
                    <dt className="text-gray-500">描述</dt>
                    <dd className="mt-0.5 whitespace-pre-wrap text-gray-700">{ticket.description}</dd>
                  </div>
                )}
              </dl>
            </InfoCard>

            {/* 当前状态 */}
            <StatusPanel ticket={ticket} client={client} actionLoading={actionLoading} onAction={transitionStatus} />

            {/* 关联客户 */}
            {client && (
              <InfoCard title="关联客户">
                <dl className="grid grid-cols-2 gap-x-8 gap-y-3 text-sm">
                  <Field label="客户名称" value={client.name} />
                  <Field label="状态" value={client.status === 'active' ? '活跃' : client.status} />
                  <div className="col-span-2">
                    <Field label="推送地址" value={<span className="font-mono text-xs break-all">{client.api_endpoint}</span>} />
                  </div>
                </dl>
              </InfoCard>
            )}

            {/* 折叠：完整流程状态 */}
            <div className="rounded-lg border border-gray-200 bg-white">
              <button onClick={() => setShowWorkflow(!showWorkflow)} className="flex w-full items-center justify-between px-5 py-4 text-left">
                <span className="text-sm font-semibold text-gray-800">完整流程状态</span>
                <span className="text-gray-400">{showWorkflow ? '▲' : '▼'}</span>
              </button>
              {showWorkflow && (
                <div className="border-t border-gray-100 px-5 py-4">
                  <div className="space-y-2">
                    {ticket.workflow_states.map((ws) => (
                      <div key={ws.id} className="flex items-center gap-3 text-sm">
                        <span>{WORKFLOW_STATUS_ICON[ws.status] ?? '⏳'}</span>
                        <span className="w-24 text-gray-500">{WORKFLOW_LABELS[ws.node_name] ?? ws.node_name}</span>
                        <span className="font-medium">{ws.status}</span>
                        {ws.error_message && <span className="text-xs text-red-500">({ws.error_message})</span>}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* 告警历史 */}
            {ticket.alert_records.length > 0 && (
              <InfoCard title="告警历史">
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
                          <td className="px-4 py-2 text-gray-500">{fmtTime(ar.received_at)}</td>
                          <td className="max-w-md truncate px-4 py-2">
                            {ar.alert_parsed ? JSON.stringify(ar.alert_parsed) : ar.alert_raw ? JSON.stringify(ar.alert_raw) : '-'}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </InfoCard>
            )}
          </div>
        </main>
      </div>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────────── */

function StatusPanel({ ticket, client, actionLoading, onAction }: {
  ticket: TicketDetail;
  client: Client | null;
  actionLoading: boolean;
  onAction: (status: string, operator?: string) => void;
}) {
  const ws = ticket.workflow_states;
  const findNode = (name: string) => ws.find((n) => n.node_name === name);
  const pushed = findNode('pushed');
  const authorized = findNode('authorized');

  // 根据工单状态渲染不同面板
  switch (ticket.status) {
    case 'pending':
      return <PendingPanel ticket={ticket} client={client} pushed={pushed} actionLoading={actionLoading} onAction={onAction} />;
    case 'in_progress':
      return <InProgressPanel ticket={ticket} client={client} authorized={authorized} actionLoading={actionLoading} onAction={onAction} />;
    case 'completed':
      return <CompletedPanel ticket={ticket} authorized={authorized} />;
    case 'failed':
      return <FailedPanel ticket={ticket} client={client} pushed={pushed} actionLoading={actionLoading} onAction={onAction} />;
    case 'rejected':
      return <RejectedPanel ticket={ticket} authorized={authorized} actionLoading={actionLoading} onAction={onAction} />;
    case 'cancelled':
      return <CancelledPanel ticket={ticket} />;
    default:
      return null;
  }
}

/* ── pending ───────────────────────────────────────────────────── */

function PendingPanel({ ticket, client, pushed, actionLoading, onAction }: {
  ticket: TicketDetail; client: Client | null; pushed?: WorkflowState; actionLoading: boolean;
  onAction: (s: string, o?: string) => void;
}) {
  return (
    <StatusCard>
      <StatusHeader icon="🟡" title="等待推送" />
      <p className="text-sm text-gray-600">
        工单已创建，{client ? `等待推送给客户"${client.name}"` : '未关联客户，无法自动推送'}
      </p>

      <div className="mt-3 space-y-1 text-sm">
        <Row label="推送状态" value={pushed?.status === 'done' ? '✅ 已推送' : pushed?.status === 'failed' ? '❌ 推送失败' : '⏳ 尚未推送'} />
        <Row label="待处理人" value="系统（自动推送）" />
      </div>

      <Divider />
      <div className="flex gap-2">
        <ActionBtn label="取消工单" variant="secondary" loading={actionLoading} onClick={() => onAction('cancelled')} />
      </div>
    </StatusCard>
  );
}

/* ── in_progress ───────────────────────────────────────────────── */

function InProgressPanel({ ticket, client, authorized, actionLoading, onAction }: {
  ticket: TicketDetail; client: Client | null; authorized?: WorkflowState; actionLoading: boolean;
  onAction: (s: string, o?: string) => void;
}) {
  const isAuthorized = authorized?.status === 'done';
  const clientName = client?.name ?? '客户系统';

  if (!isAuthorized) {
    // 等待授权
    return (
      <StatusCard>
        <StatusHeader icon="🔵" title="等待授权" />
        <p className="text-sm text-gray-600">工单已推送至客户"{clientName}"，等待客户授权处理</p>

        <div className="mt-3 space-y-1 text-sm">
          <Row label="推送状态" value="✅ 推送成功" />
          <Row label="授权状态" value="⏳ 等待客户响应" />
          <Row label="待处理人" value={`客户"${clientName}"`} />
        </div>

        <Divider />
        <p className="mb-2 text-xs font-medium text-gray-500">授权工具（模拟客户操作）</p>
        <div className="flex gap-2">
          <ActionBtn label="✅ 授权处理" variant="primary" loading={actionLoading} onClick={() => onAction('in_progress', 'authorize')} />
          <ActionBtn label="❌ 拒绝处理" variant="danger" loading={actionLoading} onClick={() => onAction('rejected', 'reject')} />
        </div>
        <p className="mt-3 text-xs font-medium text-gray-500">管理工具</p>
        <div className="mt-2 flex gap-2">
          <ActionBtn label="取消工单" variant="secondary" loading={actionLoading} onClick={() => onAction('cancelled')} />
        </div>
      </StatusCard>
    );
  }

  // 已授权，执行中
  return (
    <StatusCard>
      <StatusHeader icon="🔵" title="执行中" />
      <p className="text-sm text-gray-600">客户已授权，等待执行团队处理完成</p>

      <div className="mt-3 rounded-md bg-green-50 border border-green-200 px-3 py-2 text-sm">
        <div className="flex items-center gap-2 font-medium text-green-800">✅ 已授权</div>
        {authorized?.operator && <div className="mt-1 text-xs text-green-700">授权人: {authorized.operator}</div>}
        {authorized?.completed_at && <div className="text-xs text-green-700">授权时间: {fmtTime(authorized.completed_at)}</div>}
      </div>

      <div className="mt-3 space-y-1 text-sm">
        <Row label="执行状态" value="⏳ 等待执行" />
        <Row label="待处理人" value="执行团队" />
      </div>

      <Divider />
      <div className="flex gap-2">
        <ActionBtn label="✓ 标记完成" variant="primary" loading={actionLoading} onClick={() => onAction('completed')} />
        <ActionBtn label="取消工单" variant="secondary" loading={actionLoading} onClick={() => onAction('cancelled')} />
      </div>
    </StatusCard>
  );
}

/* ── completed ─────────────────────────────────────────────────── */

function CompletedPanel({ ticket, authorized }: { ticket: TicketDetail; authorized?: WorkflowState }) {
  return (
    <StatusCard>
      <StatusHeader icon="🟢" title="已完成" />
      <p className="text-sm text-gray-600">工单处理已完成</p>
      {authorized?.status === 'done' && (
        <div className="mt-3 rounded-md bg-green-50 border border-green-200 px-3 py-2 text-sm">
          <div className="font-medium text-green-800">✅ 已授权</div>
          {authorized.operator && <div className="mt-1 text-xs text-green-700">授权人: {authorized.operator}</div>}
          {authorized.completed_at && <div className="text-xs text-green-700">授权时间: {fmtTime(authorized.completed_at)}</div>}
        </div>
      )}
      <div className="mt-3 space-y-1 text-sm">
        <Row label="完成时间" value={fmtTime(ticket.updated_at)} />
      </div>
    </StatusCard>
  );
}

/* ── failed ────────────────────────────────────────────────────── */

function FailedPanel({ ticket, client, pushed, actionLoading, onAction }: {
  ticket: TicketDetail; client: Client | null; pushed?: WorkflowState; actionLoading: boolean;
  onAction: (s: string, o?: string) => void;
}) {
  return (
    <StatusCard>
      <StatusHeader icon="🔴" title="推送失败" />
      <p className="text-sm text-gray-600">推送至客户系统失败</p>

      <div className="mt-3 space-y-1 text-sm">
        <Row label="推送状态" value="❌ 失败" />
        {pushed?.error_message && <Row label="失败原因" value={pushed.error_message} />}
        <Row label="待处理人" value="操作员（检查客户系统）" />
      </div>

      <Divider />
      <div className="flex gap-2">
        <ActionBtn label="🔄 重试推送" variant="primary" loading={actionLoading} onClick={() => onAction('pending', 'retry')} />
        <ActionBtn label="取消工单" variant="secondary" loading={actionLoading} onClick={() => onAction('cancelled')} />
      </div>
    </StatusCard>
  );
}

/* ── rejected ──────────────────────────────────────────────────── */

function RejectedPanel({ ticket, authorized, actionLoading, onAction }: {
  ticket: TicketDetail; authorized?: WorkflowState; actionLoading: boolean;
  onAction: (s: string, o?: string) => void;
}) {
  return (
    <StatusCard>
      <StatusHeader icon="🟠" title="已拒绝" />
      <p className="text-sm text-gray-600">客户拒绝处理此工单</p>

      <div className="mt-3 rounded-md bg-orange-50 border border-orange-200 px-3 py-2 text-sm">
        <div className="font-medium text-orange-800">❌ 已拒绝</div>
        {authorized?.operator && <div className="mt-1 text-xs text-orange-700">拒绝人: {authorized.operator}</div>}
      </div>

      <div className="mt-3 space-y-1 text-sm">
        <Row label="待处理" value="需要操作员介入处理" />
      </div>

      <Divider />
      <p className="mb-2 text-xs font-medium text-gray-500">后续处理工具</p>
      <div className="flex flex-wrap gap-2">
        <ActionBtn label="🔄 重新推送" variant="primary" loading={actionLoading} onClick={() => onAction('pending', 'repush')} />
        <ActionBtn label="👤 转人工处理" variant="warning" loading={actionLoading} onClick={() => onAction('in_progress', 'escalate')} />
        <ActionBtn label="✕ 关闭工单" variant="secondary" loading={actionLoading} onClick={() => onAction('cancelled', 'close')} />
      </div>
    </StatusCard>
  );
}

/* ── cancelled ─────────────────────────────────────────────────── */

function CancelledPanel({ ticket }: { ticket: TicketDetail }) {
  return (
    <StatusCard>
      <StatusHeader icon="⚪" title="已取消" />
      <p className="text-sm text-gray-600">工单已被取消</p>
      <div className="mt-3 space-y-1 text-sm">
        <Row label="取消时间" value={fmtTime(ticket.updated_at)} />
      </div>
    </StatusCard>
  );
}

/* ────────────────────────────────────────────────────────────────── */
/* Reusable components                                                */
/* ────────────────────────────────────────────────────────────────── */

function InfoCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6">
      <h3 className="mb-4 text-base font-semibold text-gray-800">{title}</h3>
      {children}
    </div>
  );
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <dt className="text-gray-500">{label}</dt>
      <dd className="mt-0.5">{value}</dd>
    </div>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex gap-2">
      <span className="w-20 shrink-0 text-gray-500">{label}</span>
      <span>{value}</span>
    </div>
  );
}

function StatusCard({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6">
      <h3 className="mb-3 text-base font-semibold text-gray-800">当前状态</h3>
      {children}
    </div>
  );
}

function StatusHeader({ icon, title }: { icon: string; title: string }) {
  return (
    <div className="mb-2 flex items-center gap-2 text-lg font-bold text-gray-900">
      <span>{icon}</span>
      <span>{title}</span>
    </div>
  );
}

function Divider() {
  return <hr className="my-4 border-gray-100" />;
}

function ActionBtn({ label, variant, loading, onClick }: { label: string; variant: 'primary' | 'secondary' | 'danger' | 'warning'; loading: boolean; onClick: () => void }) {
  const base = 'rounded-md px-4 py-1.5 text-sm font-medium transition-colors disabled:opacity-50';
  const styles: Record<string, string> = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700',
    secondary: 'border border-gray-300 text-gray-600 hover:bg-gray-100',
    danger: 'bg-red-600 text-white hover:bg-red-700',
    warning: 'bg-amber-500 text-white hover:bg-amber-600',
  };
  return (
    <button onClick={onClick} disabled={loading} className={`${base} ${styles[variant]}`}>
      {loading ? '处理中...' : label}
    </button>
  );
}
