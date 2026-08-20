import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { revealWorkspace } from '../../../../shared/utils/reveal-workspace';
import { problemMessage } from '../../data-access/problem-details';
import {
  InvoiceAction,
  InvoiceBoard,
  InvoiceBoardItem,
  InvoiceMatchStatus,
  InvoicePaymentList,
  InvoiceStatus,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';

@Component({
  selector: 'app-invoice-board',
  imports: [DatePipe, RouterLink, HlmBadge, HlmButton, ...HlmCardImports, MoneyPipe],
  templateUrl: './invoice-board.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class InvoiceBoardPage {
  private readonly procurement = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);
  private generation = 0;

  readonly board = signal<InvoiceBoard | null>(null);
  readonly loading = signal(true);
  readonly saving = signal(false);
  readonly error = signal<string | null>(null);
  readonly success = signal<string | null>(null);
  readonly selected = signal<InvoiceBoardItem | null>(null);
  readonly mode = signal<'CREATE' | 'EDIT' | 'PAY' | 'DISPUTE' | null>(null);
  readonly invoiceNumber = signal('');
  readonly issuedOn = signal(this.today());
  readonly dueOn = signal(this.shiftedDate(30));
  readonly amount = signal('');
  readonly currency = signal('VND');
  readonly note = signal('');
  readonly paymentReference = signal('');
  readonly paymentAmount = signal('');
  readonly paymentHistory = signal<InvoicePaymentList | null>(null);
  readonly paidOn = signal(this.today());
  readonly comment = signal('');

  constructor() {
    this.load();
  }

  open(item: InvoiceBoardItem, mode: 'CREATE' | 'EDIT' | 'PAY' | 'DISPUTE'): void {
    this.selected.set(item);
    this.mode.set(mode);
    this.invoiceNumber.set(item.invoiceNumber ?? '');
    this.issuedOn.set(item.issuedOn ?? this.today());
    this.dueOn.set(item.dueOn ?? this.shiftedDate(30));
    this.amount.set(item.invoiceAmount ?? item.orderAmount);
    this.currency.set(item.invoiceCurrency ?? item.orderCurrency);
    this.note.set(item.note ?? '');
    this.paymentReference.set('');
    this.paymentAmount.set(item.remainingAmount || item.invoiceAmount || '');
    this.paymentHistory.set(null);
    this.paidOn.set(this.today());
    this.comment.set('');
    this.error.set(null);
    this.success.set(null);
    revealWorkspace('invoice-workspace');
    if (mode === 'PAY' && item.invoiceId) {
      this.procurement
        .invoicePayments(item.invoiceId)
        .pipe(takeUntilDestroyed(this.destroyRef))
        .subscribe({ next: (history) => this.paymentHistory.set(history) });
    }
  }

  openNewInvoice(item: InvoiceBoardItem): void {
    this.open(item, 'CREATE');
    this.invoiceNumber.set('');
    this.amount.set(
      item.remainingAmount && item.remainingAmount !== '0'
        ? item.remainingAmount
        : item.orderAmount,
    );
    this.note.set('Hóa đơn bổ sung cho ' + item.orderCode);
  }

  close(): void {
    this.selected.set(null);
    this.mode.set(null);
  }

  async saveInvoice(): Promise<void> {
    const item = this.selected();
    if (!item || !this.invoiceNumber().trim() || !this.amount().trim()) {
      this.error.set('Vui lòng nhập số hóa đơn và số tiền.');
      return;
    }
    await this.execute('Đã lưu hóa đơn.', async () => {
      const input = {
        invoiceNumber: this.invoiceNumber(),
        issuedOn: this.issuedOn(),
        dueOn: this.dueOn(),
        amount: this.amount(),
        currency: this.currency(),
        note: this.note(),
      };
      if (item.invoiceId) {
        await firstValueFrom(
          this.procurement.updateInvoice(item.invoiceId, {
            ...input,
            expectedVersion: item.version,
          }),
        );
      } else {
        await firstValueFrom(
          this.procurement.createInvoice(
            { ...input, purchaseOrderId: item.purchaseOrderId },
            crypto.randomUUID(),
          ),
        );
      }
    });
  }

  async transition(item: InvoiceBoardItem, action: InvoiceAction): Promise<void> {
    if (!item.invoiceId) return;
    if (action === 'DISPUTE' && !this.comment().trim()) {
      this.error.set('Vui lòng nhập lý do đánh dấu tranh chấp.');
      return;
    }
    if (action === 'MARK_PAID' && !this.paymentReference().trim()) {
      this.error.set('Vui lòng nhập mã tham chiếu thanh toán.');
      return;
    }
    const labels: Record<InvoiceAction, string> = {
      VERIFY: 'Đã xác minh đối soát ba bên.',
      DISPUTE: 'Đã chuyển hóa đơn sang trạng thái tranh chấp.',
      REOPEN: 'Đã mở lại hóa đơn để xử lý.',
      MARK_PAID: 'Đã ghi nhận thanh toán.',
    };
    await this.execute(labels[action], async () => {
      await firstValueFrom(
        this.procurement.transitionInvoice(
          item.invoiceId!,
          {
            action,
            expectedVersion: item.version,
            comment: this.comment(),
            paymentReference: this.paymentReference(),
            paidOn: action === 'MARK_PAID' ? this.paidOn() : undefined,
          },
          crypto.randomUUID(),
        ),
      );
    });
  }

  async recordPayment(item: InvoiceBoardItem): Promise<void> {
    if (!item.invoiceId || !this.paymentReference().trim() || !this.paymentAmount().trim()) {
      this.error.set('Vui lòng nhập số tiền và mã tham chiếu thanh toán.');
      return;
    }
    await this.execute('Đã ghi nhận đợt thanh toán.', async () => {
      await firstValueFrom(
        this.procurement.recordInvoicePayment(
          item.invoiceId!,
          {
            expectedVersion: item.version,
            amount: this.paymentAmount(),
            paidOn: this.paidOn(),
            paymentReference: this.paymentReference(),
            note: this.comment(),
          },
          crypto.randomUUID(),
        ),
      );
    });
  }

  statusLabel(status: InvoiceStatus | null): string {
    if (!status) return 'Chưa ghi nhận';
    return {
      RECORDED: 'Đã ghi nhận',
      VERIFIED: 'Đã xác minh',
      DISPUTED: 'Tranh chấp',
      PAID: 'Đã thanh toán',
    }[status];
  }

  matchLabel(status: InvoiceMatchStatus): string {
    return {
      NOT_RECORDED: 'Chưa có hóa đơn',
      WAITING_RECEIPT: 'Chờ nhận hàng',
      CURRENCY_MISMATCH: 'Lệch tiền tệ',
      AMOUNT_MISMATCH: 'Lệch số tiền',
      PARTIAL_MATCH: 'Hóa đơn từng phần',
      MATCHED: 'Khớp ba bên',
    }[status];
  }

  statusVariant(status: InvoiceStatus | null): 'outline' | 'secondary' | 'destructive' {
    if (status === 'DISPUTED') return 'destructive';
    return status === 'VERIFIED' || status === 'PAID' ? 'secondary' : 'outline';
  }

  private async execute(message: string, work: () => Promise<void>): Promise<void> {
    this.saving.set(true);
    this.error.set(null);
    this.success.set(null);
    try {
      await work();
      this.close();
      this.success.set(message);
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không hoàn tất được thao tác hóa đơn.'));
    } finally {
      this.saving.set(false);
    }
  }

  private load(): void {
    const generation = ++this.generation;
    this.loading.set(true);
    this.procurement
      .invoiceBoard()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.generation) return;
          this.board.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.generation) return;
          this.error.set(problemMessage(error, 'Không tải được bảng hóa đơn.'));
          this.loading.set(false);
        },
      });
  }

  private today(): string {
    return new Date().toISOString().slice(0, 10);
  }

  private shiftedDate(days: number): string {
    const date = new Date();
    date.setDate(date.getDate() + days);
    return date.toISOString().slice(0, 10);
  }
}
