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
  PurchaseRequestStatus,
  PurchaseRequestTimelinePage,
  TransitionPurchaseRequest,
  UpdatePurchaseRequest,
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
