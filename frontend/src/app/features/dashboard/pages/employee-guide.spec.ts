import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { EmployeeGuide } from './employee-guide';

describe('EmployeeGuide', () => {
  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EmployeeGuide],
      providers: [provideRouter([])],
    }).compileComponents();
  });

  it('explains the complete employee workflow and provides working shortcuts', () => {
    const fixture = TestBed.createComponent(EmployeeGuide);
    fixture.detectChanges();
    const element = fixture.nativeElement as HTMLElement;

    expect(element.querySelector('h1')?.textContent).toContain(
      'Hướng dẫn công việc dành cho Nhân viên',
    );
    expect(element.textContent).toContain('Nhân viên → Trưởng bộ phận → Tài chính → Nhân viên');
    expect(element.textContent).toContain('Chỉ lưu bản nháp');
    expect(element.textContent).toContain('CHANGES_REQUESTED');
    expect(element.textContent).toContain('Xác nhận đã nhận');
    expect(element.querySelector('a[href="/purchase-requests/new"]')).toBeTruthy();
    expect(element.querySelector('a[href="/work-center"]')).toBeTruthy();
    expect(element.querySelector('a[href="/operations"]')).toBeTruthy();
    expect(element.querySelector('a[href="/notifications"]')).toBeTruthy();
  });
});
