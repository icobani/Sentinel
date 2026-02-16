import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { FileEvent } from '../models/event.model';
import { PaginatedResponse, ApiResponse } from '../models/api-response.model';
import { environment } from '../../../environments/environment';

export interface LogQueryParams {
  page?: number;
  limit?: number;
  watcher_id?: number;
  event_type?: string;
  search?: string;
  from?: string;
  to?: string;
}

@Injectable({
  providedIn: 'root'
})
export class EventLogService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = environment.apiBaseUrl;

  getLogs(params?: LogQueryParams): Observable<PaginatedResponse<FileEvent>> {
    let httpParams = new HttpParams();

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          httpParams = httpParams.set(key, value.toString());
        }
      });
    }

    return this.http.get<PaginatedResponse<FileEvent>>(`${this.baseUrl}/logs`, { params: httpParams });
  }

  getWatcherLogs(watcherId: number, params?: LogQueryParams): Observable<PaginatedResponse<FileEvent>> {
    let httpParams = new HttpParams();

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          httpParams = httpParams.set(key, value.toString());
        }
      });
    }

    return this.http.get<PaginatedResponse<FileEvent>>(
      `${this.baseUrl}/watchers/${watcherId}/logs`,
      { params: httpParams }
    );
  }

  exportLogs(watcherId: number, params?: LogQueryParams): Observable<Blob> {
    let httpParams = new HttpParams();

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
          httpParams = httpParams.set(key, value.toString());
        }
      });
    }

    return this.http.get(
      `${this.baseUrl}/watchers/${watcherId}/logs/export`,
      { params: httpParams, responseType: 'blob' }
    );
  }

  deleteLogs(ids: number[]): Observable<ApiResponse<{ deleted: number }>> {
    return this.http.delete<ApiResponse<{ deleted: number }>>(`${this.baseUrl}/logs`, {
      body: { ids }
    });
  }
}
