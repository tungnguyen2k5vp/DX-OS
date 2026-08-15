import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable, signal } from '@angular/core';
import { tap } from 'rxjs';
import { APP_CONFIG } from '../config/app-config';
import { NotificationList } from './notification.models';

@Injectable({ providedIn: 'root' })
export class NotificationService {
  private readonly http = inject(HttpClient);
  private readonly config = inject(APP_CONFIG);
  private readonly unreadState = signal(0);

  readonly unreadCount = this.unreadState.asReadonly();

  list(page = 1, pageSize = 20, unreadOnly = false) {
    const params = new HttpParams()
      .set('page', page)
      .set('pageSize', pageSize)
      .set('unreadOnly', unreadOnly);
    return this.http
      .get<NotificationList>(`${this.config.apiBaseUrl}/api/v1/me/notifications`, { params })
      .pipe(tap((result) => this.unreadState.set(result.unreadCount)));
  }

  markRead(notificationId: string) {
    return this.http
      .post<{ read: boolean }>(
        `${this.config.apiBaseUrl}/api/v1/me/notifications/${notificationId}/read`,
        {},
      )
      .pipe(tap(() => this.unreadState.update((count) => Math.max(0, count - 1))));
  }

  markAllRead() {
    return this.http
      .post<{ markedRead: number }>(
        `${this.config.apiBaseUrl}/api/v1/me/notifications/read-all`,
        {},
      )
      .pipe(tap(() => this.unreadState.set(0)));
  }

  refreshUnreadCount(): void {
    this.list(1, 1, false).subscribe({ error: () => undefined });
  }
}
