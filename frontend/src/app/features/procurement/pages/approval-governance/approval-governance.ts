import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom, Observable } from 'rxjs';
import { problemMessage } from '../../data-access/problem-details';
import { ApprovalGovernance } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';

@Component({
  selector: 'app-approval-governance',
  imports: [HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './approval-governance.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ApprovalGovernancePage {
  private readonly service = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);

  readonly model = signal<ApprovalGovernance | null>(null);
  readonly loading = signal(true);
  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly success = signal<string | null>(null);

  readonly delegateUserId = signal('');
  readonly startsOn = signal(new Date().toISOString().slice(0, 10));
  readonly endsOn = signal(new Date(Date.now() + 7 * 86_400_000).toISOString().slice(0, 10));
  readonly delegationReason = signal('Ủy quyền xử lý công việc trong thời gian vắng mặt');

  readonly ruleName = signal('Quy trình phê duyệt mới');
  readonly minimumAmount = signal('0');
  readonly maximumAmount = signal('');
  readonly requiresManager = signal(true);
  readonly requiresFinance = signal(true);
  readonly priority = signal(100);

  constructor() {
    this.load();
  }

  async createDelegation(): Promise<void> {
    if (!this.delegateUserId()) {
      this.error.set('Hãy chọn người nhận ủy quyền.');
      return;
    }
    await this.run(
      this.service.createApprovalDelegation({
        delegateUserId: this.delegateUserId(),
        startsOn: this.startsOn(),
        endsOn: this.endsOn(),
        reason: this.delegationReason().trim(),
      }),
      'Đã tạo ủy quyền phê duyệt.',
    );
  }

  async setDelegationActive(id: string, active: boolean, version: number): Promise<void> {
    await this.run(
      this.service.setApprovalDelegationActive(id, active, version),
      active ? 'Đã kích hoạt lại ủy quyền.' : 'Đã dừng ủy quyền.',
    );
  }

  async createRule(): Promise<void> {
    await this.run(
      this.service.createApprovalRule({
        departmentId: '',
        name: this.ruleName().trim(),
        currency: 'VND',
        minimumAmount: this.minimumAmount(),
        maximumAmount: this.maximumAmount(),
        requiresManager: this.requiresManager(),
        requiresFinance: this.requiresFinance(),
        priority: this.priority(),
        active: true,
        expectedVersion: 0,
      }),
      'Đã thêm quy tắc phê duyệt.',
    );
  }

  async toggleRule(ruleId: string, active: boolean): Promise<void> {
    const rule = this.model()?.rules.find((item) => item.id === ruleId);
    if (!rule) return;
    await this.run(
      this.service.updateApprovalRule(rule.id, {
        departmentId: rule.departmentId || '',
        name: rule.name,
        currency: rule.currency,
        minimumAmount: rule.minimumAmount,
        maximumAmount: rule.maximumAmount || '',
        requiresManager: rule.requiresManager,
        requiresFinance: rule.requiresFinance,
        priority: rule.priority,
        active,
        expectedVersion: rule.version,
      }),
      active ? 'Đã kích hoạt quy tắc.' : 'Đã tạm dừng quy tắc.',
    );
  }

  private async run(request: Observable<unknown>, message: string): Promise<void> {
    this.busy.set(true);
    this.error.set(null);
    this.success.set(null);
    try {
      await firstValueFrom(request);
      this.success.set(message);
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không thực hiện được thao tác. Hãy kiểm tra dữ liệu và thử lại.'));
    } finally {
      this.busy.set(false);
    }
  }

  private load(): void {
    this.loading.set(true);
    this.service
      .approvalGovernance()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.model.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tải được cấu hình phê duyệt.'));
          this.loading.set(false);
        },
      });
  }
}
