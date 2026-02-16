import { Component, inject, signal, OnInit, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatDividerModule } from '@angular/material/divider';
import { MatChipsModule } from '@angular/material/chips';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatExpansionModule } from '@angular/material/expansion';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FileEvent } from '../../../core/models/event.model';
import { WebhookLog } from '../../../core/models/webhook-log.model';
import { Watcher } from '../../../core/models/watcher.model';
import { EventLogService } from '../../../core/services/event-log.service';
import { WatcherService } from '../../../core/services/watcher.service';
import { FileSizePipe } from '../../../shared/pipes/file-size.pipe';
import { FormatDatePipe } from '../../../shared/pipes/format-date.pipe';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';
import { dialogScale } from '../../../shared/animations/dialog.animations';

export interface LogDetailDialogData {
  event: FileEvent;
}

@Component({
  selector: 'app-log-detail-dialog',
  standalone: true,
  imports: [
    CommonModule,
    MatDialogModule,
    MatButtonModule,
    MatIconModule,
    MatDividerModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatExpansionModule,
    FileSizePipe,
    FormatDatePipe,
    TranslatePipe
  ],
  animations: [dialogScale],
  template: `
    <div @dialogScale>
      <div class="dialog-header">
        <h2 mat-dialog-title>{{ ('events.' + data.event_type) | translate }}</h2>
        <button mat-icon-button mat-dialog-close>
          <mat-icon>close</mat-icon>
        </button>
      </div>

      <mat-dialog-content>
      <div class="detail-section">
        <h3>{{ 'common.details' | translate }}</h3>
        <div class="detail-row">
          <span class="detail-label">{{ 'setup.watcher_name' | translate }}:</span>
          <span class="detail-value">{{ data.watcher_name }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ 'events.full_path' | translate }}:</span>
          <span class="detail-value">{{ data.file_path }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ 'events.file_size' | translate }}:</span>
          <span class="detail-value">{{ data.file_size | fileSize }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ 'events.timestamp' | translate }}:</span>
          <span class="detail-value">{{ data.timestamp | formatDate }}</span>
        </div>
        <div class="detail-row" *ngIf="data.event_type === 'renamed' && data.old_path">
          <span class="detail-label">{{ 'events.old_path' | translate }}:</span>
          <span class="detail-value">{{ data.old_path }}</span>
        </div>
      </div>

      <mat-divider></mat-divider>

      <div class="detail-section" *ngIf="hasWebhookConfig || hasWebhookInfo">
        <h3>{{ 'events.webhook_details' | translate }}</h3>

        <div class="detail-row" *ngIf="hasWebhookConfig">
          <span class="detail-label">{{ 'events.webhook_url' | translate }}:</span>
          <span class="detail-value webhook-url">{{ webhookUrl }}</span>
        </div>

        <div class="detail-row" *ngIf="hasWebhookInfo">
          <span class="detail-label">{{ 'events.webhook_status' | translate }}:</span>
          <span class="detail-value">
            <mat-chip [class.success-chip]="isWebhookSuccess" [class.error-chip]="!isWebhookSuccess">
              {{ data.webhook_status }}
            </mat-chip>
          </span>
        </div>

        <div class="detail-row" *ngIf="data.webhook_error">
          <span class="detail-label">{{ 'events.webhook_error' | translate }}:</span>
          <span class="detail-value error-text">{{ data.webhook_error }}</span>
        </div>

        <div class="detail-row" *ngIf="data.retry_count !== undefined">
          <span class="detail-label">{{ 'events.retry_count' | translate }}:</span>
          <span class="detail-value">
            {{ data.retry_count }}
            <span *ngIf="watcher()?.webhook?.retry?.max_attempts">
              / {{ watcher()!.webhook!.retry!.max_attempts }}
            </span>
          </span>
        </div>
      </div>

      <!-- Additional webhook logs section -->
      <div class="detail-section" *ngIf="webhookLogs().length > 0">
        <h3>{{ 'events.retry_history' | translate }}</h3>
        <mat-accordion>
          <mat-expansion-panel *ngFor="let log of webhookLogs(); let i = index">
            <mat-expansion-panel-header>
              <mat-panel-title>
                <span class="retry-title">
                  <mat-icon [class.success-icon]="log.success" [class.error-icon]="!log.success">
                    {{ log.success ? 'check_circle' : 'error' }}
                  </mat-icon>
                  {{ 'events.attempt' | translate }} {{ i + 1 }} - {{ 'events.status' | translate }} {{ log.status_code }}
                  <span class="response-time">({{ formatDuration(log.response_time) }})</span>
                </span>
              </mat-panel-title>
            </mat-expansion-panel-header>

            <div class="retry-details">
              <div class="detail-row">
                <span class="detail-label">{{ 'events.timestamp' | translate }}:</span>
                <span class="detail-value">{{ log.created_at | formatDate }}</span>
              </div>

              <div class="detail-row" *ngIf="log.error">
                <span class="detail-label">{{ 'common.error' | translate }}:</span>
                <span class="detail-value error-text">{{ log.error }}</span>
              </div>

              <div class="detail-row" *ngIf="log.request_payload">
                <span class="detail-label">{{ 'events.request_body' | translate }}:</span>
                <pre class="json-pre">{{ formatJson(log.request_payload) }}</pre>
              </div>

              <div class="detail-row" *ngIf="log.response_body">
                <span class="detail-label">{{ 'events.response_body' | translate }}:</span>
                <pre class="json-pre">{{ formatJson(log.response_body) }}</pre>
              </div>
            </div>
          </mat-expansion-panel>
        </mat-accordion>
      </div>
    </mat-dialog-content>

      <mat-dialog-actions align="end">
        <button mat-button mat-dialog-close>{{ 'common.close' | translate }}</button>
      </mat-dialog-actions>
    </div>
  `,
  styles: [`
    .dialog-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 20px 24px 0;
    }

    h2 {
      margin: 0;
      font-size: 24px;
      font-weight: 500;
    }

    mat-dialog-content {
      padding: 24px;
      min-width: 600px;
      max-width: 800px;
    }

    .detail-section {
      margin-bottom: 24px;

      &:last-child {
        margin-bottom: 0;
      }
    }

    h3 {
      font-size: 18px;
      font-weight: 500;
      margin: 0 0 16px 0;
      color: rgba(0, 0, 0, 0.87);
    }

    .detail-row {
      display: flex;
      gap: 12px;
      margin-bottom: 12px;
      font-size: 16px;

      &:last-child {
        margin-bottom: 0;
      }
    }

    .detail-label {
      font-weight: 500;
      min-width: 180px;
      color: rgba(0, 0, 0, 0.6);
    }

    .detail-value {
      flex: 1;
      word-break: break-word;
      color: rgba(0, 0, 0, 0.87);
    }

    .error-text {
      color: #f44336;
    }

    .webhook-url {
      font-family: monospace;
      font-size: 14px;
      word-break: break-all;
    }

    mat-chip {
      font-weight: 500;
    }

    .success-chip {
      background-color: #4caf50 !important;
      color: white !important;
    }

    .error-chip {
      background-color: #f44336 !important;
      color: white !important;
    }

    mat-divider {
      margin: 24px 0;
    }

    mat-dialog-actions {
      padding: 16px 24px;
    }

    mat-accordion {
      margin-top: 8px;
    }

    .retry-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 15px;
    }

    .success-icon {
      color: #4caf50;
    }

    .error-icon {
      color: #f44336;
    }

    .response-time {
      color: rgba(0, 0, 0, 0.5);
      font-size: 13px;
      margin-left: 4px;
    }

    .retry-details {
      padding: 12px 0;
    }

    .json-pre {
      background-color: rgba(0, 0, 0, 0.04);
      padding: 12px;
      border-radius: 4px;
      overflow-x: auto;
      font-family: 'Courier New', monospace;
      font-size: 13px;
      line-height: 1.5;
      margin: 8px 0 0 0;
      max-height: 300px;
      overflow-y: auto;
    }

    /* Dark theme support */
    :host-context(.dark-theme) {
      h3 {
        color: rgba(255, 255, 255, 0.87);
      }

      .detail-label {
        color: rgba(255, 255, 255, 0.6);
      }

      .detail-value {
        color: rgba(255, 255, 255, 0.87);
      }

      .response-time {
        color: rgba(255, 255, 255, 0.5);
      }

      .json-pre {
        background-color: rgba(255, 255, 255, 0.05);
        color: rgba(255, 255, 255, 0.87);
      }
    }
  `]
})
export class LogDetailDialogComponent implements OnInit {
  private readonly dialogData: LogDetailDialogData = inject(MAT_DIALOG_DATA);
  private readonly eventLogService = inject(EventLogService);
  private readonly watcherService = inject(WatcherService);
  private readonly destroyRef = inject(DestroyRef);

