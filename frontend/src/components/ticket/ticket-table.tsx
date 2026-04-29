'use client';

import { useRouter } from 'next/navigation';
import type { Ticket } from '@/types';
import TicketStatusBadge from './ticket-status-badge';
import SeverityBadge from './severity-badge';

interface TicketTableProps {
  tickets: Ticket[];
  ticketTypes?: { id: number; name: string; color: string }[];
}

export default function TicketTable({ tickets, ticketTypes }: TicketTableProps) {
  const router = useRouter();

  if (!tickets || tickets.length === 0) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-8 text-center text-gray-500">
        暂无工单数据
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-200 bg-gray-50">
            <th className="px-4 py-3 text-left font-medium text-gray-600">工单编号</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">工单类型</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">标题</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">严重级别</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">状态</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">告警源</th>
            <th className="px-4 py-3 text-left font-medium text-gray-600">创建时间</th>
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => (
            <tr
              key={ticket.id}
              onClick={() => router.push(`/tickets/${ticket.id}`)}
              className="cursor-pointer border-b border-gray-100 transition-colors hover:bg-blue-50"
            >
              <td className="px-4 py-3 font-mono text-blue-600">{ticket.ticket_no}</td>
              <td className="px-4 py-3">
                {(() => {
                  let name: string | undefined;
                  let color: string | undefined;
                  if (ticket.ticket_type) {
                    name = ticket.ticket_type.name;
                    color = ticket.ticket_type.color;
                  } else if (ticket.ticket_type_id && ticketTypes) {
                    const tt = ticketTypes.find((t) => t.id === ticket.ticket_type_id);
                    if (tt) {
                      name = tt.name;
                      color = tt.color;
                    }
                  }
                  if (name && color) {
                    return (
                      <span className="rounded px-2 py-0.5 text-xs text-white" style={{ backgroundColor: color }}>
                        {name}
                      </span>
                    );
                  }
                  return '-';
                })()}
              </td>
              <td className="max-w-xs truncate px-4 py-3">{ticket.title}</td>
              <td className="px-4 py-3">
                <SeverityBadge severity={ticket.severity} />
              </td>
              <td className="px-4 py-3">
                <TicketStatusBadge status={ticket.status} />
              </td>
              <td className="px-4 py-3 text-gray-500">{ticket.source_type}</td>
              <td className="px-4 py-3 text-gray-500">
                {new Date(ticket.created_at).toLocaleString('zh-CN')}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
