import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { BudgetDashboard } from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { BudgetDashboardPage } from './budget-dashboard';

const dashboard: BudgetDashboard = {
  allocations: [
    {
      id: 'allocation-id',
      periodCode: 'FY-2026',
      periodStart: '2026-01-01',
      periodEnd: '2026-12-31',
      costCenter: 'CC-GENERAL',
      currency: 'VND',
      allocatedAmount: '100000000000.0000',
      reservedAmount: '0.0000',
      committedAmount: '27000000.0000',
      availableAmount: '99973000000.0000',
      utilization: '0.03',
      alertLevel: 'HEALTHY',
      version: 3,
    },
  ],
  totals: [
    {
      currency: 'VND',
      allocatedAmount: '100000000000.0000',
      reservedAmount: '0.0000',
      committedAmount: '27000000.0000',
      availableAmount: '99973000000.0000',
    },
  ],
  reservations: [],
  adjustments: [],
  alertCount: 0,
  canManage: false,
};

describe('BudgetDashboardPage', () => {
  it('renders auditor mode without an adjustment button', async () => {
    await TestBed.configureTestingModule({
      imports: [BudgetDashboardPage],
      providers: [
        {
          provide: ProcurementService,
          useValue: {
            budgetDashboard: () => of(dashboard),
          },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(BudgetDashboardPage);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;

    expect(text).toContain('Chế độ kiểm toán chỉ đọc');
    expect(text).toContain('CC-GENERAL');
    expect(text).not.toContain('Điều chỉnh tổng hạn mức');
  });
});
