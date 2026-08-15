import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { NotificationList } from '../../../core/notifications/notification.models';
import { NotificationService } from '../../../core/notifications/notification.service';
import { problemMessage } from '../../procurement/data-access/problem-details';

@Component({
  selector: 'app-notification-center',
  imports: [DatePipe, RouterLink, HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './notification-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NotificationCenter {
  private readonly notifications = inject(NotificationService);
  private readonly destroyRef = inject(DestroyRef);

  readonly result = signal<NotificationList | null>(null);
  readonly page = signal(1);
  readonly unreadOnly = signal(false);
  readonly loading = signal(true);
  readonly saving = signal(false);
  readonly error = signal<string | null>(null);

  constructor() {
    this.load();
  }

  setUnreadOnly(value: boolean): void {
    this.unreadOnly.set(value);
    this.page.set(1);
    this.load();
  }

  goToPage(value: number): void {
    const pages = this.result()?.pages ?? 0;
    if (value < 1 || value > pages || value === this.page()) return;
    this.page.set(value);
    this.load();
  }

  markRead(id: string): void {
    this.saving.set(true);
    this.notifications
      .markRead(id)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: () => this.load(), error: (error) => this.fail(error) });
  }

  markAllRead(): void {
    this.saving.set(true);
    this.notifications
      .markAllRead()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({ next: () => this.load(), error: (error) => this.fail(error) });
  }

  resourceLink(resourceType: string, resourceId: string): string[] {
    if (resourceType === 'purchase_request') {
      return ['/purchase-requests', resourceId];
    }
    return ['/dashboard'];
  }

  private load(): void {
    this.loading.set(true);
    this.error.set(null);
    this.notifications
      .list(this.page(), 20, this.unreadOnly())
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.result.set(result);
          this.loading.set(false);
          this.saving.set(false);
        },
        error: (error) => this.fail(error),
      });
  }

  private fail(error: unknown): void {
    this.error.set(problemMessage(error, 'Không thể xử lý thông báo. Hãy thử lại.'));
    this.loading.set(false);
    this.saving.set(false);
  }
}
