import { Component, signal, OnInit, inject, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet, ChildrenOutletContexts } from '@angular/router';
import { SidebarComponent } from './shared/components/sidebar/sidebar.component';
import { ToolbarComponent } from './shared/components/toolbar/toolbar.component';
import { PendingRestartBannerComponent } from './shared/components/pending-restart-banner/pending-restart-banner.component';
import { RestartOverlayComponent } from './features/setup/restart-overlay/restart-overlay.component';
import { MatSidenavModule } from '@angular/material/sidenav';
import { MatDialog } from '@angular/material/dialog';
import { StatusService } from './core/services/status.service';
import { NotificationService } from './core/services/notification.service';
import { TranslateService } from './core/services/translate.service';
import { WebSocketService } from './core/services/websocket.service';
import { ConfirmDialogComponent } from './shared/components/confirm-dialog/confirm-dialog.component';
import { interval, takeWhile } from 'rxjs';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { routeAnimations } from './shared/animations/route.animations';
import { fadeInOut } from './shared/animations/fade.animations';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    RouterOutlet,
    MatSidenavModule,
    SidebarComponent,
    ToolbarComponent,
    PendingRestartBannerComponent,
    RestartOverlayComponent
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss',
  animations: [routeAnimations, fadeInOut]
})
export class App implements OnInit {
  private readonly dialog = inject(MatDialog);
  private readonly statusService = inject(StatusService);
  private readonly notificationService = inject(NotificationService);
  private readonly translateService = inject(TranslateService);
  private readonly wsService = inject(WebSocketService);
  private readonly contexts = inject(ChildrenOutletContexts);
  private readonly destroyRef = inject(DestroyRef);

  sidebarCollapsed = signal(false);
  isOnline = signal(true);
  showRestartBanner = signal(false);
  showRestartOverlay = signal(false);
  isDarkTheme = signal(false);
  currentLanguage = this.translateService.getCurrentLanguage();
  version = signal('');

  getRouteAnimationData() {
    return this.contexts.getContext('primary')?.route?.snapshot?.data?.['animation'];
  }

  ngOnInit(): void {
    // Connect WebSocket for app-wide real-time updates
    this.wsService.connect();

    // Load theme preference from localStorage
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'dark') {
      this.isDarkTheme.set(true);
      document.body.classList.add('dark-theme');
    }

    // Load system status to get version
    this.loadSystemStatus();

    // Check for pending restart on init
    this.checkPendingRestart();

    // Poll for pending restart every 10 seconds
    interval(10000)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(() => {
        this.checkPendingRestart();
      });
  }

  private loadSystemStatus(): void {
    this.statusService.getSystemStatus().subscribe({
      next: (response) => {
        if (response.success && response.data) {
          this.version.set(response.data.version);
        }
      },
      error: () => {
        // Silently fail - not critical
      }
    });
  }

  private checkPendingRestart(): void {
    this.statusService.checkPendingRestart().subscribe({
      next: (response) => {
        if (response.success && response.data) {
          this.showRestartBanner.set(response.data.pending);
        }
      },
      error: () => {
        // Silently fail - not critical
      }
    });
  }

  toggleSidebar(): void {
    this.sidebarCollapsed.update(value => !value);
  }

  onLanguageChange(language: string): void {
    this.translateService.setLanguage(language);
  }

  onThemeChange(isDark: boolean): void {
    this.isDarkTheme.set(isDark);
    localStorage.setItem('theme', isDark ? 'dark' : 'light');

    if (isDark) {
      document.body.classList.add('dark-theme');
    } else {
      document.body.classList.remove('dark-theme');
    }
  }

  onRestartService(): void {
    const dialogRef = this.dialog.open(ConfirmDialogComponent, {
      data: {
        title: 'restart.confirm_title',
        message: 'restart.confirm_message',
        confirmText: 'restart.button',
        cancelText: 'common.cancel'
      }
    });

    dialogRef.afterClosed()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(confirmed => {
        if (confirmed) {
          this.triggerRestart();
        }
      });
  }

  private triggerRestart(): void {
    this.statusService.restartService().subscribe({
      next: () => {
        this.showRestartOverlay.set(true);
        this.showRestartBanner.set(false);
      },
      error: () => {
        this.notificationService.showError('Failed to restart service');
      }
    });
  }
}
