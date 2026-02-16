import { Component, OnInit, inject, signal, computed, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatChipsModule } from '@angular/material/chips';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatDialog } from '@angular/material/dialog';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { finalize } from 'rxjs';

import { WatcherService } from '../../../core/services/watcher.service';
import { NotificationService } from '../../../core/services/notification.service';
import { Watcher, WatcherStatus, WatchMode } from '../../../core/models/watcher.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge/status-badge.component';
import { LoadingSkeletonComponent } from '../../../shared/components/loading-skeleton/loading-skeleton.component';
import { EmptyStateComponent } from '../../../shared/components/empty-state/empty-state.component';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { TruncatePipe } from '../../../shared/pipes/truncate.pipe';
import { RelativeTimePipe } from '../../../shared/pipes/relative-time.pipe';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';

@Component({
  selector: 'app-watcher-list',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
    MatTooltipModule,
    StatusBadgeComponent,
    LoadingSkeletonComponent,
    EmptyStateComponent,
    TruncatePipe,
    RelativeTimePipe,
    TranslatePipe
  ],
  templateUrl: './watcher-list.component.html',
  styleUrls: ['./watcher-list.component.scss']
})
export class WatcherListComponent implements OnInit {
  private readonly destroyRef = inject(DestroyRef);
  private readonly watcherService = inject(WatcherService);
  private readonly notificationService = inject(NotificationService);
  private readonly router = inject(Router);
  private readonly dialog = inject(MatDialog);

  readonly watchers = signal<Watcher[]>([]);
  readonly loading = signal(true);
  readonly actionInProgress = signal<number | null>(null);

  readonly WatchMode = WatchMode;
  readonly WatcherStatus = WatcherStatus;

  ngOnInit(): void {
    this.loadWatchers();
  }

  loadWatchers(): void {
    this.loading.set(true);
    this.watcherService.getWatchers()
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false))
      )
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            this.watchers.set(response.data);
          }
        },
        error: () => {
          this.notificationService.showError('Failed to load watchers');
        }
      });
  }

  onAddWatcher(): void {
    this.router.navigate(['/setup/new']);
  }

  onViewDetails(watcherId: number): void {
    this.router.navigate(['/setup', watcherId]);
  }

  onEditWatcher(watcherId: number, event: Event): void {
    event.stopPropagation();
    this.router.navigate(['/setup', watcherId, 'edit']);
  }

  onDeleteWatcher(watcher: Watcher, event: Event): void {
    event.stopPropagation();

    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'setup.delete_confirm_title',
        message: 'setup.delete_confirm_message',
        confirmText: 'common.delete',
        cancelText: 'common.cancel'
      }
    });

    dialogRef.afterClosed()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(confirmed => {
        if (confirmed) {
          this.deleteWatcher(watcher);
        }
      });
  }

  private deleteWatcher(watcher: Watcher): void {
    this.actionInProgress.set(watcher.id);
    this.watcherService.deleteWatcher(watcher.id)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.actionInProgress.set(null))
      )
      .subscribe({
        next: () => {
          this.notificationService.showSuccess('setup.watcher_deleted');
          this.loadWatchers();
        },
        error: () => {
          this.notificationService.showError('Failed to delete watcher');
        }
      });
  }

  onStartWatcher(watcherId: number, event: Event): void {
    event.stopPropagation();
    this.actionInProgress.set(watcherId);
    this.watcherService.startWatcher(watcherId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.actionInProgress.set(null))
      )
      .subscribe({
        next: () => {
          this.notificationService.showSuccess('setup.watcher_started');
          this.loadWatchers();
        },
        error: () => {
          this.notificationService.showError('Failed to start watcher');
        }
      });
  }

  onStopWatcher(watcherId: number, event: Event): void {
    event.stopPropagation();
    this.actionInProgress.set(watcherId);
    this.watcherService.stopWatcher(watcherId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.actionInProgress.set(null))
      )
      .subscribe({
        next: () => {
          this.notificationService.showSuccess('setup.watcher_stopped');
          this.loadWatchers();
        },
        error: () => {
          this.notificationService.showError('Failed to stop watcher');
        }
      });
  }

  onRestartWatcher(watcherId: number, event: Event): void {
    event.stopPropagation();
    this.actionInProgress.set(watcherId);
    this.watcherService.restartWatcher(watcherId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.actionInProgress.set(null))
      )
      .subscribe({
        next: () => {
          this.notificationService.showSuccess('setup.watcher_restarted');
          this.loadWatchers();
        },
        error: () => {
          this.notificationService.showError('Failed to restart watcher');
        }
      });
  }

  isActionInProgress(watcherId: number): boolean {
    return this.actionInProgress() === watcherId;
  }

  getModeLabel(mode: WatchMode): string {
    return mode === WatchMode.WATCH ? 'setup.watch_mode' : 'setup.poll_mode';
  }

  getModeColor(mode: WatchMode): string {
    return mode === WatchMode.WATCH ? 'primary' : 'accent';
  }
}
