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
