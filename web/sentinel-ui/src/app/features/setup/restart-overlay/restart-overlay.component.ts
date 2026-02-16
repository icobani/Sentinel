import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatIconModule } from '@angular/material/icon';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';
import { interval, takeWhile, finalize } from 'rxjs';

import { StatusService } from '../../../core/services/status.service';

@Component({
  selector: 'app-restart-overlay',
  standalone: true,
  imports: [CommonModule, MatProgressBarModule, MatIconModule, TranslatePipe],
  templateUrl: './restart-overlay.component.html',
  styleUrls: ['./restart-overlay.component.scss']
})
export class RestartOverlayComponent implements OnInit {
  private readonly statusService = inject(StatusService);

  readonly countdown = signal(10);
  readonly progress = signal(0);
  readonly retryAttempt = signal(0);
  readonly maxRetries = 3;
  readonly retryDelay = 3000; // 3 seconds

  // Circular progress ring calculations
  readonly radius = 52;
  readonly circumference = 2 * Math.PI * this.radius;

  strokeDashoffset(): number {
    const progressValue = this.progress();
    return this.circumference - (progressValue / 100) * this.circumference;
  }

  ngOnInit(): void {
    this.startCountdown();
  }

  private startCountdown(): void {
    const totalSeconds = 10;
    
    interval(1000)
      .pipe(
        takeWhile(() => this.countdown() > 0),
        finalize(() => this.onCountdownComplete())
      )
      .subscribe(() => {
        const newCount = this.countdown() - 1;
        this.countdown.set(newCount);
        this.progress.set(((totalSeconds - newCount) / totalSeconds) * 100);
      });
  }

  private onCountdownComplete(): void {
    this.attemptReload();
  }

  private attemptReload(): void {
    if (this.retryAttempt() >= this.maxRetries) {
      window.location.reload();
      return;
    }

    this.retryAttempt.update(attempt => attempt + 1);

    // Try to ping the server
    this.statusService.getSystemStatus().subscribe({
      next: () => {
        // Server is responsive, reload the page
        window.location.reload();
      },
      error: () => {
        // Server not ready yet, retry after delay
        if (this.retryAttempt() < this.maxRetries) {
          setTimeout(() => this.attemptReload(), this.retryDelay);
        } else {
          // Max retries reached, reload anyway
          window.location.reload();
        }
      }
    });
  }
}
