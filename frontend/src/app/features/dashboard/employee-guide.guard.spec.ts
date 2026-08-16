import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import {
  ActivatedRouteSnapshot,
  provideRouter,
  Router,
  RouterStateSnapshot,
  UrlTree,
} from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';
import { canAccessEmployeeGuideGuard } from './employee-guide.guard';

describe('canAccessEmployeeGuideGuard', () => {
  const roles = signal<string[]>([]);
  const route = {} as ActivatedRouteSnapshot;
  const state = {} as RouterStateSnapshot;

  beforeEach(() => {
    roles.set([]);
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: { roles: roles.asReadonly() } },
      ],
    });
  });

  it('allows employee accounts', () => {
    roles.set(['employee']);

    const result = TestBed.runInInjectionContext(() => canAccessEmployeeGuideGuard(route, state));

    expect(result).toBe(true);
  });

  it('redirects non-employee accounts to the dashboard', () => {
    roles.set(['finance']);

    const result = TestBed.runInInjectionContext(() =>
      canAccessEmployeeGuideGuard(route, state),
    ) as UrlTree;

    expect(TestBed.inject(Router).serializeUrl(result)).toBe('/dashboard');
  });
});
