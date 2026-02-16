import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

type SkeletonVariant = 'line' | 'card' | 'table' | 'circle' | 'custom';

@Component({
  selector: 'app-loading-skeleton',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="skeleton-wrapper" [class]="'skeleton-' + variant">
      <ng-container *ngIf="variant === 'card'">
        <div *ngFor="let item of items" class="skeleton-card">
          <div class="skeleton-card-header">
            <div class="skeleton-item skeleton-shimmer" style="height: 24px; width: 60%; margin-bottom: 8px;"></div>
          </div>
          <div class="skeleton-card-body">
            <div class="skeleton-item skeleton-shimmer" style="height: 16px; width: 100%; margin-bottom: 8px;"></div>
            <div class="skeleton-item skeleton-shimmer" style="height: 16px; width: 80%; margin-bottom: 8px;"></div>
            <div class="skeleton-item skeleton-shimmer" style="height: 16px; width: 90%;"></div>
          </div>
        </div>
      </ng-container>

      <ng-container *ngIf="variant === 'table'">
        <div class="skeleton-table">
          <div class="skeleton-table-header">
            <div class="skeleton-item skeleton-shimmer" style="height: 40px; width: 100%;"></div>
          </div>
          <div *ngFor="let item of items" class="skeleton-table-row">
            <div class="skeleton-item skeleton-shimmer" style="height: 48px; width: 100%;"></div>
          </div>
        </div>
      </ng-container>

      <ng-container *ngIf="variant === 'line' || variant === 'custom'">
        <div *ngFor="let item of items" class="skeleton-item skeleton-shimmer" [style.height.px]="heightValue">
        </div>
      </ng-container>

      <ng-container *ngIf="variant === 'circle'">
        <div *ngFor="let item of items" class="skeleton-circle skeleton-shimmer" [style.width.px]="heightValue" [style.height.px]="heightValue">
        </div>
      </ng-container>
    </div>
  `,
  styles: [`
    .skeleton-wrapper {
      display: flex;
      flex-direction: column;
      gap: 12px;
      width: 100%;
    }

    .skeleton-item {
      position: relative;
      background-color: #e0e0e0;
      border-radius: 4px;
      overflow: hidden;
    }

    .skeleton-shimmer::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background: linear-gradient(
        90deg,
        transparent,
        rgba(255, 255, 255, 0.6),
        transparent
      );
      animation: shimmer 1.5s infinite;
    }

    .skeleton-card {
      border: 1px solid #e0e0e0;
      border-radius: 8px;
      padding: 16px;
      background-color: white;
    }

    .skeleton-card-header {
      margin-bottom: 12px;
    }

    .skeleton-table {
      border: 1px solid #e0e0e0;
      border-radius: 8px;
      overflow: hidden;
    }

    .skeleton-table-header {
      background-color: #f5f5f5;
      padding: 8px;
    }

    .skeleton-table-row {
      padding: 8px;
      border-top: 1px solid #e0e0e0;
    }

    .skeleton-circle {
      border-radius: 50%;
      position: relative;
      background-color: #e0e0e0;
      overflow: hidden;
    }

    @keyframes shimmer {
      0% {
        transform: translateX(-100%);
      }
      100% {
        transform: translateX(100%);
      }
    }

    /* Dark theme support */
    :host-context(.dark-theme) {
      .skeleton-item {
        background-color: #2c2c2c;
      }

      .skeleton-shimmer::before {
        background: linear-gradient(
          90deg,
          transparent,
          rgba(255, 255, 255, 0.1),
          transparent
        );
      }

      .skeleton-card {
        background-color: #1e1e1e;
        border-color: #333;
      }

      .skeleton-table {
        border-color: #333;
      }

      .skeleton-table-header {
        background-color: #252525;
      }

      .skeleton-table-row {
        border-color: #333;
      }

      .skeleton-circle {
        background-color: #2c2c2c;
      }
    }
  `]
})
export class LoadingSkeletonComponent {
  @Input() count = 5;
  @Input() height: number | string = 60;
  @Input() variant: SkeletonVariant = 'line';

  get items(): number[] {
    return Array(this.count).fill(0);
  }

  get heightValue(): number {
    if (typeof this.height === 'string') {
      return parseInt(this.height, 10) || 60;
    }
    return this.height;
  }
}
