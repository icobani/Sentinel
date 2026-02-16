import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { Watcher, WatcherStats } from '../models/watcher.model';
import { ApiResponse } from '../models/api-response.model';
import { environment } from '../../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class WatcherService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiBaseUrl}/watchers`;

  getWatchers(): Observable<ApiResponse<Watcher[]>> {
    return this.http.get<ApiResponse<Watcher[]>>(this.baseUrl);
  }

  getWatcher(id: number): Observable<ApiResponse<Watcher>> {
    return this.http.get<ApiResponse<Watcher>>(`${this.baseUrl}/${id}`);
  }

  createWatcher(watcher: Partial<Watcher>): Observable<ApiResponse<Watcher>> {
    return this.http.post<ApiResponse<Watcher>>(this.baseUrl, watcher);
  }

  updateWatcher(id: number, watcher: Partial<Watcher>): Observable<ApiResponse<Watcher>> {
    return this.http.put<ApiResponse<Watcher>>(`${this.baseUrl}/${id}`, watcher);
  }

  deleteWatcher(id: number): Observable<ApiResponse<void>> {
    return this.http.delete<ApiResponse<void>>(`${this.baseUrl}/${id}`);
  }

  startWatcher(id: number): Observable<ApiResponse<void>> {
    return this.http.post<ApiResponse<void>>(`${this.baseUrl}/${id}/start`, {});
  }

  stopWatcher(id: number): Observable<ApiResponse<void>> {
    return this.http.post<ApiResponse<void>>(`${this.baseUrl}/${id}/stop`, {});
  }

  restartWatcher(id: number): Observable<ApiResponse<void>> {
    return this.http.post<ApiResponse<void>>(`${this.baseUrl}/${id}/restart`, {});
  }

  validatePath(path: string): Observable<ApiResponse<{ valid: boolean; message: string }>> {
    return this.http.post<ApiResponse<{ valid: boolean; message: string }>>(
      `${this.baseUrl}/validate-path`,
      { path }
    );
  }

  testWebhook(id: number): Observable<ApiResponse<{ status: number; response_time: number; error?: string }>> {
    return this.http.post<ApiResponse<{ status: number; response_time: number; error?: string }>>(
      `${this.baseUrl}/${id}/test`,
      {}
    );
  }

  getStats(id: number): Observable<ApiResponse<WatcherStats>> {
    return this.http.get<ApiResponse<WatcherStats>>(`${this.baseUrl}/${id}/stats`);
  }
}
