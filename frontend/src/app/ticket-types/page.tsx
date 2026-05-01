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
                          <div className="flex items-center gap-2">
                            <span className="inline-block h-4 w-4 rounded border border-gray-200" style={{ backgroundColor: tt.color }} />
                            <span className="font-mono text-xs text-gray-500">{tt.color}</span>
                          </div>
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
