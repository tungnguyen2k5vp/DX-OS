import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { ProcurementService } from '../../data-access/procurement.service';
import { InvoiceBoardPage } from './invoice-board';

describe('InvoiceBoardPage', () => {
  it('renders reconciliation counters and keeps auditor mode read-only', async () => {
    await TestBed.configureTestingModule({
      imports: [InvoiceBoardPage],
      providers: [
        provideRouter([]),
        {
          provide: ProcurementService,
          useValue: {
            invoiceBoard: () =>
              of({
                items: [
                  {
                    purchaseOrderId: 'order-id',
                    purchaseRequestId: 'request-id',
                    requestCode: 'PR-2026-001',
                    requestTitle: 'Thiết bị',
                    requesterName: 'Nhân viên',
                    departmentName: 'Kỹ thuật',
                    supplierId: 'supplier-id',
                    supplierCode: 'SUP-001',
                    supplierName: 'Nhà cung cấp mẫu',
                    orderCode: 'PO-2026-001',
                    orderStatus: 'RECEIVED',
                    orderAmount: '1000000.0000',
                    orderCurrency: 'VND',
                    actualDeliveryOn: '2026-08-15',
                    invoiceId: null,
                    invoiceNumber: null,
                    issuedOn: null,
                    dueOn: null,
                    invoiceAmount: null,
                    invoiceCurrency: null,
                    invoiceStatus: null,
                    matchStatus: 'NOT_RECORDED',
                    note: null,
                    version: 0,
                    paymentReference: null,
                    paidOn: null,
                    invoiceCreatedAt: null,
                    invoiceUpdatedAt: null,
                    paymentOverdue: false,
                    canManage: false,
                  },
                ],
                total: 1,
                awaitingInvoiceCount: 1,
                needsReviewCount: 0,
                readyToPayCount: 0,
                overdueCount: 0,
                paidCount: 0,
                canManage: false,
              }),
          },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(InvoiceBoardPage);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Hóa đơn và thanh toán');
    expect(text).toContain('Chưa có hóa đơn');
    expect(text).not.toContain('Ghi hóa đơn');
  });
});
