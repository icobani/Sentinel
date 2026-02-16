import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '../../pipes/translate.pipe';
import { WatcherStatus } from '../../../core/models/watcher.model';

@Component({
  selector: 'app-status-badge',
  standalone: true,
  imports: [CommonModule, TranslatePipe],
  template: `
    <span class="status-badge" [class]="'status-' + status.toLowerCase()">
      <span class="status-dot"></span>
      <span class="status-text">{{ getStatusText() | translate }}</span>
    </span>
  `,
  styles: [`
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: 12px;
      font-size: 14px;
      font-weight: 500;
      transition: all 0.3s ease;
    }

    .status-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      position: relative;
      transition: all 0.3s ease;
    }

    .status-active {
      background-color: rgba(76, 175, 80, 0.1);
      color: #4CAF50;
    }

    .status-active .status-dot {
      background-color: #4CAF50;
      animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    .status-active .status-dot::before {
      content: '';
      position: absolute;
      top: -2px;
      left: -2px;
      right: -2px;
      bottom: -2px;
      border-radius: 50%;
      background-color: #4CAF50;
      opacity: 0.3;
      animation: pulse-ring 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    @keyframes pulse {
      0%, 100% {
        opacity: 1;
      }
      50% {
        opacity: 0.7;
      }
    }

    @keyframes pulse-ring {
      0% {
        transform: scale(0.8);
        opacity: 0.5;
      }
      50% {
        transform: scale(1.2);
        opacity: 0;
      }
      100% {
        transform: scale(0.8);
        opacity: 0;
      }
    }

    .status-inactive {
      background-color: rgba(158, 158, 158, 0.1);
      color: #9E9E9E;
    }

    .status-inactive .status-dot {
      background-color: #9E9E9E;
    }

    .status-error {
      background-color: rgba(244, 67, 54, 0.1);
      color: #F44336;
    }

    .status-error .status-dot {
      background-color: #F44336;
      animation: pulse-error 1.5s cubic-bezier(0.4, 0, 0.6, 1) infinite;
    }

    @keyframes pulse-error {
      0%, 100% {
        opacity: 1;
      }
      50% {
        opacity: 0.5;
      }
    }
  `]
})
export class StatusBadgeComponent {
  @Input() status: WatcherStatus = WatcherStatus.INACTIVE;

  getStatusText(): string {
    switch (this.status) {
      case WatcherStatus.ACTIVE:
        return 'status.active';
      case WatcherStatus.INACTIVE:
        return 'status.inactive';
      case WatcherStatus.ERROR:
        return 'status.error';
      default:
        return 'status.unknown';
    }
  }
}
