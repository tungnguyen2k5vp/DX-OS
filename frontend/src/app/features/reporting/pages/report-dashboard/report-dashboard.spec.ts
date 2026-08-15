import { TestBed } from '@angular/core/testing';
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
  trends: [],
  departments: [],
  budgets: [],
  generatedAt: '2026-07-31T00:00:00Z',
};

describe('ReportDashboardPage', () => {
  it('renders KPI data and the Metabase link', async () => {
    await TestBed.configureTestingModule({
      imports: [ReportDashboardPage],
      providers: [
        {
          provide: ReportingService,
          useValue: { procurementDashboard: () => of(report) },
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
  });
});
