import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { ProcurementService } from '../../data-access/procurement.service';
import { ApprovalGovernancePage } from './approval-governance';

describe('ApprovalGovernancePage', () => {
  it('shows rules and role capabilities returned by the API', async () => {
    await TestBed.configureTestingModule({
      imports: [ApprovalGovernancePage],
      providers: [{
        provide: ProcurementService,
        useValue: {
          approvalGovernance: () => of({
            rules: [], delegations: [], delegateCandidates: [],
            canManageRules: false, canDelegate: true,
          }),
        },
      }],
    }).compileComponents();
    const fixture = TestBed.createComponent(ApprovalGovernancePage);
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('Ủy quyền và quy tắc phê duyệt');
  });
});
