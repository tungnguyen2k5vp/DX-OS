import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { APP_CONFIG } from '../../../core/config/app-config';
import {
  AdjustBudgetAllocation,
  BudgetCheck,
  BudgetAllocation,
  BudgetDashboard,
  BudgetSummary,
  CreatePurchaseRequest,
  PurchaseRequest,
  PurchaseRequestAttachment,
  PurchaseRequestAttachmentList,
  AttachmentDocumentType,
  PurchaseRequestPage,
  PurchaseRequestComment,
  PurchaseRequestCommentList,
  PurchaseRequestStatus,
  PurchaseRequestTimelinePage,
  TransitionPurchaseRequest,
  UpdatePurchaseRequest,
  WorkSummary,
  Supplier,
  SupplierInput,
  SupplierList,
  OperationsBoard,
  PurchaseOrder,
  CreatePurchaseOrder,
  CreateInvoice,
  InvoiceBoard,
  InvoiceBoardItem,
  TransitionInvoice,
  UpdateInvoice,
  PolicyCenter,
  SLAPolicy,
  AttachmentPolicy,
  UpdateSLAPolicy,
  UpdateAttachmentPolicy,
} from './procurement.models';

export interface ListPurchaseRequestsQuery {
  page?: number;
  pageSize?: number;
  status?: PurchaseRequestStatus;
}

@Injectable({ providedIn: 'root' })
export class ProcurementService {
  private readonly http = inject(HttpClient);
  private readonly config = inject(APP_CONFIG);
  private readonly collectionUrl = `${this.config.apiBaseUrl}/api/v1/purchase-requests`;

  list(query: ListPurchaseRequestsQuery = {}): Observable<PurchaseRequestPage> {
    let params = new HttpParams()
      .set('page', query.page ?? 1)
      .set('pageSize', query.pageSize ?? 20);
    if (query.status) {
      params = params.set('status', query.status);
    }
    return this.http.get<PurchaseRequestPage>(this.collectionUrl, { params });
  }

  get(requestId: string): Observable<PurchaseRequest> {
    return this.http.get<PurchaseRequest>(`${this.collectionUrl}/${encodeURIComponent(requestId)}`);
  }

  create(input: CreatePurchaseRequest): Observable<PurchaseRequest> {
    return this.http.post<PurchaseRequest>(this.collectionUrl, input);
  }

