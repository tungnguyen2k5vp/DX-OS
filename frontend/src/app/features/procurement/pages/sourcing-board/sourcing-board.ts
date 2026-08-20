import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom, forkJoin } from 'rxjs';
import { revealWorkspace } from '../../../../shared/utils/reveal-workspace';
import { problemMessage } from '../../data-access/problem-details';
import { SourcingBoard, SourcingCase, Supplier } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';

@Component({
  selector: 'app-sourcing-board',
  imports: [DatePipe, RouterLink, HlmBadge, HlmButton, ...HlmCardImports, MoneyPipe],
  templateUrl: './sourcing-board.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SourcingBoardPage {
  private readonly service = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);

  readonly board = signal<SourcingBoard | null>(null);
  readonly suppliers = signal<Supplier[]>([]);
  readonly selectedCase = signal<SourcingCase | null>(null);
  readonly loading = signal(true);
  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly success = signal<string | null>(null);

  readonly supplierId = signal('');
  readonly quoteReference = signal('');
  readonly amount = signal('');
  readonly deliveryOn = signal(new Date(Date.now() + 14 * 86_400_000).toISOString().slice(0, 10));
  readonly warrantyMonths = signal(12);
  readonly paymentTerms = signal('Thanh toán trong 30 ngày sau khi nghiệm thu');
  readonly note = signal('');
  readonly selectedSupplier = computed(
    () => this.suppliers().find((supplier) => supplier.id === this.supplierId()) ?? null,
  );

  constructor() {
    this.load();
  }

  open(item: SourcingCase): void {
    this.selectedCase.set(item);
    this.amount.set(item.requestAmount);
    this.quoteReference.set(`BG-${item.requestCode}`);
    this.error.set(null);
    revealWorkspace('quote-workspace');
  }

  async addQuote(): Promise<void> {
    const item = this.selectedCase();
    if (!item || !this.supplierId()) return;
    this.busy.set(true);
    this.error.set(null);
    this.success.set(null);
    try {
      await firstValueFrom(
        this.service.createSupplierQuote(
          {
            purchaseRequestId: item.purchaseRequestId,
            supplierId: this.supplierId(),
            quoteReference: this.quoteReference().trim(),
            amount: this.amount(),
            currency: item.currency,
            deliveryOn: this.deliveryOn(),
            warrantyMonths: this.warrantyMonths(),
            paymentTerms: this.paymentTerms().trim(),
            note: this.note().trim(),
          },
          crypto.randomUUID(),
        ),
      );
      this.success.set('Đã ghi nhận báo giá và tính lại điểm so sánh.');
      this.supplierId.set('');
      this.load(item.purchaseRequestId);
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không ghi nhận được báo giá. Hãy kiểm tra dữ liệu.'));
    } finally {
      this.busy.set(false);
    }
  }

  async selectQuote(item: SourcingCase, quoteId: string, quoteVersion: number): Promise<void> {
    if (!confirm('Chọn báo giá này làm kết quả cuối cùng? Các báo giá khác sẽ được đánh dấu không được chọn.')) return;
    this.busy.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.service.selectSupplierQuote(
          quoteId,
          item.version,
          quoteVersion,
          'Chọn theo điểm tổng hợp giá, tiến độ, chất lượng và tuân thủ',
          crypto.randomUUID(),
        ),
      );
      this.success.set('Đã chọn nhà cung cấp. Phiếu sẵn sàng để phát hành đơn hàng.');
      this.load(item.purchaseRequestId);
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không chọn được báo giá. Dữ liệu có thể vừa thay đổi.'));
    } finally {
      this.busy.set(false);
    }
  }

  score(value: number): string {
    return `${new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 1 }).format(value)}/100`;
  }

  orderDraftParams(item: SourcingCase, quote: SourcingCase['quotes'][number]): Record<string, string> {
    const noteParts = [
      `Tạo từ báo giá ${quote.quoteReference}.`,
      `Bảo hành ${quote.warrantyMonths} tháng.`,
      quote.paymentTerms,
      quote.note?.trim(),
    ].filter(Boolean);

    return {
      draftRequestId: item.purchaseRequestId,
      draftSupplierId: quote.supplierId,
      draftReference: quote.quoteReference,
      draftDeliveryOn: quote.deliveryOn,
      draftNote: noteParts.join(' '),
    };
  }

  private load(reopenRequestId?: string): void {
    this.loading.set(true);
    forkJoin({ board: this.service.sourcingBoard(), suppliers: this.service.suppliers() })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: ({ board, suppliers }) => {
          this.board.set(board);
          this.suppliers.set(suppliers.items.filter((item) => item.status === 'ACTIVE'));
          if (reopenRequestId) this.selectedCase.set(board.items.find((item) => item.purchaseRequestId === reopenRequestId) || null);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tải được bảng so sánh báo giá.'));
          this.loading.set(false);
        },
      });
  }
}
