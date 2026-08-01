import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { HlmBadge, type BadgeVariants } from '@spartan-ng/helm/badge';
import { PurchaseRequestStatus } from '../data-access/procurement.models';

export const purchaseRequestStatusLabels: Record<PurchaseRequestStatus, string> = {
  DRAFT: 'Bản nháp',
  SUBMITTED: 'Đã gửi',
  MANAGER_APPROVED: 'Trưởng bộ phận đã duyệt',
  CHANGES_REQUESTED: 'Yêu cầu chỉnh sửa',
  APPROVED: 'Đã duyệt',
  REJECTED: 'Từ chối',
  CANCELLED: 'Đã hủy',
};

const statusVariants: Record<PurchaseRequestStatus, BadgeVariants['variant']> = {
  DRAFT: 'outline',
  SUBMITTED: 'secondary',
  MANAGER_APPROVED: 'secondary',
  CHANGES_REQUESTED: 'destructive',
  APPROVED: 'default',
  REJECTED: 'destructive',
  CANCELLED: 'outline',
};

@Component({
  selector: 'app-purchase-request-status-badge',
  imports: [HlmBadge],
  template: `<span hlmBadge [variant]="variant()">{{ label() }}</span>`,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PurchaseRequestStatusBadge {
  readonly status = input.required<PurchaseRequestStatus>();
  readonly label = computed(() => purchaseRequestStatusLabels[this.status()]);
  readonly variant = computed(() => statusVariants[this.status()]);
}
