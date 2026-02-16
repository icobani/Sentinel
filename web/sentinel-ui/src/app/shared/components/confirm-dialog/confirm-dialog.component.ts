import { Component, Inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatDialogModule, MAT_DIALOG_DATA, MatDialogRef } from '@angular/material/dialog';
import { MatButtonModule } from '@angular/material/button';
import { TranslatePipe } from '../../pipes/translate.pipe';
import { dialogScale } from '../../animations/dialog.animations';

export interface ConfirmDialogData {
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
}

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  imports: [CommonModule, MatDialogModule, MatButtonModule, TranslatePipe],
  animations: [dialogScale],
  template: `
    <div @dialogScale>
      <h2 mat-dialog-title>{{ data.title | translate }}</h2>
      <mat-dialog-content>
        <p>{{ data.message | translate }}</p>
      </mat-dialog-content>
      <mat-dialog-actions align="end">
        <button mat-button [mat-dialog-close]="false">
          {{ (data.cancelText || 'common.cancel') | translate }}
        </button>
        <button mat-raised-button color="warn" [mat-dialog-close]="true">
          {{ (data.confirmText || 'common.confirm') | translate }}
        </button>
      </mat-dialog-actions>
    </div>
  `,
  styles: [`
    mat-dialog-content {
      padding: 20px 0;
    }

    p {
      font-size: 16px;
      line-height: 1.6;
      margin: 0;
    }
  `]
})
export class ConfirmDialogComponent {
  constructor(
    public dialogRef: MatDialogRef<ConfirmDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: ConfirmDialogData
  ) {}
}
