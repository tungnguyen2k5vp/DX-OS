import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { APP_CONFIG } from '../../../core/config/app-config';
import {
  CreatePurchaseRequest,
  PurchaseRequest,
  UpdatePurchaseRequest,
} from './procurement.models';
import { ProcurementService } from './procurement.service';

describe('ProcurementService', () => {
  let service: ProcurementService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: APP_CONFIG,
          useValue: {
            apiBaseUrl: 'http://api.test',
            oidc: {
              url: 'http://keycloak.test',
              realm: 'dx-os',
              clientId: 'dx-web',
            },
          },
        },
      ],
    });
    service = TestBed.inject(ProcurementService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('sends pagination and status when listing purchase requests', () => {
    service.list({ page: 2, pageSize: 10, status: 'DRAFT' }).subscribe((result) => {
      expect(result.page).toBe(2);
    });

    const request = http.expectOne(
      (candidate) =>
        candidate.url === 'http://api.test/api/v1/purchase-requests' &&
        candidate.params.get('page') === '2' &&
        candidate.params.get('pageSize') === '10' &&
        candidate.params.get('status') === 'DRAFT',
    );
    expect(request.request.method).toBe('GET');
    request.flush({
      items: [],
      page: 2,
      pageSize: 10,
      total: 0,
      pages: 0,
    });
  });

  it('posts only the create contract fields', () => {
    const input: CreatePurchaseRequest = {
      title: 'Laptop cho nhóm thiết kế',
      reason: 'Thay thế thiết bị đã hết vòng đời sử dụng.',
      currency: 'VND',
      costCenter: 'CC-GENERAL',
      items: [
        {
          description: 'Laptop',
          quantity: '2',
          unit: 'chiếc',
          unitPrice: '25000000',
        },
      ],
    };
    service.create(input).subscribe();

    const request = http.expectOne('http://api.test/api/v1/purchase-requests');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual(input);
    request.flush({ id: 'request-id' } as PurchaseRequest);
  });

  it('patches a draft with the expected version', () => {
    const input: UpdatePurchaseRequest = {
      title: 'Laptop cho nhóm thiết kế',
      reason: 'Cập nhật lý do mua sắm trước khi gửi duyệt.',
      currency: 'VND',
      costCenter: 'CC-GENERAL',
      expectedVersion: 3,
      items: [
        {
          description: 'Laptop',
          quantity: '2',
          unit: 'chiếc',
          unitPrice: '25000000',
        },
      ],
    };
    service.update('request-id', input).subscribe();

    const request = http.expectOne('http://api.test/api/v1/purchase-requests/request-id');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual(input);
    request.flush({ id: 'request-id' } as PurchaseRequest);
  });

  it('posts transitions with an idempotency key', () => {
    service
      .transition(
        'request-id',
        { action: 'APPROVE', expectedVersion: 4, comment: 'Approved.' },
        'approval-key-0001',
      )
      .subscribe();

    const request = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/transitions',
    );
    expect(request.request.method).toBe('POST');
    expect(request.request.headers.get('Idempotency-Key')).toBe('approval-key-0001');
    request.flush({ id: 'request-id' } as PurchaseRequest);
  });

  it('gets a paginated timeline without internal metadata parameters', () => {
    service.timeline('request-id', 2, 10).subscribe();

    const request = http.expectOne(
      (candidate) =>
        candidate.url === 'http://api.test/api/v1/purchase-requests/request-id/timeline' &&
        candidate.params.get('page') === '2' &&
        candidate.params.get('pageSize') === '10',
    );
    expect(request.request.method).toBe('GET');
    request.flush({
      items: [],
      page: 2,
      pageSize: 10,
      total: 0,
      pages: 0,
    });
  });

  it('lists and posts independent purchase request comments', () => {
    service.comments('request-id').subscribe();
    const listRequest = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/comments',
    );
    expect(listRequest.request.method).toBe('GET');
    listRequest.flush({ items: [], total: 0 });

    service.addComment('request-id', 'Please confirm delivery.').subscribe();
    const createRequest = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/comments',
    );
    expect(createRequest.request.method).toBe('POST');
    expect(createRequest.request.body).toEqual({ body: 'Please confirm delivery.' });
    createRequest.flush({ id: 'comment-id', body: 'Please confirm delivery.' });
  });

  it('gets the role-aware work summary', () => {
    service.taskSummary().subscribe((result) => expect(result.total).toBe(1));
    const request = http.expectOne('http://api.test/api/v1/me/tasks-summary');
    expect(request.request.method).toBe('GET');
    request.flush({ items: [{}], total: 1, overdueCount: 0, dueSoonCount: 0 });
  });

  it('manages suppliers and the fulfillment lifecycle', () => {
    const supplier = {
      code: 'VEN-01',
      name: 'Demo Vendor',
      taxCode: '',
      contactName: '',
      email: '',
      phone: '',
      status: 'ACTIVE' as const,
      riskLevel: 'LOW' as const,
    };
    service.createSupplier(supplier).subscribe();
    const supplierRequest = http.expectOne('http://api.test/api/v1/suppliers');
    expect(supplierRequest.request.method).toBe('POST');
    supplierRequest.flush({ id: 'supplier-id', ...supplier });

    service
      .createPurchaseOrder(
        {
          purchaseRequestId: 'request-id',
          supplierId: 'supplier-id',
          externalReference: 'ERP-01',
          expectedDeliveryOn: '2026-12-30',
          note: 'Deliver to reception.',
        },
        'purchase-order-0001',
      )
      .subscribe();
    const orderRequest = http.expectOne('http://api.test/api/v1/procurement-operations/orders');
    expect(orderRequest.request.headers.get('Idempotency-Key')).toBe('purchase-order-0001');
    orderRequest.flush({ status: 'ORDERED' });

    service.confirmReceipt('request-id', 1, '2026-08-14').subscribe();
    const receiptRequest = http.expectOne(
      'http://api.test/api/v1/procurement-operations/orders/request-id/receipt',
    );
    expect(receiptRequest.request.body).toEqual({
      expectedVersion: 1,
      actualDeliveryOn: '2026-08-14',
    });
    receiptRequest.flush({ status: 'RECEIVED' });
  });

  it('gets the budget check for a purchase request', () => {
    service.budgetCheck('request-id').subscribe();

    const request = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/budget-check',
    );
    expect(request.request.method).toBe('GET');
    request.flush({
      configured: true,
      result: 'AVAILABLE',
      requestedAmount: '100.0000',
      reservationState: null,
      summary: null,
    });
  });

  it('gets a budget summary with cost center and currency', () => {
    service.budgetSummary('CC-GENERAL', 'VND').subscribe();

    const request = http.expectOne(
      (candidate) =>
        candidate.url === 'http://api.test/api/v1/budgets/summary' &&
        candidate.params.get('costCenter') === 'CC-GENERAL' &&
        candidate.params.get('currency') === 'VND',
    );
    expect(request.request.method).toBe('GET');
    request.flush({
      periodCode: 'FY-2026',
      periodStart: '2026-01-01',
      periodEnd: '2026-12-31',
      costCenter: 'CC-GENERAL',
      currency: 'VND',
      allocatedAmount: '1000.0000',
      reservedAmount: '0.0000',
      committedAmount: '0.0000',
      availableAmount: '1000.0000',
    });
  });

  it('gets the finance and audit budget dashboard', () => {
    service.budgetDashboard().subscribe();

    const request = http.expectOne('http://api.test/api/v1/budgets/dashboard');
    expect(request.request.method).toBe('GET');
    request.flush({
      allocations: [],
      totals: [],
      reservations: [],
      adjustments: [],
      alertCount: 0,
      canManage: true,
    });
  });

  it('patches a budget allocation with version and idempotency', () => {
    const input = {
      allocatedAmount: '120000000000',
      expectedVersion: 2,
      reason: 'Approved annual allocation increase.',
    };
    service.adjustBudget('allocation-id', input, 'budget-adjustment-0001').subscribe();

    const request = http.expectOne('http://api.test/api/v1/budgets/allocations/allocation-id');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual(input);
    expect(request.request.headers.get('Idempotency-Key')).toBe('budget-adjustment-0001');
    request.flush({ id: 'allocation-id' });
  });

  it('lists attachment policy and metadata for a purchase request', () => {
    service.attachments('request-id').subscribe((result) => {
      expect(result.required).toBe(true);
    });
    const request = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/attachments',
    );
    expect(request.request.method).toBe('GET');
    request.flush({
      items: [],
      required: true,
      requirementMet: false,
      requiredDocumentType: 'QUOTATION',
      thresholdAmount: '20000000.0000',
      maxSizeBytes: 10485760,
      allowedContentTypes: ['application/pdf'],
    });
  });

  it('uploads attachment fields as multipart form data', () => {
    const file = new File(['%PDF-test'], 'bao-gia.pdf', { type: 'application/pdf' });
    service.uploadAttachment('request-id', 'QUOTATION', file).subscribe();

    const request = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/attachments',
    );
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toBeInstanceOf(FormData);
    const body = request.request.body as FormData;
    expect(body.get('documentType')).toBe('QUOTATION');
    expect((body.get('file') as File).name).toBe('bao-gia.pdf');
    request.flush({ id: 'attachment-id' });
  });

  it('downloads attachment content as a blob', () => {
    service.downloadAttachment('request-id', 'attachment-id').subscribe();
    const request = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/attachments/attachment-id/content',
    );
    expect(request.request.method).toBe('GET');
    expect(request.request.responseType).toBe('blob');
    request.flush(new Blob(['document'], { type: 'application/pdf' }));
  });

  it('deletes an attachment through the scoped API', () => {
    service.deleteAttachment('request-id', 'attachment-id').subscribe();
    const request = http.expectOne(
      'http://api.test/api/v1/purchase-requests/request-id/attachments/attachment-id',
    );
    expect(request.request.method).toBe('DELETE');
    request.flush(null);
  });

  it('loads the invoice reconciliation board', () => {
    service.invoiceBoard().subscribe();
    const request = http.expectOne('http://api.test/api/v1/invoices');
    expect(request.request.method).toBe('GET');
    request.flush({ items: [], total: 0, canManage: true });
  });

  it('creates an invoice with an idempotency key', () => {
    const input = {
      purchaseOrderId: 'order-id',
      invoiceNumber: 'INV-2026-001',
      issuedOn: '2026-08-15',
      dueOn: '2026-09-15',
      amount: '1250000.0000',
      currency: 'VND',
      note: '',
    };
    service.createInvoice(input, 'invoice-create-0001').subscribe();
    const request = http.expectOne('http://api.test/api/v1/invoices');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual(input);
    expect(request.request.headers.get('Idempotency-Key')).toBe('invoice-create-0001');
    request.flush({ invoiceId: 'invoice-id' });
  });

  it('records payment using version and idempotency', () => {
    service
      .transitionInvoice(
        'invoice-id',
        {
          action: 'MARK_PAID',
          expectedVersion: 3,
          paymentReference: 'BANK-2026-001',
          paidOn: '2026-08-15',
        },
        'invoice-payment-0001',
      )
      .subscribe();
    const request = http.expectOne('http://api.test/api/v1/invoices/invoice-id/transitions');
    expect(request.request.method).toBe('POST');
    expect(request.request.body.expectedVersion).toBe(3);
    expect(request.request.headers.get('Idempotency-Key')).toBe('invoice-payment-0001');
    request.flush({ invoiceId: 'invoice-id', invoiceStatus: 'PAID' });
  });

  it('loads and updates versioned operating policies', () => {
    service.policyCenter().subscribe();
    const list = http.expectOne('http://api.test/api/v1/admin/policies');
    expect(list.request.method).toBe('GET');
    list.flush({ slaPolicies: [], attachmentRules: [], canManage: true });

    const input = { targetHours: 48, active: true, expectedVersion: 2 };
    service.updateSLAPolicy('PURCHASE_REQUEST_APPROVAL', input).subscribe();
    const update = http.expectOne(
      'http://api.test/api/v1/admin/policies/sla/PURCHASE_REQUEST_APPROVAL',
    );
    expect(update.request.method).toBe('PATCH');
    expect(update.request.body).toEqual(input);
    update.flush({ processName: 'PURCHASE_REQUEST_APPROVAL', version: 3 });
  });
});
