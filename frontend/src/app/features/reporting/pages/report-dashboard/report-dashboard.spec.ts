import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { APP_CONFIG } from '../../../../core/config/app-config';
import { ProcurementReportDashboard } from '../../data-access/reporting.models';
import { ReportingService } from '../../data-access/reporting.service';
import { ReportDashboardPage } from './report-dashboard';

const report: ProcurementReportDashboard = {
  filters: { from: '2026-07-01', to: '2026-07-31' },
  summary: {
    totalRequests: 4,
    approvedCount: 2,
    rejectedCount: 1,
    returnedCount: 1,
    slaBreachedCount: 0,
    averageLeadTimeHours: '12.50',
    attachmentRequiredCount: 2,
    attachmentCompliantCount: 2,
    attachmentComplianceRate: '100.00',
  },
  currencyTotals: [{ currency: 'VND', requestCount: 4, totalAmount: '100000000.0000' }],
  statuses: [
    { status: 'APPROVED', currency: 'VND', requestCount: 2, totalAmount: '50000000.0000' },
  ],
  trends: [
    {
      date: '2026-07-20',
      currency: 'VND',
      requestCount: 1,
      approvedCount: 0,
      totalAmount: '25000000.0000',
    },
  ],
  departments: [],
  budgets: [],
  generatedAt: '2026-07-31T00:00:00Z',
};

describe('ReportDashboardPage', () => {
  it('renders KPI data and the Metabase link', async () => {
    await TestBed.configureTestingModule({
      imports: [ReportDashboardPage],
      providers: [
        provideRouter([]),
        {
          provide: ReportingService,
          useValue: {
            procurementDashboard: () => of(report),
            dailyRequests: () =>
              of({
                items: [
                  {
                    id: 'request-id',
                    requestCode: 'PR-2026-000001',
                    title: 'Mua máy tính văn phòng',
                    requesterUsername: 'employee.demo',
                    requesterName: 'Nguyễn Minh Anh',
                    departmentName: 'Phòng ban chung',
                    status: 'SUBMITTED',
                    currency: 'VND',
                    totalAmount: '25000000.0000',
                    createdAt: '2026-07-20T08:30:00Z',
                  },
                ],
                total: 1,
              }),
          },
        },
        {
          provide: APP_CONFIG,
          useValue: {
            apiBaseUrl: 'http://api.test',
            metabaseUrl: 'http://metabase.test',
            oidc: { url: 'http://keycloak.test', realm: 'dx-os', clientId: 'dx-web' },
          },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(ReportDashboardPage);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.textContent).toContain('Báo cáo vận hành');
    expect(element.textContent).toContain('100.000.000 VND');
    expect(element.querySelector<HTMLAnchorElement>('a[href="http://metabase.test"]')).toBeTruthy();
    expect(element.textContent).toContain('Xuất CSV');

    const dayButton = element.querySelector<HTMLButtonElement>(
      'button[aria-controls="daily-detail-2026-07-20-VND"]',
    );
    expect(dayButton).toBeTruthy();
    dayButton?.click();
    fixture.detectChanges();
    expect(element.textContent).toContain('PR-2026-000001');
    expect(element.textContent).toContain('Mua máy tính văn phòng');
    expect(element.querySelector('a[href="/purchase-requests/request-id"]')).toBeTruthy();
  });
});
