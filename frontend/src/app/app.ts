import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { AuthService } from './core/auth/auth.service';
import { navigationForRoles, primaryRoleLabel } from './core/navigation/navigation.model';
import { NotificationService } from './core/notifications/notification.service';
import { AppIcon } from './shared/ui/app-icon/app-icon';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, HlmButton, HlmBadge, AppIcon],
  templateUrl: './app.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class App {
  readonly auth = inject(AuthService);
  readonly notifications = inject(NotificationService);
  readonly isNavigationOpen = signal(false);
  readonly navigationGroups = computed(() => navigationForRoles(this.auth.roles()));
  readonly primaryRoleLabel = computed(() => primaryRoleLabel(this.auth.roles()));
  readonly userInitials = computed(() =>
    this.auth
      .username()
      .split(/[._\-\s]+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part[0]?.toUpperCase())
      .join('') || 'DX',
  );

  constructor() {
    this.notifications.refreshUnreadCount();
  }

  toggleNavigation(): void {
    this.isNavigationOpen.update((isOpen) => !isOpen);
  }

  closeNavigation(): void {
    this.isNavigationOpen.set(false);
  }

  logout(): void {
    void this.auth.logout();
  }
}
