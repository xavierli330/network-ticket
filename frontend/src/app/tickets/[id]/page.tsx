'use client';

import { useState, useEffect, use } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import type { Ticket, WorkflowState, AlertRecord } from '@/types';
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
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    async function load() {
      try {
        const data = await api.get<TicketDetail>(`/tickets/${id}`);
        setTicket(data);
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
                <dt className="text-gray-500">严重级别</dt>
                <dd className="mt-0.5"><SeverityBadge severity={ticket.severity} /></dd>
              </div>
              <div>
                <dt className="text-gray-500">工单类型</dt>
                <dd className="mt-0.5">
                  {ticket.ticket_type ? (
                    <span className="rounded px-2 py-0.5 text-xs text-white" style={{ backgroundColor: ticket.ticket_type.color }}>
                      {ticket.ticket_type.name}
                    </span>
                  ) : (
                    '-'
                  )}
                </dd>
              </div>
              <div className="col-span-2">
                <dt className="text-gray-500">描述</dt>
                <dd className="mt-0.5 whitespace-pre-wrap">{ticket.description || '-'}</dd>
              </div>
              <div>
                <dt className="text-gray-500">告警源类型</dt>
                <dd className="mt-0.5">{ticket.source_type}</dd>
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
