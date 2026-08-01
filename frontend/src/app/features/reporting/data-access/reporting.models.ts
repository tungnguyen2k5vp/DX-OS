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
