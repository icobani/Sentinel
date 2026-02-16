export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

export interface PaginatedData<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface PaginatedResponse<T> extends ApiResponse<PaginatedData<T>> {}

export interface SystemStatus {
  status: string;
  version: string;
  uptime: string;
  uptime_seconds: number;
  start_time: string;
  go_version: string;
  num_goroutines: number;
  database_status: string;
  config_loaded: boolean;
  watcher_count: number;
  watchers_total: number;
  watchers_active: number;
  events_today: number;
  webhook_success_rate: number;
  pending_restart: boolean;
}
