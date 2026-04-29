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
              placeholder="操作人"
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
            />
            <button
              onClick={() => { setOperator(''); setPage(1); }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm text-gray-600 transition-colors hover:bg-gray-100"
            >
              重置
            </button>
          </div>

          {/* Table */}
          {loading ? (
            <div className="py-12 text-center text-gray-500">加载中...</div>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-gray-200">
              <table className="min-w-full text-sm">
                <thead className="bg-gray-50 text-left text-xs font-semibold uppercase text-gray-600">
                  <tr>
                    <th className="px-4 py-3">时间</th>
                    <th className="px-4 py-3">操作</th>
                    <th className="px-4 py-3">资源类型</th>
                    <th className="px-4 py-3">资源ID</th>
                    <th className="px-4 py-3">操作人</th>
                    <th className="px-4 py-3">详情</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 text-gray-700">
                  {logs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50">
                      <td className="whitespace-nowrap px-4 py-3">
                        {new Date(log.created_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3">{log.action}</td>
                      <td className="px-4 py-3">{log.resource_type}</td>
                      <td className="px-4 py-3">{log.resource_id ?? '-'}</td>
                      <td className="px-4 py-3">{log.actor}</td>
                      <td className="max-w-xs truncate px-4 py-3" title={JSON.stringify(log.detail)}>
                        {JSON.stringify(log.detail)}
                      </td>
                    </tr>
                  ))}
                  {logs.length === 0 && (
                    <tr>
                      <td colSpan={6} className="py-12 text-center text-gray-400">
                        暂无数据
                      </td>
                    </tr>
                  )}
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
