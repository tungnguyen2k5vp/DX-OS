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

  severityLabel(severity: AuditCaseSeverity): string {
    return {
      LOW: 'Thấp',
      MEDIUM: 'Trung bình',
      HIGH: 'Cao',
      CRITICAL: 'Nghiêm trọng',
    }[severity];
  }

  statusLabel(status: string | null): string {
    if (!status) return '';
    return (
      {
        DRAFT: 'Bản nháp',
        SUBMITTED: 'Đã gửi',
        CHANGES_REQUESTED: 'Yêu cầu chỉnh sửa',
        MANAGER_APPROVED: 'Trưởng bộ phận đã duyệt',
        APPROVED: 'Đã phê duyệt',
        REJECTED: 'Đã từ chối',
        CANCELLED: 'Đã hủy',
        ACTIVE: 'Đang hoạt động',
        INACTIVE: 'Ngừng hoạt động',
        ORDERED: 'Đã đặt hàng',
        PARTIALLY_RECEIVED: 'Đã nhận một phần',
        RECEIVED: 'Đã nhận đủ',
        RECORDED: 'Đã ghi nhận',
        VERIFIED: 'Đã xác minh',
        DISPUTED: 'Đang đối soát',
        PARTIALLY_PAID: 'Đã thanh toán một phần',
        PAID: 'Đã thanh toán',
        OPEN: 'Mới mở',
        IN_REMEDIATION: 'Đang khắc phục',
        RESOLVED: 'Đã giải quyết',
        CLOSED: 'Đã đóng',
        PENDING: 'Đang chờ',
        DISMISSED: 'Đã bỏ qua',
        true: 'Đang hoạt động',
        false: 'Ngừng hoạt động',
      }[status] ?? status
    );
  }

  auditActionLabel(action: string): string {
    return (
      {
        DRAFT_CREATED: 'Tạo bản nháp',
        DRAFT_UPDATED: 'Cập nhật bản nháp',
        SUBMITTED: 'Gửi phê duyệt',
        RESUBMITTED: 'Gửi lại phiếu',
        MANAGER_APPROVED: 'Trưởng bộ phận phê duyệt',
        FINANCE_APPROVED: 'Bộ phận Tài chính phê duyệt',
        CHANGES_REQUESTED: 'Yêu cầu chỉnh sửa',
        REJECTED: 'Từ chối phiếu',
        CANCELLED: 'Hủy phiếu',
        COMMENT_ADDED: 'Thêm trao đổi',
        ATTACHMENT_UPLOADED: 'Tải tệp lên',
        ATTACHMENT_DELETED: 'Xóa tệp',
        ORDER_PLACED: 'Đặt hàng',
        PURCHASE_ORDER_CREATED: 'Tạo đơn hàng',
        PURCHASE_ORDER_UPDATED: 'Cập nhật đơn hàng',
        PURCHASE_ORDER_CANCELLED: 'Hủy đơn hàng',
        DELIVERY_RECEIVED: 'Xác nhận đã nhận hàng',
        RECEIPT_PARTIAL: 'Ghi nhận nhận hàng một phần',
        RECEIPT_COMPLETE: 'Ghi nhận nhận đủ hàng',
        RECEIPT_EXCEPTION: 'Ghi nhận sự cố giao nhận',
        SUPPLIER_CREATED: 'Tạo nhà cung cấp',
        SUPPLIER_UPDATED: 'Cập nhật nhà cung cấp',
        BUDGET_ALLOCATION_ADJUSTED: 'Điều chỉnh hạn mức ngân sách',
        SLA_POLICY_UPDATED: 'Cập nhật quy tắc thời hạn xử lý',
        ATTACHMENT_POLICY_UPDATED: 'Cập nhật quy tắc tài liệu',
        APPROVAL_RULE_CREATED: 'Tạo quy tắc phê duyệt',
        APPROVAL_RULE_UPDATED: 'Cập nhật quy tắc phê duyệt',
        APPROVAL_DELEGATION_CREATED: 'Tạo ủy quyền phê duyệt',
        APPROVAL_DELEGATION_STATUS_UPDATED: 'Cập nhật trạng thái ủy quyền',
        SUPPLIER_QUOTE_RECORDED: 'Ghi nhận báo giá nhà cung cấp',
        SUPPLIER_QUOTE_UPDATED: 'Cập nhật báo giá nhà cung cấp',
        SUPPLIER_QUOTE_SELECTED: 'Chọn báo giá nhà cung cấp',
        INVOICE_RECORDED: 'Ghi nhận hóa đơn',
        INVOICE_UPDATED: 'Cập nhật hóa đơn',
        INVOICE_VERIFIED: 'Xác minh hóa đơn',
        INVOICE_DISPUTED: 'Đưa hóa đơn vào đối soát',
        INVOICE_REOPENED: 'Mở lại hóa đơn',
        INVOICE_MARKED_PAID: 'Đánh dấu hóa đơn đã thanh toán',
        INVOICE_PARTIAL_PAYMENT_RECORDED: 'Ghi nhận thanh toán một phần',
        INVOICE_FULL_PAYMENT_RECORDED: 'Ghi nhận thanh toán đủ',
        AUDIT_CASE_CREATED: 'Mở hồ sơ kiểm toán',
        AUDIT_CASE_UPDATED: 'Cập nhật hồ sơ kiểm toán',
        USER_ADMIN_UPDATED: 'Cập nhật tài khoản người dùng',
        DEPARTMENT_CREATED: 'Tạo phòng ban',
        DEPARTMENT_UPDATED: 'Cập nhật phòng ban',
        AI_RECOMMENDATION_DECIDED: 'Ra quyết định cho khuyến nghị AI',
      }[action] ?? action
    );
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
        purchase_invoice: 'Hóa đơn',
        budget_allocation: 'Ngân sách',
        supplier_quote: 'Báo giá nhà cung cấp',
        sourcing_case: 'Hồ sơ so sánh báo giá',
        approval_rule: 'Quy tắc phê duyệt',
        approval_delegation: 'Ủy quyền phê duyệt',
        operating_policy: 'Chính sách vận hành',
        attachment_policy: 'Quy tắc chứng từ',
        ai_recommendation: 'Khuyến nghị kiểm soát',
        audit_case: 'Hồ sơ kiểm toán',
        user: 'Người dùng',
        department: 'Phòng ban',
      }[type] ?? type
    );
  }

  actorRolesLabel(roles: string[]): string {
    const labels: Record<string, string> = {
      employee: 'Nhân viên',
      department_manager: 'Trưởng bộ phận',
      finance: 'Tài chính',
      auditor: 'Kiểm toán',
      dx_admin: 'Quản trị DX-OS',
      ai_operator: 'Vận hành khuyến nghị',
    };
    const visible = roles.filter((role) => labels[role]).map((role) => labels[role]);
    return visible.length ? visible.join(', ') : 'Quyền hệ thống mặc định';
  }

  statusChangeLabel(fromStatus: string | null, toStatus: string | null): string {
    if (!fromStatus && !toStatus) return 'Không có thay đổi trạng thái';
    return `${this.statusLabel(fromStatus) || 'Chưa có trạng thái'} → ${this.statusLabel(toStatus) || 'Chưa có trạng thái'}`;
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
