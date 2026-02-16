import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse, SystemStatus } from '../models/api-response.model';
import { environment } from '../../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class StatusService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = environment.apiBaseUrl;

  getSystemStatus(): Observable<ApiResponse<SystemStatus>> {
    return this.http.get<ApiResponse<SystemStatus>>(`${this.baseUrl}/status`);
  }

  restartService(): Observable<ApiResponse<void>> {
    return this.http.post<ApiResponse<void>>(`${this.baseUrl}/service/restart`, {});
  }

  checkPendingRestart(): Observable<ApiResponse<{ pending: boolean }>> {
    return this.http.get<ApiResponse<{ pending: boolean }>>(`${this.baseUrl}/config/pending`);
  }
}
