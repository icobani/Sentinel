import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '../../pipes/translate.pipe';

@Component({
  selector: 'app-empty-state',
  standalone: true,
  imports: [CommonModule, TranslatePipe],
  template: `
    <div class="empty-state">
      <div class="empty-icon">{{ icon }}</div>
      <h3 class="empty-title">{{ title | translate }}</h3>
      <p class="empty-message">{{ message | translate }}</p>
      <ng-content></ng-content>
    </div>
  `,
  styles: [`
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 64px 24px;
      text-align: center;
    }

    .empty-icon {
      font-size: 64px;
      margin-bottom: 16px;
      opacity: 0.3;
    }

    .empty-title {
      font-size: 24px;
      font-weight: 500;
      margin: 0 0 8px 0;
      color: rgba(0, 0, 0, 0.87);
    }

    .empty-message {
      font-size: 16px;
      margin: 0 0 24px 0;
      color: rgba(0, 0, 0, 0.6);
      max-width: 400px;
    }
  `]
})
export class EmptyStateComponent {
  @Input() icon = '📭';
  @Input() title = 'empty.no_data';
  @Input() message = 'empty.no_data';
}
