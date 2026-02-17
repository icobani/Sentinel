export enum WatchMode {
  WATCH = 'watch',
  POLL = 'poll'
}

export enum WatcherStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  ERROR = 'error'
}

export interface WatcherFilters {
  include?: string[];
  exclude?: string[];
}

export interface WebhookRetry {
  max_attempts: number;
  backoff: string;
}

export interface WebhookConfig {
  url: string;
  headers?: Record<string, string>;
  timeout: string;
  retry?: WebhookRetry;
  attach_file?: boolean;
}

export interface Watcher {
  id: number;
  name: string;
  path: string;
  mode: WatchMode;
  recursive: boolean;
  poll_interval?: string;
  filters?: WatcherFilters;
  webhook?: WebhookConfig;
  status?: WatcherStatus;
  last_event_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface WatcherStats {
  total_events: number;
  webhook_success: number;
  webhook_failed: number;
  success_rate: number;
}
