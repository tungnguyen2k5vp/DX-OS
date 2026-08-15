import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { APP_CONFIG } from '../../../core/config/app-config';
import { ReportingService } from './reporting.service';

describe('ReportingService', () => {
  it('sends normalized report filters', () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: APP_CONFIG,
          useValue: {
            apiBaseUrl: 'http://api.test',
            metabaseUrl: 'http://metabase.test',
            oidc: { url: 'http://keycloak.test', realm: 'dx-os', clientId: 'dx-web' },
          },
        },
      ],
    });
    const service = TestBed.inject(ReportingService);
    const http = TestBed.inject(HttpTestingController);

    service
      .procurementDashboard({
        from: '2026-07-01',
        to: '2026-07-31',
        costCenter: 'CC-GENERAL',
        currency: 'VND',
      })
      .subscribe();

    const request = http.expectOne(
      (candidate) =>
        candidate.url === 'http://api.test/api/v1/reports/procurement' &&
        candidate.params.get('from') === '2026-07-01' &&
        candidate.params.get('to') === '2026-07-31' &&
        candidate.params.get('costCenter') === 'CC-GENERAL' &&
        candidate.params.get('currency') === 'VND',
    );
    expect(request.request.method).toBe('GET');
    request.flush({
      filters: { from: '2026-07-01', to: '2026-07-31' },
      summary: {},
      currencyTotals: [],
      statuses: [],
      trends: [],
      departments: [],
      budgets: [],
      generatedAt: '2026-07-31T00:00:00Z',
    });
    http.verify();
  });

  it('sends audit pagination and evidence filters', () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        {
          provide: APP_CONFIG,
          useValue: {
            apiBaseUrl: 'http://api.test',
            oidc: { url: 'http://keycloak.test', realm: 'dx-os', clientId: 'dx-web' },
          },
        },
      ],
    });
    const service = TestBed.inject(ReportingService);
    const http = TestBed.inject(HttpTestingController);
    service.auditEvents({ page: 2, pageSize: 10, resourceType: 'supplier' }).subscribe();
    const request = http.expectOne(
      (candidate) =>
        candidate.url === 'http://api.test/api/v1/audit/events' &&
        candidate.params.get('page') === '2' &&
        candidate.params.get('resourceType') === 'supplier',
    );
    expect(request.request.method).toBe('GET');
    request.flush({ items: [], page: 2, pageSize: 10, total: 0, pages: 0 });
    http.verify();
  });
});
