export interface WebhookLog {
  id: number;
  event_id: number;
  watcher_id: number;
  status_code: number;
  success: boolean;
  error?: string;
  retry_count: number;
  response_time: number;
  request_payload?: string;
  response_body?: string;
  created_at: string;
}
