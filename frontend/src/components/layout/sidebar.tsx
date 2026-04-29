'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useEffect, useState } from 'react';

interface StoredUser {
  id: number;
  username: string;
  role: string;
}

const ALL_NAV_ITEMS = [
  { label: '工单管理', href: '/tickets' },
  { label: '告警源管理', href: '/sources' },
  { label: '审计日志', href: '/audit-logs' },
  { label: '客户管理', href: '/clients', adminOnly: true },
  { label: '用户管理', href: '/users', adminOnly: true },
];

export default function Sidebar() {
  const pathname = usePathname();
  const [role, setRole] = useState<string | null>(null);

  useEffect(() => {
    try {
      const raw = localStorage.getItem('user');
      if (raw) {
        const user: StoredUser = JSON.parse(raw);
        setRole(user.role);
      }
    } catch {
      // ignore parse errors
    }
  }, []);

  const navItems = ALL_NAV_ITEMS.filter((item) => !item.adminOnly || role === 'admin');

  return (
    <aside className="flex h-screen w-56 flex-col border-r border-gray-200 bg-white">
      <div className="border-b border-gray-200 px-5 py-4">
        <h1 className="text-lg font-bold text-gray-800">网络工单平台</h1>
      </div>
      <nav className="flex-1 space-y-1 px-3 py-4">
        {navItems.map((item) => {
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
