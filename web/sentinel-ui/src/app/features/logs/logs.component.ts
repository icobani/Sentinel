import { Component, OnInit, inject, signal, computed, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormControl, ReactiveFormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatTableModule } from '@angular/material/table';
import { MatPaginatorModule, PageEvent } from '@angular/material/paginator';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatDatepickerModule } from '@angular/material/datepicker';
import { MatNativeDateModule } from '@angular/material/core';
import { MatDialog, MatDialogModule } from '@angular/material/dialog';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { debounceTime, distinctUntilChanged } from 'rxjs/operators';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EventLogService, LogQueryParams } from '../../core/services/event-log.service';
import { WatcherService } from '../../core/services/watcher.service';
import { NotificationService } from '../../core/services/notification.service';
import { WebSocketService } from '../../core/services/websocket.service';
import { FileEvent, EventType } from '../../core/models/event.model';
import { Watcher } from '../../core/models/watcher.model';
import { WsMessage, WsMessageType } from '../../core/models/ws-message.model';
import { FileSizePipe } from '../../shared/pipes/file-size.pipe';
import { RelativeTimePipe } from '../../shared/pipes/relative-time.pipe';
import { FormatDatePipe } from '../../shared/pipes/format-date.pipe';
import { TranslatePipe } from '../../shared/pipes/translate.pipe';
import { LoadingSkeletonComponent } from '../../shared/components/loading-skeleton/loading-skeleton.component';
import { EmptyStateComponent } from '../../shared/components/empty-state/empty-state.component';
import { LogDetailDialogComponent } from './log-detail-dialog/log-detail-dialog.component';
import { ConfirmDialogComponent } from '../../shared/components/confirm-dialog/confirm-dialog.component';