  update(requestId: string, input: UpdatePurchaseRequest): Observable<PurchaseRequest> {
    return this.http.patch<PurchaseRequest>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}`,
      input,
    );
  }

  timeline(requestId: string, page = 1, pageSize = 20): Observable<PurchaseRequestTimelinePage> {
    const params = new HttpParams().set('page', page).set('pageSize', pageSize);
    return this.http.get<PurchaseRequestTimelinePage>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/timeline`,
      { params },
    );
  }

  comments(requestId: string): Observable<PurchaseRequestCommentList> {
    return this.http.get<PurchaseRequestCommentList>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/comments`,
    );
  }

  addComment(requestId: string, body: string): Observable<PurchaseRequestComment> {
    return this.http.post<PurchaseRequestComment>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/comments`,
      { body },
    );
  }

  taskSummary(): Observable<WorkSummary> {
    return this.http.get<WorkSummary>(`${this.config.apiBaseUrl}/api/v1/me/tasks-summary`);
  }

  suppliers(): Observable<SupplierList> {
    return this.http.get<SupplierList>(`${this.config.apiBaseUrl}/api/v1/suppliers`);
  }

  createSupplier(input: SupplierInput): Observable<Supplier> {
    return this.http.post<Supplier>(`${this.config.apiBaseUrl}/api/v1/suppliers`, input);
  }

  updateSupplier(supplierId: string, input: SupplierInput): Observable<Supplier> {
    return this.http.patch<Supplier>(
      `${this.config.apiBaseUrl}/api/v1/suppliers/${encodeURIComponent(supplierId)}`,
      input,
    );
  }

  operationsBoard(): Observable<OperationsBoard> {
    return this.http.get<OperationsBoard>(
      `${this.config.apiBaseUrl}/api/v1/procurement-operations`,
    );
  }

  createPurchaseOrder(
    input: CreatePurchaseOrder,
    idempotencyKey: string,
  ): Observable<PurchaseOrder> {
    return this.http.post<PurchaseOrder>(
      `${this.config.apiBaseUrl}/api/v1/procurement-operations/orders`,
      input,
      { headers: new HttpHeaders({ 'Idempotency-Key': idempotencyKey }) },
    );
  }

  confirmReceipt(
    requestId: string,
    expectedVersion: number,
    actualDeliveryOn: string,
  ): Observable<PurchaseOrder> {
    return this.http.post<PurchaseOrder>(
      `${this.config.apiBaseUrl}/api/v1/procurement-operations/orders/${encodeURIComponent(requestId)}/receipt`,
      { expectedVersion, actualDeliveryOn },
    );
  }

  invoiceBoard(): Observable<InvoiceBoard> {
    return this.http.get<InvoiceBoard>(`${this.config.apiBaseUrl}/api/v1/invoices`);
  }

  createInvoice(input: CreateInvoice, idempotencyKey: string): Observable<InvoiceBoardItem> {
    return this.http.post<InvoiceBoardItem>(`${this.config.apiBaseUrl}/api/v1/invoices`, input, {
      headers: new HttpHeaders({ 'Idempotency-Key': idempotencyKey }),
    });
  }

  updateInvoice(invoiceId: string, input: UpdateInvoice): Observable<InvoiceBoardItem> {
    return this.http.patch<InvoiceBoardItem>(
      `${this.config.apiBaseUrl}/api/v1/invoices/${encodeURIComponent(invoiceId)}`,
      input,
    );
  }

  transitionInvoice(
    invoiceId: string,
    input: TransitionInvoice,
    idempotencyKey: string,
  ): Observable<InvoiceBoardItem> {
    return this.http.post<InvoiceBoardItem>(
      `${this.config.apiBaseUrl}/api/v1/invoices/${encodeURIComponent(invoiceId)}/transitions`,
      input,
      { headers: new HttpHeaders({ 'Idempotency-Key': idempotencyKey }) },
    );
  }

  policyCenter(): Observable<PolicyCenter> {
    return this.http.get<PolicyCenter>(`${this.config.apiBaseUrl}/api/v1/admin/policies`);
  }

  updateSLAPolicy(processName: string, input: UpdateSLAPolicy): Observable<SLAPolicy> {
    return this.http.patch<SLAPolicy>(
      `${this.config.apiBaseUrl}/api/v1/admin/policies/sla/${encodeURIComponent(processName)}`,
      input,
    );
  }

  updateAttachmentPolicy(
    ruleId: string,
    input: UpdateAttachmentPolicy,
  ): Observable<AttachmentPolicy> {
    return this.http.patch<AttachmentPolicy>(
      `${this.config.apiBaseUrl}/api/v1/admin/policies/attachments/${encodeURIComponent(ruleId)}`,
      input,
    );
  }

  budgetCheck(requestId: string): Observable<BudgetCheck> {
    return this.http.get<BudgetCheck>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/budget-check`,
    );
  }

  budgetSummary(costCenter: string, currency: string): Observable<BudgetSummary> {
    const params = new HttpParams().set('costCenter', costCenter).set('currency', currency);
    return this.http.get<BudgetSummary>(`${this.config.apiBaseUrl}/api/v1/budgets/summary`, {
      params,
    });
  }

  budgetDashboard(): Observable<BudgetDashboard> {
    return this.http.get<BudgetDashboard>(`${this.config.apiBaseUrl}/api/v1/budgets/dashboard`);
  }

  adjustBudget(
    allocationId: string,
    input: AdjustBudgetAllocation,
    idempotencyKey: string,
  ): Observable<BudgetAllocation> {
    return this.http.patch<BudgetAllocation>(
      `${this.config.apiBaseUrl}/api/v1/budgets/allocations/${encodeURIComponent(allocationId)}`,
      input,
      { headers: new HttpHeaders({ 'Idempotency-Key': idempotencyKey }) },
    );
  }

  transition(
    requestId: string,
    input: TransitionPurchaseRequest,
    idempotencyKey: string,
  ): Observable<PurchaseRequest> {
    return this.http.post<PurchaseRequest>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/transitions`,
      input,
      {
        headers: new HttpHeaders({ 'Idempotency-Key': idempotencyKey }),
      },
    );
  }

  attachments(requestId: string): Observable<PurchaseRequestAttachmentList> {
    return this.http.get<PurchaseRequestAttachmentList>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/attachments`,
    );
  }

  uploadAttachment(
    requestId: string,
    documentType: AttachmentDocumentType,
    file: File,
  ): Observable<PurchaseRequestAttachment> {
    const body = new FormData();
    body.set('documentType', documentType);
    body.set('file', file, file.name);
    return this.http.post<PurchaseRequestAttachment>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/attachments`,
      body,
    );
  }

  downloadAttachment(requestId: string, attachmentId: string): Observable<Blob> {
    return this.http.get(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/attachments/${encodeURIComponent(attachmentId)}/content`,
      { responseType: 'blob' },
    );
  }

  deleteAttachment(requestId: string, attachmentId: string): Observable<void> {
    return this.http.delete<void>(
      `${this.collectionUrl}/${encodeURIComponent(requestId)}/attachments/${encodeURIComponent(attachmentId)}`,
    );
  }
}
