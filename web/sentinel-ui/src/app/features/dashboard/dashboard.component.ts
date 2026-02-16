import { Component, OnInit, signal, DestroyRef, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { interval } from 'rxjs';
import { SummaryCardsComponent } from './summary-cards/summary-cards.component';
import { GroupedEventsComponent } from './grouped-events/grouped-events.component';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { TranslatePipe } from '../../shared/pipes/translate.pipe';
import { StatusService } from '../../core/services/status.service';
import { EventLogService } from '../../core/services/event-log.service';
import { WebSocketService } from '../../core/services/websocket.service';
import { SystemStatus } from '../../core/models/api-response.model';
import { FileEvent, EventType } from '../../core/models/event.model';
import { WsMessage, WsMessageType } from '../../core/models/ws-message.model';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    MatIconModule,
    SummaryCardsComponent,
    GroupedEventsComponent,
    LoadingSkeletonComponent,
    TranslatePipe
  ],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.scss'
})
export class DashboardComponent implements OnInit {
  private destroyRef = inject(DestroyRef);
  private statusService = inject(StatusService);
  private eventLogService = inject(EventLogService);
  private wsService = inject(WebSocketService);

  systemStatus = signal<SystemStatus | null>(null);
  recentEvents = signal<FileEvent[]>([]);
  isLoading = signal(true);
  wsConnectionError = signal(false);

  ngOnInit(): void {
    this.loadSystemStatus();
    this.loadRecentEvents();
    this.setupWebSocket();
    this.setupAutoRefresh();
  }

  private loadSystemStatus(): void {
    this.statusService.getSystemStatus().subscribe({
      next: (response) => {
        if (response.success && response.data) {
          this.systemStatus.set(response.data);
        }
      },
      error: () => {
        // Silently fail - not critical
      }
    });
  }

  private loadRecentEvents(): void {
    this.eventLogService.getLogs({ page: 1, limit: 20 }).subscribe({
      next: (response) => {
        if (response.success && response.data) {
          this.recentEvents.set(response.data.items);
        }
        this.isLoading.set(false);
      },
      error: () => {
        // Show error to user but don't fail completely
        this.isLoading.set(false);
      }
    });
  }

  private setupWebSocket(): void {
    this.wsService.connectionStatus.pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: (isConnected) => {
        this.wsConnectionError.set(!isConnected);
      }
    });

    this.wsService.messages.pipe(takeUntilDestroyed(this.destroyRef)).subscribe({
      next: (message: WsMessage) => {
        this.handleWebSocketMessage(message);
      }
    });
  }

  private handleWebSocketMessage(message: WsMessage): void {
    switch (message.type) {
      case WsMessageType.FILE_EVENT:
        this.handleFileEvent(message);
        break;
      case WsMessageType.WEBHOOK_RESULT:
        this.handleWebhookResult(message);
        break;
      case WsMessageType.WATCHER_STATUS:
        // Refresh system status when watcher status changes
        this.loadSystemStatus();
        break;
    }
  }

  private handleFileEvent(message: WsMessage): void {
    const { data } = message;

    // Create new event object
    const newEvent: FileEvent = {
      id: Date.now(), // Temporary ID for animation
      watcher_id: data.watcher_id,
      watcher_name: data.watcher_name,
      event_type: data.event as EventType,
      file_path: data.file_path || '',
      file_name: data.file_name || '',
      file_size: data.file_size || 0,
      timestamp: data.timestamp || new Date().toISOString(),
      webhook_status: data.webhook_status,
      webhook_error: data.webhook_error,
      retry_count: data.retry_count
    };

    // Add to the beginning of the list (with animation)
    this.recentEvents.update(events => [newEvent, ...events.slice(0, 19)]);

    // Refresh system status to update event count
    this.loadSystemStatus();
  }

  private handleWebhookResult(message: WsMessage): void {
    const { data } = message;

    // Update the webhook status of the corresponding event
    this.recentEvents.update(events =>
      events.map(event => {
        if (event.watcher_id === data.watcher_id &&
            event.file_name === data.file_name &&
            event.timestamp === data.timestamp) {
          return {
            ...event,
            webhook_status: data.webhook_status,
            webhook_error: data.webhook_error,
            retry_count: data.retry_count
          };
        }
        return event;
      })
    );
  }

  private setupAutoRefresh(): void {
    // Refresh system status every 30 seconds
    interval(30000)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => {
        this.loadSystemStatus();
      });
  }
}
