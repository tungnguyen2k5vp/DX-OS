import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { AuthService } from './core/auth/auth.service';
import { NotificationService } from './core/notifications/notification.service';

const roleLabels: Record<string, string> = {
  employee: 'Nhân viên',
  department_manager: 'Trưởng bộ phận',
  finance: 'Tài chính',
  dx_admin: 'Quản trị DX-OS',
  auditor: 'Kiểm toán',
  ai_operator: 'Điều phối AI',
};

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, HlmButton, HlmBadge],
  templateUrl: './app.html',
  styleUrl: './app.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class App {
  readonly auth = inject(AuthService);
  readonly notifications = inject(NotificationService);
  readonly isEmployee = computed(() => this.auth.roles().includes('employee'));
  readonly canReview = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('department_manager') || roles.includes('finance');
  });
  readonly canAccessBudgets = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('finance') || roles.includes('auditor');
  });
  readonly canAccessInvoices = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('finance') || roles.includes('auditor');
  });
  readonly canAccessReports = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('finance') || roles.includes('auditor') || roles.includes('dx_admin');
  });
  readonly canAccessOperations = computed(() => {
    const roles = this.auth.roles();
    return ['employee', 'department_manager', 'finance', 'auditor'].some((role) =>
      roles.includes(role),
    );
  });
  readonly canAccessSuppliers = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('finance') || roles.includes('auditor');
  });
  readonly canAccessAudit = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('auditor') || roles.includes('dx_admin');
  });
  readonly canAccessPolicies = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('dx_admin') || roles.includes('auditor');
  });

  readonly primaryRoleLabel = computed(() => {
    const role = this.auth.roles().find((candidate) => roleLabels[candidate]);
    return role ? roleLabels[role] : 'Đã xác thực';
  });

  constructor() {
    this.notifications.refreshUnreadCount();
  }

  logout(): void {
    void this.auth.logout();
  }
}
