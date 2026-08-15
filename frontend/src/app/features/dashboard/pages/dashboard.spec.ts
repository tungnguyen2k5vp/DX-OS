import { provideRouter } from '@angular/router';
import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { IdentityService } from '../../../core/http/identity.service';
import {
  BudgetDashboard,
  PurchaseRequest,
  PurchaseRequestPage,
} from '../../procurement/data-access/procurement.models';
import { ProcurementService } from '../../procurement/data-access/procurement.service';
import { ProcurementReportDashboard } from '../../reporting/data-access/reporting.models';
import { ReportingService } from '../../reporting/data-access/reporting.service';
import { Dashboard } from './dashboard';

const request: PurchaseRequest = {
  id: 'request-id',
  requestCode: 'PR-2026-0001',
  requesterId: 'employee-id',
  requesterName: 'Nguyễn Văn A',
  departmentId: 'department-id',
  departmentName: 'Kinh doanh',
  title: 'Mua máy tính phục vụ dự án',
  reason: 'Thiết bị hiện tại không đáp ứng',
  currency: 'VND',
  totalAmount: '25000000.0000',
  costCenter: 'CC-SALES',
  status: 'MANAGER_APPROVED',
  version: 2,
  createdAt: '2026-08-01T00:00:00Z',
  updatedAt: '2026-08-02T00:00:00Z',
};

const budget: BudgetDashboard = {
  allocations: [],
  totals: [],
  reservations: [],
  adjustments: [],
  alertCount: 2,
  canManage: true,
};

const report: ProcurementReportDashboard = {
  filters: { from: '2026-07-01', to: '2026-07-31' },
  summary: {
    totalRequests: 8,
    approvedCount: 5,
    rejectedCount: 1,
    returnedCount: 1,
    slaBreachedCount: 1,
    averageLeadTimeHours: '16.5',
    attachmentRequiredCount: 3,
    attachmentCompliantCount: 3,
    attachmentComplianceRate: '100',
  },
  currencyTotals: [],
  statuses: [],
  trends: [],
  departments: [],
  budgets: [],
  generatedAt: '2026-08-14T00:00:00Z',
};

describe('Dashboard', () => {
  it('shows finance work, live metrics and role-specific shortcuts', async () => {
    const list = vi.fn((query: { status?: string }) => {
      if (query.status === 'MANAGER_APPROVED') {
        return of(page(3, [request]));
      }
      if (query.status === 'APPROVED') {
        return of(page(12));
      }
      return of(page(18, [request]));
    });

    await TestBed.configureTestingModule({
      imports: [Dashboard],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: { roles: () => ['finance'] } },
        {
          provide: IdentityService,
          useValue: {
            getCurrentUser: () =>
              of({
                subject: 'finance-id',
                username: 'finance1',
                roles: ['finance'],
              }),
          },
        },
        {
          provide: ProcurementService,
          useValue: { list, budgetDashboard: () => of(budget) },
        },
        {
          provide: ReportingService,
          useValue: { procurementDashboard: () => of(report) },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(Dashboard);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.textContent).toContain('Tài chính');
    expect(element.textContent).toContain('Chờ duyệt tài chính');
    expect(element.textContent).toContain('PR-2026-0001');
    expect(element.querySelector('a[href="/approvals"]')).toBeTruthy();
    expect(element.querySelector('a[href="/budgets"]')).toBeTruthy();
    expect(element.querySelector('a[href="/reports"]')).toBeTruthy();
    expect(list).toHaveBeenCalledTimes(3);
  });
});

function page(total: number, items: PurchaseRequest[] = []): PurchaseRequestPage {
  return {
    items,
    page: 1,
    pageSize: 5,
    total,
    pages: Math.ceil(total / 5),
  };
}
