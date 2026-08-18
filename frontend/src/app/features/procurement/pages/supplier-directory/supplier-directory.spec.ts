import { TestBed } from '@angular/core/testing';
import { of } from 'rxjs';
import { ProcurementService } from '../../data-access/procurement.service';
import { SupplierDirectory } from './supplier-directory';

describe('SupplierDirectory', () => {
  it('renders supplier risk and finance management action', async () => {
    await TestBed.configureTestingModule({
      imports: [SupplierDirectory],
      providers: [
        {
          provide: ProcurementService,
          useValue: {
            suppliers: () =>
              of({
                items: [
                  {
                    id: 's1',
                    code: 'VEN-01',
                    name: 'Demo Vendor',
                    status: 'ACTIVE',
                    riskLevel: 'HIGH',
                    version: 1,
                  },
                ],
                total: 1,
                canManage: true,
              }),
          },
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(SupplierDirectory);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Demo Vendor');
    expect(text).toContain('Cao');
    expect(text).toContain('Thêm nhà cung cấp');
  });
});
