export enum EventType {
  CREATED = 'created',
  MODIFIED = 'modified',
  DELETED = 'deleted',
  RENAMED = 'renamed'
}

export interface FileEvent {
  id: number;
  watcher_id: number;
  watcher_name?: string;
  event_type: EventType;
  file_path: string;
  file_name: string;
  file_size: number;
  old_path?: string;
  timestamp: string;
  webhook_status?: number;
  webhook_error?: string;
  retry_count?: number;
  created_at?: string;
}

export interface GroupedEvents {
  path: string;
  watcher_id: number;
  watcher_name: string;
  events: FileEvent[];
}
