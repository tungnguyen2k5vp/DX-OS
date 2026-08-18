import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { CsvValue, downloadCsv } from '../../../../shared/utils/csv-export';
import { problemMessage } from '../../data-access/problem-details';
import { BudgetAllocation, BudgetDashboard } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';
import { AppIcon } from '../../../../shared/ui/app-icon/app-icon';

@Component({
  selector: 'app-budget-dashboard',
  imports: [
    DatePipe,
    ReactiveFormsModule,
    RouterLink,
    HlmBadge,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
    AppIcon,
  ],
  templateUrl: './budget-dashboard.html',
  styleUrl: './budget-dashboard.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BudgetDashboardPage {
  private readonly procurement = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly formBuilder = inject(FormBuilder);
  private loadGeneration = 0;

  readonly dashboard = signal<BudgetDashboard | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly adjustingId = signal<string | null>(null);
  readonly saving = signal(false);
  readonly saveError = signal<string | null>(null);
  readonly success = signal<string | null>(null);
  readonly allocationSearch = signal('');
  readonly filteredAllocations = computed(() => {
    const query = this.allocationSearch().trim().toLocaleLowerCase('vi-VN');
    const allocations = this.dashboard()?.allocations ?? [];
    if (!query) {
      return allocations;
    }
    return allocations.filter((allocation) =>
      [allocation.costCenter, allocation.periodCode, allocation.currency]
        .join(' ')
        .toLocaleLowerCase('vi-VN')
        .includes(query),
    );
  });

  readonly adjustmentForm = this.formBuilder.nonNullable.group({
    allocatedAmount: [
      '',
      [Validators.required, Validators.pattern(/^(0|[1-9][0-9]{0,14})(\.[0-9]{1,4})?$/)],
    ],
    reason: ['', [Validators.required, Validators.minLength(10), Validators.maxLength(1000)]],
  });

  constructor() {
    this.load();
  }

  retry(): void {
    this.load();
  }

  exportCsv(): void {
    const dashboard = this.dashboard();
    if (!dashboard) {
      return;
    }

    const rows: CsvValue[][] = [
      ['BÁO CÁO NGÂN SÁCH DX-OS'],
      ['Thời điểm xuất', new Date().toISOString()],
      ['Số cảnh báo', dashboard.alertCount],
      [],
      [
        'TRUNG TÂM CHI PHÍ',
        'Kỳ',
        'Tiền tệ',
        'Hạn mức',
        'Đang giữ',
        'Cam kết',
        'Khả dụng',
        'Sử dụng (%)',
        'Mức cảnh báo',
      ],
      ...dashboard.allocations.map((item) => [
        item.costCenter,
        item.periodCode,
        item.currency,
        item.allocatedAmount,
        item.reservedAmount,
        item.committedAmount,
        item.availableAmount,
        item.utilization,
        item.alertLevel,
      ]),
      [],
      ['KHOẢN GIỮ NGÂN SÁCH', 'Mã phiếu', 'Trung tâm chi phí', 'Tiền tệ', 'Số tiền', 'Trạng thái'],
      ...dashboard.reservations.map((item) => [
        item.requestTitle,
        item.requestCode,
        item.costCenter,
        item.currency,
        item.amount,
        item.status,
      ]),
      [],
      [
        'LỊCH SỬ ĐIỀU CHỈNH',
        'Trung tâm chi phí',
        'Tiền tệ',
        'Trước',
        'Sau',
        'Người thực hiện',
        'Lý do',
      ],
      ...dashboard.adjustments.map((item) => [
        item.createdAt,
        item.costCenter,
        item.currency,
        item.previousAmount,
        item.adjustedAmount,
        item.actorName,
        item.reason,
      ]),
    ];

    downloadCsv(`dx-os-budget-${dateInput()}.csv`, rows);
  }

  startAdjustment(allocation: BudgetAllocation): void {
    if (!this.dashboard()?.canManage) {
      return;
    }
    this.adjustingId.set(allocation.id);
    this.adjustmentForm.reset({
      allocatedAmount: allocation.allocatedAmount,
      reason: '',
    });
    this.saveError.set(null);
    this.success.set(null);
  }

  cancelAdjustment(): void {
    this.adjustingId.set(null);
    this.saveError.set(null);
  }

  async saveAdjustment(): Promise<void> {
    const allocation = this.dashboard()?.allocations.find((item) => item.id === this.adjustingId());
    if (!allocation || this.adjustmentForm.invalid) {
      this.adjustmentForm.markAllAsTouched();
      return;
    }

    this.saving.set(true);
    this.saveError.set(null);
    try {
      await firstValueFrom(
        this.procurement.adjustBudget(
          allocation.id,
          {
            allocatedAmount: this.adjustmentForm.controls.allocatedAmount.value,
            expectedVersion: allocation.version,
            reason: this.adjustmentForm.controls.reason.value.trim(),
          },
          crypto.randomUUID(),
        ),
      );
      this.adjustingId.set(null);
      this.success.set(`Đã cập nhật hạn mức cho ${allocation.costCenter}.`);
      this.load();
    } catch (error: unknown) {
      this.saveError.set(
        problemMessage(
          error,
          'Không điều chỉnh được hạn mức. Hãy tải lại bảng điều khiển và thử lại.',
        ),
      );
    } finally {
      this.saving.set(false);
    }
  }

  allocationProgress(allocation: BudgetAllocation): number {
    return Math.max(0, Math.min(100, Number(allocation.utilization)));
  }

  private load(): void {
    const generation = ++this.loadGeneration;
    this.loading.set(true);
    this.error.set(null);
    this.procurement
      .budgetDashboard()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (dashboard) => {
          if (generation !== this.loadGeneration) {
            return;
          }
          this.dashboard.set(dashboard);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.loadGeneration) {
            return;
          }
          this.error.set(problemMessage(error, 'Không tải được bảng điều khiển ngân sách.'));
          this.loading.set(false);
        },
      });
  }
}

function dateInput(): string {
  const date = new Date();
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
