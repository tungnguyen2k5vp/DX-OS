import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import { AuditCenter as AuditCenterModel } from '../../data-access/reporting.models';
import { ReportingService } from '../../data-access/reporting.service';

@Component({
  selector: 'app-audit-center',
  imports: [DatePipe, HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './audit-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuditCenter {
  private readonly reporting = inject(ReportingService);
  private readonly destroyRef = inject(DestroyRef);
  private generation = 0;

  readonly result = signal<AuditCenterModel | null>(null);
  readonly page = signal(1);
  readonly pageSize = 20;
  readonly resourceType = signal('');
  readonly action = signal('');
  readonly from = signal('');
  readonly to = signal('');
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  constructor() {
    this.load();
  }

  applyFilters(): void {
    this.page.set(1);
    this.load();
  }

  clearFilters(): void {
    this.resourceType.set('');
    this.action.set('');
    this.from.set('');
    this.to.set('');
    this.applyFilters();
  }

  goToPage(target: number): void {
    const pages = this.result()?.pages ?? 0;
    if (target < 1 || target > pages || target === this.page()) return;
    this.page.set(target);
    this.load();
  }

  resourceLabel(type: string): string {
    return (
      {
        purchase_request: 'Phiếu mua sắm',
        supplier: 'Nhà cung cấp',
        purchase_order: 'Đơn hàng',
        budget_allocation: 'Ngân sách',
      }[type] ?? type
    );
  }

  private load(): void {
    const generation = ++this.generation;
    this.loading.set(true);
    this.error.set(null);
    this.reporting
      .auditEvents({
        page: this.page(),
        pageSize: this.pageSize,
        resourceType: this.resourceType() || undefined,
        action: this.action() || undefined,
        from: this.from() || undefined,
        to: this.to() || undefined,
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.generation) return;
          this.result.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.generation) return;
          this.error.set(problemMessage(error, 'Không tải được bằng chứng kiểm toán.'));
          this.loading.set(false);
        },
      });
  }
}
