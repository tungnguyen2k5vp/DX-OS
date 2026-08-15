import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { ReportingService } from '../../data-access/reporting.service';
import { AuditCenter } from './audit-center';

describe('AuditCenter', () => {
  it('renders audit evidence summary', async () => {
    await TestBed.configureTestingModule({
      imports: [AuditCenter],
      providers: [
        {
          provide: ReportingService,
          useValue: {
            auditEvents: () =>
              of({
                items: [],
                page: 1,
                pageSize: 20,
                total: 0,
                pages: 0,
                todayCount: 5,
                supplierChangeCount: 2,
                purchaseOrderEventCount: 3,
              }),
          },
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(AuditCenter);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Trung tâm kiểm toán');
    expect(text).toContain('Sự kiện hôm nay');
    expect(text).toContain('5');
  });
});
