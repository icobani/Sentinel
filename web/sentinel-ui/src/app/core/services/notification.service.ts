import { Injectable, inject } from '@angular/core';
import { MatSnackBar, MatSnackBarConfig } from '@angular/material/snack-bar';

@Injectable({
  providedIn: 'root'
})
export class NotificationService {
  private readonly snackBar = inject(MatSnackBar);

  private readonly defaultConfig: MatSnackBarConfig = {
    duration: 5000,
    horizontalPosition: 'end',
    verticalPosition: 'top'
  };

  success(message: string, duration?: number): void {
    this.snackBar.open(message, '✓', {
      ...this.defaultConfig,
      duration: duration ?? this.defaultConfig.duration,
      panelClass: ['snackbar-success']
    });
  }

  error(message: string, duration?: number): void {
    this.snackBar.open(message, '✕', {
      ...this.defaultConfig,
      duration: duration ?? this.defaultConfig.duration,
      panelClass: ['snackbar-error']
    });
  }

  warning(message: string, duration?: number): void {
    this.snackBar.open(message, '⚠', {
      ...this.defaultConfig,
      duration: duration ?? this.defaultConfig.duration,
      panelClass: ['snackbar-warning']
    });
  }

  info(message: string, duration?: number): void {
    this.snackBar.open(message, 'ⓘ', {
      ...this.defaultConfig,
      duration: duration ?? this.defaultConfig.duration,
      panelClass: ['snackbar-info']
    });
  }

  // Aliases for convenience
  showSuccess(message: string, duration?: number): void {
    this.success(message, duration);
  }

  showError(message: string, duration?: number): void {
    this.error(message, duration);
  }

  showWarning(message: string, duration?: number): void {
    this.warning(message, duration);
  }

  showInfo(message: string, duration?: number): void {
    this.info(message, duration);
  }
}
