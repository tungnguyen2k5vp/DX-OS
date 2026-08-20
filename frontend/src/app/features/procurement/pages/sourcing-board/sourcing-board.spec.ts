import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { ProcurementService } from '../../data-access/procurement.service';
import { SourcingBoardPage } from './sourcing-board';

describe('SourcingBoardPage', () => {
  it('renders the tenant sourcing summary', async () => {
    await TestBed.configureTestingModule({
      imports: [SourcingBoardPage],
      providers: [{
        provide: ProcurementService,
        useValue: {
          sourcingBoard: () => of({
            items: [], total: 0, awaitingQuotes: 0, inComparison: 0, awarded: 0, canManage: true,
          }),
          suppliers: () => of({ items: [], total: 0, canManage: true }),
        },
      }],
    }).compileComponents();
    const fixture = TestBed.createComponent(SourcingBoardPage);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('So sánh báo giá');
  });
});
