import { Component, Output, EventEmitter, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatTooltipModule } from '@angular/material/tooltip';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';

@Component({
  selector: 'app-toolbar',
  standalone: true,
  imports: [
    CommonModule,
    MatToolbarModule,
    MatButtonModule,
    MatIconModule,
    MatMenuModule,
    MatTooltipModule,
    MatSlideToggleModule
  ],
  template: `
    <mat-toolbar class="toolbar">
      <button mat-icon-button (click)="toggleSidebar.emit()" matTooltip="Toggle Menu">
        <mat-icon>menu</mat-icon>
      </button>

      <span class="spacer"></span>

      <button mat-icon-button [matMenuTriggerFor]="languageMenu" matTooltip="Language">
        <mat-icon>translate</mat-icon>
      </button>
      <mat-menu #languageMenu="matMenu">
        <button mat-menu-item (click)="changeLanguage('tr')">
          <span class="flag">🇹🇷</span>
          <span>Türkçe</span>
        </button>
        <button mat-menu-item (click)="changeLanguage('en')">
          <span class="flag">🇬🇧</span>
          <span>English</span>
        </button>
      </mat-menu>

      <button
        mat-icon-button
        (click)="toggleTheme()"
        [matTooltip]="isDarkTheme() ? 'Light Mode' : 'Dark Mode'"
      >
        <mat-icon>{{ isDarkTheme() ? 'light_mode' : 'dark_mode' }}</mat-icon>
      </button>

      <button mat-icon-button matTooltip="Notifications">
        <mat-icon>notifications</mat-icon>
      </button>
    </mat-toolbar>
  `,
  styles: [`
    .toolbar {
      background-color: white;
      color: rgba(0, 0, 0, 0.87);
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
      z-index: 10;
    }

    .spacer {
      flex: 1;
    }

    .flag {
      margin-right: 8px;
      font-size: 20px;
    }

    button {
      margin: 0 4px;
    }
  `]
})
export class ToolbarComponent {
  @Output() toggleSidebar = new EventEmitter<void>();
  @Output() languageChange = new EventEmitter<string>();
  @Output() themeChange = new EventEmitter<boolean>();

  isDarkTheme = signal(false);

  changeLanguage(lang: string): void {
    this.languageChange.emit(lang);
  }

  toggleTheme(): void {
    this.isDarkTheme.update(value => !value);
    this.themeChange.emit(this.isDarkTheme());
  }
}
