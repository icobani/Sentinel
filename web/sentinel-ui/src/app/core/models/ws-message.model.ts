export enum WsMessageType {
  FILE_EVENT = 'file_event',
  WEBHOOK_RESULT = 'webhook_result',
  WATCHER_STATUS = 'watcher_status'
}

export interface WsMessageData {
  watcher_id: number;
  watcher_name: string;
  event?: string;
  file_name?: string;
  file_path?: string;
  file_size?: number;
  timestamp?: string;
  webhook_status?: number;
  webhook_error?: string;
  retry_count?: number;
  max_retries?: number;
  status?: string;
}

export interface WsMessage {
  type: WsMessageType;
  data: WsMessageData;
}
