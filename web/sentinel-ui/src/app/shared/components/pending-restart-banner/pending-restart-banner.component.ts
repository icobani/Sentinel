import { Component, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { TranslatePipe } from '../../pipes/translate.pipe';

@Component({
  selector: 'app-pending-restart-banner',
  standalone: true,
  imports: [CommonModule, MatButtonModule, MatIconModule, TranslatePipe],
  template: `
    <div class="restart-banner">
      <mat-icon>warning</mat-icon>
      <span class="banner-message">
        {{ 'restart.banner' | translate }}
      </span>
      <button mat-raised-button color="warn" (click)="restart.emit()">
        {{ 'restart.button' | translate }}
      </button>
    </div>
  `,
  styles: [`
    .restart-banner {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 12px 24px;
      background-color: #fff3cd;
      border-bottom: 2px solid #ffc107;
      color: #856404;
    }

    mat-icon {
      color: #ffc107;
    }

    .banner-message {
      flex: 1;
      font-size: 16px;
      font-weight: 500;
    }

    button {
      white-space: nowrap;
    }
  `]
})
export class PendingRestartBannerComponent {
  @Output() restart = new EventEmitter<void>();
}
