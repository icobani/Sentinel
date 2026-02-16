import { Pipe, PipeTransform, inject } from '@angular/core';
import { TranslateService } from '../../core/services/translate.service';

@Pipe({
  name: 'translate',
  standalone: true,
  pure: false // Impure to re-evaluate on every CD cycle (language signal drives CD via zoneless scheduler)
})
export class TranslatePipe implements PipeTransform {
  private translateService = inject(TranslateService);

  transform(key: string, params?: Record<string, string>): string {
    return this.translateService.get(key, params);
  }
}
