import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { WorkSummary } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { WorkCenter } from './work-center';

const summary: WorkSummary = {
  total: 1,
  overdueCount: 1,
  dueSoonCount: 0,
  items: [
    {
      purchaseRequestId: 'request-id',
      requestCode: 'PR-2026-000001',
      title: 'Laptop cho nhóm thiết kế',
      requesterName: 'employee.demo',
      departmentName: 'General Department',
      status: 'SUBMITTED',
      taskType: 'MANAGER_REVIEW',
      currency: 'VND',
      totalAmount: '25000000.0000',
      dueAt: '2026-08-13T10:00:00Z',
      overdue: true,
      urgency: 'OVERDUE',
      updatedAt: '2026-08-10T10:00:00Z',
    },
  ],
};

describe('WorkCenter', () => {
  it('renders role task and SLA urgency', async () => {
    await TestBed.configureTestingModule({
      imports: [WorkCenter],
      providers: [
        provideRouter([]),
        {
          provide: ProcurementService,
          useValue: { taskSummary: () => of(summary) },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(WorkCenter);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;

    expect(text).toContain('Việc của tôi');
    expect(text).toContain('Trưởng bộ phận xem xét');
    expect(text).toContain('Quá hạn');
    expect(text).toContain('PR-2026-000001');
  });
});
