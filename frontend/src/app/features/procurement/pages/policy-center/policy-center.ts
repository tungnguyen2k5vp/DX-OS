import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { revealWorkspace } from '../../../../shared/utils/reveal-workspace';
import { problemMessage } from '../../data-access/problem-details';
import {
  AttachmentDocumentType,
  AttachmentPolicy,
  PolicyCenter,
  SLAPolicy,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';

@Component({
  selector: 'app-policy-center',
  imports: [HlmBadge, HlmButton, ...HlmCardImports, MoneyPipe],
  templateUrl: './policy-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PolicyCenterPage {
  private readonly procurement = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);
  readonly center = signal<PolicyCenter | null>(null);
  readonly loading = signal(true);
  readonly saving = signal(false);
  readonly error = signal<string | null>(null);
  readonly success = signal<string | null>(null);
  readonly selectedSLA = signal<SLAPolicy | null>(null);
  readonly selectedAttachment = signal<AttachmentPolicy | null>(null);
  readonly targetHours = signal(72);
  readonly thresholdAmount = signal('0');
  readonly documentType = signal<AttachmentDocumentType>('QUOTATION');

  constructor() {
    this.load();
  }

  editSLA(policy: SLAPolicy): void {
    this.selectedAttachment.set(null);
    this.selectedSLA.set(policy);
    this.targetHours.set(policy.targetHours);
    this.clearMessages();
    revealWorkspace('sla-policy-workspace');
  }

  editAttachment(policy: AttachmentPolicy): void {
    this.selectedSLA.set(null);
    this.selectedAttachment.set(policy);
    this.thresholdAmount.set(policy.thresholdAmount);
    this.documentType.set(policy.requiredDocumentType);
    this.clearMessages();
    revealWorkspace('attachment-policy-workspace');
  }

  cancel(): void {
    this.selectedSLA.set(null);
    this.selectedAttachment.set(null);
  }

  async saveSLA(): Promise<void> {
    const policy = this.selectedSLA();
    if (!policy || this.targetHours() < 1 || this.targetHours() > 720) {
      this.error.set('Thời hạn xử lý phải nằm trong khoảng 1 đến 720 giờ.');
      return;
    }
    await this.execute('Đã cập nhật thời hạn xử lý phê duyệt.', () =>
      firstValueFrom(
        this.procurement.updateSLAPolicy(policy.processName, {
          targetHours: this.targetHours(),
          active: policy.active,
          expectedVersion: policy.version,
        }),
      ),
    );
  }

  async saveAttachment(): Promise<void> {
    const policy = this.selectedAttachment();
    if (!policy || !this.thresholdAmount().trim()) {
      this.error.set('Vui lòng nhập ngưỡng chứng từ.');
      return;
    }
    await this.execute('Đã cập nhật quy tắc chứng từ.', () =>
      firstValueFrom(
        this.procurement.updateAttachmentPolicy(policy.id, {
          thresholdAmount: this.thresholdAmount(),
          requiredDocumentType: this.documentType(),
          active: policy.active,
          expectedVersion: policy.version,
        }),
      ),
    );
  }

  processLabel(processName: string): string {
    return processName === 'PURCHASE_REQUEST_APPROVAL' ? 'Phê duyệt phiếu mua sắm' : processName;
  }

  documentLabel(type: AttachmentDocumentType): string {
    return {
      QUOTATION: 'Báo giá',
      SPECIFICATION: 'Đặc tả',
      CONTRACT: 'Hợp đồng',
      OTHER: 'Tài liệu khác',
    }[type];
  }

  private async execute(message: string, work: () => Promise<unknown>): Promise<void> {
    this.saving.set(true);
    this.clearMessages();
    try {
      await work();
      this.cancel();
      this.success.set(message);
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không cập nhật được chính sách.'));
    } finally {
      this.saving.set(false);
    }
  }

  private load(): void {
    this.loading.set(true);
    this.procurement
      .policyCenter()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.center.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tải được chính sách vận hành.'));
          this.loading.set(false);
        },
      });
  }

  private clearMessages(): void {
    this.error.set(null);
    this.success.set(null);
  }
}
