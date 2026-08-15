import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { problemMessage } from '../../data-access/problem-details';
import {
  FulfillmentStatus,
  OperationsBoard as OperationsBoardModel,
  PurchaseOrder,
  Supplier,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';

@Component({
  selector: 'app-operations-board',
  imports: [DatePipe, RouterLink, HlmBadge, HlmButton, ...HlmCardImports, MoneyPipe],
  templateUrl: './operations-board.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class OperationsBoard {
  private readonly procurement = inject(ProcurementService);
  private readonly auth = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);
  private generation = 0;

  readonly board = signal<OperationsBoardModel | null>(null);
  readonly suppliers = signal<Supplier[]>([]);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly selected = signal<PurchaseOrder | null>(null);
  readonly supplierId = signal('');
  readonly externalReference = signal('');
  readonly expectedDeliveryOn = signal(this.tomorrow());
  readonly note = signal('');
  readonly saving = signal(false);
  readonly receivingId = signal<string | null>(null);

  constructor() {
    this.load();
    if (this.auth.roles().includes('finance')) {
      this.procurement
        .suppliers()
        .pipe(takeUntilDestroyed(this.destroyRef))
        .subscribe({
          next: (result) =>
            this.suppliers.set(result.items.filter((item) => item.status === 'ACTIVE')),
          error: (error: unknown) =>
            this.error.set(problemMessage(error, 'Không tải được nhà cung cấp để đặt hàng.')),
        });
    }
  }

  openOrder(order: PurchaseOrder): void {
    this.selected.set(order);
    this.supplierId.set(this.suppliers()[0]?.id ?? '');
    this.externalReference.set('');
    this.expectedDeliveryOn.set(this.tomorrow());
    this.note.set('');
    this.error.set(null);
  }

  cancelOrder(): void {
    this.selected.set(null);
  }

  async placeOrder(): Promise<void> {
    const request = this.selected();
    if (!request || !this.supplierId() || !this.expectedDeliveryOn()) {
      this.error.set('Vui lòng chọn nhà cung cấp và ngày giao dự kiến.');
      return;
    }
    this.saving.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.procurement.createPurchaseOrder(
          {
            purchaseRequestId: request.purchaseRequestId,
            supplierId: this.supplierId(),
            externalReference: this.externalReference(),
            expectedDeliveryOn: this.expectedDeliveryOn(),
            note: this.note(),
          },
          crypto.randomUUID(),
        ),
      );
      this.selected.set(null);
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không tạo được đơn đặt hàng.'));
    } finally {
      this.saving.set(false);
    }
  }

  async confirmReceipt(order: PurchaseOrder): Promise<void> {
    this.receivingId.set(order.purchaseRequestId);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.procurement.confirmReceipt(
          order.purchaseRequestId,
          order.version,
          new Date().toISOString().slice(0, 10),
        ),
      );
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không xác nhận được việc nhận hàng.'));
    } finally {
      this.receivingId.set(null);
    }
  }

  statusLabel(status: FulfillmentStatus): string {
    return {
      AWAITING_ORDER: 'Chờ đặt hàng',
      ORDERED: 'Đang giao',
      RECEIVED: 'Đã nhận',
    }[status];
  }

  statusVariant(status: FulfillmentStatus): 'outline' | 'secondary' {
    return status === 'ORDERED' ? 'secondary' : 'outline';
  }

  private load(): void {
    const generation = ++this.generation;
    this.loading.set(true);
    this.procurement
      .operationsBoard()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.generation) return;
          this.board.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.generation) return;
          this.error.set(problemMessage(error, 'Không tải được bảng giao nhận.'));
          this.loading.set(false);
        },
      });
  }

  private tomorrow(): string {
    const date = new Date();
    date.setDate(date.getDate() + 1);
    return date.toISOString().slice(0, 10);
  }
}
