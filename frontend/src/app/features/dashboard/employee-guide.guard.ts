import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';

export const canAccessEmployeeGuideGuard: CanActivateFn = () => {
  if (inject(AuthService).roles().includes('employee')) return true;
  return inject(Router).createUrlTree(['/dashboard']);
};
