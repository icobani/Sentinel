import { Component, OnInit, inject, signal, DestroyRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormBuilder, FormGroup, FormArray, Validators, ReactiveFormsModule } from '@angular/forms';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { MatChipsModule, MatChipInputEvent } from '@angular/material/chips';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatRadioModule } from '@angular/material/radio';
import { COMMA, ENTER } from '@angular/cdk/keycodes';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { finalize } from 'rxjs';

import { WatcherService } from '../../../core/services/watcher.service';
import { NotificationService } from '../../../core/services/notification.service';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';
import { Watcher, WatchMode } from '../../../core/models/watcher.model';

interface WebhookHeader {
  key: string;
  value: string;
}

@Component({
  selector: 'app-watcher-form',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSlideToggleModule,
    MatChipsModule,
    MatProgressSpinnerModule,
    MatRadioModule,
    TranslatePipe
  ],
  templateUrl: './watcher-form.component.html',
  styleUrls: ['./watcher-form.component.scss']
})
export class WatcherFormComponent implements OnInit {
  private readonly destroyRef = inject(DestroyRef);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly fb = inject(FormBuilder);
  private readonly watcherService = inject(WatcherService);
  private readonly notificationService = inject(NotificationService);

  readonly loading = signal(false);
  readonly saving = signal(false);
  readonly validatingPath = signal(false);
  readonly pathValidation = signal<{ valid: boolean; message: string } | null>(null);
  readonly testingWebhook = signal(false);
  readonly webhookTestResult = signal<{ status: number; response_time: number; error?: string } | null>(null);

  form!: FormGroup;
  isEditMode = false;
  watcherId: number | null = null;

  readonly WatchMode = WatchMode;
  readonly separatorKeysCodes: number[] = [ENTER, COMMA];

  ngOnInit(): void {
    this.initForm();

    this.route.params
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(params => {
        if (params['id']) {
          this.watcherId = +params['id'];
          this.isEditMode = true;
          this.loadWatcher();
        }
      });

    this.form.get('mode')?.valueChanges
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(mode => {
        const pollIntervalControl = this.form.get('poll_interval');
        if (mode === WatchMode.POLL) {
          pollIntervalControl?.setValidators([Validators.required]);
        } else {
          pollIntervalControl?.clearValidators();
        }
        pollIntervalControl?.updateValueAndValidity();
      });
  }

  private initForm(): void {
    this.form = this.fb.group({
      name: ['', Validators.required],
      path: ['', Validators.required],
      mode: [WatchMode.WATCH, Validators.required],
      recursive: [true],
      poll_interval: ['5s'],
      include_filters: [[]],
      exclude_filters: [[]],
      webhook_url: [''],
      webhook_timeout: ['10s'],
      webhook_headers: this.fb.array([]),
      max_attempts: [3, [Validators.required, Validators.min(1)]],
      backoff: ['2s', Validators.required]
    });
  }

