export interface Ticket {
  id: number;
  ticket_no: string;
  source_type: string;
  title: string;
  description: string;
  severity: string;
  status: string;
  client_id?: number;
  fingerprint?: string;
  created_at: string;
  updated_at: string;
}

export interface WorkflowState {
  id: number;
  ticket_id: number;
  node_name: string;
  status: string;
  operator?: string;
  input_data?: unknown;
  output_data?: unknown;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface AlertRecord {
  id: number;
  ticket_id: number;
  alert_raw: unknown;
  alert_parsed: unknown;
  received_at: string;
}

export interface Client {
  id: number;
  name: string;
  api_endpoint: string;
  callback_url?: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface AlertSource {
  id: number;
  name: string;
  type: string;
  poll_endpoint?: string;
  poll_interval: number;
  dedup_window_sec: number;
  status: string;
  created_at: string;
}

export interface User {
  id: number;
  username: string;
  role: 'admin' | 'operator';
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}
