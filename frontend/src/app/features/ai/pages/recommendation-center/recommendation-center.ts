import { DatePipe, KeyValuePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import {
  AIRecommendation,
  AIRecommendationList,
  AIRecommendationStatus,
  AIRiskLevel,
} from '../../data-access/ai.models';
import { AIService } from '../../data-access/ai.service';

@Component({
  selector: 'app-recommendation-center',
  imports: [DatePipe, KeyValuePipe, RouterLink, HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './recommendation-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RecommendationCenterPage {
  private readonly service = inject(AIService);
  private readonly destroyRef = inject(DestroyRef);

  readonly result = signal<AIRecommendationList | null>(null);
  readonly loading = signal(true);
  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly comments = signal<Record<string, string>>({});

  constructor() {
    this.load();
  }

  generate(): void {
    this.busy.set(true);
    this.error.set(null);
    this.service
      .generate()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.result.set(result);
          this.busy.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tạo được khuyến nghị.'));
          this.busy.set(false);
        },
      });
  }

  updateComment(id: string, value: string): void {
    this.comments.update((current) => ({ ...current, [id]: value }));
  }

  riskLabel(risk: AIRiskLevel): string {
    return {
      LOW: 'Thấp',
      MEDIUM: 'Trung bình',
      HIGH: 'Cao',
      CRITICAL: 'Nghiêm trọng',
    }[risk];
  }

  statusLabel(status: AIRecommendationStatus): string {
    return {
      PENDING: 'Chờ quyết định',
      APPROVED: 'Đã chấp thuận',
      REJECTED: 'Đã từ chối',
      DISMISSED: 'Đã bỏ qua',
    }[status];
  }

  typeLabel(type: string): string {
    return {
      SLA_BREACH: 'Rủi ro quá hạn xử lý',
      SLA_BREACH_RISK: 'Rủi ro quá hạn xử lý',
      HIGH_VALUE_REVIEW: 'Rà soát phiếu giá trị lớn',
      SUPPLIER_RISK: 'Rủi ro nhà cung cấp',
      DUPLICATE_REQUEST_RISK: 'Nguy cơ tạo phiếu trùng',
      SPLIT_PURCHASE_RISK: 'Nguy cơ chia nhỏ đơn hàng',
      PRICE_ANOMALY: 'Đơn giá khác thường',
      PAYMENT_OVERDUE: 'Thanh toán quá hạn',
      SUPPLIER_MASTER_CHANGED: 'Thông tin nhà cung cấp thay đổi',
      ROLE_CONFLICT: 'Xung đột phân quyền',
    }[type] || type;
  }

  evidenceLabel(key: string): string {
    return {
      amount: 'Giá trị phiếu',
      currency: 'Tiền tệ',
      requestCode: 'Mã phiếu',
      status: 'Trạng thái phiếu',
      requestAmount: 'Giá trị phiếu đang kiểm tra',
      rollingSevenDayAmount: 'Tổng giá trị các phiếu trong 7 ngày',
      threshold: 'Ngưỡng cảnh báo',
      matchingRequestCode: 'Phiếu tương tự',
      matchingAmount: 'Giá trị phiếu tương tự',
      createdDaysApart: 'Khoảng cách ngày tạo',
      supplierName: 'Nhà cung cấp',
      supplierCode: 'Mã nhà cung cấp',
      riskLevel: 'Mức rủi ro',
      complianceStatus: 'Tình trạng tuân thủ',
      invoiceNumber: 'Số hóa đơn',
      dueOn: 'Hạn thanh toán',
      slaDueAt: 'Hạn xử lý',
      invoiceAmount: 'Giá trị hóa đơn',
      paidAmount: 'Đã thanh toán',
      remainingAmount: 'Còn phải thanh toán',
      supplierUpdatedAt: 'Thời điểm cập nhật nhà cung cấp',
      orderedAt: 'Thời điểm đặt hàng',
      bankAccountNumber: 'Tài khoản thanh toán',
      itemDescription: 'Hàng hóa',
      unitPrice: 'Đơn giá hiện tại',
      historicalAverage: 'Đơn giá trung bình trước đây',
      sampleSize: 'Số mẫu so sánh',
      conflictingRoles: 'Các quyền xung đột',
      roles: 'Các quyền của người thực hiện',
      actor: 'Người thực hiện',
      action: 'Hành động đã thực hiện',
      occurredAt: 'Thời điểm thực hiện',
    }[key] || key;
  }

  evidenceValue(evidence: Record<string, unknown>, key: string, value: unknown): string {
    if (value === null || value === undefined || value === '') return '—';
    const currency = typeof evidence['currency'] === 'string' ? evidence['currency'] : 'VND';
    if (['amount', 'requestAmount', 'rollingSevenDayAmount', 'threshold', 'matchingAmount', 'unitPrice', 'historicalAverage', 'invoiceAmount', 'paidAmount', 'remainingAmount'].includes(key)) {
      const numeric = Number(value);
      return Number.isFinite(numeric)
        ? `${new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 2 }).format(numeric)} ${currency}`
        : String(value);
    }
    if (key === 'createdDaysApart') return `${value} ngày`;
    if (key === 'sampleSize') return `${value} mẫu so sánh`;
    if (key === 'riskLevel') return this.riskLabel(String(value) as AIRiskLevel);
    if (key === 'complianceStatus') {
      return { ACTIVE: 'Đạt yêu cầu', EXPIRED: 'Đã hết hiệu lực', BLOCKED: 'Bị chặn' }[String(value)] || String(value);
    }
    if (key === 'status') {
      return { SUBMITTED: 'Đã gửi', MANAGER_APPROVED: 'Trưởng bộ phận đã duyệt', APPROVED: 'Đã phê duyệt' }[String(value)] || String(value);
    }
    if (key === 'bankAccountNumber') {
      const account = String(value);
      return account.length > 4 ? `•••• ${account.slice(-4)}` : account;
    }
    if (['dueOn', 'slaDueAt', 'supplierUpdatedAt', 'orderedAt', 'occurredAt'].includes(key)) {
      const parsed = new Date(String(value));
      return Number.isNaN(parsed.getTime()) ? String(value) : new Intl.DateTimeFormat('vi-VN', { dateStyle: 'short', timeStyle: key === 'occurredAt' || key === 'supplierUpdatedAt' || key === 'orderedAt' ? 'short' : undefined }).format(parsed);
    }
    if (Array.isArray(value)) return value.map((item) => this.roleLabel(String(item))).join(', ');
    return key === 'action' ? this.actionLabel(String(value)) : String(value);
  }

  private roleLabel(role: string): string {
    return {
      employee: 'Nhân viên',
      department_manager: 'Trưởng bộ phận',
      finance: 'Tài chính',
      auditor: 'Kiểm toán',
      dx_admin: 'Quản trị DX-OS',
      ai_operator: 'Vận hành khuyến nghị',
    }[role] || role;
  }

  private actionLabel(action: string): string {
    return {
      MANAGER_APPROVED: 'Trưởng bộ phận phê duyệt',
      FINANCE_APPROVED: 'Tài chính phê duyệt',
      SUPPLIER_UPDATED: 'Cập nhật nhà cung cấp',
    }[action] || action;
  }

  decide(item: AIRecommendation, status: Exclude<AIRecommendationStatus, 'PENDING'>): void {
    const comment = this.comments()[item.id]?.trim() || '';
    if (comment.length < 5) {
      this.error.set('Nhập lý do quyết định ít nhất 5 ký tự.');
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    this.service
      .decide(item.id, { status, comment, expectedVersion: item.version })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.busy.set(false);
          this.load();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không lưu được quyết định.'));
          this.busy.set(false);
        },
      });
  }

  private load(): void {
    this.loading.set(true);
    this.service
      .list()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.result.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tải được trung tâm khuyến nghị.'));
          this.loading.set(false);
        },
      });
  }
}
