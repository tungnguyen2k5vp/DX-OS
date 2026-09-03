export interface ReportFilters {
  from: string;
  to: string;
  departmentId?: string;
  costCenter?: string;
  currency?: string;
}

export interface ReportSummary {
  totalRequests: number;
  approvedCount: number;
  rejectedCount: number;
  returnedCount: number;
  slaBreachedCount: number;
  averageLeadTimeHours: string;
  attachmentRequiredCount: number;
  attachmentCompliantCount: number;
  attachmentComplianceRate: string;
}

export interface ReportCurrencyTotal {
  currency: string;
  requestCount: number;
  totalAmount: string;
}

export interface ReportStatusBreakdown {
  status: string;
  currency: string;
  requestCount: number;
  totalAmount: string;
}

export interface ReportDailyTrend {
  date: string;
  currency: string;
  requestCount: number;
  approvedCount: number;
  totalAmount: string;
}

export interface ReportDailyRequest {
  id: string;
  requestCode: string;
  title: string;
  requesterUsername: string;
  requesterName: string;
  departmentName: string;
  status: string;
  currency: string;
  totalAmount: string;
  createdAt: string;
}

export interface ReportDailyRequestList {
  items: ReportDailyRequest[];
  total: number;
}

export interface ReportDepartmentBreakdown {
  departmentId: string;
  departmentName: string;
  currency: string;
  requestCount: number;
  approvedCount: number;
  totalAmount: string;
}

export interface ReportBudgetUtilization {
  periodCode: string;
  periodStart: string;
  periodEnd: string;
  costCenter: string;
  currency: string;
  allocatedAmount: string;
  reservedAmount: string;
  committedAmount: string;
  availableAmount: string;
  utilizationPercent: string;
}

export interface ProcurementReportDashboard {
  filters: ReportFilters;
  summary: ReportSummary;
  currencyTotals: ReportCurrencyTotal[];
  statuses: ReportStatusBreakdown[];
  trends: ReportDailyTrend[];
  departments: ReportDepartmentBreakdown[];
  budgets: ReportBudgetUtilization[];
  generatedAt: string;
}

export interface ProcurementReportQuery {
  from: string;
  to: string;
  departmentId?: string;
  costCenter?: string;
  currency?: string;
}

export interface ReportDailyRequestQuery {
  date: string;
  departmentId?: string;
  costCenter?: string;
  currency?: string;
}

export interface AuditEvent {
  id: string;
  resourceType: string;
  resourceId: string;
  action: string;
  actorName: string;
  actorRoles: string[];
  fromStatus: string | null;
  toStatus: string | null;
  correlationId: string | null;
  occurredAt: string;
}

export interface AuditCenter {
  items: AuditEvent[];
  page: number;
  pageSize: number;
  total: number;
  pages: number;
  todayCount: number;
  supplierChangeCount: number;
  purchaseOrderEventCount: number;
}

export interface AuditQuery {
  page?: number;
  pageSize?: number;
  resourceType?: string;
  action?: string;
  from?: string;
  to?: string;
}

export type AuditCaseSeverity = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
export type AuditCaseStatus = 'OPEN' | 'IN_REMEDIATION' | 'RESOLVED' | 'CLOSED';

export interface AuditCase {
  id: string;
  caseCode: string;
  title: string;
  description: string;
  severity: AuditCaseSeverity;
  status: AuditCaseStatus;
  resourceType?: string;
  resourceId?: string;
  ownerUserId?: string;
  ownerName?: string;
  dueOn?: string;
  resolution?: string;
  createdBy: string;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface AuditCaseList {
  items: AuditCase[];
  total: number;
  canManage: boolean;
  canExport: boolean;
}

export interface SaveAuditCase {
  title: string;
  description: string;
  severity: AuditCaseSeverity;
  status: AuditCaseStatus;
  resourceType: string;
  resourceId: string;
  ownerUserId: string;
  dueOn: string;
  resolution: string;
  expectedVersion: number;
}
