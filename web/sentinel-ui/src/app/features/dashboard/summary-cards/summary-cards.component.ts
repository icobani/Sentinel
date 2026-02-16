import { Component, Input, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { SystemStatus } from '../../../core/models/api-response.model';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';
import { TranslateService } from '../../../core/services/translate.service';

interface StatCard {
  titleKey: string;
  value: string;
  subtitle?: string;
  icon: string;
  color: string;
}

@Component({
  selector: 'app-summary-cards',
  standalone: true,
  imports: [CommonModule, MatCardModule, MatIconModule, TranslatePipe],
  template: `
    <div class="summary-cards-container">
      <mat-card *ngFor="let card of cards" [class]="'stat-card ' + card.color">
        <mat-card-content>
          <div class="card-header">
            <mat-icon class="card-icon">{{ card.icon }}</mat-icon>
            <h3 class="card-title">{{ card.titleKey | translate }}</h3>
          </div>
          <div class="card-body">
            <p class="card-value">{{ card.value }}</p>
            <p class="card-subtitle" *ngIf="card.subtitle">{{ card.subtitle }}</p>
          </div>
        </mat-card-content>
      </mat-card>
    </div>
  `,
  styles: [`
    .summary-cards-container {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 24px;
      margin-bottom: 32px;
    }

    .stat-card {
      padding: 0;
      border-left: 4px solid;
      transition: transform 0.2s, box-shadow 0.2s;
      cursor: default;
    }

    .stat-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
    }

    .stat-card.primary {
      border-color: #1976d2;
    }

    .stat-card.success {
      border-color: #388e3c;
    }

    .stat-card.warning {
      border-color: #f57c00;
    }

    .stat-card.info {
      border-color: #0288d1;
    }

    mat-card-content {
      padding: 20px;
    }

    .card-header {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 16px;
    }

    .card-icon {
      font-size: 32px;
      width: 32px;
      height: 32px;
      color: inherit;
      opacity: 0.8;
    }

    .card-title {
      font-size: 16px;
      font-weight: 500;
      margin: 0;
      color: rgba(0, 0, 0, 0.7);
    }

    .card-body {
      margin-left: 44px;
    }

    .card-value {
      font-size: 36px;
      font-weight: 600;
      margin: 0 0 4px 0;
      line-height: 1.2;
    }

    .card-subtitle {
      font-size: 14px;
      margin: 0;
      color: rgba(0, 0, 0, 0.6);
    }

    @media (max-width: 768px) {
      .summary-cards-container {
        grid-template-columns: 1fr;
      }
    }
  `]
})
export class SummaryCardsComponent {
  private translateService = inject(TranslateService);
  @Input() status: SystemStatus | null = null;

  get cards(): StatCard[] {
    if (!this.status) {
      return [];
    }

    const watchersTotal = this.status.watchers_total ?? 0;
    const watchersActive = this.status.watchers_active ?? 0;
    const eventsToday = this.status.events_today ?? 0;
    const webhookSuccessRate = this.status.webhook_success_rate ?? 0;
    const uptimeSeconds = this.status.uptime_seconds ?? 0;

    return [
      {
        titleKey: 'dashboard.total_watchers',
        value: watchersTotal.toString(),
        subtitle: `${watchersActive} ${this.translateService.get('dashboard.active_watchers')}`,
        icon: 'visibility',
        color: 'primary'
      },
      {
        titleKey: 'dashboard.events_today',
        value: eventsToday.toString(),
        subtitle: undefined,
        icon: 'event_note',
        color: 'info'
      },
      {
        titleKey: 'dashboard.webhook_success_rate',
        value: `${webhookSuccessRate.toFixed(1)}%`,
        subtitle: undefined,
        icon: 'check_circle',
        color: webhookSuccessRate >= 90 ? 'success' : 'warning'
      },
      {
        titleKey: 'dashboard.system_uptime',
        value: this.formatUptime(uptimeSeconds),
        subtitle: undefined,
        icon: 'schedule',
        color: 'success'
      }
    ];
  }

  private formatUptime(seconds: number): string {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (days > 0) {
      return `${days}d ${hours}h`;
    } else if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else {
      return `${minutes}m`;
    }
  }
}
