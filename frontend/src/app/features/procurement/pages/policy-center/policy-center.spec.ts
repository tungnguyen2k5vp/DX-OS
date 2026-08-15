import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { ProcurementService } from '../../data-access/procurement.service';
import { PolicyCenterPage } from './policy-center';

describe('PolicyCenterPage', () => {
  it('renders auditor policy evidence without edit actions', async () => {
    await TestBed.configureTestingModule({
      imports: [PolicyCenterPage],
      providers: [
        {
          provide: ProcurementService,
          useValue: {
            policyCenter: () =>
              of({
                slaPolicies: [
                  {
                    processName: 'PURCHASE_REQUEST_APPROVAL',
                    targetHours: 72,
                    active: true,
                    version: 1,
                  },
                ],
                attachmentRules: [],
                canManage: false,
              }),
          },
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(PolicyCenterPage);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Chính sách vận hành');
    expect(text).toContain('Chỉ đọc cho kiểm toán');
    expect(text).toContain('72 giờ');
    expect(text).not.toContain('Sửa');
  });
});
