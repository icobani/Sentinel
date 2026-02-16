import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { MatListModule } from '@angular/material/list';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { TranslatePipe } from '../../pipes/translate.pipe';

export interface MenuItem {
  labelKey: string;
  icon: string;
  route: string;
}

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [CommonModule, RouterModule, MatListModule, MatIconModule, MatTooltipModule, TranslatePipe],
  template: `
    <div class="sidebar" [class.collapsed]="collapsed">
      <div class="logo-section">
        <h1 class="logo">{{ collapsed ? 'S' : ('app.title' | translate) }}</h1>
      </div>

      <mat-nav-list class="menu-list">
        <a
          mat-list-item
          *ngFor="let item of menuItems"
          [routerLink]="item.route"
          routerLinkActive="active"
          [matTooltip]="collapsed ? (item.labelKey | translate) : ''"
          matTooltipPosition="right"
        >
          <mat-icon matListItemIcon>{{ item.icon }}</mat-icon>
          <span matListItemTitle *ngIf="!collapsed">{{ item.labelKey | translate }}</span>
        </a>
      </mat-nav-list>

      <div class="sidebar-footer">
        <div class="status-indicator" [class.online]="isOnline">
          <span class="status-dot"></span>
          <span *ngIf="!collapsed" class="status-text">
            {{ (isOnline ? 'status.online' : 'status.offline') | translate }}
          </span>
        </div>
        <div class="version-info" *ngIf="!collapsed && version">
          <span>v{{ version }}</span>
        </div>
      </div>

      <div class="powered-by" *ngIf="!collapsed">
        <span class="powered-label">Powered by</span>
        <span class="powered-company">Sentinel</span>
      </div>
    </div>
  `,
  styles: [`
    .sidebar {
      height: 100%;
      display: flex;
      flex-direction: column;
      background-color: #1e293b;
      color: white;
      transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1), transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      width: 260px;
      position: relative;
      z-index: 100;
    }

    .sidebar.collapsed {
      width: 72px;
    }

    .logo-section {
      padding: 24px 16px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    }

    .logo {
      font-size: 28px;
      font-weight: 700;
      margin: 0;
      text-align: center;
      white-space: nowrap;
      overflow: hidden;
      color: #f1f5f9;
    }

    .menu-list {
      flex: 1;
      padding: 16px 0;
    }

    .menu-list a {
      color: #e2e8f0;
      margin: 4px 8px;
      border-radius: 8px;
      transition: all 0.2s;
    }

    .menu-list a ::ng-deep .mdc-list-item__primary-text {
      color: #e2e8f0 !important;
      font-size: 15px;
      font-weight: 400;
    }

    .menu-list a ::ng-deep .mat-icon {
      color: #cbd5e1 !important;
    }

    .menu-list a:hover {
      background-color: rgba(255, 255, 255, 0.15);
      color: #ffffff;
    }

    .menu-list a:hover ::ng-deep .mdc-list-item__primary-text,
    .menu-list a:hover ::ng-deep .mat-icon {
      color: #ffffff !important;
    }

    .menu-list a.active {
      background-color: #3b82f6;
      color: #ffffff;
    }

    .menu-list a.active ::ng-deep .mdc-list-item__primary-text,
    .menu-list a.active ::ng-deep .mat-icon {
      color: #ffffff !important;
    }

    .sidebar-footer {
      padding: 16px;
      border-top: 1px solid rgba(255, 255, 255, 0.1);
    }

    .status-indicator {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
    }

    .status-dot {
      width: 10px;
      height: 10px;
      border-radius: 50%;
      background-color: #ef4444;
      flex-shrink: 0;
    }

    .status-indicator.online .status-dot {
      background-color: #22c55e;
    }

    .status-text {
      white-space: nowrap;
      color: #e2e8f0;
    }

    .version-info {
      margin-top: 4px;
      font-size: 11px;
      color: #64748b;
      padding-left: 18px;
    }

    .powered-by {
      padding: 12px 16px;
      text-align: center;
      border-top: 1px solid rgba(255, 255, 255, 0.05);
    }

    .powered-by {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 2px;
      opacity: 0.6;
    }

    .powered-label {
      font-size: 10px;
      color: #94a3b8;
      text-transform: uppercase;
      letter-spacing: 1px;
    }

    .powered-company {
      font-size: 13px;
      font-weight: 600;
      color: #e2e8f0;
    }

    // Responsive behavior
    @media (max-width: 1024px) {
      .sidebar {
        position: fixed;
        top: 64px; // Below toolbar
        left: 0;
        bottom: 0;
        box-shadow: 2px 0 8px rgba(0, 0, 0, 0.3);
        z-index: 1000;
        transform: translateX(0);
        transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      }

      .sidebar.collapsed {
        transform: translateX(-100%);
        width: 260px;
      }

      // Backdrop for mobile
      .sidebar:not(.collapsed)::before {
        content: '';
        position: fixed;
        top: 64px;
        left: 260px;
        right: 0;
        bottom: 0;
        background-color: rgba(0, 0, 0, 0.5);
        z-index: -1;
        animation: fadeIn 0.3s ease;
      }
    }

    @keyframes fadeIn {
      from {
        opacity: 0;
      }
      to {
        opacity: 1;
      }
    }

    @media (max-width: 768px) {
      .sidebar {
        width: 280px;
      }

      .logo-section {
        padding: 20px 16px;
      }

      .logo {
        font-size: 24px;
      }

      .menu-list a {
        font-size: 16px;
        padding: 12px 16px;
        min-height: 48px; // Touch-friendly
      }

      mat-icon {
        font-size: 24px;
        width: 24px;
        height: 24px;
      }
    }
  `]
})
export class SidebarComponent {
  @Input() collapsed = false;
  @Input() isOnline = true;
  @Input() version = '';
  @Input() menuItems: MenuItem[] = [
    { labelKey: 'menu.dashboard', icon: 'dashboard', route: '/dashboard' },
    { labelKey: 'menu.setup', icon: 'settings', route: '/setup' },
    { labelKey: 'menu.logs', icon: 'article', route: '/logs' }
  ];
  @Output() toggle = new EventEmitter<void>();
}
