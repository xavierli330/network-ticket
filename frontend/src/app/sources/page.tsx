'use client';

import { useState, useEffect, useCallback } from 'react';
import { api } from '@/lib/api';
import type { AlertSource, PaginatedResponse } from '@/types';
import Sidebar from '@/components/layout/sidebar';
import Header from '@/components/layout/header';

interface SourceForm {
  name: string;
  type: string;
  poll_endpoint: string;
  poll_interval: number;
  parser_config: string;
}

const EMPTY_FORM: SourceForm = {
  name: '',
  type: 'zabbix',
  poll_endpoint: '',
  poll_interval: 60,
  parser_config: '{}',
};

const TYPE_OPTIONS = [
  { value: 'zabbix', label: 'Zabbix' },
  { value: 'prometheus', label: 'Prometheus' },
  { value: 'generic', label: '通用' },
];

export default function SourcesPage() {
  const [sources, setSources] = useState<AlertSource[]>([]);
  const [loading, setLoading] = useState(true);
  const [showDialog, setShowDialog] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<SourceForm>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null);

  const fetchSources = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api.get<PaginatedResponse<AlertSource>>('/alert-sources');
      setSources(data.items);
    } catch {
      // error handled by api client
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSources();
  }, [fetchSources]);

  function openCreate() {
    setEditingId(null);
    setForm(EMPTY_FORM);
    setShowDialog(true);
  }

  function openEdit(source: AlertSource) {
    setEditingId(source.id);
    setForm({
      name: source.name,
      type: source.type,
      poll_endpoint: source.poll_endpoint || '',
      poll_interval: source.poll_interval,
      parser_config: '{}',
    });
    setShowDialog(true);
  }

  async function handleSave() {
    setSaving(true);
    try {
      const payload = {
        ...form,
        parser_config: JSON.parse(form.parser_config),
      };
      if (editingId) {
        await api.put(`/alert-sources/${editingId}`, payload);
      } else {
        await api.post('/alert-sources', payload);
      }
      setShowDialog(false);
      fetchSources();
    } catch {
      // error handled by api client
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: number) {
    try {
      await api.delete(`/alert-sources/${id}`);
      setDeleteConfirm(null);
      fetchSources();
    } catch {
      // error handled by api client
    }
  }

  function updateForm(field: keyof SourceForm, value: string | number) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  return (
    <div className="flex h-screen">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-xl font-bold text-gray-800">告警源管理</h2>
            <button
              onClick={openCreate}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
            >
              创建告警源
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
                    <th className="px-4 py-3 text-left font-medium text-gray-600">名称</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">类型</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">轮询地址</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">轮询间隔(秒)</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">状态</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">创建时间</th>
                    <th className="px-4 py-3 text-left font-medium text-gray-600">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {sources.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="px-4 py-8 text-center text-gray-500">
                        暂无告警源数据
                      </td>
                    </tr>
                  ) : (
                    sources.map((source) => (
                      <tr key={source.id} className="border-b border-gray-100">
                        <td className="px-4 py-3 text-gray-500">{source.id}</td>
                        <td className="px-4 py-3 font-medium">{source.name}</td>
                        <td className="px-4 py-3">{source.type}</td>
                        <td className="max-w-xs truncate px-4 py-3 text-gray-500">
                          {source.poll_endpoint || '-'}
                        </td>
                        <td className="px-4 py-3 text-gray-500">{source.poll_interval}</td>
                        <td className="px-4 py-3">
                          <span className={`inline-block rounded border px-2 py-0.5 text-xs font-medium ${
                            source.status === 'active'
                              ? 'border-green-300 bg-green-100 text-green-800'
                              : 'border-gray-300 bg-gray-100 text-gray-800'
                          }`}>
                            {source.status === 'active' ? '活跃' : source.status}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-gray-500">
                          {new Date(source.created_at).toLocaleString('zh-CN')}
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex gap-2">
                            <button
                              onClick={() => openEdit(source)}
                              className="text-blue-600 hover:text-blue-800"
                            >
                              编辑
                            </button>
                            <button
                              onClick={() => setDeleteConfirm(source.id)}
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
                  {editingId ? '编辑告警源' : '创建告警源'}
                </h3>
                <div className="space-y-3">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">名称</label>
                    <input
                      value={form.name}
                      onChange={(e) => updateForm('name', e.target.value)}
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">类型</label>
                    <select
                      value={form.type}
                      onChange={(e) => updateForm('type', e.target.value)}
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none"
                    >
                      {TYPE_OPTIONS.map((opt) => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">轮询地址</label>
                    <input
                      value={form.poll_endpoint}
                      onChange={(e) => updateForm('poll_endpoint', e.target.value)}
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">轮询间隔(秒)</label>
                    <input
                      type="number"
                      value={form.poll_interval}
                      onChange={(e) => updateForm('poll_interval', Number(e.target.value))}
                      min={1}
                      className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">解析器配置 (JSON)</label>
                    <textarea
                      value={form.parser_config}
                      onChange={(e) => updateForm('parser_config', e.target.value)}
                      rows={4}
                      className="w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                    />
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
                    disabled={saving}
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
                <p className="mb-4 text-sm text-gray-600">确定要删除该告警源吗？此操作不可恢复。</p>
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
