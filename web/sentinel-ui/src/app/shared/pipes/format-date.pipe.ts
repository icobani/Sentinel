import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'formatDate',
  standalone: true
})
export class FormatDatePipe implements PipeTransform {
  transform(value: string | Date | null | undefined): string {
    if (!value) return '';

    const date = typeof value === 'string' ? new Date(value) : value;
    if (isNaN(date.getTime())) return '';

    const day = String(date.getDate()).padStart(2, '0');
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const year = date.getFullYear();
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');

    // Get timezone offset in hours (e.g., +3, -5)
    const offsetMinutes = -date.getTimezoneOffset();
    const offsetHours = Math.floor(offsetMinutes / 60);
    const offsetSign = offsetHours >= 0 ? '+' : '';

    return `${day}.${month}.${year} ${hours}:${minutes}:${seconds} ${offsetSign}${offsetHours}`;
  }
}