  readonly data = this.dialogData.event;
  readonly webhookLogs = signal<WebhookLog[]>([]);
  readonly watcher = signal<Watcher | null>(null);
  readonly isLoadingWebhook = signal(false);

  get hasWebhookInfo(): boolean {
    return this.data.webhook_status !== undefined && this.data.webhook_status !== null;
  }

  get isWebhookSuccess(): boolean {
    return this.data.webhook_status === 200;
  }

  get hasWebhookConfig(): boolean {
    const w = this.watcher();
    return !!(w && w.webhook && w.webhook.url);
  }

  get webhookUrl(): string {
    const w = this.watcher();
    return w?.webhook?.url || '';
  }

  ngOnInit(): void {
    // Load watcher details to get webhook URL
    if (this.data.watcher_id) {
      this.watcherService.getWatcher(this.data.watcher_id)
        .pipe(takeUntilDestroyed(this.destroyRef))
        .subscribe({
          next: (response) => {
            if (response.success && response.data) {
              this.watcher.set(response.data);
            }
          },
          error: () => {
            // Silently fail - not critical
          }
        });
    }

    // Load webhook logs if event has webhook info
    if (this.hasWebhookInfo && this.data.id) {
      this.loadWebhookLogs();
    }
  }

  loadWebhookLogs(): void {
    this.isLoadingWebhook.set(true);

    // Since we don't have the API endpoint yet, we'll work with existing data
    // TODO: Implement backend endpoint GET /api/v1/events/:id/webhook-logs
    // For now, display available webhook info from the event
    this.isLoadingWebhook.set(false);
  }

  formatDuration(ms: number): string {
    if (ms < 1000) {
      return `${ms}ms`;
    }
    return `${(ms / 1000).toFixed(2)}s`;
  }

  formatJson(jsonString: string | undefined): string {
    if (!jsonString) return '';
    try {
      return JSON.stringify(JSON.parse(jsonString), null, 2);
    } catch {
      return jsonString;
    }
  }
}