  private loadWatcher(): void {
    if (!this.watcherId) return;

    this.loading.set(true);
    this.watcherService.getWatcher(this.watcherId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.loading.set(false))
      )
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            this.populateForm(response.data);
          }
        },
        error: () => {
          this.notificationService.showError('Failed to load watcher');
          this.router.navigate(['/setup']);
        }
      });
  }

  private populateForm(watcher: Watcher): void {
    this.form.patchValue({
      name: watcher.name,
      path: watcher.path,
      mode: watcher.mode,
      recursive: watcher.recursive,
      poll_interval: watcher.poll_interval || '5s',
      include_filters: watcher.filters?.include || [],
      exclude_filters: watcher.filters?.exclude || [],
      webhook_url: watcher.webhook?.url || '',
      webhook_timeout: watcher.webhook?.timeout || '10s',
      max_attempts: watcher.webhook?.retry?.max_attempts || 3,
      backoff: watcher.webhook?.retry?.backoff || '2s'
    });

    if (watcher.webhook?.headers) {
      const headersArray = this.webhookHeaders;
      headersArray.clear();
      Object.entries(watcher.webhook.headers).forEach(([key, value]) => {
        headersArray.push(this.fb.group({
          key: [key, Validators.required],
          value: [value, Validators.required]
        }));
      });
    }
  }

  get webhookHeaders(): FormArray {
    return this.form.get('webhook_headers') as FormArray;
  }

  get isPollMode(): boolean {
    return this.form.get('mode')?.value === WatchMode.POLL;
  }

  get hasWebhook(): boolean {
    return !!this.form.get('webhook_url')?.value;
  }

  addIncludeFilter(event: MatChipInputEvent): void {
    const value = (event.value || '').trim();
    if (value) {
      const current = this.form.get('include_filters')?.value || [];
      this.form.get('include_filters')?.setValue([...current, value]);
      event.chipInput!.clear();
    }
  }

  removeIncludeFilter(filter: string): void {
    const current = this.form.get('include_filters')?.value || [];
    const index = current.indexOf(filter);
    if (index >= 0) {
      current.splice(index, 1);
      this.form.get('include_filters')?.setValue([...current]);
    }
  }

  addExcludeFilter(event: MatChipInputEvent): void {
    const value = (event.value || '').trim();
    if (value) {
      const current = this.form.get('exclude_filters')?.value || [];
      this.form.get('exclude_filters')?.setValue([...current, value]);
      event.chipInput!.clear();
    }
  }

  removeExcludeFilter(filter: string): void {
    const current = this.form.get('exclude_filters')?.value || [];
    const index = current.indexOf(filter);
    if (index >= 0) {
      current.splice(index, 1);
      this.form.get('exclude_filters')?.setValue([...current]);
    }
  }

  addHeader(): void {
    this.webhookHeaders.push(this.fb.group({
      key: ['', Validators.required],
      value: ['', Validators.required]
    }));
  }

  removeHeader(index: number): void {
    this.webhookHeaders.removeAt(index);
  }

  onValidatePath(): void {
    const path = this.form.get('path')?.value;
    if (!path) {
      this.notificationService.showWarning('Please enter a directory path first');
      return;
    }

    this.validatingPath.set(true);
    this.pathValidation.set(null);

    this.watcherService.validatePath(path)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.validatingPath.set(false))
      )
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            this.pathValidation.set(response.data);
          }
        },
        error: () => {
          this.notificationService.showError('Failed to validate path');
        }
      });
  }

  onTestWebhook(): void {
    if (!this.watcherId) {
      this.notificationService.showWarning('Please save the watcher first before testing webhook');
      return;
    }

    this.testingWebhook.set(true);
    this.webhookTestResult.set(null);

    this.watcherService.testWebhook(this.watcherId)
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.testingWebhook.set(false))
      )
      .subscribe({
        next: (response) => {
          if (response.success && response.data) {
            this.webhookTestResult.set(response.data);
            if (response.data.status === 200) {
              this.notificationService.showSuccess('setup.webhook_test_success');
            } else {
              this.notificationService.showWarning('setup.webhook_test_failed');
            }
          }
        },
        error: () => {
          this.notificationService.showError('Failed to test webhook');
        }
      });
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      this.notificationService.showWarning('Please fill in all required fields');
      return;
    }

    const formValue = this.form.value;
    const watcherData: Partial<Watcher> = {
      name: formValue.name,
      path: formValue.path,
      mode: formValue.mode,
      recursive: formValue.recursive,
      poll_interval: formValue.mode === WatchMode.POLL ? formValue.poll_interval : undefined,
      filters: {
        include: formValue.include_filters.length > 0 ? formValue.include_filters : undefined,
        exclude: formValue.exclude_filters.length > 0 ? formValue.exclude_filters : undefined
      }
    };

    if (formValue.webhook_url) {
      const headers: Record<string, string> = {};
      formValue.webhook_headers.forEach((h: WebhookHeader) => {
        if (h.key && h.value) {
          headers[h.key] = h.value;
        }
      });

      watcherData.webhook = {
        url: formValue.webhook_url,
        headers: Object.keys(headers).length > 0 ? headers : undefined,
        timeout: formValue.webhook_timeout,
        retry: {
          max_attempts: formValue.max_attempts,
          backoff: formValue.backoff
        }
      };
    }

    this.saving.set(true);

    const request = this.isEditMode && this.watcherId
      ? this.watcherService.updateWatcher(this.watcherId, watcherData)
      : this.watcherService.createWatcher(watcherData);

    request
      .pipe(
        takeUntilDestroyed(this.destroyRef),
        finalize(() => this.saving.set(false))
      )
      .subscribe({
        next: () => {
          const message = this.isEditMode ? 'setup.watcher_updated' : 'setup.watcher_created';
          this.notificationService.showSuccess(message);
          this.notificationService.showInfo('setup.restart_required');
          this.router.navigate(['/setup']);
        },
        error: () => {
          const message = this.isEditMode ? 'Failed to update watcher' : 'Failed to create watcher';
          this.notificationService.showError(message);
        }
      });
  }

  onCancel(): void {
    this.router.navigate(['/setup']);
  }
}
