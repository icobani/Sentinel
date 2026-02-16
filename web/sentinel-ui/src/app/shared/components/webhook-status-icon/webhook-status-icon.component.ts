import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatTooltipModule } from '@angular/material/tooltip';

@Component({
  selector: 'app-webhook-status-icon',
  standalone: true,
  imports: [CommonModule, MatTooltipModule],
  template: `
    <span
      *ngIf="statusCode !== undefined"
      class="webhook-icon"
      [class.success]="isSuccess()"
      [class.warning]="isWarning()"
      [class.error]="isError()"
      [matTooltip]="getTooltip()"
    >
      {{ getIcon() }}
    </span>
  `,
  styles: [`
    .webhook-icon {
      font-size: 18px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 24px;
      height: 24px;
      cursor: help;
    }

    .success {
      color: #4CAF50;
    }

    .warning {
      color: #FF9800;
    }

    .error {
      color: #F44336;
    }
  `]
})
export class WebhookStatusIconComponent {
  @Input() statusCode?: number;
  @Input() errorMessage?: string;
  @Input() retryCount?: number;
  @Input() maxRetries?: number;

  isSuccess(): boolean {
    return this.statusCode === 200;
  }

  isWarning(): boolean {
    return this.statusCode !== undefined && this.statusCode > 0 && this.statusCode !== 200;
  }

  isError(): boolean {
    return this.statusCode === undefined || this.statusCode === 0;
  }

  getIcon(): string {
    if (this.isSuccess()) {
      return '✓';
    } else if (this.isWarning()) {
      return '✕';
    } else {
      return '⚠';
    }
  }

  getTooltip(): string {
    if (this.isSuccess()) {
      return '200 OK';
    } else if (this.isWarning()) {
      return this.errorMessage || `${this.statusCode} Error`;
    } else {
      const retryInfo = this.retryCount !== undefined && this.maxRetries !== undefined
        ? ` (${this.retryCount}/${this.maxRetries} attempts)`
        : '';
      return `Connection failed${retryInfo}`;
    }
  }
}
