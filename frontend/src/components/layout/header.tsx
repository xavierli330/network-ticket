'use client';

import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';

export default function Header() {
  const router = useRouter();

  function getUsername(): string {
    if (typeof window === 'undefined') return '';
    const token = api.getToken();
    if (!token) return '';
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      return payload.username || payload.sub || '';
    } catch {
      return '';
    }
  }

  function handleLogout() {
    api.clearToken();
    router.push('/login');
  }

  const username = getUsername();

  return (
    <header className="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-3">
      <div />
      <div className="flex items-center gap-4">
        {username && (
          <span className="text-sm text-gray-600">
            当前用户: <span className="font-medium text-gray-800">{username}</span>
          </span>
        )}
        <button
          onClick={handleLogout}
          className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-100"
        >
          登出
        </button>
      </div>
    </header>
  );
}
