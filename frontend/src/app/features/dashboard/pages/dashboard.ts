import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { AuthService } from '../../../core/auth/auth.service';
import { CurrentUser, IdentityService } from '../../../core/http/identity.service';
import { ProcurementService } from '../../procurement/data-access/procurement.service';

@Component({
  selector: 'app-dashboard',
  imports: [RouterLink, ...HlmCardImports, HlmBadge],
  templateUrl: './dashboard.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Dashboard {
  private readonly identity = inject(IdentityService);
  private readonly procurement = inject(ProcurementService);
  private readonly auth = inject(AuthService);

  readonly currentUser = signal<CurrentUser | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly pendingApprovals = signal<number | null>(null);
  readonly canReview = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('department_manager') || roles.includes('finance');
  });

  constructor() {
    this.identity
      .getCurrentUser()
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: (user) => {
          this.currentUser.set(user);
          this.loading.set(false);
        },
        error: () => {
          this.error.set('Không gọi được Go API. Hãy kiểm tra container api và đăng nhập lại.');
          this.loading.set(false);
        },
      });

    if (this.canReview()) {
      this.procurement
        .list({
          page: 1,
          pageSize: 1,
          status: this.auth.roles().includes('finance') ? 'MANAGER_APPROVED' : 'SUBMITTED',
        })
        .pipe(takeUntilDestroyed())
        .subscribe({
          next: (result) => this.pendingApprovals.set(result.total),
          error: () => this.pendingApprovals.set(null),
        });
    }
  }
}
