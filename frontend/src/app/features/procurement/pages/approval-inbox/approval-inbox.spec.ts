import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { ProcurementService } from '../../data-access/procurement.service';
import { ApprovalInbox } from './approval-inbox';

describe('ApprovalInbox', () => {
  it('loads submitted requests for a department manager', async () => {
    const list = vi.fn().mockReturnValue(
      of({
        items: [],
        page: 1,
        pageSize: 20,
        total: 0,
        pages: 0,
      }),
    );
    await TestBed.configureTestingModule({
      imports: [ApprovalInbox],
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            roles: signal(['department_manager']).asReadonly(),
          },
        },
        {
          provide: ProcurementService,
          useValue: { list },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(ApprovalInbox);
    fixture.detectChanges();

    expect(list).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'SUBMITTED',
    });
    expect((fixture.nativeElement as HTMLElement).textContent).toContain(
      'Không còn phiếu chờ xử lý',
    );
  });

  it('loads manager-approved requests for finance', async () => {
    const list = vi.fn().mockReturnValue(
      of({
        items: [],
        page: 1,
        pageSize: 20,
        total: 0,
        pages: 0,
      }),
    );
    await TestBed.configureTestingModule({
      imports: [ApprovalInbox],
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            roles: signal(['finance']).asReadonly(),
          },
        },
        {
          provide: ProcurementService,
          useValue: { list },
        },
      ],
    }).compileComponents();

    const fixture = TestBed.createComponent(ApprovalInbox);
    fixture.detectChanges();

    expect(list).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'MANAGER_APPROVED',
    });
  });
});
