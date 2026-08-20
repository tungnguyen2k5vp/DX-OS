import { DatePipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { AuthService } from '../../../../core/auth/auth.service';
import { problemMessage } from '../../data-access/problem-details';
import { PurchaseRequest, PurchaseRequestStatus } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';
import { PurchaseRequestStatusBadge } from '../../ui/purchase-request-status-badge';
import { AppIcon } from '../../../../shared/ui/app-icon/app-icon';

@Component({
  selector: 'app-approval-inbox',
  imports: [
    DatePipe,
    RouterLink,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
    PurchaseRequestStatusBadge,
    AppIcon,
  ],
  templateUrl: './approval-inbox.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ApprovalInbox {
  private readonly procurement = inject(ProcurementService);
  private readonly auth = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);
  private requestGeneration = 0;

  readonly page = signal(1);
  readonly pageSize = 20;
  readonly items = signal<PurchaseRequest[]>([]);
  readonly total = signal(0);
  readonly pages = signal(0);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly batchMessage = signal<string | null>(null);
  readonly selectedIds = signal<Set<string>>(new Set());
  readonly batchBusy = signal(false);
  readonly selectedCount = computed(() => this.selectedIds().size);
  readonly reviewStatus = computed<PurchaseRequestStatus>(() =>
    this.auth.roles().includes('finance') ? 'MANAGER_APPROVED' : 'SUBMITTED',
  );
  readonly reviewRoleLabel = computed(() =>
    this.reviewStatus() === 'MANAGER_APPROVED' ? 'Tài chính' : 'Trưởng bộ phận',
  );
  readonly scopeDescription = computed(() =>
    this.reviewStatus() === 'MANAGER_APPROVED'
      ? 'Phiếu đã qua vòng trưởng bộ phận trong cùng tổ chức.'
      : 'Phiếu đã gửi thuộc cùng phòng ban với bạn.',
  );

  constructor() {
    this.load();
  }

  retry(): void {
    this.load();
  }

  goToPage(target: number): void {
    if (target < 1 || target > this.pages() || target === this.page()) {
      return;
    }
    this.page.set(target);
    this.load();
  }

  toggleSelection(requestId: string, selected: boolean): void {
    const next = new Set(this.selectedIds());
    if (selected) next.add(requestId);
    else next.delete(requestId);
    this.selectedIds.set(next);
  }

  toggleAll(selected: boolean): void {
    this.selectedIds.set(selected ? new Set(this.items().map((item) => item.id)) : new Set());
  }

  async approveSelected(): Promise<void> {
    const selected = this.items().filter((item) => this.selectedIds().has(item.id));
    if (selected.length === 0) return;
    if (!confirm(`Phê duyệt ${selected.length} phiếu đã chọn?`)) return;
    this.batchBusy.set(true);
    this.error.set(null);
    this.batchMessage.set(null);
    const results = await Promise.allSettled(
      selected.map((request) =>
        new Promise<void>((resolve, reject) => {
          this.procurement
            .transition(
              request.id,
              { action: 'APPROVE', expectedVersion: request.version, comment: 'Phê duyệt hàng loạt sau khi đã kiểm tra danh sách' },
              crypto.randomUUID(),
            )
            .subscribe({ next: () => resolve(), error: reject });
        }),
      ),
    );
    const succeeded = results.filter((item) => item.status === 'fulfilled').length;
    const failed = results.length - succeeded;
    this.batchMessage.set(`Đã phê duyệt ${succeeded}/${results.length} phiếu.${failed ? ` ${failed} phiếu cần mở lại để kiểm tra.` : ''}`);
    this.selectedIds.set(new Set());
    this.batchBusy.set(false);
    this.load();
  }

  private load(): void {
    const generation = ++this.requestGeneration;
    this.loading.set(true);
    this.error.set(null);
    this.procurement
      .list({
        page: this.page(),
        pageSize: this.pageSize,
        status: this.reviewStatus(),
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.requestGeneration) {
            return;
          }
          this.items.set(result.items);
          this.selectedIds.set(new Set());
          this.total.set(result.total);
          this.pages.set(result.pages);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.requestGeneration) {
            return;
          }
          this.error.set(
            problemMessage(error, 'Không tải được danh sách chờ phê duyệt. Hãy thử lại.'),
          );
          this.loading.set(false);
        },
      });
  }
}