@Component({
  selector: 'app-logs',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatTableModule,
    MatPaginatorModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatDatepickerModule,
    MatNativeDateModule,
    MatDialogModule,
    MatTooltipModule,
    MatCheckboxModule,
    FileSizePipe,
    RelativeTimePipe,
    FormatDatePipe,
    TranslatePipe,
    LoadingSkeletonComponent,
    EmptyStateComponent
  ],
  template: `
    <div class="logs-container">
      <div class="header">
        <h1>{{ 'logs.title' | translate }}</h1>
        <div class="header-actions">
          <button
            mat-raised-button
            color="warn"
            (click)="deleteSelected()"
            *ngIf="hasSelection()"
          >
            <mat-icon>delete</mat-icon>
            {{ 'common.delete' | translate }} ({{ selection().size }})
          </button>
          <button
            mat-raised-button
            color="primary"
            (click)="exportLogs()"
            [disabled]="!hasEvents() || isExporting()"
          >
            <mat-icon>download</mat-icon>
            {{ 'logs.export' | translate }}
          </button>
        </div>
      </div>

      <mat-card class="filters-card">
        <mat-card-content>
          <div class="filters-grid">
            <!-- Watcher filter -->
            <mat-form-field appearance="outline">
              <mat-label>{{ 'logs.filter_by_watcher' | translate }}</mat-label>
              <mat-select [formControl]="watcherFilter" multiple>
                <mat-option [value]="null">{{ 'form.all' | translate }}</mat-option>
                <mat-option *ngFor="let watcher of watchers()" [value]="watcher.id">
                  {{ watcher.name }}
                </mat-option>
              </mat-select>
            </mat-form-field>

            <!-- Event type filter (chip toggles) -->
            <div class="event-type-filter">
              <label class="filter-label">{{ 'logs.filter_by_event_type' | translate }}</label>
              <mat-chip-listbox class="event-chip-list" aria-label="Event type selection" multiple>
                <mat-chip-option
                  *ngFor="let type of eventTypes"
                  [selected]="isEventTypeSelected(type.value)"
                  (click)="toggleEventType(type.value)"
                  [class]="'event-chip-option-' + type.color"
                >
                  {{ type.label | translate }}
                </mat-chip-option>
              </mat-chip-listbox>
            </div>

            <!-- Date from -->
            <mat-form-field appearance="outline">
              <mat-label>{{ 'form.from' | translate }}</mat-label>
              <input matInput [matDatepicker]="fromPicker" [formControl]="dateFromFilter">
              <mat-datepicker-toggle matIconSuffix [for]="fromPicker"></mat-datepicker-toggle>
              <mat-datepicker #fromPicker></mat-datepicker>
            </mat-form-field>

            <!-- Date to -->
            <mat-form-field appearance="outline">
              <mat-label>{{ 'form.to' | translate }}</mat-label>
              <input matInput [matDatepicker]="toPicker" [formControl]="dateToFilter">
              <mat-datepicker-toggle matIconSuffix [for]="toPicker"></mat-datepicker-toggle>
              <mat-datepicker #toPicker></mat-datepicker>
            </mat-form-field>

            <!-- File name search -->
            <mat-form-field appearance="outline" class="full-width">
              <mat-label>{{ 'form.search_by_filename' | translate }}</mat-label>
              <input matInput [formControl]="searchFilter" />
              <mat-icon matPrefix>search</mat-icon>
            </mat-form-field>
          </div>
        </mat-card-content>
      </mat-card>

      <!-- Loading skeleton -->
      <mat-card *ngIf="isLoading()" class="table-card">
        <mat-card-content>
          <app-loading-skeleton [count]="10" [height]="56"></app-loading-skeleton>
        </mat-card-content>
      </mat-card>

      <!-- Empty state -->
      <mat-card *ngIf="!isLoading() && !hasEvents()" class="table-card">
        <mat-card-content>
          <app-empty-state
            icon="📋"
            title="empty.no_logs"
            message="empty.no_events_message"
          ></app-empty-state>
        </mat-card-content>
      </mat-card>

      <!-- Data table -->
      <mat-card *ngIf="!isLoading() && hasEvents()" class="table-card">
        <mat-card-content>
          <div class="table-wrapper">
            <table mat-table [dataSource]="events()" class="logs-table">
              <!-- Select column -->
              <ng-container matColumnDef="select">
                <th mat-header-cell *matHeaderCellDef>
                  <mat-checkbox
                    [checked]="isAllSelected()"
                    [indeterminate]="hasSelection() && !isAllSelected()"
                    (change)="toggleAllSelection()"
                  ></mat-checkbox>
                </th>
                <td mat-cell *matCellDef="let event" (click)="$event.stopPropagation()">
                  <mat-checkbox
                    [checked]="isSelected(event.id)"
                    (change)="toggleSelection(event.id)"
                  ></mat-checkbox>
                </td>
              </ng-container>

              <!-- Timestamp column -->
              <ng-container matColumnDef="timestamp">
                <th mat-header-cell *matHeaderCellDef>{{ 'events.timestamp' | translate }}</th>
                <td mat-cell *matCellDef="let event" [matTooltip]="event.timestamp | relativeTime">
                  {{ event.timestamp | formatDate }}
                </td>
              </ng-container>

              <!-- Watcher column -->
              <ng-container matColumnDef="watcher">
                <th mat-header-cell *matHeaderCellDef>{{ 'setup.watcher_name' | translate }}</th>
                <td mat-cell *matCellDef="let event">
                  {{ event.watcher_name }}
                </td>
              </ng-container>

              <!-- Event type column -->
              <ng-container matColumnDef="event_type">
                <th mat-header-cell *matHeaderCellDef>{{ 'logs.event_type' | translate }}</th>
                <td mat-cell *matCellDef="let event">
                  <mat-chip [class]="'event-chip-' + event.event_type">
                    {{ ('events.' + event.event_type) | translate }}
                  </mat-chip>
                </td>
              </ng-container>

              <!-- File name column -->
              <ng-container matColumnDef="file_name">
                <th mat-header-cell *matHeaderCellDef>{{ 'logs.columns.file' | translate }}</th>
                <td mat-cell *matCellDef="let event" class="file-name-cell">
                  {{ event.file_name }}
                </td>
              </ng-container>

              <!-- File path column -->
              <ng-container matColumnDef="file_path">
                <th mat-header-cell *matHeaderCellDef>{{ 'events.full_path' | translate }}</th>
                <td mat-cell *matCellDef="let event" class="file-path-cell" [matTooltip]="event.file_path">
                  {{ event.file_path }}
                </td>
              </ng-container>

              <!-- File size column -->
              <ng-container matColumnDef="file_size">
                <th mat-header-cell *matHeaderCellDef>{{ 'events.file_size' | translate }}</th>
                <td mat-cell *matCellDef="let event">
                  {{ event.file_size | fileSize }}
                </td>
              </ng-container>

              <!-- Webhook status column -->
              <ng-container matColumnDef="webhook_status">
                <th mat-header-cell *matHeaderCellDef>{{ 'events.webhook_status' | translate }}</th>
                <td mat-cell *matCellDef="let event">
                  <span *ngIf="event.webhook_status === 200" class="webhook-icon success">✅</span>
                  <span *ngIf="event.webhook_status && event.webhook_status !== 200"
                        class="webhook-icon error"
                        [matTooltip]="event.webhook_status + (event.webhook_error ? ': ' + event.webhook_error : '')">
                    ❌
                  </span>
                  <span *ngIf="!event.webhook_status && event.retry_count !== undefined && event.retry_count > 0"
                        class="webhook-icon warning"
                        [matTooltip]="event.retry_count + ' retries'">
                    ⚠️
                  </span>
                  <span *ngIf="!event.webhook_status && (!event.retry_count || event.retry_count === 0)">—</span>
                </td>
              </ng-container>

              <!-- Actions column -->
              <ng-container matColumnDef="actions">
                <th mat-header-cell *matHeaderCellDef></th>
                <td mat-cell *matCellDef="let event">
                  <button mat-icon-button (click)="openDetail(event)" [matTooltip]="'common.details' | translate">
                    <mat-icon>info</mat-icon>
                  </button>
                </td>
              </ng-container>

              <tr mat-header-row *matHeaderRowDef="displayedColumns"></tr>
              <tr mat-row *matRowDef="let row; columns: displayedColumns;" class="clickable-row" (click)="openDetail(row)"></tr>
            </table>
          </div>

          <mat-paginator
            [length]="totalCount()"
            [pageSize]="pageSize()"
            [pageSizeOptions]="[10, 20, 50]"
            [pageIndex]="currentPage() - 1"
            (page)="onPageChange($event)"
            showFirstLastButtons
          ></mat-paginator>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: [`
    .logs-container {
      padding: 24px;
      max-width: 1400px;
      margin: 0 auto;
    }

    .header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 24px;
    }

    .header-actions {
      display: flex;
      gap: 8px;
      align-items: center;
    }

    h1 {
      margin: 0;
      font-size: 32px;
      font-weight: 500;
    }

    .filters-card {
      margin-bottom: 24px;
    }

    .filters-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
    }

    .full-width {
      grid-column: 1 / -1;
    }

    .event-type-filter {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .filter-label {
      font-size: 14px;
      font-weight: 500;
      color: rgba(0, 0, 0, 0.6);
    }

    .event-chip-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }

    mat-chip-option {
      font-weight: 500;
      transition: all 0.2s ease;
      cursor: pointer;

      &.event-chip-option-created {
        &.mat-mdc-chip-selected {
          background-color: #4caf50 !important;
          color: white !important;
        }
        &:not(.mat-mdc-chip-selected) {
          background-color: rgba(76, 175, 80, 0.12) !important;
          color: #4caf50 !important;
        }
      }

      &.event-chip-option-modified {
        &.mat-mdc-chip-selected {
          background-color: #2196f3 !important;
          color: white !important;
        }
        &:not(.mat-mdc-chip-selected) {
          background-color: rgba(33, 150, 243, 0.12) !important;
          color: #2196f3 !important;
        }
      }

      &.event-chip-option-deleted {
        &.mat-mdc-chip-selected {
          background-color: #f44336 !important;
          color: white !important;
        }
        &:not(.mat-mdc-chip-selected) {
          background-color: rgba(244, 67, 54, 0.12) !important;
          color: #f44336 !important;
        }
      }

      &.event-chip-option-renamed {
        &.mat-mdc-chip-selected {
          background-color: #ff9800 !important;
          color: white !important;
        }
        &:not(.mat-mdc-chip-selected) {
          background-color: rgba(255, 152, 0, 0.12) !important;
          color: #ff9800 !important;
        }
      }

      &:hover {
        transform: scale(1.05);
      }
    }

    .table-card {
      mat-card-content {
        padding: 0;
      }
    }

    .table-wrapper {
      overflow-x: auto;
    }

    .logs-table {
      width: 100%;
      font-size: 16px;

      th {
        font-weight: 600;
        font-size: 16px;
        background-color: rgba(0, 0, 0, 0.02);
        white-space: nowrap;
      }

      td, th {
        padding: 16px 12px;
      }

      // Define column widths for better proportions
      .mat-column-select {
        width: 48px;
        min-width: 48px;
        max-width: 48px;
        padding-right: 0 !important;
      }

      .mat-column-timestamp {
        width: 140px;
        min-width: 140px;
      }

      .mat-column-watcher {
        width: 180px;
        min-width: 180px;
      }

      .mat-column-event_type {
        width: 150px;
        min-width: 150px;
      }

      .mat-column-file_name {
        width: 250px;
        min-width: 250px;
      }

      .mat-column-file_path {
        width: auto;
        min-width: 300px;
      }

      .mat-column-file_size {
        width: 120px;
        min-width: 120px;
      }

      .mat-column-webhook_status {
        width: 100px;
        min-width: 100px;
        text-align: center;
      }

      .mat-column-actions {
        width: 60px;
        min-width: 60px;
        text-align: center;
      }
    }

    .clickable-row {
      cursor: pointer;
      transition: background-color 0.2s;

      &:hover {
        background-color: rgba(0, 0, 0, 0.04);
      }
    }

    .file-name-cell {
      font-weight: 500;
    }

    .file-path-cell {
      max-width: 400px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .webhook-icon {
      font-size: 20px;
      display: inline-block;
      cursor: help;

      &.success {
        color: #4caf50;
      }

      &.error {
        color: #f44336;
      }

      &.warning {
        color: #ff9800;
      }
    }

    mat-chip {
      font-weight: 500;
      font-size: 14px;
    }

    .event-chip-created {
      background-color: #4caf50 !important;
      color: white !important;
    }

    .event-chip-modified {
      background-color: #2196f3 !important;
      color: white !important;
    }

    .event-chip-deleted {
      background-color: #f44336 !important;
      color: white !important;
    }

    .event-chip-renamed {
      background-color: #ff9800 !important;
      color: white !important;
    }

    mat-paginator {
      border-top: 1px solid rgba(0, 0, 0, 0.12);
    }

    /* Dark theme support */
    :host-context(.dark-theme) {
      .logs-table th {
        background-color: rgba(255, 255, 255, 0.05);
      }

      .clickable-row:hover {
        background-color: rgba(255, 255, 255, 0.08);
      }

      mat-paginator {
        border-top-color: rgba(255, 255, 255, 0.12);
      }

      .filter-label {
        color: rgba(255, 255, 255, 0.6);
      }
    }

    /* Responsive */
    @media (max-width: 768px) {
      .logs-container {
        padding: 16px;
      }

      .header {
        flex-direction: column;
        align-items: flex-start;
        gap: 16px;
      }

      .filters-grid {
        grid-template-columns: 1fr;
      }

      .file-path-cell {
        max-width: 200px;
      }
    }
  `]
})
export class LogsComponent implements OnInit {
  private readonly destroyRef = inject(DestroyRef);
  private readonly eventLogService = inject(EventLogService);
  private readonly watcherService = inject(WatcherService);
  private readonly notificationService = inject(NotificationService);
  private readonly wsService = inject(WebSocketService);
  private readonly dialog = inject(MatDialog);

  // Signals
  readonly isLoading = signal(false);
  readonly isExporting = signal(false);
  readonly events = signal<FileEvent[]>([]);
  readonly watchers = signal<Watcher[]>([]);
  readonly totalCount = signal(0);
  readonly currentPage = signal(1);
  readonly pageSize = signal(20);
  readonly selection = signal<Set<number>>(new Set());

  // Form controls
  readonly watcherFilter = new FormControl<number[] | null>(null);
  readonly dateFromFilter = new FormControl<Date | null>(null);
  readonly dateToFilter = new FormControl<Date | null>(null);
  readonly searchFilter = new FormControl<string>('');

  // Event type filters as signals for chip toggles
  readonly selectedEventTypes = signal<Set<string>>(new Set());

  // Event types for chip toggles
  readonly eventTypes = [
    { value: EventType.CREATED, label: 'events.created', color: 'created' },
    { value: EventType.MODIFIED, label: 'events.modified', color: 'modified' },
    { value: EventType.DELETED, label: 'events.deleted', color: 'deleted' },
    { value: EventType.RENAMED, label: 'events.renamed', color: 'renamed' }
  ];

  // Table columns
  readonly displayedColumns = [
    'select',
    'timestamp',
    'watcher',
    'event_type',
    'file_name',
    'file_path',
    'file_size',
    'webhook_status',
    'actions'
  ];

  readonly hasEvents = computed(() => this.events().length > 0);
  readonly hasSelection = computed(() => this.selection().size > 0);
  readonly isAllSelected = computed(() => {
    const events = this.events();
    return events.length > 0 && this.selection().size === events.length;
  });

  constructor() {
    // Setup filter listeners with debounce
    this.searchFilter.valueChanges
      .pipe(
        debounceTime(300),
        distinctUntilChanged(),
        takeUntilDestroyed(this.destroyRef)
      )
      .subscribe(() => this.loadLogs());

    this.watcherFilter.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.loadLogs());

    this.dateFromFilter.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.loadLogs());

    this.dateToFilter.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => this.loadLogs());
  }

  toggleEventType(eventType: string): void {
    const types = new Set(this.selectedEventTypes());
    if (types.has(eventType)) {
      types.delete(eventType);
    } else {
      types.add(eventType);
    }
    this.selectedEventTypes.set(types);
    this.loadLogs();
  }

  isEventTypeSelected(eventType: string): boolean {
    return this.selectedEventTypes().has(eventType);
  }

  ngOnInit(): void {
    this.loadWatchers();
    this.loadLogs();
    this.setupWebSocket();
  }

  private setupWebSocket(): void {
    this.wsService.messages
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (message: WsMessage) => {
          if (message.type === WsMessageType.FILE_EVENT) {
            // Reload current page when new file event arrives
            this.loadLogs();
          }
        }
      });
  }

  loadWatchers(): void {
    this.watcherService.getWatchers()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            this.watchers.set(response.data);
          }
        },
        error: () => {
          this.notificationService.error('Failed to load watchers');
        }
      });
  }

  loadLogs(): void {
    this.selection.set(new Set());
    this.isLoading.set(true);

    const params: LogQueryParams = {
      page: this.currentPage(),
      limit: this.pageSize()
    };

    // Add filters if set
    const watcherIds = this.watcherFilter.value;
    if (watcherIds && watcherIds.length > 0) {
      // API only accepts single watcher_id, so we'll use the first one
      // In a real implementation, you might want to handle multiple watchers differently
      params.watcher_id = watcherIds[0];
    }

    const eventTypes = Array.from(this.selectedEventTypes());
    if (eventTypes.length > 0) {
      // API accepts single event type, use first selected
      params.event_type = eventTypes[0];
    }

    const search = this.searchFilter.value;
    if (search && search.trim()) {
      params.search = search.trim();
    }

    const dateFrom = this.dateFromFilter.value;
    if (dateFrom) {
      const startOfDay = new Date(dateFrom);
      startOfDay.setHours(0, 0, 0, 0);
      params.from = startOfDay.toISOString();
    }

    const dateTo = this.dateToFilter.value;
    if (dateTo) {
      const endOfDay = new Date(dateTo);
      endOfDay.setHours(23, 59, 59, 999);
      params.to = endOfDay.toISOString();
    }

    this.eventLogService.getLogs(params)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response) => {
          this.isLoading.set(false);
          if (response.success && response.data) {
            this.events.set(response.data.items);
            this.totalCount.set(response.data.total);
          }
        },
        error: () => {
          this.isLoading.set(false);
          this.notificationService.error('Failed to load logs');
        }
      });
  }

  onPageChange(event: PageEvent): void {
    this.currentPage.set(event.pageIndex + 1);
    this.pageSize.set(event.pageSize);
    this.loadLogs();
  }

  openDetail(event: FileEvent): void {
    this.dialog.open(LogDetailDialogComponent, {
      width: '800px',
      maxWidth: '95vw',
      maxHeight: '90vh',
      data: { event }
    });
  }

  exportLogs(): void {
    const watcherIds = this.watcherFilter.value;

    if (!watcherIds || watcherIds.length === 0) {
      this.notificationService.warning('Please select a watcher to export logs');
      return;
    }

    this.isExporting.set(true);

    const params: LogQueryParams = {};

    // Add filters
    const eventTypes = Array.from(this.selectedEventTypes());
    if (eventTypes.length > 0) {
      params.event_type = eventTypes[0];
    }

    const search = this.searchFilter.value;
    if (search && search.trim()) {
      params.search = search.trim();
    }

    const dateFrom = this.dateFromFilter.value;
    if (dateFrom) {
      const startOfDay = new Date(dateFrom);
      startOfDay.setHours(0, 0, 0, 0);
      params.from = startOfDay.toISOString();
    }

    const dateTo = this.dateToFilter.value;
    if (dateTo) {
      const endOfDay = new Date(dateTo);
      endOfDay.setHours(23, 59, 59, 999);
      params.to = endOfDay.toISOString();
    }

    // Export first selected watcher
    this.eventLogService.exportLogs(watcherIds[0], params)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (blob) => {
          this.isExporting.set(false);

          // Create download link
          const url = window.URL.createObjectURL(blob);
          const link = document.createElement('a');
          link.href = url;

          const today = new Date().toISOString().split('T')[0];
          link.download = `sentinel-logs-${today}.xlsx`;

          document.body.appendChild(link);
          link.click();
          document.body.removeChild(link);

          window.URL.revokeObjectURL(url);

          this.notificationService.success('Logs exported successfully');
        },
        error: () => {
          this.isExporting.set(false);
          this.notificationService.error('Failed to export logs');
        }
      });
  }

  isSelected(id: number): boolean {
    return this.selection().has(id);
  }

  toggleSelection(id: number): void {
    const sel = new Set(this.selection());
    if (sel.has(id)) {
      sel.delete(id);
    } else {
      sel.add(id);
    }
    this.selection.set(sel);
  }

  toggleAllSelection(): void {
    if (this.isAllSelected()) {
      this.selection.set(new Set());
    } else {
      const allIds = new Set(this.events().map(e => e.id));
      this.selection.set(allIds);
    }
  }

  deleteSelected(): void {
    const ids = Array.from(this.selection());
    if (ids.length === 0) return;

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'logs.delete_confirm_title',
        message: 'logs.delete_confirm_message',
        confirmText: 'common.delete',
        cancelText: 'common.cancel'
      }
    });

    dialogRef.afterClosed().subscribe(confirmed => {
      if (confirmed) {
        this.eventLogService.deleteLogs(ids).subscribe({
          next: (response) => {
            if (response.success) {
              this.notificationService.success('logs.delete_success');
              this.selection.set(new Set());
              this.loadLogs();
            }
          },
          error: () => {
            this.notificationService.error('logs.delete_error');
          }
        });
      }
    });
  }
}
