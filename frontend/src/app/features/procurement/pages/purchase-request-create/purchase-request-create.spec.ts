import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { ProcurementService } from '../../data-access/procurement.service';
import { PurchaseRequestCreate } from './purchase-request-create';

describe('PurchaseRequestCreate', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [PurchaseRequestCreate],
      providers: [
        provideRouter([]),
        {
          provide: ProcurementService,
          useValue: { create: vi.fn() },
        },
      ],
    }).compileComponents();
  });

  it('rejects zero quantity before sending the form', () => {
    const fixture = TestBed.createComponent(PurchaseRequestCreate);
    const component = fixture.componentInstance;

    component.items.at(0).controls.quantity.setValue('0');

    expect(component.items.at(0).controls.quantity.invalid).toBe(true);
  });

  it('keeps at least one item and supports adding another item', () => {
    const fixture = TestBed.createComponent(PurchaseRequestCreate);
    const component = fixture.componentInstance;

    component.removeItem(0);
    expect(component.items.length).toBe(1);

    component.addItem();
    expect(component.items.length).toBe(2);
  });
});
