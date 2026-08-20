import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { revealWorkspace } from '../../../../shared/utils/reveal-workspace';
import { AuthService } from '../../../../core/auth/auth.service';
import { problemMessage } from '../../data-access/problem-details';
import {
  FulfillmentStatus,
  OperationsBoard as OperationsBoardModel,
  PurchaseOrder,
  PurchaseRequestItem,
  ReceiptCondition,
  ReceiptHistory,
  ReceiptOutcome,
  RecordReceiptItem,
  Supplier,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';

interface OrderDraftFromQuote {
  requestId: string;
  supplierId: string;
  reference: string;
  deliveryOn: string;
  note: string;
}

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
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private generation = 0;
  private pendingOrderDraft = this.readOrderDraft();

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
  readonly receiptOrder = signal<PurchaseOrder | null>(null);
  readonly receiptHistory = signal<ReceiptHistory | null>(null);
  readonly receiptOutcome = signal<ReceiptOutcome>('COMPLETE');
  readonly receiptDate = signal(new Date().toISOString().slice(0, 10));
  readonly receiptNote = signal('Đã kiểm tra hàng hóa và chứng từ giao nhận.');
  readonly receiptLines = signal<
    Array<
      PurchaseRequestItem & { quantityReceived: string; condition: ReceiptCondition; note: string }
    >
  >([]);
  readonly managingOrder = signal<PurchaseOrder | null>(null);
  readonly manageSupplierId = signal('');
  readonly manageExternalReference = signal('');
  readonly manageExpectedDeliveryOn = signal('');
  readonly manageNote = signal('');
  readonly cancellationReason = signal('');

  constructor() {
    this.load();
    if (this.auth.roles().includes('finance')) {
      this.procurement
        .suppliers()
        .pipe(takeUntilDestroyed(this.destroyRef))
        .subscribe({
          next: (result) => {
            this.suppliers.set(result.items.filter((item) => item.status === 'ACTIVE'));
            this.openOrderDraftWhenReady();
          },
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
    revealWorkspace('order-workspace');
  }

  cancelOrder(): void {
    this.selected.set(null);
  }

  async openReceipt(order: PurchaseOrder): Promise<void> {
    this.receivingId.set(order.purchaseRequestId);
    this.error.set(null);
    try {
      const [request, history] = await Promise.all([
        firstValueFrom(this.procurement.get(order.purchaseRequestId)),
        firstValueFrom(this.procurement.receipts(order.purchaseRequestId)),
      ]);
      this.receiptOrder.set(order);
      this.receiptHistory.set(history);
      this.receiptOutcome.set('COMPLETE');
      this.receiptDate.set(new Date().toISOString().slice(0, 10));
      this.receiptNote.set('Đã kiểm tra hàng hóa và chứng từ giao nhận.');
      this.receiptLines.set(
        (request.items ?? []).map((item) => ({
          ...item,
          quantityReceived: item.quantity,
          condition: 'ACCEPTED' as ReceiptCondition,
          note: '',
        })),
      );
      revealWorkspace('receipt-workspace');
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không tải được thông tin để lập biên bản nhận hàng.'));
    } finally {
      this.receivingId.set(null);
    }
  }

  closeReceipt(): void {
    this.receiptOrder.set(null);
    this.receiptHistory.set(null);
  }

  changeReceiptOutcome(value: string): void {
    const outcome = value as ReceiptOutcome;
    this.receiptOutcome.set(outcome);
    if (outcome === 'COMPLETE') {
      this.receiptLines.update((items) =>
        items.map((item) => ({ ...item, quantityReceived: item.quantity, condition: 'ACCEPTED' })),
      );
    }
  }

  updateReceiptLine(
    itemId: string,
    field: 'quantityReceived' | 'condition' | 'note',
    value: string,
  ): void {
    this.receiptLines.update((items) =>
      items.map((item) => (item.id === itemId ? { ...item, [field]: value } : item)),
    );
  }

  async submitReceipt(): Promise<void> {
    const order = this.receiptOrder();
    if (!order || this.receiptNote().trim().length < 5) {
      this.error.set('Ghi chú biên bản phải có ít nhất 5 ký tự.');
      return;
    }
    const items: RecordReceiptItem[] = this.receiptLines().map((item) => ({
      purchaseRequestItemId: item.id,
      quantityReceived: item.quantityReceived || '0',
      condition: item.condition,
      note: item.note,
    }));
    this.receivingId.set(order.purchaseRequestId);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.procurement.recordReceipt(
          order.purchaseRequestId,
          {
            expectedVersion: order.version,
            outcome: this.receiptOutcome(),
            receivedOn: this.receiptDate(),
            note: this.receiptNote(),
            items,
          },
          crypto.randomUUID(),
        ),
      );
      this.closeReceipt();
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không ghi nhận được biên bản giao nhận.'));
    } finally {
      this.receivingId.set(null);
    }
  }

  openManageOrder(order: PurchaseOrder): void {
    this.managingOrder.set(order);
    this.manageSupplierId.set(order.supplierId ?? '');
    this.manageExternalReference.set(order.externalReference ?? '');
    this.manageExpectedDeliveryOn.set(order.expectedDeliveryOn ?? this.tomorrow());
    this.manageNote.set(order.note ?? '');
    this.cancellationReason.set('');
    revealWorkspace('manage-order-workspace');
  }

  closeManageOrder(): void {
    this.managingOrder.set(null);
  }

  async saveOrderChanges(): Promise<void> {
    const order = this.managingOrder();
    if (!order || !this.manageSupplierId() || !this.manageExpectedDeliveryOn()) return;
    this.saving.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.procurement.updatePurchaseOrder(order.purchaseRequestId, {
          supplierId: this.manageSupplierId(),
          externalReference: this.manageExternalReference(),
          expectedDeliveryOn: this.manageExpectedDeliveryOn(),
          note: this.manageNote(),
          expectedVersion: order.version,
        }),
      );
      this.closeManageOrder();
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không cập nhật được đơn hàng.'));
    } finally {
      this.saving.set(false);
    }
  }

  async cancelManagedOrder(): Promise<void> {
    const order = this.managingOrder();
    if (!order || this.cancellationReason().trim().length < 10) {
      this.error.set('Lý do hủy đơn phải có ít nhất 10 ký tự.');
      return;
    }
    this.saving.set(true);
    this.error.set(null);
    try {
      await firstValueFrom(
        this.procurement.cancelPurchaseOrder(
          order.purchaseRequestId,
          order.version,
          this.cancellationReason(),
          crypto.randomUUID(),
        ),
      );
      this.closeManageOrder();
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không hủy được đơn hàng.'));
    } finally {
      this.saving.set(false);
    }
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
      PARTIALLY_RECEIVED: 'Đã nhận một phần',
      RECEIPT_EXCEPTION: 'Có sự cố giao nhận',
      RECEIVED: 'Đã nhận',
      CANCELLED: 'Đã hủy',
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
          this.openOrderDraftWhenReady();
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

  private readOrderDraft(): OrderDraftFromQuote | null {
    const params = this.route.snapshot.queryParamMap;
    const requestId = params.get('draftRequestId')?.trim() ?? '';
    const supplierId = params.get('draftSupplierId')?.trim() ?? '';
    const reference = params.get('draftReference')?.trim() ?? '';
    const deliveryOn = params.get('draftDeliveryOn')?.trim() ?? '';

    if (!requestId || !supplierId || !reference || !deliveryOn) return null;

    return {
      requestId,
      supplierId,
      reference,
      deliveryOn,
      note: params.get('draftNote')?.trim() ?? '',
    };
  }

  private openOrderDraftWhenReady(): void {
    const draft = this.pendingOrderDraft;
    const board = this.board();
    if (!draft || !board || this.suppliers().length === 0) return;

    const request = board.items.find(
      (item) => item.purchaseRequestId === draft.requestId && item.canPlaceOrder,
    );
    const supplierAvailable = this.suppliers().some((supplier) => supplier.id === draft.supplierId);

    this.pendingOrderDraft = null;
    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {},
      replaceUrl: true,
    });

    if (!request || !supplierAvailable) {
      this.error.set(
        'Không thể tạo đơn từ báo giá này vì phiếu hoặc nhà cung cấp vừa thay đổi. Hãy tải lại bảng so sánh.',
      );
      return;
    }

    this.selected.set(request);
    this.supplierId.set(draft.supplierId);
    this.externalReference.set(draft.reference);
    this.expectedDeliveryOn.set(draft.deliveryOn);
    this.note.set(draft.note);
    this.error.set(null);
    revealWorkspace('order-workspace');
  }
}
