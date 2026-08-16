import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { AuthService } from './core/auth/auth.service';
import { NotificationService } from './core/notifications/notification.service';
import { App } from './app';

describe('App', () => {
  let roles: ReturnType<typeof signal<string[]>>;

  beforeEach(async () => {
    roles = signal<string[]>(['employee']);
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        provideRouter([]),
        {
          provide: AuthService,
          useValue: {
            username: signal('employee.demo').asReadonly(),
            roles: roles.asReadonly(),
            logout: vi.fn().mockResolvedValue(undefined),
          },
        },
        {
          provide: NotificationService,
          useValue: {
            unreadCount: signal(2).asReadonly(),
            refreshUnreadCount: vi.fn(),
          },
        },
      ],
    }).compileComponents();
  });

  it('creates the application shell', () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();

    expect(fixture.componentInstance).toBeTruthy();
    expect((fixture.nativeElement as HTMLElement).textContent).toContain('DX-OS');
    expect((fixture.nativeElement as HTMLElement).textContent).toContain('Nhân viên');
  });

  it('shows the employee guide only for the employee role', () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();

    expect(
      (fixture.nativeElement as HTMLElement).querySelector('a[href="/employee-guide"]'),
    ).toBeTruthy();

    roles.set(['finance']);
    fixture.detectChanges();

    expect(
      (fixture.nativeElement as HTMLElement).querySelector('a[href="/employee-guide"]'),
    ).toBeNull();
  });
});
