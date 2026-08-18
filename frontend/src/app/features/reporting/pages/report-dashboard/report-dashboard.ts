import { DatePipe, DecimalPipe, PercentPipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { APP_CONFIG } from '../../../../core/config/app-config';
import { CsvValue, downloadCsv } from '../../../../shared/utils/csv-export';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import { MoneyPipe } from '../../../procurement/ui/money.pipe';
import {
  ProcurementReportDashboard,
  ReportStatusBreakdown,
} from '../../data-access/reporting.models';
import { ReportingService } from '../../data-access/reporting.service';

const statusLabels: Record<string, string> = {
  DRAFT: 'Bản nháp',
  SUBMITTED: 'Chờ trưởng bộ phận',
  MANAGER_APPROVED: 'Chờ tài chính',
  CHANGES_REQUESTED: 'Yêu cầu chỉnh sửa',
  APPROVED: 'Đã phê duyệt',
  REJECTED: 'Đã từ chối',
  CANCELLED: 'Đã hủy',
};

interface StatusView extends ReportStatusBreakdown {
  label: string;
  width: number;
}

@Component({
  selector: 'app-report-dashboard',
  imports: [
    DatePipe,
    DecimalPipe,
    PercentPipe,
    ReactiveFormsModule,
    HlmBadge,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
  ],
  templateUrl: './report-dashboard.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ReportDashboardPage {
  private readonly reporting = inject(ReportingService);
  private readonly config = inject(APP_CONFIG);
  private readonly formBuilder = inject(FormBuilder);
  private readonly destroyRef = inject(DestroyRef);
  private loadGeneration = 0;

  readonly metabaseUrl = this.config.metabaseUrl;
  readonly dashboard = signal<ProcurementReportDashboard | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly filters = this.formBuilder.nonNullable.group({
    from: [dateInput(-29), Validators.required],
    to: [dateInput(0), Validators.required],
    costCenter: [''],
    currency: [''],
  });
  readonly statusViews = computed<StatusView[]>(() => {
    const statuses = this.dashboard()?.statuses ?? [];
    const maximum = Math.max(1, ...statuses.map((item) => item.requestCount));
    return statuses.map((item) => ({
      ...item,
      label: statusLabels[item.status] ?? item.status,
      width: Math.max(4, Math.round((item.requestCount / maximum) * 100)),
    }));
  });
  readonly approvalRate = computed(() => {
    const summary = this.dashboard()?.summary;
    if (!summary || summary.totalRequests === 0) {
      return 0;
    }
    return summary.approvedCount / summary.totalRequests;
  });
  readonly returnRate = computed(() => this.summaryRate('returnedCount'));
  readonly rejectionRate = computed(() => this.summaryRate('rejectedCount'));

  constructor() {
    this.load();
  }

  applyFilters(): void {
    if (this.filters.invalid) {
      this.filters.markAllAsTouched();
      return;
    }
    this.load();
  }

  resetFilters(): void {
    this.filters.reset({
      from: dateInput(-29),
      to: dateInput(0),
      costCenter: '',
      currency: '',
    });
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
      ['BÁO CÁO VẬN HÀNH MUA SẮM DX-OS'],
      ['Từ ngày', dashboard.filters.from],
      ['Đến ngày', dashboard.filters.to],
      ['Trung tâm chi phí', dashboard.filters.costCenter ?? 'Tất cả'],
      ['Tiền tệ', dashboard.filters.currency ?? 'Tất cả'],
      [],
      ['KPI', 'Giá trị'],
      ['Tổng số phiếu', dashboard.summary.totalRequests],
      ['Đã phê duyệt', dashboard.summary.approvedCount],
      ['Bị từ chối', dashboard.summary.rejectedCount],
      ['Yêu cầu chỉnh sửa', dashboard.summary.returnedCount],
      ['Quá SLA', dashboard.summary.slaBreachedCount],
      ['Thời gian xử lý trung bình (giờ)', dashboard.summary.averageLeadTimeHours],
      ['Tỷ lệ tuân thủ tài liệu (%)', dashboard.summary.attachmentComplianceRate],
      [],
      ['TRẠNG THÁI', 'Tiền tệ', 'Số phiếu', 'Tổng giá trị'],
      ...dashboard.statuses.map((item) => [
        statusLabels[item.status] ?? item.status,
        item.currency,
        item.requestCount,
        item.totalAmount,
      ]),
      [],
      ['PHÒNG BAN', 'Tiền tệ', 'Số phiếu', 'Đã duyệt', 'Tổng giá trị'],
      ...dashboard.departments.map((item) => [
        item.departmentName,
        item.currency,
        item.requestCount,
        item.approvedCount,
        item.totalAmount,
      ]),
      [],
      ['NGÂN SÁCH', 'Kỳ', 'Tiền tệ', 'Hạn mức', 'Đang giữ', 'Cam kết', 'Khả dụng', 'Sử dụng (%)'],
      ...dashboard.budgets.map((item) => [
        item.costCenter,
        item.periodCode,
        item.currency,
        item.allocatedAmount,
        item.reservedAmount,
        item.committedAmount,
        item.availableAmount,
        item.utilizationPercent,
      ]),
    ];

    downloadCsv(`dx-os-procurement-${dashboard.filters.from}-${dashboard.filters.to}.csv`, rows);
  }

  private load(): void {
    const generation = ++this.loadGeneration;
    const values = this.filters.getRawValue();
    this.loading.set(true);
    this.error.set(null);
    this.reporting
      .procurementDashboard({
        from: values.from,
        to: values.to,
        costCenter: values.costCenter.trim() || undefined,
        currency: values.currency.trim().toUpperCase() || undefined,
      })
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
          this.error.set(problemMessage(error, 'Không tải được báo cáo vận hành.'));
          this.loading.set(false);
        },
      });
  }

  private summaryRate(field: 'returnedCount' | 'rejectedCount'): number {
    const summary = this.dashboard()?.summary;
    if (!summary || summary.totalRequests === 0) {
      return 0;
    }
    return summary[field] / summary.totalRequests;
  }
}

function dateInput(offsetDays: number): string {
  const date = new Date();
  date.setDate(date.getDate() + offsetDays);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
