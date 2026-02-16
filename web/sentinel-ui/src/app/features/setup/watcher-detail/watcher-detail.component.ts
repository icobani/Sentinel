import { Component, OnInit, inject, signal, computed, ElementRef, ViewChild, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatNativeDateModule } from '@angular/material/core';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { debounceTime, finalize } from 'rxjs';

import { WatcherService } from '../../../core/services/watcher.service';
import { EventLogService, LogQueryParams } from '../../../core/services/event-log.service';
import { NotificationService } from '../../../core/services/notification.service';
import { WebSocketService } from '../../../core/services/websocket.service';
import { Watcher, WatchMode, WatcherStatus } from '../../../core/models/watcher.model';
import { FileEvent, EventType } from '../../../core/models/event.model';
import { WsMessage, WsMessageType } from '../../../core/models/ws-message.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge/status-badge.component';
import { LoadingSkeletonComponent } from '../../../shared/components/loading-skeleton/loading-skeleton.component';
import { EmptyStateComponent } from '../../../shared/components/empty-state/empty-state.component';
import { WebhookStatusIconComponent } from '../../../shared/components/webhook-status-icon/webhook-status-icon.component';
import { FormatDatePipe } from '../../../shared/pipes/format-date.pipe';
import { FileSizePipe } from '../../../shared/pipes/file-size.pipe';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';

@Component({
  selector: 'app-watcher-detail',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatDatepickerModule,
    MatNativeDateModule,
    MatProgressSpinnerModule,
    StatusBadgeComponent,
    LoadingSkeletonComponent,
    EmptyStateComponent,
    WebhookStatusIconComponent,
    FormatDatePipe,
    FileSizePipe,
    TranslatePipe
  ],
  templateUrl: './watcher-detail.component.html',
  styleUrls: ['./watcher-detail.component.scss']
})
export class WatcherDetailComponent implements OnInit {
  @ViewChild('logsContainer') logsContainer!: ElementRef;

  private readonly destroyRef = inject(DestroyRef);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly watcherService = inject(WatcherService);
  private readonly eventLogService = inject(EventLogService);
  private readonly notificationService = inject(NotificationService);
  private readonly wsService = inject(WebSocketService);

  readonly watcher = signal<Watcher | null>(null);
  readonly events = signal<FileEvent[]>([]);
  readonly loading = signal(true);
  readonly loadingMore = signal(false);
  readonly hasMore = signal(true);
  readonly expandedEventId = signal<number | null>(null);

  readonly searchControl = new FormControl('');
  readonly eventTypeControl = new FormControl<EventType | ''>('');
  readonly fromDateControl = new FormControl<Date | null>(null);
  readonly toDateControl = new FormControl<Date | null>(null);

  private currentPage = 1;
  private readonly pageSize = 20;
  private watcherId = 0;

  readonly WatchMode = WatchMode;
  readonly WatcherStatus = WatcherStatus;
  readonly EventType = EventType;
  readonly eventTypes = Object.values(EventType);

  ngOnInit(): void {
    this.route.params
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(params => {
        this.watcherId = +params['id'];
        this.loadWatcher();
        this.loadEvents(true);
      });

    // Debounce search input
    this.searchControl.valueChanges
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        debounceTime(300)
      )
      .subscribe(() => {
        this.loadEvents(true);
      });

    // Filter changes
    this.eventTypeControl.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.loadEvents(true));

    this.fromDateControl.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.loadEvents(true));

    this.toDateControl.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.loadEvents(true));

    // Setup WebSocket real-time updates
    this.setupWebSocket();
  }

  private setupWebSocket(): void {
    this.wsService.messages
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (message: WsMessage) => {
          if (message.type === WsMessageType.FILE_EVENT && message.data.watcher_id === this.watcherId) {
            // Reload events and watcher info when file event occurs for this watcher
            this.loadEvents(true);
            this.loadWatcher();
          } else if (message.type === WsMessageType.WATCHER_STATUS && message.data.watcher_id === this.watcherId) {
            // Reload watcher info when status changes
            this.loadWatcher();
          }
        }
      });
  }

  loadWatcher(): void {
    this.watcherService.getWatcher(this.watcherId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            this.watcher.set(response.data);
          }
        },
        error: () => {
          this.notificationService.showError('Failed to load watcher details');
        }
      });
  }

  loadEvents(reset: boolean = false): void {
    if (reset) {
      this.currentPage = 1;
      this.events.set([]);
      this.hasMore.set(true);
      this.loading.set(true);
    } else {
      this.loadingMore.set(true);
    }

    const params: LogQueryParams = {
      page: this.currentPage,
      limit: this.pageSize
    };

    if (this.searchControl.value) {
      params.search = this.searchControl.value;
    }

    if (this.eventTypeControl.value) {
      params.event_type = this.eventTypeControl.value;
    }

    if (this.fromDateControl.value) {
      params.from = this.fromDateControl.value.toISOString();
    }

    if (this.toDateControl.value) {
      params.to = this.toDateControl.value.toISOString();
    }

    this.eventLogService.getWatcherLogs(this.watcherId, params)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => {
          this.loading.set(false);
          this.loadingMore.set(false);
        })
      )
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            const newEvents = response.data.items;

            if (reset) {
              this.events.set(newEvents);
            } else {
              this.events.update(current => [...current, ...newEvents]);
            }

            this.hasMore.set(this.currentPage < response.data.total_pages);
          }
        },
        error: () => {
          this.notificationService.showError('Failed to load event logs');
        }
      });
  }

  onScroll(event: Event): void {
    const element = event.target as HTMLElement;
    const threshold = 100;
    const position = element.scrollTop + element.clientHeight;
    const height = element.scrollHeight;

    if (position > height - threshold && !this.loadingMore() && this.hasMore()) {
      this.currentPage++;
      this.loadEvents(false);
    }
  }

  onLoadMore(): void {
    if (!this.loadingMore() && this.hasMore()) {
      this.currentPage++;
      this.loadEvents(false);
    }
  }

  toggleEventDetails(eventId: number): void {
    this.expandedEventId.update(current =>
      current === eventId ? null : eventId
    );
  }

  isEventExpanded(eventId: number): boolean {
    return this.expandedEventId() === eventId;
  }

  onEditWatcher(): void {
    this.router.navigate(['/setup', this.watcherId, 'edit']);
  }

  onBackToList(): void {
    this.router.navigate(['/setup']);
  }

  getEventTypeIcon(type: EventType): string {
    switch (type) {
      case EventType.CREATED:
        return 'add_circle';
      case EventType.MODIFIED:
        return 'edit';
      case EventType.DELETED:
        return 'delete';
      case EventType.RENAMED:
        return 'drive_file_rename_outline';
      default:
        return 'description';
    }
  }

  getEventTypeColor(type: EventType): string {
    switch (type) {
      case EventType.CREATED:
        return 'success';
      case EventType.MODIFIED:
        return 'primary';
      case EventType.DELETED:
        return 'warn';
      case EventType.RENAMED:
        return 'accent';
      default:
        return '';
    }
  }
}
