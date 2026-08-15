import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { ProcurementService } from '../../data-access/procurement.service';
import { OperationsBoard } from './operations-board';

describe('OperationsBoard', () => {
  it('renders fulfillment counters', async () => {
    await TestBed.configureTestingModule({
      imports: [OperationsBoard],
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: { roles: signal(['finance']) } },
        {
          provide: ProcurementService,
          useValue: {
            suppliers: () => of({ items: [], total: 0, canManage: true }),
            operationsBoard: () =>
              of({
                items: [],
                total: 0,
                awaitingOrderCount: 2,
                inDeliveryCount: 1,
                overdueDeliveryCount: 1,
                receivedCount: 3,
              }),
          },
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(OperationsBoard);
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent as string;
    expect(text).toContain('Đặt hàng và giao nhận');
    expect(text).toContain('Giao trễ');
    expect(text).toContain('3');
  });
});
