import { Injectable, OnDestroy } from '@angular/core';
import { Subject, Observable, timer, Subscription } from 'rxjs';
import { WsMessage } from '../models/ws-message.model';
import { environment } from '../../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class WebSocketService implements OnDestroy {
  private ws: WebSocket | null = null;
  private messages$ = new Subject<WsMessage>();
  private connectionStatus$ = new Subject<boolean>();
  private reconnectAttempts = 0;
  private readonly maxReconnectAttempts = 10;
  private reconnectSubscription: Subscription | null = null;

  get messages(): Observable<WsMessage> {
    return this.messages$.asObservable();
  }

  get connectionStatus(): Observable<boolean> {
    return this.connectionStatus$.asObservable();
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = environment.wsUrl || `${protocol}//${location.host}/ws`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        this.connectionStatus$.next(true);
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WsMessage = JSON.parse(event.data);
          this.messages$.next(message);
        } catch (error) {
          // Invalid message format - ignore
        }
      };

      this.ws.onerror = () => {
        this.connectionStatus$.next(false);
      };

      this.ws.onclose = () => {
        this.connectionStatus$.next(false);
        this.reconnect();
      };
    } catch (error) {
      this.connectionStatus$.next(false);
      this.reconnect();
    }
  }

  private reconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      return;
    }

    // Clean up previous reconnect subscription if exists
    if (this.reconnectSubscription) {
      this.reconnectSubscription.unsubscribe();
      this.reconnectSubscription = null;
    }

    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);

    this.reconnectSubscription = timer(delay).subscribe(() => {
      this.connect();
    });
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  ngOnDestroy(): void {
    // Clean up reconnect subscription
    if (this.reconnectSubscription) {
      this.reconnectSubscription.unsubscribe();
      this.reconnectSubscription = null;
    }

    // Close WebSocket connection
    this.disconnect();

    // Complete all subjects
    this.messages$.complete();
    this.connectionStatus$.complete();
  }
}
