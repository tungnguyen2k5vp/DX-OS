import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../data-access/problem-details';
import { WorkSummary, WorkTaskType, WorkTaskUrgency } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';
import { PurchaseRequestStatusBadge } from '../../ui/purchase-request-status-badge';

const taskLabels: Record<WorkTaskType, string> = {
  COMPLETE_REQUEST: 'Hoàn thiện và gửi phiếu',
  MANAGER_REVIEW: 'Trưởng bộ phận xem xét',
  FINANCE_REVIEW: 'Tài chính thẩm định',
  SLA_MONITOR: 'Theo dõi SLA',
};

const urgencyLabels: Record<WorkTaskUrgency, string> = {
  NORMAL: 'Trong hạn',
  DUE_SOON: 'Sắp đến hạn',
  OVERDUE: 'Quá hạn',
};

@Component({
  selector: 'app-work-center',
  imports: [
    DatePipe,
    RouterLink,
    HlmBadge,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
    PurchaseRequestStatusBadge,
  ],
  templateUrl: './work-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkCenter {
  private readonly procurement = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);
  private generation = 0;

  readonly summary = signal<WorkSummary | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  constructor() {
    this.load();
  }

  refresh(): void {
    this.load();
  }

  taskLabel(type: WorkTaskType): string {
    return taskLabels[type];
  }

  urgencyLabel(urgency: WorkTaskUrgency): string {
    return urgencyLabels[urgency];
  }

  urgencyVariant(urgency: WorkTaskUrgency): 'destructive' | 'secondary' | 'outline' {
    if (urgency === 'OVERDUE') {
      return 'destructive';
    }
    return urgency === 'DUE_SOON' ? 'secondary' : 'outline';
  }

  private load(): void {
    const generation = ++this.generation;
    this.loading.set(true);
    this.error.set(null);
    this.procurement
      .taskSummary()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (summary) => {
          if (generation !== this.generation) {
            return;
          }
          this.summary.set(summary);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.generation) {
            return;
          }
          this.error.set(problemMessage(error, 'Không tải được danh sách công việc. Hãy thử lại.'));
          this.loading.set(false);
        },
      });
  }
}
