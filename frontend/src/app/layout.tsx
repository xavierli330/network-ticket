import type { Metadata } from 'next';
import './globals.css';

export const metadata: Metadata = {
  title: '网络工单系统',
  description: '网络告警工单自动化处理平台',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-CN">
      <body className="min-h-screen bg-gray-50 text-gray-900 antialiased">
        {children}
      </body>
    </html>
  );
}
