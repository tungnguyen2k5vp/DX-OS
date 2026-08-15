import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { APP_CONFIG } from '../../../core/config/app-config';
import {
  AuditCenter,
  AuditQuery,
  ProcurementReportDashboard,
  ProcurementReportQuery,
} from './reporting.models';

@Injectable({ providedIn: 'root' })
export class ReportingService {
  private readonly http = inject(HttpClient);
  private readonly config = inject(APP_CONFIG);

  procurementDashboard(query: ProcurementReportQuery): Observable<ProcurementReportDashboard> {
    let params = new HttpParams().set('from', query.from).set('to', query.to);
    if (query.departmentId) {
      params = params.set('departmentId', query.departmentId);
    }
    if (query.costCenter) {
      params = params.set('costCenter', query.costCenter);
    }
    if (query.currency) {
      params = params.set('currency', query.currency);
    }
    return this.http.get<ProcurementReportDashboard>(
      `${this.config.apiBaseUrl}/api/v1/reports/procurement`,
      { params },
    );
  }

  auditEvents(query: AuditQuery = {}): Observable<AuditCenter> {
    let params = new HttpParams()
      .set('page', query.page ?? 1)
      .set('pageSize', query.pageSize ?? 20);
    for (const [key, value] of Object.entries({
      resourceType: query.resourceType,
      action: query.action,
      from: query.from,
      to: query.to,
    })) {
      if (value) {
        params = params.set(key, value);
      }
    }
    return this.http.get<AuditCenter>(`${this.config.apiBaseUrl}/api/v1/audit/events`, {
      params,
    });
  }
}
