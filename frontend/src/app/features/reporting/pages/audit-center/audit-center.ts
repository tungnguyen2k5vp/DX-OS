import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import {
  AuditCase,
  AuditCaseList,
  AuditCaseSeverity,
  AuditCaseStatus,
  AuditCenter as AuditCenterModel,
  SaveAuditCase,
} from '../../data-access/reporting.models';
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
  readonly cases = signal<AuditCaseList | null>(null);
  readonly caseBusy = signal(false);
  readonly caseTitle = signal('');
  readonly caseDescription = signal('');
  readonly caseSeverity = signal<AuditCaseSeverity>('MEDIUM');
  readonly caseDueOn = signal('');
  readonly caseResourceId = signal('');
  readonly evidenceRequestId = signal('');
  readonly resolutionDrafts = signal<Record<string, string>>({});

  constructor() {
    this.load();
    this.loadCases();
  }

  createCase(): void {
    const title = this.caseTitle().trim();
    const description = this.caseDescription().trim();
    if (title.length < 3 || description.length < 10) {
      this.error.set('Tiêu đề cần ít nhất 3 ký tự và mô tả cần ít nhất 10 ký tự.');
      return;
    }
    this.caseBusy.set(true);
    this.reporting
      .createAuditCase({
        title,
        description,
        severity: this.caseSeverity(),
        status: 'OPEN',
        resourceType: this.caseResourceId().trim() ? 'purchase_request' : '',
        resourceId: this.caseResourceId().trim(),
        ownerUserId: '',
        dueOn: this.caseDueOn(),
        resolution: '',
        expectedVersion: 0,
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.caseTitle.set('');
          this.caseDescription.set('');
          this.caseResourceId.set('');
          this.caseDueOn.set('');
          this.caseBusy.set(false);
          this.loadCases();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tạo được hồ sơ kiểm toán.'));
          this.caseBusy.set(false);
        },
      });
  }

  updateResolution(caseId: string, value: string): void {
    this.resolutionDrafts.update((current) => ({ ...current, [caseId]: value }));
  }

  advanceCase(item: AuditCase): void {
    const target: Record<AuditCaseStatus, AuditCaseStatus | null> = {
      OPEN: 'IN_REMEDIATION',
      IN_REMEDIATION: 'RESOLVED',
      RESOLVED: 'CLOSED',
      CLOSED: null,
    };
    const status = target[item.status];
    if (!status) return;
    const resolution = this.resolutionDrafts()[item.id]?.trim() || item.resolution || '';
    if ((status === 'RESOLVED' || status === 'CLOSED') && resolution.length < 5) {
      this.error.set('Nhập kết quả khắc phục ít nhất 5 ký tự trước khi giải quyết hồ sơ.');
      return;
    }
    this.caseBusy.set(true);
    this.reporting
      .updateAuditCase(item.id, this.casePayload(item, status, resolution))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.caseBusy.set(false);
          this.loadCases();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không cập nhật được hồ sơ kiểm toán.'));
          this.caseBusy.set(false);
        },
      });
  }

  downloadEvidence(): void {
    const requestId = this.evidenceRequestId().trim();
    if (!requestId) return;
    this.reporting
      .evidencePackage(requestId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (blob) => {
          const url = URL.createObjectURL(blob);
          const anchor = document.createElement('a');
          anchor.href = url;
          anchor.download = `dx-os-evidence-${requestId}.json`;
          anchor.click();
          URL.revokeObjectURL(url);
        },
        error: (error: unknown) =>
          this.error.set(problemMessage(error, 'Không xuất được gói bằng chứng.')),
      });
  }

  caseActionLabel(status: AuditCaseStatus): string {
    return {
      OPEN: 'Bắt đầu khắc phục',
      IN_REMEDIATION: 'Đánh dấu đã giải quyết',
      RESOLVED: 'Đóng hồ sơ',
      CLOSED: 'Đã đóng',
    }[status];
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

  private loadCases(): void {
    this.reporting
      .auditCases()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => this.cases.set(result),
        error: (error: unknown) =>
          this.error.set(problemMessage(error, 'Không tải được hồ sơ kiểm toán.')),
      });
  }

  private casePayload(item: AuditCase, status: AuditCaseStatus, resolution: string): SaveAuditCase {
    return {
      title: item.title,
      description: item.description,
      severity: item.severity,
      status,
      resourceType: item.resourceType || '',
      resourceId: item.resourceId || '',
      ownerUserId: item.ownerUserId || '',
      dueOn: item.dueOn || '',
      resolution,
      expectedVersion: item.version,
    };
  }
}
